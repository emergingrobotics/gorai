// Package config provides configuration loading and management.
package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Robot represents the top-level robot configuration.
type Robot struct {
	Components []ComponentConfig `json:"components"`
	Services   []ServiceConfig   `json:"services"`
}

// ComponentConfig represents a component configuration.
type ComponentConfig struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Model      string         `json:"model"`
	Attributes map[string]any `json:"attributes,omitempty"`
	DependsOn  []string       `json:"depends_on,omitempty"`
}

// ServiceConfig represents a service configuration.
type ServiceConfig struct {
	Name       string         `json:"name"`
	Type       string         `json:"type"`
	Model      string         `json:"model"`
	Attributes map[string]any `json:"attributes,omitempty"`
	DependsOn  []string       `json:"depends_on,omitempty"`
}

// Load loads configuration from a JSON file.
func Load(path string) (*Robot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Robot
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	return &cfg, nil
}

// LoadFromBytes loads configuration from JSON bytes.
func LoadFromBytes(data []byte) (*Robot, error) {
	var cfg Robot
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	return &cfg, nil
}
