// Package remote provides a remote camera component that subscribes to
// camera frames from the NATS message bus.
package remote

import (
	"fmt"
	"time"

	"github.com/gorai/gorai/pkg/resource"
)

// Config holds the configuration for the Remote Camera component.
type Config struct {
	// Topic is the NATS topic to subscribe to for camera frames.
	Topic string `json:"topic"`

	// Width is the expected frame width in pixels.
	Width int `json:"width"`

	// Height is the expected frame height in pixels.
	Height int `json:"height"`

	// BufferSize is the size of the internal frame ring buffer.
	BufferSize int `json:"buffer_size"`

	// StaleThresholdMs is the time after which connection is considered stale.
	StaleThresholdMs int64 `json:"stale_threshold_ms"`
}

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		Width:            640,
		Height:           480,
		BufferSize:       10,
		StaleThresholdMs: 5000,
	}

	// Parse topic (required)
	if val, ok := conf.Attributes["topic"].(string); ok {
		cfg.Topic = val
	}

	// Parse width
	switch val := conf.Attributes["width"].(type) {
	case float64:
		cfg.Width = int(val)
	case int:
		cfg.Width = val
	}

	// Parse height
	switch val := conf.Attributes["height"].(type) {
	case float64:
		cfg.Height = int(val)
	case int:
		cfg.Height = val
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

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Topic == "" {
		return fmt.Errorf("topic is required")
	}

	if c.Width < 1 {
		return fmt.Errorf("width must be at least 1, got %d", c.Width)
	}

	if c.Height < 1 {
		return fmt.Errorf("height must be at least 1, got %d", c.Height)
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
