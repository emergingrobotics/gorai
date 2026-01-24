// Package power defines the power component interface.
//
// Power components represent energy storage and distribution devices such as
// batteries, power supplies, and power distribution units.
package power

import (
	"context"

	"github.com/gorai/gorai/components"
	"github.com/gorai/gorai/pkg/resource"
)

// Power represents a power source or energy storage component.
type Power interface {
	component.Component
	resource.Power
}

// Properties describes power component capabilities.
type Properties struct {
	// Type indicates the power source type (e.g., "battery", "power_supply", "pdu").
	Type string

	// NominalVoltage is the designed operating voltage.
	NominalVoltage float64

	// MaxCurrent is the maximum current output.
	MaxCurrent float64

	// CellCount is the number of cells (for batteries).
	CellCount int

	// Chemistry is the battery chemistry (e.g., "li-ion", "lifepo4", "lead-acid").
	Chemistry string

	// SupportsCharging indicates whether the power source can be charged.
	SupportsCharging bool

	// SupportsCellMonitoring indicates per-cell voltage monitoring.
	SupportsCellMonitoring bool
}

// Status represents the current health status of a power source.
type Status int

const (
	// StatusUnknown indicates the status cannot be determined.
	StatusUnknown Status = iota
	// StatusGood indicates normal operation.
	StatusGood
	// StatusLow indicates low charge level.
	StatusLow
	// StatusCritical indicates critically low charge level.
	StatusCritical
	// StatusCharging indicates the power source is charging.
	StatusCharging
	// StatusFull indicates the power source is fully charged.
	StatusFull
	// StatusFault indicates a fault condition.
	StatusFault
)

// String returns the string representation of a Status.
func (s Status) String() string {
	switch s {
	case StatusUnknown:
		return "unknown"
	case StatusGood:
		return "good"
	case StatusLow:
		return "low"
	case StatusCritical:
		return "critical"
	case StatusCharging:
		return "charging"
	case StatusFull:
		return "full"
	case StatusFault:
		return "fault"
	default:
		return "unknown"
	}
}

// Extended is an optional extended interface for power components with
// additional capabilities beyond the base Power interface.
type Extended interface {
	Power

	// GetStatus returns the overall power source status.
	GetStatus(ctx context.Context) (Status, error)

	// GetCellVoltages returns individual cell voltages (for multi-cell batteries).
	GetCellVoltages(ctx context.Context) ([]float64, error)

	// GetTemperature returns the temperature in Celsius.
	GetTemperature(ctx context.Context) (float64, error)

	// GetPower returns the current power consumption/production in watts.
	GetPower(ctx context.Context) (float64, error)

	// GetTimeRemaining returns estimated time remaining in seconds.
	// Returns -1 if unknown.
	GetTimeRemaining(ctx context.Context) (float64, error)

	// GetCycleCount returns the number of charge cycles (for batteries).
	GetCycleCount(ctx context.Context) (int, error)

	// GetProperties returns the power source properties.
	GetProperties(ctx context.Context) (Properties, error)
}
