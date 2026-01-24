package pub_test

import (
	"testing"

	"github.com/gorai/gorai/api/gen/gorai/std"
	"github.com/gorai/gorai/pkg/pub"
	"github.com/nats-io/nats.go"
)

// mockNATSGetter implements pub.NATSGetter for testing
type mockNATSGetter struct {
	nc *nats.Conn
	js nats.JetStreamContext
}

func (m *mockNATSGetter) NATS() *nats.Conn {
	return m.nc
}

func (m *mockNATSGetter) JetStream() nats.JetStreamContext {
	return m.js
}

func TestQoS_String(t *testing.T) {
	tests := []struct {
		qos  pub.QoS
		want string
	}{
		{pub.BestEffort, "BestEffort"},
		{pub.Reliable, "Reliable"},
		{pub.Retained, "Retained"},
		{pub.History, "History"},
		{pub.QoS(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.qos.String(); got != tt.want {
				t.Errorf("QoS.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestPublisher_Topic(t *testing.T) {
	mock := &mockNATSGetter{}
	p := pub.New[*std.Header](mock, "test.topic")

	if p.Topic() != "test.topic" {
		t.Errorf("Topic() = %q, want 'test.topic'", p.Topic())
	}
}

func TestPublisher_QoS_Default(t *testing.T) {
	mock := &mockNATSGetter{}
	p := pub.New[*std.Header](mock, "test.topic")

	if p.QoS() != pub.BestEffort {
		t.Errorf("QoS() = %v, want BestEffort", p.QoS())
	}
}

func TestPublisher_QoS_WithOptions(t *testing.T) {
	mock := &mockNATSGetter{}

	tests := []struct {
		name string
		opts []pub.Option
		want pub.QoS
	}{
		{"reliable", []pub.Option{pub.WithQoS(pub.Reliable)}, pub.Reliable},
		{"retained", []pub.Option{pub.WithRetain()}, pub.Retained},
		{"history", []pub.Option{pub.WithHistory(10)}, pub.History},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := pub.New[*std.Header](mock, "test.topic", tt.opts...)
			if p.QoS() != tt.want {
				t.Errorf("QoS() = %v, want %v", p.QoS(), tt.want)
			}
		})
	}
}

func TestPublisher_Close(t *testing.T) {
	mock := &mockNATSGetter{}
	p := pub.New[*std.Header](mock, "test.topic")

	err := p.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}
}
