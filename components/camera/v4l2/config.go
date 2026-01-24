package v4l2

import (
	"fmt"
	"time"

	"github.com/gorai/gorai/pkg/resource"
)

// Config holds the configuration for the V4L2 camera component.
type Config struct {
	// Device is the V4L2 device path (e.g., "/dev/video0").
	Device string `json:"device"`

	// Width is the image width in pixels.
	Width int `json:"width"`

	// Height is the image height in pixels.
	Height int `json:"height"`

	// FrameRate is the capture frame rate in Hz.
	FrameRate float64 `json:"frame_rate"`

	// JPEGQuality is the JPEG compression quality (1-100).
	JPEGQuality int `json:"jpeg_quality"`

	// PublishToBus enables NATS message publishing.
	PublishToBus bool `json:"publish_to_bus"`

	// PublishRateHz is the NATS publish rate (can be lower than capture rate).
	PublishRateHz float64 `json:"publish_rate_hz"`

	// FrameID is the TF frame ID for output messages.
	FrameID string `json:"frame_id"`
}

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		Device:        "/dev/video0",
		Width:         640,
		Height:        480,
		FrameRate:     30,
		JPEGQuality:   80,
		PublishToBus:  true,
		PublishRateHz: 15,
		FrameID:       "camera_link",
	}

	// Parse device path
	if val, ok := conf.Attributes["device"].(string); ok {
		cfg.Device = val
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

	// Parse frame rate
	switch val := conf.Attributes["frame_rate"].(type) {
	case float64:
		cfg.FrameRate = val
	case int:
		cfg.FrameRate = float64(val)
	}

	// Parse JPEG quality
	switch val := conf.Attributes["jpeg_quality"].(type) {
	case float64:
		cfg.JPEGQuality = int(val)
	case int:
		cfg.JPEGQuality = val
	}

	// Parse publish_to_bus
	if val, ok := conf.Attributes["publish_to_bus"].(bool); ok {
		cfg.PublishToBus = val
	}

	// Parse publish rate
	switch val := conf.Attributes["publish_rate_hz"].(type) {
	case float64:
		cfg.PublishRateHz = val
	case int:
		cfg.PublishRateHz = float64(val)
	}

	// Parse frame ID
	if val, ok := conf.Attributes["frame_id"].(string); ok {
		cfg.FrameID = val
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Device == "" {
		return fmt.Errorf("device path cannot be empty")
	}

	if c.Width <= 0 {
		return fmt.Errorf("width must be positive, got %d", c.Width)
	}

	if c.Height <= 0 {
		return fmt.Errorf("height must be positive, got %d", c.Height)
	}

	if c.FrameRate <= 0 {
		return fmt.Errorf("frame_rate must be positive, got %f", c.FrameRate)
	}

	if c.FrameRate > 120 {
		return fmt.Errorf("frame_rate %f exceeds maximum (120)", c.FrameRate)
	}

	if c.JPEGQuality < 1 || c.JPEGQuality > 100 {
		return fmt.Errorf("jpeg_quality must be 1-100, got %d", c.JPEGQuality)
	}

	if c.PublishRateHz <= 0 {
		return fmt.Errorf("publish_rate_hz must be positive, got %f", c.PublishRateHz)
	}

	if c.PublishRateHz > c.FrameRate {
		return fmt.Errorf("publish_rate_hz (%f) cannot exceed frame_rate (%f)", c.PublishRateHz, c.FrameRate)
	}

	return nil
}

// PublishInterval returns the interval between published frames.
func (c *Config) PublishInterval() time.Duration {
	if c.PublishRateHz >= c.FrameRate {
		return 0 // Publish every frame
	}
	return time.Duration(float64(time.Second) / c.PublishRateHz)
}

// CaptureInterval returns the interval between frame captures.
func (c *Config) CaptureInterval() time.Duration {
	return time.Duration(float64(time.Second) / c.FrameRate)
}

