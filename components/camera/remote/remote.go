// Package remote provides a remote camera component that subscribes to
// camera frames from the NATS message bus.
package remote

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

	"github.com/gorai/gorai/components/camera"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterComponent("camera", "remote", New)
}

// State represents the operational state of the component.
type State int

const (
	StateClosed State = iota
	StateStarting
	StateConnected
	StateStale
	StateError
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateStarting:
		return "starting"
	case StateConnected:
		return "connected"
	case StateStale:
		return "stale"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// Frame represents a captured frame with metadata.
type Frame struct {
	Data      []byte
	Timestamp time.Time
}

// RemoteCamera implements camera.Camera by subscribing to NATS frames.
type RemoteCamera struct {
	name   resource.Name
	config *Config
	logger *slog.Logger

	// NATS
	nc  *nats.Conn
	sub *nats.Subscription

	// State
	mu            sync.RWMutex
	state         State
	errorMsg      string
	lastFrameTime time.Time

	// Frame buffer (ring buffer)
	frames     []Frame
	frameHead  int
	frameCount int
	frameMu    sync.RWMutex

	// Stream channel
	streamCh     chan image.Image
	streamChMu   sync.Mutex
	streamChOpen bool

	// Statistics
	framesReceived  atomic.Uint64
	bufferOverflows atomic.Uint64

	// FPS tracking
	receiveTimes []time.Time
	receiveMu    sync.Mutex
	receiveFPS   float64

	// Control
	stopCh chan struct{}
	doneCh chan struct{}
}

// New creates a new Remote Camera component.
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
	name := "remote_camera"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	// Get logger from dependencies or use default
	logger := slog.Default()
	if loggerRes, err := deps.Get("logger"); err == nil {
		if l, ok := loggerRes.(*slog.Logger); ok {
			logger = l
		}
	}

	r := &RemoteCamera{
		name:         resource.NewComponentName("gorai", "camera", name),
		config:       cfg,
		logger:       logger.With("component", "remote_camera", "name", name),
		state:        StateClosed,
		frames:       make([]Frame, cfg.BufferSize),
		receiveTimes: make([]time.Time, 0, 100),
		streamCh:     make(chan image.Image, 5),
		streamChOpen: true,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}

	// Get NATS connection from dependencies
	if natsRes, err := deps.Get("nats"); err == nil {
		if nc, ok := natsRes.(*nats.Conn); ok {
			r.nc = nc
		}
	}

	// Start the component
	if err := r.start(ctx); err != nil {
		return nil, err
	}

	return r, nil
}

// start initializes the NATS subscription and begins processing.
func (r *RemoteCamera) start(ctx context.Context) error {
	r.mu.Lock()
	r.state = StateStarting
	r.mu.Unlock()

	if r.nc == nil {
		r.mu.Lock()
		r.state = StateError
		r.errorMsg = "NATS connection not available"
		r.mu.Unlock()
		return fmt.Errorf("NATS connection not available")
	}

	// Subscribe to camera frames
	sub, err := r.nc.Subscribe(r.config.Topic, r.handleFrame)
	if err != nil {
		r.mu.Lock()
		r.state = StateError
		r.errorMsg = fmt.Sprintf("failed to subscribe: %v", err)
		r.mu.Unlock()
		return fmt.Errorf("failed to subscribe to %s: %w", r.config.Topic, err)
	}
	r.sub = sub

	r.mu.Lock()
	r.state = StateConnected
	r.lastFrameTime = time.Now() // Start fresh
	r.mu.Unlock()

	// Start stale detection loop
	go r.staleDetectionLoop()

	r.logger.Info("Remote camera started",
		"topic", r.config.Topic,
		"buffer_size", r.config.BufferSize,
		"width", r.config.Width,
		"height", r.config.Height,
	)

	return nil
}

// handleFrame processes incoming NATS messages containing JPEG frames.
func (r *RemoteCamera) handleFrame(msg *nats.Msg) {
	r.framesReceived.Add(1)
	r.updateReceiveFPS()

	now := time.Now()

	// Update last frame time and state
	r.mu.Lock()
	r.lastFrameTime = now
	if r.state == StateStale {
		r.state = StateConnected
		r.logger.Info("remote camera connection restored")
	}
	r.mu.Unlock()

	// Store frame in ring buffer
	r.frameMu.Lock()
	frame := Frame{
		Data:      make([]byte, len(msg.Data)),
		Timestamp: now,
	}
	copy(frame.Data, msg.Data)

	r.frames[r.frameHead] = frame
	r.frameHead = (r.frameHead + 1) % len(r.frames)
	if r.frameCount < len(r.frames) {
		r.frameCount++
	} else {
		r.bufferOverflows.Add(1)
	}
	r.frameMu.Unlock()

	// Try to decode and send to stream channel (non-blocking)
	r.sendToStream(msg.Data)
}

// sendToStream decodes the frame and sends it to the stream channel.
func (r *RemoteCamera) sendToStream(jpegData []byte) {
	r.streamChMu.Lock()
	defer r.streamChMu.Unlock()

	if !r.streamChOpen {
		return
	}

	// Decode JPEG to image
	img, err := jpeg.Decode(bytes.NewReader(jpegData))
	if err != nil {
		r.logger.Debug("failed to decode JPEG frame", "error", err)
		return
	}

	// Non-blocking send
	select {
	case r.streamCh <- img:
	default:
		// Channel full, drop frame
	}
}

// updateReceiveFPS updates the receive FPS calculation.
func (r *RemoteCamera) updateReceiveFPS() {
	r.receiveMu.Lock()
	defer r.receiveMu.Unlock()

	now := time.Now()

	// Add new timestamp
	r.receiveTimes = append(r.receiveTimes, now)

	// Remove timestamps older than 1 second
	cutoff := now.Add(-time.Second)
	for len(r.receiveTimes) > 0 && r.receiveTimes[0].Before(cutoff) {
		r.receiveTimes = r.receiveTimes[1:]
	}

	r.receiveFPS = float64(len(r.receiveTimes))
}

// getReceiveFPS returns the current receive FPS.
func (r *RemoteCamera) getReceiveFPS() float64 {
	r.receiveMu.Lock()
	defer r.receiveMu.Unlock()
	return r.receiveFPS
}

// staleDetectionLoop monitors for connection staleness.
func (r *RemoteCamera) staleDetectionLoop() {
	defer close(r.doneCh)

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.checkStale()
		}
	}
}

// checkStale checks if the connection has become stale.
func (r *RemoteCamera) checkStale() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Skip if not connected
	if r.state != StateConnected && r.state != StateStale {
		return
	}

	timeSinceLast := time.Since(r.lastFrameTime)
	wasStale := r.state == StateStale
	nowStale := timeSinceLast > r.config.StaleThreshold()

	if nowStale && !wasStale {
		// Became stale
		r.state = StateStale
		r.logger.Warn("remote camera connection stale",
			"last_frame", timeSinceLast.String(),
			"threshold", r.config.StaleThreshold().String(),
		)
	} else if !nowStale && wasStale {
		// Recovered
		r.state = StateConnected
		r.logger.Info("remote camera connection restored")
	}
}

// getLatestFrame returns the most recent frame from the buffer.
func (r *RemoteCamera) getLatestFrame() (*Frame, error) {
	r.frameMu.RLock()
	defer r.frameMu.RUnlock()

	if r.frameCount == 0 {
		return nil, fmt.Errorf("no frames available")
	}

	// Get the most recent frame (one before head in ring buffer)
	idx := (r.frameHead - 1 + len(r.frames)) % len(r.frames)
	frame := &r.frames[idx]

	return frame, nil
}

// Name returns the resource name.
func (r *RemoteCamera) Name() resource.Name {
	return r.name
}

// Reconfigure updates the component configuration.
func (r *RemoteCamera) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Check if topic changed (requires restart)
	r.mu.RLock()
	topicChanged := r.config.Topic != cfg.Topic
	r.mu.RUnlock()

	if topicChanged {
		return fmt.Errorf("topic cannot be changed at runtime, restart required")
	}

	r.mu.Lock()
	r.config = cfg
	r.mu.Unlock()

	return nil
}

// DoCommand handles arbitrary commands.
func (r *RemoteCamera) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)

	switch cmdName {
	case "get_state":
		r.mu.RLock()
		state := r.state.String()
		errorMsg := r.errorMsg
		lastFrame := r.lastFrameTime
		r.mu.RUnlock()

		return map[string]any{
			"state":         state,
			"error_message": errorMsg,
			"topic":         r.config.Topic,
			"last_frame_ms": time.Since(lastFrame).Milliseconds(),
			"width":         r.config.Width,
			"height":        r.config.Height,
		}, nil

	case "get_stats":
		r.frameMu.RLock()
		bufferedFrames := r.frameCount
		r.frameMu.RUnlock()

		return map[string]any{
			"frames_received":  r.framesReceived.Load(),
			"buffer_overflows": r.bufferOverflows.Load(),
			"buffered_frames":  bufferedFrames,
			"receive_fps":      r.getReceiveFPS(),
		}, nil

	case "get_config":
		return map[string]any{
			"topic":              r.config.Topic,
			"width":              r.config.Width,
			"height":             r.config.Height,
			"buffer_size":        r.config.BufferSize,
			"stale_threshold_ms": r.config.StaleThresholdMs,
		}, nil

	case "capture_snapshot":
		frame, err := r.getLatestFrame()
		if err != nil {
			return nil, err
		}
		return map[string]any{
			"data":      frame.Data,
			"timestamp": frame.Timestamp.Format(time.RFC3339Nano),
			"size":      len(frame.Data),
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close releases all resources.
func (r *RemoteCamera) Close(ctx context.Context) error {
	r.mu.Lock()
	if r.state == StateClosed {
		r.mu.Unlock()
		return nil
	}
	r.mu.Unlock()

	// Signal stop
	close(r.stopCh)

	// Wait for stale detection loop
	<-r.doneCh

	// Unsubscribe
	if r.sub != nil {
		r.sub.Unsubscribe()
	}

	// Close stream channel
	r.streamChMu.Lock()
	if r.streamChOpen {
		r.streamChOpen = false
		close(r.streamCh)
	}
	r.streamChMu.Unlock()

	r.mu.Lock()
	r.state = StateClosed
	r.mu.Unlock()

	r.logger.Info("Remote camera closed")
	return nil
}

// Image returns a single image from the camera.
func (r *RemoteCamera) Image(ctx context.Context) (image.Image, error) {
	frame, err := r.getLatestFrame()
	if err != nil {
		// Wait briefly for a frame
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Second):
			frame, err = r.getLatestFrame()
			if err != nil {
				return nil, err
			}
		}
	}

	return jpeg.Decode(bytes.NewReader(frame.Data))
}

// Stream returns a channel of images for continuous streaming.
func (r *RemoteCamera) Stream(ctx context.Context) (<-chan image.Image, error) {
	r.mu.RLock()
	state := r.state
	r.mu.RUnlock()

	if state != StateConnected && state != StateStale {
		return nil, fmt.Errorf("not connected (state: %s)", state.String())
	}

	return r.streamCh, nil
}

// Properties returns the camera's properties.
func (r *RemoteCamera) Properties(ctx context.Context) (camera.Properties, error) {
	return camera.Properties{
		Width:     r.config.Width,
		Height:    r.config.Height,
		FrameRate: r.getReceiveFPS(),
	}, nil
}

// GetState returns the current operational state.
func (r *RemoteCamera) GetState() State {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

// GetStats returns receive statistics.
func (r *RemoteCamera) GetStats() (received, overflows uint64) {
	return r.framesReceived.Load(), r.bufferOverflows.Load()
}

// GetLatestFrameRaw returns the latest frame as raw JPEG bytes.
func (r *RemoteCamera) GetLatestFrameRaw() ([]byte, time.Time, error) {
	frame, err := r.getLatestFrame()
	if err != nil {
		return nil, time.Time{}, err
	}
	return frame.Data, frame.Timestamp, nil
}

// Verify interface compliance
var _ camera.Camera = (*RemoteCamera)(nil)
