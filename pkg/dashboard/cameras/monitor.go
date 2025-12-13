// Package cameras provides camera monitoring and streaming for the dashboard.
package cameras

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/gorai/gorai/pkg/config"
	gorainats "github.com/gorai/gorai/pkg/nats"
	"github.com/gorai/gorai/pkg/topics"
	"github.com/nats-io/nats.go"
)

// CameraInfo represents camera configuration and current status.
type CameraInfo struct {
	Name      string    `json:"name"`
	Device    string    `json:"device"`
	Online    bool      `json:"online"`
	FPS       float64   `json:"fps"`
	Width     int       `json:"width"`
	Height    int       `json:"height"`
	LastFrame time.Time `json:"last_frame"`
}

// CameraStatus is sent via WebSocket for real-time updates.
type CameraStatus struct {
	Type       string  `json:"type"`
	Camera     string  `json:"camera"`
	Online     bool    `json:"online"`
	FPS        float64 `json:"fps"`
	Resolution string  `json:"resolution,omitempty"`
	LastFrame  string  `json:"last_frame,omitempty"`
}

// cameraState holds internal state for tracking a camera.
type cameraState struct {
	info         CameraInfo
	frameCount   int
	fpsWindow    time.Time
	subscription *nats.Subscription
}

// Monitor tracks camera status by observing NATS frames.
type Monitor struct {
	mu       sync.RWMutex
	cameras  map[string]*cameraState
	nats     *gorainats.Client
	topics   *topics.Builder
	robotCfg *config.RDL
	logger   *slog.Logger

	// Callback for status changes
	onStatusChange func(CameraStatus)

	ctx        context.Context
	cancel     context.CancelFunc
	offlineTTL time.Duration
}

// MonitorOption configures a Monitor.
type MonitorOption func(*Monitor)

// WithMonitorLogger sets the logger for the monitor.
func WithMonitorLogger(logger *slog.Logger) MonitorOption {
	return func(m *Monitor) {
		m.logger = logger
	}
}

// WithOfflineTTL sets the duration after which a camera is marked offline.
func WithOfflineTTL(ttl time.Duration) MonitorOption {
	return func(m *Monitor) {
		m.offlineTTL = ttl
	}
}

// NewMonitor creates a new camera monitor.
func NewMonitor(natsClient *gorainats.Client, topicsBuilder *topics.Builder, robotCfg *config.RDL, opts ...MonitorOption) *Monitor {
	m := &Monitor{
		cameras:    make(map[string]*cameraState),
		nats:       natsClient,
		topics:     topicsBuilder,
		robotCfg:   robotCfg,
		logger:     slog.Default(),
		offlineTTL: 5 * time.Second,
	}

	for _, opt := range opts {
		opt(m)
	}

	// Initialize camera states from config
	if robotCfg != nil {
		for _, comp := range robotCfg.Components {
			if comp.Type == "camera" && !comp.Disabled {
				device := "/dev/video0"
				width := 640
				height := 480

				if comp.Attributes != nil {
					if d, ok := comp.Attributes["device"].(string); ok {
						device = d
					}
					if w, ok := comp.Attributes["width"].(float64); ok {
						width = int(w)
					}
					if h, ok := comp.Attributes["height"].(float64); ok {
						height = int(h)
					}
				}

				m.cameras[comp.Name] = &cameraState{
					info: CameraInfo{
						Name:   comp.Name,
						Device: device,
						Width:  width,
						Height: height,
						Online: false,
					},
					fpsWindow: time.Now(),
				}
			}
		}
	}

	return m
}

// Start begins monitoring all configured cameras.
func (m *Monitor) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	// Subscribe to each camera's data topic
	m.mu.Lock()
	for name := range m.cameras {
		if err := m.subscribeToCamera(name); err != nil {
			m.logger.Warn("Failed to subscribe to camera", "camera", name, "error", err)
		}
	}
	m.mu.Unlock()

	// Start offline detection goroutine
	go m.offlineDetectionLoop()

	m.logger.Info("Camera monitor started", "cameras", len(m.cameras))
	return nil
}

// Stop stops the camera monitor.
func (m *Monitor) Stop() {
	if m.cancel != nil {
		m.cancel()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, state := range m.cameras {
		if state.subscription != nil {
			state.subscription.Unsubscribe()
			state.subscription = nil
		}
	}

	m.logger.Info("Camera monitor stopped")
}

// subscribeToCamera subscribes to a camera's data topic.
// Must be called with m.mu held.
func (m *Monitor) subscribeToCamera(cameraName string) error {
	if m.nats == nil || m.topics == nil {
		return nil
	}

	topic := m.topics.ComponentData(cameraName)

	sub, err := m.nats.Subscribe(topic, func(msg *nats.Msg) {
		m.handleFrame(cameraName, msg.Data)
	})
	if err != nil {
		return err
	}

	m.cameras[cameraName].subscription = sub
	m.logger.Debug("Subscribed to camera", "camera", cameraName, "topic", topic)
	return nil
}

// handleFrame processes an incoming camera frame.
func (m *Monitor) handleFrame(cameraName string, jpeg []byte) {
	m.mu.Lock()
	state, ok := m.cameras[cameraName]
	if !ok {
		m.mu.Unlock()
		return
	}

	wasOffline := !state.info.Online
	state.info.Online = true
	state.info.LastFrame = time.Now()
	state.frameCount++

	// Calculate FPS every second
	shouldNotify := false
	if time.Since(state.fpsWindow) >= time.Second {
		state.info.FPS = float64(state.frameCount) / time.Since(state.fpsWindow).Seconds()
		state.frameCount = 0
		state.fpsWindow = time.Now()
		shouldNotify = true
	}

	// Also notify if camera just came online
	if wasOffline {
		shouldNotify = true
	}

	m.mu.Unlock()

	// Notify status change outside of lock
	if shouldNotify && m.onStatusChange != nil {
		m.onStatusChange(CameraStatus{
			Type:   "camera_status",
			Camera: cameraName,
			Online: true,
			FPS:    state.info.FPS,
		})
	}
}

// offlineDetectionLoop periodically checks for offline cameras.
func (m *Monitor) offlineDetectionLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkOffline()
		}
	}
}

// checkOffline marks cameras as offline if no frames received recently.
func (m *Monitor) checkOffline() {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	for name, state := range m.cameras {
		if state.info.Online && now.Sub(state.info.LastFrame) > m.offlineTTL {
			state.info.Online = false
			state.info.FPS = 0

			// Notify outside of lock via goroutine
			if m.onStatusChange != nil {
				go m.onStatusChange(CameraStatus{
					Type:   "camera_status",
					Camera: name,
					Online: false,
					FPS:    0,
				})
			}

			m.logger.Info("Camera offline", "camera", name)
		}
	}
}

// OnStatusChange sets a callback for status updates.
func (m *Monitor) OnStatusChange(fn func(CameraStatus)) {
	m.onStatusChange = fn
}

// GetCameras returns current status of all cameras.
func (m *Monitor) GetCameras() []CameraInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cameras := make([]CameraInfo, 0, len(m.cameras))
	for _, state := range m.cameras {
		cameras = append(cameras, state.info)
	}
	return cameras
}

// GetCamera returns status of a specific camera.
func (m *Monitor) GetCamera(name string) (CameraInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, ok := m.cameras[name]
	if !ok {
		return CameraInfo{}, false
	}
	return state.info, true
}
