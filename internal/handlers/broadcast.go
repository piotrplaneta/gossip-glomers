package handlers

import (
	"encoding/json"
	"sync"
	"time"

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

var seenMessages = SafeLastSeenMessages{v: make(map[NodeId]map[int]bool)}

func propagateBroadcasts(n *maelstrom.Node) {
	seenMessages.mutex.Lock()

	for nodeId, lastSeenForNode := range seenMessages.v {
		messagesToSend := make([]int, len(messages.v))

		for message := range messages.v {
			if !lastSeenForNode[message] {
				messagesToSend = append(messagesToSend, message)
			}
		}

		for _, message := range messagesToSend {
			n.RPC(string(nodeId), map[string]any{"type": "propagate_broadcast", "message": message}, func(msg maelstrom.Message) error {
				seenMessages.mutex.Lock()
				seenMessages.v[nodeId][message] = true
				seenMessages.mutex.Unlock()

				return nil
			})
		}

	}
	seenMessages.mutex.Unlock()

	time.Sleep(time.Second)
	go propagateBroadcasts(n)
}

func PropagateBroadcastHandler(msg maelstrom.Message, n *maelstrom.Node) error {
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

	return n.Reply(msg, map[string]string{"type": "propagate_broadcast_ok"})
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

	for message := range messages.v {
		messagesToSend = append(messagesToSend, message)
	}

	return n.Reply(msg, map[string]any{"type": "read_ok", "messages": messagesToSend})
}

func TopologyHandler(msg maelstrom.Message, n *maelstrom.Node) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	seenMessages.mutex.Lock()
	for _, nodeId := range n.NodeIDs() {
		if nodeId != n.ID() {
			seenMessages.v[NodeId(nodeId)] = make(map[int]bool)
		}
	}
	seenMessages.mutex.Unlock()

	go propagateBroadcasts(n)

	return n.Reply(msg, map[string]string{"type": "topology_ok"})
}
