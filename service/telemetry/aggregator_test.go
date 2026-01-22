package telemetry

import (
	"testing"
	"time"

	"github.com/gorai/gorai/pkg/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				RobotID:       "test-robot",
				PublishRateHz: 10,
				Sources: []*SourceConfig{
					{Name: "imu", Topic: "gorai.test.imu.data", Type: SourceTypeIMU, StaleThresholdMs: 1000},
				},
			},
			expectErr: false,
		},
		{
			name: "missing robot_id",
			config: &Config{
				RobotID:       "",
				PublishRateHz: 10,
			},
			expectErr: true,
			errMsg:    "robot_id is required",
		},
		{
			name: "invalid publish rate - zero",
			config: &Config{
				RobotID:       "test-robot",
				PublishRateHz: 0,
			},
			expectErr: true,
			errMsg:    "publish_rate_hz must be positive",
		},
		{
			name: "invalid publish rate - too high",
			config: &Config{
				RobotID:       "test-robot",
				PublishRateHz: 200,
			},
			expectErr: true,
			errMsg:    "exceeds maximum",
		},
		{
			name: "motor states without topic",
			config: &Config{
				RobotID:            "test-robot",
				PublishRateHz:      10,
				IncludeMotorStates: true,
				MotorStateTopic:    "",
			},
			expectErr: true,
			errMsg:    "motor_state_topic is required",
		},
		{
			name: "duplicate source names",
			config: &Config{
				RobotID:       "test-robot",
				PublishRateHz: 10,
				Sources: []*SourceConfig{
					{Name: "imu", Topic: "gorai.test.imu.data", Type: SourceTypeIMU, StaleThresholdMs: 1000},
					{Name: "imu", Topic: "gorai.test.imu2.data", Type: SourceTypeIMU, StaleThresholdMs: 1000},
				},
			},
			expectErr: true,
			errMsg:    "duplicate source name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSourceConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *SourceConfig
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid source",
			config: &SourceConfig{
				Name:             "imu",
				Topic:            "gorai.test.imu.data",
				Type:             SourceTypeIMU,
				StaleThresholdMs: 1000,
			},
			expectErr: false,
		},
		{
			name: "missing name",
			config: &SourceConfig{
				Name:             "",
				Topic:            "gorai.test.imu.data",
				Type:             SourceTypeIMU,
				StaleThresholdMs: 1000,
			},
			expectErr: true,
			errMsg:    "name is required",
		},
		{
			name: "missing topic",
			config: &SourceConfig{
				Name:             "imu",
				Topic:            "",
				Type:             SourceTypeIMU,
				StaleThresholdMs: 1000,
			},
			expectErr: true,
			errMsg:    "topic is required",
		},
		{
			name: "invalid type",
			config: &SourceConfig{
				Name:             "imu",
				Topic:            "gorai.test.imu.data",
				Type:             "invalid",
				StaleThresholdMs: 1000,
			},
			expectErr: true,
			errMsg:    "invalid source type",
		},
		{
			name: "invalid stale threshold",
			config: &SourceConfig{
				Name:             "imu",
				Topic:            "gorai.test.imu.data",
				Type:             SourceTypeIMU,
				StaleThresholdMs: 0,
			},
			expectErr: true,
			errMsg:    "stale_threshold_ms must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigParsing(t *testing.T) {
	resConf := resource.Config{
		Attributes: map[string]any{
			"robot_id":             "my-robot",
			"publish_rate_hz":      20.0,
			"include_system_stats": false,
			"include_motor_states": true,
			"motor_state_topic":    "gorai.my-robot.motors.state",
			"sources": []any{
				map[string]any{
					"name":               "imu",
					"topic":              "gorai.my-robot.imu.data",
					"type":               "imu",
					"required":           true,
					"stale_threshold_ms": 500.0,
				},
				map[string]any{
					"name":               "gps",
					"topic":              "gorai.my-robot.gps.data",
					"type":               "gps",
					"required":           false,
					"stale_threshold_ms": 2000.0,
				},
			},
		},
	}

	cfg, err := NewConfigFromResource(resConf)
	require.NoError(t, err)

	assert.Equal(t, "my-robot", cfg.RobotID)
	assert.Equal(t, 20.0, cfg.PublishRateHz)
	assert.False(t, cfg.IncludeSystemStats)
	assert.True(t, cfg.IncludeMotorStates)
	assert.Equal(t, "gorai.my-robot.motors.state", cfg.MotorStateTopic)

	require.Len(t, cfg.Sources, 2)

	assert.Equal(t, "imu", cfg.Sources[0].Name)
	assert.Equal(t, "gorai.my-robot.imu.data", cfg.Sources[0].Topic)
	assert.Equal(t, SourceTypeIMU, cfg.Sources[0].Type)
	assert.True(t, cfg.Sources[0].Required)
	assert.Equal(t, int64(500), cfg.Sources[0].StaleThresholdMs)

	assert.Equal(t, "gps", cfg.Sources[1].Name)
	assert.Equal(t, SourceTypeGPS, cfg.Sources[1].Type)
	assert.False(t, cfg.Sources[1].Required)
}

func TestConfigDefaults(t *testing.T) {
	resConf := resource.Config{
		Attributes: map[string]any{
			"robot_id": "test-robot",
		},
	}

	cfg, err := NewConfigFromResource(resConf)
	require.NoError(t, err)

	assert.Equal(t, "test-robot", cfg.RobotID)
	assert.Equal(t, 10.0, cfg.PublishRateHz)
	assert.True(t, cfg.IncludeSystemStats)
	assert.False(t, cfg.IncludeMotorStates)
	assert.Empty(t, cfg.Sources)
}

func TestPublishInterval(t *testing.T) {
	tests := []struct {
		name             string
		publishRateHz    float64
		expectedInterval time.Duration
	}{
		{
			name:             "10 Hz",
			publishRateHz:    10,
			expectedInterval: 100 * time.Millisecond,
		},
		{
			name:             "1 Hz",
			publishRateHz:    1,
			expectedInterval: time.Second,
		},
		{
			name:             "50 Hz",
			publishRateHz:    50,
			expectedInterval: 20 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &Config{PublishRateHz: tt.publishRateHz}
			assert.Equal(t, tt.expectedInterval, cfg.PublishInterval())
		})
	}
}

func TestSourceTypeIsValid(t *testing.T) {
	validTypes := []SourceType{
		SourceTypeIMU,
		SourceTypeGPS,
		SourceTypeBattery,
		SourceTypeTemperature,
		SourceTypeRange,
		SourceTypeGeneric,
	}

	for _, st := range validTypes {
		assert.True(t, st.IsValid(), "expected %s to be valid", st)
	}

	assert.False(t, SourceType("invalid").IsValid())
	assert.False(t, SourceType("").IsValid())
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateStopped, "stopped"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateDegraded, "degraded"},
		{StateError, "error"},
		{State(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestHealthString(t *testing.T) {
	tests := []struct {
		health   Health
		expected string
	}{
		{HealthOK, "ok"},
		{HealthDegraded, "degraded"},
		{HealthError, "error"},
		{HealthUnspecified, "unspecified"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.health.String())
		})
	}
}

func TestSourceStatsUpdateRate(t *testing.T) {
	stats := &SourceStats{}

	// Simulate 10 messages in quick succession
	for i := 0; i < 10; i++ {
		stats.MessagesReceived.Add(1)
		stats.UpdateRate()
		time.Sleep(10 * time.Millisecond)
	}

	rate := stats.GetRate()
	assert.Greater(t, rate, float64(5)) // Should be around 10
}

func TestExtractGenericValue(t *testing.T) {
	tests := []struct {
		name     string
		data     *SourceData
		expected float64
	}{
		{
			name:     "nil data",
			data:     nil,
			expected: 0,
		},
		{
			name:     "float64",
			data:     &SourceData{Data: float64(42.5)},
			expected: 42.5,
		},
		{
			name:     "float32",
			data:     &SourceData{Data: float32(42.5)},
			expected: 42.5,
		},
		{
			name:     "int",
			data:     &SourceData{Data: int(42)},
			expected: 42,
		},
		{
			name:     "int64",
			data:     &SourceData{Data: int64(42)},
			expected: 42,
		},
		{
			name:     "unsupported type",
			data:     &SourceData{Data: "string"},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractGenericValue(tt.data)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetSourceConfig(t *testing.T) {
	cfg := &Config{
		RobotID: "test",
		Sources: []*SourceConfig{
			{Name: "imu", Topic: "gorai.test.imu.data", Type: SourceTypeIMU},
			{Name: "gps", Topic: "gorai.test.gps.data", Type: SourceTypeGPS},
		},
	}

	// Found
	src := cfg.GetSourceConfig("imu")
	require.NotNil(t, src)
	assert.Equal(t, "imu", src.Name)

	// Not found
	src = cfg.GetSourceConfig("nonexistent")
	assert.Nil(t, src)
}

func TestStaleThreshold(t *testing.T) {
	src := &SourceConfig{
		StaleThresholdMs: 500,
	}

	threshold := src.StaleThreshold()
	assert.Equal(t, 500*time.Millisecond, threshold)

	// Calling again should return cached value
	threshold = src.StaleThreshold()
	assert.Equal(t, 500*time.Millisecond, threshold)
}

func TestHealthDetermination(t *testing.T) {
	// Create an aggregator with test configuration
	cfg := &Config{
		RobotID:       "test",
		PublishRateHz: 10,
		Sources: []*SourceConfig{
			{Name: "imu", Topic: "t1", Type: SourceTypeIMU, Required: true, StaleThresholdMs: 500, staleThreshold: 500 * time.Millisecond},
			{Name: "gps", Topic: "t2", Type: SourceTypeGPS, Required: false, StaleThresholdMs: 2000, staleThreshold: 2000 * time.Millisecond},
		},
	}

	a := &Aggregator{
		config:      cfg,
		latestData:  make(map[string]*SourceData),
		sourceStats: make(map[string]*SourceStats),
	}

	// Test HEALTH_OK - all sources fresh
	a.latestData["imu"] = &SourceData{Data: map[string]any{"accel_x": 0.1}, ReceivedAt: time.Now(), Type: SourceTypeIMU}
	a.latestData["gps"] = &SourceData{Data: map[string]any{"latitude": 37.0}, ReceivedAt: time.Now(), Type: SourceTypeGPS}
	assert.Equal(t, HealthOK, a.DetermineHealth())

	// Test HEALTH_DEGRADED - optional source stale
	a.latestData["gps"] = &SourceData{Data: map[string]any{"latitude": 37.0}, ReceivedAt: time.Now().Add(-3 * time.Second), Type: SourceTypeGPS}
	assert.Equal(t, HealthDegraded, a.DetermineHealth())

	// Test HEALTH_ERROR - required source missing
	delete(a.latestData, "imu")
	assert.Equal(t, HealthError, a.DetermineHealth())

	// Test HEALTH_ERROR - required source stale
	a.latestData["imu"] = &SourceData{Data: map[string]any{"accel_x": 0.1}, ReceivedAt: time.Now().Add(-2 * time.Second), Type: SourceTypeIMU}
	assert.Equal(t, HealthError, a.DetermineHealth())
}

func TestBuildTelemetryFrame(t *testing.T) {
	cfg := &Config{
		RobotID:            "test",
		PublishRateHz:      10,
		IncludeSystemStats: false,
		Sources: []*SourceConfig{
			{Name: "imu", Topic: "t1", Type: SourceTypeIMU, StaleThresholdMs: 1000},
		},
	}

	a := &Aggregator{
		config:      cfg,
		latestData:  make(map[string]*SourceData),
		sourceStats: make(map[string]*SourceStats),
	}
	a.sourceStats["imu"] = &SourceStats{}

	// Add some IMU data
	a.latestData["imu"] = &SourceData{
		Data: map[string]any{
			"accel_x": 0.1,
			"accel_y": 0.2,
			"accel_z": 9.8,
		},
		ReceivedAt: time.Now(),
		Type:       SourceTypeIMU,
	}

	frame := a.BuildTelemetryFrame()

	require.NotNil(t, frame)
	assert.NotZero(t, frame.Timestamp)
	assert.NotZero(t, frame.Seq)
	assert.Equal(t, "telemetry", frame.FrameID)

	require.NotNil(t, frame.IMU)
	assert.True(t, frame.IMU.Available)
	assert.False(t, frame.IMU.Stale)
	assert.Equal(t, 0.1, frame.IMU.AccelX)
	assert.Equal(t, 0.2, frame.IMU.AccelY)
	assert.Equal(t, 9.8, frame.IMU.AccelZ)

	require.Len(t, frame.Sources, 1)
	assert.Equal(t, "imu", frame.Sources[0].Name)
	assert.True(t, frame.Sources[0].Connected)
}

func TestAggregatorHandleSourceMessage(t *testing.T) {
	cfg := &Config{
		RobotID:       "test",
		PublishRateHz: 10,
		Sources: []*SourceConfig{
			{Name: "imu", Topic: "t1", Type: SourceTypeIMU, StaleThresholdMs: 1000},
		},
	}

	a := &Aggregator{
		config:      cfg,
		latestData:  make(map[string]*SourceData),
		sourceStats: make(map[string]*SourceStats),
	}
	a.sourceStats["imu"] = &SourceStats{}

	// Handle a message
	imuData := map[string]any{"accel_x": 1.0}
	a.HandleSourceMessage("imu", SourceTypeIMU, imuData)

	// Check data was stored
	a.dataMu.RLock()
	data := a.latestData["imu"]
	a.dataMu.RUnlock()

	require.NotNil(t, data)
	assert.Equal(t, SourceTypeIMU, data.Type)
	assert.Equal(t, imuData, data.Data)

	// Check stats were updated
	a.statsMu.RLock()
	stats := a.sourceStats["imu"]
	a.statsMu.RUnlock()

	assert.Equal(t, uint64(1), stats.MessagesReceived.Load())
}
