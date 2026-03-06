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

	// Pin is the GPIO pin number on the device (e.g., 6 for GPIO6).
	// Used as both the GPIO pin for configuration and the PWM channel for commands.
	Pin int `json:"pin"`

	// FrequencyHz is the PWM frequency in Hz. Default: 50 (servo).
	FrequencyHz float64 `json:"frequency_hz"`

	// MinPulseUs is the minimum pulse width in microseconds. Default: 1000.
	MinPulseUs float64 `json:"min_pulse_us"`

	// MaxPulseUs is the maximum pulse width in microseconds. Default: 2000.
	MaxPulseUs float64 `json:"max_pulse_us"`

	// InitialPulseUs is the pulse width to set on startup. Default: 1500.
	InitialPulseUs float64 `json:"initial_pulse_us"`

	// FailsafePulseUs is the pulse width applied when the device enters failsafe.
	// Default: center of min/max range.
	FailsafePulseUs float64 `json:"failsafe_pulse_us"`

	// AutoConfigure controls whether Start() sends the GPIO_CONFIG + PWM_CONFIG
	// provisioning sequence. Set to false for firmware that has pins preconfigured
	// at compile time (e.g. old TinyGo firmware). Default: true.
	AutoConfigure bool `json:"auto_configure"`
}

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		FrequencyHz:     50,
		MinPulseUs:      1000,
		MaxPulseUs:      2000,
		InitialPulseUs:  1500,
		FailsafePulseUs: -1,
		AutoConfigure:   true,
	}

	if val, ok := conf.Attributes["nats_subject_prefix"].(string); ok {
		cfg.NATSSubjectPrefix = val
	}

	if val, ok := conf.Attributes["device_id"].(string); ok {
		cfg.DeviceID = val
	}

	switch val := conf.Attributes["pin"].(type) {
	case float64:
		cfg.Pin = int(val)
	case int:
		cfg.Pin = val
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

	switch val := conf.Attributes["failsafe_pulse_us"].(type) {
	case float64:
		cfg.FailsafePulseUs = val
	case int:
		cfg.FailsafePulseUs = float64(val)
	}

	if val, ok := conf.Attributes["auto_configure"].(bool); ok {
		cfg.AutoConfigure = val
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

	if c.Pin < 0 || c.Pin > 28 {
		return fmt.Errorf("pin must be between 0 and 28, got %d", c.Pin)
	}

	if c.FrequencyHz <= 0 {
		return fmt.Errorf("frequency_hz must be positive, got %f", c.FrequencyHz)
	}

	if c.MinPulseUs < 0 {
		return fmt.Errorf("min_pulse_us must be non-negative, got %f", c.MinPulseUs)
	}

	if c.MaxPulseUs <= c.MinPulseUs {
		return fmt.Errorf("max_pulse_us (%f) must be greater than min_pulse_us (%f)", c.MaxPulseUs, c.MinPulseUs)
	}

	if c.InitialPulseUs < c.MinPulseUs || c.InitialPulseUs > c.MaxPulseUs {
		return fmt.Errorf("initial_pulse_us (%f) must be within min/max range [%f, %f]", c.InitialPulseUs, c.MinPulseUs, c.MaxPulseUs)
	}

	// Default failsafe to center of range if not explicitly set
	if c.FailsafePulseUs < 0 {
		c.FailsafePulseUs = (c.MinPulseUs + c.MaxPulseUs) / 2.0
	}

	if c.FailsafePulseUs < c.MinPulseUs || c.FailsafePulseUs > c.MaxPulseUs {
		return fmt.Errorf("failsafe_pulse_us (%f) must be within min/max range [%f, %f]", c.FailsafePulseUs, c.MinPulseUs, c.MaxPulseUs)
	}

	return nil
}

// CommandSubject returns the NATS subject for sending a command type to this device.
func (c *Config) CommandSubject(commandType string) string {
	return fmt.Sprintf("%s.%s.tx.command.%s", c.NATSSubjectPrefix, c.DeviceID, commandType)
}
