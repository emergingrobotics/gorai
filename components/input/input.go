// Package input provides interfaces for input devices.
package input

import (
	"context"

	"github.com/gorai/gorai/components"
)

// Modifiers represents the state of modifier keys.
type Modifiers struct {
	Shift    bool
	Ctrl     bool
	Alt      bool
	Meta     bool
	CapsLock bool
	NumLock  bool
}

// KeyEvent represents a keyboard key press or release event.
type KeyEvent struct {
	Key       string    // Human-readable key name (e.g., "W", "SPACE", "UP")
	Code      uint16    // Linux key code
	Pressed   bool      // True if pressed, false if released
	Modifiers Modifiers // Modifier state at time of event
	Repeat    bool      // True if this is a repeat event
}

// Keyboard is an input component that captures keyboard events.
type Keyboard interface {
	component.Component

	// Events returns a channel of key events.
	// The channel is closed when the context is cancelled or the keyboard is closed.
	Events(ctx context.Context) (<-chan KeyEvent, error)

	// IsPressed returns true if the specified key is currently pressed.
	// The key parameter is the human-readable key name (e.g., "W", "SPACE").
	IsPressed(ctx context.Context, key string) (bool, error)

	// GetPressedKeys returns a list of currently pressed key names.
	GetPressedKeys(ctx context.Context) ([]string, error)

	// GetModifiers returns the current state of modifier keys.
	GetModifiers(ctx context.Context) (Modifiers, error)
}

