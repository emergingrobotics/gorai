// Package imu provides the IMU formatter service that converts raw I2C data
// from IMU devices into standardized sensor.Imu protobuf messages.
package imu

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterService("formatter", "imu_formatter", New)
}

// State represents the operational state of the formatter.
type State int

const (
	StateStopped State = iota
	StateStarting
	StateRunning
	StateError
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateStarting:
		return "starting"
	case StateRunning:
		return "running"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Formatter is the IMU formatter service.
type Formatter struct {
	name   resource.Name
	config *Config
	logger *slog.Logger

	parser   Parser
	mpu6050  *MPU6050Parser // Type-asserted for variance calculations

	mu              sync.RWMutex
	state           State
	errorMsg        string
	lastMessageTime time.Time
	actualRateHz    float64

	// Rate estimation
	rateUpdateTime  time.Time
	rateSampleCount int

	// Sequence number
	seq atomic.Uint64

	// Metrics
	messagesReceived   atomic.Uint64
	messagesPublished  atomic.Uint64
	parseErrors        atomic.Uint64
	validationErrors   atomic.Uint64

	// Data callback (set by caller or NATS integration)
	onIMUData         func(data *IMUMessage)
	onTemperatureData func(temp float64, timestamp time.Time)

	stopCh chan struct{}
	doneCh chan struct{}
}

// IMUMessage holds the formatted IMU data ready for publishing.
type IMUMessage struct {
	Timestamp time.Time
	FrameID   string
	Seq       uint64

	// Acceleration in m/s²
	AccelX float64
	AccelY float64
	AccelZ float64

	// Angular velocity in rad/s
	GyroX float64
	GyroY float64
	GyroZ float64

	// Covariances (diagonal values, 3x3 matrices)
	AccelVariance float64
	GyroVariance  float64
}

// New creates a new IMU formatter service.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	// Convert registry.Config to resource.Config
	resConf := resource.NewConfig(conf)

	cfg, err := NewConfigFromResource(resConf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Get name from config
	name := "imu_formatter"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	// Create parser
	parser, err := NewParser(cfg.DeviceType, cfg.AccelScale, cfg.GyroScale)
	if err != nil {
		return nil, fmt.Errorf("failed to create parser: %w", err)
	}

	f := &Formatter{
		name:   resource.NewServiceName("gorai", "formatter", name),
		config: cfg,
		logger: slog.Default().With("service", "imu_formatter", "name", name),
		parser: parser,
		state:  StateStarting,
		stopCh: make(chan struct{}),
		doneCh: make(chan struct{}),
	}

	// Type assert for MPU6050 parser to get variance calculations
	if mpu, ok := parser.(*MPU6050Parser); ok {
		f.mpu6050 = mpu
	}

	// Start heartbeat loop
	go f.heartbeatLoop()

	f.mu.Lock()
	f.state = StateRunning
	f.mu.Unlock()

	f.logger.Info("IMU formatter started",
		"i2c_bridge", cfg.I2CBridgeComponent,
		"device", cfg.DeviceName,
		"type", cfg.DeviceType,
		"accel_scale", cfg.AccelScale,
		"gyro_scale", cfg.GyroScale,
	)

	return f, nil
}

// SetIMUDataCallback sets the callback for formatted IMU data.
func (f *Formatter) SetIMUDataCallback(callback func(data *IMUMessage)) {
	f.onIMUData = callback
}

// SetTemperatureCallback sets the callback for temperature data.
func (f *Formatter) SetTemperatureCallback(callback func(temp float64, timestamp time.Time)) {
	f.onTemperatureData = callback
}

// ProcessRawData processes raw I2C data and produces formatted IMU output.
// This is called by the I2C bridge integration or tests.
func (f *Formatter) ProcessRawData(data []byte, address uint8, deviceName string, timestamp time.Time) error {
	f.messagesReceived.Add(1)

	// Validate address
	if address != f.config.I2CAddress {
		f.validationErrors.Add(1)
		return fmt.Errorf("address mismatch: expected 0x%02X, got 0x%02X", f.config.I2CAddress, address)
	}

	// Validate device name
	if deviceName != f.config.DeviceName {
		f.validationErrors.Add(1)
		return fmt.Errorf("device mismatch: expected %q, got %q", f.config.DeviceName, deviceName)
	}

	// Parse the data
	imuData, temp, err := f.parser.Parse(data)
	if err != nil {
		f.parseErrors.Add(1)
		return fmt.Errorf("parse error: %w", err)
	}

	// Calculate variances
	var accelVariance, gyroVariance float64
	if f.mpu6050 != nil {
		accelVariance = f.mpu6050.AccelVariance()
		gyroVariance = f.mpu6050.GyroVariance()
	} else {
		// Default variances if parser doesn't provide them
		accelVariance = 0.01
		gyroVariance = 0.001
	}

	// Build IMU message
	msg := &IMUMessage{
		Timestamp:     timestamp,
		FrameID:       f.config.FrameID,
		Seq:           f.seq.Add(1),
		AccelX:        imuData.AccelX,
		AccelY:        imuData.AccelY,
		AccelZ:        imuData.AccelZ,
		GyroX:         imuData.GyroX,
		GyroY:         imuData.GyroY,
		GyroZ:         imuData.GyroZ,
		AccelVariance: accelVariance,
		GyroVariance:  gyroVariance,
	}

	// Call IMU data callback
	if f.onIMUData != nil {
		f.onIMUData(msg)
	}

	f.messagesPublished.Add(1)

	// Update timing
	f.mu.Lock()
	f.lastMessageTime = time.Now()
	f.updateRateEstimate()
	f.mu.Unlock()

	// Optionally call temperature callback
	if f.config.PublishTemperature && f.onTemperatureData != nil {
		f.onTemperatureData(temp, timestamp)
	}

	return nil
}

// updateRateEstimate updates the actual message rate estimation.
// Must be called with mutex held.
func (f *Formatter) updateRateEstimate() {
	now := time.Now()

	if f.rateUpdateTime.IsZero() {
		f.rateUpdateTime = now
		f.rateSampleCount = 0
		return
	}

	f.rateSampleCount++
	elapsed := now.Sub(f.rateUpdateTime)

	// Update every second
	if elapsed >= time.Second {
		f.actualRateHz = float64(f.rateSampleCount) / elapsed.Seconds()
		f.rateUpdateTime = now
		f.rateSampleCount = 0
	}
}

// heartbeatLoop publishes state periodically.
func (f *Formatter) heartbeatLoop() {
	defer close(f.doneCh)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-f.stopCh:
			return
		case <-ticker.C:
			// Heartbeat - could publish state/status here
			// For now, just check for stale data
			f.mu.RLock()
			if f.state == StateRunning && !f.lastMessageTime.IsZero() {
				if time.Since(f.lastMessageTime) > 5*time.Second {
					f.logger.Warn("no IMU data received for 5 seconds")
				}
			}
			f.mu.RUnlock()
		}
	}
}

// Name returns the resource name.
func (f *Formatter) Name() resource.Name {
	return f.name
}

// Reconfigure updates the service configuration.
func (f *Formatter) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	// For now, require restart for config changes
	return fmt.Errorf("reconfiguration not supported, restart required")
}

// DoCommand handles arbitrary commands.
func (f *Formatter) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)

	switch cmdName {
	case "get_state":
		f.mu.RLock()
		state := f.state.String()
		rateHz := f.actualRateHz
		lastMsg := f.lastMessageTime
		f.mu.RUnlock()

		result := map[string]any{
			"state":          state,
			"actual_rate_hz": rateHz,
		}
		if !lastMsg.IsZero() {
			result["last_message_time"] = lastMsg.Format(time.RFC3339)
		}
		return result, nil

	case "get_stats":
		return map[string]any{
			"messages_received":  f.messagesReceived.Load(),
			"messages_published": f.messagesPublished.Load(),
			"parse_errors":       f.parseErrors.Load(),
			"validation_errors":  f.validationErrors.Load(),
		}, nil

	case "reset_stats":
		f.messagesReceived.Store(0)
		f.messagesPublished.Store(0)
		f.parseErrors.Store(0)
		f.validationErrors.Store(0)
		return map[string]any{"success": true}, nil

	case "get_config":
		return map[string]any{
			"i2c_bridge_component": f.config.I2CBridgeComponent,
			"device_name":          f.config.DeviceName,
			"device_type":          f.config.DeviceType,
			"i2c_address":          fmt.Sprintf("0x%02X", f.config.I2CAddress),
			"accel_scale":          f.config.AccelScale,
			"gyro_scale":           f.config.GyroScale,
			"frame_id":             f.config.FrameID,
			"publish_temperature":  f.config.PublishTemperature,
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close stops the service and releases resources.
func (f *Formatter) Close(ctx context.Context) error {
	f.mu.Lock()
	if f.state == StateStopped {
		f.mu.Unlock()
		return nil
	}
	f.state = StateStopped
	f.mu.Unlock()

	close(f.stopCh)
	<-f.doneCh

	f.logger.Info("IMU formatter stopped")
	return nil
}

// GetState returns the current state.
func (f *Formatter) GetState() State {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.state
}

// GetStats returns the current statistics.
func (f *Formatter) GetStats() (received, published, parseErrors, validationErrors uint64) {
	return f.messagesReceived.Load(),
		f.messagesPublished.Load(),
		f.parseErrors.Load(),
		f.validationErrors.Load()
}

// GetConfig returns the service configuration.
func (f *Formatter) GetConfig() *Config {
	return f.config
}

// GetActualRateHz returns the actual message processing rate.
func (f *Formatter) GetActualRateHz() float64 {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.actualRateHz
}

