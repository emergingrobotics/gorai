// Package valve defines the valve/fluid control component interface.
//
// Valves control fluid flow in pneumatic, hydraulic, and liquid systems.
// This interface supports both simple on/off solenoid valves and
// proportional control valves.
//
// Examples:
//   - Solenoid valves (on/off)
//   - Servo/ball valves (proportional)
//   - Pneumatic cylinders
package valve

import (
	"context"

	"github.com/gorai/gorai/components"
)

// Valve represents a fluid control valve.
type Valve interface {
	component.Actuator

	// Open fully opens the valve.
	Open(ctx context.Context) error

	// Shut fully closes the valve.
	// Named "Shut" to avoid conflict with resource.Resource.Close().
	Shut(ctx context.Context) error

	// SetPosition sets the valve position from 0.0 (closed) to 1.0 (open).
	// For on/off valves, values < 0.5 close and >= 0.5 open.
	SetPosition(ctx context.Context, position float64) error

	// GetPosition returns the current valve position (0.0-1.0).
	// For on/off valves without feedback, returns 0.0 or 1.0.
	GetPosition(ctx context.Context) (float64, error)

	// IsOpen returns true if the valve is fully open.
	IsOpen(ctx context.Context) (bool, error)

	// IsClosed returns true if the valve is fully closed.
	IsClosed(ctx context.Context) (bool, error)
}

// Properties describes valve capabilities and configuration.
type Properties struct {
	// IsProportional indicates if the valve supports proportional control.
	// If false, only fully open/closed states are available.
	IsProportional bool

	// HasFeedback indicates if the valve can report its actual position.
	HasFeedback bool

	// NormallyOpen indicates if the valve is open when unpowered.
	NormallyOpen bool

	// MaxFlowRate is the maximum flow rate in appropriate units (L/min, etc.).
	MaxFlowRate float64

	// ValveType identifies the valve mechanism.
	// Values: "solenoid", "ball", "butterfly", "gate", "check"
	ValveType string
}
