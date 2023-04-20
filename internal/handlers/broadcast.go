package handlers

import (
	"encoding/json"
	"log"
	"sync"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

type NodeId string

type SafeMessages struct {
	mutex sync.Mutex
	v     map[int]bool
}

var messages = SafeMessages{v: make(map[int]bool)}

type SafeLastSeenMessages struct {
	mutex sync.Mutex
	v     map[NodeId]map[int]bool
}

var ackedMessages = SafeLastSeenMessages{v: make(map[NodeId]map[int]bool)}

func propagateBroadcasts(n *maelstrom.Node) {
	ackedMessages.mutex.Lock()

	for nodeId, lastSeenForNode := range ackedMessages.v {

		messages.mutex.Lock()
		messagesToSend := make([]int, 0)

		for message := range messages.v {
			if !lastSeenForNode[message] {
				messagesToSend = append(messagesToSend, message)
			}
		}
		messages.mutex.Unlock()

		for _, message := range messagesToSend {
			n.RPC(string(nodeId), map[string]any{"type": "propagate_broadcast", "message": message}, func(msg maelstrom.Message) error {
				var body map[string]any
				if err := json.Unmarshal(msg.Body, &body); err != nil {
					return err
				}

				ackedMessage := int(body["acked_message"].(float64))
				ackedMessages.mutex.Lock()
				ackedMessages.v[NodeId(msg.Src)][ackedMessage] = true
				ackedMessages.mutex.Unlock()

				return nil
			})
		}

	}
	ackedMessages.mutex.Unlock()
}

func PropagateBroadcastHandler(msg maelstrom.Message, n *maelstrom.Node) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	message := int(body["message"].(float64))

	messages.mutex.Lock()
	ackedMessages.mutex.Lock()
	if !messages.v[message] {
		messages.v[message] = true
	}
	ackedMessages.v[NodeId(msg.Src)][message] = true
	ackedMessages.mutex.Unlock()
	messages.mutex.Unlock()

	propagateBroadcasts(n)

	return n.Reply(msg, map[string]any{"type": "propagate_broadcast_ok", "acked_message": message})
}

func BroadcastHandler(msg maelstrom.Message, n *maelstrom.Node) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	message := int(body["message"].(float64))

	messages.mutex.Lock()
	if !messages.v[message] {
		messages.v[message] = true

	}
	messages.mutex.Unlock()

	return n.Reply(msg, map[string]string{"type": "broadcast_ok"})
}

func ReadHandler(msg maelstrom.Message, n *maelstrom.Node) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	messagesToSend := make([]int, len(messages.v))

	messages.mutex.Lock()
	for message := range messages.v {
		messagesToSend = append(messagesToSend, message)
	}
	messages.mutex.Unlock()

	return n.Reply(msg, map[string]any{"type": "read_ok", "messages": messagesToSend})
}

type TopologyType struct {
	Topology map[string][]string `json:"topology"`
}

func TopologyHandler(msg maelstrom.Message, n *maelstrom.Node) error {
	var body TopologyType
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	topology := body.Topology
	neighbours := topology[n.ID()]

	ackedMessages.mutex.Lock()
	for _, nodeId := range neighbours {
		if nodeId != n.ID() {
			ackedMessages.v[NodeId(nodeId)] = make(map[int]bool)
		}
	}

	log.Println(ackedMessages.v)
	ackedMessages.mutex.Unlock()

	go propagateBroadcasts(n)

	return n.Reply(msg, map[string]string{"type": "topology_ok"})
}
