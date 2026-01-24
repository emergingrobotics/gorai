package gpiod

import (
	"fmt"

	"github.com/gorai/gorai/pkg/registry"
)

// Config holds configuration for a gpiod PWM component.
type Config struct {
	// Chip is the GPIO chip device path (e.g., "/dev/gpiochip4" on RPi5).
	Chip string `json:"chip"`

	// Pin is the GPIO pin number (BCM numbering).
	Pin int `json:"pin"`

	// FrequencyHz is the PWM frequency in Hz. Default: 50 (servo standard).
	FrequencyHz float64 `json:"frequency_hz"`

	// MinPulseUs is the minimum pulse width in microseconds. Default: 1000.
	MinPulseUs float64 `json:"min_pulse_us"`

	// MaxPulseUs is the maximum pulse width in microseconds. Default: 2000.
	MaxPulseUs float64 `json:"max_pulse_us"`

	// InitialPulseUs is the initial pulse width at startup. Default: 1500 (center).
	InitialPulseUs float64 `json:"initial_pulse_us"`

	// Invert inverts the signal (active-low). Default: false.
	Invert bool `json:"invert"`
}

// DefaultConfig returns a Config with default values for servo control.
func DefaultConfig() Config {
	return Config{
		Chip:           "/dev/gpiochip4",
		Pin:            18,
		FrequencyHz:    50.0,
		MinPulseUs:     1000.0,
		MaxPulseUs:     2000.0,
		InitialPulseUs: 1500.0,
		Invert:         false,
	}
}

// ParseConfig parses configuration from the registry config map.
func ParseConfig(conf registry.Config) (Config, error) {
	cfg := DefaultConfig()

	// Required fields
	if chip, ok := conf["chip"].(string); ok && chip != "" {
		cfg.Chip = chip
	} else {
		return cfg, fmt.Errorf("chip is required")
	}

	if pin, ok := conf["pin"].(float64); ok {
		cfg.Pin = int(pin)
	} else if pin, ok := conf["pin"].(int); ok {
		cfg.Pin = pin
	} else {
		return cfg, fmt.Errorf("pin is required")
	}

	// Optional fields
	if freq, ok := conf["frequency_hz"].(float64); ok {
		cfg.FrequencyHz = freq
	}

	if minPulse, ok := conf["min_pulse_us"].(float64); ok {
		cfg.MinPulseUs = minPulse
	}

	if maxPulse, ok := conf["max_pulse_us"].(float64); ok {
		cfg.MaxPulseUs = maxPulse
	}

	if initialPulse, ok := conf["initial_pulse_us"].(float64); ok {
		cfg.InitialPulseUs = initialPulse
	}

	if invert, ok := conf["invert"].(bool); ok {
		cfg.Invert = invert
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	if c.Chip == "" {
		return fmt.Errorf("chip path is required")
	}

	if c.Pin < 0 || c.Pin > 27 {
		return fmt.Errorf("pin must be between 0 and 27, got %d", c.Pin)
	}

	if c.FrequencyHz <= 0 {
		return fmt.Errorf("frequency_hz must be positive, got %f", c.FrequencyHz)
	}

	if c.FrequencyHz > 1000 {
		return fmt.Errorf("frequency_hz must be <= 1000 Hz for software PWM, got %f", c.FrequencyHz)
	}

	if c.MinPulseUs < 0 {
		return fmt.Errorf("min_pulse_us must be non-negative, got %f", c.MinPulseUs)
	}

	if c.MaxPulseUs <= c.MinPulseUs {
		return fmt.Errorf("max_pulse_us (%f) must be greater than min_pulse_us (%f)", c.MaxPulseUs, c.MinPulseUs)
	}

	if c.InitialPulseUs < c.MinPulseUs || c.InitialPulseUs > c.MaxPulseUs {
		return fmt.Errorf("initial_pulse_us (%f) must be within [%f, %f]", c.InitialPulseUs, c.MinPulseUs, c.MaxPulseUs)
	}

	// Check that pulse widths fit within a single period
	periodUs := 1_000_000.0 / c.FrequencyHz
	if c.MaxPulseUs > periodUs {
		return fmt.Errorf("max_pulse_us (%f) exceeds period (%f µs at %f Hz)", c.MaxPulseUs, periodUs, c.FrequencyHz)
	}

	return nil
}

// PeriodUs returns the PWM period in microseconds.
func (c Config) PeriodUs() float64 {
	return 1_000_000.0 / c.FrequencyHz
}

// PeriodNs returns the PWM period in nanoseconds.
func (c Config) PeriodNs() int64 {
	return int64(1_000_000_000.0 / c.FrequencyHz)
}

// CenterPulseUs returns the center pulse width in microseconds.
func (c Config) CenterPulseUs() float64 {
	return (c.MinPulseUs + c.MaxPulseUs) / 2.0
}

// PulseRangeUs returns half the pulse range (for normalized calculations).
func (c Config) PulseRangeUs() float64 {
	return (c.MaxPulseUs - c.MinPulseUs) / 2.0
}

