package valve_test

import (
	"context"
	"testing"

	"github.com/gorai/gorai/component"
	"github.com/gorai/gorai/component/valve"
	"github.com/gorai/gorai/component/valve/fake"
	"github.com/gorai/gorai/pkg/resource"
)

func TestValve_IsActuator(t *testing.T) {
	var _ component.Actuator = (valve.Valve)(nil)
}

func TestFakeValve_Open(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "valve", "test")
	v := fake.NewWithName(name)

	if err := v.Open(ctx); err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	isOpen, err := v.IsOpen(ctx)
	if err != nil {
		t.Fatalf("IsOpen failed: %v", err)
	}
	if !isOpen {
		t.Error("expected valve to be open after Open()")
	}

	pos, err := v.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 1.0 {
		t.Errorf("position = %v, want 1.0", pos)
	}
}

func TestFakeValve_Shut(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "valve", "test")
	v := fake.NewWithName(name)

	// Open first
	if err := v.Open(ctx); err != nil {
		t.Fatalf("Open failed: %v", err)
	}

	// Then close
	if err := v.Shut(ctx); err != nil {
		t.Fatalf("Shut failed: %v", err)
	}

	isClosed, err := v.IsClosed(ctx)
	if err != nil {
		t.Fatalf("IsClosed failed: %v", err)
	}
	if !isClosed {
		t.Error("expected valve to be closed after Shut()")
	}

	pos, err := v.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 0.0 {
		t.Errorf("position = %v, want 0.0", pos)
	}
}

func TestFakeValve_SetPosition_Binary(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "valve", "test")
	v := fake.NewWithName(name)

	// Default is non-proportional (binary) valve
	// Values < 0.5 should close, >= 0.5 should open
	if err := v.SetPosition(ctx, 0.3); err != nil {
		t.Fatalf("SetPosition failed: %v", err)
	}

	pos, err := v.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 0.0 {
		t.Errorf("position = %v, want 0.0 (binary valve, input 0.3)", pos)
	}

	if err := v.SetPosition(ctx, 0.7); err != nil {
		t.Fatalf("SetPosition failed: %v", err)
	}

	pos, err = v.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 1.0 {
		t.Errorf("position = %v, want 1.0 (binary valve, input 0.7)", pos)
	}
}

func TestFakeValve_SetPosition_Proportional(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "valve", "test")
	v := fake.NewWithName(name)

	// Make it proportional
	v.SetProperties(true, true, false, 10.0, "ball")

	if err := v.SetPosition(ctx, 0.3); err != nil {
		t.Fatalf("SetPosition failed: %v", err)
	}

	pos, err := v.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 0.3 {
		t.Errorf("position = %v, want 0.3 (proportional valve)", pos)
	}

	if err := v.SetPosition(ctx, 0.7); err != nil {
		t.Fatalf("SetPosition failed: %v", err)
	}

	pos, err = v.GetPosition(ctx)
	if err != nil {
		t.Fatalf("GetPosition failed: %v", err)
	}
	if pos != 0.7 {
		t.Errorf("position = %v, want 0.7 (proportional valve)", pos)
	}
}

func TestFakeValve_SetPosition_OutOfRange(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "valve", "test")
	v := fake.NewWithName(name)

	err := v.SetPosition(ctx, 1.5)
	if err == nil {
		t.Error("expected error for position > 1.0")
	}

	err = v.SetPosition(ctx, -0.1)
	if err == nil {
		t.Error("expected error for position < 0.0")
	}
}

func TestFakeValve_IsOpen_IsClosed(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "valve", "test")
	v := fake.NewWithName(name)

	// Initially closed (position 0)
	isOpen, _ := v.IsOpen(ctx)
	isClosed, _ := v.IsClosed(ctx)

	if isOpen {
		t.Error("expected IsOpen to be false initially")
	}
	if !isClosed {
		t.Error("expected IsClosed to be true initially")
	}

	// Open the valve
	v.Open(ctx)

	isOpen, _ = v.IsOpen(ctx)
	isClosed, _ = v.IsClosed(ctx)

	if !isOpen {
		t.Error("expected IsOpen to be true after Open()")
	}
	if isClosed {
		t.Error("expected IsClosed to be false after Open()")
	}
}

func TestFakeValve_Stop(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "valve", "test")
	v := fake.NewWithName(name)

	v.SetMoving(true)

	if err := v.Stop(ctx); err != nil {
		t.Fatalf("Stop failed: %v", err)
	}

	moving, err := v.IsMoving(ctx)
	if err != nil {
		t.Fatalf("IsMoving failed: %v", err)
	}
	if moving {
		t.Error("expected valve to not be moving after Stop")
	}
}

func TestFakeValve_Name(t *testing.T) {
	name := resource.NewComponentName("gorai", "valve", "test_valve")
	v := fake.NewWithName(name)

	if v.Name().String() != "gorai:component:valve/test_valve" {
		t.Errorf("Name() = %q, want 'gorai:component:valve/test_valve'", v.Name().String())
	}
}

func TestFakeValve_Close(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "valve", "test")
	v := fake.NewWithName(name)

	v.SetMoving(true)

	// Note: Close() is the resource.Resource method (cleanup),
	// not the valve close operation (which is Shut())
	if err := v.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	moving, _ := v.IsMoving(ctx)
	if moving {
		t.Error("expected valve to not be moving after Close")
	}
}
