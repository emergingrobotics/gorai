package fake

import (
	"context"
	"fmt"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("encoder", "fake", NewEncoder)
}

// Encoder is a fake encoder sensor for testing.
type Encoder struct {
	name resource.Name

	// Current state
	position float64
	velocity float64

	// Properties
	ticksPerRevolution int
	angleSupported     bool
	isAbsolute         bool
	resolution         int
}

// NewEncoder creates a new fake encoder.
func NewEncoder(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "encoder", nameStr)

	ticksPerRev := 4096
	if v, ok := conf["ticks_per_revolution"].(float64); ok {
		ticksPerRev = int(v)
	}

	return &Encoder{
		name:               name,
		ticksPerRevolution: ticksPerRev,
		angleSupported:     true,
		resolution:         ticksPerRev,
	}, nil
}

// NewEncoderWithName creates a fake encoder with a specific resource name.
func NewEncoderWithName(name resource.Name) *Encoder {
	return &Encoder{
		name:               name,
		ticksPerRevolution: 4096,
		angleSupported:     true,
		resolution:         4096,
	}
}

// Name returns the resource name.
func (e *Encoder) Name() resource.Name {
	return e.name
}

// Reconfigure updates the configuration.
func (e *Encoder) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (e *Encoder) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "set_position":
			if pos, ok := cmd["position"].(float64); ok {
				e.position = pos
				return map[string]any{"status": "ok"}, nil
			}
		case "get_state":
			return map[string]any{
				"position": e.position,
				"velocity": e.velocity,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (e *Encoder) Close(ctx context.Context) error {
	return nil
}

// Readings returns all sensor readings as a map.
func (e *Encoder) Readings(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"position":   e.position,
		"velocity":   e.velocity,
		"resolution": e.resolution,
	}, nil
}

// Position returns the current position.
func (e *Encoder) Position(ctx context.Context) (float64, error) {
	return e.position, nil
}

// ResetPosition sets the current position as zero.
func (e *Encoder) ResetPosition(ctx context.Context) error {
	e.position = 0
	return nil
}

// Properties returns encoder properties.
func (e *Encoder) Properties(ctx context.Context) (sensor.EncoderProperties, error) {
	return sensor.EncoderProperties{
		TicksPerRevolution:    e.ticksPerRevolution,
		AngleDegreesSupported: e.angleSupported,
		IsAbsolute:            e.isAbsolute,
	}, nil
}

// GetVelocity returns the velocity.
func (e *Encoder) GetVelocity(ctx context.Context) (float64, error) {
	return e.velocity, nil
}

// GetResolution returns the encoder resolution.
func (e *Encoder) GetResolution(ctx context.Context) (int, error) {
	return e.resolution, nil
}

// SetPosition sets the position for testing.
func (e *Encoder) SetPosition(pos float64) {
	e.position = pos
}

// SetVelocity sets the velocity for testing.
func (e *Encoder) SetVelocity(vel float64) {
	e.velocity = vel
}

// SetProperties sets the encoder properties for testing.
func (e *Encoder) SetProperties(ticksPerRev int, angleSupported, isAbsolute bool) {
	e.ticksPerRevolution = ticksPerRev
	e.angleSupported = angleSupported
	e.isAbsolute = isAbsolute
	e.resolution = ticksPerRev
}

// Verify interface compliance.
var _ sensor.Encoder = (*Encoder)(nil)
