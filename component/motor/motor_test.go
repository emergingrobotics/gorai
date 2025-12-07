package motor_test

import (
	"context"
	"testing"

	"github.com/gorai/gorai/component"
	"github.com/gorai/gorai/component/motor"
	"github.com/gorai/gorai/component/motor/fake"
	"github.com/gorai/gorai/pkg/resource"
)

func TestMotor_IsActuator(t *testing.T) {
	// Motor must implement component.Actuator
	var _ component.Actuator = (motor.Motor)(nil)
}

func TestFakeMotor_SetPower(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "motor", "test")
	m := fake.NewWithName(name)

	if err := m.SetPower(ctx, 0.5); err != nil {
		t.Fatalf("SetPower failed: %v", err)
	}

	powered, power, err := m.IsPowered(ctx)
	if err != nil {
		t.Fatalf("IsPowered failed: %v", err)
	}
	if !powered {
		t.Error("expected motor to be powered")
	}
	if power != 0.5 {
		t.Errorf("power = %v, want 0.5", power)
	}
}

func TestFakeMotor_SetVelocity(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "motor", "test")
	m := fake.NewWithName(name)

	if err := m.SetVelocity(ctx, 10.5); err != nil {
		t.Fatalf("SetVelocity failed: %v", err)
	}

	velocity, err := m.GetVelocity(ctx)
	if err != nil {
		t.Fatalf("GetVelocity failed: %v", err)
	}
	if velocity != 10.5 {
		t.Errorf("velocity = %v, want 10.5", velocity)
	}

	moving, err := m.IsMoving(ctx)
	if err != nil {
		t.Fatalf("IsMoving failed: %v", err)
	}
	if !moving {
		t.Error("expected motor to be moving after SetVelocity")
	}
}

func TestFakeMotor_GoTo(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "motor", "test")
	m := fake.NewWithName(name)

	if err := m.GoTo(ctx, 100.0, 10.0); err != nil {
		t.Fatalf("GoTo failed: %v", err)
	}

	pos, err := m.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 100.0 {
		t.Errorf("position = %v, want 100.0", pos)
	}

	vel, err := m.GetVelocity(ctx)
	if err != nil {
		t.Fatalf("GetVelocity failed: %v", err)
	}
	if vel != 10.0 {
		t.Errorf("velocity = %v, want 10.0", vel)
	}
}

func TestFakeMotor_GoFor(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "motor", "test")
	m := fake.NewWithName(name)

	// Start at position 0
	if err := m.GoFor(ctx, 10.0, 5.0); err != nil {
		t.Fatalf("GoFor failed: %v", err)
	}

	pos, err := m.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 5.0 {
		t.Errorf("position = %v, want 5.0", pos)
	}

	// Add more revolutions
	if err := m.GoFor(ctx, 10.0, 3.0); err != nil {
		t.Fatalf("GoFor failed: %v", err)
	}

	pos, err = m.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 8.0 {
		t.Errorf("position = %v, want 8.0", pos)
	}
}

func TestFakeMotor_Stop(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "motor", "test")
	m := fake.NewWithName(name)

	// Set power and velocity first
	if err := m.SetPower(ctx, 1.0); err != nil {
		t.Fatalf("SetPower failed: %v", err)
	}
	if err := m.SetVelocity(ctx, 5.0); err != nil {
		t.Fatalf("SetVelocity failed: %v", err)
	}

	moving, err := m.IsMoving(ctx)
	if err != nil {
		t.Fatalf("IsMoving failed: %v", err)
	}
	if !moving {
		t.Error("expected motor to be moving after SetVelocity")
	}

	// Now stop
	if err := m.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	moving, err = m.IsMoving(ctx)
	if err != nil {
		t.Fatalf("IsMoving failed: %v", err)
	}
	if moving {
		t.Error("expected motor to not be moving after Stop")
	}

	powered, _, err := m.IsPowered(ctx)
	if err != nil {
		t.Fatalf("IsPowered failed: %v", err)
	}
	if powered {
		t.Error("expected motor to not be powered after Stop")
	}

	velocity, err := m.GetVelocity(ctx)
	if err != nil {
		t.Fatalf("GetVelocity failed: %v", err)
	}
	if velocity != 0 {
		t.Errorf("velocity after Stop = %v, want 0", velocity)
	}
}

func TestFakeMotor_ResetZeroPosition(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "motor", "test")
	m := fake.NewWithName(name)

	// Move to a position
	if err := m.GoTo(ctx, 50.0, 10.0); err != nil {
		t.Fatalf("GoTo failed: %v", err)
	}

	// Reset with offset
	if err := m.ResetZeroPosition(ctx, 10.0); err != nil {
		t.Fatalf("ResetZeroPosition failed: %v", err)
	}

	pos, err := m.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 10.0 {
		t.Errorf("position = %v, want 10.0 after reset", pos)
	}
}

func TestFakeMotor_Properties(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "motor", "test")
	m := fake.NewWithName(name)

	props, err := m.Properties(ctx)
	if err != nil {
		t.Fatalf("Properties failed: %v", err)
	}

	if !props.PositionReporting {
		t.Error("expected PositionReporting to be true for fake motor")
	}
	if !props.VelocityReporting {
		t.Error("expected VelocityReporting to be true for fake motor")
	}
	if !props.SupportsGoTo {
		t.Error("expected SupportsGoTo to be true for fake motor")
	}
}

func TestFakeMotor_Name(t *testing.T) {
	name := resource.NewComponentName("gorai", "motor", "test_motor")
	m := fake.NewWithName(name)

	if m.Name().String() != "gorai:component:motor/test_motor" {
		t.Errorf("Name() = %q, want 'gorai:component:motor/test_motor'", m.Name().String())
	}
}

func TestFakeMotor_DoCommand(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "motor", "test")
	m := fake.NewWithName(name)

	// Test get_state command
	result, err := m.DoCommand(ctx, map[string]any{
		"command": "get_state",
	})
	if err != nil {
		t.Fatalf("DoCommand get_state failed: %v", err)
	}

	if result["power"].(float64) != 0 {
		t.Errorf("initial power = %v, want 0", result["power"])
	}
	if result["velocity"].(float64) != 0 {
		t.Errorf("initial velocity = %v, want 0", result["velocity"])
	}

	// Test set_position command
	_, err = m.DoCommand(ctx, map[string]any{
		"command":  "set_position",
		"position": 42.0,
	})
	if err != nil {
		t.Fatalf("DoCommand set_position failed: %v", err)
	}

	pos, _ := m.GetPosition(ctx)
	if pos != 42.0 {
		t.Errorf("position after set_position = %v, want 42.0", pos)
	}

	// Test unknown command
	_, err = m.DoCommand(ctx, map[string]any{
		"command": "unknown",
	})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestFakeMotor_Close(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "motor", "test")
	m := fake.NewWithName(name)

	// Set power first
	if err := m.SetPower(ctx, 1.0); err != nil {
		t.Fatalf("SetPower failed: %v", err)
	}

	// Close should stop the motor
	if err := m.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	powered, _, _ := m.IsPowered(ctx)
	if powered {
		t.Error("expected motor to not be powered after Close")
	}
}
