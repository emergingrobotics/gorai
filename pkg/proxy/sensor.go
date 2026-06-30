package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nats-io/nats.go"

	"github.com/emergingrobotics/gorai/pkg/mesh"
)

// RemoteSensor is a proxy that implements the Sensor interface via NATS.
type RemoteSensor struct {
	meshClient *mesh.Client
	natsConn   *nats.Conn
	descriptor mesh.ServiceDescriptor
	logger     *slog.Logger

	dataSubject string
	sensorType  string // e.g., "imu", "gps", "temperature"

	mu           sync.RWMutex
	lastReadings map[string]any
	lastReadAt   time.Time

	dataSub *nats.Subscription
	cancel  context.CancelFunc
	done    chan struct{}
}

// RemoteSensorOption configures a RemoteSensor.
type RemoteSensorOption func(*RemoteSensor)

// WithRemoteSensorLogger sets the logger.
func WithRemoteSensorLogger(logger *slog.Logger) RemoteSensorOption {
	return func(s *RemoteSensor) {
		s.logger = logger
	}
}

// WithSensorType sets the sensor type for subject matching.
func WithSensorType(sensorType string) RemoteSensorOption {
	return func(s *RemoteSensor) {
		s.sensorType = sensorType
	}
}

// NewRemoteSensor creates a new remote sensor proxy.
func NewRemoteSensor(mc *mesh.Client, nc *nats.Conn, desc mesh.ServiceDescriptor, opts ...RemoteSensorOption) (*RemoteSensor, error) {
	s := &RemoteSensor{
		meshClient:   mc,
		natsConn:     nc,
		descriptor:   desc,
		logger:       slog.Default(),
		lastReadings: make(map[string]any),
		done:         make(chan struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	// Detect sensor type from descriptor
	if s.sensorType == "" {
		s.sensorType = detectSensorType(desc)
	}

	// Find data subject
	var err error
	s.dataSubject, err = findSensorDataSubject(desc, s.sensorType)
	if err != nil {
		// Use wildcard subscription
		s.dataSubject = fmt.Sprintf("gsp.%s.rx.sensor.>", desc.Name)
	}

	// Subscribe to sensor data
	s.dataSub, err = nc.Subscribe(s.dataSubject, s.handleData)
	if err != nil {
		return nil, fmt.Errorf("failed to subscribe to sensor data: %w", err)
	}

	s.logger.Debug("created remote sensor",
		"name", desc.Name,
		"type", s.sensorType,
		"data_subject", s.dataSubject,
	)

	return s, nil
}

// detectSensorType detects the sensor type from the descriptor.
func detectSensorType(desc mesh.ServiceDescriptor) string {
	// Check subtype
	switch desc.Subtype {
	case "imu", "gps", "temperature", "pressure", "humidity", "distance", "encoder":
		return desc.Subtype
	}

	// Check metadata
	if caps, ok := desc.Metadata["capabilities"]; ok {
		if containsAny(caps, "IMU") {
			return "imu"
		}
		if containsAny(caps, "GPS") {
			return "gps"
		}
	}

	// Check publish subjects
	for _, pub := range desc.Publishes {
		if containsAny(pub, "imu") {
			return "imu"
		}
		if containsAny(pub, "gps") {
			return "gps"
		}
		if containsAny(pub, "temp") {
			return "temperature"
		}
	}

	return "generic"
}

// findSensorDataSubject finds the data subject for a sensor type.
func findSensorDataSubject(desc mesh.ServiceDescriptor, sensorType string) (string, error) {
	// Look for specific sensor type in publish subjects
	for _, pub := range desc.Publishes {
		if containsAny(pub, sensorType+"_data", sensorType) {
			return pub, nil
		}
	}

	// Use generic data subject
	return GetDataSubject(desc)
}

// handleData processes incoming sensor data.
func (s *RemoteSensor) handleData(msg *nats.Msg) {
	var envelope struct {
		DeviceID   string         `json:"device_id"`
		Type       string         `json:"type"`
		Timestamp  time.Time      `json:"ts"`
		ReceivedAt time.Time      `json:"received_at"`
		Data       map[string]any `json:"data"`
	}

	if err := json.Unmarshal(msg.Data, &envelope); err != nil {
		// Try parsing as just data
		var data map[string]any
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			s.logger.Debug("failed to parse sensor data", "error", err)
			return
		}
		envelope.Data = data
	}

	s.mu.Lock()
	// Merge new readings
	for k, v := range envelope.Data {
		s.lastReadings[k] = v
	}
	s.lastReadings["_type"] = envelope.Type
	s.lastReadings["_timestamp"] = envelope.Timestamp
	s.lastReadAt = time.Now()
	s.mu.Unlock()
}

// Readings returns the latest sensor readings.
func (s *RemoteSensor) Readings(ctx context.Context) (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return a copy
	result := make(map[string]any)
	for k, v := range s.lastReadings {
		result[k] = v
	}

	return result, nil
}

// GetReading returns a specific reading by key.
func (s *RemoteSensor) GetReading(ctx context.Context, key string) (any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if v, ok := s.lastReadings[key]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("reading %q not found", key)
}

// LastReadAt returns when the last reading was received.
func (s *RemoteSensor) LastReadAt() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastReadAt
}

// Close releases resources.
func (s *RemoteSensor) Close(ctx context.Context) error {
	if s.dataSub != nil {
		s.dataSub.Unsubscribe()
	}
	if s.cancel != nil {
		s.cancel()
	}
	close(s.done)
	return nil
}

// Name returns the sensor name.
func (s *RemoteSensor) Name() string {
	return s.descriptor.Name
}

// SensorType returns the sensor type.
func (s *RemoteSensor) SensorType() string {
	return s.sensorType
}

// Descriptor returns the service descriptor.
func (s *RemoteSensor) Descriptor() mesh.ServiceDescriptor {
	return s.descriptor
}

// DoCommand sends a command to the sensor.
func (s *RemoteSensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdSubject, err := GetCommandSubject(s.descriptor)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return nil, err
	}

	msg, err := s.natsConn.RequestWithContext(ctx, cmdSubject, data)
	if err != nil {
		return nil, err
	}

	var resp map[string]any
	if err := json.Unmarshal(msg.Data, &resp); err != nil {
		return nil, err
	}

	return resp, nil
}

// IMUReading returns IMU-specific readings (convenience method).
func (s *RemoteSensor) IMUReading(ctx context.Context) (*IMUData, error) {
	readings, err := s.Readings(ctx)
	if err != nil {
		return nil, err
	}

	return &IMUData{
		AccelX: toFloat64(readings["ax"]),
		AccelY: toFloat64(readings["ay"]),
		AccelZ: toFloat64(readings["az"]),
		GyroX:  toFloat64(readings["gx"]),
		GyroY:  toFloat64(readings["gy"]),
		GyroZ:  toFloat64(readings["gz"]),
		Time:   toInt64(readings["time"]),
	}, nil
}

// GPSReading returns GPS-specific readings (convenience method).
func (s *RemoteSensor) GPSReading(ctx context.Context) (*GPSData, error) {
	readings, err := s.Readings(ctx)
	if err != nil {
		return nil, err
	}

	return &GPSData{
		Latitude:   toFloat64(readings["latitude"]),
		Longitude:  toFloat64(readings["longitude"]),
		Altitude:   toFloat64(readings["altitude"]),
		Speed:      toFloat64(readings["speed"]),
		Heading:    toFloat64(readings["heading"]),
		Satellites: int(toInt64(readings["satellites"])),
		FixQuality: toString(readings["fix_quality"]),
	}, nil
}

// IMUData holds IMU sensor data.
type IMUData struct {
	AccelX, AccelY, AccelZ float64
	GyroX, GyroY, GyroZ    float64
	MagX, MagY, MagZ       float64
	Time                   int64
}

// GPSData holds GPS sensor data.
type GPSData struct {
	Latitude   float64
	Longitude  float64
	Altitude   float64
	Speed      float64
	Heading    float64
	Satellites int
	FixQuality string
	HDOP       float64
}

// Helper functions for type conversion

func toFloat64(v any) float64 {
	switch x := v.(type) {
	case float64:
		return x
	case float32:
		return float64(x)
	case int:
		return float64(x)
	case int64:
		return float64(x)
	case json.Number:
		f, _ := x.Float64()
		return f
	}
	return 0
}

func toInt64(v any) int64 {
	switch x := v.(type) {
	case int64:
		return x
	case int:
		return int64(x)
	case float64:
		return int64(x)
	case json.Number:
		i, _ := x.Int64()
		return i
	}
	return 0
}

func toString(v any) string {
	if s, ok := v.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", v)
}
