package stepper_test

import (
	"context"
	"testing"

	"github.com/gorai/gorai/component"
	"github.com/gorai/gorai/component/stepper"
	"github.com/gorai/gorai/component/stepper/fake"
	"github.com/gorai/gorai/pkg/resource"
)

func TestStepper_IsActuator(t *testing.T) {
	var _ component.Actuator = (stepper.Stepper)(nil)
}

func TestFakeStepper_Step(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "stepper", "test")
	s := fake.NewWithName(name)

	if err := s.Step(ctx, 100); err != nil {
		t.Fatalf("Step failed: %v", err)
	}

	pos, err := s.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 100 {
		t.Errorf("position = %v, want 100", pos)
	}

	// Step more
	if err := s.Step(ctx, 50); err != nil {
		t.Fatalf("Step failed: %v", err)
	}

	pos, err = s.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 150 {
		t.Errorf("position = %v, want 150", pos)
	}

	// Step backwards
	if err := s.Step(ctx, -200); err != nil {
		t.Fatalf("Step failed: %v", err)
	}

	pos, err = s.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != -50 {
		t.Errorf("position = %v, want -50", pos)
	}
}

func TestFakeStepper_SetMicrostepping(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "stepper", "test")
	s := fake.NewWithName(name)

	// Valid divisors
	for _, div := range []int{1, 2, 4, 8, 16, 32, 64, 128, 256} {
		if err := s.SetMicrostepping(ctx, div); err != nil {
			t.Errorf("SetMicrostepping(%d) failed: %v", div, err)
		}
	}

	// Invalid divisor (not power of 2)
	err := s.SetMicrostepping(ctx, 3)
	if err == nil {
		t.Error("expected error for non-power-of-2 divisor")
	}

	// Invalid divisor (too large)
	err = s.SetMicrostepping(ctx, 512)
	if err == nil {
		t.Error("expected error for divisor > max")
	}
}

func TestFakeStepper_SetCurrent(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "stepper", "test")
	s := fake.NewWithName(name)

	if err := s.SetCurrent(ctx, 1500, 500); err != nil {
		t.Fatalf("SetCurrent failed: %v", err)
	}

	// Invalid current (too high)
	err := s.SetCurrent(ctx, 3000, 500)
	if err == nil {
		t.Error("expected error for current > max")
	}

	// Invalid current (negative)
	err = s.SetCurrent(ctx, -100, 500)
	if err == nil {
		t.Error("expected error for negative current")
	}
}

func TestFakeStepper_ResetPosition(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "stepper", "test")
	s := fake.NewWithName(name)

	s.SetPosition(1000)

	if err := s.ResetPosition(ctx); err != nil {
		t.Fatalf("ResetPosition failed: %v", err)
	}

	pos, err := s.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 0 {
		t.Errorf("position after reset = %v, want 0", pos)
	}
}

func TestFakeStepper_Home(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "stepper", "test")
	s := fake.NewWithName(name)

	s.SetPosition(5000)

	if err := s.Home(ctx, false); err != nil {
		t.Fatalf("Home failed: %v", err)
	}

	pos, err := s.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 0 {
		t.Errorf("position after home = %v, want 0", pos)
	}
}

func TestFakeStepper_Properties(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "stepper", "test")
	s := fake.NewWithName(name)

	props, err := s.GetProperties(ctx)
	if err != nil {
		t.Fatalf("GetProperties failed: %v", err)
	}

	if props.StepsPerRevolution != 200 {
		t.Errorf("StepsPerRevolution = %v, want 200", props.StepsPerRevolution)
	}
	if props.MaxMicrostepping != 256 {
		t.Errorf("MaxMicrostepping = %v, want 256", props.MaxMicrostepping)
	}
	if !props.HasStallDetection {
		t.Error("expected HasStallDetection to be true")
	}
	if props.Driver != "tmc2209" {
		t.Errorf("Driver = %q, want 'tmc2209'", props.Driver)
	}
}

func TestFakeStepper_Stop(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "stepper", "test")
	s := fake.NewWithName(name)

	s.SetMoving(true)

	if err := s.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	moving, err := s.IsMoving(ctx)
	if err != nil {
		t.Fatalf("IsMoving failed: %v", err)
	}
	if moving {
		t.Error("expected stepper to not be moving after Stop")
	}
}

func TestFakeStepper_Name(t *testing.T) {
	name := resource.NewComponentName("gorai", "stepper", "test_stepper")
	s := fake.NewWithName(name)

	if s.Name().String() != "gorai:component:stepper/test_stepper" {
		t.Errorf("Name() = %q, want 'gorai:component:stepper/test_stepper'", s.Name().String())
	}
}
