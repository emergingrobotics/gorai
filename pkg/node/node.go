// Package node provides the core Node implementation for Gorai.
//
// A Node is the primary building block of a Gorai application. It manages
// the lifecycle of publishers, subscribers, services, and actions, and
// handles connection to the NATS messaging system.
package node

import (
	"context"
	"fmt"
	"sync"

	"github.com/nats-io/nats.go"
)

// Node represents a Gorai node.
type Node struct {
	name string
	nc   *nats.Conn
	js   nats.JetStreamContext

	mu       sync.RWMutex
	running  bool
	shutdown chan struct{}
}

// Option configures a Node.
type Option func(*Node) error

// WithNATS sets the NATS server URL.
func WithNATS(url string) Option {
	return func(n *Node) error {
		nc, err := nats.Connect(url)
		if err != nil {
			return fmt.Errorf("failed to connect to NATS: %w", err)
		}
		n.nc = nc

		js, err := nc.JetStream()
		if err != nil {
			return fmt.Errorf("failed to get JetStream context: %w", err)
		}
		n.js = js

		return nil
	}
}

// WithConnection uses an existing NATS connection.
func WithConnection(nc *nats.Conn) Option {
	return func(n *Node) error {
		n.nc = nc
		js, err := nc.JetStream()
		if err != nil {
			return fmt.Errorf("failed to get JetStream context: %w", err)
		}
		n.js = js
		return nil
	}
}

// New creates a new Node with the given name and options.
func New(name string, opts ...Option) (*Node, error) {
	n := &Node{
		name:     name,
		shutdown: make(chan struct{}),
	}

	for _, opt := range opts {
		if err := opt(n); err != nil {
			return nil, err
		}
	}

	return n, nil
}

// Name returns the node's name.
func (n *Node) Name() string {
	return n.name
}

// NATS returns the underlying NATS connection.
func (n *Node) NATS() *nats.Conn {
	return n.nc
}

// JetStream returns the JetStream context.
func (n *Node) JetStream() nats.JetStreamContext {
	return n.js
}

// Spin runs the node until the context is canceled.
func (n *Node) Spin(ctx context.Context) error {
	n.mu.Lock()
	n.running = true
	n.mu.Unlock()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-n.shutdown:
		return nil
	}
}

// Close shuts down the node and releases resources.
func (n *Node) Close() error {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		close(n.shutdown)
		n.running = false
	}

	if n.nc != nil {
		n.nc.Close()
	}

	return nil
}
