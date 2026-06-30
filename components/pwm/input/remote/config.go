// Package remote provides a remote PWM input component that subscribes to
// PWM_INPUT_DATA messages from the NATS bus, bridged from a physical device
// via gorai-nats-gw.
package remote

import (
	"fmt"

	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds the configuration for the Remote PWM Input component.
type Config struct {
	// NATSSubjectPrefix is the NATS subject prefix matching gorai-nats-gw config.
	NATSSubjectPrefix string `json:"nats_subject_prefix"`

	// DeviceID is the device identifier in gorai-nats-gw (e.g., "gsp-pico").
	DeviceID string `json:"device_id"`

	// Pin is the GPIO pin number on the device (0-28 for RP2040).
	Pin int `json:"pin"`

	// Pull is the pull resistor setting: "none", "up", or "down". Default: "down".
	Pull string `json:"pull"`
}

const gpio_mode_pwm_input = 0x04

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		Pull: "down",
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

	if val, ok := conf.Attributes["pull"].(string); ok {
		cfg.Pull = val
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

	switch c.Pull {
	case "none", "up", "down":
	default:
		return fmt.Errorf("pull must be \"none\", \"up\", or \"down\", got %q", c.Pull)
	}

	return nil
}

// CommandSubject returns the NATS subject for sending a command type to this device.
func (c *Config) CommandSubject(command_type string) string {
	return fmt.Sprintf("%s.%s.tx.command.%s", c.NATSSubjectPrefix, c.DeviceID, command_type)
}

// PullID returns the GSP/2 pull byte for this config.
func (c *Config) PullID() uint8 {
	switch c.Pull {
	case "up":
		return 0x01
	case "down":
		return 0x02
	default:
		return 0x00
	}
}
