package services

import (
	"sync"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
)

func TestExtractServiceName(t *testing.T) {
	tests := []struct {
		subject string
		want    string
	}{
		{"gorai.robot.suntimes.heartbeat", "suntimes"},
		{"gorai.robot.light-controller.heartbeat", "light-controller"},
		{"gorai.my-robot.svc.heartbeat", "svc"},
		{"gorai.robot.heartbeat", ""},
		{"gorai.robot", ""},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractServiceName(tt.subject)
		if got != tt.want {
			t.Errorf("extractServiceName(%q) = %q, want %q", tt.subject, got, tt.want)
		}
	}
}

func TestHandleHeartbeat_CachesServiceData(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	msg := &nats.Msg{
		Subject: "gorai.robot.suntimes.heartbeat",
		Data:    []byte(`{"service":"suntimes","status":"healthy","status_value":"\u219107:12 \u219318:03","status_value_type":"string","today_sunrise":"2026-02-22T07:12:00-08:00","today_sunset":"2026-02-22T18:03:00-08:00"}`),
	}
	m.handleHeartbeatMessage(msg)

	data, ok := m.GetServiceData("suntimes")
	if !ok {
		t.Fatal("expected suntimes data to be cached")
	}
	if data["today_sunrise"] != "2026-02-22T07:12:00-08:00" {
		t.Errorf("today_sunrise = %v", data["today_sunrise"])
	}
}

func TestHandleHeartbeat_IgnoresSystem(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	msg := &nats.Msg{
		Subject: "gorai.robot.system.heartbeat",
		Data:    []byte(`{"service":"system","status":"healthy"}`),
	}
	m.handleHeartbeatMessage(msg)

	_, ok := m.GetServiceData("system")
	if ok {
		t.Error("system heartbeats should be filtered out")
	}
}

func TestHandleHeartbeat_IgnoresNonServiceMessages(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	// Component data messages do not have a "service" field
	msg := &nats.Msg{
		Subject: "gorai.robot.plug_a.heartbeat",
		Data:    []byte(`{"status_value":"on","status_value_type":"binary"}`),
	}
	m.handleHeartbeatMessage(msg)

	_, ok := m.GetServiceData("plug_a")
	if ok {
		t.Error("component messages without 'service' field should be ignored")
	}
}

func TestOnStatusChange_Fires(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	var mu sync.Mutex
	var gotName string
	var gotData *ServiceData

	m.OnStatusChange(func(name string, sd *ServiceData) {
		mu.Lock()
		gotName = name
		gotData = sd
		mu.Unlock()
	})

	msg := &nats.Msg{
		Subject: "gorai.robot.suntimes.heartbeat",
		Data:    []byte(`{"service":"suntimes","status_value":"test","status_value_type":"string"}`),
	}
	m.handleHeartbeatMessage(msg)

	mu.Lock()
	defer mu.Unlock()

	if gotName != "suntimes" {
		t.Errorf("callback name = %q, want %q", gotName, "suntimes")
	}
	if gotData == nil {
		t.Fatal("callback got nil data")
	}
	if gotData.StatusValue != "test" {
		t.Errorf("status_value = %q, want %q", gotData.StatusValue, "test")
	}
}

func TestFormatStatusValue(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	got := m.FormatStatusValue("nonexistent")
	if got != "" {
		t.Errorf("FormatStatusValue for nonexistent = %q, want empty", got)
	}

	m.SetServiceData("suntimes", map[string]any{
		"service":           "suntimes",
		"status_value":      "\u219107:12 \u219318:03",
		"status_value_type": "string",
	})

	got = m.FormatStatusValue("suntimes")
	if got != "\u219107:12 \u219318:03" {
		t.Errorf("FormatStatusValue = %q", got)
	}
}

func TestFormatServiceDetail_Suntimes(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	m.SetServiceData("suntimes", map[string]any{
		"service":           "suntimes",
		"status_value":      "test",
		"status_value_type": "string",
		"today_sunrise":     "2026-02-22T07:12:00-08:00",
		"today_sunset":      "2026-02-22T18:03:00-08:00",
	})

	rows := m.FormatServiceDetail("suntimes")
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Label != "Sunrise" {
		t.Errorf("row[0].Label = %q, want Sunrise", rows[0].Label)
	}
	if rows[1].Label != "Sunset" {
		t.Errorf("row[1].Label = %q, want Sunset", rows[1].Label)
	}
}

func TestFormatServiceDetail_LightController(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	m.SetServiceData("light-controller", map[string]any{
		"service":           "light-controller",
		"status_value":      "2 schedules, 1 active",
		"status_value_type": "string",
		"schedules": []any{
			map[string]any{
				"name":     "pool-lights",
				"on_time":  "2026-02-22T17:33:00-08:00",
				"off_time": "2026-02-22T22:03:00-08:00",
				"active":   true,
				"devices":  []any{"pool_light_a"},
			},
			map[string]any{
				"name":     "bug-zappers",
				"on_time":  "2026-02-23T06:12:00-08:00",
				"off_time": "2026-02-23T08:42:00-08:00",
				"active":   false,
				"devices":  []any{"bug_zapper_a"},
			},
		},
	})

	rows := m.FormatServiceDetail("light-controller")
	if len(rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(rows))
	}
	if rows[0].Label != "pool-lights" {
		t.Errorf("row[0].Label = %q", rows[0].Label)
	}
	if rows[0].Status != "active" {
		t.Errorf("row[0].Status = %q, want active", rows[0].Status)
	}
	if rows[1].Status != "pending" {
		t.Errorf("row[1].Status = %q, want pending", rows[1].Status)
	}
}

func TestGetStatusValue(t *testing.T) {
	m := NewMonitor(nil, nil, nil)

	_, _, _, ok := m.GetStatusValue("nonexistent")
	if ok {
		t.Error("expected ok=false for nonexistent")
	}

	m.SetServiceData("suntimes", map[string]any{
		"service":           "suntimes",
		"status_value":      "test",
		"status_value_type": "string",
	})

	sv, svt, lastSeen, ok := m.GetStatusValue("suntimes")
	if !ok {
		t.Fatal("expected ok=true")
	}
	if sv != "test" {
		t.Errorf("status_value = %q", sv)
	}
	if svt != "string" {
		t.Errorf("status_value_type = %q", svt)
	}
	if time.Since(lastSeen) > time.Second {
		t.Error("lastSeen should be recent")
	}
}
