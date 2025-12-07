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
	"github.com/gorai/gorai/pkg/resource"
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
