// Package navigation defines the navigation service interface.
package navigation

import (
	"context"

	"github.com/gorai/gorai/services"
)

// Service provides autonomous navigation capabilities.
type Service interface {
	service.Service

	// Mode returns the current navigation mode.
	Mode(ctx context.Context) (Mode, error)

	// SetMode sets the navigation mode.
	SetMode(ctx context.Context, mode Mode) error

	// Location returns the current location.
	Location(ctx context.Context) (GeoPoint, error)

	// Waypoints returns the list of waypoints.
	Waypoints(ctx context.Context) ([]Waypoint, error)

	// AddWaypoint adds a waypoint to the list.
	AddWaypoint(ctx context.Context, point GeoPoint, name string) error

	// RemoveWaypoint removes a waypoint.
	RemoveWaypoint(ctx context.Context, id string) error

	// Paths returns planned paths.
	Paths(ctx context.Context) ([]Path, error)

	// Properties returns navigation service properties.
	Properties(ctx context.Context) (Properties, error)
}

// Mode represents the navigation mode.
type Mode int

const (
	ModeUnspecified Mode = iota
	ModeManual
	ModeWaypoint
	ModeExplore
)

// GeoPoint represents a geographic location.
type GeoPoint struct {
	Latitude  float64
	Longitude float64
	Altitude  float64
}

// Waypoint represents a navigation waypoint.
type Waypoint struct {
	ID       string
	Location GeoPoint
	Name     string
}

// Path represents a planned path.
type Path struct {
	DestinationWaypointID string
	Geopoints             []GeoPoint
}

// Properties describes navigation service capabilities.
type Properties struct {
	// MapType describes the map representation used.
	MapType string
}
