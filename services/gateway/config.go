// Package gateway provides a GSP-NATS bridge service for serial devices.
package gateway

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// Config holds gateway configuration.
type Config struct {
	// Ports is the list of serial ports to manage.
	Ports []PortConfig `json:"ports"`

	// PingInterval is how often to send PING to devices.
	// Default: 1 second.
	PingInterval time.Duration `json:"ping_interval"`

	// PingTimeout is how long to wait for PONG response.
	// Default: 3 seconds.
	PingTimeout time.Duration `json:"ping_timeout"`

	// ReconnectDelay is how long to wait before reconnecting after disconnect.
	// Default: 5 seconds.
	ReconnectDelay time.Duration `json:"reconnect_delay"`
}

// PortConfig configures a single serial port.
type PortConfig struct {
	// Device is the serial port path (e.g., "/dev/ttyACM0").
	Device string `json:"device"`

	// BaudRate is the serial baud rate. Default: 115200.
	BaudRate int `json:"baud_rate"`

	// DataBits is the number of data bits. Default: 8.
	DataBits int `json:"data_bits"`

	// StopBits is the number of stop bits. Default: 1.
	StopBits int `json:"stop_bits"`

	// Parity is the parity mode: "none", "odd", "even". Default: "none".
	Parity string `json:"parity"`

	// DeviceID is the unique identifier for this device.
	// Used in NATS subject construction.
	DeviceID string `json:"device_id"`

	// SubjectPrefix is the NATS subject prefix (e.g., "gorai.robot1").
	SubjectPrefix string `json:"subject_prefix"`
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() *Config {
	return &Config{
		PingInterval:   time.Second,
		PingTimeout:    3 * time.Second,
		ReconnectDelay: 5 * time.Second,
	}
}

// DefaultPortConfig returns a PortConfig with default values.
func DefaultPortConfig() PortConfig {
	return PortConfig{
		BaudRate: 115200,
		DataBits: 8,
		StopBits: 1,
		Parity:   "none",
	}
}

// Validate checks the configuration for errors.
func (c *Config) Validate() error {
	if len(c.Ports) == 0 {
		return ErrNoPortsConfigured
	}
	for _, p := range c.Ports {
		if err := p.Validate(); err != nil {
			return err
		}
	}
	return nil
}

// validDevicePathPrefixes are the allowed prefixes for serial device paths.
var validDevicePathPrefixes = []string{
	"/dev/tty",
	"/dev/serial/",
	"/dev/cu.",
}

// validateDevicePath checks that a device path is a valid serial device.
func validateDevicePath(path string) error {
	// Canonicalize to prevent path traversal (e.g. /dev/tty/../sda1)
	cleaned := filepath.Clean(path)
	for _, prefix := range validDevicePathPrefixes {
		if strings.HasPrefix(cleaned, prefix) {
			return nil
		}
	}
	return fmt.Errorf("invalid device path %q: must start with /dev/tty, /dev/serial/, or /dev/cu.", path)
}

// Validate checks the port configuration for errors.
func (p *PortConfig) Validate() error {
	if p.Device == "" {
		return ErrNoDevicePath
	}
	if err := validateDevicePath(p.Device); err != nil {
		return err
	}
	if p.DeviceID == "" {
		return ErrNoDeviceID
	}
	if p.SubjectPrefix == "" {
		return ErrNoSubjectPrefix
	}
	return nil
}
