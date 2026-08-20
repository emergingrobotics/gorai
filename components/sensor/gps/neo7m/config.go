// Package neo7m implements a u-blox NEO-7M GPS driver. It reads NMEA sentences
// from a serial port, parses GGA/RMC messages, and exposes them through the
// sensor.GPS interface.
package neo7m

import (
	"fmt"

	"github.com/emergingrobotics/gorai/driver/serial"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds configuration for the NEO-7M GPS.
type Config struct {
	// Path is the serial device path. Default: /dev/serial0.
	Path string `json:"path"`

	// BaudRate is the serial baud rate. Default: 9600 (NEO-7M factory default).
	BaudRate int `json:"baud_rate"`
}

// DefaultConfig returns NEO-7M defaults.
func DefaultConfig() Config {
	return Config{
		Path:     "/dev/serial0",
		BaudRate: 9600,
	}
}

// ParseConfig builds a Config from a registry config map, applying defaults.
func ParseConfig(conf registry.Config) (Config, error) {
	c := resource.NewConfig(conf)
	cfg := DefaultConfig()

	if v, ok := c.GetString("path"); ok {
		cfg.Path = v
	}
	if v, ok := c.GetInt("baud_rate"); ok {
		cfg.BaudRate = v
	}
	return cfg, nil
}

// Validate checks the configuration.
func (c Config) Validate() error {
	if c.Path == "" {
		return fmt.Errorf("path is required")
	}
	if c.BaudRate <= 0 {
		return fmt.Errorf("baud_rate must be positive, got %d", c.BaudRate)
	}
	return nil
}

// serialConfig converts the component config into a serial.Config.
func (c Config) serialConfig() serial.Config {
	sc := serial.DefaultConfig(c.Path)
	sc.BaudRate = c.BaudRate
	sc.ReadTimeout = 1000
	return sc
}
