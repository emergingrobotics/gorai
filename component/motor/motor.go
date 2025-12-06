// Package motor defines the motor component interface.
package motor

import (
	"context"

	"github.com/gorai/gorai/component"
)

// Motor represents a controllable motor.
type Motor interface {
	component.Actuator

	// SetPower sets the motor power from -1.0 (full reverse) to 1.0 (full forward).
	SetPower(ctx context.Context, power float64) error

	// GoFor moves the motor for the specified number of revolutions at the given RPM.
	// Positive RPM moves forward, negative moves backward.
	GoFor(ctx context.Context, rpm, revolutions float64) error

	// GoTo moves the motor to the specified position (in revolutions from home).
	GoTo(ctx context.Context, rpm, position float64) error

	// ResetZeroPosition sets the current position as the zero position.
	ResetZeroPosition(ctx context.Context, offset float64) error

	// Position returns the current position in revolutions.
	Position(ctx context.Context) (float64, error)

	// Properties returns the motor's properties.
	Properties(ctx context.Context) (Properties, error)

	// IsPowered returns whether the motor is currently receiving power
	// and the current power level.
	IsPowered(ctx context.Context) (bool, float64, error)
}

// Properties describes motor capabilities.
type Properties struct {
	// PositionReporting indicates whether the motor can report its position.
	PositionReporting bool
}
