// Package serial provides serial port (UART) driver interfaces.
package serial

import (
	"context"
	"io"

	"github.com/emergingrobotics/gorai/driver"
)

// Port represents a serial port.
type Port interface {
	driver.Driver
	io.ReadWriter

	// SetBaudRate sets the baud rate.
	SetBaudRate(baud int) error

	// SetDataBits sets the data bits (5, 6, 7, or 8).
	SetDataBits(bits int) error

	// SetStopBits sets the stop bits.
	SetStopBits(bits StopBits) error

	// SetParity sets the parity mode.
	SetParity(parity Parity) error

	// Flush discards data in the input and output buffers.
	Flush(ctx context.Context) error

	// Available returns the number of bytes available to read.
	Available() (int, error)
}

// StopBits represents stop bit configuration.
type StopBits int

const (
	StopBits1 StopBits = iota
	StopBits15
	StopBits2
)

// Parity represents parity configuration.
type Parity int

const (
	ParityNone Parity = iota
	ParityOdd
	ParityEven
	ParityMark
	ParitySpace
)

// Config holds serial port configuration.
type Config struct {
	// Path is the device path (e.g., "/dev/ttyUSB0").
	Path string
	// BaudRate is the baud rate (e.g., 115200).
	BaudRate int
	// DataBits is the number of data bits (5, 6, 7, or 8).
	DataBits int
	// StopBits is the stop bit configuration.
	StopBits StopBits
	// Parity is the parity configuration.
	Parity Parity
	// ReadTimeout is the read timeout in milliseconds (0 for blocking).
	ReadTimeout int
}

// DefaultConfig returns a default serial configuration.
func DefaultConfig(path string) Config {
	return Config{
		Path:        path,
		BaudRate:    115200,
		DataBits:    8,
		StopBits:    StopBits1,
		Parity:      ParityNone,
		ReadTimeout: 0,
	}
}
