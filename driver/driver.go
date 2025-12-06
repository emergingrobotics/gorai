// Package driver provides hardware driver interfaces for Gorai.
//
// Drivers provide low-level access to hardware interfaces like GPIO, I2C,
// SPI, and serial ports. The core repository only contains pure Go drivers;
// drivers requiring CGo should be in separate satellite repositories.
package driver

import "context"

// Driver is the base interface for hardware drivers.
type Driver interface {
	// Name returns the driver name.
	Name() string

	// Open initializes the driver.
	Open(ctx context.Context) error

	// Close releases driver resources.
	Close(ctx context.Context) error
}

// Pin represents a single I/O pin.
type Pin interface {
	// Number returns the pin number.
	Number() int

	// Name returns the pin name.
	Name() string
}
