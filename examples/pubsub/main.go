// Example pubsub demonstrates publishing and subscribing to topics.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorai/gorai/pkg/node"
	"github.com/nats-io/nats.go"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a node
	n, err := node.New("pubsub_demo", node.WithNATS("nats://localhost:4222"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create node: %v\n", err)
		os.Exit(1)
	}
	defer n.Close()

	// Subscribe to a topic
	_, err = n.NATS().Subscribe("demo.messages", func(msg *nats.Msg) {
		fmt.Printf("Received: %s\n", string(msg.Data))
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to subscribe: %v\n", err)
		os.Exit(1)
	}

	// Publish messages in a goroutine
	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		count := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				count++
				msg := fmt.Sprintf("Hello #%d from Gorai!", count)
				if err := n.NATS().Publish("demo.messages", []byte(msg)); err != nil {
					fmt.Fprintf(os.Stderr, "Failed to publish: %v\n", err)
				}
			}
		}
	}()

	fmt.Println("Pub/Sub demo running. Press Ctrl+C to exit.")

	// Spin until shutdown
	if err := n.Spin(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
