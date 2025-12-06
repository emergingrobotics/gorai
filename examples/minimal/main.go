// Example minimal demonstrates a basic Gorai node.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorai/gorai/pkg/node"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Create a node connected to NATS
	n, err := node.New("minimal_node", node.WithNATS("nats://localhost:4222"))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create node: %v\n", err)
		os.Exit(1)
	}
	defer n.Close()

	fmt.Println("Node started. Press Ctrl+C to exit.")

	// Spin until shutdown
	if err := n.Spin(ctx); err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Shutting down.")
}
