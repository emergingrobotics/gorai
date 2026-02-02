// Package keyboard_publisher provides a bridge service that publishes local
// keyboard events to the NATS message bus for distributed robot control.
package keyboard_publisher

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorai/gorai/components/input"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterService("bridge", "keyboard_publisher", New)
}

// State represents the operational state of the service.
type State int

const (
	StateStopped State = iota
	StateStarting
	StateRunning
	StateDisconnected
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
	case StateDisconnected:
		return "disconnected"
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

// KeyEventMessage is the message format published to NATS.
// Uses a simple JSON-serializable struct for compatibility.
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

// ModifiersData holds modifier key state.
type ModifiersData struct {
	Shift    bool `json:"shift"`
	Ctrl     bool `json:"ctrl"`
	Alt      bool `json:"alt"`
	Meta     bool `json:"meta"`
	CapsLock bool `json:"caps_lock"`
	NumLock  bool `json:"num_lock"`
}

// Publisher is the keyboard publisher bridge service.
type Publisher struct {
	name   resource.Name
	config *Config
	logger *slog.Logger

	// Dependencies
	keyboard input.Keyboard
	nc       *nats.Conn

	// State
	mu       sync.RWMutex
	state    State
	errorMsg string

	// Sequence number
	seq atomic.Uint64

	// Statistics
	eventsReceived  atomic.Uint64
	eventsPublished atomic.Uint64
	eventsDropped   atomic.Uint64

	// Rate tracking
	publishTimes []time.Time
	publishMu    sync.Mutex
	publishRate  float64

	// Control
	stopCh chan struct{}
	doneCh chan struct{}
}

// New creates a new Keyboard Publisher service.
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
	name := "keyboard_publisher"
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

	p := &Publisher{
		name:         resource.NewServiceName("gorai", "bridge", name),
		config:       cfg,
		logger:       logger.With("service", "keyboard_publisher", "name", name),
		publishTimes: make([]time.Time, 0, 100),
		state:        StateStopped,
		stopCh:       make(chan struct{}),
		doneCh:       make(chan struct{}),
	}

	// Get NATS connection from dependencies
	if natsRes, err := deps.Get("nats"); err == nil {
		if nc, ok := natsRes.(*nats.Conn); ok {
			p.nc = nc
		}
	}

	p.logger.Info("Keyboard publisher created",
		"keyboard", cfg.Keyboard,
		"topic", cfg.GetTopic(),
	)

	return p, nil
}

// Name returns the resource name.
func (p *Publisher) Name() resource.Name {
	return p.name
}

// Reconfigure updates the service configuration and resolves dependencies.
func (p *Publisher) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Stop if running
	p.mu.Lock()
	wasRunning := p.state == StateRunning
	p.mu.Unlock()

	if wasRunning {
		if err := p.stop(); err != nil {
			p.logger.Warn("failed to stop during reconfigure", "error", err)
		}
	}

	// Resolve keyboard dependency
	kbName := resource.NewComponentName("gorai", "input", cfg.Keyboard)
	kbRes, err := deps.Get(kbName)
	if err != nil {
		return fmt.Errorf("keyboard component %q not found: %w", cfg.Keyboard, err)
	}

	keyboard, ok := kbRes.(input.Keyboard)
	if !ok {
		return fmt.Errorf("component %q is not an input.Keyboard", cfg.Keyboard)
	}

	// Update configuration
	p.mu.Lock()
	p.config = cfg
	p.keyboard = keyboard
	p.mu.Unlock()

	// Start if it was running
	if wasRunning {
		if err := p.start(ctx); err != nil {
			return fmt.Errorf("failed to restart: %w", err)
		}
	}

	p.logger.Info("Keyboard publisher reconfigured",
		"keyboard", cfg.Keyboard,
		"topic", cfg.GetTopic(),
	)

	return nil
}

// Start begins the keyboard event forwarding.
func (p *Publisher) Start(ctx context.Context) error {
	return p.start(ctx)
}

// start begins the event forwarding loop.
func (p *Publisher) start(ctx context.Context) error {
	p.mu.Lock()
	if p.state == StateRunning {
		p.mu.Unlock()
		return nil
	}

	if p.keyboard == nil {
		p.mu.Unlock()
		return fmt.Errorf("keyboard not configured, call Reconfigure first")
	}

	p.state = StateStarting
	p.stopCh = make(chan struct{})
	p.doneCh = make(chan struct{})
	p.mu.Unlock()

	// Get events channel from keyboard
	eventsCh, err := p.keyboard.Events(ctx)
	if err != nil {
		p.mu.Lock()
		p.state = StateError
		p.errorMsg = fmt.Sprintf("failed to get keyboard events: %v", err)
		p.mu.Unlock()
		return err
	}

	p.mu.Lock()
	p.state = StateRunning
	p.mu.Unlock()

	// Start event forwarding goroutine
	go p.eventLoop(eventsCh)

	// Start heartbeat goroutine
	go p.heartbeatLoop()

	p.logger.Info("Keyboard publisher started",
		"topic", p.config.GetTopic(),
	)

	return nil
}

// stop halts the event forwarding.
func (p *Publisher) stop() error {
	p.mu.Lock()
	if p.state == StateStopped {
		p.mu.Unlock()
		return nil
	}
	p.state = StateStopped
	p.mu.Unlock()

	// Signal stop
	close(p.stopCh)

	// Wait for goroutines to finish
	<-p.doneCh

	p.logger.Info("Keyboard publisher stopped")
	return nil
}

// eventLoop forwards keyboard events to NATS.
func (p *Publisher) eventLoop(eventsCh <-chan input.KeyEvent) {
	defer close(p.doneCh)

	// Publish connected status on start
	p.publishStatus(StatusConnected)

	for {
		select {
		case <-p.stopCh:
			// Publish disconnect status before stopping
			p.publishStatus(StatusDisconnected)
			return

		case event, ok := <-eventsCh:
			if !ok {
				// Channel closed, keyboard disconnected
				p.mu.Lock()
				p.state = StateDisconnected
				p.errorMsg = "keyboard event channel closed"
				p.mu.Unlock()

				// Publish disconnect status
				p.publishStatus(StatusDisconnected)
				p.logger.Warn("keyboard event channel closed")
				return
			}

			p.eventsReceived.Add(1)

			// Filter repeat events if configured
			if event.Repeat && !p.config.PublishRepeat {
				continue
			}

			// Build message
			msg := KeyEventMessage{
				Type:      MessageTypeKey,
				Timestamp: time.Now().UnixNano(),
				Seq:       p.seq.Add(1),
				Key:       event.Key,
				Code:      event.Code,
				Pressed:   event.Pressed,
				Repeat:    event.Repeat,
			}

			if p.config.IncludeModifiers {
				msg.Modifiers = ModifiersData{
					Shift:    event.Modifiers.Shift,
					Ctrl:     event.Modifiers.Ctrl,
					Alt:      event.Modifiers.Alt,
					Meta:     event.Modifiers.Meta,
					CapsLock: event.Modifiers.CapsLock,
					NumLock:  event.Modifiers.NumLock,
				}
			}

			// Publish to NATS
			if err := p.publish(msg); err != nil {
				p.eventsDropped.Add(1)
				p.logger.Warn("failed to publish event", "key", event.Key, "error", err)
			} else {
				p.eventsPublished.Add(1)
				p.updatePublishRate()
				p.logger.Debug("published key event", "key", event.Key, "pressed", event.Pressed)
			}
		}
	}
}

// publishStatus publishes a status message to NATS.
func (p *Publisher) publishStatus(status StatusType) {
	msg := KeyEventMessage{
		Type:      MessageTypeStatus,
		Timestamp: time.Now().UnixNano(),
		Seq:       p.seq.Add(1),
		Status:    status,
	}

	if err := p.publish(msg); err != nil {
		p.logger.Warn("failed to publish status", "status", status, "error", err)
	} else {
		p.logger.Info("published keyboard status", "status", status)
	}
}

// publish sends a key event message to NATS.
func (p *Publisher) publish(msg KeyEventMessage) error {
	if p.nc == nil {
		return fmt.Errorf("NATS connection not available")
	}

	topic := p.config.GetTopic()

	// Encode as JSON
	data, err := encodeJSON(msg)
	if err != nil {
		return fmt.Errorf("failed to encode message: %w", err)
	}

	return p.nc.Publish(topic, data)
}

// updatePublishRate updates the publish rate calculation.
func (p *Publisher) updatePublishRate() {
	p.publishMu.Lock()
	defer p.publishMu.Unlock()

	now := time.Now()

	// Add new timestamp
	p.publishTimes = append(p.publishTimes, now)

	// Remove timestamps older than 1 second
	cutoff := now.Add(-time.Second)
	for len(p.publishTimes) > 0 && p.publishTimes[0].Before(cutoff) {
		p.publishTimes = p.publishTimes[1:]
	}

	p.publishRate = float64(len(p.publishTimes))
}

// getPublishRate returns the current publish rate.
func (p *Publisher) getPublishRate() float64 {
	p.publishMu.Lock()
	defer p.publishMu.Unlock()
	return p.publishRate
}

// heartbeatLoop publishes status messages.
func (p *Publisher) heartbeatLoop() {
	ticker := time.NewTicker(p.config.HeartbeatInterval())
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			// Just update internal state for now
			// Status messages would be published here if needed
		}
	}
}

// DoCommand handles arbitrary commands.
func (p *Publisher) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)

	switch cmdName {
	case "get_state":
		p.mu.RLock()
		state := p.state.String()
		errorMsg := p.errorMsg
		p.mu.RUnlock()

		return map[string]any{
			"state":         state,
			"error_message": errorMsg,
			"topic":         p.config.GetTopic(),
			"keyboard":      p.config.Keyboard,
		}, nil

	case "get_stats":
		return map[string]any{
			"events_received":  p.eventsReceived.Load(),
			"events_published": p.eventsPublished.Load(),
			"events_dropped":   p.eventsDropped.Load(),
			"publish_rate_hz":  p.getPublishRate(),
		}, nil

	case "reset_stats":
		p.eventsReceived.Store(0)
		p.eventsPublished.Store(0)
		p.eventsDropped.Store(0)
		return map[string]any{"success": true}, nil

	case "get_config":
		return map[string]any{
			"keyboard":           p.config.Keyboard,
			"topic":              p.config.GetTopic(),
			"publish_repeat":     p.config.PublishRepeat,
			"heartbeat_interval": p.config.HeartbeatIntervalMs,
			"include_modifiers":  p.config.IncludeModifiers,
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close stops the service and releases resources.
func (p *Publisher) Close(ctx context.Context) error {
	if err := p.stop(); err != nil {
		return err
	}
	p.logger.Info("Keyboard publisher closed")
	return nil
}

// GetState returns the current operational state.
func (p *Publisher) GetState() State {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.state
}

// GetStats returns event statistics.
func (p *Publisher) GetStats() (received, published, dropped uint64) {
	return p.eventsReceived.Load(), p.eventsPublished.Load(), p.eventsDropped.Load()
}

// encodeJSON is a simple JSON encoder for the message.
func encodeJSON(msg KeyEventMessage) ([]byte, error) {
	// Simple manual JSON encoding to avoid import cycles
	if msg.Type == MessageTypeStatus {
		return []byte(fmt.Sprintf(
			`{"type":%q,"timestamp":%d,"seq":%d,"status":%q}`,
			msg.Type, msg.Timestamp, msg.Seq, msg.Status,
		)), nil
	}

	return []byte(fmt.Sprintf(
		`{"type":%q,"timestamp":%d,"seq":%d,"key":%q,"code":%d,"pressed":%t,"repeat":%t,"modifiers":{"shift":%t,"ctrl":%t,"alt":%t,"meta":%t,"caps_lock":%t,"num_lock":%t}}`,
		msg.Type, msg.Timestamp, msg.Seq, msg.Key, msg.Code, msg.Pressed, msg.Repeat,
		msg.Modifiers.Shift, msg.Modifiers.Ctrl, msg.Modifiers.Alt, msg.Modifiers.Meta,
		msg.Modifiers.CapsLock, msg.Modifiers.NumLock,
	)), nil
}
