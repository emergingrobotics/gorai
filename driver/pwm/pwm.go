// Package pwm provides PWM (Pulse Width Modulation) driver interfaces.
package pwm

import (
	"context"

	"github.com/emergingrobotics/gorai/driver"
)

// Chip represents a PWM controller with one or more channels.
type Chip interface {
	driver.Driver

	// Channels returns the number of available PWM channels.
	Channels() int

	// Channel returns a PWM channel by index.
	Channel(n int) (Channel, error)
}

// Channel represents a single PWM output channel.
type Channel interface {
	// Enable starts PWM signal generation.
	Enable(ctx context.Context) error

	// Disable stops PWM signal generation.
	Disable(ctx context.Context) error

	// Enabled returns whether PWM output is currently enabled.
	Enabled() bool

	// SetPeriod sets the PWM period in nanoseconds.
	SetPeriod(ctx context.Context, ns uint64) error

	// Period returns the current period in nanoseconds.
	Period() uint64

	// SetDuty sets the duty cycle (high time) in nanoseconds.
	SetDuty(ctx context.Context, ns uint64) error

	// Duty returns the current duty in nanoseconds.
	Duty() uint64

	// SetDutyCycle sets the duty cycle as a fraction (0.0 to 1.0).
	SetDutyCycle(ctx context.Context, duty float64) error

	// DutyCycle returns the duty cycle as a fraction (0.0 to 1.0).
	DutyCycle() float64

	// SetFrequency sets the PWM frequency in Hz.
	SetFrequency(ctx context.Context, hz float64) error

	// Frequency returns the current frequency in Hz.
	Frequency() float64
}

// ServoChannel extends Channel with servo-specific methods.
type ServoChannel interface {
	Channel

	// SetPulse sets the pulse width in microseconds.
	// Standard servo range is 1000-2000µs with 1500µs center.
	SetPulse(ctx context.Context, us float64) error

	// Pulse returns the current pulse width in microseconds.
	Pulse() float64

	// SetNormalized sets position using -1.0 to 1.0 range.
	// -1.0 = min pulse, 0.0 = center, 1.0 = max pulse.
	SetNormalized(ctx context.Context, value float64) error

	// Normalized returns the current position as -1.0 to 1.0.
	Normalized() float64

	// SetAngle sets position in degrees (0-180 for standard servo).
	SetAngle(ctx context.Context, degrees float64) error

	// Angle returns the current position in degrees.
	Angle() float64
}

// Config holds PWM channel configuration.
type Config struct {
	// Frequency in Hz (default: 1000)
	Frequency float64

	// DutyCycle as fraction 0.0-1.0 (default: 0.0)
	DutyCycle float64

	// Inverted sets active-low output (default: false)
	Inverted bool
}

// ServoConfig holds servo-specific configuration.
type ServoConfig struct {
	Config

	// MinPulseUs is the minimum pulse width in microseconds (default: 1000)
	MinPulseUs float64

	// MaxPulseUs is the maximum pulse width in microseconds (default: 2000)
	MaxPulseUs float64

	// MinAngle is the minimum angle in degrees (default: 0)
	MinAngle float64

	// MaxAngle is the maximum angle in degrees (default: 180)
	MaxAngle float64
}

// DefaultConfig returns default PWM configuration.
func DefaultConfig() Config {
	return Config{
		Frequency: 1000,
		DutyCycle: 0.0,
		Inverted:  false,
	}
}

// DefaultServoConfig returns default servo configuration.
func DefaultServoConfig() ServoConfig {
	return ServoConfig{
		Config: Config{
			Frequency: 50,
			DutyCycle: 0.0,
			Inverted:  false,
		},
		MinPulseUs: 1000,
		MaxPulseUs: 2000,
		MinAngle:   0,
		MaxAngle:   180,
	}
}
