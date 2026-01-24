package servo_test

import (
	"context"
	"testing"

	"github.com/gorai/gorai/components"
	"github.com/gorai/gorai/components/servo"
	"github.com/gorai/gorai/components/servo/fake"
	"github.com/gorai/gorai/pkg/resource"
)

func TestServo_IsActuator(t *testing.T) {
	var _ component.Actuator = (servo.Servo)(nil)
}

func TestFakeServo_SetAngle(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "servo", "test")
	s := fake.NewWithName(name)

	if err := s.SetAngle(ctx, 45.0); err != nil {
		t.Fatalf("SetAngle failed: %v", err)
	}

	angle, err := s.GetAngle(ctx)
	if err != nil {
		t.Fatalf("GetAngle failed: %v", err)
	}
	if angle != 45.0 {
		t.Errorf("angle = %v, want 45.0", angle)
	}
}

func TestFakeServo_SetAngle_OutOfRange(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "servo", "test")
	s := fake.NewWithName(name)

	// Default range is -90 to +90
	err := s.SetAngle(ctx, 100.0)
	if err == nil {
		t.Error("expected error for out of range angle")
	}

	err = s.SetAngle(ctx, -100.0)
	if err == nil {
		t.Error("expected error for out of range angle")
	}
}

func TestFakeServo_SetSpeed(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "servo", "test")
	s := fake.NewWithName(name)

	if err := s.SetSpeed(ctx, 0.5); err != nil {
		t.Fatalf("SetSpeed failed: %v", err)
	}
}

func TestFakeServo_SetTorqueLimit(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "servo", "test")
	s := fake.NewWithName(name)

	if err := s.SetTorqueLimit(ctx, 0.8); err != nil {
		t.Fatalf("SetTorqueLimit failed: %v", err)
	}

	// Invalid range
	err := s.SetTorqueLimit(ctx, 1.5)
	if err == nil {
		t.Error("expected error for torque limit > 1.0")
	}

	err = s.SetTorqueLimit(ctx, -0.1)
	if err == nil {
		t.Error("expected error for torque limit < 0.0")
	}
}

func TestFakeServo_Properties(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "servo", "test")
	s := fake.NewWithName(name)

	props, err := s.GetProperties(ctx)
	if err != nil {
		t.Fatalf("GetProperties failed: %v", err)
	}

	if props.MinAngle != -90.0 {
		t.Errorf("MinAngle = %v, want -90.0", props.MinAngle)
	}
	if props.MaxAngle != 90.0 {
		t.Errorf("MaxAngle = %v, want 90.0", props.MaxAngle)
	}
	if props.Protocol != "pwm" {
		t.Errorf("Protocol = %q, want 'pwm'", props.Protocol)
	}
}

func TestFakeServo_Stop(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "servo", "test")
	s := fake.NewWithName(name)

	s.SetMoving(true)

	moving, err := s.IsMoving(ctx)
	if err != nil {
		t.Fatalf("IsMoving failed: %v", err)
	}
	if !moving {
		t.Error("expected servo to be moving")
	}

	if err := s.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	moving, err = s.IsMoving(ctx)
	if err != nil {
		t.Fatalf("IsMoving failed: %v", err)
	}
	if moving {
		t.Error("expected servo to not be moving after Stop")
	}
}

func TestFakeServo_Name(t *testing.T) {
	name := resource.NewComponentName("gorai", "servo", "test_servo")
	s := fake.NewWithName(name)

	if s.Name().String() != "gorai:component:servo/test_servo" {
		t.Errorf("Name() = %q, want 'gorai:component:servo/test_servo'", s.Name().String())
	}
}

func TestFakeServo_Close(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "servo", "test")
	s := fake.NewWithName(name)

	s.SetMoving(true)

	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	moving, _ := s.IsMoving(ctx)
	if moving {
		t.Error("expected servo to not be moving after Close")
	}
}
