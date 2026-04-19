package telemetry

import (
	"fmt"
	"time"

	"github.com/gorai/gorai/pkg/resource"
)

// SourceType represents the type of a data source.
type SourceType string

const (
	SourceTypeIMU         SourceType = "imu"
	SourceTypeGPS         SourceType = "gps"
	SourceTypeBattery     SourceType = "battery"
	SourceTypeTemperature SourceType = "temperature"
	SourceTypeRange       SourceType = "range"
	SourceTypeGeneric     SourceType = "generic"
)

// Config holds the configuration for the Telemetry service.
type Config struct {
	// PublishRateHz is the telemetry publish rate in Hz.
	PublishRateHz float64 `json:"publish_rate_hz"`

	// RobotID is the robot identifier for topic construction.
	RobotID string `json:"robot_id"`

	// Sources is the list of sensor sources to aggregate.
	Sources []*SourceConfig `json:"sources"`

	// IncludeSystemStats enables CPU/memory/temp stats.
	IncludeSystemStats bool `json:"include_system_stats"`

	// IncludeMotorStates enables motor controller state aggregation.
	IncludeMotorStates bool `json:"include_motor_states"`

	// MotorStateSubject is the subject for motor state messages.
	MotorStateSubject string `json:"motor_state_subject"`
}

// SourceConfig holds the configuration for a single data source.
type SourceConfig struct {
	// Name is a unique identifier for this source.
	Name string `json:"name"`

	// Subject is the NATS subject to subscribe to.
	Subject string `json:"subject"`

	// Type is the source type (imu, gps, battery, etc.).
	Type SourceType `json:"type"`

	// Required marks this source as required for healthy status.
	Required bool `json:"required"`

	// StaleThresholdMs is the time after which data is considered stale.
	StaleThresholdMs int64 `json:"stale_threshold_ms"`

	// Computed fields
	staleThreshold time.Duration
}

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		PublishRateHz:      10,
		IncludeSystemStats: true,
		IncludeMotorStates: false,
	}

	// Parse publish_rate_hz
	switch val := conf.Attributes["publish_rate_hz"].(type) {
	case float64:
		cfg.PublishRateHz = val
	case int:
		cfg.PublishRateHz = float64(val)
	}

	// Parse robot_id (required)
	if val, ok := conf.Attributes["robot_id"].(string); ok {
		cfg.RobotID = val
	}

	// Parse include_system_stats
	if val, ok := conf.Attributes["include_system_stats"].(bool); ok {
		cfg.IncludeSystemStats = val
	}

	// Parse include_motor_states
	if val, ok := conf.Attributes["include_motor_states"].(bool); ok {
		cfg.IncludeMotorStates = val
	}

	// Parse motor_state_subject
	if val, ok := conf.Attributes["motor_state_subject"].(string); ok {
		cfg.MotorStateSubject = val
	}

	// Parse sources
	if sourcesRaw, ok := conf.Attributes["sources"].([]any); ok {
		for i, srcRaw := range sourcesRaw {
			srcMap, ok := srcRaw.(map[string]any)
			if !ok {
				return nil, fmt.Errorf("source %d: expected map, got %T", i, srcRaw)
			}

			src, err := parseSourceConfig(srcMap)
			if err != nil {
				return nil, fmt.Errorf("source %d: %w", i, err)
			}
			cfg.Sources = append(cfg.Sources, src)
		}
	}

	return cfg, nil
}

// parseSourceConfig parses a single source configuration.
func parseSourceConfig(m map[string]any) (*SourceConfig, error) {
	src := &SourceConfig{
		Required:         false,
		StaleThresholdMs: 1000,
	}

	// Parse name (required)
	if val, ok := m["name"].(string); ok {
		src.Name = val
	}

	// Parse subject (required)
	if val, ok := m["subject"].(string); ok {
		src.Subject = val
	}

	// Parse type (required)
	if val, ok := m["type"].(string); ok {
		src.Type = SourceType(val)
	}

	// Parse required
	if val, ok := m["required"].(bool); ok {
		src.Required = val
	}

	// Parse stale_threshold_ms
	switch val := m["stale_threshold_ms"].(type) {
	case float64:
		src.StaleThresholdMs = int64(val)
	case int:
		src.StaleThresholdMs = int64(val)
	case int64:
		src.StaleThresholdMs = val
	}

	// Compute duration
	src.staleThreshold = time.Duration(src.StaleThresholdMs) * time.Millisecond

	return src, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.RobotID == "" {
		return fmt.Errorf("robot_id is required")
	}

	if c.PublishRateHz <= 0 {
		return fmt.Errorf("publish_rate_hz must be positive, got %f", c.PublishRateHz)
	}

	if c.PublishRateHz > 100 {
		return fmt.Errorf("publish_rate_hz %f exceeds maximum (100)", c.PublishRateHz)
	}

	if c.IncludeMotorStates && c.MotorStateSubject == "" {
		return fmt.Errorf("motor_state_subject is required when include_motor_states is true")
	}

	// Validate sources
	sourceNames := make(map[string]bool)
	for i, src := range c.Sources {
		if err := src.Validate(); err != nil {
			return fmt.Errorf("source %d (%s): %w", i, src.Name, err)
		}

		if sourceNames[src.Name] {
			return fmt.Errorf("duplicate source name: %s", src.Name)
		}
		sourceNames[src.Name] = true
	}

	return nil
}

// Validate checks if the source configuration is valid.
func (s *SourceConfig) Validate() error {
	if s.Name == "" {
		return fmt.Errorf("name is required")
	}

	if s.Subject == "" {
		return fmt.Errorf("subject is required")
	}

	if !s.Type.IsValid() {
		return fmt.Errorf("invalid source type: %s", s.Type)
	}

	if s.StaleThresholdMs <= 0 {
		return fmt.Errorf("stale_threshold_ms must be positive, got %d", s.StaleThresholdMs)
	}

	return nil
}

// IsValid returns true if the source type is valid.
func (t SourceType) IsValid() bool {
	switch t {
	case SourceTypeIMU, SourceTypeGPS, SourceTypeBattery,
		SourceTypeTemperature, SourceTypeRange, SourceTypeGeneric:
		return true
	default:
		return false
	}
}

// PublishInterval returns the interval between published frames.
func (c *Config) PublishInterval() time.Duration {
	return time.Duration(float64(time.Second) / c.PublishRateHz)
}

// StaleThreshold returns the stale threshold duration.
func (s *SourceConfig) StaleThreshold() time.Duration {
	if s.staleThreshold == 0 {
		s.staleThreshold = time.Duration(s.StaleThresholdMs) * time.Millisecond
	}
	return s.staleThreshold
}

// GetSourceConfig returns the source config by name.
func (c *Config) GetSourceConfig(name string) *SourceConfig {
	for _, src := range c.Sources {
		if src.Name == name {
			return src
		}
	}
	return nil
}

