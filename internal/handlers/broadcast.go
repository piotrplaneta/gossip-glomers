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

type SafeAckedMessages struct {
	mutex                         sync.Mutex
	v                             map[NodeId]map[int]bool
	lastPropagateBroadcastAttempt map[NodeId]map[int]time.Time
}

var ackedMessages = SafeAckedMessages{
	v:                             make(map[NodeId]map[int]bool),
	lastPropagateBroadcastAttempt: make(map[NodeId]map[int]time.Time),
}

var propagateBroadcastsTicker = time.NewTicker(1000 * time.Millisecond)
var propagateBroadcastsChannel = make(chan time.Time)

func propagateBroadcasts(n *maelstrom.Node) {
	ackedMessages.mutex.Lock()

	for nodeId, ackedByNode := range ackedMessages.v {
		messages.mutex.Lock()
		messagesToSend := make([]int, 0)

		for message := range messages.v {
			if !ackedByNode[message] && time.Since(ackedMessages.lastPropagateBroadcastAttempt[nodeId][message]) > time.Second {
				messagesToSend = append(messagesToSend, message)
			}
		}
		messages.mutex.Unlock()

		for _, message := range messagesToSend {
			ackedMessages.lastPropagateBroadcastAttempt[nodeId][message] = time.Now()
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

	ackedMessages.mutex.Lock()
	messages.mutex.Lock()
	if !messages.v[message] {
		messages.v[message] = true
	}
	messages.mutex.Unlock()
	ackedMessages.v[NodeId(msg.Src)][message] = true
	ackedMessages.mutex.Unlock()

	propagateBroadcastsChannel <- time.Now()

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

	propagateBroadcastsChannel <- time.Now()

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

func TopologyHandler(msg maelstrom.Message, n *maelstrom.Node, finishingChannel chan string) error {
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
			ackedMessages.lastPropagateBroadcastAttempt[NodeId(nodeId)] = make(map[int]time.Time)
		}
	}

	ackedMessages.mutex.Unlock()

	go tickBroadcasts(n, finishingChannel)
	go startTickerAndOnDemandBroadcastsPropagation(n, finishingChannel)

	return n.Reply(msg, map[string]string{"type": "topology_ok"})
}

func tickBroadcasts(n *maelstrom.Node, finishingChannel chan string) {
	go func() {
		for {
			select {
			case <-propagateBroadcastsTicker.C:
				propagateBroadcastsChannel <- time.Now()
			case <-finishingChannel:
				return
			}
		}
	}()
}

func startTickerAndOnDemandBroadcastsPropagation(n *maelstrom.Node, finishingChannel chan string) {
	go func() {
		for {
			select {
			case <-propagateBroadcastsChannel:
				propagateBroadcasts(n)
			case <-finishingChannel:
				return
			}
		}
	}()
}
