// Package bno085 implements a CEVA/Hillcrest BNO085 AHRS driver over I2C using
// the SH-2 protocol carried on SHTP. It enables the rotation vector,
// accelerometer, gyroscope, and magnetometer reports and exposes fused
// orientation plus calibration status through the sensor.AHRS interface.
package bno085

import (
	"fmt"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds configuration for the BNO085 AHRS.
type Config struct {
	// Bus is the I2C bus device path. Default: /dev/i2c-1.
	Bus string `json:"bus"`

	// Address is the I2C address (0x4A default, 0x4B when DI is high).
	Address uint16 `json:"address"`

	// ReportIntervalMs is the requested report cadence in milliseconds.
	// Default: 20 (50 Hz).
	ReportIntervalMs int `json:"report_interval_ms"`

	// Mounting maps each body axis to a signed IMU axis, describing how the
	// sensor is physically mounted. Default: identity.
	Mounting sensor.Mounting `json:"mounting"`

	// OffsetDeg is a hardcoded orientation offset (roll, pitch, yaw) in degrees
	// used as the zero reference until a runtime calibration overrides it.
	OffsetDeg [3]float64 `json:"offset_deg"`
}

// identityMounting is the no-remap mounting.
func identityMounting() sensor.Mounting {
	return sensor.Mounting{X: "+x", Y: "+y", Z: "+z"}
}

// DefaultConfig returns BNO085 defaults.
func DefaultConfig() Config {
	return Config{
		Bus:              "/dev/i2c-1",
		Address:          0x4A,
		ReportIntervalMs: 20,
		Mounting:         identityMounting(),
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
	if v, ok := c.GetInt("report_interval_ms"); ok {
		cfg.ReportIntervalMs = v
	}
	if m, ok := parseMounting(conf["mounting"]); ok {
		cfg.Mounting = m
	}
	if o, ok := parseOffset(conf["offset"]); ok {
		cfg.OffsetDeg = o
	}
	return cfg, nil
}

// parseMounting reads a {"x","y","z"} axis-remap object, falling back to
// identity for any field left unset.
func parseMounting(raw any) (sensor.Mounting, bool) {
	m, ok := raw.(map[string]any)
	if !ok {
		return sensor.Mounting{}, false
	}
	out := identityMounting()
	if v, ok := m["x"].(string); ok && v != "" {
		out.X = v
	}
	if v, ok := m["y"].(string); ok && v != "" {
		out.Y = v
	}
	if v, ok := m["z"].(string); ok && v != "" {
		out.Z = v
	}
	return out, true
}

// parseOffset reads a {"roll_deg","pitch_deg","yaw_deg"} object.
func parseOffset(raw any) ([3]float64, bool) {
	m, ok := raw.(map[string]any)
	if !ok {
		return [3]float64{}, false
	}
	var out [3]float64
	if v, ok := toFloatCfg(m["roll_deg"]); ok {
		out[0] = v
	}
	if v, ok := toFloatCfg(m["pitch_deg"]); ok {
		out[1] = v
	}
	if v, ok := toFloatCfg(m["yaw_deg"]); ok {
		out[2] = v
	}
	return out, true
}

// toFloatCfg coerces a JSON-decoded numeric value to float64.
func toFloatCfg(v any) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}

// Validate checks the configuration.
func (c Config) Validate() error {
	if c.Bus == "" {
		return fmt.Errorf("bus is required")
	}
	if c.ReportIntervalMs <= 0 {
		return fmt.Errorf("report_interval_ms must be positive, got %d", c.ReportIntervalMs)
	}
	if _, err := mountingMatrix(c.Mounting); err != nil {
		return fmt.Errorf("invalid mounting: %w", err)
	}
	return nil
}
