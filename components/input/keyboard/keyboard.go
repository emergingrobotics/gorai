// Package keyboard implements a keyboard input component using Linux evdev.
package keyboard

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorai/gorai/components/input"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("input", "keyboard", New)
}

// State represents the operational state of the keyboard component.
type State int

const (
	StateClosed State = iota
	StateOpening
	StateRunning
	StateDisconnected
	StateError
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpening:
		return "opening"
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

// Keyboard implements the input.Keyboard interface using Linux evdev.
type Keyboard struct {
	name   resource.Name
	config *Config
	logger *slog.Logger

	file       *os.File
	grabbed    bool
	deviceName string

	mu               sync.RWMutex
	state            State
	pressedKeys      map[uint16]bool
	modifiers        input.Modifiers
	leftShiftPressed bool
	rightShiftPressed bool
	leftCtrlPressed  bool
	rightCtrlPressed bool
	leftAltPressed   bool
	rightAltPressed  bool
	leftMetaPressed  bool
	rightMetaPressed bool

	stopCh chan struct{}
	doneCh chan struct{}

	eventCh       chan input.KeyEvent
	eventChMu     sync.Mutex
	eventChClosed bool

	eventsReceived  atomic.Uint64
	eventsPublished atomic.Uint64

	errorMsg string
}

// New creates a new keyboard component.
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
	name := "keyboard"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	k := &Keyboard{
		name:        resource.NewComponentName("gorai", "input", name),
		config:      cfg,
		logger:      slog.Default().With("component", "keyboard", "name", name),
		state:       StateClosed,
		pressedKeys: make(map[uint16]bool),
		stopCh:      make(chan struct{}),
		doneCh:      make(chan struct{}),
		eventCh:     make(chan input.KeyEvent, 100),
	}

	if err := k.open(); err != nil {
		return nil, fmt.Errorf("failed to open device: %w", err)
	}

	return k, nil
}

// open initializes the keyboard device.
func (k *Keyboard) open() error {
	k.mu.Lock()
	defer k.mu.Unlock()

	k.state = StateOpening

	// Open the input device
	file, err := os.Open(k.config.Device)
	if err != nil {
		k.state = StateError
		k.errorMsg = fmt.Sprintf("failed to open device: %v", err)
		return fmt.Errorf("failed to open %s: %w", k.config.Device, err)
	}
	k.file = file

	// Get device name
	fd := int(file.Fd())
	deviceName, err := getDeviceName(fd)
	if err != nil {
		k.logger.Warn("failed to get device name", "error", err)
		deviceName = "unknown"
	}
	k.deviceName = deviceName
	k.logger.Info("opened keyboard device", "device", k.config.Device, "name", deviceName)

	// Optionally grab the device for exclusive access
	if k.config.GrabDevice {
		if err := grabDevice(fd); err != nil {
			k.logger.Warn("failed to grab device", "error", err)
		} else {
			k.grabbed = true
			k.logger.Info("grabbed device exclusively")
		}
	}

	k.state = StateRunning
	k.errorMsg = ""

	// Start the event reading goroutine
	go k.eventLoop()

	return nil
}

// eventLoop reads events from the input device and publishes them.
func (k *Keyboard) eventLoop() {
	defer close(k.doneCh)

	buf := make([]byte, inputEventSize)

	for {
		select {
		case <-k.stopCh:
			return
		default:
		}

		// Read one event (blocking)
		n, err := k.file.Read(buf)
		if err != nil {
			k.mu.Lock()
			if k.state == StateRunning {
				k.state = StateDisconnected
				k.errorMsg = fmt.Sprintf("read error: %v", err)
				k.logger.Error("device read error", "error", err)
			}
			k.mu.Unlock()

			// Check if we should stop
			select {
			case <-k.stopCh:
				return
			default:
				// Try reconnection after delay
				time.Sleep(time.Second)
				continue
			}
		}

		if n < inputEventSize {
			k.logger.Warn("incomplete event", "bytes", n)
			continue
		}

		// Parse the event
		ev, err := parseInputEvent(buf)
		if err != nil {
			k.logger.Warn("failed to parse event", "error", err)
			continue
		}

		k.eventsReceived.Add(1)

		// Only process key events
		if !ev.isKeyEvent() {
			continue
		}

		// Handle the key event
		if ev.isPress() {
			k.processKeyPress(ev.Code)
		} else if ev.isRelease() {
			k.processKeyRelease(ev.Code)
		} else if ev.isRepeat() && k.config.PublishRepeat {
			k.processKeyRepeat(ev.Code)
		}
	}
}

// processKeyPress handles a key press event.
func (k *Keyboard) processKeyPress(code uint16) {
	k.mu.Lock()

	// Update pressed keys map
	k.pressedKeys[code] = true

	// Update modifiers
	k.updateModifiers(code, true)

	// Get current modifiers (copy under lock)
	mods := k.modifiers

	k.mu.Unlock()

	// Build and publish event
	keyName := KeyCodeToName(code)
	event := input.KeyEvent{
		Key:       keyName,
		Code:      code,
		Pressed:   true,
		Modifiers: mods,
		Repeat:    false,
	}

	k.publishEvent(event)
	k.logger.Debug("key pressed", "key", keyName, "code", code)
}

// processKeyRelease handles a key release event.
func (k *Keyboard) processKeyRelease(code uint16) {
	k.mu.Lock()

	// Update pressed keys map
	delete(k.pressedKeys, code)

	// Update modifiers
	k.updateModifiers(code, false)

	// Get current modifiers (copy under lock)
	mods := k.modifiers

	k.mu.Unlock()

	// Build and publish event
	keyName := KeyCodeToName(code)
	event := input.KeyEvent{
		Key:       keyName,
		Code:      code,
		Pressed:   false,
		Modifiers: mods,
		Repeat:    false,
	}

	k.publishEvent(event)
	k.logger.Debug("key released", "key", keyName, "code", code)
}

// processKeyRepeat handles a key repeat event.
func (k *Keyboard) processKeyRepeat(code uint16) {
	k.mu.RLock()
	mods := k.modifiers
	k.mu.RUnlock()

	keyName := KeyCodeToName(code)
	event := input.KeyEvent{
		Key:       keyName,
		Code:      code,
		Pressed:   true,
		Modifiers: mods,
		Repeat:    true,
	}

	k.publishEvent(event)
}

// updateModifiers updates the modifier state based on a key event.
// Must be called with mu held.
func (k *Keyboard) updateModifiers(code uint16, pressed bool) {
	switch code {
	case KeyLeftShift:
		k.leftShiftPressed = pressed
		k.modifiers.Shift = k.leftShiftPressed || k.rightShiftPressed
	case KeyRightShift:
		k.rightShiftPressed = pressed
		k.modifiers.Shift = k.leftShiftPressed || k.rightShiftPressed
	case KeyLeftCtrl:
		k.leftCtrlPressed = pressed
		k.modifiers.Ctrl = k.leftCtrlPressed || k.rightCtrlPressed
	case KeyRightCtrl:
		k.rightCtrlPressed = pressed
		k.modifiers.Ctrl = k.leftCtrlPressed || k.rightCtrlPressed
	case KeyLeftAlt:
		k.leftAltPressed = pressed
		k.modifiers.Alt = k.leftAltPressed || k.rightAltPressed
	case KeyRightAlt:
		k.rightAltPressed = pressed
		k.modifiers.Alt = k.leftAltPressed || k.rightAltPressed
	case KeyLeftMeta:
		k.leftMetaPressed = pressed
		k.modifiers.Meta = k.leftMetaPressed || k.rightMetaPressed
	case KeyRightMeta:
		k.rightMetaPressed = pressed
		k.modifiers.Meta = k.leftMetaPressed || k.rightMetaPressed
	case KeyCapsLock:
		if pressed {
			k.modifiers.CapsLock = !k.modifiers.CapsLock // Toggle
		}
	case KeyNumLock:
		if pressed {
			k.modifiers.NumLock = !k.modifiers.NumLock // Toggle
		}
	}
}

// publishEvent sends an event to all subscribers.
func (k *Keyboard) publishEvent(event input.KeyEvent) {
	k.eventChMu.Lock()
	defer k.eventChMu.Unlock()

	if k.eventChClosed {
		return
	}

	select {
	case k.eventCh <- event:
		k.eventsPublished.Add(1)
	default:
		// Channel full, drop event
		k.logger.Warn("event channel full, dropping event", "key", event.Key)
	}
}

// Name returns the resource name.
func (k *Keyboard) Name() resource.Name {
	return k.name
}

// Reconfigure updates the keyboard configuration.
func (k *Keyboard) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Check if device changed
	k.mu.Lock()
	deviceChanged := k.config.Device != cfg.Device
	k.mu.Unlock()

	if deviceChanged {
		// Close and reopen with new device
		if err := k.Close(ctx); err != nil {
			return fmt.Errorf("failed to close for reconfigure: %w", err)
		}

		k.config = cfg
		k.stopCh = make(chan struct{})
		k.doneCh = make(chan struct{})
		k.eventCh = make(chan input.KeyEvent, 100)
		k.eventChClosed = false

		if err := k.open(); err != nil {
			return fmt.Errorf("failed to reopen: %w", err)
		}
	} else {
		k.mu.Lock()
		k.config = cfg
		k.mu.Unlock()
	}

	return nil
}

// DoCommand handles arbitrary commands.
func (k *Keyboard) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)

	switch cmdName {
	case "get_state":
		k.mu.RLock()
		state := k.state.String()
		k.mu.RUnlock()
		return map[string]any{"state": state}, nil

	case "get_stats":
		return map[string]any{
			"events_received":  k.eventsReceived.Load(),
			"events_published": k.eventsPublished.Load(),
		}, nil

	case "get_device_info":
		k.mu.RLock()
		info := map[string]any{
			"device_path": k.config.Device,
			"device_name": k.deviceName,
			"grabbed":     k.grabbed,
		}
		k.mu.RUnlock()
		return info, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close releases all resources.
func (k *Keyboard) Close(ctx context.Context) error {
	// Signal stop
	close(k.stopCh)

	// Close event channel
	k.eventChMu.Lock()
	if !k.eventChClosed {
		k.eventChClosed = true
		close(k.eventCh)
	}
	k.eventChMu.Unlock()

	// Wait for event loop to finish
	select {
	case <-k.doneCh:
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(5 * time.Second):
		k.logger.Warn("timeout waiting for event loop to stop")
	}

	k.mu.Lock()
	defer k.mu.Unlock()

	// Release device grab
	if k.grabbed && k.file != nil {
		if err := releaseDevice(int(k.file.Fd())); err != nil {
			k.logger.Warn("failed to release device grab", "error", err)
		}
		k.grabbed = false
	}

	// Close file
	if k.file != nil {
		if err := k.file.Close(); err != nil {
			k.logger.Warn("failed to close device file", "error", err)
		}
		k.file = nil
	}

	k.state = StateClosed
	k.logger.Info("keyboard closed")

	return nil
}

// Events returns a channel of key events.
func (k *Keyboard) Events(ctx context.Context) (<-chan input.KeyEvent, error) {
	k.mu.RLock()
	state := k.state
	k.mu.RUnlock()

	if state != StateRunning {
		return nil, fmt.Errorf("keyboard not running (state: %s)", state.String())
	}

	return k.eventCh, nil
}

// IsPressed returns true if the specified key is currently pressed.
func (k *Keyboard) IsPressed(ctx context.Context, key string) (bool, error) {
	code, ok := KeyNameToCode(key)
	if !ok {
		return false, fmt.Errorf("unknown key: %s", key)
	}

	k.mu.RLock()
	pressed := k.pressedKeys[code]
	k.mu.RUnlock()

	return pressed, nil
}

// GetPressedKeys returns a list of currently pressed key names.
func (k *Keyboard) GetPressedKeys(ctx context.Context) ([]string, error) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	keys := make([]string, 0, len(k.pressedKeys))
	for code := range k.pressedKeys {
		keys = append(keys, KeyCodeToName(code))
	}

	return keys, nil
}

// GetModifiers returns the current state of modifier keys.
func (k *Keyboard) GetModifiers(ctx context.Context) (input.Modifiers, error) {
	k.mu.RLock()
	mods := k.modifiers
	k.mu.RUnlock()

	return mods, nil
}

// GetState returns the current operational state.
func (k *Keyboard) GetState() State {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.state
}

// GetStats returns event statistics.
func (k *Keyboard) GetStats() (received, published uint64) {
	return k.eventsReceived.Load(), k.eventsPublished.Load()
}

// Verify interface compliance
var _ input.Keyboard = (*Keyboard)(nil)

