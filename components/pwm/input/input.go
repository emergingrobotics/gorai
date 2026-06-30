// Package input defines the PWM input component interface for reading external
// PWM signals such as RC receivers, motor encoders, or other pulse-width
// modulated sources.
//
// PWM input components measure the pulse width and period of incoming signals,
// allowing the host to derive frequency and duty cycle.
package input

import (
	"context"

	component "github.com/emergingrobotics/gorai/components"
)

// PWMInput represents a PWM input measurement component.
type PWMInput interface {
	component.Component

	// GetPulse returns the most recently measured pulse width in microseconds.
	GetPulse(ctx context.Context) (float64, error)

	// GetPeriod returns the most recently measured period in microseconds.
	GetPeriod(ctx context.Context) (float64, error)

	// GetFrequency returns the derived frequency in Hz (1e6 / period_us).
	GetFrequency(ctx context.Context) (float64, error)

	// IsActive returns true if a valid measurement has been received recently.
	IsActive(ctx context.Context) (bool, error)

	// Properties returns the PWM input configuration.
	Properties(ctx context.Context) (Properties, error)
}

// Properties describes PWM input configuration.
type Properties struct {
	Pin      int    `json:"pin"`
	DeviceID string `json:"device_id"`
	Mode     string `json:"mode"`
}
