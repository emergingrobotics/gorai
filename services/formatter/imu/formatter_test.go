package imu

import (
	"math"
	"testing"
	"time"

	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid config",
			config: &Config{
				I2CBridgeComponent: "i2c_primary",
				DeviceName:         "mpu6050",
				DeviceType:         "mpu6050",
				I2CAddress:         0x68,
				AccelScale:         0,
				GyroScale:          0,
				FrameID:            "imu_link",
			},
			wantErr: false,
		},
		{
			name: "missing i2c_bridge_component",
			config: &Config{
				DeviceName: "mpu6050",
				DeviceType: "mpu6050",
			},
			wantErr: true,
			errMsg:  "i2c_bridge_component is required",
		},
		{
			name: "missing device_name",
			config: &Config{
				I2CBridgeComponent: "i2c_primary",
				DeviceType:         "mpu6050",
			},
			wantErr: true,
			errMsg:  "device_name is required",
		},
		{
			name: "unsupported device_type",
			config: &Config{
				I2CBridgeComponent: "i2c_primary",
				DeviceName:         "sensor",
				DeviceType:         "unknown_imu",
			},
			wantErr: true,
			errMsg:  "unsupported device_type",
		},
		{
			name: "accel_scale out of range",
			config: &Config{
				I2CBridgeComponent: "i2c_primary",
				DeviceName:         "mpu6050",
				DeviceType:         "mpu6050",
				AccelScale:         4,
			},
			wantErr: true,
			errMsg:  "accel_scale must be 0-3",
		},
		{
			name: "gyro_scale out of range",
			config: &Config{
				I2CBridgeComponent: "i2c_primary",
				DeviceName:         "mpu6050",
				DeviceType:         "mpu6050",
				GyroScale:          -1,
			},
			wantErr: true,
			errMsg:  "gyro_scale must be 0-3",
		},
		{
			name: "all scales valid",
			config: &Config{
				I2CBridgeComponent: "i2c_primary",
				DeviceName:         "mpu6050",
				DeviceType:         "mpu6050",
				AccelScale:         3,
				GyroScale:          3,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigParsing(t *testing.T) {
	attrs := map[string]any{
		"i2c_bridge_component": "i2c_primary",
		"device_name":          "mpu6050",
		"device_type":          "mpu6050",
		"i2c_address":          float64(104), // JSON numbers are float64
		"accel_scale":          float64(2),
		"gyro_scale":           float64(1),
		"frame_id":             "base_imu",
		"publish_temperature":  true,
	}

	conf := resource.NewConfig(attrs)
	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	assert.Equal(t, "i2c_primary", cfg.I2CBridgeComponent)
	assert.Equal(t, "mpu6050", cfg.DeviceName)
	assert.Equal(t, "mpu6050", cfg.DeviceType)
	assert.Equal(t, uint8(104), cfg.I2CAddress)
	assert.Equal(t, 2, cfg.AccelScale)
	assert.Equal(t, 1, cfg.GyroScale)
	assert.Equal(t, "base_imu", cfg.FrameID)
	assert.True(t, cfg.PublishTemperature)
}

func TestConfigDefaults(t *testing.T) {
	attrs := map[string]any{
		"i2c_bridge_component": "i2c_primary",
		"device_name":          "mpu6050",
	}

	conf := resource.NewConfig(attrs)
	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	// Check defaults
	assert.Equal(t, "mpu6050", cfg.DeviceType)
	assert.Equal(t, uint8(0x68), cfg.I2CAddress)
	assert.Equal(t, 0, cfg.AccelScale)
	assert.Equal(t, 0, cfg.GyroScale)
	assert.Equal(t, "imu_link", cfg.FrameID)
	assert.False(t, cfg.PublishTemperature)
}

func TestMPU6050Parsing(t *testing.T) {
	parser := NewMPU6050Parser(0, 0)

	// Test data: Z-axis at 1g, others at 0
	// Raw value 16384 = 1g at ±2g scale
	testData := []byte{
		0x00, 0x00, // Accel X = 0
		0x00, 0x00, // Accel Y = 0
		0x40, 0x00, // Accel Z = 16384 (0x4000)
		0x00, 0x00, // Temp = 0
		0x00, 0x00, // Gyro X = 0
		0x00, 0x00, // Gyro Y = 0
		0x00, 0x00, // Gyro Z = 0
	}

	imuData, temp, err := parser.Parse(testData)
	require.NoError(t, err)

	// Check accelerometer (should be ~1g in Z)
	assert.InDelta(t, 0.0, imuData.AccelX, 0.001)
	assert.InDelta(t, 0.0, imuData.AccelY, 0.001)
	assert.InDelta(t, Gravity, imuData.AccelZ, 0.001) // Should be ~9.80665 m/s²

	// Check gyroscope (should all be 0)
	assert.InDelta(t, 0.0, imuData.GyroX, 0.0001)
	assert.InDelta(t, 0.0, imuData.GyroY, 0.0001)
	assert.InDelta(t, 0.0, imuData.GyroZ, 0.0001)

	// Check temperature (raw 0 → (0/340) + 36.53 = 36.53°C)
	assert.InDelta(t, 36.53, temp, 0.01)
}

func TestMPU6050ScaleFactors(t *testing.T) {
	// Test all accelerometer scales
	accelTests := []struct {
		scale         int
		rawValue      int16
		expectedAccel float64 // in m/s²
	}{
		{0, 16384, Gravity},         // ±2g: 16384 LSB/g
		{1, 8192, Gravity},          // ±4g: 8192 LSB/g
		{2, 4096, Gravity},          // ±8g: 4096 LSB/g
		{3, 2048, Gravity},          // ±16g: 2048 LSB/g
		{0, -16384, -Gravity},       // Negative 1g
		{0, 32767, 2.0 * Gravity},   // Near max (almost 2g at ±2g scale)
	}

	for _, tt := range accelTests {
		t.Run("accel_scale", func(t *testing.T) {
			parser := NewMPU6050Parser(tt.scale, 0)
			data := make([]byte, 14)
			// Put raw value in Accel Z position (bytes 4-5)
			data[4] = byte(tt.rawValue >> 8)
			data[5] = byte(tt.rawValue & 0xFF)

			imuData, _, err := parser.Parse(data)
			require.NoError(t, err)
			assert.InDelta(t, tt.expectedAccel, imuData.AccelZ, 0.01)
		})
	}

	// Test all gyroscope scales
	// At scale 0: 131 LSB/(°/s), so 131 raw = 1°/s = π/180 rad/s
	gyroTests := []struct {
		scale        int
		rawValue     int16
		expectedGyro float64 // in rad/s
	}{
		{0, 131, DegToRad},        // ±250°/s: 131 LSB/(°/s) → 1°/s
		{1, 65, DegToRad},         // ±500°/s: 65.5 LSB/(°/s) → ~1°/s
		{2, 33, DegToRad},         // ±1000°/s: 32.8 LSB/(°/s) → ~1°/s
		{3, 16, DegToRad},         // ±2000°/s: 16.4 LSB/(°/s) → ~1°/s
		{0, -131, -DegToRad},      // Negative direction
	}

	for _, tt := range gyroTests {
		t.Run("gyro_scale", func(t *testing.T) {
			parser := NewMPU6050Parser(0, tt.scale)
			data := make([]byte, 14)
			// Put raw value in Gyro Z position (bytes 12-13)
			data[12] = byte(tt.rawValue >> 8)
			data[13] = byte(tt.rawValue & 0xFF)

			imuData, _, err := parser.Parse(data)
			require.NoError(t, err)
			assert.InDelta(t, tt.expectedGyro, imuData.GyroZ, 0.1*DegToRad) // Allow 0.1° tolerance
		})
	}
}

func TestMPU6050TemperatureConversion(t *testing.T) {
	parser := NewMPU6050Parser(0, 0)

	tests := []struct {
		name     string
		rawTemp  int16
		expected float64
	}{
		{"zero", 0, 36.53},
		{"positive", 340, 37.53},      // (340/340) + 36.53 = 37.53
		{"negative", -340, 35.53},     // (-340/340) + 36.53 = 35.53
		{"room_temp", 1190, 40.03},    // (~25°C offset from 36.53)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data := make([]byte, 14)
			// Put raw temp value in bytes 6-7
			data[6] = byte(tt.rawTemp >> 8)
			data[7] = byte(tt.rawTemp & 0xFF)

			_, temp, err := parser.Parse(data)
			require.NoError(t, err)
			assert.InDelta(t, tt.expected, temp, 0.01)
		})
	}
}

func TestMPU6050ParseError(t *testing.T) {
	parser := NewMPU6050Parser(0, 0)

	// Test insufficient data
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too_short", []byte{0x00, 0x01, 0x02}},
		{"13_bytes", make([]byte, 13)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := parser.Parse(tt.data)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "insufficient data")
		})
	}
}

func TestMPU6050Variance(t *testing.T) {
	// Test variance increases with scale
	for scale := 0; scale < 4; scale++ {
		parser := NewMPU6050Parser(scale, scale)

		accelVar := parser.AccelVariance()
		gyroVar := parser.GyroVariance()

		// Variance should be positive
		assert.Greater(t, accelVar, 0.0)
		assert.Greater(t, gyroVar, 0.0)

		// Higher scales should have higher variance
		if scale > 0 {
			prevParser := NewMPU6050Parser(scale-1, scale-1)
			assert.Greater(t, accelVar, prevParser.AccelVariance())
			assert.Greater(t, gyroVar, prevParser.GyroVariance())
		}
	}
}

func TestParserInterface(t *testing.T) {
	// Test NewParser factory
	parser, err := NewParser("mpu6050", 0, 0)
	require.NoError(t, err)
	assert.NotNil(t, parser)
	assert.Equal(t, 14, parser.RequiredBytes())

	// Test unsupported device
	_, err = NewParser("unknown_device", 0, 0)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported device type")
}

func TestSIUnitConversion(t *testing.T) {
	parser := NewMPU6050Parser(0, 0)

	// Create data with known values
	// Accel: X=0.5g, Y=-0.25g, Z=1g
	// Gyro: X=10°/s, Y=-5°/s, Z=0°/s
	data := []byte{
		0x20, 0x00, // Accel X = 8192 (0.5g at ±2g scale)
		0xF0, 0x00, // Accel Y = -4096 (-0.25g at ±2g scale)
		0x40, 0x00, // Accel Z = 16384 (1g at ±2g scale)
		0x00, 0x00, // Temp
		0x05, 0x1E, // Gyro X = 1310 (~10°/s at ±250°/s scale)
		0xFD, 0x71, // Gyro Y = -655 (~-5°/s at ±250°/s scale)
		0x00, 0x00, // Gyro Z = 0
	}

	imuData, _, err := parser.Parse(data)
	require.NoError(t, err)

	// Check accelerations are in m/s²
	assert.InDelta(t, 0.5*Gravity, imuData.AccelX, 0.1)
	assert.InDelta(t, -0.25*Gravity, imuData.AccelY, 0.1)
	assert.InDelta(t, Gravity, imuData.AccelZ, 0.01)

	// Check angular velocities are in rad/s
	expectedGyroX := 10.0 * DegToRad
	expectedGyroY := -5.0 * DegToRad
	assert.InDelta(t, expectedGyroX, imuData.GyroX, 0.02)
	assert.InDelta(t, expectedGyroY, imuData.GyroY, 0.02)
	assert.InDelta(t, 0.0, imuData.GyroZ, 0.001)
}

func TestFormatterProcessRawData(t *testing.T) {
	// Create formatter with mock config
	cfg := &Config{
		I2CBridgeComponent: "i2c_primary",
		DeviceName:         "mpu6050",
		DeviceType:         "mpu6050",
		I2CAddress:         0x68,
		AccelScale:         0,
		GyroScale:          0,
		FrameID:            "test_imu",
	}

	parser := NewMPU6050Parser(cfg.AccelScale, cfg.GyroScale)
	f := &Formatter{
		config:  cfg,
		parser:  parser,
		mpu6050: parser,
		state:   StateRunning,
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}

	// Track callback invocations
	var receivedMsg *IMUMessage
	f.SetIMUDataCallback(func(data *IMUMessage) {
		receivedMsg = data
	})

	// Valid data
	testData := []byte{
		0x00, 0x00, 0x00, 0x00, 0x40, 0x00, // Accel: 0, 0, 1g
		0x00, 0x00,                         // Temp
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Gyro: 0, 0, 0
	}

	err := f.ProcessRawData(testData, 0x68, "mpu6050", time.Now())
	require.NoError(t, err)

	// Verify callback was called
	require.NotNil(t, receivedMsg)
	assert.Equal(t, "test_imu", receivedMsg.FrameID)
	assert.InDelta(t, Gravity, receivedMsg.AccelZ, 0.01)
}

func TestFormatterAddressValidation(t *testing.T) {
	cfg := &Config{
		I2CBridgeComponent: "i2c_primary",
		DeviceName:         "mpu6050",
		DeviceType:         "mpu6050",
		I2CAddress:         0x68,
	}

	parser := NewMPU6050Parser(0, 0)
	f := &Formatter{
		config:  cfg,
		parser:  parser,
		mpu6050: parser,
		state:   StateRunning,
	}

	testData := make([]byte, 14)

	// Wrong address
	err := f.ProcessRawData(testData, 0x69, "mpu6050", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "address mismatch")
	assert.Equal(t, uint64(1), f.validationErrors.Load())

	// Wrong device name
	err = f.ProcessRawData(testData, 0x68, "other_device", time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "device mismatch")
	assert.Equal(t, uint64(2), f.validationErrors.Load())
}

func TestFormatterStats(t *testing.T) {
	cfg := &Config{
		I2CBridgeComponent: "i2c_primary",
		DeviceName:         "mpu6050",
		DeviceType:         "mpu6050",
		I2CAddress:         0x68,
	}

	parser := NewMPU6050Parser(0, 0)
	f := &Formatter{
		config:  cfg,
		parser:  parser,
		mpu6050: parser,
		state:   StateRunning,
	}

	testData := make([]byte, 14)

	// Process valid data
	for i := 0; i < 10; i++ {
		_ = f.ProcessRawData(testData, 0x68, "mpu6050", time.Now())
	}

	// Process invalid data (short)
	for i := 0; i < 3; i++ {
		_ = f.ProcessRawData([]byte{0x00}, 0x68, "mpu6050", time.Now())
	}

	received, published, parseErr, valErr := f.GetStats()
	assert.Equal(t, uint64(13), received)
	assert.Equal(t, uint64(10), published)
	assert.Equal(t, uint64(3), parseErr)
	assert.Equal(t, uint64(0), valErr)
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateStopped, "stopped"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateError, "error"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestInt16BEParsing(t *testing.T) {
	tests := []struct {
		name   string
		data   []byte
		expect int16
	}{
		{"zero", []byte{0x00, 0x00}, 0},
		{"positive", []byte{0x40, 0x00}, 16384},
		{"max_positive", []byte{0x7F, 0xFF}, 32767},
		{"negative_one", []byte{0xFF, 0xFF}, -1},
		{"min_negative", []byte{0x80, 0x00}, -32768},
		{"mid_negative", []byte{0xC0, 0x00}, -16384},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt16BE(tt.data)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestGravityConstant(t *testing.T) {
	// Verify the gravity constant is correct
	assert.InDelta(t, 9.80665, Gravity, 0.00001)
}

func TestDegToRadConstant(t *testing.T) {
	// Verify the conversion constant
	assert.InDelta(t, math.Pi/180.0, DegToRad, 0.000001)

	// 180° should equal π radians
	assert.InDelta(t, math.Pi, 180.0*DegToRad, 0.000001)

	// 90° should equal π/2 radians
	assert.InDelta(t, math.Pi/2, 90.0*DegToRad, 0.000001)
}

