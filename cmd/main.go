package main

import (
	"internal/handlers"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()
	finishingChannel := make(chan string)

	n.Handle("echo", func(msg maelstrom.Message) error {
		return handlers.EchoHandler(msg, n)
	})

	n.Handle("generate", func(msg maelstrom.Message) error {
		return handlers.GenerateHandler(msg, n)
	})

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		return handlers.BroadcastHandler(msg, n)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		return handlers.ReadHandler(msg, n)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		return handlers.TopologyHandler(msg, n, finishingChannel)
	})

	n.Handle("propagate_broadcast", func(msg maelstrom.Message) error {
		return handlers.PropagateBroadcastHandler(msg, n)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}

	finishingChannel <- "finish"
}
