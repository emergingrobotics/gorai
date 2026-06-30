// Package motor defines the motor component interface.
package motor

import (
	"context"

	"github.com/emergingrobotics/gorai/components"
)

// Motor represents a controllable motor.
type Motor interface {
	component.Actuator

	// SetPower sets the motor power from -1.0 (full reverse) to 1.0 (full forward).
	SetPower(ctx context.Context, power float64) error

	// SetVelocity sets target velocity (rad/s or m/s depending on motor type).
	SetVelocity(ctx context.Context, velocity float64) error

	// GoTo moves the motor to the specified absolute position at the given velocity.
	GoTo(ctx context.Context, position, velocity float64) error

	// GoFor moves the motor for the specified number of revolutions at the given RPM.
	// Positive RPM moves forward, negative moves backward.
	GoFor(ctx context.Context, rpm, revolutions float64) error

	// GetPosition returns the current position (in revolutions from zero).
	GetPosition(ctx context.Context) (float64, error)

	// GetVelocity returns the current velocity.
	GetVelocity(ctx context.Context) (float64, error)

	// ResetZeroPosition sets the current position as the zero position.
	// The offset parameter allows setting a specific value as the new position.
	ResetZeroPosition(ctx context.Context, offset float64) error

	// IsPowered returns whether the motor is currently receiving power
	// and the current power level.
	IsPowered(ctx context.Context) (bool, float64, error)

	// Properties returns the motor's properties.
	Properties(ctx context.Context) (Properties, error)
}

// Properties describes motor capabilities.
type Properties struct {
	// PositionReporting indicates whether the motor can report its position.
	PositionReporting bool

	// VelocityReporting indicates whether the motor can report its velocity.
	VelocityReporting bool

	// SupportsGoTo indicates whether the motor supports absolute positioning.
	SupportsGoTo bool
}
