// Package thruster defines the underwater thruster component interface.
//
// Thrusters are specialized motors designed for underwater propulsion,
// typically using brushless DC motors with marine-grade ESCs.
//
// Examples:
//   - Blue Robotics T100, T200, M200
//   - Custom ROV/AUV propulsion systems
package thruster

import (
	"context"

	"github.com/gorai/gorai/components"
)

// Thruster represents an underwater propulsion thruster.
type Thruster interface {
	component.Actuator

	// SetThrust sets the thrust level from -1.0 (full reverse) to 1.0 (full forward).
	// Values within the deadband are treated as zero thrust.
	SetThrust(ctx context.Context, thrust float64) error

	// GetRPM returns the current motor RPM.
	// Returns ErrNotSupported if the ESC doesn't provide telemetry.
	GetRPM(ctx context.Context) (int, error)

	// GetTemperature returns the motor/ESC temperature in degrees Celsius.
	// Returns ErrNotSupported if temperature sensing is not available.
	GetTemperature(ctx context.Context) (float64, error)

	// GetCurrent returns the current draw in Amps.
	// Returns ErrNotSupported if current sensing is not available.
	GetCurrent(ctx context.Context) (float64, error)

	// GetProperties returns the thruster's properties.
	GetProperties(ctx context.Context) (Properties, error)
}

// Properties describes thruster capabilities and configuration.
type Properties struct {
	// MaxThrustForward is the maximum forward thrust in kgf.
	MaxThrustForward float64

	// MaxThrustReverse is the maximum reverse thrust in kgf.
	MaxThrustReverse float64

	// DeadbandWidth is the deadband width around zero (0.0-1.0 scale).
	DeadbandWidth float64

	// IsBidirectional indicates if the thruster supports reverse.
	IsBidirectional bool

	// HasTelemetry indicates if the ESC provides RPM/temp/current data.
	HasTelemetry bool

	// Protocol identifies the control protocol.
	// Values: "pwm", "i2c", "can", "dshot"
	Protocol string
}
