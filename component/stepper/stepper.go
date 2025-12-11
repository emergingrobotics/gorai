// Package stepper defines the stepper motor component interface.
//
// Stepper motors provide precise discrete-step positioning and are commonly
// used in CNC machines, 3D printers, and precision positioning systems.
//
// Examples:
//   - NEMA 17/23/34 motors with A4988, DRV8825, or TMC drivers
//   - Integrated closed-loop steppers
package stepper

import (
	"context"

	"github.com/gorai/gorai/component"
)

// Stepper represents a stepper motor with step/direction control.
type Stepper interface {
	component.Actuator

	// Step moves the motor by the specified number of steps.
	// Positive steps move in one direction, negative in the other.
	Step(ctx context.Context, steps int64) error

	// SetMicrostepping sets the microstepping divisor (1, 2, 4, 8, 16, 32, 256).
	// Returns ErrNotSupported if the driver doesn't support runtime configuration.
	SetMicrostepping(ctx context.Context, divisor int) error

	// SetCurrent sets the run and hold current in milliamps.
	// Returns ErrNotSupported if the driver doesn't support current control.
	SetCurrent(ctx context.Context, runMA, holdMA int) error

	// GetPosition returns the current position in steps from the zero position.
	GetPosition(ctx context.Context) (int64, error)

	// ResetPosition sets the current position as zero.
	ResetPosition(ctx context.Context) error

	// Home performs a homing operation using endstop or stall detection.
	// direction: true = positive direction, false = negative direction.
	Home(ctx context.Context, direction bool) error

	// GetProperties returns the stepper's properties.
	GetProperties(ctx context.Context) (Properties, error)
}

// Properties describes stepper motor capabilities and configuration.
type Properties struct {
	// StepsPerRevolution is the native steps per revolution (typically 200).
	StepsPerRevolution int

	// MaxMicrostepping is the maximum microstepping divisor supported.
	MaxMicrostepping int

	// MaxCurrent is the maximum current in milliamps.
	MaxCurrent int

	// HasStallDetection indicates if the driver supports stall detection.
	HasStallDetection bool

	// Driver identifies the driver type.
	// Values: "a4988", "drv8825", "tmc2209", "tmc5160"
	Driver string
}
