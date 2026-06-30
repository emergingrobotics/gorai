// Package gpio defines the GPIO component interface for digital input/output.
//
// GPIO components provide digital pin control on remote devices such as
// microcontrollers connected via GSP/2. They support:
//   - Output mode: set pin high, low, or toggle
//   - Input mode: read current pin state, subscribe to changes
//
// Pin configuration (mode, pull resistor) is handled automatically on Start()
// when the component provisions the remote device.
package gpio

import (
	"context"

	component "github.com/emergingrobotics/gorai/components"
)

// GPIO represents a digital GPIO pin component.
type GPIO interface {
	component.Component

	// Set sets the output pin value. Only valid for output-mode pins.
	// Values: 0 = low, 1 = high, 2 = toggle.
	Set(ctx context.Context, value uint8) error

	// Get returns the current pin value (0 or 1).
	Get(ctx context.Context) (uint8, error)

	// IsConfigured returns true if the pin has been provisioned on the device.
	IsConfigured(ctx context.Context) (bool, error)

	// Properties returns the GPIO configuration.
	Properties(ctx context.Context) (Properties, error)
}

// Properties describes GPIO configuration and state.
type Properties struct {
	// Pin is the GPIO pin number on the device.
	Pin int

	// Mode is the pin mode: "input" or "output".
	Mode string

	// Pull is the pull resistor setting: "none", "up", or "down".
	Pull string

	// DeviceID is the remote device identifier.
	DeviceID string
}
