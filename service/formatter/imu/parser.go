package imu

import "math"

// Physical constants
const (
	// Gravity is the standard acceleration due to gravity in m/s².
	Gravity = 9.80665

	// DegToRad converts degrees to radians.
	DegToRad = math.Pi / 180.0
)

// IMUData holds parsed IMU sensor data in SI units.
type IMUData struct {
	// AccelX, AccelY, AccelZ are linear accelerations in m/s².
	AccelX float64
	AccelY float64
	AccelZ float64

	// GyroX, GyroY, GyroZ are angular velocities in rad/s.
	GyroX float64
	GyroY float64
	GyroZ float64
}

// Parser defines the interface for IMU data parsers.
type Parser interface {
	// Parse converts raw bytes to IMU data and temperature.
	// Returns IMU data in SI units, temperature in °C, and any error.
	Parse(data []byte) (*IMUData, float64, error)

	// RequiredBytes returns the minimum number of bytes required for parsing.
	RequiredBytes() int
}

// NewParser creates a parser for the specified device type.
func NewParser(deviceType string, accelScale, gyroScale int) (Parser, error) {
	switch deviceType {
	case "mpu6050":
		return NewMPU6050Parser(accelScale, gyroScale), nil
	default:
		return nil, &UnsupportedDeviceError{DeviceType: deviceType}
	}
}

// UnsupportedDeviceError indicates an unsupported device type.
type UnsupportedDeviceError struct {
	DeviceType string
}

func (e *UnsupportedDeviceError) Error() string {
	return "unsupported device type: " + e.DeviceType
}

// ParseError indicates an error parsing IMU data.
type ParseError struct {
	Message string
}

func (e *ParseError) Error() string {
	return e.Message
}

