// Package remote provides a remote keyboard component that subscribes to
// keyboard events from the NATS message bus.
package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorai/gorai/components/input"
	"github.com/gorai/gorai/components/input/keyboard"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterComponent("input", "remote", New)
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

// KeyEventMessage matches the JSON format from keyboard_publisher.
type KeyEventMessage struct {
	Timestamp int64         `json:"timestamp"`
	Seq       uint64        `json:"seq"`
	Key       string        `json:"key"`
	Code      uint16        `json:"code"`
	Pressed   bool          `json:"pressed"`
	Repeat    bool          `json:"repeat"`
	Modifiers ModifiersData `json:"modifiers"`
}

// ModifiersData matches the JSON format from keyboard_publisher.
type ModifiersData struct {
	Shift    bool `json:"shift"`
	Ctrl     bool `json:"ctrl"`
	Alt      bool `json:"alt"`
	Meta     bool `json:"meta"`
	CapsLock bool `json:"caps_lock"`
	NumLock  bool `json:"num_lock"`
}

// RemoteKeyboard implements input.Keyboard by subscribing to NATS events.
type RemoteKeyboard struct {
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
	pressedKeys   map[uint16]bool
	modifiers     input.Modifiers
	lastEventTime time.Time

	// Event channel
	eventCh       chan input.KeyEvent
	eventChMu     sync.Mutex
	eventChClosed bool

	// Statistics
	eventsReceived atomic.Uint64

	// Rate tracking
	receiveTimes []time.Time
	receiveMu    sync.Mutex
	receiveRate  float64

	// Control
	stopCh chan struct{}
	doneCh chan struct{}
}

// New creates a new Remote Keyboard component.
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
	name := "remote_keyboard"
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

	r := &RemoteKeyboard{
		name:         resource.NewComponentName("gorai", "input", name),
		config:       cfg,
		logger:       logger.With("component", "remote_keyboard", "name", name),
		state:        StateClosed,
		pressedKeys:  make(map[uint16]bool),
		eventCh:      make(chan input.KeyEvent, cfg.BufferSize),
		receiveTimes: make([]time.Time, 0, 100),
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
func (r *RemoteKeyboard) start(ctx context.Context) error {
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

	// Subscribe to keyboard events
	sub, err := r.nc.Subscribe(r.config.Topic, r.handleMessage)
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
	r.lastEventTime = time.Now() // Start fresh
	r.mu.Unlock()

	// Start stale detection loop
	go r.staleDetectionLoop()

	r.logger.Info("Remote keyboard started",
		"topic", r.config.Topic,
		"buffer_size", r.config.BufferSize,
	)

	return nil
}

// handleMessage processes incoming NATS messages.
func (r *RemoteKeyboard) handleMessage(msg *nats.Msg) {
	r.eventsReceived.Add(1)
	r.updateReceiveRate()

	// Update last event time
	r.mu.Lock()
	r.lastEventTime = time.Now()

	// If we were stale, recover
	if r.state == StateStale {
		r.state = StateConnected
		r.logger.Info("remote keyboard connection restored")
	}
	r.mu.Unlock()

	// Decode message
	var keyMsg KeyEventMessage
	if err := json.Unmarshal(msg.Data, &keyMsg); err != nil {
		r.logger.Warn("failed to decode message", "error", err)
		return
	}

	// Log received event
	r.logger.Debug("received key event",
		"key", keyMsg.Key,
		"pressed", keyMsg.Pressed,
		"seq", keyMsg.Seq,
	)

	// Convert to input.KeyEvent
	event := input.KeyEvent{
		Key:     keyMsg.Key,
		Code:    keyMsg.Code,
		Pressed: keyMsg.Pressed,
		Repeat:  keyMsg.Repeat,
		Modifiers: input.Modifiers{
			Shift:    keyMsg.Modifiers.Shift,
			Ctrl:     keyMsg.Modifiers.Ctrl,
			Alt:      keyMsg.Modifiers.Alt,
			Meta:     keyMsg.Modifiers.Meta,
			CapsLock: keyMsg.Modifiers.CapsLock,
			NumLock:  keyMsg.Modifiers.NumLock,
		},
	}

	// Update internal state
	r.mu.Lock()
	if event.Pressed {
		r.pressedKeys[event.Code] = true
	} else {
		delete(r.pressedKeys, event.Code)
	}
	r.modifiers = event.Modifiers
	r.mu.Unlock()

	// Send to event channel (non-blocking)
	r.sendEvent(event)
}

// sendEvent sends an event to the Events() channel, handling overflow.
func (r *RemoteKeyboard) sendEvent(event input.KeyEvent) {
	r.eventChMu.Lock()
	defer r.eventChMu.Unlock()

	if r.eventChClosed {
		return
	}

	select {
	case r.eventCh <- event:
		// Sent successfully
	default:
		// Channel full, drop oldest and retry
		select {
		case <-r.eventCh:
			r.logger.Debug("dropped oldest event due to buffer overflow")
		default:
		}
		select {
		case r.eventCh <- event:
		default:
			r.logger.Warn("failed to send event, buffer full")
		}
	}
}

// updateReceiveRate updates the receive rate calculation.
func (r *RemoteKeyboard) updateReceiveRate() {
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

	r.receiveRate = float64(len(r.receiveTimes))
}

// getReceiveRate returns the current receive rate.
func (r *RemoteKeyboard) getReceiveRate() float64 {
	r.receiveMu.Lock()
	defer r.receiveMu.Unlock()
	return r.receiveRate
}

// staleDetectionLoop monitors for connection staleness.
func (r *RemoteKeyboard) staleDetectionLoop() {
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
func (r *RemoteKeyboard) checkStale() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Skip if not connected
	if r.state != StateConnected && r.state != StateStale {
		return
	}

	timeSinceLast := time.Since(r.lastEventTime)
	wasStale := r.state == StateStale
	nowStale := timeSinceLast > r.config.StaleThreshold()

	if nowStale && !wasStale {
		// Became stale
		r.state = StateStale
		r.logger.Warn("remote keyboard connection stale",
			"last_event", timeSinceLast.String(),
			"threshold", r.config.StaleThreshold().String(),
		)

		if r.config.AutoReleaseOnDisconnect {
			r.releaseAllKeysLocked()
		}
	} else if !nowStale && wasStale {
		// Recovered
		r.state = StateConnected
		r.logger.Info("remote keyboard connection restored")
	}
}

// releaseAllKeysLocked releases all pressed keys. Must be called with mu held.
func (r *RemoteKeyboard) releaseAllKeysLocked() {
	for code := range r.pressedKeys {
		event := input.KeyEvent{
			Key:     keyboard.KeyCodeToName(code),
			Code:    code,
			Pressed: false,
			Repeat:  false,
		}

		// Send release event (non-blocking, mutex already held)
		r.eventChMu.Lock()
		if !r.eventChClosed {
			select {
			case r.eventCh <- event:
			default:
			}
		}
		r.eventChMu.Unlock()
	}

	r.pressedKeys = make(map[uint16]bool)
	r.modifiers = input.Modifiers{}

	r.logger.Info("released all keys due to disconnect")
}

// Name returns the resource name.
func (r *RemoteKeyboard) Name() resource.Name {
	return r.name
}

// Reconfigure updates the component configuration.
func (r *RemoteKeyboard) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
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
func (r *RemoteKeyboard) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)

	switch cmdName {
	case "get_state":
		r.mu.RLock()
		state := r.state.String()
		errorMsg := r.errorMsg
		lastEvent := r.lastEventTime
		r.mu.RUnlock()

		return map[string]any{
			"state":         state,
			"error_message": errorMsg,
			"topic":         r.config.Topic,
			"last_event_ms": time.Since(lastEvent).Milliseconds(),
		}, nil

	case "get_stats":
		return map[string]any{
			"events_received": r.eventsReceived.Load(),
			"receive_rate_hz": r.getReceiveRate(),
		}, nil

	case "get_pressed_keys":
		keys, _ := r.GetPressedKeys(ctx)
		return map[string]any{
			"pressed_keys": keys,
		}, nil

	case "get_modifiers":
		mods, _ := r.GetModifiers(ctx)
		return map[string]any{
			"shift":     mods.Shift,
			"ctrl":      mods.Ctrl,
			"alt":       mods.Alt,
			"meta":      mods.Meta,
			"caps_lock": mods.CapsLock,
			"num_lock":  mods.NumLock,
		}, nil

	case "release_all":
		r.mu.Lock()
		r.releaseAllKeysLocked()
		r.mu.Unlock()
		return map[string]any{"success": true}, nil

	case "get_config":
		return map[string]any{
			"topic":                      r.config.Topic,
			"buffer_size":                r.config.BufferSize,
			"stale_threshold_ms":         r.config.StaleThresholdMs,
			"auto_release_on_disconnect": r.config.AutoReleaseOnDisconnect,
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close releases all resources.
func (r *RemoteKeyboard) Close(ctx context.Context) error {
	// Signal stop
	close(r.stopCh)

	// Wait for stale detection loop
	<-r.doneCh

	// Unsubscribe
	if r.sub != nil {
		r.sub.Unsubscribe()
	}

	// Release all keys
	r.mu.Lock()
	r.releaseAllKeysLocked()
	r.state = StateClosed
	r.mu.Unlock()

	// Close event channel
	r.eventChMu.Lock()
	if !r.eventChClosed {
		r.eventChClosed = true
		close(r.eventCh)
	}
	r.eventChMu.Unlock()

	r.logger.Info("Remote keyboard closed")
	return nil
}

// Events returns a channel of key events.
func (r *RemoteKeyboard) Events(ctx context.Context) (<-chan input.KeyEvent, error) {
	r.mu.RLock()
	state := r.state
	r.mu.RUnlock()

	if state != StateConnected && state != StateStale {
		return nil, fmt.Errorf("not connected (state: %s)", state.String())
	}

	return r.eventCh, nil
}

// IsPressed returns true if the specified key is currently pressed.
func (r *RemoteKeyboard) IsPressed(ctx context.Context, key string) (bool, error) {
	code, ok := keyboard.KeyNameToCode(key)
	if !ok {
		return false, fmt.Errorf("unknown key: %s", key)
	}

	r.mu.RLock()
	pressed := r.pressedKeys[code]
	r.mu.RUnlock()

	return pressed, nil
}

// GetPressedKeys returns a list of currently pressed key names.
func (r *RemoteKeyboard) GetPressedKeys(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	keys := make([]string, 0, len(r.pressedKeys))
	for code := range r.pressedKeys {
		keys = append(keys, keyboard.KeyCodeToName(code))
	}

	return keys, nil
}

// GetModifiers returns the current state of modifier keys.
func (r *RemoteKeyboard) GetModifiers(ctx context.Context) (input.Modifiers, error) {
	r.mu.RLock()
	mods := r.modifiers
	r.mu.RUnlock()

	return mods, nil
}

// GetState returns the current operational state.
func (r *RemoteKeyboard) GetState() State {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.state
}

// GetStats returns receive statistics.
func (r *RemoteKeyboard) GetStats() uint64 {
	return r.eventsReceived.Load()
}

// Verify interface compliance
var _ input.Keyboard = (*RemoteKeyboard)(nil)
