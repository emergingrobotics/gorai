// Package pwm defines the PWM component interface for servo and motor control.
//
// PWM components provide software-based Pulse Width Modulation output on GPIO
// pins. They support multiple input modes:
//   - Direct pulse width in microseconds
//   - Duty cycle as a fraction (0.0 to 1.0)
//   - Normalized value (-1.0 to 1.0) mapped to configured min/max pulse
//
// This package is designed for hobby servo control (50 Hz, 1000-2000µs pulse)
// but can be configured for other PWM applications.
package pwm

import (
	"context"

	component "github.com/gorai/gorai/components"
)

// PWM represents a PWM output component.
type PWM interface {
	component.Component

	// SetPulse sets the pulse width in microseconds.
	// The value is clamped to the configured min/max range.
	SetPulse(ctx context.Context, pulseUs float64) error

	// SetNormalized sets the output using a normalized value from -1.0 to 1.0.
	// -1.0 maps to min_pulse_us, 0.0 maps to center, 1.0 maps to max_pulse_us.
	SetNormalized(ctx context.Context, value float64) error

	// SetDuty sets the duty cycle as a fraction from 0.0 to 1.0.
	// Note: For servo control, use SetPulse or SetNormalized instead.
	SetDuty(ctx context.Context, duty float64) error

	// Enable starts PWM signal generation.
	Enable(ctx context.Context) error

	// Disable stops PWM signal generation and sets the pin LOW.
	Disable(ctx context.Context) error

	// IsEnabled returns true if PWM is currently generating a signal.
	IsEnabled(ctx context.Context) (bool, error)

	// GetPulse returns the current pulse width in microseconds.
	GetPulse(ctx context.Context) (float64, error)

	// GetNormalized returns the current value as normalized (-1.0 to 1.0).
	GetNormalized(ctx context.Context) (float64, error)

	// Properties returns the PWM configuration and capabilities.
	Properties(ctx context.Context) (Properties, error)
}

// Properties describes PWM configuration and capabilities.
type Properties struct {
	// FrequencyHz is the PWM frequency in Hz.
	FrequencyHz float64

	// MinPulseUs is the minimum pulse width in microseconds.
	MinPulseUs float64

	// MaxPulseUs is the maximum pulse width in microseconds.
	MaxPulseUs float64

	// Pin is the GPIO pin number (resolved by HAL).
	Pin int

	// Board is the board name from HAL (e.g., "raspberrypi5", "orangepi5b").
	Board string

	// Inverted indicates if the signal is active-low.
	Inverted bool

	// Mode indicates whether hardware or software PWM is being used.
	// Values: "hardware", "software"
	Mode string
}
