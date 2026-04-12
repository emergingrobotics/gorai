// Package fake provides a fake keyboard implementation for testing.
package fake

import (
	"context"
	"sync"

	"github.com/gorai/gorai/components/input"
	"github.com/gorai/gorai/components/input/keyboard"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("input", "fake_keyboard", New)
}

type eventSubscriber struct {
	ch  chan input.KeyEvent
	ctx context.Context
}

// FakeKeyboard is a fake keyboard for testing.
type FakeKeyboard struct {
	name resource.Name

	mu          sync.RWMutex
	pressedKeys map[string]bool
	modifiers   input.Modifiers

	// Fan-out event subscribers
	subscribers   []*eventSubscriber
	subscribersMu sync.Mutex
	closed        bool

	// Call tracking for test verification
	EventsSent []input.KeyEvent
}

// New creates a new fake keyboard.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	name := "fake_keyboard"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	return &FakeKeyboard{
		name:        resource.NewComponentName("gorai", "input", name),
		pressedKeys: make(map[string]bool),
		EventsSent:  make([]input.KeyEvent, 0),
	}, nil
}

// Name returns the resource name.
func (f *FakeKeyboard) Name() resource.Name {
	return f.name
}

// Reconfigure updates the configuration.
func (f *FakeKeyboard) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand handles arbitrary commands.
func (f *FakeKeyboard) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, nil
}

// Close releases all resources.
func (f *FakeKeyboard) Close(ctx context.Context) error {
	f.subscribersMu.Lock()
	defer f.subscribersMu.Unlock()

	if !f.closed {
		f.closed = true
		for _, sub := range f.subscribers {
			close(sub.ch)
		}
		f.subscribers = nil
	}
	return nil
}

// Events returns a new channel of key events for this caller.
// Each caller gets an independent channel; every event is broadcast to all.
func (f *FakeKeyboard) Events(ctx context.Context) (<-chan input.KeyEvent, error) {
	ch := make(chan input.KeyEvent, 100)
	sub := &eventSubscriber{ch: ch, ctx: ctx}

	f.subscribersMu.Lock()
	if f.closed {
		f.subscribersMu.Unlock()
		close(ch)
		return nil, nil
	}
	f.subscribers = append(f.subscribers, sub)
	f.subscribersMu.Unlock()

	go f.watchSubscriberContext(sub)
	return ch, nil
}

func (f *FakeKeyboard) watchSubscriberContext(sub *eventSubscriber) {
	<-sub.ctx.Done()
	f.removeSubscriber(sub)
}

func (f *FakeKeyboard) removeSubscriber(sub *eventSubscriber) {
	f.subscribersMu.Lock()
	defer f.subscribersMu.Unlock()

	for i, s := range f.subscribers {
		if s == sub {
			f.subscribers = append(f.subscribers[:i], f.subscribers[i+1:]...)
			close(sub.ch)
			return
		}
	}
}

// broadcastEvent sends an event to all subscriber channels (non-blocking).
func (f *FakeKeyboard) broadcastEvent(event input.KeyEvent) {
	f.subscribersMu.Lock()
	defer f.subscribersMu.Unlock()

	if f.closed {
		return
	}

	for _, sub := range f.subscribers {
		select {
		case sub.ch <- event:
		default:
		}
	}
}

// IsPressed returns true if the specified key is currently pressed.
func (f *FakeKeyboard) IsPressed(ctx context.Context, key string) (bool, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.pressedKeys[key], nil
}

// GetPressedKeys returns a list of currently pressed key names.
func (f *FakeKeyboard) GetPressedKeys(ctx context.Context) ([]string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	keys := make([]string, 0, len(f.pressedKeys))
	for key := range f.pressedKeys {
		keys = append(keys, key)
	}
	return keys, nil
}

// GetModifiers returns the current state of modifier keys.
func (f *FakeKeyboard) GetModifiers(ctx context.Context) (input.Modifiers, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.modifiers, nil
}

// SimulateKeyPress simulates a key press event.
func (f *FakeKeyboard) SimulateKeyPress(key string) {
	f.mu.Lock()
	f.pressedKeys[key] = true
	f.updateModifiers(key, true)
	mods := f.modifiers
	f.mu.Unlock()

	code, _ := keyboard.KeyNameToCode(key)
	event := input.KeyEvent{
		Key:       key,
		Code:      code,
		Pressed:   true,
		Modifiers: mods,
		Repeat:    false,
	}

	f.mu.Lock()
	f.EventsSent = append(f.EventsSent, event)
	f.mu.Unlock()

	f.broadcastEvent(event)
}

// SimulateKeyRelease simulates a key release event.
func (f *FakeKeyboard) SimulateKeyRelease(key string) {
	f.mu.Lock()
	delete(f.pressedKeys, key)
	f.updateModifiers(key, false)
	mods := f.modifiers
	f.mu.Unlock()

	code, _ := keyboard.KeyNameToCode(key)
	event := input.KeyEvent{
		Key:       key,
		Code:      code,
		Pressed:   false,
		Modifiers: mods,
		Repeat:    false,
	}

	f.mu.Lock()
	f.EventsSent = append(f.EventsSent, event)
	f.mu.Unlock()

	f.broadcastEvent(event)
}

// SimulateKeyRepeat simulates a key repeat event.
func (f *FakeKeyboard) SimulateKeyRepeat(key string) {
	f.mu.RLock()
	mods := f.modifiers
	f.mu.RUnlock()

	code, _ := keyboard.KeyNameToCode(key)
	event := input.KeyEvent{
		Key:       key,
		Code:      code,
		Pressed:   true,
		Modifiers: mods,
		Repeat:    true,
	}

	f.mu.Lock()
	f.EventsSent = append(f.EventsSent, event)
	f.mu.Unlock()

	f.broadcastEvent(event)
}

// SetModifiers sets the modifier state directly.
func (f *FakeKeyboard) SetModifiers(mods input.Modifiers) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.modifiers = mods
}

// updateModifiers updates modifier state based on key press/release.
// Must be called with mu held.
func (f *FakeKeyboard) updateModifiers(key string, pressed bool) {
	switch key {
	case "LSHIFT", "RSHIFT":
		f.modifiers.Shift = pressed || f.pressedKeys["LSHIFT"] || f.pressedKeys["RSHIFT"]
	case "LCTRL", "RCTRL":
		f.modifiers.Ctrl = pressed || f.pressedKeys["LCTRL"] || f.pressedKeys["RCTRL"]
	case "LALT", "RALT":
		f.modifiers.Alt = pressed || f.pressedKeys["LALT"] || f.pressedKeys["RALT"]
	case "LMETA", "RMETA":
		f.modifiers.Meta = pressed || f.pressedKeys["LMETA"] || f.pressedKeys["RMETA"]
	case "CAPSLOCK":
		if pressed {
			f.modifiers.CapsLock = !f.modifiers.CapsLock
		}
	case "NUMLOCK":
		if pressed {
			f.modifiers.NumLock = !f.modifiers.NumLock
		}
	}
}

// Verify interface compliance
var _ input.Keyboard = (*FakeKeyboard)(nil)
