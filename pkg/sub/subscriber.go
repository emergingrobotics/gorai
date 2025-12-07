// Package sub provides subscribers for Gorai topics.
package sub

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// Subscriber subscribes to messages on a topic.
type Subscriber[T proto.Message] struct {
	nc       *nats.Conn
	js       nats.JetStreamContext
	topic    string
	sub      *nats.Subscription
	jsSub    *nats.Subscription
	qos      QoS
	opts     options
	logger   *slog.Logger
	cancel   context.CancelFunc
}

// NATSGetter is an interface for types that provide a NATS connection.
type NATSGetter interface {
	NATS() *nats.Conn
}

// JetStreamGetter is an interface for types that provide a JetStream context.
type JetStreamGetter interface {
	JetStream() nats.JetStreamContext
}

// Handler is a callback function for received messages.
type Handler[T proto.Message] func(msg T)

// MessageHandler is a callback with message metadata.
type MessageHandler[T proto.Message] func(msg T, meta *MessageMeta)

// MessageMeta contains metadata about a received message.
type MessageMeta struct {
	Topic     string
	Timestamp time.Time
	Sequence  uint64
	Redelivered bool
}

// options holds subscriber configuration.
type options struct {
	qos          QoS
	durableName  string
	deliverNew   bool
	deliverAll   bool
	deliverLast  bool
	ackWait      time.Duration
	maxDeliver   int
	logger       *slog.Logger
}

// Option configures a subscriber.
type Option func(*options)

// WithSubQoS sets the quality of service level.
func WithSubQoS(qos QoS) Option {
	return func(o *options) {
		o.qos = qos
	}
}

// WithDurable creates a durable subscription with the given name.
func WithDurable(name string) Option {
	return func(o *options) {
		o.qos = Durable
		o.durableName = name
	}
}

// WithDeliverNew delivers only new messages (default).
func WithDeliverNew() Option {
	return func(o *options) {
		o.deliverNew = true
		o.deliverAll = false
		o.deliverLast = false
	}
}

// WithDeliverAll delivers all available messages.
func WithDeliverAll() Option {
	return func(o *options) {
		o.deliverAll = true
		o.deliverNew = false
		o.deliverLast = false
	}
}

// WithDeliverLast delivers the last message first, then new.
func WithDeliverLast() Option {
	return func(o *options) {
		o.deliverLast = true
		o.deliverNew = false
		o.deliverAll = false
	}
}

// WithAckWait sets the acknowledgment wait time.
func WithAckWait(d time.Duration) Option {
	return func(o *options) {
		o.ackWait = d
	}
}

// WithMaxDeliver sets the maximum delivery attempts.
func WithMaxDeliver(n int) Option {
	return func(o *options) {
		o.maxDeliver = n
	}
}

// WithSubLogger sets the logger for the subscriber.
func WithSubLogger(logger *slog.Logger) Option {
	return func(o *options) {
		o.logger = logger
	}
}

// New creates a new Subscriber for the given topic.
func New[T proto.Message](n NATSGetter, topic string, handler Handler[T], opts ...Option) (*Subscriber[T], error) {
	s := &Subscriber[T]{
		nc:     n.NATS(),
		topic:  topic,
		qos:    BestEffort,
		logger: slog.Default(),
	}

	// Apply options
	for _, opt := range opts {
		opt(&s.opts)
	}
	s.qos = s.opts.qos
	if s.opts.logger != nil {
		s.logger = s.opts.logger
	}

	// Try to get JetStream if available
	if jsg, ok := n.(JetStreamGetter); ok {
		s.js = jsg.JetStream()
	}

	if s.nc == nil {
		return nil, fmt.Errorf("no NATS connection")
	}

	var err error
	switch s.qos {
	case BestEffort:
		err = s.subscribeBestEffort(handler)
	case Reliable, Durable:
		err = s.subscribeJetStream(handler)
	}

	if err != nil {
		return nil, err
	}

	return s, nil
}

// subscribeBestEffort creates a regular NATS subscription.
func (s *Subscriber[T]) subscribeBestEffort(handler Handler[T]) error {
	sub, err := s.nc.Subscribe(s.topic, func(m *nats.Msg) {
		var msg T
		// Create a new instance of T
		msg = msg.ProtoReflect().New().Interface().(T)

		if err := proto.Unmarshal(m.Data, msg); err != nil {
			s.logger.Error("failed to unmarshal message", "error", err, "topic", s.topic)
			return
		}

		handler(msg)
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe: %w", err)
	}

	s.sub = sub
	return nil
}

// subscribeJetStream creates a JetStream subscription.
func (s *Subscriber[T]) subscribeJetStream(handler Handler[T]) error {
	if s.js == nil {
		return fmt.Errorf("JetStream not available for QoS %s", s.qos)
	}

	// Build consumer config
	consumerOpts := []nats.SubOpt{}

	if s.opts.durableName != "" {
		consumerOpts = append(consumerOpts, nats.Durable(s.opts.durableName))
	}

	if s.opts.deliverAll {
		consumerOpts = append(consumerOpts, nats.DeliverAll())
	} else if s.opts.deliverLast {
		consumerOpts = append(consumerOpts, nats.DeliverLast())
	} else {
		consumerOpts = append(consumerOpts, nats.DeliverNew())
	}

	if s.opts.ackWait > 0 {
		consumerOpts = append(consumerOpts, nats.AckWait(s.opts.ackWait))
	}

	if s.opts.maxDeliver > 0 {
		consumerOpts = append(consumerOpts, nats.MaxDeliver(s.opts.maxDeliver))
	}

	// Manual ack mode
	consumerOpts = append(consumerOpts, nats.ManualAck())

	// Subscribe via JetStream
	sub, err := s.js.Subscribe(s.topic, func(m *nats.Msg) {
		var msg T
		msg = msg.ProtoReflect().New().Interface().(T)

		if err := proto.Unmarshal(m.Data, msg); err != nil {
			s.logger.Error("failed to unmarshal message", "error", err, "topic", s.topic)
			m.Nak() // Negative ack to trigger redelivery
			return
		}

		handler(msg)
		m.Ack() // Acknowledge successful processing
	}, consumerOpts...)

	if err != nil {
		// If stream doesn't exist, fall back to best effort
		if strings.Contains(err.Error(), "stream not found") ||
		   strings.Contains(err.Error(), "no stream") {
			s.logger.Warn("JetStream stream not found, falling back to best effort", "topic", s.topic)
			s.qos = BestEffort
			return s.subscribeBestEffort(handler)
		}
		return fmt.Errorf("failed to subscribe via JetStream: %w", err)
	}

	s.jsSub = sub
	return nil
}

// Topic returns the topic name.
func (s *Subscriber[T]) Topic() string {
	return s.topic
}

// QoS returns the quality of service level.
func (s *Subscriber[T]) QoS() QoS {
	return s.qos
}

// Pending returns the number of pending messages.
func (s *Subscriber[T]) Pending() (int, error) {
	if s.jsSub != nil {
		info, err := s.jsSub.ConsumerInfo()
		if err != nil {
			return 0, err
		}
		return int(info.NumPending), nil
	}
	if s.sub != nil {
		msgs, _, err := s.sub.Pending()
		return msgs, err
	}
	return 0, nil
}

// Unsubscribe stops receiving messages.
func (s *Subscriber[T]) Unsubscribe() error {
	if s.cancel != nil {
		s.cancel()
	}
	if s.jsSub != nil {
		return s.jsSub.Unsubscribe()
	}
	if s.sub != nil {
		return s.sub.Unsubscribe()
	}
	return nil
}

// Drain unsubscribes and waits for pending messages to be processed.
func (s *Subscriber[T]) Drain() error {
	if s.jsSub != nil {
		return s.jsSub.Drain()
	}
	if s.sub != nil {
		return s.sub.Drain()
	}
	return nil
}
