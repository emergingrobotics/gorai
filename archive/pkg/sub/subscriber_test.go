package sub_test

import (
	"testing"

	"github.com/emergingrobotics/gorai/api/gen/gorai/std"
	"github.com/emergingrobotics/gorai/pkg/sub"
	"github.com/nats-io/nats.go"
)

// mockNATSGetter implements sub.NATSGetter for testing
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

func TestSubQoS_String(t *testing.T) {
	tests := []struct {
		qos  sub.QoS
		want string
	}{
		{sub.BestEffort, "BestEffort"},
		{sub.Reliable, "Reliable"},
		{sub.Durable, "Durable"},
		{sub.QoS(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.qos.String(); got != tt.want {
				t.Errorf("QoS.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSubscriber_NoNATS(t *testing.T) {
	mock := &mockNATSGetter{nc: nil}
	handler := func(msg *std.Header) {}

	_, err := sub.New[*std.Header](mock, "test.topic", handler)
	if err == nil {
		t.Error("expected error when no NATS connection")
	}
}
