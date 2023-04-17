package main

import (
	"internal/handlers"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()

	n.Handle("echo", func(msg maelstrom.Message) error {
		return handlers.EchoHandler(msg, n)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
