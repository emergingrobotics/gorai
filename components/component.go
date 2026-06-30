// Package component defines the base interfaces for Gorai components.
//
// Components represent hardware abstractions such as motors, cameras,
// and sensors. Each component type has a specific interface that
// implementations must satisfy.
//
// All components implement the resource.Resource interface, which provides:
//   - Name() for unique resource identification
//   - Reconfigure() for runtime configuration updates
//   - DoCommand() for extensibility
//   - Close() for cleanup
package component

import (
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Component is the base interface for all hardware components.
// All components must implement the resource.Resource interface.
type Component interface {
	resource.Resource
}

// Actuator is a component that can perform physical actions.
type Actuator interface {
	Component
	resource.Actuator
}

// Sensor is a component that produces sensor readings.
type Sensor interface {
	Component
	resource.Sensor
}

// Power is a component that provides power information.
// Examples: batteries, power supplies, solar panels.
type Power interface {
	Component
	resource.Power
}

// Space is a component that represents a spatial region.
// Examples: containers, zones, workspaces.
type Space interface {
	Component
	resource.Space
}

// Link is a component that provides communication capabilities.
// Examples: serial ports, IP connections, NATS channels.
type Link interface {
	Component
	resource.Link
}
