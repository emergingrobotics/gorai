// Package reader provides platform-specific temperature reading capabilities.
package reader

import (
	"context"
	"fmt"
	"runtime"
)

// Reading represents a temperature reading from a thermal zone.
type Reading struct {
	Zone         string
	TemperatureC float64
	CriticalC    float64 // 0 if unknown
	WarningC     float64 // 0 if unknown
}

// Reader reads temperature from the host system.
type Reader interface {
	// Platform returns the platform name (e.g., "linux", "darwin").
	Platform() string

	// Zones returns available thermal zones.
	Zones(ctx context.Context) ([]string, error)

	// Read returns temperature for a specific zone.
	// Use "" or "default" for the primary zone.
	Read(ctx context.Context, zone string) (Reading, error)

	// Close releases resources.
	Close() error
}

// New creates a Reader for the current platform.
func New() (Reader, error) {
	switch runtime.GOOS {
	case "linux":
		return newLinuxReader()
	case "darwin":
		return newDarwinReader()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// CelsiusToFahrenheit converts Celsius to Fahrenheit.
func CelsiusToFahrenheit(celsius float64) float64 {
	return celsius*9/5 + 32
}
