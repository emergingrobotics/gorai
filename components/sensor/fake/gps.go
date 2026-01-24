package fake

import (
	"context"
	"fmt"

	"github.com/gorai/gorai/components/sensor"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("gps", "fake", NewGPS)
}

// GPS is a fake GPS sensor for testing.
type GPS struct {
	name resource.Name

	// Position
	Latitude, Longitude, Altitude float64

	// Velocity (ENU: East, North, Up)
	VelX, VelY, VelZ float64

	// Accuracy
	HorizontalAccuracy, VerticalAccuracy float64

	// Fix info
	FixType        sensor.FixType
	Heading        float64
	SatellitesUsed int
}

// NewGPS creates a new fake GPS.
func NewGPS(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "gps", nameStr)
	return &GPS{
		name:               name,
		FixType:            sensor.FixNone,
		HorizontalAccuracy: 10.0,
		VerticalAccuracy:   15.0,
	}, nil
}

// NewGPSWithName creates a fake GPS with a specific resource name.
func NewGPSWithName(name resource.Name) *GPS {
	return &GPS{
		name:               name,
		FixType:            sensor.FixNone,
		HorizontalAccuracy: 10.0,
		VerticalAccuracy:   15.0,
	}
}

// Name returns the resource name.
func (g *GPS) Name() resource.Name {
	return g.name
}

// Reconfigure updates the configuration.
func (g *GPS) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (g *GPS) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_state":
			return map[string]any{
				"position":   []float64{g.Latitude, g.Longitude, g.Altitude},
				"velocity":   []float64{g.VelX, g.VelY, g.VelZ},
				"fix":        g.FixType,
				"satellites": g.SatellitesUsed,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (g *GPS) Close(ctx context.Context) error {
	return nil
}

// Readings returns all sensor readings as a map.
func (g *GPS) Readings(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"latitude":            g.Latitude,
		"longitude":           g.Longitude,
		"altitude":            g.Altitude,
		"velocity":            map[string]float64{"x": g.VelX, "y": g.VelY, "z": g.VelZ},
		"horizontal_accuracy": g.HorizontalAccuracy,
		"vertical_accuracy":   g.VerticalAccuracy,
		"fix":                 int(g.FixType),
		"heading":             g.Heading,
		"satellites_used":     g.SatellitesUsed,
	}, nil
}

// Position returns latitude, longitude, and altitude.
func (g *GPS) Position(ctx context.Context) (lat, lng, alt float64, err error) {
	return g.Latitude, g.Longitude, g.Altitude, nil
}

// LinearVelocity returns velocity in m/s (ENU).
func (g *GPS) LinearVelocity(ctx context.Context) (x, y, z float64, err error) {
	return g.VelX, g.VelY, g.VelZ, nil
}

// Accuracy returns position accuracy in meters.
func (g *GPS) Accuracy(ctx context.Context) (horizontal, vertical float64, err error) {
	return g.HorizontalAccuracy, g.VerticalAccuracy, nil
}

// Fix returns the current fix type.
func (g *GPS) Fix(ctx context.Context) (sensor.FixType, error) {
	return g.FixType, nil
}

// GetHeading returns heading in degrees (0-360).
func (g *GPS) GetHeading(ctx context.Context) (float64, error) {
	return g.Heading, nil
}

// GetSatellitesUsed returns the number of satellites.
func (g *GPS) GetSatellitesUsed(ctx context.Context) (int, error) {
	return g.SatellitesUsed, nil
}

// SetPosition sets the GPS position for testing.
func (g *GPS) SetPosition(lat, lng, alt float64) {
	g.Latitude, g.Longitude, g.Altitude = lat, lng, alt
}

// SetVelocity sets the velocity for testing.
func (g *GPS) SetVelocity(x, y, z float64) {
	g.VelX, g.VelY, g.VelZ = x, y, z
}

// SetFix sets the fix type for testing.
func (g *GPS) SetFix(fix sensor.FixType, satellites int) {
	g.FixType = fix
	g.SatellitesUsed = satellites
}

// SetHeading sets the heading for testing.
func (g *GPS) SetHeading(heading float64) {
	g.Heading = heading
}

// Verify interface compliance.
var _ sensor.GPS = (*GPS)(nil)
