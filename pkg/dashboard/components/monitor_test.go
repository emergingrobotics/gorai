package components

import (
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestExtractComponentName(t *testing.T) {
	tests := []struct {
		subject string
		want    string
	}{
		{"gorai.robot.plug_a.data", "plug_a"},
		{"gorai.robot.solar_inverter.data", "solar_inverter"},
		{"gorai.my-robot.cam1.data", "cam1"},
		{"gorai.robot.data", ""},
		{"gorai.robot", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractComponentName(tt.subject)
		if got != tt.want {
			t.Errorf("extractComponentName(%q) = %q, want %q", tt.subject, got, tt.want)
		}
	}
}

func TestOnStatusChange_FiresOnNewValue(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	var mu sync.Mutex
	var gotName string
	var gotValue *ComponentValue

	m.OnStatusChange(func(name string, cv *ComponentValue) {
		mu.Lock()
		gotName = name
		gotValue = cv
		mu.Unlock()
	})

	msg := &nats.Msg{
		Subject: "gorai.robot.plug_a.data",
		Data:    []byte(`{"status_value":"on","status_value_type":"binary"}`),
	}
	m.handleDataMessage(msg)

	mu.Lock()
	defer mu.Unlock()

	if gotName != "plug_a" {
		t.Errorf("callback got name %q, want %q", gotName, "plug_a")
	}
	if gotValue == nil {
		t.Fatal("callback got nil value")
	}
	if gotValue.ValueType != "binary" {
		t.Errorf("callback got value_type %q, want %q", gotValue.ValueType, "binary")
	}
}

func TestOnStatusChange_FiresOnValueChange(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	callCount := 0
	var lastValue any

	m.OnStatusChange(func(name string, cv *ComponentValue) {
		callCount++
		lastValue = cv.Value
	})

	m.handleDataMessage(&nats.Msg{
		Subject: "gorai.robot.plug_a.data",
		Data:    []byte(`{"status_value":"on","status_value_type":"binary"}`),
	})
	m.handleDataMessage(&nats.Msg{
		Subject: "gorai.robot.plug_a.data",
		Data:    []byte(`{"status_value":"off","status_value_type":"binary"}`),
	})

	if callCount != 2 {
		t.Errorf("callback called %d times, want 2", callCount)
	}
	if lastValue != "off" {
		t.Errorf("last value = %v, want %q", lastValue, "off")
	}
}

func TestOnStatusChange_NotFiredWithoutCallback(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	// Should not panic when no callback is set
	m.handleDataMessage(&nats.Msg{
		Subject: "gorai.robot.plug_a.data",
		Data:    []byte(`{"status_value":"on","status_value_type":"binary"}`),
	})

	val, _, _, _, ok := m.GetStatusValue("plug_a")
	if !ok {
		t.Fatal("expected value to be cached")
	}
	if val != "on" {
		t.Errorf("cached value = %v, want %q", val, "on")
	}
}

func TestFormatStatusValue_Binary(t *testing.T) {
	m := NewMonitor(nil, nil, nil)
	m.values["plug_a"] = &ComponentValue{Value: "on", ValueType: "binary", LastSeen: time.Now()}

	got := m.FormatStatusValue("plug_a")
	if got != "on" {
		t.Errorf("FormatStatusValue = %q, want %q", got, "on")
	}
}

func TestFormatStatusValue_Number(t *testing.T) {
	m := NewMonitor(nil, nil, nil)
	m.values["solar"] = &ComponentValue{Value: 1234.5, ValueType: "number", Unit: "W", LastSeen: time.Now()}

	got := m.FormatStatusValue("solar")
	if got != "1234.5 W" {
		t.Errorf("FormatStatusValue = %q, want %q", got, "1234.5 W")
	}
}

func TestFormatStatusValue_Missing(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	got := m.FormatStatusValue("nonexistent")
	if got != "" {
		t.Errorf("FormatStatusValue = %q, want empty", got)
	}
}

func TestGetStatusValue(t *testing.T) {
	m := NewMonitor(nil, nil, nil)
	now := time.Now()
	m.values["plug_a"] = &ComponentValue{Value: "off", ValueType: "binary", Unit: "", LastSeen: now}

	val, vtype, unit, lastSeen, ok := m.GetStatusValue("plug_a")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if val != "off" {
		t.Errorf("value = %v, want %q", val, "off")
	}
	if vtype != "binary" {
		t.Errorf("type = %q, want %q", vtype, "binary")
	}
	if unit != "" {
		t.Errorf("unit = %q, want empty", unit)
	}
	if !lastSeen.Equal(now) {
		t.Errorf("lastSeen mismatch")
	}
}
