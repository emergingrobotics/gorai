package gateway

import "errors"

// Configuration errors.
var (
	ErrNoPortsConfigured = errors.New("gateway: no ports configured")
	ErrNoDevicePath      = errors.New("gateway: device path is required")
	ErrNoDeviceID        = errors.New("gateway: device_id is required")
	ErrNoSubjectPrefix   = errors.New("gateway: subject_prefix is required")
)

// Runtime errors.
var (
	ErrNotRunning    = errors.New("gateway: not running")
	ErrAlreadyRunning = errors.New("gateway: already running")
	ErrPortNotFound  = errors.New("gateway: port not found")
	ErrPortClosed    = errors.New("gateway: port closed")
)
