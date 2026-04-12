package componentregistry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	os.WriteFile(path, []byte(`{
		"version": "1",
		"components": {
			"sensor/hc-sr04": {
				"module": "github.com/emergingrobotics/gorai-driver-hcsr04",
				"type": "sensor",
				"model": "hc-sr04",
				"description": "HC-SR04 ultrasonic",
				"version": "v0.1.0",
				"tags": ["ultrasonic", "gpio"]
			}
		}
	}`), 0644)

	reg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reg.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(reg.Components))
	}
	c := reg.Components["sensor/hc-sr04"]
	if c.Module != "github.com/emergingrobotics/gorai-driver-hcsr04" {
		t.Fatalf("wrong module: %s", c.Module)
	}
}

func TestSearch(t *testing.T) {
	reg := &Registry{
		Components: map[string]Component{
			"sensor/hc-sr04":  {Type: "sensor", Model: "hc-sr04", Description: "HC-SR04 ultrasonic", Tags: []string{"ultrasonic"}},
			"servo/dynamixel": {Type: "servo", Model: "dynamixel", Description: "Dynamixel smart servo", Tags: []string{"serial"}},
		},
	}
	results := reg.Search("ultrasonic")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Model != "hc-sr04" {
		t.Fatalf("wrong result: %s", results[0].Model)
	}
}

func TestSearchByType(t *testing.T) {
	reg := &Registry{
		Components: map[string]Component{
			"sensor/hc-sr04":  {Type: "sensor", Model: "hc-sr04", Description: "ultrasonic"},
			"sensor/bno055":   {Type: "sensor", Model: "bno055", Description: "IMU"},
			"servo/dynamixel": {Type: "servo", Model: "dynamixel", Description: "servo"},
		},
	}
	results := reg.Search("sensor")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestLookup(t *testing.T) {
	reg := &Registry{
		Components: map[string]Component{
			"sensor/hc-sr04": {Module: "github.com/example/hcsr04", Type: "sensor", Model: "hc-sr04"},
		},
	}
	c, ok := reg.Lookup("sensor/hc-sr04")
	if !ok {
		t.Fatal("expected to find sensor/hc-sr04")
	}
	if c.Module != "github.com/example/hcsr04" {
		t.Fatalf("wrong module: %s", c.Module)
	}

	_, ok = reg.Lookup("sensor/nonexistent")
	if ok {
		t.Fatal("expected not to find sensor/nonexistent")
	}
}
