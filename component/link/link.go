// Package link defines the link component interface.
//
// Link components represent communication channels between nodes, which can be
// either bidirectional (point-to-point) or broadcast (one-to-many). Supports
// various transport mechanisms including serial, IP, NATS, CAN, I2C, and SPI.
package link

import (
	"context"

	"github.com/gorai/gorai/component"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

// Link represents a communication link between nodes.
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

// NATSLink is a specialized link interface for NATS connections.
// NATS is a special kind of IP-based link that provides broadcast semantics
// via pub/sub, optional persistence via JetStream, and built-in clustering.
type NATSLink interface {
	Link

	// GetConnection returns the underlying NATS connection.
	GetConnection() *nats.Conn

	// GetSubject returns the primary subject for this link.
	GetSubject() string

	// Publish publishes a message to the subject.
	Publish(ctx context.Context, data []byte) error

	// Subscribe subscribes to messages on the subject.
	Subscribe(ctx context.Context, handler func([]byte)) error

	// Unsubscribe removes the subscription.
	Unsubscribe(ctx context.Context) error

	// Request performs a request-reply operation.
	Request(ctx context.Context, data []byte) ([]byte, error)
}

// SerialLink is a specialized link interface for serial connections.
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

// IPLink is a specialized link interface for IP-based connections.
type IPLink interface {
	Link

	// GetRemoteAddress returns the remote address.
	GetRemoteAddress(ctx context.Context) (string, error)

	// GetLocalAddress returns the local address.
	GetLocalAddress(ctx context.Context) (string, error)

	// GetPort returns the port number.
	GetPort(ctx context.Context) (int, error)

	// GetProtocol returns the protocol (tcp, udp).
	GetProtocol(ctx context.Context) (string, error)
}
