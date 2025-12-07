// Package node provides the core Node implementation for Gorai.
//
// A Node is the primary building block of a Gorai application. It manages
// the lifecycle of publishers, subscribers, services, and actions, and
// handles connection to the NATS messaging system.
package node

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
)

// Node represents a Gorai node.
type Node struct {
	name      string
	namespace string
	nc        *nats.Conn
	js        nats.JetStreamContext
	logger    *slog.Logger

	mu       sync.RWMutex
	running  bool
	shutdown chan struct{}

	// Subscriptions tracked for SpinOnce
	subs []*nats.Subscription
}

// Option configures a Node.
type Option func(*Node) error

// WithNATS sets the NATS server URL.
func WithNATS(url string) Option {
	return func(n *Node) error {
		nc, err := nats.Connect(url,
			nats.Name(n.FullName()),
			nats.ReconnectWait(time.Second),
			nats.MaxReconnects(-1),
		)
		if err != nil {
			return fmt.Errorf("failed to connect to NATS: %w", err)
		}
		n.nc = nc

		js, err := nc.JetStream()
		if err != nil {
			// JetStream might not be available, that's ok for basic pub/sub
			n.logger.Warn("JetStream not available", "error", err)
		} else {
			n.js = js
		}

		return nil
	}
}

// WithNATSOptions sets the NATS server URL with additional options.
func WithNATSOptions(url string, opts ...nats.Option) Option {
	return func(n *Node) error {
		// Prepend our default options
		allOpts := append([]nats.Option{
			nats.Name(n.FullName()),
			nats.ReconnectWait(time.Second),
			nats.MaxReconnects(-1),
		}, opts...)

		nc, err := nats.Connect(url, allOpts...)
		if err != nil {
			return fmt.Errorf("failed to connect to NATS: %w", err)
		}
		n.nc = nc

		js, err := nc.JetStream()
		if err != nil {
			n.logger.Warn("JetStream not available", "error", err)
		} else {
			n.js = js
		}

		return nil
	}
}

// WithConnection uses an existing NATS connection.
func WithConnection(nc *nats.Conn) Option {
	return func(n *Node) error {
		n.nc = nc
		js, err := nc.JetStream()
		if err != nil {
			n.logger.Warn("JetStream not available", "error", err)
		} else {
			n.js = js
		}
		return nil
	}
}

// WithNamespace sets the node's namespace.
func WithNamespace(ns string) Option {
	return func(n *Node) error {
		n.namespace = ns
		return nil
	}
}

// WithLogger sets the node's logger.
func WithLogger(logger *slog.Logger) Option {
	return func(n *Node) error {
		n.logger = logger
		return nil
	}
}

// New creates a new Node with the given name and options.
func New(name string, opts ...Option) (*Node, error) {
	n := &Node{
		name:     name,
		shutdown: make(chan struct{}),
		logger:   slog.Default(),
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

// Namespace returns the node's namespace.
func (n *Node) Namespace() string {
	return n.namespace
}

// FullName returns the fully qualified node name (namespace.name or just name).
func (n *Node) FullName() string {
	if n.namespace != "" {
		return n.namespace + "." + n.name
	}
	return n.name
}

// NATS returns the underlying NATS connection.
func (n *Node) NATS() *nats.Conn {
	return n.nc
}

// JetStream returns the JetStream context.
// Returns nil if JetStream is not available.
func (n *Node) JetStream() nats.JetStreamContext {
	return n.js
}

// HasJetStream returns true if JetStream is available.
func (n *Node) HasJetStream() bool {
	return n.js != nil
}

// Logger returns the node's logger.
func (n *Node) Logger() *slog.Logger {
	return n.logger
}

// IsConnected returns true if the NATS connection is established.
func (n *Node) IsConnected() bool {
	return n.nc != nil && n.nc.IsConnected()
}

// Spin runs the node until the context is canceled.
func (n *Node) Spin(ctx context.Context) error {
	n.mu.Lock()
	if n.running {
		n.mu.Unlock()
		return fmt.Errorf("node is already running")
	}
	n.running = true
	n.mu.Unlock()

	defer func() {
		n.mu.Lock()
		n.running = false
		n.mu.Unlock()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-n.shutdown:
		return nil
	}
}

// SpinOnce processes pending messages once and returns.
// This is useful for testing and single-threaded scenarios.
func (n *Node) SpinOnce(ctx context.Context, timeout time.Duration) error {
	if n.nc == nil {
		return nil
	}

	// Flush to ensure all pending operations are sent
	if err := n.nc.FlushTimeout(timeout); err != nil {
		if err == nats.ErrTimeout {
			return nil // Timeout is ok, just means nothing pending
		}
		return fmt.Errorf("flush failed: %w", err)
	}

	return nil
}

// IsRunning returns true if the node is currently spinning.
func (n *Node) IsRunning() bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	return n.running
}

// Shutdown signals the node to stop spinning.
func (n *Node) Shutdown() {
	n.mu.Lock()
	defer n.mu.Unlock()

	if n.running {
		select {
		case <-n.shutdown:
			// Already closed
		default:
			close(n.shutdown)
		}
	}
}

// Close shuts down the node and releases resources.
func (n *Node) Close() error {
	n.Shutdown()

	n.mu.Lock()
	defer n.mu.Unlock()

	// Unsubscribe all tracked subscriptions
	for _, sub := range n.subs {
		if sub.IsValid() {
			sub.Unsubscribe()
		}
	}
	n.subs = nil

	if n.nc != nil {
		n.nc.Close()
		n.nc = nil
	}

	return nil
}

// TrackSubscription adds a subscription to be cleaned up on Close.
func (n *Node) TrackSubscription(sub *nats.Subscription) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.subs = append(n.subs, sub)
}
