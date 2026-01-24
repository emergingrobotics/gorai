package imu

import "fmt"

// MPU6050 constants
const (
	// MPU6050RequiredBytes is the minimum bytes needed (14 bytes from register 0x3B).
	MPU6050RequiredBytes = 14

	// MPU6050DefaultAddress is the default I2C address (AD0 = 0).
	MPU6050DefaultAddress = 0x68

	// MPU6050AltAddress is the alternate I2C address (AD0 = 1).
	MPU6050AltAddress = 0x69
)

// MPU6050 accelerometer sensitivity values (LSB/g) for each scale setting.
var mpu6050AccelSensitivity = []float64{
	16384.0, // ±2g
	8192.0,  // ±4g
	4096.0,  // ±8g
	2048.0,  // ±16g
}

// MPU6050 gyroscope sensitivity values (LSB/(°/s)) for each scale setting.
var mpu6050GyroSensitivity = []float64{
	131.0, // ±250°/s
	65.5,  // ±500°/s
	32.8,  // ±1000°/s
	16.4,  // ±2000°/s
}

// MPU6050Parser parses raw data from the MPU6050 IMU.
type MPU6050Parser struct {
	accelScale      int
	gyroScale       int
	accelSensitivity float64
	gyroSensitivity  float64
}

// NewMPU6050Parser creates a new MPU6050 parser with the specified scale settings.
func NewMPU6050Parser(accelScale, gyroScale int) *MPU6050Parser {
	// Clamp scales to valid range
	if accelScale < 0 {
		accelScale = 0
	} else if accelScale > 3 {
		accelScale = 3
	}
	if gyroScale < 0 {
		gyroScale = 0
	} else if gyroScale > 3 {
		gyroScale = 3
	}

	return &MPU6050Parser{
		accelScale:       accelScale,
		gyroScale:        gyroScale,
		accelSensitivity: mpu6050AccelSensitivity[accelScale],
		gyroSensitivity:  mpu6050GyroSensitivity[gyroScale],
	}
}

// Parse converts raw MPU6050 bytes to IMU data and temperature.
func (p *MPU6050Parser) Parse(data []byte) (*IMUData, float64, error) {
	if len(data) < MPU6050RequiredBytes {
		return nil, 0, &ParseError{
			Message: fmt.Sprintf("insufficient data: need %d bytes, got %d", MPU6050RequiredBytes, len(data)),
		}
	}

	// Parse 16-bit signed values (big-endian)
	accelXRaw := parseInt16BE(data[0:2])
	accelYRaw := parseInt16BE(data[2:4])
	accelZRaw := parseInt16BE(data[4:6])
	tempRaw := parseInt16BE(data[6:8])
	gyroXRaw := parseInt16BE(data[8:10])
	gyroYRaw := parseInt16BE(data[10:12])
	gyroZRaw := parseInt16BE(data[12:14])

	// Convert accelerometer to m/s²
	// Formula: accel = (raw / sensitivity) * gravity
	accelX := (float64(accelXRaw) / p.accelSensitivity) * Gravity
	accelY := (float64(accelYRaw) / p.accelSensitivity) * Gravity
	accelZ := (float64(accelZRaw) / p.accelSensitivity) * Gravity

	// Convert gyroscope to rad/s
	// Formula: gyro = (raw / sensitivity) * deg_to_rad
	gyroX := (float64(gyroXRaw) / p.gyroSensitivity) * DegToRad
	gyroY := (float64(gyroYRaw) / p.gyroSensitivity) * DegToRad
	gyroZ := (float64(gyroZRaw) / p.gyroSensitivity) * DegToRad

	// Convert temperature to °C
	// Formula from MPU6050 datasheet: Temp = (raw / 340) + 36.53
	temperature := (float64(tempRaw) / 340.0) + 36.53

	return &IMUData{
		AccelX: accelX,
		AccelY: accelY,
		AccelZ: accelZ,
		GyroX:  gyroX,
		GyroY:  gyroY,
		GyroZ:  gyroZ,
	}, temperature, nil
}

// RequiredBytes returns the minimum bytes required for parsing.
func (p *MPU6050Parser) RequiredBytes() int {
	return MPU6050RequiredBytes
}

// parseInt16BE parses a 16-bit signed integer in big-endian format.
func parseInt16BE(data []byte) int16 {
	return int16(data[0])<<8 | int16(data[1])
}

// AccelVariance returns the accelerometer variance based on scale setting.
// Based on MPU6050 datasheet: 400 µg/√Hz noise density.
func (p *MPU6050Parser) AccelVariance() float64 {
	// Noise density: 400 µg/√Hz = 400e-6 g/√Hz
	// Convert to m/s²: 400e-6 * 9.80665 ≈ 0.00392 m/s²/√Hz
	// Variance at 100 Hz bandwidth: (0.00392)² * 100 ≈ 0.00154 (m/s²)²
	// Round to 0.0016 for simplicity
	baseVariance := 0.0016
	// Scale increases noise proportionally (approximation)
	scaleMultiplier := uint(1) << uint(p.accelScale)
	return baseVariance * float64(scaleMultiplier)
}

// GyroVariance returns the gyroscope variance based on scale setting.
// Based on MPU6050 datasheet: 0.005 °/s/√Hz noise density.
func (p *MPU6050Parser) GyroVariance() float64 {
	// Noise density: 0.005 °/s/√Hz
	// Convert to rad/s: 0.005 * π/180 ≈ 8.73e-5 rad/s/√Hz
	// Variance at 100 Hz bandwidth: (8.73e-5)² * 100 ≈ 7.6e-7 (rad/s)²
	baseVariance := 7.6e-7
	// Scale increases noise proportionally (approximation)
	scaleMultiplier := uint(1) << uint(p.gyroScale)
	return baseVariance * float64(scaleMultiplier)
}

