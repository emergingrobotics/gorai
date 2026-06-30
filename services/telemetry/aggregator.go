// Package telemetry provides a service that aggregates sensor data from multiple
// sources on the NATS message bus and publishes consolidated telemetry frames.
package telemetry

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
	registry.RegisterService("telemetry", "telemetry_aggregator", New)
}

// State represents the operational state of the service.
type State int32

const (
	StateStopped State = iota
	StateStarting
	StateRunning
	StateDegraded
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
	case StateDegraded:
		return "degraded"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Health represents the health status of the service.
type Health int

const (
	HealthUnspecified Health = iota
	HealthOK
	HealthDegraded
	HealthError
)

// String returns the string representation of health.
func (h Health) String() string {
	switch h {
	case HealthOK:
		return "ok"
	case HealthDegraded:
		return "degraded"
	case HealthError:
		return "error"
	default:
		return "unspecified"
	}
}

// TelemetryFrame represents a consolidated telemetry message.
type TelemetryFrame struct {
	Timestamp time.Time
	Seq       uint64
	FrameID   string

	// Sensor data
	IMU     *IMUData
	GPS     *GPSData
	Battery *BatteryData

	// Motor states
	Motors []*MotorData

	// Generic readings
	Readings map[string]float64

	// System stats
	System *SystemStatsData

	// Source statuses
	Sources []*SourceStatusData
}

// IMUData represents simplified IMU data for telemetry.
type IMUData struct {
	Available bool
	Stale     bool
	AgeMs     int64

	AccelX, AccelY, AccelZ          float64
	GyroX, GyroY, GyroZ             float64
	OrientationX, OrientationY      float64
	OrientationZ, OrientationW      float64
}

// GPSData represents simplified GPS data for telemetry.
type GPSData struct {
	Available  bool
	Stale      bool
	AgeMs      int64
	Latitude   float64
	Longitude  float64
	Altitude   float64
	FixStatus  int32
	Satellites uint32
}

// BatteryData represents simplified battery data for telemetry.
type BatteryData struct {
	Available    bool
	Stale        bool
	AgeMs        int64
	Voltage      float64
	Current      float64
	Percentage   float64
	TemperatureC float64
	Status       uint32
}

// MotorData represents the state of a motor.
type MotorData struct {
	Name    string
	Enabled bool
	Angle   float64
	Speed   float64
	PulseUs float64
}

// SystemStatsData contains system health metrics.
type SystemStatsData struct {
	Available      bool
	CPUPercent     float64
	MemoryPercent  float64
	DiskPercent    float64
	TemperatureC   float64
	UptimeSeconds  int64
	LoadAverage1m  float64
}

// SourceStatusData reports the health of a data source.
type SourceStatusData struct {
	Name             string
	Type             string
	Connected        bool
	Stale            bool
	AgeMs            int64
	MessagesReceived uint64
	MessagesDropped  uint64
	ActualRateHz     float64
}

// Aggregator is the telemetry aggregator service.
type Aggregator struct {
	name   resource.Name
	config *Config
	logger *slog.Logger

	// Data storage
	latestData map[string]*SourceData
	dataMu     sync.RWMutex

	// Source statistics
	sourceStats map[string]*SourceStats
	statsMu     sync.RWMutex

	// System stats collector
	sysStats SystemStatsCollector

	// State
	state   State
	stateMu sync.RWMutex

	// Sequence number
	seq atomic.Uint64

	// Statistics
	framesPublished atomic.Uint64
	publishErrors   atomic.Uint64

	// Rate tracking
	publishTimes []time.Time
	publishMu    sync.Mutex
	publishRate  float64

	// Frame callback (for NATS integration)
	onFrame func(frame *TelemetryFrame)

	// Control
	stopCh chan struct{}
	doneCh chan struct{}
}

// New creates a new Telemetry Aggregator service.
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
	name := "telemetry"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	a := &Aggregator{
		name:         resource.NewServiceName("gorai", "telemetry", name),
		config:       cfg,
		logger:       slog.Default().With("service", "telemetry", "name", name),
		latestData:   make(map[string]*SourceData),
		sourceStats:  make(map[string]*SourceStats),
		publishTimes: make([]time.Time, 0, 100),
		state:        StateStopped,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}

	// Initialize system stats collector
	if cfg.IncludeSystemStats {
		a.sysStats = NewSystemStatsCollector()
	}

	// Initialize source stats
	for _, src := range cfg.Sources {
		a.sourceStats[src.Name] = &SourceStats{}
	}

	// Start the service
	if err := a.start(ctx); err != nil {
		return nil, err
	}

	return a, nil
}

// start initializes the service and begins the publish loop.
func (a *Aggregator) start(ctx context.Context) error {
	a.setState(StateStarting)
	a.setState(StateRunning)

	// Start publish loop
	go a.publishLoop(ctx)

	// Start heartbeat loop
	go a.heartbeatLoop(ctx)

	a.logger.Info("Telemetry aggregator started",
		"robot_id", a.config.RobotID,
		"publish_rate_hz", a.config.PublishRateHz,
		"source_count", len(a.config.Sources),
	)

	return nil
}

// SetFrameCallback sets the callback for telemetry frame publishing.
func (a *Aggregator) SetFrameCallback(fn func(frame *TelemetryFrame)) {
	a.onFrame = fn
}

// HandleSourceMessage processes an incoming message from any source.
func (a *Aggregator) HandleSourceMessage(sourceName string, sourceType SourceType, data any) {
	now := time.Now()

	// Store latest data
	a.dataMu.Lock()
	a.latestData[sourceName] = &SourceData{
		Data:       data,
		ReceivedAt: now,
		Type:       sourceType,
	}
	a.dataMu.Unlock()

	// Update statistics
	a.statsMu.Lock()
	stats := a.sourceStats[sourceName]
	if stats != nil {
		stats.MessagesReceived.Add(1)
		stats.UpdateRate()
	}
	a.statsMu.Unlock()
}

// publishLoop runs the main publish loop.
func (a *Aggregator) publishLoop(ctx context.Context) {
	ticker := time.NewTicker(a.config.PublishInterval())
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			frame := a.BuildTelemetryFrame()

			if a.onFrame != nil {
				a.onFrame(frame)
			}

			a.framesPublished.Add(1)
			a.updatePublishRate()

			// Update state based on health
			a.updateState()
		}
	}
}

// BuildTelemetryFrame constructs a telemetry frame from current data.
func (a *Aggregator) BuildTelemetryFrame() *TelemetryFrame {
	now := time.Now()

	frame := &TelemetryFrame{
		Timestamp: now,
		Seq:       a.seq.Add(1),
		FrameID:   "telemetry",
		Readings:  make(map[string]float64),
		Sources:   make([]*SourceStatusData, 0, len(a.config.Sources)),
	}

	a.dataMu.RLock()
	a.statsMu.RLock()
	defer a.dataMu.RUnlock()
	defer a.statsMu.RUnlock()

	// Process each configured source
	for _, src := range a.config.Sources {
		data := a.latestData[src.Name]
		stats := a.sourceStats[src.Name]

		var ageMs int64
		var stale, available bool

		if data != nil && data.Data != nil {
			ageMs = now.Sub(data.ReceivedAt).Milliseconds()
			stale = ageMs > src.StaleThresholdMs
			available = true
		} else {
			ageMs = -1
			stale = true
			available = false
		}

		// Build typed data
		switch src.Type {
		case SourceTypeIMU:
			frame.IMU = a.buildIMUData(data, ageMs, stale, available)
		case SourceTypeGPS:
			frame.GPS = a.buildGPSData(data, ageMs, stale, available)
		case SourceTypeBattery:
			frame.Battery = a.buildBatteryData(data, ageMs, stale, available)
		case SourceTypeGeneric:
			if available {
				frame.Readings[src.Name] = ExtractGenericValue(data)
			}
		}

		// Build source status
		frame.Sources = append(frame.Sources, a.buildSourceStatus(
			src.Name, src.Type, data, stats, src,
		))
	}

	// Add system stats if configured
	if a.config.IncludeSystemStats && a.sysStats != nil {
		sysStats := a.sysStats.Collect()
		if sysStats != nil {
			frame.System = &SystemStatsData{
				Available:      sysStats.Available,
				CPUPercent:     sysStats.CpuPercent,
				MemoryPercent:  sysStats.MemoryPercent,
				DiskPercent:    sysStats.DiskPercent,
				TemperatureC:   sysStats.TemperatureC,
				UptimeSeconds:  sysStats.UptimeSeconds,
				LoadAverage1m:  sysStats.LoadAverage_1M,
			}
		}
	}

	return frame
}

// buildIMUData extracts IMU data into telemetry format.
func (a *Aggregator) buildIMUData(data *SourceData, ageMs int64, stale, available bool) *IMUData {
	imuData := &IMUData{
		Available: available,
		Stale:     stale,
		AgeMs:     ageMs,
	}

	if !available || data == nil || data.Data == nil {
		return imuData
	}

	// Data would be populated from actual IMU messages
	// For now, handle generic map data
	if m, ok := data.Data.(map[string]any); ok {
		if v, ok := m["accel_x"].(float64); ok {
			imuData.AccelX = v
		}
		if v, ok := m["accel_y"].(float64); ok {
			imuData.AccelY = v
		}
		if v, ok := m["accel_z"].(float64); ok {
			imuData.AccelZ = v
		}
		if v, ok := m["gyro_x"].(float64); ok {
			imuData.GyroX = v
		}
		if v, ok := m["gyro_y"].(float64); ok {
			imuData.GyroY = v
		}
		if v, ok := m["gyro_z"].(float64); ok {
			imuData.GyroZ = v
		}
	}

	return imuData
}

// buildGPSData extracts GPS data into telemetry format.
func (a *Aggregator) buildGPSData(data *SourceData, ageMs int64, stale, available bool) *GPSData {
	gpsData := &GPSData{
		Available: available,
		Stale:     stale,
		AgeMs:     ageMs,
	}

	if !available || data == nil || data.Data == nil {
		return gpsData
	}

	// Handle generic map data
	if m, ok := data.Data.(map[string]any); ok {
		if v, ok := m["latitude"].(float64); ok {
			gpsData.Latitude = v
		}
		if v, ok := m["longitude"].(float64); ok {
			gpsData.Longitude = v
		}
		if v, ok := m["altitude"].(float64); ok {
			gpsData.Altitude = v
		}
		if v, ok := m["fix_status"].(int32); ok {
			gpsData.FixStatus = v
		}
	}

	return gpsData
}

// buildBatteryData extracts battery data into telemetry format.
func (a *Aggregator) buildBatteryData(data *SourceData, ageMs int64, stale, available bool) *BatteryData {
	batteryData := &BatteryData{
		Available: available,
		Stale:     stale,
		AgeMs:     ageMs,
	}

	if !available || data == nil || data.Data == nil {
		return batteryData
	}

	// Handle generic map data
	if m, ok := data.Data.(map[string]any); ok {
		if v, ok := m["voltage"].(float64); ok {
			batteryData.Voltage = v
		}
		if v, ok := m["current"].(float64); ok {
			batteryData.Current = v
		}
		if v, ok := m["percentage"].(float64); ok {
			batteryData.Percentage = v
		}
	}

	return batteryData
}

// buildSourceStatus builds a source status.
func (a *Aggregator) buildSourceStatus(name string, sourceType SourceType, data *SourceData, stats *SourceStats, cfg *SourceConfig) *SourceStatusData {
	now := time.Now()

	status := &SourceStatusData{
		Name:             name,
		Type:             string(sourceType),
		MessagesReceived: stats.MessagesReceived.Load(),
		MessagesDropped:  stats.MessagesDropped.Load(),
		ActualRateHz:     stats.GetRate(),
	}

	if data != nil && data.Data != nil {
		status.Connected = true
		status.AgeMs = now.Sub(data.ReceivedAt).Milliseconds()
		status.Stale = status.AgeMs > cfg.StaleThresholdMs
	} else {
		status.Connected = false
		status.Stale = true
		status.AgeMs = -1
	}

	return status
}

// updatePublishRate updates the publish rate calculation.
func (a *Aggregator) updatePublishRate() {
	a.publishMu.Lock()
	defer a.publishMu.Unlock()

	now := time.Now()

	// Add timestamp
	a.publishTimes = append(a.publishTimes, now)

	// Remove old timestamps
	cutoff := now.Add(-time.Second)
	for len(a.publishTimes) > 0 && a.publishTimes[0].Before(cutoff) {
		a.publishTimes = a.publishTimes[1:]
	}

	a.publishRate = float64(len(a.publishTimes))
}

// getPublishRate returns the current publish rate.
func (a *Aggregator) getPublishRate() float64 {
	a.publishMu.Lock()
	defer a.publishMu.Unlock()
	return a.publishRate
}

// updateState determines and sets the current state based on source health.
func (a *Aggregator) updateState() {
	health := a.DetermineHealth()

	switch health {
	case HealthOK:
		a.setState(StateRunning)
	case HealthDegraded:
		a.setState(StateDegraded)
	case HealthError:
		a.setState(StateError)
	}
}

// DetermineHealth calculates the current health status.
func (a *Aggregator) DetermineHealth() Health {
	a.dataMu.RLock()
	defer a.dataMu.RUnlock()

	now := time.Now()
	requiredMissing := 0
	anyStale := false

	for _, src := range a.config.Sources {
		data := a.latestData[src.Name]

		if data == nil || data.Data == nil {
			if src.Required {
				requiredMissing++
			}
			continue
		}

		age := now.Sub(data.ReceivedAt)
		if age > src.StaleThreshold() {
			anyStale = true
			if src.Required {
				requiredMissing++
			}
		}
	}

	if requiredMissing > 0 {
		return HealthError
	}
	if anyStale {
		return HealthDegraded
	}
	return HealthOK
}

// setState updates the service state.
func (a *Aggregator) setState(newState State) {
	a.stateMu.Lock()
	defer a.stateMu.Unlock()

	if a.state != newState {
		a.logger.Info("state changed", "from", a.state.String(), "to", newState.String())
		a.state = newState
	}
}

// getState returns the current state.
func (a *Aggregator) getState() State {
	a.stateMu.RLock()
	defer a.stateMu.RUnlock()
	return a.state
}

// heartbeatLoop publishes state and status messages.
func (a *Aggregator) heartbeatLoop(ctx context.Context) {
	defer close(a.doneCh)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			// Heartbeat - state updates happen in publish loop
		}
	}
}

// Name returns the resource name.
func (a *Aggregator) Name() resource.Name {
	return a.name
}

// Reconfigure updates the service configuration.
func (a *Aggregator) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return fmt.Errorf("reconfiguration not supported, restart required")
}

// DoCommand handles arbitrary commands.
func (a *Aggregator) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)

	switch cmdName {
	case "get_state":
		connectedSources := uint32(0)
		a.dataMu.RLock()
		for _, data := range a.latestData {
			if data != nil && data.Data != nil {
				connectedSources++
			}
		}
		a.dataMu.RUnlock()

		return map[string]any{
			"state":             a.getState().String(),
			"publish_rate_hz":   a.config.PublishRateHz,
			"source_count":      len(a.config.Sources),
			"connected_sources": connectedSources,
			"frames_published":  a.framesPublished.Load(),
			"actual_rate_hz":    a.getPublishRate(),
		}, nil

	case "get_stats":
		a.dataMu.RLock()
		a.statsMu.RLock()
		sourceStatuses := make(map[string]any)
		var totalReceived uint64
		for _, src := range a.config.Sources {
			stats := a.sourceStats[src.Name]
			data := a.latestData[src.Name]

			srcStatus := map[string]any{
				"type":              string(src.Type),
				"required":          src.Required,
				"messages_received": stats.MessagesReceived.Load(),
				"actual_rate_hz":    stats.GetRate(),
			}

			if data != nil && data.Data != nil {
				srcStatus["connected"] = true
				srcStatus["age_ms"] = time.Since(data.ReceivedAt).Milliseconds()
				srcStatus["stale"] = time.Since(data.ReceivedAt) > src.StaleThreshold()
			} else {
				srcStatus["connected"] = false
				srcStatus["stale"] = true
			}

			sourceStatuses[src.Name] = srcStatus
			totalReceived += stats.MessagesReceived.Load()
		}
		a.statsMu.RUnlock()
		a.dataMu.RUnlock()

		return map[string]any{
			"frames_published":        a.framesPublished.Load(),
			"publish_errors":          a.publishErrors.Load(),
			"actual_rate_hz":          a.getPublishRate(),
			"total_messages_received": totalReceived,
			"sources":                 sourceStatuses,
		}, nil

	case "reset_stats":
		a.framesPublished.Store(0)
		a.publishErrors.Store(0)
		a.statsMu.Lock()
		for _, stats := range a.sourceStats {
			stats.MessagesReceived.Store(0)
			stats.MessagesDropped.Store(0)
		}
		a.statsMu.Unlock()
		return map[string]any{"success": true}, nil

	case "get_latest":
		frame := a.BuildTelemetryFrame()
		return map[string]any{
			"frame": frame,
		}, nil

	case "get_source_status":
		sourceName, _ := cmd["name"].(string)
		if sourceName == "" {
			return nil, fmt.Errorf("name parameter required")
		}

		a.dataMu.RLock()
		a.statsMu.RLock()
		data := a.latestData[sourceName]
		stats := a.sourceStats[sourceName]
		src := a.config.GetSourceConfig(sourceName)
		a.statsMu.RUnlock()
		a.dataMu.RUnlock()

		if src == nil {
			return nil, fmt.Errorf("source not found: %s", sourceName)
		}

		result := map[string]any{
			"name":              sourceName,
			"type":              string(src.Type),
			"subject":           src.Subject,
			"required":          src.Required,
			"stale_threshold":   src.StaleThresholdMs,
			"messages_received": stats.MessagesReceived.Load(),
			"actual_rate_hz":    stats.GetRate(),
		}

		if data != nil && data.Data != nil {
			result["connected"] = true
			result["age_ms"] = time.Since(data.ReceivedAt).Milliseconds()
			result["stale"] = time.Since(data.ReceivedAt) > src.StaleThreshold()
		} else {
			result["connected"] = false
			result["stale"] = true
		}

		return result, nil

	case "get_config":
		sources := make([]map[string]any, len(a.config.Sources))
		for i, src := range a.config.Sources {
			sources[i] = map[string]any{
				"name":            src.Name,
				"subject":         src.Subject,
				"type":            string(src.Type),
				"required":        src.Required,
				"stale_threshold": src.StaleThresholdMs,
			}
		}
		return map[string]any{
			"robot_id":             a.config.RobotID,
			"publish_rate_hz":      a.config.PublishRateHz,
			"include_system_stats": a.config.IncludeSystemStats,
			"include_motor_states": a.config.IncludeMotorStates,
			"sources":              sources,
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close stops the service and releases resources.
func (a *Aggregator) Close(ctx context.Context) error {
	a.stateMu.Lock()
	if a.state == StateStopped {
		a.stateMu.Unlock()
		return nil
	}
	a.state = StateStopped
	a.stateMu.Unlock()

	// Stop loops
	close(a.stopCh)
	<-a.doneCh

	a.logger.Info("Telemetry aggregator closed")
	return nil
}

// GetState returns the current state (for testing).
func (a *Aggregator) GetState() State {
	return a.getState()
}

// GetStats returns statistics (for testing).
func (a *Aggregator) GetStats() (framesPublished, publishErrors uint64) {
	return a.framesPublished.Load(), a.publishErrors.Load()
}

// GetConfig returns the configuration (for testing).
func (a *Aggregator) GetConfig() *Config {
	return a.config
}
