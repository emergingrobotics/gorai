// Package sub provides subscribers for Gorai topics.
package sub

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// Subscriber subscribes to messages on a topic.
type Subscriber[T proto.Message] struct {
	nc    *nats.Conn
	topic string
	sub   *nats.Subscription
}

// NATSGetter is an interface for types that provide a NATS connection.
type NATSGetter interface {
	NATS() *nats.Conn
}

// Handler is a callback function for received messages.
type Handler[T proto.Message] func(msg T)

// New creates a new Subscriber for the given topic.
func New[T proto.Message](n NATSGetter, topic string, handler Handler[T]) (*Subscriber[T], error) {
	s := &Subscriber[T]{
		nc:    n.NATS(),
		topic: topic,
	}

	sub, err := s.nc.Subscribe(topic, func(m *nats.Msg) {
		var msg T
		// Create a new instance of T
		msg = msg.ProtoReflect().New().Interface().(T)

		if err := proto.Unmarshal(m.Data, msg); err != nil {
			// Log error but continue
			return
		}

		handler(msg)
	})
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe: %w", err)
	}

	s.sub = sub
	return s, nil
}

// Topic returns the topic name.
func (s *Subscriber[T]) Topic() string {
	return s.topic
}

// Unsubscribe stops receiving messages.
func (s *Subscriber[T]) Unsubscribe() error {
	if s.sub != nil {
		return s.sub.Unsubscribe()
	}
	return nil
}
