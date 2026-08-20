package tankdrive

import (
	"context"
	"testing"
	"time"

	pwmfake "github.com/emergingrobotics/gorai/components/pwm/fake"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// driveTestDeps resolves the left/right fake PWMs by name.
type driveTestDeps struct {
	left  any
	right any
}

func (d driveTestDeps) Get(name string) (any, error) {
	switch name {
	case "left_thruster":
		return d.left, nil
	case "right_thruster":
		return d.right, nil
	}
	return nil, nil
}
func (d driveTestDeps) GetByType(subtype string) ([]any, error) { return nil, nil }

// newTestController builds a controller wired to two fake PWMs, with the given
// config overrides merged over defaults.
func newTestController(t *testing.T, overrides registry.Config) (*Controller, *pwmfake.PWM, *pwmfake.PWM) {
	t.Helper()
	left := pwmfake.NewWithName(resource.NewComponentName("gorai", "pwm", "left_thruster"))
	right := pwmfake.NewWithName(resource.NewComponentName("gorai", "pwm", "right_thruster"))

	conf := registry.Config{"name": "drive"}
	for k, v := range overrides {
		conf[k] = v
	}

	c, err := New(context.Background(), driveTestDeps{left: left, right: right}, conf)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	return c.(*Controller), left, right
}

func TestTankDriveMixing(t *testing.T) {
	// slew_per_s 0 => instant, so one tick reaches the target.
	c, left, right := newTestController(t, registry.Config{"slew_per_s": 0.0})

	// Forward + starboard turn.
	if err := c.SetIntent(context.Background(), 0.5, 0.5); err != nil {
		t.Fatalf("SetIntent: %v", err)
	}
	c.tick(context.Background(), 0.02)

	if got := left.SetNormalizedCalls[len(left.SetNormalizedCalls)-1]; got != 1.0 {
		t.Errorf("left: want 1.0, got %v", got)
	}
	if got := right.SetNormalizedCalls[len(right.SetNormalizedCalls)-1]; got != 0.0 {
		t.Errorf("right: want 0.0, got %v", got)
	}
}

func TestTankDriveDeadband(t *testing.T) {
	c, left, right := newTestController(t, registry.Config{"slew_per_s": 0.0, "deadband": 0.1})
	c.SetIntent(context.Background(), 0.05, 0.05) // below deadband
	c.tick(context.Background(), 0.02)

	if got := left.SetNormalizedCalls[len(left.SetNormalizedCalls)-1]; got != 0.0 {
		t.Errorf("left within deadband: want 0.0, got %v", got)
	}
	if got := right.SetNormalizedCalls[len(right.SetNormalizedCalls)-1]; got != 0.0 {
		t.Errorf("right within deadband: want 0.0, got %v", got)
	}
}

func TestTankDriveSlewLimiting(t *testing.T) {
	// slew_per_s=1 => max delta per tick = 1 * dt.
	c, left, _ := newTestController(t, registry.Config{"slew_per_s": 1.0})
	c.SetIntent(context.Background(), 1.0, 0.0)

	c.tick(context.Background(), 0.1) // max delta 0.1
	if got := left.SetNormalizedCalls[len(left.SetNormalizedCalls)-1]; got < 0.09 || got > 0.11 {
		t.Errorf("after one slewed tick want ~0.1, got %v", got)
	}
	c.tick(context.Background(), 0.1)
	if got := left.SetNormalizedCalls[len(left.SetNormalizedCalls)-1]; got < 0.19 || got > 0.21 {
		t.Errorf("after two slewed ticks want ~0.2, got %v", got)
	}
}

func TestTankDriveFailsafe(t *testing.T) {
	c, left, right := newTestController(t, registry.Config{"slew_per_s": 0.0, "command_timeout_ms": 100})
	c.SetIntent(context.Background(), 1.0, 0.0)
	c.tick(context.Background(), 0.02)
	if got := left.SetNormalizedCalls[len(left.SetNormalizedCalls)-1]; got != 1.0 {
		t.Fatalf("expected full command before timeout, got %v", got)
	}

	// Simulate command staleness.
	c.mu.Lock()
	c.lastCmd = time.Now().Add(-time.Second)
	c.mu.Unlock()
	c.tick(context.Background(), 0.02)

	if got := left.SetNormalizedCalls[len(left.SetNormalizedCalls)-1]; got != 0.0 {
		t.Errorf("failsafe left: want 0.0, got %v", got)
	}
	if got := right.SetNormalizedCalls[len(right.SetNormalizedCalls)-1]; got != 0.0 {
		t.Errorf("failsafe right: want 0.0, got %v", got)
	}
	st, _ := c.State(context.Background())
	if st.Active {
		t.Errorf("expected Active=false during failsafe")
	}
}

func TestTankDriveArm(t *testing.T) {
	c, left, right := newTestController(t, nil)
	if err := c.Arm(context.Background()); err != nil {
		t.Fatalf("Arm: %v", err)
	}
	if left.ArmCalls != 1 || right.ArmCalls != 1 {
		t.Errorf("expected both thrusters armed, got left=%d right=%d", left.ArmCalls, right.ArmCalls)
	}
	st, _ := c.State(context.Background())
	if !st.Armed {
		t.Errorf("expected Armed=true after Arm")
	}
}

func TestTankDriveInvert(t *testing.T) {
	c, left, right := newTestController(t, registry.Config{"slew_per_s": 0.0, "invert_right": true})
	c.SetIntent(context.Background(), 1.0, 0.0)
	c.tick(context.Background(), 0.02)
	if got := left.SetNormalizedCalls[len(left.SetNormalizedCalls)-1]; got != 1.0 {
		t.Errorf("left: want 1.0, got %v", got)
	}
	if got := right.SetNormalizedCalls[len(right.SetNormalizedCalls)-1]; got != -1.0 {
		t.Errorf("inverted right: want -1.0, got %v", got)
	}
}
