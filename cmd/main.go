package main

import (
	"internal/echo"
	"log"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()

	n.Handle("echo", func(msg maelstrom.Message) error {
		return echo.EchoHandler(msg, n)
	})

	if err := n.Run(); err != nil {
		log.Fatal(err)
	}
}
