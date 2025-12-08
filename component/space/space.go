// Package space defines the space component interface.
//
// Space components represent physical volumes that can contain things, such as
// containers, cargo bays, workspaces, and defined zones.
package space

import (
	"context"

	"github.com/gorai/gorai/component"
	"github.com/gorai/gorai/pkg/resource"
)

// Space represents a physical volume that can contain things.
type Space interface {
	component.Component
	resource.Space
}

// SpaceType identifies the type of space.
type SpaceType int

const (
	// SpaceTypeContainer is a generic container (cargo bay, hopper, tank).
	SpaceTypeContainer SpaceType = iota
	// SpaceTypeWorkspace is a robot's operating area.
	SpaceTypeWorkspace
	// SpaceTypeZone is a defined region (safety zone, charging zone).
	SpaceTypeZone
)

// String returns the string representation of a SpaceType.
func (st SpaceType) String() string {
	switch st {
	case SpaceTypeContainer:
		return "container"
	case SpaceTypeWorkspace:
		return "workspace"
	case SpaceTypeZone:
		return "zone"
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
