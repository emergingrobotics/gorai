package robot

import (
	"testing"
)

func TestComponentDeps_GetNatsAndLogger(t *testing.T) {
	deps := newComponentDeps(nil, nil)
	_, err := deps.Get("nats")
	if err != nil {
		t.Fatalf("expected nats to be available: %v", err)
	}
	_, err = deps.Get("logger")
	if err != nil {
		t.Fatalf("expected logger to be available: %v", err)
	}
}

func TestComponentDeps_GetCreatedComponent(t *testing.T) {
	deps := newComponentDeps(nil, nil)
	deps.Add("mcu", "fake-mcu-instance")

	val, err := deps.Get("mcu")
	if err != nil {
		t.Fatalf("expected mcu to be available: %v", err)
	}
	if val != "fake-mcu-instance" {
		t.Fatalf("expected fake-mcu-instance, got %v", val)
	}
}

func TestComponentDeps_GetMissing(t *testing.T) {
	deps := newComponentDeps(nil, nil)
	_, err := deps.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}
