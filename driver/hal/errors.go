// Package hal provides the Hardware Abstraction Layer for Gorai.
package hal

import "errors"

var (
	// ErrHALClosed is returned when operations are attempted on a closed HAL.
	ErrHALClosed = errors.New("HAL is closed")

	// ErrPinNotFound is returned when a pin reference cannot be resolved.
	ErrPinNotFound = errors.New("pin not found")

	// ErrBusNotAvailable is returned when a requested bus doesn't exist.
	ErrBusNotAvailable = errors.New("bus not available")

	// ErrChipNotAvailable is returned when a requested chip doesn't exist.
	ErrChipNotAvailable = errors.New("chip not available")

	// ErrFeatureNotSupported is returned for unsupported features.
	ErrFeatureNotSupported = errors.New("feature not supported on this board")

	// ErrUnsupportedBoard is returned when the board is not supported.
	ErrUnsupportedBoard = errors.New("unsupported board")

	// ErrInvalidPinRef is returned for invalid pin references.
	ErrInvalidPinRef = errors.New("invalid pin reference")
)
