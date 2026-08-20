// Package fake provides a fake pressure/temperature sensor for bench testing
// the telemetry stack without hardware.
package fake

import (
	"context"
	"fmt"
	"sync"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("sensor", "fake_pressure", New)
}

// Sensor is a fake pressure/temperature sensor.
type Sensor struct {
	name resource.Name

	mu          sync.RWMutex
	PressurePa  float64
	Temperature float64
	AltitudeM   float64
}

// New creates a fake pressure sensor with sea-level defaults.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	return &Sensor{
		name:        resource.NewComponentName("gorai", "sensor", nameStr),
		PressurePa:  101325.0,
		Temperature: 21.0,
		AltitudeM:   0.0,
	}, nil
}

// Name returns the resource name.
func (s *Sensor) Name() resource.Name {
	return s.name
}

// Reconfigure is a no-op.
func (s *Sensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand exposes a state query.
func (s *Sensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if name, _ := cmd["command"].(string); name == "get_state" {
		return s.Readings(ctx)
	}
	return nil, fmt.Errorf("unknown command: %v", cmd["command"])
}

// Close is a no-op.
func (s *Sensor) Close(ctx context.Context) error {
	return nil
}

// Readings returns the fake readings.
func (s *Sensor) Readings(ctx context.Context) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]any{
		"pressure_pa":   s.PressurePa,
		"temperature_c": s.Temperature,
		"altitude_m":    s.AltitudeM,
	}, nil
}

// Set updates the fake readings for testing.
func (s *Sensor) Set(pressurePa, temperatureC, altitudeM float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.PressurePa = pressurePa
	s.Temperature = temperatureC
	s.AltitudeM = altitudeM
}

// Verify interface compliance.
var _ sensor.Sensor = (*Sensor)(nil)
