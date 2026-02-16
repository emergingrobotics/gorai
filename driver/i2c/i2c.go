// Package i2c provides I2C (Inter-Integrated Circuit) driver interfaces.
package i2c

import (
	"context"

	"github.com/gorai/gorai/driver"
)

// Bus represents an I2C bus.
type Bus interface {
	driver.Driver

	// Device returns an I2C device at the given address.
	Device(addr uint16) Device
}

// Device represents an I2C device.
type Device interface {
	// Address returns the device address.
	Address() uint16

	// Read reads n bytes from the device.
	Read(ctx context.Context, n int) ([]byte, error)

	// Write writes data to the device.
	Write(ctx context.Context, data []byte) error

	// WriteRead writes data then reads n bytes.
	WriteRead(ctx context.Context, write []byte, readLen int) ([]byte, error)

	// ReadReg reads n bytes from a register.
	ReadReg(ctx context.Context, reg byte, n int) ([]byte, error)

	// WriteReg writes data to a register.
	WriteReg(ctx context.Context, reg byte, data []byte) error

	// ReadByteCtx reads a single byte.
	ReadByteCtx(ctx context.Context) (byte, error)

	// WriteByteCtx writes a single byte.
	WriteByteCtx(ctx context.Context, b byte) error

	// ReadByteReg reads a single byte from a register.
	ReadByteReg(ctx context.Context, reg byte) (byte, error)

	// WriteByteReg writes a single byte to a register.
	WriteByteReg(ctx context.Context, reg, value byte) error
}
