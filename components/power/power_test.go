package power_test

import (
	"context"
	"testing"

	"github.com/emergingrobotics/gorai/components"
	"github.com/emergingrobotics/gorai/components/power"
	"github.com/emergingrobotics/gorai/components/power/fake"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func TestPower_IsComponent(t *testing.T) {
	// Power must implement component.Component
	var _ component.Component = (power.Power)(nil)
}

func TestFakePower_GetCapacity(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	capacity, err := p.GetCapacity(ctx)
	if err != nil {
		t.Fatalf("GetCapacity failed: %v", err)
	}
	if capacity != 100.0 {
		t.Errorf("capacity = %v, want 100.0", capacity)
	}
}

func TestFakePower_GetLevel(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	// Initial level should be 1.0 (fully charged)
	level, err := p.GetLevel(ctx)
	if err != nil {
		t.Fatalf("GetLevel failed: %v", err)
	}
	if level != 1.0 {
		t.Errorf("initial level = %v, want 1.0", level)
	}

	// Set level to 0.5
	p.SetLevel(0.5)
	level, err = p.GetLevel(ctx)
	if err != nil {
		t.Fatalf("GetLevel failed: %v", err)
	}
	if level != 0.5 {
		t.Errorf("level = %v, want 0.5", level)
	}
}

func TestFakePower_GetVoltage(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	voltage, err := p.GetVoltage(ctx)
	if err != nil {
		t.Fatalf("GetVoltage failed: %v", err)
	}
	if voltage != 12.0 {
		t.Errorf("voltage = %v, want 12.0", voltage)
	}
}

func TestFakePower_GetCurrent(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	// Initial current should be 0
	current, err := p.GetCurrent(ctx)
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if current != 0 {
		t.Errorf("initial current = %v, want 0", current)
	}

	// Set current
	p.SetCurrent(5.0)
	current, err = p.GetCurrent(ctx)
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if current != 5.0 {
		t.Errorf("current = %v, want 5.0", current)
	}
}

func TestFakePower_IsCharging(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	// Initially not charging
	charging, err := p.IsCharging(ctx)
	if err != nil {
		t.Fatalf("IsCharging failed: %v", err)
	}
	if charging {
		t.Error("expected not charging initially")
	}

	// Set charging
	p.SetCharging(true)
	charging, err = p.IsCharging(ctx)
	if err != nil {
		t.Fatalf("IsCharging failed: %v", err)
	}
	if !charging {
		t.Error("expected charging after SetCharging(true)")
	}
}

func TestFakePower_Status(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	// Initial status should be full
	status, err := p.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status != power.StatusFull {
		t.Errorf("initial status = %v, want StatusFull", status)
	}

	// Set level to 0.5 - should be good
	p.SetLevel(0.5)
	status, err = p.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status != power.StatusGood {
		t.Errorf("status at 0.5 = %v, want StatusGood", status)
	}

	// Set level to 0.15 - should be low
	p.SetLevel(0.15)
	status, err = p.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status != power.StatusLow {
		t.Errorf("status at 0.15 = %v, want StatusLow", status)
	}

	// Set level to 0.05 - should be critical
	p.SetLevel(0.05)
	status, err = p.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status != power.StatusCritical {
		t.Errorf("status at 0.05 = %v, want StatusCritical", status)
	}

	// Set charging - should be charging
	p.SetCharging(true)
	status, err = p.GetStatus(ctx)
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}
	if status != power.StatusCharging {
		t.Errorf("status while charging = %v, want StatusCharging", status)
	}
}

func TestFakePower_GetCellVoltages(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	cells, err := p.GetCellVoltages(ctx)
	if err != nil {
		t.Fatalf("GetCellVoltages failed: %v", err)
	}
	if len(cells) != 4 {
		t.Errorf("cell count = %d, want 4", len(cells))
	}
	for i, v := range cells {
		if v != 3.0 {
			t.Errorf("cell[%d] = %v, want 3.0", i, v)
		}
	}
}

func TestFakePower_GetTemperature(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	temp, err := p.GetTemperature(ctx)
	if err != nil {
		t.Fatalf("GetTemperature failed: %v", err)
	}
	if temp != 25.0 {
		t.Errorf("temperature = %v, want 25.0", temp)
	}

	p.SetTemperature(35.0)
	temp, err = p.GetTemperature(ctx)
	if err != nil {
		t.Fatalf("GetTemperature failed: %v", err)
	}
	if temp != 35.0 {
		t.Errorf("temperature = %v, want 35.0", temp)
	}
}

func TestFakePower_GetPower(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	// Set current to 2A
	p.SetCurrent(2.0)

	watts, err := p.GetPower(ctx)
	if err != nil {
		t.Fatalf("GetPower failed: %v", err)
	}
	// 12V * 2A = 24W
	if watts != 24.0 {
		t.Errorf("power = %v, want 24.0", watts)
	}
}

func TestFakePower_GetTimeRemaining(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	// No current draw - should return -1
	remaining, err := p.GetTimeRemaining(ctx)
	if err != nil {
		t.Fatalf("GetTimeRemaining failed: %v", err)
	}
	if remaining != -1 {
		t.Errorf("time remaining with no draw = %v, want -1", remaining)
	}

	// Set current draw
	p.SetCurrent(2.0) // 2A at 12V = 24W
	p.SetLevel(0.5)   // 50Wh remaining

	remaining, err = p.GetTimeRemaining(ctx)
	if err != nil {
		t.Fatalf("GetTimeRemaining failed: %v", err)
	}
	// 50Wh / 24W = 2.083 hours = 7500 seconds
	expected := 7500.0
	if remaining < expected-1 || remaining > expected+1 {
		t.Errorf("time remaining = %v, want ~%v", remaining, expected)
	}
}

func TestFakePower_GetCycleCount(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	cycles, err := p.GetCycleCount(ctx)
	if err != nil {
		t.Fatalf("GetCycleCount failed: %v", err)
	}
	if cycles != 0 {
		t.Errorf("initial cycles = %d, want 0", cycles)
	}

	p.IncrementCycles()
	p.IncrementCycles()

	cycles, err = p.GetCycleCount(ctx)
	if err != nil {
		t.Fatalf("GetCycleCount failed: %v", err)
	}
	if cycles != 2 {
		t.Errorf("cycles = %d, want 2", cycles)
	}
}

func TestFakePower_GetProperties(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	props, err := p.GetProperties(ctx)
	if err != nil {
		t.Fatalf("GetProperties failed: %v", err)
	}

	if props.Type != "battery" {
		t.Errorf("type = %q, want 'battery'", props.Type)
	}
	if props.NominalVoltage != 12.0 {
		t.Errorf("nominal voltage = %v, want 12.0", props.NominalVoltage)
	}
	if props.CellCount != 4 {
		t.Errorf("cell count = %d, want 4", props.CellCount)
	}
	if !props.SupportsCharging {
		t.Error("expected SupportsCharging to be true")
	}
	if !props.SupportsCellMonitoring {
		t.Error("expected SupportsCellMonitoring to be true")
	}
}

func TestFakePower_Name(t *testing.T) {
	name := resource.NewComponentName("gorai", "power", "main_battery")
	p := fake.NewWithName(name)

	if p.Name().String() != "gorai:component:power/main_battery" {
		t.Errorf("Name() = %q, want 'gorai:component:power/main_battery'", p.Name().String())
	}
}

func TestFakePower_DoCommand(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	// Test get_state command
	result, err := p.DoCommand(ctx, map[string]any{
		"command": "get_state",
	})
	if err != nil {
		t.Fatalf("DoCommand get_state failed: %v", err)
	}

	if result["capacity"].(float64) != 100.0 {
		t.Errorf("capacity = %v, want 100.0", result["capacity"])
	}
	if result["level"].(float64) != 1.0 {
		t.Errorf("level = %v, want 1.0", result["level"])
	}

	// Test set_level command
	_, err = p.DoCommand(ctx, map[string]any{
		"command": "set_level",
		"level":   0.75,
	})
	if err != nil {
		t.Fatalf("DoCommand set_level failed: %v", err)
	}

	level, _ := p.GetLevel(ctx)
	if level != 0.75 {
		t.Errorf("level after set_level = %v, want 0.75", level)
	}

	// Test set_charging command
	_, err = p.DoCommand(ctx, map[string]any{
		"command":  "set_charging",
		"charging": true,
	})
	if err != nil {
		t.Fatalf("DoCommand set_charging failed: %v", err)
	}

	charging, _ := p.IsCharging(ctx)
	if !charging {
		t.Error("expected charging after set_charging")
	}

	// Test unknown command
	_, err = p.DoCommand(ctx, map[string]any{
		"command": "unknown",
	})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestFakePower_Close(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "power", "test")
	p := fake.NewWithName(name)

	if err := p.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}

func TestStatus_String(t *testing.T) {
	tests := []struct {
		status power.Status
		want   string
	}{
		{power.StatusUnknown, "unknown"},
		{power.StatusGood, "good"},
		{power.StatusLow, "low"},
		{power.StatusCritical, "critical"},
		{power.StatusCharging, "charging"},
		{power.StatusFull, "full"},
		{power.StatusFault, "fault"},
		{power.Status(99), "unknown"},
	}

	for _, tt := range tests {
		got := tt.status.String()
		if got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}
