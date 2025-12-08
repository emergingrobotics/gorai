// Package service defines the base interfaces for Gorai services.
//
// Services represent software capabilities such as vision processing,
// ML inference, SLAM, and navigation. Unlike components which abstract
// hardware, services abstract algorithms and processing pipelines.
//
// All services implement the resource.Resource interface, which provides:
//   - Name() for unique resource identification
//   - Reconfigure() for runtime configuration updates
//   - DoCommand() for extensibility
//   - Close() for cleanup
package service

import (
	"github.com/gorai/gorai/pkg/resource"
)

// Service is the base interface for all services.
// All services must implement the resource.Resource interface.
type Service interface {
	resource.Resource
}

// Behavior is a service type that represents decision-making logic.
// Behaviors encapsulate autonomous actions and can expose derived sensors.
// See the service/behavior package for the full interface.
type Behavior interface {
	Service
}

// Coordinator is a service type that orchestrates multiple behaviors.
// Coordinators do not directly use components - they work through behaviors.
// See the service/coordinator package for the full interface.
type Coordinator interface {
	Service
}
