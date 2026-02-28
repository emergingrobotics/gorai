// Package remote provides a remote GPIO component that publishes commands
// to the NATS message bus, bridged to a physical device via gorai-nats-gw.
package remote

import (
	"fmt"

	"github.com/gorai/gorai/pkg/resource"
)

// Config holds the configuration for the Remote GPIO component.
type Config struct {
	// NATSSubjectPrefix is the NATS subject prefix matching gorai-nats-gw config.
	NATSSubjectPrefix string `json:"nats_subject_prefix"`

	// DeviceID is the device identifier in gorai-nats-gw (e.g., "gsp-pico").
	DeviceID string `json:"device_id"`

	// Pin is the GPIO pin number on the device (0-28 for RP2040).
	Pin int `json:"pin"`

	// Mode is the pin direction: "input" or "output".
	Mode string `json:"mode"`

	// Pull is the pull resistor setting: "none", "up", or "down". Default: "none".
	Pull string `json:"pull"`

	// InitialValue is the output value to set on startup (0 or 1). Only for output mode.
	InitialValue uint8 `json:"initial_value"`
}

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		Pull: "none",
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

	if val, ok := conf.Attributes["mode"].(string); ok {
		cfg.Mode = val
	}

	if val, ok := conf.Attributes["pull"].(string); ok {
		cfg.Pull = val
	}

	switch val := conf.Attributes["initial_value"].(type) {
	case float64:
		cfg.InitialValue = uint8(val)
	case int:
		cfg.InitialValue = uint8(val)
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

	switch c.Mode {
	case "input", "output":
	default:
		return fmt.Errorf("mode must be \"input\" or \"output\", got %q", c.Mode)
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

// ModeID returns the GSP/2 mode byte for this config.
func (c *Config) ModeID() uint8 {
	if c.Mode == "output" {
		return 0x01
	}
	return 0x00
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
