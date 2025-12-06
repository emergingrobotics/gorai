// Package component defines the base interfaces for Gorai components.
//
// Components represent hardware abstractions such as motors, cameras,
// and sensors. Each component type has a specific interface that
// implementations must satisfy.
package component

import "context"

// Component is the base interface for all hardware components.
type Component interface {
	// Name returns the component's unique name.
	Name() string

	// Reconfigure updates the component with new configuration.
	Reconfigure(ctx context.Context, conf map[string]any) error

	// Close releases all resources held by the component.
	Close(ctx context.Context) error
}

// Actuator is a component that can perform physical actions.
type Actuator interface {
	Component

	// IsMoving returns true if the actuator is currently in motion.
	IsMoving(ctx context.Context) (bool, error)

	// Stop immediately halts the actuator.
	Stop(ctx context.Context) error
}

// Sensor is a component that produces sensor readings.
type Sensor interface {
	Component

	// Readings returns the current sensor readings.
	Readings(ctx context.Context) (map[string]any, error)
}
