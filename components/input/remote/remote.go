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

	"github.com/emergingrobotics/gorai/components/input"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
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

// MessageType identifies the type of keyboard message.
type MessageType string

const (
	// MessageTypeKey is a regular key event.
	MessageTypeKey MessageType = "key"
	// MessageTypeStatus is a status update (connect/disconnect).
	MessageTypeStatus MessageType = "status"
)

// StatusType identifies the keyboard connection status.
type StatusType string

const (
	// StatusConnected indicates the keyboard is connected and publishing.
	StatusConnected StatusType = "connected"
	// StatusDisconnected indicates the keyboard has disconnected.
	StatusDisconnected StatusType = "disconnected"
)

// KeyEventMessage matches the JSON format from keyboard_publisher.
type KeyEventMessage struct {
	Type      MessageType   `json:"type"`
	Timestamp int64         `json:"timestamp"`
	Seq       uint64        `json:"seq"`
	Key       string        `json:"key,omitempty"`
	Code      uint16        `json:"code,omitempty"`
	Pressed   bool          `json:"pressed,omitempty"`
	Repeat    bool          `json:"repeat,omitempty"`
	Modifiers ModifiersData `json:"modifiers,omitempty"`
	Status    StatusType    `json:"status,omitempty"`
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

type eventSubscriber struct {
	ch  chan input.KeyEvent
	ctx context.Context
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

	// Fan-out event subscribers
	subscribers   []*eventSubscriber
	subscribersMu sync.Mutex
	closed        bool

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
	sub, err := r.nc.Subscribe(r.config.Subject, r.handleMessage)
	if err != nil {
		r.mu.Lock()
		r.state = StateError
		r.errorMsg = fmt.Sprintf("failed to subscribe: %v", err)
		r.mu.Unlock()
		return fmt.Errorf("failed to subscribe to %s: %w", r.config.Subject, err)
	}
	r.sub = sub

	r.mu.Lock()
	r.state = StateConnected
	r.lastEventTime = time.Now() // Start fresh
	r.mu.Unlock()

	// Start stale detection loop
	go r.staleDetectionLoop()

	r.logger.Info("Remote keyboard started",
		"subject", r.config.Subject,
		"buffer_size", r.config.BufferSize,
	)

	return nil
}

// handleMessage processes incoming NATS messages.
func (r *RemoteKeyboard) handleMessage(msg *nats.Msg) {
	r.eventsReceived.Add(1)
	r.updateReceiveRate()

	// Decode message
	var keyMsg KeyEventMessage
	if err := json.Unmarshal(msg.Data, &keyMsg); err != nil {
		r.logger.Warn("failed to decode message", "error", err)
		return
	}

	// Handle status messages
	if keyMsg.Type == MessageTypeStatus {
		r.handleStatusMessage(keyMsg)
		return
	}

	// Update last event time for key events
	r.mu.Lock()
	r.lastEventTime = time.Now()

	// If we were disconnected/stale, recover
	if r.state == StateStale || r.state == StateError {
		r.state = StateConnected
		r.logger.Info("remote keyboard connection restored")
	}
	r.mu.Unlock()

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

// handleStatusMessage processes keyboard status messages.
func (r *RemoteKeyboard) handleStatusMessage(msg KeyEventMessage) {
	r.logger.Info("received keyboard status",
		"status", msg.Status,
		"seq", msg.Seq,
	)

	r.mu.Lock()
	defer r.mu.Unlock()

	switch msg.Status {
	case StatusConnected:
		// Keyboard connected/reconnected
		if r.state != StateConnected {
			r.state = StateConnected
			r.logger.Info("remote keyboard connected")
		}

	case StatusDisconnected:
		// Keyboard disconnected - release all keys
		r.state = StateStale
		r.logger.Warn("remote keyboard disconnected, releasing all keys")
		r.releaseAllKeysLocked()
	}
}

// sendEvent broadcasts an event to all subscriber channels.
func (r *RemoteKeyboard) sendEvent(event input.KeyEvent) {
	r.subscribersMu.Lock()
	defer r.subscribersMu.Unlock()

	if r.closed {
		return
	}

	for _, sub := range r.subscribers {
		select {
		case sub.ch <- event:
		default:
			// Buffer full for this subscriber, drop oldest and retry
			select {
			case <-sub.ch:
				r.logger.Debug("dropped oldest event due to subscriber buffer overflow")
			default:
			}
			select {
			case sub.ch <- event:
			default:
				r.logger.Warn("failed to send event to subscriber, buffer full")
			}
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
// Note: This is for monitoring only. Keys are only released when an explicit
// disconnect message is received from the keyboard publisher.
func (r *RemoteKeyboard) checkStale() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Skip if not connected (stale detection only monitors connected state)
	if r.state != StateConnected {
		return
	}

	// Note: We no longer transition to stale or release keys based on timeout.
	// The stale threshold is now only used for monitoring/logging purposes.
	// Keys are released only when an explicit disconnect message is received.
	timeSinceLast := time.Since(r.lastEventTime)
	if timeSinceLast > r.config.StaleThreshold() {
		// Log for monitoring, but don't change state or release keys
		r.logger.Debug("no keyboard events received recently",
			"last_event", timeSinceLast.String(),
			"threshold", r.config.StaleThreshold().String(),
		)
	}
}

// releaseAllKeysLocked releases all pressed keys. Must be called with mu held.
func (r *RemoteKeyboard) releaseAllKeysLocked() {
	for code := range r.pressedKeys {
		event := input.KeyEvent{
			Key:     input.KeyCodeToName(code),
			Code:    code,
			Pressed: false,
			Repeat:  false,
		}
		r.sendEvent(event)
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

	// Check if subject changed (requires restart)
	r.mu.RLock()
	subjectChanged := r.config.Subject != cfg.Subject
	r.mu.RUnlock()

	if subjectChanged {
		return fmt.Errorf("subject cannot be changed at runtime, restart required")
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
			"subject":       r.config.Subject,
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
			"subject":                    r.config.Subject,
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

	// Close all subscriber channels
	r.subscribersMu.Lock()
	r.closed = true
	for _, sub := range r.subscribers {
		close(sub.ch)
	}
	r.subscribers = nil
	r.subscribersMu.Unlock()

	r.logger.Info("Remote keyboard closed")
	return nil
}

// Events returns a new channel of key events for this caller.
// Each caller gets an independent channel; every event is broadcast to all.
func (r *RemoteKeyboard) Events(ctx context.Context) (<-chan input.KeyEvent, error) {
	r.mu.RLock()
	state := r.state
	r.mu.RUnlock()

	if state != StateConnected && state != StateStale {
		return nil, fmt.Errorf("not connected (state: %s)", state.String())
	}

	ch := make(chan input.KeyEvent, r.config.BufferSize)
	sub := &eventSubscriber{ch: ch, ctx: ctx}

	r.subscribersMu.Lock()
	if r.closed {
		r.subscribersMu.Unlock()
		close(ch)
		return nil, fmt.Errorf("keyboard is closed")
	}
	r.subscribers = append(r.subscribers, sub)
	r.subscribersMu.Unlock()

	go r.watchSubscriberContext(sub)
	return ch, nil
}

// watchSubscriberContext removes and closes a subscriber when its context is done.
func (r *RemoteKeyboard) watchSubscriberContext(sub *eventSubscriber) {
	<-sub.ctx.Done()
	r.removeSubscriber(sub)
}

// removeSubscriber unregisters a subscriber and closes its channel.
func (r *RemoteKeyboard) removeSubscriber(sub *eventSubscriber) {
	r.subscribersMu.Lock()
	defer r.subscribersMu.Unlock()

	for i, s := range r.subscribers {
		if s == sub {
			r.subscribers = append(r.subscribers[:i], r.subscribers[i+1:]...)
			close(sub.ch)
			return
		}
	}
}

// IsPressed returns true if the specified key is currently pressed.
func (r *RemoteKeyboard) IsPressed(ctx context.Context, key string) (bool, error) {
	code, ok := input.KeyNameToCode(key)
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
		keys = append(keys, input.KeyCodeToName(code))
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
