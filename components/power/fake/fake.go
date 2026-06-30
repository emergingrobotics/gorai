// Package fake provides a fake power implementation for testing.
package fake

import (
	"context"
	"fmt"
	"sync"

	"github.com/emergingrobotics/gorai/components/power"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("power", "fake", New)
}

// Power is a fake power source for testing.
type Power struct {
	name        resource.Name
	mu          sync.RWMutex
	capacity    float64 // Wh
	level       float64 // 0.0 - 1.0
	voltage     float64 // Volts
	current     float64 // Amps (positive = discharging)
	charging    bool
	status      power.Status
	temperature float64
	cellCount   int
	cells       []float64
	cycleCount  int
}

// New creates a new fake power source.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "power", nameStr)

	// Default to a typical 12V, 100Wh battery
	capacity := 100.0
	if c, ok := conf["capacity"].(float64); ok {
		capacity = c
	}

	voltage := 12.0
	if v, ok := conf["voltage"].(float64); ok {
		voltage = v
	}

	cellCount := 4
	if c, ok := conf["cell_count"].(float64); ok {
		cellCount = int(c)
	}

	p := &Power{
		name:        name,
		capacity:    capacity,
		level:       1.0, // Start fully charged
		voltage:     voltage,
		current:     0,
		charging:    false,
		status:      power.StatusFull,
		temperature: 25.0,
		cellCount:   cellCount,
		cells:       make([]float64, cellCount),
		cycleCount:  0,
	}

	// Initialize cell voltages
	cellVoltage := voltage / float64(cellCount)
	for i := range p.cells {
		p.cells[i] = cellVoltage
	}

	return p, nil
}

// NewWithName creates a fake power source with a specific resource name.
func NewWithName(name resource.Name) *Power {
	p := &Power{
		name:        name,
		capacity:    100.0,
		level:       1.0,
		voltage:     12.0,
		current:     0,
		charging:    false,
		status:      power.StatusFull,
		temperature: 25.0,
		cellCount:   4,
		cells:       []float64{3.0, 3.0, 3.0, 3.0},
		cycleCount:  0,
	}
	return p
}

// Name returns the power source's resource name.
func (p *Power) Name() resource.Name {
	return p.name
}

// Reconfigure updates the power source configuration.
func (p *Power) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if cap, ok := conf.GetFloat("capacity"); ok {
		p.capacity = cap
	}
	if voltage, ok := conf.GetFloat("voltage"); ok {
		p.voltage = voltage
	}
	return nil
}

// DoCommand executes arbitrary commands for extensibility.
func (p *Power) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "set_level":
			if level, ok := cmd["level"].(float64); ok {
				p.SetLevel(level)
				return map[string]any{"status": "ok"}, nil
			}
			return nil, fmt.Errorf("invalid level value")
		case "set_charging":
			if charging, ok := cmd["charging"].(bool); ok {
				p.SetCharging(charging)
				return map[string]any{"status": "ok"}, nil
			}
			return nil, fmt.Errorf("invalid charging value")
		case "get_state":
			p.mu.RLock()
			defer p.mu.RUnlock()
			return map[string]any{
				"capacity":    p.capacity,
				"level":       p.level,
				"voltage":     p.voltage,
				"current":     p.current,
				"charging":    p.charging,
				"status":      p.status.String(),
				"temperature": p.temperature,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (p *Power) Close(ctx context.Context) error {
	return nil
}

// GetCapacity returns the total capacity in watt-hours.
func (p *Power) GetCapacity(ctx context.Context) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.capacity, nil
}

// GetLevel returns the current charge level as a ratio (0.0 - 1.0).
func (p *Power) GetLevel(ctx context.Context) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.level, nil
}

// GetVoltage returns the current voltage.
func (p *Power) GetVoltage(ctx context.Context) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.voltage, nil
}

// GetCurrent returns the current draw in amps.
func (p *Power) GetCurrent(ctx context.Context) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.current, nil
}

// IsCharging returns whether the power source is charging.
func (p *Power) IsCharging(ctx context.Context) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.charging, nil
}

// Extended interface methods

// GetStatus returns the overall power source status.
func (p *Power) GetStatus(ctx context.Context) (power.Status, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.status, nil
}

// GetCellVoltages returns individual cell voltages.
func (p *Power) GetCellVoltages(ctx context.Context) ([]float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	result := make([]float64, len(p.cells))
	copy(result, p.cells)
	return result, nil
}

// GetTemperature returns the temperature in Celsius.
func (p *Power) GetTemperature(ctx context.Context) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.temperature, nil
}

// GetPower returns the current power in watts.
func (p *Power) GetPower(ctx context.Context) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.voltage * p.current, nil
}

// GetTimeRemaining returns estimated time remaining in seconds.
func (p *Power) GetTimeRemaining(ctx context.Context) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.current <= 0 {
		return -1, nil // Not discharging
	}

	// Calculate remaining energy in Wh
	remainingEnergy := p.capacity * p.level
	// Calculate power being drawn
	powerDraw := p.voltage * p.current
	if powerDraw <= 0 {
		return -1, nil
	}

	// Time in hours, convert to seconds
	return (remainingEnergy / powerDraw) * 3600, nil
}

// GetCycleCount returns the number of charge cycles.
func (p *Power) GetCycleCount(ctx context.Context) (int, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.cycleCount, nil
}

// GetProperties returns the power source properties.
func (p *Power) GetProperties(ctx context.Context) (power.Properties, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return power.Properties{
		Type:                   "battery",
		NominalVoltage:         p.voltage,
		MaxCurrent:             10.0,
		CellCount:              p.cellCount,
		Chemistry:              "li-ion",
		SupportsCharging:       true,
		SupportsCellMonitoring: true,
	}, nil
}

// Test helper methods

// SetLevel sets the charge level (for testing).
func (p *Power) SetLevel(level float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.level = level
	p.updateStatus()
}

// SetCharging sets the charging state (for testing).
func (p *Power) SetCharging(charging bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.charging = charging
	if charging {
		p.current = -2.0 // Negative current = charging
	} else {
		p.current = 0
	}
	p.updateStatus()
}

// SetCurrent sets the current draw (for testing).
func (p *Power) SetCurrent(current float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.current = current
}

// SetTemperature sets the temperature (for testing).
func (p *Power) SetTemperature(temp float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.temperature = temp
}

// IncrementCycles increments the cycle count (for testing).
func (p *Power) IncrementCycles() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cycleCount++
}

// updateStatus updates the status based on current state.
func (p *Power) updateStatus() {
	if p.charging {
		if p.level >= 1.0 {
			p.status = power.StatusFull
		} else {
			p.status = power.StatusCharging
		}
	} else if p.level >= 0.9 {
		p.status = power.StatusFull
	} else if p.level >= 0.2 {
		p.status = power.StatusGood
	} else if p.level >= 0.1 {
		p.status = power.StatusLow
	} else {
		p.status = power.StatusCritical
	}
}

// Verify interface compliance.
var _ power.Power = (*Power)(nil)
var _ power.Extended = (*Power)(nil)
