package gpiod

import (
	"fmt"
	"strings"

	"github.com/emergingrobotics/gorai/pkg/registry"
)

// HWMode specifies the PWM hardware mode selection.
type HWMode string

const (
	// HWModeAuto uses hardware PWM if the pin supports it, otherwise falls back
	// to software PWM with a warning. This is the default.
	HWModeAuto HWMode = "auto"

	// HWModeHardware requires hardware PWM. Fails if the pin doesn't support it.
	HWModeHardware HWMode = "hardware"

	// HWModeSoftware always uses software PWM (GPIO bit-banging).
	// Timing may be inaccurate due to OS scheduling.
	HWModeSoftware HWMode = "software"
)

// Config holds configuration for a gpiod PWM component.
type Config struct {
	// Pin accepts multiple formats:
	// - Integer: 17 (GPIO number)
	// - String: "GPIO17", "PIN12", "PWM0", "18"
	// The HAL will resolve this to the correct GPIO number for the board.
	Pin any `json:"pin"`

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

	// HWMode controls hardware vs software PWM selection:
	// - "auto" (default): Use hardware PWM if available, software otherwise (with warning)
	// - "hardware": Require hardware PWM, fail if not available
	// - "software": Always use software PWM
	HWMode HWMode `json:"hw_mode"`
}

// DefaultConfig returns a Config with default values for servo control.
func DefaultConfig() Config {
	return Config{
		Pin:            18,
		FrequencyHz:    50.0,
		MinPulseUs:     1000.0,
		MaxPulseUs:     2000.0,
		InitialPulseUs: 1500.0,
		Invert:         false,
		HWMode:         HWModeAuto,
	}
}

// ParseConfig parses configuration from the registry config map.
func ParseConfig(conf registry.Config) (Config, error) {
	cfg := DefaultConfig()

	// Pin is required - can be int, float64, or string
	if pin, ok := conf["pin"]; ok {
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

	if hwMode, ok := conf["hw_mode"].(string); ok {
		switch strings.ToLower(hwMode) {
		case "auto", "":
			cfg.HWMode = HWModeAuto
		case "hardware", "hw":
			cfg.HWMode = HWModeHardware
		case "software", "sw":
			cfg.HWMode = HWModeSoftware
		default:
			return cfg, fmt.Errorf("invalid hw_mode %q, must be 'auto', 'hardware', or 'software'", hwMode)
		}
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c Config) Validate() error {
	// Pin validation is deferred to HAL resolution
	if c.Pin == nil {
		return fmt.Errorf("pin is required")
	}

	if c.FrequencyHz <= 0 {
		return fmt.Errorf("frequency_hz must be positive, got %f", c.FrequencyHz)
	}

	// Software PWM has lower frequency limits due to OS scheduling jitter.
	// Hardware PWM can handle much higher frequencies (up to MHz range).
	// We only validate software limits here; hardware limits are validated at runtime.
	if c.HWMode == HWModeSoftware && c.FrequencyHz > 1000 {
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

	// Validate hw_mode
	switch c.HWMode {
	case HWModeAuto, HWModeHardware, HWModeSoftware, "":
		// Valid
	default:
		return fmt.Errorf("invalid hw_mode %q", c.HWMode)
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
