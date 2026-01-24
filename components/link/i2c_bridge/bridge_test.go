package i2c_bridge

import (
	"testing"

	"github.com/gorai/gorai/pkg/resource"
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
				Device: "/dev/i2c-1",
				Devices: []DeviceConfig{
					{
						Name:         "mpu6050",
						Address:      0x68,
						ReadRegister: 0x3B,
						ReadLength:   14,
						PollRateHz:   100,
						Enabled:      true,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "missing device path",
			config: &Config{
				Device: "",
				Devices: []DeviceConfig{
					{
						Name:       "test",
						Address:    0x68,
						ReadLength: 14,
					},
				},
			},
			wantErr: true,
			errMsg:  "device path is required",
		},
		{
			name: "no devices",
			config: &Config{
				Device:  "/dev/i2c-1",
				Devices: []DeviceConfig{},
			},
			wantErr: true,
			errMsg:  "at least one device is required",
		},
		{
			name: "address out of range",
			config: &Config{
				Device: "/dev/i2c-1",
				Devices: []DeviceConfig{
					{
						Name:       "test",
						Address:    128, // > 127
						ReadLength: 14,
					},
				},
			},
			wantErr: true,
			errMsg:  "out of range",
		},
		{
			name: "duplicate device names",
			config: &Config{
				Device: "/dev/i2c-1",
				Devices: []DeviceConfig{
					{
						Name:       "sensor",
						Address:    0x68,
						ReadLength: 14,
						PollRateHz: 100,
					},
					{
						Name:       "sensor",
						Address:    0x69,
						ReadLength: 14,
						PollRateHz: 100,
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate device name",
		},
		{
			name: "duplicate addresses",
			config: &Config{
				Device: "/dev/i2c-1",
				Devices: []DeviceConfig{
					{
						Name:       "sensor1",
						Address:    0x68,
						ReadLength: 14,
						PollRateHz: 100,
					},
					{
						Name:       "sensor2",
						Address:    0x68,
						ReadLength: 14,
						PollRateHz: 100,
					},
				},
			},
			wantErr: true,
			errMsg:  "already used",
		},
		{
			name: "invalid read length",
			config: &Config{
				Device: "/dev/i2c-1",
				Devices: []DeviceConfig{
					{
						Name:       "test",
						Address:    0x68,
						ReadLength: 0,
					},
				},
			},
			wantErr: true,
			errMsg:  "read_length must be positive",
		},
		{
			name: "read length too large",
			config: &Config{
				Device: "/dev/i2c-1",
				Devices: []DeviceConfig{
					{
						Name:       "test",
						Address:    0x68,
						ReadLength: 300,
						PollRateHz: 100,
					},
				},
			},
			wantErr: true,
			errMsg:  "exceeds maximum",
		},
		{
			name: "invalid poll rate",
			config: &Config{
				Device: "/dev/i2c-1",
				Devices: []DeviceConfig{
					{
						Name:       "test",
						Address:    0x68,
						ReadLength: 14,
						PollRateHz: 0,
					},
				},
			},
			wantErr: true,
			errMsg:  "poll_rate_hz must be positive",
		},
		{
			name: "poll rate too high",
			config: &Config{
				Device: "/dev/i2c-1",
				Devices: []DeviceConfig{
					{
						Name:       "test",
						Address:    0x68,
						ReadLength: 14,
						PollRateHz: 2000,
					},
				},
			},
			wantErr: true,
			errMsg:  "exceeds maximum",
		},
		{
			name: "raw read (no register)",
			config: &Config{
				Device: "/dev/i2c-1",
				Devices: []DeviceConfig{
					{
						Name:         "test",
						Address:      0x68,
						ReadRegister: -1,
						ReadLength:   14,
						PollRateHz:   100,
					},
				},
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
		"device": "/dev/i2c-1",
		"devices": []any{
			map[string]any{
				"name":          "mpu6050",
				"address":       float64(104), // JSON numbers are float64
				"read_register": float64(59),
				"read_length":   float64(14),
				"poll_rate_hz":  float64(100),
				"enabled":       true,
			},
			map[string]any{
				"name":        "sensor2",
				"address":     float64(105),
				"read_length": float64(6),
			},
		},
	}

	conf := resource.NewConfig(attrs)
	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	assert.Equal(t, "/dev/i2c-1", cfg.Device)
	require.Len(t, cfg.Devices, 2)

	// Check first device
	assert.Equal(t, "mpu6050", cfg.Devices[0].Name)
	assert.Equal(t, uint8(104), cfg.Devices[0].Address)
	assert.Equal(t, 59, cfg.Devices[0].ReadRegister)
	assert.Equal(t, 14, cfg.Devices[0].ReadLength)
	assert.Equal(t, 100.0, cfg.Devices[0].PollRateHz)
	assert.True(t, cfg.Devices[0].Enabled)

	// Check second device (with defaults)
	assert.Equal(t, "sensor2", cfg.Devices[1].Name)
	assert.Equal(t, uint8(105), cfg.Devices[1].Address)
	assert.Equal(t, -1, cfg.Devices[1].ReadRegister) // default
	assert.Equal(t, 6, cfg.Devices[1].ReadLength)
	assert.Equal(t, 100.0, cfg.Devices[1].PollRateHz) // default
	assert.True(t, cfg.Devices[1].Enabled)            // default
}

func TestConfigDefaults(t *testing.T) {
	attrs := map[string]any{
		"device": "/dev/i2c-1",
		"devices": []any{
			map[string]any{
				"name":        "test",
				"address":     float64(104),
				"read_length": float64(14),
			},
		},
	}

	conf := resource.NewConfig(attrs)
	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	// Check defaults
	assert.Equal(t, -1, cfg.Devices[0].ReadRegister)
	assert.Equal(t, 100.0, cfg.Devices[0].PollRateHz)
	assert.True(t, cfg.Devices[0].Enabled)
}

func TestParseBusID(t *testing.T) {
	tests := []struct {
		path    string
		wantID  int
		wantErr bool
	}{
		{"/dev/i2c-0", 0, false},
		{"/dev/i2c-1", 1, false},
		{"/dev/i2c-10", 10, false},
		{"/dev/i2c-", 0, true},
		{"/dev/spi0", 0, true},
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			id, err := ParseBusID(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpening, "opening"},
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

func TestAddressValidation(t *testing.T) {
	// Valid 7-bit addresses: 0x00 - 0x7F (0-127)
	// Reserved: 0x00-0x07 and 0x78-0x7F (but we allow them)

	validAddresses := []uint8{0x08, 0x20, 0x50, 0x68, 0x77}
	for _, addr := range validAddresses {
		config := &Config{
			Device: "/dev/i2c-1",
			Devices: []DeviceConfig{
				{
					Name:       "test",
					Address:    addr,
					ReadLength: 14,
					PollRateHz: 100,
				},
			},
		}
		assert.NoError(t, config.Validate(), "address 0x%02X should be valid", addr)
	}

	// Invalid: > 127
	config := &Config{
		Device: "/dev/i2c-1",
		Devices: []DeviceConfig{
			{
				Name:       "test",
				Address:    128,
				ReadLength: 14,
				PollRateHz: 100,
			},
		},
	}
	assert.Error(t, config.Validate())
}

func TestMultipleDevices(t *testing.T) {
	config := &Config{
		Device: "/dev/i2c-1",
		Devices: []DeviceConfig{
			{
				Name:         "imu",
				Address:      0x68,
				ReadRegister: 0x3B,
				ReadLength:   14,
				PollRateHz:   100,
			},
			{
				Name:         "magnetometer",
				Address:      0x1E,
				ReadRegister: 0x03,
				ReadLength:   6,
				PollRateHz:   50,
			},
			{
				Name:         "barometer",
				Address:      0x77,
				ReadRegister: 0x00,
				ReadLength:   6,
				PollRateHz:   25,
			},
		},
	}

	err := config.Validate()
	assert.NoError(t, err)
}

func TestDeviceConfigDisabled(t *testing.T) {
	attrs := map[string]any{
		"device": "/dev/i2c-1",
		"devices": []any{
			map[string]any{
				"name":        "test",
				"address":     float64(104),
				"read_length": float64(14),
				"enabled":     false,
			},
		},
	}

	conf := resource.NewConfig(attrs)
	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	assert.False(t, cfg.Devices[0].Enabled)
}

