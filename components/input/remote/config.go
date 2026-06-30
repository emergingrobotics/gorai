package remote

import (
	"fmt"
	"time"

	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds the configuration for the Remote Keyboard component.
type Config struct {
	// Subject is the NATS subject to subscribe to for keyboard events.
	Subject string `json:"subject"`

	// BufferSize is the size of the internal event channel buffer.
	BufferSize int `json:"buffer_size"`

	// StaleThresholdMs is the time after which connection is considered stale.
	StaleThresholdMs int64 `json:"stale_threshold_ms"`

	// AutoReleaseOnDisconnect automatically releases all keys when connection is lost.
	AutoReleaseOnDisconnect bool `json:"auto_release_on_disconnect"`
}

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		BufferSize:              100,
		StaleThresholdMs:        5000,
		AutoReleaseOnDisconnect: true,
	}

	// Parse subject (required)
	if val, ok := conf.Attributes["subject"].(string); ok {
		cfg.Subject = val
	}

	// Parse buffer_size
	switch val := conf.Attributes["buffer_size"].(type) {
	case float64:
		cfg.BufferSize = int(val)
	case int:
		cfg.BufferSize = val
	}

	// Parse stale_threshold_ms
	switch val := conf.Attributes["stale_threshold_ms"].(type) {
	case float64:
		cfg.StaleThresholdMs = int64(val)
	case int:
		cfg.StaleThresholdMs = int64(val)
	case int64:
		cfg.StaleThresholdMs = val
	}

	// Parse auto_release_on_disconnect
	if val, ok := conf.Attributes["auto_release_on_disconnect"].(bool); ok {
		cfg.AutoReleaseOnDisconnect = val
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if c.BufferSize < 1 {
		return fmt.Errorf("buffer_size must be at least 1, got %d", c.BufferSize)
	}

	if c.StaleThresholdMs < 100 {
		return fmt.Errorf("stale_threshold_ms must be at least 100, got %d", c.StaleThresholdMs)
	}

	return nil
}

// StaleThreshold returns the stale threshold as a duration.
func (c *Config) StaleThreshold() time.Duration {
	return time.Duration(c.StaleThresholdMs) * time.Millisecond
}

