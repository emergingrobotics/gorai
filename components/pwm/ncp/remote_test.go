package ncp

import (
	"context"
	"testing"
	"time"

	pwmfake "github.com/emergingrobotics/gorai/components/pwm/fake"
	"github.com/emergingrobotics/gorai/pkg/embeddednats"
	"github.com/emergingrobotics/gorai/pkg/ncp"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/emergingrobotics/gorai/pkg/subjects"
	"github.com/nats-io/nats.go"
)

// testDeps provides a nats and logger dependency for component construction.
type testDeps struct{ nc *nats.Conn }

func (d testDeps) Get(name string) (any, error) {
	if name == "nats" {
		return d.nc, nil
	}
	return nil, nil
}
func (d testDeps) GetByType(subtype string) ([]any, error) { return nil, nil }

func startNATS(t *testing.T) *nats.Conn {
	t.Helper()
	srv, err := embeddednats.New(embeddednats.Config{Host: "127.0.0.1", Port: -1})
	if err != nil {
		t.Fatalf("embedded nats: %v", err)
	}
	if err := srv.Start(); err != nil {
		t.Fatalf("start nats: %v", err)
	}
	t.Cleanup(srv.Shutdown)
	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(nc.Close)
	return nc
}

// TestRemotePWMDrivesServer verifies the full ground-station -> surf path:
// a pwm/ncp client SetPulse is delivered to a server-exposed fake PWM.
func TestRemotePWMDrivesServer(t *testing.T) {
	nc := startNATS(t)
	ctx := context.Background()

	// surf side: expose a fake PWM as "thruster".
	fake := pwmfake.NewWithName(resource.NewComponentName("gorai", "pwm", "thruster"))
	srv := ncp.NewServer(nc, subjects.NewBuilder("surf"), nil)
	t.Cleanup(func() { _ = srv.Close() })
	if _, err := srv.Expose("thruster", fake); err != nil {
		t.Fatalf("expose: %v", err)
	}

	// ground-station side: build the pwm/ncp client.
	conf := registry.Config{
		"name":      "thruster",
		"robot":     "surf",
		"component": "thruster",
	}
	compAny, err := New(ctx, testDeps{nc: nc}, conf)
	if err != nil {
		t.Fatalf("New client: %v", err)
	}
	client := compAny.(*RemotePWM)
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Start client: %v", err)
	}

	if err := client.SetPulse(ctx, 1600); err != nil {
		t.Fatalf("SetPulse: %v", err)
	}
	if len(fake.SetPulseCalls) != 1 || fake.SetPulseCalls[0] != 1600 {
		t.Errorf("expected remote SetPulse(1600), got %v", fake.SetPulseCalls)
	}
	if got, _ := client.GetPulse(ctx); got != 1600 {
		t.Errorf("expected cached pulse 1600, got %f", got)
	}

	// SetNormalized(1.0) -> max pulse (2000).
	if err := client.SetNormalized(ctx, 1.0); err != nil {
		t.Fatalf("SetNormalized: %v", err)
	}
	if last := fake.SetPulseCalls[len(fake.SetPulseCalls)-1]; last != 2000 {
		t.Errorf("expected SetNormalized(1.0) -> 2000, got %f", last)
	}

	// Disable propagates.
	if err := client.Disable(ctx); err != nil {
		t.Fatalf("Disable: %v", err)
	}
	if fake.DisableCalls != 1 {
		t.Errorf("expected 1 remote Disable, got %d", fake.DisableCalls)
	}
}

// TestRemotePWMArm verifies an arm request travels client -> server -> fake.
func TestRemotePWMArm(t *testing.T) {
	nc := startNATS(t)
	ctx := context.Background()

	fake := pwmfake.NewWithName(resource.NewComponentName("gorai", "pwm", "thruster"))
	srv := ncp.NewServer(nc, subjects.NewBuilder("surf"), nil)
	t.Cleanup(func() { _ = srv.Close() })
	if _, err := srv.Expose("thruster", fake); err != nil {
		t.Fatalf("expose: %v", err)
	}

	compAny, err := New(ctx, testDeps{nc: nc}, registry.Config{
		"name": "thruster", "robot": "surf", "component": "thruster",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	client := compAny.(*RemotePWM)
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := client.Arm(ctx); err != nil {
		t.Fatalf("Arm: %v", err)
	}
	if fake.ArmCalls != 1 {
		t.Errorf("expected 1 remote Arm, got %d", fake.ArmCalls)
	}
}

func TestRemotePWMClampsLocally(t *testing.T) {
	nc := startNATS(t)
	ctx := context.Background()

	fake := pwmfake.NewWithName(resource.NewComponentName("gorai", "pwm", "thruster"))
	srv := ncp.NewServer(nc, subjects.NewBuilder("surf"), nil)
	t.Cleanup(func() { _ = srv.Close() })
	if _, err := srv.Expose("thruster", fake); err != nil {
		t.Fatalf("expose: %v", err)
	}

	compAny, err := New(ctx, testDeps{nc: nc}, registry.Config{
		"name": "thruster", "robot": "surf", "component": "thruster",
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	client := compAny.(*RemotePWM)
	if err := client.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if err := client.SetPulse(ctx, 5000); err != nil {
		t.Fatalf("SetPulse: %v", err)
	}
	if last := fake.SetPulseCalls[len(fake.SetPulseCalls)-1]; last != 2000 {
		t.Errorf("expected client clamp to 2000, remote got %f", last)
	}
}

func TestRemotePWMRequiresTarget(t *testing.T) {
	nc := startNATS(t)
	_, err := New(context.Background(), testDeps{nc: nc}, registry.Config{"name": "x"})
	if err == nil {
		t.Errorf("expected error when robot/component missing")
	}
	_ = time.Second
}
