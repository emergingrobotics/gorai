package keyboard

import (
	"fmt"

	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds the configuration for the keyboard component.
type Config struct {
	// Device is the path to the input device (e.g., "/dev/input/event0").
	Device string `json:"device"`

	// GrabDevice exclusively grabs the device, preventing other applications
	// from receiving events. Use with caution in desktop environments.
	GrabDevice bool `json:"grab_device"`

	// PublishRepeat enables publishing of key repeat events (held keys).
	PublishRepeat bool `json:"publish_repeat"`
}

// NewConfigFromResource parses a resource.Config into a keyboard Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		GrabDevice:    false,
		PublishRepeat: false,
	}

	// Parse device path (required)
	if device, ok := conf.Attributes["device"].(string); ok {
		cfg.Device = device
	}

	// Parse grab_device (optional)
	if grab, ok := conf.Attributes["grab_device"].(bool); ok {
		cfg.GrabDevice = grab
	}

	// Parse publish_repeat (optional)
	if repeat, ok := conf.Attributes["publish_repeat"].(bool); ok {
		cfg.PublishRepeat = repeat
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Device == "" {
		return fmt.Errorf("device path is required")
	}
	return nil
}

