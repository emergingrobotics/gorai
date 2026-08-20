// Package bmp180 implements a Bosch BMP180 barometric pressure/temperature
// sensor driver over I2C. It reads factory calibration coefficients and applies
// the datasheet compensation to produce pressure (Pa), temperature (°C), and
// estimated altitude (m) through the generic sensor readings interface.
package bmp180

import (
	"fmt"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds configuration for the BMP180 sensor.
type Config struct {
	// Bus is the I2C bus device path. Default: /dev/i2c-1.
	Bus string `json:"bus"`

	// Address is the I2C address. Default: 0x77 (fixed for BMP180).
	Address uint16 `json:"address"`

	// Oversampling selects pressure oversampling (0-3). Higher is lower-noise
	// but slower. Default: 3 (ultra high resolution).
	Oversampling uint `json:"oversampling"`
}

// DefaultConfig returns BMP180 defaults.
func DefaultConfig() Config {
	return Config{
		Bus:          "/dev/i2c-1",
		Address:      0x77,
		Oversampling: 3,
	}
}

// ParseConfig builds a Config from a registry config map, applying defaults.
func ParseConfig(conf registry.Config) (Config, error) {
	c := resource.NewConfig(conf)
	cfg := DefaultConfig()

	if v, ok := c.GetString("bus"); ok {
		cfg.Bus = v
	}
	if v, ok := c.GetInt("address"); ok {
		cfg.Address = uint16(v)
	}
	if v, ok := c.GetInt("oversampling"); ok {
		cfg.Oversampling = uint(v)
	}
	return cfg, nil
}

// Validate checks the configuration.
func (c Config) Validate() error {
	if c.Bus == "" {
		return fmt.Errorf("bus is required")
	}
	if c.Oversampling > 3 {
		return fmt.Errorf("oversampling must be 0-3, got %d", c.Oversampling)
	}
	return nil
}
