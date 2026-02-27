// Package remote provides a remote PWM component that publishes commands
// to the NATS message bus, bridged to a physical device via gorai-nats-gw.
package remote

import (
	"fmt"

	"github.com/gorai/gorai/pkg/resource"
)

// Config holds the configuration for the Remote PWM component.
type Config struct {
	// NATSSubjectPrefix is the NATS subject prefix matching gorai-nats-gw config.
	NATSSubjectPrefix string `json:"nats_subject_prefix"`

	// DeviceID is the device identifier in gorai-nats-gw (e.g., "pico-pwm").
	DeviceID string `json:"device_id"`

	// Channel is the GSP/2 PWM channel number on the device (e.g., 6 for GPIO6).
	Channel int `json:"channel"`

	// FrequencyHz is the PWM frequency in Hz. Default: 50 (servo).
	FrequencyHz float64 `json:"frequency_hz"`

	// MinPulseUs is the minimum pulse width in microseconds. Default: 1000.
	MinPulseUs float64 `json:"min_pulse_us"`

	// MaxPulseUs is the maximum pulse width in microseconds. Default: 2000.
	MaxPulseUs float64 `json:"max_pulse_us"`

	// InitialPulseUs is the pulse width to set on startup. Default: 1500.
	InitialPulseUs float64 `json:"initial_pulse_us"`
}

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		FrequencyHz:    50,
		MinPulseUs:     1000,
		MaxPulseUs:     2000,
		InitialPulseUs: 1500,
	}

	if val, ok := conf.Attributes["nats_subject_prefix"].(string); ok {
		cfg.NATSSubjectPrefix = val
	}

	if val, ok := conf.Attributes["device_id"].(string); ok {
		cfg.DeviceID = val
	}

	switch val := conf.Attributes["channel"].(type) {
	case float64:
		cfg.Channel = int(val)
	case int:
		cfg.Channel = val
	}

	switch val := conf.Attributes["frequency_hz"].(type) {
	case float64:
		cfg.FrequencyHz = val
	case int:
		cfg.FrequencyHz = float64(val)
	}

	switch val := conf.Attributes["min_pulse_us"].(type) {
	case float64:
		cfg.MinPulseUs = val
	case int:
		cfg.MinPulseUs = float64(val)
	}

	switch val := conf.Attributes["max_pulse_us"].(type) {
	case float64:
		cfg.MaxPulseUs = val
	case int:
		cfg.MaxPulseUs = float64(val)
	}

	switch val := conf.Attributes["initial_pulse_us"].(type) {
	case float64:
		cfg.InitialPulseUs = val
	case int:
		cfg.InitialPulseUs = float64(val)
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.NATSSubjectPrefix == "" {
		return fmt.Errorf("nats_subject_prefix is required")
	}

	if c.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}

	if c.Channel < 0 || c.Channel > 255 {
		return fmt.Errorf("channel must be between 0 and 255, got %d", c.Channel)
	}

	if c.FrequencyHz <= 0 {
		return fmt.Errorf("frequency_hz must be positive, got %f", c.FrequencyHz)
	}

	if c.MinPulseUs <= 0 {
		return fmt.Errorf("min_pulse_us must be positive, got %f", c.MinPulseUs)
	}

	if c.MaxPulseUs <= c.MinPulseUs {
		return fmt.Errorf("max_pulse_us (%f) must be greater than min_pulse_us (%f)", c.MaxPulseUs, c.MinPulseUs)
	}

	if c.InitialPulseUs < c.MinPulseUs || c.InitialPulseUs > c.MaxPulseUs {
		return fmt.Errorf("initial_pulse_us (%f) must be within min/max range [%f, %f]", c.InitialPulseUs, c.MinPulseUs, c.MaxPulseUs)
	}

	return nil
}

// CommandSubject returns the NATS subject for sending a command type to this device.
func (c *Config) CommandSubject(commandType string) string {
	return fmt.Sprintf("%s.%s.tx.command.%s", c.NATSSubjectPrefix, c.DeviceID, commandType)
}
