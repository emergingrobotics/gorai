// Package v4l2 provides a V4L2 camera component for Gorai that wraps the
// V4L2 driver and adds NATS message bus publishing.
package v4l2

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	v4l2driver "github.com/emergingrobotics/gorai/driver/camera/v4l2"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("camera", "v4l2", New)
}

// State represents the operational state of the camera.
type State int

const (
	StateClosed State = iota
	StateOpening
	StateStreaming
	StateError
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpening:
		return "opening"
	case StateStreaming:
		return "streaming"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Camera is the V4L2 camera component.
type Camera struct {
	name   resource.Name
	config *Config
	logger *slog.Logger

	driver *v4l2driver.Camera

	mu       sync.RWMutex
	state    State
	errorMsg string

	// Rate limiting
	publishInterval time.Duration
	lastPublishTime time.Time

	// Latest frame for Image() calls
	latestFrame     []byte
	latestFrameTime time.Time
	frameMu         sync.RWMutex

	// Sequence number
	seq atomic.Uint64

	// Metrics
	framesCaptured  atomic.Uint64
	framesPublished atomic.Uint64
	framesDropped   atomic.Uint64
	encodeErrors    atomic.Uint64
	publishErrors   atomic.Uint64
	totalFrameBytes atomic.Uint64
	frameCount      atomic.Uint64

	// FPS tracking
	captureTimes  []time.Time
	publishTimes  []time.Time
	fpsCalcMu     sync.Mutex
	captureFPS    float64
	publishFPS    float64

	// Callbacks (for NATS integration)
	onFrame   func(jpeg []byte, timestamp time.Time, seq uint64, frameID string)
	onFrameMu sync.RWMutex

	stopCh chan struct{}
	doneCh chan struct{}
}

// New creates a new V4L2 camera component.
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
	name := "camera"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	c := &Camera{
		name:            resource.NewComponentName("gorai", "camera", name),
		config:          cfg,
		logger:          slog.Default().With("component", "camera", "name", name),
		state:           StateClosed,
		publishInterval: cfg.PublishInterval(),
		captureTimes:    make([]time.Time, 0, 100),
		publishTimes:    make([]time.Time, 0, 100),
		stopCh:          make(chan struct{}),
		doneCh:          make(chan struct{}),
	}

	// Create V4L2 driver
	driverCfg := &v4l2driver.Config{
		Device:      cfg.Device,
		Width:       uint32(cfg.Width),
		Height:      uint32(cfg.Height),
		FrameRate:   cfg.FrameRate,
		JPEGQuality: cfg.JPEGQuality,
	}

	driver, err := v4l2driver.New(driverCfg,
		v4l2driver.WithLogger(c.logger),
		v4l2driver.WithOnFrame(c.handleFrame),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create V4L2 driver: %w", err)
	}
	c.driver = driver

	// Open and start the camera
	c.mu.Lock()
	c.state = StateOpening
	c.mu.Unlock()

	if err := driver.Open(); err != nil {
		c.mu.Lock()
		c.state = StateError
		c.errorMsg = err.Error()
		c.mu.Unlock()
		return nil, fmt.Errorf("failed to open camera: %w", err)
	}

	if err := driver.Start(ctx); err != nil {
		driver.Close()
		c.mu.Lock()
		c.state = StateError
		c.errorMsg = err.Error()
		c.mu.Unlock()
		return nil, fmt.Errorf("failed to start camera: %w", err)
	}

	c.mu.Lock()
	c.state = StateStreaming
	c.mu.Unlock()

	// Start heartbeat loop
	go c.heartbeatLoop()

	c.logger.Info("Camera component started",
		"device", cfg.Device,
		"width", cfg.Width,
		"height", cfg.Height,
		"frame_rate", cfg.FrameRate,
		"publish_rate", cfg.PublishRateHz,
	)

	return c, nil
}

// handleFrame is called by the V4L2 driver for each captured frame.
func (c *Camera) handleFrame(jpegData []byte, timestamp time.Time) {
	c.framesCaptured.Add(1)
	c.updateCaptureFPS(timestamp)

	// Store latest frame for Image() calls
	c.frameMu.Lock()
	c.latestFrame = make([]byte, len(jpegData))
	copy(c.latestFrame, jpegData)
	c.latestFrameTime = timestamp
	c.frameMu.Unlock()

	// Rate limiting for publishing
	c.fpsCalcMu.Lock()
	lastPublish := c.lastPublishTime
	c.fpsCalcMu.Unlock()
	if c.publishInterval > 0 && !lastPublish.IsZero() {
		elapsed := timestamp.Sub(lastPublish)
		if elapsed < c.publishInterval {
			c.framesDropped.Add(1)
			return
		}
	}

	// Update statistics
	c.totalFrameBytes.Add(uint64(len(jpegData)))
	c.frameCount.Add(1)

	// Get sequence number
	seq := c.seq.Add(1)

	// Call frame callback if set (for NATS publishing)
	c.onFrameMu.RLock()
	callback := c.onFrame
	c.onFrameMu.RUnlock()
	if callback != nil {
		callback(jpegData, timestamp, seq, c.config.FrameID)
	}

	c.framesPublished.Add(1)
	c.fpsCalcMu.Lock()
	c.lastPublishTime = timestamp
	c.fpsCalcMu.Unlock()
	c.updatePublishFPS(timestamp)
}

// updateCaptureFPS updates the capture FPS calculation.
func (c *Camera) updateCaptureFPS(t time.Time) {
	c.fpsCalcMu.Lock()
	defer c.fpsCalcMu.Unlock()

	// Add new timestamp
	c.captureTimes = append(c.captureTimes, t)

	// Remove timestamps older than 1 second
	cutoff := t.Add(-time.Second)
	for len(c.captureTimes) > 0 && c.captureTimes[0].Before(cutoff) {
		c.captureTimes = c.captureTimes[1:]
	}

	c.captureFPS = float64(len(c.captureTimes))
}

// updatePublishFPS updates the publish FPS calculation.
func (c *Camera) updatePublishFPS(t time.Time) {
	c.fpsCalcMu.Lock()
	defer c.fpsCalcMu.Unlock()

	// Add new timestamp
	c.publishTimes = append(c.publishTimes, t)

	// Remove timestamps older than 1 second
	cutoff := t.Add(-time.Second)
	for len(c.publishTimes) > 0 && c.publishTimes[0].Before(cutoff) {
		c.publishTimes = c.publishTimes[1:]
	}

	c.publishFPS = float64(len(c.publishTimes))
}

// SetFrameCallback sets the callback for frame publishing.
func (c *Camera) SetFrameCallback(fn func(jpeg []byte, timestamp time.Time, seq uint64, frameID string)) {
	c.onFrameMu.Lock()
	c.onFrame = fn
	c.onFrameMu.Unlock()
}

// heartbeatLoop runs the heartbeat loop.
func (c *Camera) heartbeatLoop() {
	defer close(c.doneCh)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			// Heartbeat - check for stale frames
			c.frameMu.RLock()
			lastFrame := c.latestFrameTime
			c.frameMu.RUnlock()

			c.mu.RLock()
			state := c.state
			c.mu.RUnlock()

			if state == StateStreaming && !lastFrame.IsZero() {
				if time.Since(lastFrame) > 5*time.Second {
					c.logger.Warn("no frames received for 5 seconds")
				}
			}
		}
	}
}

// Name returns the resource name.
func (c *Camera) Name() resource.Name {
	return c.name
}

// Reconfigure updates the component configuration.
func (c *Camera) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	// For now, require restart for config changes
	return fmt.Errorf("reconfiguration not supported, restart required")
}

// DoCommand handles arbitrary commands.
func (c *Camera) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)

	switch cmdName {
	case "get_state":
		c.mu.RLock()
		state := c.state.String()
		errorMsg := c.errorMsg
		c.mu.RUnlock()

		c.fpsCalcMu.Lock()
		captureFPS := c.captureFPS
		publishFPS := c.publishFPS
		c.fpsCalcMu.Unlock()

		return map[string]any{
			"state":         state,
			"error_message": errorMsg,
			"width":         c.config.Width,
			"height":        c.config.Height,
			"frame_rate":    c.config.FrameRate,
			"capture_fps":   captureFPS,
			"publish_fps":   publishFPS,
		}, nil

	case "get_stats":
		frameCount := c.frameCount.Load()
		totalBytes := c.totalFrameBytes.Load()
		avgSizeKB := float64(0)
		if frameCount > 0 {
			avgSizeKB = float64(totalBytes) / float64(frameCount) / 1024.0
		}

		c.fpsCalcMu.Lock()
		captureFPS := c.captureFPS
		publishFPS := c.publishFPS
		c.fpsCalcMu.Unlock()

		return map[string]any{
			"frames_captured":    c.framesCaptured.Load(),
			"frames_published":   c.framesPublished.Load(),
			"frames_dropped":     c.framesDropped.Load(),
			"encode_errors":      c.encodeErrors.Load(),
			"publish_errors":     c.publishErrors.Load(),
			"capture_fps":        captureFPS,
			"publish_fps":        publishFPS,
			"avg_frame_size_kb":  avgSizeKB,
		}, nil

	case "reset_stats":
		c.framesCaptured.Store(0)
		c.framesPublished.Store(0)
		c.framesDropped.Store(0)
		c.encodeErrors.Store(0)
		c.publishErrors.Store(0)
		c.totalFrameBytes.Store(0)
		c.frameCount.Store(0)
		return map[string]any{"success": true}, nil

	case "capture_snapshot":
		// Return the latest frame
		c.frameMu.RLock()
		frame := c.latestFrame
		timestamp := c.latestFrameTime
		c.frameMu.RUnlock()

		if frame == nil {
			return nil, fmt.Errorf("no frame available")
		}

		return map[string]any{
			"data":      frame,
			"timestamp": timestamp.Format(time.RFC3339Nano),
			"size":      len(frame),
		}, nil

	case "get_config":
		return map[string]any{
			"device":          c.config.Device,
			"width":           c.config.Width,
			"height":         c.config.Height,
			"frame_rate":      c.config.FrameRate,
			"jpeg_quality":    c.config.JPEGQuality,
			"publish_to_bus":  c.config.PublishToBus,
			"publish_rate_hz": c.config.PublishRateHz,
			"frame_id":        c.config.FrameID,
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close stops the camera and releases resources.
func (c *Camera) Close(ctx context.Context) error {
	c.mu.Lock()
	if c.state == StateClosed {
		c.mu.Unlock()
		return nil
	}
	c.state = StateClosed
	c.mu.Unlock()

	// Stop heartbeat loop
	close(c.stopCh)
	<-c.doneCh

	// Stop and close driver
	if c.driver != nil {
		c.driver.Stop()
		c.driver.Close()
	}

	c.logger.Info("Camera component closed")
	return nil
}

// Image returns a single image from the camera.
func (c *Camera) Image(ctx context.Context) (image.Image, error) {
	c.frameMu.RLock()
	frame := c.latestFrame
	c.frameMu.RUnlock()

	if frame == nil {
		// Wait for a frame
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
			c.frameMu.RLock()
			frame = c.latestFrame
			c.frameMu.RUnlock()

			if frame == nil {
				return nil, fmt.Errorf("no frame available")
			}
		}
	}

	return jpeg.Decode(bytes.NewReader(frame))
}

// Stream returns a channel of images for continuous capture.
func (c *Camera) Stream(ctx context.Context) (<-chan image.Image, error) {
	imgCh := make(chan image.Image, 2)

	go func() {
		defer close(imgCh)

		ticker := time.NewTicker(c.config.CaptureInterval())
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-c.stopCh:
				return
			case <-ticker.C:
				c.frameMu.RLock()
				frame := c.latestFrame
				c.frameMu.RUnlock()

				if frame == nil {
					continue
				}

				img, err := jpeg.Decode(bytes.NewReader(frame))
				if err != nil {
					c.logger.Warn("failed to decode frame", "error", err)
					continue
				}

				select {
				case imgCh <- img:
				default:
					// Channel full, skip frame
				}
			}
		}
	}()

	return imgCh, nil
}

// Properties returns the camera's properties.
func (c *Camera) Properties(ctx context.Context) (Properties, error) {
	return Properties{
		Width:     c.config.Width,
		Height:    c.config.Height,
		FrameRate: c.config.FrameRate,
	}, nil
}

// Properties describes camera capabilities.
type Properties struct {
	Width     int
	Height    int
	FrameRate float64
}

// GetState returns the current state.
func (c *Camera) GetState() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// GetStats returns the current statistics.
func (c *Camera) GetStats() (captured, published, dropped, encodeErr, publishErr uint64) {
	return c.framesCaptured.Load(),
		c.framesPublished.Load(),
		c.framesDropped.Load(),
		c.encodeErrors.Load(),
		c.publishErrors.Load()
}

// GetConfig returns the camera configuration.
func (c *Camera) GetConfig() *Config {
	return c.config
}

// GetFPS returns the current capture and publish FPS.
func (c *Camera) GetFPS() (captureFPS, publishFPS float64) {
	c.fpsCalcMu.Lock()
	defer c.fpsCalcMu.Unlock()
	return c.captureFPS, c.publishFPS
}

// GetLatestFrame returns the latest captured frame.
func (c *Camera) GetLatestFrame() ([]byte, time.Time) {
	c.frameMu.RLock()
	defer c.frameMu.RUnlock()
	return c.latestFrame, c.latestFrameTime
}

