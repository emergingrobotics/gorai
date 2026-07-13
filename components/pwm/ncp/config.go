// Package ncp provides a generic NCP remote client for PWM capabilities.
//
// It implements the pwm.PWM interface by issuing NCP requests over NATS to a
// PWM component exposed on another robot at
// gorai.<robot>.<component>.command. This is how a ground station drives a
// thruster physically attached to a different robot.
package ncp

import (
	"fmt"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds configuration for the NCP remote PWM client.
type Config struct {
	// Robot is the target robot id (effective namespace) hosting the component.
	Robot string `json:"robot"`

	// Component is the target component name on the remote robot.
	Component string `json:"component"`

	// MinPulseUs is the minimum pulse width in microseconds. Default: 1000.
	MinPulseUs float64 `json:"min_pulse_us"`

	// MaxPulseUs is the maximum pulse width in microseconds. Default: 2000.
	MaxPulseUs float64 `json:"max_pulse_us"`

	// InitialPulseUs is the assumed pulse width before the first state update.
	InitialPulseUs float64 `json:"initial_pulse_us"`

	// TimeoutMs bounds each remote request. Default: 2000.
	TimeoutMs int `json:"timeout_ms"`

	// ArmTimeoutMs bounds an arm request, which blocks while the remote runs
	// its (multi-second) arming sequence. Default: 10000.
	ArmTimeoutMs int `json:"arm_timeout_ms"`
}

// DefaultConfig returns configuration defaults for a standard 50 Hz ESC/servo.
func DefaultConfig() Config {
	return Config{
		MinPulseUs:     1000.0,
		MaxPulseUs:     2000.0,
		InitialPulseUs: 1500.0,
		TimeoutMs:      2000,
		ArmTimeoutMs:   10000,
	}
}

// ParseConfig builds a Config from a registry config map, applying defaults.
func ParseConfig(conf registry.Config) (Config, error) {
	c := resource.NewConfig(conf)
	cfg := DefaultConfig()

	if v, ok := c.GetString("robot"); ok {
		cfg.Robot = v
	}
	if v, ok := c.GetString("component"); ok {
		cfg.Component = v
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
	if v, ok := c.GetInt("timeout_ms"); ok {
		cfg.TimeoutMs = v
	}
	if v, ok := c.GetInt("arm_timeout_ms"); ok {
		cfg.ArmTimeoutMs = v
	}

	return cfg, nil
}

// Validate checks required fields and consistency.
func (c Config) Validate() error {
	if c.Robot == "" {
		return fmt.Errorf("robot is required (target robot id)")
	}
	if c.Component == "" {
		return fmt.Errorf("component is required (target component name)")
	}
	if c.MinPulseUs >= c.MaxPulseUs {
		return fmt.Errorf("min_pulse_us (%f) must be less than max_pulse_us (%f)",
			c.MinPulseUs, c.MaxPulseUs)
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
