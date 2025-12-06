// Package sensor defines generic sensor component interfaces.
package sensor

import (
	"context"

	"github.com/gorai/gorai/component"
)

// Sensor is a generic sensor that returns key-value readings.
type Sensor interface {
	component.Sensor
}

// IMU represents an inertial measurement unit.
type IMU interface {
	component.Component

	// LinearAcceleration returns acceleration in m/s².
	LinearAcceleration(ctx context.Context) (x, y, z float64, err error)

	// AngularVelocity returns rotation rate in rad/s.
	AngularVelocity(ctx context.Context) (x, y, z float64, err error)

	// Orientation returns the current orientation as a quaternion.
	Orientation(ctx context.Context) (x, y, z, w float64, err error)
}

// GPS represents a GPS/GNSS receiver.
type GPS interface {
	component.Component

	// Position returns latitude, longitude in degrees and altitude in meters.
	Position(ctx context.Context) (lat, lng, alt float64, err error)

	// LinearVelocity returns velocity in m/s.
	LinearVelocity(ctx context.Context) (x, y, z float64, err error)

	// Accuracy returns position accuracy in meters.
	Accuracy(ctx context.Context) (horizontal, vertical float64, err error)

	// Fix returns the current fix type.
	Fix(ctx context.Context) (FixType, error)
}

// FixType represents GPS fix quality.
type FixType int

const (
	FixNone FixType = iota
	FixGPS
	FixDGPS
	FixRTK
)

// Encoder represents a rotary or linear encoder.
type Encoder interface {
	component.Component

	// Position returns the current position.
	Position(ctx context.Context) (float64, error)

	// ResetPosition sets the current position as zero.
	ResetPosition(ctx context.Context) error

	// Properties returns encoder properties.
	Properties(ctx context.Context) (EncoderProperties, error)
}

// EncoderProperties describes encoder capabilities.
type EncoderProperties struct {
	TicksPerRevolution int
	AngleDegreesSupported bool
}

// RangeFinder represents a distance sensor.
type RangeFinder interface {
	component.Component

	// Range returns the measured distance in meters.
	Range(ctx context.Context) (float64, error)

	// Properties returns range finder properties.
	Properties(ctx context.Context) (RangeFinderProperties, error)
}

// RangeFinderProperties describes range finder capabilities.
type RangeFinderProperties struct {
	MinRange float64
	MaxRange float64
	FieldOfView float64 // radians
}
