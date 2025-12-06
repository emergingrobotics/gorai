// Package spi provides SPI (Serial Peripheral Interface) driver interfaces.
package spi

import (
	"context"

	"github.com/gorai/gorai/driver"
)

// Bus represents an SPI bus.
type Bus interface {
	driver.Driver

	// Device returns an SPI device with the given chip select.
	Device(cs int) Device
}

// Device represents an SPI device.
type Device interface {
	// ChipSelect returns the chip select number.
	ChipSelect() int

	// Transfer performs a simultaneous write and read.
	Transfer(ctx context.Context, write []byte) ([]byte, error)

	// Write writes data to the device.
	Write(ctx context.Context, data []byte) error

	// Read reads n bytes from the device.
	Read(ctx context.Context, n int) ([]byte, error)

	// SetSpeed sets the SPI clock speed in Hz.
	SetSpeed(speed int) error

	// SetMode sets the SPI mode (0-3).
	SetMode(mode Mode) error

	// SetBitsPerWord sets the word size.
	SetBitsPerWord(bits int) error
}

// Mode represents SPI mode (clock polarity and phase).
type Mode int

const (
	Mode0 Mode = iota // CPOL=0, CPHA=0
	Mode1             // CPOL=0, CPHA=1
	Mode2             // CPOL=1, CPHA=0
	Mode3             // CPOL=1, CPHA=1
)
