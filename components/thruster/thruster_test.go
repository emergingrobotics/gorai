package thruster_test

import (
	"context"
	"testing"

	"github.com/gorai/gorai/components"
	"github.com/gorai/gorai/components/thruster"
	"github.com/gorai/gorai/components/thruster/fake"
	"github.com/gorai/gorai/pkg/resource"
)

func TestThruster_IsActuator(t *testing.T) {
	var _ component.Actuator = (thruster.Thruster)(nil)
}

func TestFakeThruster_SetThrust(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thruster", "test")
	th := fake.NewWithName(name)

	if err := th.SetThrust(ctx, 0.5); err != nil {
		t.Fatalf("SetThrust failed: %v", err)
	}

	moving, err := th.IsMoving(ctx)
	if err != nil {
		t.Fatalf("IsMoving failed: %v", err)
	}
	if !moving {
		t.Error("expected thruster to be moving at 50% thrust")
	}
}

func TestFakeThruster_SetThrust_Deadband(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thruster", "test")
	th := fake.NewWithName(name)

	// Small thrust within deadband should result in stop
	if err := th.SetThrust(ctx, 0.03); err != nil {
		t.Fatalf("SetThrust failed: %v", err)
	}

	moving, err := th.IsMoving(ctx)
	if err != nil {
		t.Fatalf("IsMoving failed: %v", err)
	}
	if moving {
		t.Error("expected thruster to not be moving within deadband")
	}
}

func TestFakeThruster_SetThrust_Reverse(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thruster", "test")
	th := fake.NewWithName(name)

	if err := th.SetThrust(ctx, -0.5); err != nil {
		t.Fatalf("SetThrust failed: %v", err)
	}

	moving, err := th.IsMoving(ctx)
	if err != nil {
		t.Fatalf("IsMoving failed: %v", err)
	}
	if !moving {
		t.Error("expected thruster to be moving in reverse")
	}
}

func TestFakeThruster_SetThrust_OutOfRange(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thruster", "test")
	th := fake.NewWithName(name)

	err := th.SetThrust(ctx, 1.5)
	if err == nil {
		t.Error("expected error for thrust > 1.0")
	}

	err = th.SetThrust(ctx, -1.5)
	if err == nil {
		t.Error("expected error for thrust < -1.0")
	}
}

func TestFakeThruster_GetRPM(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thruster", "test")
	th := fake.NewWithName(name)

	if err := th.SetThrust(ctx, 0.5); err != nil {
		t.Fatalf("SetThrust failed: %v", err)
	}

	rpm, err := th.GetRPM(ctx)
	if err != nil {
		t.Fatalf("GetRPM failed: %v", err)
	}
	if rpm <= 0 {
		t.Errorf("rpm = %v, expected > 0 at 50%% thrust", rpm)
	}
}

func TestFakeThruster_GetTemperature(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thruster", "test")
	th := fake.NewWithName(name)

	temp, err := th.GetTemperature(ctx)
	if err != nil {
		t.Fatalf("GetTemperature failed: %v", err)
	}
	if temp != 25.0 {
		t.Errorf("temperature = %v, want 25.0 (ambient)", temp)
	}
}

func TestFakeThruster_GetCurrent(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thruster", "test")
	th := fake.NewWithName(name)

	if err := th.SetThrust(ctx, 1.0); err != nil {
		t.Fatalf("SetThrust failed: %v", err)
	}

	current, err := th.GetCurrent(ctx)
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if current <= 0 {
		t.Errorf("current = %v, expected > 0 at full thrust", current)
	}
}

func TestFakeThruster_Properties(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thruster", "test")
	th := fake.NewWithName(name)

	props, err := th.GetProperties(ctx)
	if err != nil {
		t.Fatalf("GetProperties failed: %v", err)
	}

	if props.MaxThrustForward <= 0 {
		t.Errorf("MaxThrustForward = %v, want > 0", props.MaxThrustForward)
	}
	if !props.IsBidirectional {
		t.Error("expected IsBidirectional to be true")
	}
	if !props.HasTelemetry {
		t.Error("expected HasTelemetry to be true")
	}
	if props.Protocol != "pwm" {
		t.Errorf("Protocol = %q, want 'pwm'", props.Protocol)
	}
}

func TestFakeThruster_Stop(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thruster", "test")
	th := fake.NewWithName(name)

	if err := th.SetThrust(ctx, 1.0); err != nil {
		t.Fatalf("SetThrust failed: %v", err)
	}

	if err := th.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	moving, err := th.IsMoving(ctx)
	if err != nil {
		t.Fatalf("IsMoving failed: %v", err)
	}
	if moving {
		t.Error("expected thruster to not be moving after Stop")
	}
}

func TestFakeThruster_Name(t *testing.T) {
	name := resource.NewComponentName("gorai", "thruster", "test_thruster")
	th := fake.NewWithName(name)

	if th.Name().String() != "gorai:component:thruster/test_thruster" {
		t.Errorf("Name() = %q, want 'gorai:component:thruster/test_thruster'", th.Name().String())
	}
}

func TestFakeThruster_Close(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thruster", "test")
	th := fake.NewWithName(name)

	if err := th.SetThrust(ctx, 1.0); err != nil {
		t.Fatalf("SetThrust failed: %v", err)
	}

	if err := th.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	moving, _ := th.IsMoving(ctx)
	if moving {
		t.Error("expected thruster to not be moving after Close")
	}
}
