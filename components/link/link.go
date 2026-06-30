// Package link defines the link component interface.
//
// Link components represent additional communication channels beyond the primary
// NATS connection. All Gorai components assume IP connectivity to a NATS server—
// that's the baseline infrastructure, not a "Link."
//
// A Link component represents an extra communication path, typically for:
//   - Microcontroller bridges: Serial connections to TinyGo devices without IP
//   - Telemetry channels: Radio links for remote monitoring or control
//   - Legacy protocols: CAN bus, RS-485, or other industrial networks
//   - Redundant paths: Backup communication for safety-critical systems
//
// Links can be bidirectional (serial, I2C) or broadcast (CAN).
package link

import (
	"context"

	"github.com/emergingrobotics/gorai/components"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Link represents an additional communication channel beyond NATS.
type Link interface {
	component.Component
	resource.Link
}

// Properties describes link component capabilities.
type Properties struct {
	// Type indicates the link transport type.
	Type resource.LinkType

	// Direction indicates bidirectional or broadcast.
	Direction resource.LinkDirection

	// Name is a human-readable name for the link.
	Name string

	// MaxBandwidth is the maximum bandwidth in bytes/sec (0 = unlimited).
	MaxBandwidth uint64

	// MaxMessageSize is the maximum message size in bytes.
	MaxMessageSize uint32

	// SupportsQoS indicates whether the link supports QoS levels.
	SupportsQoS bool

	// SupportsEncryption indicates whether the link supports encryption.
	SupportsEncryption bool

	// SupportsPersistence indicates whether messages can be persisted.
	SupportsPersistence bool
}

// Extended is an optional extended interface for link components with
// additional capabilities beyond the base Link interface.
type Extended interface {
	Link

	// GetProperties returns the link properties.
	GetProperties(ctx context.Context) (Properties, error)

	// Send sends data over the link.
	Send(ctx context.Context, data []byte) error

	// Receive receives data from the link (blocking).
	Receive(ctx context.Context) ([]byte, error)

	// SetTimeout sets the read/write timeout.
	SetTimeout(ctx context.Context, timeoutMS int) error

	// Flush ensures all buffered data is sent.
	Flush(ctx context.Context) error

	// Reset resets the link connection.
	Reset(ctx context.Context) error
}

// SerialLink is a specialized link interface for serial connections.
// Common use: bridging NATS to microcontrollers without IP capability.
type SerialLink interface {
	Link

	// GetBaudRate returns the current baud rate.
	GetBaudRate(ctx context.Context) (int, error)

	// SetBaudRate sets the baud rate.
	SetBaudRate(ctx context.Context, baud int) error

	// GetDataBits returns the number of data bits.
	GetDataBits(ctx context.Context) (int, error)

	// SetDataBits sets the number of data bits.
	SetDataBits(ctx context.Context, bits int) error

	// GetStopBits returns the number of stop bits.
	GetStopBits(ctx context.Context) (float64, error)

	// SetStopBits sets the number of stop bits.
	SetStopBits(ctx context.Context, bits float64) error

	// GetParity returns the parity mode.
	GetParity(ctx context.Context) (string, error)

	// SetParity sets the parity mode.
	SetParity(ctx context.Context, parity string) error

	// Available returns the number of bytes available to read.
	Available(ctx context.Context) (int, error)
}

// RadioLink is a specialized link interface for RF/wireless connections.
// Common use: telemetry when out of WiFi range, long-range robot control.
type RadioLink interface {
	Link

	// GetFrequency returns the operating frequency in Hz.
	GetFrequency(ctx context.Context) (float64, error)

	// SetFrequency sets the operating frequency in Hz.
	SetFrequency(ctx context.Context, freq float64) error

	// GetTxPower returns the transmit power in dBm.
	GetTxPower(ctx context.Context) (int, error)

	// SetTxPower sets the transmit power in dBm.
	SetTxPower(ctx context.Context, power int) error

	// GetRSSI returns the received signal strength indicator.
	GetRSSI(ctx context.Context) (int, error)

	// GetSNR returns the signal-to-noise ratio.
	GetSNR(ctx context.Context) (float64, error)
}
