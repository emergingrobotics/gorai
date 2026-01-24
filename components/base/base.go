// Package base defines the mobile base component interface.
package base

import (
	"context"

	"github.com/gorai/gorai/components"
)

// Base represents a mobile robot base.
type Base interface {
	component.Actuator

	// SetVelocity sets the linear and angular velocity.
	// Linear velocity is in m/s, angular velocity is in rad/s.
	SetVelocity(ctx context.Context, linear, angular float64) error

	// MoveStraight moves the base straight for the given distance at the given speed.
	// Distance is in meters, speed is in m/s.
	MoveStraight(ctx context.Context, distance, speed float64) error

	// Spin rotates the base by the given angle at the given speed.
	// Angle is in radians, speed is in rad/s.
	Spin(ctx context.Context, angle, speed float64) error

	// SetPower sets raw power to the base motors.
	// Linear and angular are -1.0 to 1.0.
	SetPower(ctx context.Context, linear, angular float64) error

	// Properties returns the base's properties.
	Properties(ctx context.Context) (Properties, error)
}

// Properties describes mobile base capabilities.
type Properties struct {
	// WheelCircumference is the wheel circumference in meters.
	WheelCircumference float64
	// WheelBase is the distance between wheels in meters.
	WheelBase float64
	// MaxLinearVelocity is the maximum linear velocity in m/s.
	MaxLinearVelocity float64
	// MaxAngularVelocity is the maximum angular velocity in rad/s.
	MaxAngularVelocity float64
}
