// Package servo defines the servo motor component interface.
//
// Servos are position-controlled motors commonly used in robotics for precise
// angular positioning. This package supports both traditional RC/PWM servos
// and smart serial bus servos.
//
// Examples:
//   - RC servos: Standard PWM-controlled hobby servos
//   - Smart servos: Dynamixel, LX-16A, Feetech SCS/STS series
package servo

import (
	"context"

	"github.com/emergingrobotics/gorai/components"
)

// Servo represents a position-controlled servo motor.
type Servo interface {
	component.Actuator

	// SetAngle moves the servo to the specified angle in degrees.
	// The valid range depends on the servo (typically -90 to +90 or 0 to 180).
	SetAngle(ctx context.Context, degrees float64) error

	// GetAngle returns the current angle in degrees.
	// For servos without feedback, this returns the last commanded angle.
	GetAngle(ctx context.Context) (float64, error)

	// SetSpeed sets the movement speed.
	// Units depend on the servo type (degrees/sec or 0-1 normalized).
	SetSpeed(ctx context.Context, speed float64) error

	// SetTorqueLimit sets the torque limit as a fraction (0.0-1.0).
	// Returns ErrNotSupported for servos without torque control.
	SetTorqueLimit(ctx context.Context, limit float64) error

	// GetProperties returns the servo's properties.
	GetProperties(ctx context.Context) (Properties, error)
}

// Properties describes servo capabilities and configuration.
type Properties struct {
	// MinAngle is the minimum angle in degrees.
	MinAngle float64

	// MaxAngle is the maximum angle in degrees.
	MaxAngle float64

	// IsContinuous indicates if this is a continuous rotation servo.
	IsContinuous bool

	// HasFeedback indicates if the servo can report its actual position.
	HasFeedback bool

	// Protocol identifies the control protocol.
	// Values: "pwm", "dynamixel", "lx16a", "feetech"
	Protocol string
}
