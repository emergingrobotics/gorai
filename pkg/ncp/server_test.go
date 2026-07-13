package ncp

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	pwmfake "github.com/emergingrobotics/gorai/components/pwm/fake"
	"github.com/emergingrobotics/gorai/pkg/embeddednats"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/emergingrobotics/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// startTestNATS starts an ephemeral embedded NATS server and returns a
// connected client. The server and connection are cleaned up on test end.
func startTestNATS(t *testing.T) *nats.Conn {
	t.Helper()
	srv, err := embeddednats.New(embeddednats.Config{Host: "127.0.0.1", Port: -1, JetStream: false})
	if err != nil {
		t.Fatalf("failed to create embedded NATS: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("failed to start embedded NATS: %v", err)
	}
	t.Cleanup(srv.Shutdown)

	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("failed to connect to NATS: %v", err)
	}
	t.Cleanup(nc.Close)
	return nc
}

func TestServerExposesPWMCommand(t *testing.T) {
	nc := startTestNATS(t)
	builder := subjects.NewBuilder("surf")

	fake := pwmfake.NewWithName(resource.NewComponentName("gorai", "pwm", "thruster"))

	srv := NewServer(nc, builder, nil)
	t.Cleanup(func() { _ = srv.Close() })

	exposed, err := srv.Expose("thruster", fake)
	if err != nil {
		t.Fatalf("Expose failed: %v", err)
	}
	if !exposed {
		t.Fatalf("expected fake PWM to be exposed")
	}

	req := Request{Method: "set_pulse", Args: map[string]any{"pulse_us": 1600.0}}
	data, _ := json.Marshal(req)

	subject := builder.ComponentCommand("thruster")
	msg, err := nc.Request(subject, data, 2*time.Second)
	if err != nil {
		t.Fatalf("NCP request failed: %v", err)
	}

	var resp Response
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected ok response, got error: %s", resp.Error)
	}

	if len(fake.SetPulseCalls) != 1 || fake.SetPulseCalls[0] != 1600.0 {
		t.Errorf("expected SetPulse(1600), got calls %v", fake.SetPulseCalls)
	}
	if got, _ := fake.GetPulse(context.Background()); got != 1600.0 {
		t.Errorf("expected pulse 1600, got %f", got)
	}
	if resp.Result["pulse_us"] != 1600.0 {
		t.Errorf("expected result pulse_us 1600, got %v", resp.Result["pulse_us"])
	}
}

func TestServerArmCommand(t *testing.T) {
	nc := startTestNATS(t)
	builder := subjects.NewBuilder("surf")
	fake := pwmfake.NewWithName(resource.NewComponentName("gorai", "pwm", "thruster"))

	srv := NewServer(nc, builder, nil)
	t.Cleanup(func() { _ = srv.Close() })
	if _, err := srv.Expose("thruster", fake); err != nil {
		t.Fatalf("Expose failed: %v", err)
	}

	req := Request{Method: "arm"}
	data, _ := json.Marshal(req)
	msg, err := nc.Request(builder.ComponentCommand("thruster"), data, 2*time.Second)
	if err != nil {
		t.Fatalf("NCP arm request failed: %v", err)
	}
	var resp Response
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if !resp.OK {
		t.Fatalf("expected ok response, got error: %s", resp.Error)
	}
	if fake.ArmCalls != 1 {
		t.Errorf("expected 1 arm call, got %d", fake.ArmCalls)
	}
}

func TestServerUnknownMethod(t *testing.T) {
	nc := startTestNATS(t)
	builder := subjects.NewBuilder("surf")
	fake := pwmfake.NewWithName(resource.NewComponentName("gorai", "pwm", "thruster"))

	srv := NewServer(nc, builder, nil)
	t.Cleanup(func() { _ = srv.Close() })
	if _, err := srv.Expose("thruster", fake); err != nil {
		t.Fatalf("Expose failed: %v", err)
	}

	req := Request{Method: "bogus"}
	data, _ := json.Marshal(req)
	msg, err := nc.Request(builder.ComponentCommand("thruster"), data, 2*time.Second)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	var resp Response
	_ = json.Unmarshal(msg.Data, &resp)
	if resp.OK {
		t.Errorf("expected failure for unknown method")
	}
}

func TestServerStatePublished(t *testing.T) {
	nc := startTestNATS(t)
	builder := subjects.NewBuilder("surf")
	fake := pwmfake.NewWithName(resource.NewComponentName("gorai", "pwm", "thruster"))

	stateSub, err := nc.SubscribeSync(builder.ComponentState("thruster"))
	if err != nil {
		t.Fatalf("subscribe state failed: %v", err)
	}

	srv := NewServer(nc, builder, nil)
	t.Cleanup(func() { _ = srv.Close() })
	if _, err := srv.Expose("thruster", fake); err != nil {
		t.Fatalf("Expose failed: %v", err)
	}

	// Initial snapshot published on Expose.
	if _, err := stateSub.NextMsg(2 * time.Second); err != nil {
		t.Fatalf("expected initial state message: %v", err)
	}

	req := Request{Method: "set_pulse", Args: map[string]any{"pulse_us": 1700.0}}
	data, _ := json.Marshal(req)
	if _, err := nc.Request(builder.ComponentCommand("thruster"), data, 2*time.Second); err != nil {
		t.Fatalf("request failed: %v", err)
	}

	msg, err := stateSub.NextMsg(2 * time.Second)
	if err != nil {
		t.Fatalf("expected state after command: %v", err)
	}
	var state map[string]any
	_ = json.Unmarshal(msg.Data, &state)
	if state["pulse_us"] != 1700.0 {
		t.Errorf("expected published state pulse_us 1700, got %v", state["pulse_us"])
	}
}

func TestExposeNonCapabilityReturnsFalse(t *testing.T) {
	nc := startTestNATS(t)
	srv := NewServer(nc, subjects.NewBuilder("surf"), nil)
	t.Cleanup(func() { _ = srv.Close() })

	exposed, err := srv.Expose("random", struct{}{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exposed {
		t.Errorf("expected non-capability component to not be exposed")
	}
}
