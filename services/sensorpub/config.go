// Package sensorpub provides a service that polls sensor components at
// independent rates and publishes their readings as JSON telemetry on
// gorai.<robot>.<name>.data. It is deliberately decoupled from any control
// loop: each source runs on its own ticker.
package sensorpub

import (
	"fmt"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Source describes one sensor to poll and publish.
type Source struct {
	// Name is the sensor component name (resolved via dependencies).
	Name string `json:"name"`

	// Kind is a free-form label (gps|ahrs|imu|sensor) included in telemetry.
	Kind string `json:"kind"`

	// RateHz is the polling/publish rate for this source.
	RateHz float64 `json:"rate_hz"`
}

// Config holds configuration for the sensor publisher.
type Config struct {
	Sources []Source `json:"sources"`
}

// ParseConfig builds a Config from a registry config map.
func ParseConfig(conf registry.Config) (Config, error) {
	c := resource.NewConfig(conf)
	var cfg Config
	if err := c.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	for i := range cfg.Sources {
		if cfg.Sources[i].RateHz <= 0 {
			cfg.Sources[i].RateHz = 5
		}
	}
	return cfg, nil
}

// Validate checks the configuration.
func (c Config) Validate() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("at least one source is required")
	}
	for _, s := range c.Sources {
		if s.Name == "" {
			return fmt.Errorf("source name is required")
		}
	}
	return nil
}
