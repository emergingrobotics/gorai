// Package ncp provides a generic NCP remote client for AHRS/orientation
// sensors.
//
// It implements the sensor.AHRS and sensor.OrientationConfigurable interfaces
// by issuing NCP requests over NATS to an AHRS component exposed on another
// robot at gorai.<robot>.<component>.command, and caches the orientation
// snapshot published on gorai.<robot>.<component>.state. This is how a ground
// station calibrates and reorients an IMU running on the vehicle.
package ncp

import (
	"fmt"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds configuration for the NCP remote AHRS client.
type Config struct {
	// Robot is the target robot id (effective namespace) hosting the component.
	Robot string `json:"robot"`

	// Component is the target AHRS component name on the remote robot.
	Component string `json:"component"`

	// TimeoutMs bounds each remote request. Default: 2000.
	TimeoutMs int `json:"timeout_ms"`

	// CalibrateTimeoutMs bounds a calibrate request, which blocks while the
	// remote samples orientation. Default: 5000.
	CalibrateTimeoutMs int `json:"calibrate_timeout_ms"`
}

// DefaultConfig returns configuration defaults for the AHRS client.
func DefaultConfig() Config {
	return Config{
		TimeoutMs:          2000,
		CalibrateTimeoutMs: 5000,
	}
}

// ParseConfig builds a Config from a registry config map, applying defaults.
func ParseConfig(conf registry.Config) (Config, error) {
	c := resource.NewConfig(conf)
	cfg := DefaultConfig()

	if v, ok := c.GetString("robot"); ok {
		cfg.Robot = v
	}
	if v, ok := c.GetString("component"); ok {
		cfg.Component = v
	}
	if v, ok := c.GetInt("timeout_ms"); ok {
		cfg.TimeoutMs = v
	}
	if v, ok := c.GetInt("calibrate_timeout_ms"); ok {
		cfg.CalibrateTimeoutMs = v
	}

	return cfg, nil
}

// Validate checks required fields.
func (c Config) Validate() error {
	if c.Robot == "" {
		return fmt.Errorf("robot is required (target robot id)")
	}
	if c.Component == "" {
		return fmt.Errorf("component is required (target component name)")
	}
	return nil
}
