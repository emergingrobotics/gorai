package space_test

import (
	"context"
	"testing"

	"github.com/gorai/gorai/components"
	"github.com/gorai/gorai/components/space"
	"github.com/gorai/gorai/components/space/fake"
	"github.com/gorai/gorai/pkg/resource"
)

func TestSpace_IsComponent(t *testing.T) {
	// Space must implement component.Component
	var _ component.Component = (space.Space)(nil)
}

func TestFakeSpace_GetVolume(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "test")
	s := fake.NewWithName(name)

	volume, err := s.GetVolume(ctx)
	if err != nil {
		t.Fatalf("GetVolume failed: %v", err)
	}
	if volume != 1.0 {
		t.Errorf("volume = %v, want 1.0", volume)
	}
}

func TestFakeSpace_GetBounds(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "test")
	s := fake.NewWithName(name)

	bounds, err := s.GetBounds(ctx)
	if err != nil {
		t.Fatalf("GetBounds failed: %v", err)
	}

	if bounds.MinX != 0 || bounds.MinY != 0 || bounds.MinZ != 0 {
		t.Errorf("min bounds = (%v, %v, %v), want (0, 0, 0)", bounds.MinX, bounds.MinY, bounds.MinZ)
	}
	if bounds.MaxX != 1 || bounds.MaxY != 1 || bounds.MaxZ != 1 {
		t.Errorf("max bounds = (%v, %v, %v), want (1, 1, 1)", bounds.MaxX, bounds.MaxY, bounds.MaxZ)
	}

	// Test custom bounds
	s.SetBounds(resource.Bounds{
		MinX: -1, MinY: -2, MinZ: -3,
		MaxX: 4, MaxY: 5, MaxZ: 6,
	})

	bounds, err = s.GetBounds(ctx)
	if err != nil {
		t.Fatalf("GetBounds failed: %v", err)
	}

	if bounds.MinX != -1 || bounds.MaxX != 4 {
		t.Errorf("bounds X = (%v, %v), want (-1, 4)", bounds.MinX, bounds.MaxX)
	}
}

func TestFakeSpace_GetContents(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "test")
	s := fake.NewWithName(name)

	// Initially empty
	contents, err := s.GetContents(ctx)
	if err != nil {
		t.Fatalf("GetContents failed: %v", err)
	}
	if len(contents) != 0 {
		t.Errorf("initial contents count = %d, want 0", len(contents))
	}

	// Add items
	if err := s.AddContent(ctx, "item1"); err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}
	if err := s.AddContent(ctx, "item2"); err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}

	contents, err = s.GetContents(ctx)
	if err != nil {
		t.Fatalf("GetContents failed: %v", err)
	}
	if len(contents) != 2 {
		t.Errorf("contents count = %d, want 2", len(contents))
	}
}

func TestFakeSpace_IsEmpty(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "test")
	s := fake.NewWithName(name)

	// Initially empty
	empty, err := s.IsEmpty(ctx)
	if err != nil {
		t.Fatalf("IsEmpty failed: %v", err)
	}
	if !empty {
		t.Error("expected empty initially")
	}

	// Add item
	if err := s.AddContent(ctx, "item1"); err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}

	empty, err = s.IsEmpty(ctx)
	if err != nil {
		t.Fatalf("IsEmpty failed: %v", err)
	}
	if empty {
		t.Error("expected not empty after adding item")
	}
}

func TestFakeSpace_AddRemoveContent(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "test")
	s := fake.NewWithName(name)

	// Add items
	if err := s.AddContent(ctx, "item1"); err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}
	if err := s.AddContent(ctx, "item2"); err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}

	// Try to add duplicate
	if err := s.AddContent(ctx, "item1"); err == nil {
		t.Error("expected error when adding duplicate item")
	}

	// Remove item
	if err := s.RemoveContent(ctx, "item1"); err != nil {
		t.Fatalf("RemoveContent failed: %v", err)
	}

	contents, _ := s.GetContents(ctx)
	if len(contents) != 1 {
		t.Errorf("contents count after remove = %d, want 1", len(contents))
	}

	// Try to remove non-existent item
	if err := s.RemoveContent(ctx, "nonexistent"); err == nil {
		t.Error("expected error when removing non-existent item")
	}
}

func TestFakeSpace_ClearContents(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "test")
	s := fake.NewWithName(name)

	// Add items
	if err := s.AddContent(ctx, "item1"); err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}
	if err := s.AddContent(ctx, "item2"); err != nil {
		t.Fatalf("AddContent failed: %v", err)
	}

	// Clear
	if err := s.ClearContents(ctx); err != nil {
		t.Fatalf("ClearContents failed: %v", err)
	}

	empty, _ := s.IsEmpty(ctx)
	if !empty {
		t.Error("expected empty after ClearContents")
	}
}

func TestFakeSpace_GetUsedVolume(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "test")
	s := fake.NewWithName(name)

	// Initially 0
	used, err := s.GetUsedVolume(ctx)
	if err != nil {
		t.Fatalf("GetUsedVolume failed: %v", err)
	}
	if used != 0 {
		t.Errorf("initial used volume = %v, want 0", used)
	}

	// Set used volume
	s.SetUsedVolume(0.5)
	used, err = s.GetUsedVolume(ctx)
	if err != nil {
		t.Fatalf("GetUsedVolume failed: %v", err)
	}
	if used != 0.5 {
		t.Errorf("used volume = %v, want 0.5", used)
	}
}

func TestFakeSpace_GetWeight(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "test")
	s := fake.NewWithName(name)

	// Initially 0
	weight, err := s.GetWeight(ctx)
	if err != nil {
		t.Fatalf("GetWeight failed: %v", err)
	}
	if weight != 0 {
		t.Errorf("initial weight = %v, want 0", weight)
	}

	// Set weight
	s.SetWeight(25.5)
	weight, err = s.GetWeight(ctx)
	if err != nil {
		t.Fatalf("GetWeight failed: %v", err)
	}
	if weight != 25.5 {
		t.Errorf("weight = %v, want 25.5", weight)
	}
}

func TestFakeSpace_IsFull(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "test")
	s := fake.NewWithName(name)

	// Initially not full
	full, err := s.IsFull(ctx)
	if err != nil {
		t.Fatalf("IsFull failed: %v", err)
	}
	if full {
		t.Error("expected not full initially")
	}

	// Fill by volume
	s.SetUsedVolume(1.0)
	full, err = s.IsFull(ctx)
	if err != nil {
		t.Fatalf("IsFull failed: %v", err)
	}
	if !full {
		t.Error("expected full when volume is at capacity")
	}

	// Reset and fill by weight
	s.SetUsedVolume(0)
	s.SetWeight(100.0)
	full, err = s.IsFull(ctx)
	if err != nil {
		t.Fatalf("IsFull failed: %v", err)
	}
	if !full {
		t.Error("expected full when weight is at capacity")
	}
}

func TestFakeSpace_GetProperties(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "cargo")
	s := fake.NewWithName(name)

	props, err := s.GetProperties(ctx)
	if err != nil {
		t.Fatalf("GetProperties failed: %v", err)
	}

	if props.Type != space.SpaceTypeContainer {
		t.Errorf("type = %v, want SpaceTypeContainer", props.Type)
	}
	if props.Name != "cargo" {
		t.Errorf("name = %q, want 'cargo'", props.Name)
	}
	if props.MaxVolume != 1.0 {
		t.Errorf("max volume = %v, want 1.0", props.MaxVolume)
	}
	if props.MaxWeight != 100.0 {
		t.Errorf("max weight = %v, want 100.0", props.MaxWeight)
	}
	if !props.CanTrackContents {
		t.Error("expected CanTrackContents to be true")
	}
	if !props.CanMeasureVolume {
		t.Error("expected CanMeasureVolume to be true")
	}
}

func TestFakeSpace_Name(t *testing.T) {
	name := resource.NewComponentName("gorai", "space", "cargo_bay")
	s := fake.NewWithName(name)

	if s.Name().String() != "gorai:component:space/cargo_bay" {
		t.Errorf("Name() = %q, want 'gorai:component:space/cargo_bay'", s.Name().String())
	}
}

func TestFakeSpace_DoCommand(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "test")
	s := fake.NewWithName(name)

	// Test get_state command
	result, err := s.DoCommand(ctx, map[string]any{
		"command": "get_state",
	})
	if err != nil {
		t.Fatalf("DoCommand get_state failed: %v", err)
	}

	if result["volume"].(float64) != 1.0 {
		t.Errorf("volume = %v, want 1.0", result["volume"])
	}

	// Test add_content command
	_, err = s.DoCommand(ctx, map[string]any{
		"command": "add_content",
		"id":      "test_item",
	})
	if err != nil {
		t.Fatalf("DoCommand add_content failed: %v", err)
	}

	contents, _ := s.GetContents(ctx)
	if len(contents) != 1 {
		t.Errorf("contents count = %d, want 1", len(contents))
	}

	// Test remove_content command
	_, err = s.DoCommand(ctx, map[string]any{
		"command": "remove_content",
		"id":      "test_item",
	})
	if err != nil {
		t.Fatalf("DoCommand remove_content failed: %v", err)
	}

	contents, _ = s.GetContents(ctx)
	if len(contents) != 0 {
		t.Errorf("contents count after remove = %d, want 0", len(contents))
	}

	// Test clear_contents command
	s.AddContent(ctx, "item1")
	s.AddContent(ctx, "item2")
	_, err = s.DoCommand(ctx, map[string]any{
		"command": "clear_contents",
	})
	if err != nil {
		t.Fatalf("DoCommand clear_contents failed: %v", err)
	}

	empty, _ := s.IsEmpty(ctx)
	if !empty {
		t.Error("expected empty after clear_contents")
	}

	// Test unknown command
	_, err = s.DoCommand(ctx, map[string]any{
		"command": "unknown",
	})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestFakeSpace_Close(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "test")
	s := fake.NewWithName(name)

	if err := s.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestSpaceType_String(t *testing.T) {
	tests := []struct {
		st   space.SpaceType
		want string
	}{
		{space.SpaceTypeContainer, "container"},
		{space.SpaceTypeTank, "tank"},
		{space.SpaceTypeCompartment, "compartment"},
		{space.SpaceType(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.st.String()
		if got != tt.want {
			t.Errorf("SpaceType(%d).String() = %q, want %q", tt.st, got, tt.want)
		}
	}
}

func TestFakeSpace_GetComponents(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "ballast_tank")
	s := fake.NewWithName(name)

	// Initially empty
	components, err := s.GetComponents(ctx)
	if err != nil {
		t.Fatalf("GetComponents failed: %v", err)
	}
	if len(components) != 0 {
		t.Errorf("initial components count = %d, want 0", len(components))
	}

	// Add components (valves and sensors that would control/monitor this space)
	fillValve := resource.NewComponentName("gorai", "actuator", "fill_valve")
	drainValve := resource.NewComponentName("gorai", "actuator", "drain_valve")
	levelSensor := resource.NewComponentName("gorai", "sensor", "level_sensor")

	s.AddComponent(fillValve)
	s.AddComponent(drainValve)
	s.AddComponent(levelSensor)

	components, err = s.GetComponents(ctx)
	if err != nil {
		t.Fatalf("GetComponents failed: %v", err)
	}
	if len(components) != 3 {
		t.Errorf("components count = %d, want 3", len(components))
	}

	// Verify component names
	found := map[string]bool{}
	for _, c := range components {
		found[c.Name] = true
	}
	if !found["fill_valve"] || !found["drain_valve"] || !found["level_sensor"] {
		t.Error("expected to find fill_valve, drain_valve, and level_sensor")
	}
}

func TestFakeSpace_SetComponents(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "cargo_bay")
	s := fake.NewWithName(name)

	// Set components directly
	doorServo := resource.NewComponentName("gorai", "actuator", "door_servo")
	presenceSensor := resource.NewComponentName("gorai", "sensor", "presence_sensor")

	s.SetComponents([]resource.Name{doorServo, presenceSensor})

	components, err := s.GetComponents(ctx)
	if err != nil {
		t.Fatalf("GetComponents failed: %v", err)
	}
	if len(components) != 2 {
		t.Errorf("components count = %d, want 2", len(components))
	}
}

func TestFakeSpace_GetProperties_WithComponents(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "space", "ballast_tank")
	s := fake.NewWithName(name)
	s.SetType(space.SpaceTypeTank)

	// Add components
	fillValve := resource.NewComponentName("gorai", "actuator", "fill_valve")
	s.AddComponent(fillValve)

	props, err := s.GetProperties(ctx)
	if err != nil {
		t.Fatalf("GetProperties failed: %v", err)
	}

	if props.Type != space.SpaceTypeTank {
		t.Errorf("type = %v, want SpaceTypeTank", props.Type)
	}
	if len(props.ComponentNames) != 1 {
		t.Errorf("component names count = %d, want 1", len(props.ComponentNames))
	}
	if props.ComponentNames[0].Name != "fill_valve" {
		t.Errorf("component name = %q, want 'fill_valve'", props.ComponentNames[0].Name)
	}
}
