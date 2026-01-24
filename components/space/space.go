// Package space defines the space component interface.
//
// Space components represent virtual abstractions over physical volumes on the robot.
// A Space doesn't directly interface with hardware—it aggregates and coordinates
// other components (actuators, sensors) that control or monitor a physical area.
//
// Use Spaces for:
//   - Storage areas with doors or hatches (cargo bay, sample drawer)
//   - Tanks with valves and level sensors (ballast tank, fuel tank)
//   - Compartments with environmental controls (battery bay, equipment compartment)
//
// Example: A ballast tank Space would reference:
//   - A fill valve (actuator)
//   - A drain valve (actuator)
//   - A level sensor (sensor)
package space

import (
	"context"

	"github.com/gorai/gorai/components"
	"github.com/gorai/gorai/pkg/resource"
)

// Space represents a virtual container on the robot that aggregates other components.
type Space interface {
	component.Component
	resource.Space
}

// SpaceType identifies the type of space.
type SpaceType int

const (
	// SpaceTypeContainer is a storage volume (cargo bay, hopper, sample drawer).
	SpaceTypeContainer SpaceType = iota
	// SpaceTypeTank is a fluid storage space (ballast tank, fuel tank, coolant reservoir).
	SpaceTypeTank
	// SpaceTypeCompartment is an enclosed area (equipment bay, battery compartment).
	SpaceTypeCompartment
)

// String returns the string representation of a SpaceType.
func (st SpaceType) String() string {
	switch st {
	case SpaceTypeContainer:
		return "container"
	case SpaceTypeTank:
		return "tank"
	case SpaceTypeCompartment:
		return "compartment"
	default:
		return "unknown"
	}
}

// Properties describes space component capabilities.
type Properties struct {
	// Type indicates the space type.
	Type SpaceType

	// Name is a human-readable name for the space.
	Name string

	// MaxVolume is the maximum volume in cubic meters.
	MaxVolume float64

	// MaxWeight is the maximum weight capacity in kg (0 = unlimited).
	MaxWeight float64

	// CanTrackContents indicates whether contents can be tracked.
	CanTrackContents bool

	// CanMeasureVolume indicates whether actual volume can be measured.
	CanMeasureVolume bool

	// ComponentNames lists the names of associated components (valves, doors, sensors).
	ComponentNames []resource.Name
}

// Extended is an optional extended interface for space components with
// additional capabilities beyond the base Space interface.
type Extended interface {
	Space

	// GetProperties returns the space properties.
	GetProperties(ctx context.Context) (Properties, error)

	// GetUsedVolume returns the volume currently occupied in cubic meters.
	GetUsedVolume(ctx context.Context) (float64, error)

	// GetWeight returns the current weight of contents in kg.
	GetWeight(ctx context.Context) (float64, error)

	// AddContent adds an item to the space contents.
	AddContent(ctx context.Context, id string) error

	// RemoveContent removes an item from the space contents.
	RemoveContent(ctx context.Context, id string) error

	// ClearContents removes all tracked contents.
	ClearContents(ctx context.Context) error

	// IsFull returns true if the space is at capacity.
	IsFull(ctx context.Context) (bool, error)
}
