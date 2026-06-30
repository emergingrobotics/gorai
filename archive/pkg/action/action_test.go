package action_test

import (
	"context"
	"testing"

	"github.com/emergingrobotics/gorai/api/gen/gorai/std"
	"github.com/emergingrobotics/gorai/pkg/action"
	"github.com/nats-io/nats.go"
)

// mockNATSGetter implements action.NATSGetter for testing
type mockNATSGetter struct {
	nc *nats.Conn
}

func (m *mockNATSGetter) NATS() *nats.Conn {
	return m.nc
}

func TestGoalState_String(t *testing.T) {
	tests := []struct {
		state action.GoalState
		want  string
	}{
		{action.GoalStatePending, "Pending"},
		{action.GoalStateActive, "Active"},
		{action.GoalStateSucceeded, "Succeeded"},
		{action.GoalStateAborted, "Aborted"},
		{action.GoalStateCanceled, "Canceled"},
		{action.GoalStateLost, "Lost"},
		{action.GoalState(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.state.String(); got != tt.want {
				t.Errorf("GoalState.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewServer_NoNATS(t *testing.T) {
	mock := &mockNATSGetter{nc: nil}

	// Using std.Header as a stand-in for goal, feedback, and result types
	_, err := action.NewServer[*std.Header, *std.Header, *std.Header](
		mock, "test_action",
		func(ctx context.Context, handle *action.GoalHandle[*std.Header, *std.Header, *std.Header]) {},
	)

	if err == nil {
		t.Error("expected error when no NATS connection")
	}
}

func TestNewClient_NoNATS(t *testing.T) {
	mock := &mockNATSGetter{nc: nil}

	_, err := action.NewClient[*std.Header, *std.Header, *std.Header](mock, "test_action")

	if err == nil {
		t.Error("expected error when no NATS connection")
	}
}

// Note: Full integration tests would require a running NATS server.
// These tests verify the basic error handling paths.
