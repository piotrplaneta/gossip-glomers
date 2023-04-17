package handlers

import (
	"encoding/json"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func EchoHandler(msg maelstrom.Message, n *maelstrom.Node) error {
	var body map[string]any
	if err := json.Unmarshal(msg.Body, &body); err != nil {
		return err
	}

	body["type"] = "echo_ok"

	return n.Reply(msg, body)
}