// Package pub provides publishers for Gorai topics.
package pub

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// Publisher publishes messages to a topic.
type Publisher[T proto.Message] struct {
	nc    *nats.Conn
	topic string
}

// NATSGetter is an interface for types that provide a NATS connection.
type NATSGetter interface {
	NATS() *nats.Conn
}

// New creates a new Publisher for the given topic.
func New[T proto.Message](n NATSGetter, topic string) *Publisher[T] {
	return &Publisher[T]{
		nc:    n.NATS(),
		topic: topic,
	}
}

// Publish publishes a message to the topic.
func (p *Publisher[T]) Publish(ctx context.Context, msg T) error {
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	if err := p.nc.Publish(p.topic, data); err != nil {
		return fmt.Errorf("failed to publish message: %w", err)
	}

	return nil
}

// Topic returns the topic name.
func (p *Publisher[T]) Topic() string {
	return p.topic
}
