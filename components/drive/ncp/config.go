// Package ncp provides a generic NCP remote client for drive capabilities.
//
// It implements the drive.Drive interface by issuing NCP requests over NATS to
// a drive component exposed on another robot at
// gorai.<robot>.<component>.command. This is how a ground station commands a
// tank-drive controller running on the vehicle.
package ncp

import (
	"fmt"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds configuration for the NCP remote drive client.
type Config struct {
	// Robot is the target robot id (effective namespace) hosting the component.
	Robot string `json:"robot"`

	// Component is the target drive component name on the remote robot.
	Component string `json:"component"`

	// TimeoutMs bounds each remote request. Default: 2000.
	TimeoutMs int `json:"timeout_ms"`

	// ArmTimeoutMs bounds an arm request, which blocks while the remote runs
	// its (multi-second) arming sequence. Default: 15000.
	ArmTimeoutMs int `json:"arm_timeout_ms"`
}

// DefaultConfig returns configuration defaults for the drive client.
func DefaultConfig() Config {
	return Config{
		TimeoutMs:    2000,
		ArmTimeoutMs: 15000,
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
	if v, ok := c.GetInt("arm_timeout_ms"); ok {
		cfg.ArmTimeoutMs = v
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
