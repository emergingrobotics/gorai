package imu

import (
	"fmt"

	"github.com/gorai/gorai/pkg/resource"
)

// Config holds the configuration for the IMU formatter service.
type Config struct {
	// I2CBridgeComponent is the name of the I2C bridge component to subscribe to.
	I2CBridgeComponent string `json:"i2c_bridge_component"`

	// DeviceName is the device name as configured in the I2C bridge.
	DeviceName string `json:"device_name"`

	// DeviceType is the IMU device type for parsing (e.g., "mpu6050").
	DeviceType string `json:"device_type"`

	// I2CAddress is the expected I2C address for validation.
	I2CAddress uint8 `json:"i2c_address"`

	// AccelScale is the accelerometer scale setting (0-3).
	AccelScale int `json:"accel_scale"`

	// GyroScale is the gyroscope scale setting (0-3).
	GyroScale int `json:"gyro_scale"`

	// FrameID is the TF frame ID for output messages.
	FrameID string `json:"frame_id"`

	// PublishTemperature enables publishing temperature data.
	PublishTemperature bool `json:"publish_temperature"`
}

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		DeviceType: "mpu6050",
		I2CAddress: 0x68, // 104
		AccelScale: 0,
		GyroScale:  0,
		FrameID:    "imu_link",
	}

	// Required fields
	if val, ok := conf.Attributes["i2c_bridge_component"].(string); ok {
		cfg.I2CBridgeComponent = val
	}

	if val, ok := conf.Attributes["device_name"].(string); ok {
		cfg.DeviceName = val
	}

	// Optional fields with defaults
	if val, ok := conf.Attributes["device_type"].(string); ok {
		cfg.DeviceType = val
	}

	switch addr := conf.Attributes["i2c_address"].(type) {
	case float64:
		cfg.I2CAddress = uint8(addr)
	case int:
		cfg.I2CAddress = uint8(addr)
	}

	switch scale := conf.Attributes["accel_scale"].(type) {
	case float64:
		cfg.AccelScale = int(scale)
	case int:
		cfg.AccelScale = scale
	}

	switch scale := conf.Attributes["gyro_scale"].(type) {
	case float64:
		cfg.GyroScale = int(scale)
	case int:
		cfg.GyroScale = scale
	}

	if val, ok := conf.Attributes["frame_id"].(string); ok {
		cfg.FrameID = val
	}

	if val, ok := conf.Attributes["publish_temperature"].(bool); ok {
		cfg.PublishTemperature = val
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.I2CBridgeComponent == "" {
		return fmt.Errorf("i2c_bridge_component is required")
	}

	if c.DeviceName == "" {
		return fmt.Errorf("device_name is required")
	}

	if c.DeviceType != "mpu6050" {
		return fmt.Errorf("unsupported device_type: %s (supported: mpu6050)", c.DeviceType)
	}

	if c.AccelScale < 0 || c.AccelScale > 3 {
		return fmt.Errorf("accel_scale must be 0-3, got %d", c.AccelScale)
	}

	if c.GyroScale < 0 || c.GyroScale > 3 {
		return fmt.Errorf("gyro_scale must be 0-3, got %d", c.GyroScale)
	}

	if c.I2CAddress > 127 {
		return fmt.Errorf("i2c_address %d out of range (0-127)", c.I2CAddress)
	}

	return nil
}

