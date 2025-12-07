// Package pub provides publishers for Gorai topics.
package pub

import (
	"context"
	"fmt"
	"strings"

	"github.com/nats-io/nats.go"
	"google.golang.org/protobuf/proto"
)

// Publisher publishes messages to a topic.
type Publisher[T proto.Message] struct {
	nc     *nats.Conn
	js     nats.JetStreamContext
	topic  string
	qos    QoS
	opts   options
	stream string
}

// NATSGetter is an interface for types that provide a NATS connection.
type NATSGetter interface {
	NATS() *nats.Conn
}

// JetStreamGetter is an interface for types that provide a JetStream context.
type JetStreamGetter interface {
	JetStream() nats.JetStreamContext
}

// NodeGetter combines NATS and JetStream getters.
type NodeGetter interface {
	NATSGetter
	JetStreamGetter
}

// options holds publisher configuration.
type options struct {
	qos         QoS
	historySize int
	streamName  string
}

// Option configures a publisher.
type Option func(*options)

// WithQoS sets the quality of service level.
func WithQoS(qos QoS) Option {
	return func(o *options) {
		o.qos = qos
	}
}

// WithHistory sets the history depth (requires QoS = History).
func WithHistory(size int) Option {
	return func(o *options) {
		o.qos = History
		o.historySize = size
	}
}

// WithRetain enables last-value retention (sets QoS = Retained).
func WithRetain() Option {
	return func(o *options) {
		o.qos = Retained
	}
}

// WithStreamName sets a custom JetStream stream name.
func WithStreamName(name string) Option {
	return func(o *options) {
		o.streamName = name
	}
}

// New creates a new Publisher for the given topic.
func New[T proto.Message](n NATSGetter, topic string, opts ...Option) *Publisher[T] {
	p := &Publisher[T]{
		nc:    n.NATS(),
		topic: topic,
		qos:   BestEffort,
	}

	// Apply options
	for _, opt := range opts {
		opt(&p.opts)
	}
	p.qos = p.opts.qos

	// Try to get JetStream if available
	if jsg, ok := n.(JetStreamGetter); ok {
		p.js = jsg.JetStream()
	}

	return p
}

// ensureStream creates or updates the JetStream stream if needed.
func (p *Publisher[T]) ensureStream() error {
	if p.js == nil {
		return fmt.Errorf("JetStream not available for QoS %s", p.qos)
	}

	streamName := p.opts.streamName
	if streamName == "" {
		// Generate stream name from topic
		streamName = strings.ReplaceAll(p.topic, ".", "_")
		streamName = strings.ReplaceAll(streamName, "*", "STAR")
		streamName = strings.ReplaceAll(streamName, ">", "GT")
	}
	p.stream = streamName

	cfg := &nats.StreamConfig{
		Name:     streamName,
		Subjects: []string{p.topic},
	}

	switch p.qos {
	case Reliable:
		cfg.Storage = nats.FileStorage
		cfg.Retention = nats.LimitsPolicy
		cfg.MaxMsgs = -1
		cfg.MaxBytes = -1

	case Retained:
		cfg.Storage = nats.FileStorage
		cfg.Retention = nats.LimitsPolicy
		cfg.MaxMsgsPerSubject = 1 // Keep only the last message

	case History:
		cfg.Storage = nats.FileStorage
		cfg.Retention = nats.LimitsPolicy
		historySize := p.opts.historySize
		if historySize <= 0 {
			historySize = 10 // Default history size
		}
		cfg.MaxMsgsPerSubject = int64(historySize)
	}

	// Try to add, if exists update
	_, err := p.js.AddStream(cfg)
	if err != nil {
		// Stream might already exist, try to update
		if nats.ErrStreamNameAlreadyInUse.Error() == err.Error() ||
		   strings.Contains(err.Error(), "already in use") {
			_, err = p.js.UpdateStream(cfg)
		}
	}

	return err
}

// Publish publishes a message to the topic.
func (p *Publisher[T]) Publish(ctx context.Context, msg T) error {
	if p.nc == nil {
		return fmt.Errorf("no NATS connection")
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	switch p.qos {
	case BestEffort:
		if err := p.nc.Publish(p.topic, data); err != nil {
			return fmt.Errorf("failed to publish message: %w", err)
		}

	case Reliable, Retained, History:
		if p.stream == "" {
			if err := p.ensureStream(); err != nil {
				return fmt.Errorf("failed to ensure stream: %w", err)
			}
		}

		_, err := p.js.Publish(p.topic, data)
		if err != nil {
			return fmt.Errorf("failed to publish to JetStream: %w", err)
		}
	}

	return nil
}

// PublishAsync publishes a message asynchronously (best effort only).
func (p *Publisher[T]) PublishAsync(msg T) error {
	if p.nc == nil {
		return fmt.Errorf("no NATS connection")
	}

	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return p.nc.Publish(p.topic, data)
}

// Topic returns the topic name.
func (p *Publisher[T]) Topic() string {
	return p.topic
}

// QoS returns the quality of service level.
func (p *Publisher[T]) QoS() QoS {
	return p.qos
}

// Close releases any resources (currently a no-op).
func (p *Publisher[T]) Close() error {
	return nil
}
