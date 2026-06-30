package fake

import (
	"context"
	"fmt"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("presence_sensor", "fake", NewPresenceSensor)
}

// PresenceSensor is a fake presence sensor for testing.
type PresenceSensor struct {
	name resource.Name

	// Current state
	presenceDetected bool
	distance         float64
	motionState      sensor.MotionState
	supportsDistance bool
}

// NewPresenceSensor creates a new fake presence sensor.
func NewPresenceSensor(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "presence_sensor", nameStr)

	supportsDistance := false
	if v, ok := conf["supports_distance"].(bool); ok {
		supportsDistance = v
	}

	return &PresenceSensor{
		name:             name,
		motionState:      sensor.MotionUnknown,
		supportsDistance: supportsDistance,
	}, nil
}

// NewPresenceSensorWithName creates a fake presence sensor with a specific resource name.
func NewPresenceSensorWithName(name resource.Name) *PresenceSensor {
	return &PresenceSensor{
		name:        name,
		motionState: sensor.MotionUnknown,
	}
}

// Name returns the resource name.
func (p *PresenceSensor) Name() resource.Name {
	return p.name
}

// Reconfigure updates the configuration.
func (p *PresenceSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (p *PresenceSensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "set_presence":
			if detected, ok := cmd["detected"].(bool); ok {
				p.presenceDetected = detected
				return map[string]any{"status": "ok"}, nil
			}
		case "get_state":
			return map[string]any{
				"presence_detected": p.presenceDetected,
				"distance":          p.distance,
				"motion_state":      int(p.motionState),
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (p *PresenceSensor) Close(ctx context.Context) error {
	return nil
}

// Readings returns all sensor readings as a map.
func (p *PresenceSensor) Readings(ctx context.Context) (map[string]any, error) {
	readings := map[string]any{
		"presence_detected": p.presenceDetected,
		"motion_state":      int(p.motionState),
	}
	if p.supportsDistance {
		readings["distance"] = p.distance
	}
	return readings, nil
}

// IsPresenceDetected returns true if presence is detected.
func (p *PresenceSensor) IsPresenceDetected(ctx context.Context) (bool, error) {
	return p.presenceDetected, nil
}

// GetDistance returns distance to detected target in meters.
func (p *PresenceSensor) GetDistance(ctx context.Context) (float64, error) {
	if !p.supportsDistance {
		return 0, fmt.Errorf("distance measurement not supported")
	}
	return p.distance, nil
}

// GetMotionState returns the current motion state.
func (p *PresenceSensor) GetMotionState(ctx context.Context) (sensor.MotionState, error) {
	return p.motionState, nil
}

// SetPresence sets the presence state for testing.
func (p *PresenceSensor) SetPresence(detected bool) {
	p.presenceDetected = detected
}

// SetDistance sets the distance for testing.
func (p *PresenceSensor) SetDistance(distance float64) {
	p.distance = distance
	p.supportsDistance = true
}

// SetMotionState sets the motion state for testing.
func (p *PresenceSensor) SetMotionState(state sensor.MotionState) {
	p.motionState = state
}

// Verify interface compliance.
var _ sensor.PresenceSensor = (*PresenceSensor)(nil)
