// Package linux provides a native PWM component backed by the Linux sysfs
// PWM interface (/sys/class/pwm). It drives hardware PWM channels directly
// from the host SBC (e.g. Raspberry Pi 5 GPIO13 = pwmchip2 channel 1),
// making it suitable for servo/ESC control without a co-processor.
package linux

import (
	"fmt"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds configuration for the native sysfs PWM component.
type Config struct {
	// Chip is the sysfs PWM chip number (e.g. 2 for /sys/class/pwm/pwmchip2).
	Chip int `json:"chip"`

	// Channel is the channel index within the chip (e.g. 1 for pwm1).
	Channel int `json:"channel"`

	// FrequencyHz is the PWM frequency in Hz. Default: 50 (servo/ESC).
	FrequencyHz float64 `json:"frequency_hz"`

	// MinPulseUs is the minimum pulse width in microseconds. Default: 1000.
	MinPulseUs float64 `json:"min_pulse_us"`

	// MaxPulseUs is the maximum pulse width in microseconds. Default: 2000.
	MaxPulseUs float64 `json:"max_pulse_us"`

	// InitialPulseUs is the pulse width set on startup. Default: center of range.
	InitialPulseUs float64 `json:"initial_pulse_us"`

	// Invert enables active-low output when the controller supports polarity.
	Invert bool `json:"invert"`

	// EnableOnStart starts signal generation during construction. Default: true.
	EnableOnStart bool `json:"enable_on_start"`

	// ArmSequence is the ordered list of pulse/hold steps used to arm the ESC.
	// When unset it defaults to a max-pulse hold followed by a neutral hold.
	ArmSequence []ArmStep `json:"arm_sequence"`
}

// ArmStep is a single phase of an ESC arming sequence: hold a pulse width for
// a fixed duration before advancing to the next step.
type ArmStep struct {
	// PulseUs is the pulse width to hold in microseconds.
	PulseUs float64 `json:"pulse_us"`

	// HoldMs is how long to hold the pulse in milliseconds.
	HoldMs int `json:"hold_ms"`
}

// DefaultConfig returns configuration defaults for a standard 50 Hz ESC/servo.
func DefaultConfig() Config {
	return Config{
		FrequencyHz:    50.0,
		MinPulseUs:     1000.0,
		MaxPulseUs:     2000.0,
		InitialPulseUs: 1500.0,
		EnableOnStart:  true,
	}
}

// ParseConfig builds a Config from a registry config map, applying defaults
// for any unset fields.
func ParseConfig(conf registry.Config) (Config, error) {
	c := resource.NewConfig(conf)
	cfg := DefaultConfig()

	if v, ok := c.GetInt("chip"); ok {
		cfg.Chip = v
	}
	if v, ok := c.GetInt("channel"); ok {
		cfg.Channel = v
	}
	if v, ok := c.GetFloat("frequency_hz"); ok {
		cfg.FrequencyHz = v
	}
	if v, ok := c.GetFloat("min_pulse_us"); ok {
		cfg.MinPulseUs = v
	}
	if v, ok := c.GetFloat("max_pulse_us"); ok {
		cfg.MaxPulseUs = v
	}
	if v, ok := c.GetFloat("initial_pulse_us"); ok {
		cfg.InitialPulseUs = v
	} else {
		cfg.InitialPulseUs = (cfg.MinPulseUs + cfg.MaxPulseUs) / 2
	}
	if v, ok := c.GetBool("invert"); ok {
		cfg.Invert = v
	}
	if v, ok := c.GetBool("enable_on_start"); ok {
		cfg.EnableOnStart = v
	}

	if steps, ok := parseArmSequence(conf["arm_sequence"]); ok {
		cfg.ArmSequence = steps
	} else {
		cfg.ArmSequence = defaultArmSequence(cfg)
	}

	return cfg, nil
}

// defaultArmSequence returns the standard ESC arming sequence: hold the maximum
// pulse for one second, then neutral (initial) pulse for one second.
func defaultArmSequence(cfg Config) []ArmStep {
	return []ArmStep{
		{PulseUs: cfg.MaxPulseUs, HoldMs: 1000},
		{PulseUs: cfg.InitialPulseUs, HoldMs: 1000},
	}
}

// parseArmSequence converts a raw config value (a JSON array of step objects)
// into a slice of ArmStep. It returns ok=false when the value is absent or empty
// so the caller can fall back to the default sequence.
func parseArmSequence(raw any) ([]ArmStep, bool) {
	list, ok := raw.([]any)
	if !ok || len(list) == 0 {
		return nil, false
	}
	steps := make([]ArmStep, 0, len(list))
	for _, item := range list {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		var step ArmStep
		if v, ok := toFloat(m["pulse_us"]); ok {
			step.PulseUs = v
		}
		if v, ok := toFloat(m["hold_ms"]); ok {
			step.HoldMs = int(v)
		}
		steps = append(steps, step)
	}
	if len(steps) == 0 {
		return nil, false
	}
	return steps, true
}

// toFloat coerces a JSON-decoded numeric value to float64.
func toFloat(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// Validate checks that the configuration is internally consistent.
func (c Config) Validate() error {
	if c.Chip < 0 {
		return fmt.Errorf("chip must be non-negative, got %d", c.Chip)
	}
	if c.Channel < 0 {
		return fmt.Errorf("channel must be non-negative, got %d", c.Channel)
	}
	if c.FrequencyHz <= 0 {
		return fmt.Errorf("frequency_hz must be positive, got %f", c.FrequencyHz)
	}
	if c.MinPulseUs <= 0 || c.MaxPulseUs <= 0 {
		return fmt.Errorf("pulse widths must be positive")
	}
	if c.MinPulseUs >= c.MaxPulseUs {
		return fmt.Errorf("min_pulse_us (%f) must be less than max_pulse_us (%f)",
			c.MinPulseUs, c.MaxPulseUs)
	}
	// A pulse must fit within the period, or the kernel rejects the duty write.
	periodUs := 1_000_000.0 / c.FrequencyHz
	if c.MaxPulseUs > periodUs {
		return fmt.Errorf("max_pulse_us (%f) exceeds period (%f us) at %f Hz",
			c.MaxPulseUs, periodUs, c.FrequencyHz)
	}
	for i, step := range c.ArmSequence {
		if step.HoldMs <= 0 {
			return fmt.Errorf("arm_sequence[%d] hold_ms must be positive, got %d", i, step.HoldMs)
		}
		if step.PulseUs < c.MinPulseUs || step.PulseUs > c.MaxPulseUs {
			return fmt.Errorf("arm_sequence[%d] pulse_us (%f) must be within [%f, %f]",
				i, step.PulseUs, c.MinPulseUs, c.MaxPulseUs)
		}
	}
	return nil
}

// CenterPulseUs returns the midpoint of the pulse range.
func (c Config) CenterPulseUs() float64 {
	return (c.MinPulseUs + c.MaxPulseUs) / 2
}

// PulseRangeUs returns the half-range used for normalized mapping.
func (c Config) PulseRangeUs() float64 {
	return (c.MaxPulseUs - c.MinPulseUs) / 2
}

// PeriodUs returns the PWM period in microseconds.
func (c Config) PeriodUs() float64 {
	return 1_000_000.0 / c.FrequencyHz
}
