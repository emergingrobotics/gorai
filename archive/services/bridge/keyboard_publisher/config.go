package keyboard_publisher

import (
	"fmt"
	"time"

	"github.com/gorai/gorai/pkg/resource"
)

// Config holds the configuration for the Keyboard Publisher service.
type Config struct {
	// Keyboard is the name of the local keyboard component to subscribe to.
	Keyboard string `json:"keyboard"`

	// Topic is the NATS topic to publish events to.
	// Default: gorai.<robot_name>.keyboard.events
	Topic string `json:"topic"`

	// PublishRepeat determines whether to publish key repeat events.
	PublishRepeat bool `json:"publish_repeat"`

	// HeartbeatIntervalMs is the interval for heartbeat/status messages.
	HeartbeatIntervalMs int `json:"heartbeat_interval_ms"`

	// IncludeModifiers determines whether to include modifier state in events.
	IncludeModifiers bool `json:"include_modifiers"`

	// RobotName is used for default topic construction.
	RobotName string `json:"robot_name"`
}

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		PublishRepeat:       false,
		HeartbeatIntervalMs: 1000,
		IncludeModifiers:    true,
	}

	// Parse keyboard (required)
	if val, ok := conf.Attributes["keyboard"].(string); ok {
		cfg.Keyboard = val
	}

	// Parse topic
	if val, ok := conf.Attributes["topic"].(string); ok {
		cfg.Topic = val
	}

	// Parse publish_repeat
	if val, ok := conf.Attributes["publish_repeat"].(bool); ok {
		cfg.PublishRepeat = val
	}

	// Parse heartbeat_interval_ms
	switch val := conf.Attributes["heartbeat_interval_ms"].(type) {
	case float64:
		cfg.HeartbeatIntervalMs = int(val)
	case int:
		cfg.HeartbeatIntervalMs = val
	}

	// Parse include_modifiers
	if val, ok := conf.Attributes["include_modifiers"].(bool); ok {
		cfg.IncludeModifiers = val
	}

	// Parse robot_name (from runtime config)
	if val, ok := conf.Attributes["robot_name"].(string); ok {
		cfg.RobotName = val
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Keyboard == "" {
		return fmt.Errorf("keyboard component name is required")
	}

	if c.HeartbeatIntervalMs < 100 {
		return fmt.Errorf("heartbeat_interval_ms must be at least 100, got %d", c.HeartbeatIntervalMs)
	}

	return nil
}

// HeartbeatInterval returns the heartbeat interval as a duration.
func (c *Config) HeartbeatInterval() time.Duration {
	return time.Duration(c.HeartbeatIntervalMs) * time.Millisecond
}

// GetTopic returns the publish topic, using default if not specified.
func (c *Config) GetTopic() string {
	if c.Topic != "" {
		return c.Topic
	}
	if c.RobotName != "" {
		return fmt.Sprintf("gorai.%s.keyboard.events", c.RobotName)
	}
	return "gorai.keyboard.events"
}

