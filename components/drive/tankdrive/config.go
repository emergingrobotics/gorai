// Package tankdrive provides a tank-drive (differential thrust) controller
// component. It mixes a normalized surge/yaw intent into two PWM actuators and
// applies them on a fixed-rate control loop with slew limiting and a
// command-timeout failsafe. The loop cadence regulates when setpoints are
// handed to the PWM drivers; it does not generate the PWM carrier itself.
package tankdrive

import (
	"fmt"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds configuration for the tank-drive controller.
type Config struct {
	// Left and Right are the names of the PWM components to drive.
	Left  string `json:"left"`
	Right string `json:"right"`

	// RateHz is the control-loop frequency in Hz. Default: 50.
	RateHz float64 `json:"rate_hz"`

	// CommandTimeoutMs is the failsafe window: if no intent is received within
	// this many milliseconds, the drive ramps to neutral. Default: 500.
	CommandTimeoutMs int `json:"command_timeout_ms"`

	// SlewPerS caps how fast an output may change, in normalized units per
	// second. 0 disables slew limiting (instant). Default: 3.0.
	SlewPerS float64 `json:"slew_per_s"`

	// YawGain scales the yaw contribution to the mix. Default: 1.0.
	YawGain float64 `json:"yaw_gain"`

	// Deadband zeroes intent magnitudes below this threshold. Default: 0.02.
	Deadband float64 `json:"deadband"`

	// InvertLeft/InvertRight flip the sign of an output for reversed mounts.
	InvertLeft  bool `json:"invert_left"`
	InvertRight bool `json:"invert_right"`
}

// DefaultConfig returns controller defaults for a 50 Hz tank-drive USV.
func DefaultConfig() Config {
	return Config{
		Left:             "left_thruster",
		Right:            "right_thruster",
		RateHz:           50.0,
		CommandTimeoutMs: 500,
		SlewPerS:         3.0,
		YawGain:          1.0,
		Deadband:         0.02,
	}
}

// ParseConfig builds a Config from a registry config map, applying defaults.
func ParseConfig(conf registry.Config) (Config, error) {
	c := resource.NewConfig(conf)
	cfg := DefaultConfig()

	if v, ok := c.GetString("left"); ok {
		cfg.Left = v
	}
	if v, ok := c.GetString("right"); ok {
		cfg.Right = v
	}
	if v, ok := c.GetFloat("rate_hz"); ok {
		cfg.RateHz = v
	}
	if v, ok := c.GetInt("command_timeout_ms"); ok {
		cfg.CommandTimeoutMs = v
	}
	if v, ok := c.GetFloat("slew_per_s"); ok {
		cfg.SlewPerS = v
	}
	if v, ok := c.GetFloat("yaw_gain"); ok {
		cfg.YawGain = v
	}
	if v, ok := c.GetFloat("deadband"); ok {
		cfg.Deadband = v
	}
	if v, ok := c.GetBool("invert_left"); ok {
		cfg.InvertLeft = v
	}
	if v, ok := c.GetBool("invert_right"); ok {
		cfg.InvertRight = v
	}

	return cfg, nil
}

// Validate checks that the configuration is internally consistent.
func (c Config) Validate() error {
	if c.Left == "" || c.Right == "" {
		return fmt.Errorf("left and right component names are required")
	}
	if c.Left == c.Right {
		return fmt.Errorf("left and right must be different components")
	}
	if c.RateHz <= 0 {
		return fmt.Errorf("rate_hz must be positive, got %f", c.RateHz)
	}
	if c.CommandTimeoutMs < 0 {
		return fmt.Errorf("command_timeout_ms must be non-negative, got %d", c.CommandTimeoutMs)
	}
	if c.SlewPerS < 0 {
		return fmt.Errorf("slew_per_s must be non-negative, got %f", c.SlewPerS)
	}
	if c.Deadband < 0 || c.Deadband >= 1 {
		return fmt.Errorf("deadband must be in [0, 1), got %f", c.Deadband)
	}
	return nil
}
