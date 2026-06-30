// Package fake provides a fake valve implementation for testing.
package fake

import (
	"context"
	"fmt"

	"github.com/emergingrobotics/gorai/components/valve"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("valve", "fake", New)
}

// Valve is a fake valve for testing.
type Valve struct {
	name resource.Name

	// Current state
	position float64
	moving   bool

	// Properties
	isProportional bool
	hasFeedback    bool
	normallyOpen   bool
	maxFlowRate    float64
	valveType      string
}

// New creates a new fake valve.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "valve", nameStr)

	isProportional := false
	if v, ok := conf["is_proportional"].(bool); ok {
		isProportional = v
	}

	normallyOpen := false
	if v, ok := conf["normally_open"].(bool); ok {
		normallyOpen = v
	}

	// Set initial position based on normally open/closed state
	initialPosition := 0.0
	if normallyOpen {
		initialPosition = 1.0
	}

	return &Valve{
		name:           name,
		position:       initialPosition,
		isProportional: isProportional,
		hasFeedback:    true,
		normallyOpen:   normallyOpen,
		maxFlowRate:    10.0, // L/min
		valveType:      "solenoid",
	}, nil
}

// NewWithName creates a fake valve with a specific resource name.
func NewWithName(name resource.Name) *Valve {
	return &Valve{
		name:           name,
		position:       0.0,
		isProportional: false,
		hasFeedback:    true,
		normallyOpen:   false,
		maxFlowRate:    10.0,
		valveType:      "solenoid",
	}
}

// Name returns the resource name.
func (v *Valve) Name() resource.Name {
	return v.name
}

// Reconfigure updates the configuration.
func (v *Valve) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (v *Valve) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_state":
			return map[string]any{
				"position": v.position,
				"is_open":  v.position >= 0.5,
				"moving":   v.moving,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources (stops valve motion, doesn't close the valve).
func (v *Valve) Close(ctx context.Context) error {
	return v.Stop(ctx)
}

// IsMoving returns whether the valve is moving.
func (v *Valve) IsMoving(ctx context.Context) (bool, error) {
	return v.moving, nil
}

// Stop stops valve movement.
func (v *Valve) Stop(ctx context.Context) error {
	v.moving = false
	return nil
}

// Open fully opens the valve.
func (v *Valve) Open(ctx context.Context) error {
	v.position = 1.0
	v.moving = false
	return nil
}

// Shut fully closes the valve.
func (v *Valve) Shut(ctx context.Context) error {
	v.position = 0.0
	v.moving = false
	return nil
}

// SetPosition sets the valve position (0.0-1.0).
func (v *Valve) SetPosition(ctx context.Context, position float64) error {
	if position < 0 || position > 1 {
		return fmt.Errorf("position must be 0.0-1.0, got %v", position)
	}

	if !v.isProportional {
		// Binary valve: snap to fully open or closed
		if position < 0.5 {
			v.position = 0.0
		} else {
			v.position = 1.0
		}
	} else {
		v.position = position
	}

	v.moving = false
	return nil
}

// GetPosition returns the current valve position.
func (v *Valve) GetPosition(ctx context.Context) (float64, error) {
	return v.position, nil
}

// IsOpen returns true if the valve is fully open.
func (v *Valve) IsOpen(ctx context.Context) (bool, error) {
	return v.position >= 0.99, nil
}

// IsClosed returns true if the valve is fully closed.
func (v *Valve) IsClosed(ctx context.Context) (bool, error) {
	return v.position <= 0.01, nil
}

// SetPositionDirectly sets position without validation (for testing).
func (v *Valve) SetPositionDirectly(position float64) {
	v.position = position
}

// SetMoving sets the moving state for testing.
func (v *Valve) SetMoving(moving bool) {
	v.moving = moving
}

// SetProperties sets the valve properties for testing.
func (v *Valve) SetProperties(isProportional, hasFeedback, normallyOpen bool, maxFlow float64, vType string) {
	v.isProportional = isProportional
	v.hasFeedback = hasFeedback
	v.normallyOpen = normallyOpen
	v.maxFlowRate = maxFlow
	v.valveType = vType
}

// Verify interface compliance.
var _ valve.Valve = (*Valve)(nil)
