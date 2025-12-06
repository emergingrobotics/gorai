// Package gpio provides GPIO (General Purpose Input/Output) driver interfaces.
package gpio

import (
	"context"
	"time"

	"github.com/gorai/gorai/driver"
)

// Driver provides GPIO access.
type Driver interface {
	driver.Driver

	// Pin returns a GPIO pin by number.
	Pin(number int) (Pin, error)

	// PinByName returns a GPIO pin by name.
	PinByName(name string) (Pin, error)
}

// Pin represents a GPIO pin.
type Pin interface {
	driver.Pin

	// SetDirection sets the pin direction.
	SetDirection(ctx context.Context, dir Direction) error

	// Direction returns the current pin direction.
	Direction() Direction

	// Read returns the current pin state.
	Read(ctx context.Context) (bool, error)

	// Write sets the pin state.
	Write(ctx context.Context, high bool) error

	// PWM returns the PWM interface for this pin, if supported.
	PWM() (PWM, error)
}

// Direction represents pin direction.
type Direction int

const (
	Input Direction = iota
	Output
)

// PWM provides PWM control for a pin.
type PWM interface {
	// SetDuty sets the PWM duty cycle (0.0 to 1.0).
	SetDuty(ctx context.Context, duty float64) error

	// SetFrequency sets the PWM frequency in Hz.
	SetFrequency(ctx context.Context, freq float64) error

	// Duty returns the current duty cycle.
	Duty() float64

	// Frequency returns the current frequency.
	Frequency() float64
}

// Edge represents edge detection types.
type Edge int

const (
	EdgeNone Edge = iota
	EdgeRising
	EdgeFalling
	EdgeBoth
)

// WaitForEdge waits for an edge on the pin.
type EdgeWaiter interface {
	// WaitForEdge waits for the specified edge, with timeout.
	WaitForEdge(ctx context.Context, edge Edge, timeout time.Duration) (bool, error)
}
