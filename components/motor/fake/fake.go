// Package fake provides a fake motor implementation for testing.
package fake

import (
	"context"
	"fmt"
	"sync"

	"github.com/emergingrobotics/gorai/components/motor"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("motor", "fake", New)
}

// Motor is a fake motor for testing.
type Motor struct {
	name     resource.Name
	mu       sync.RWMutex
	power    float64
	velocity float64
	position float64
	moving   bool
}

// New creates a new fake motor.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "motor", nameStr)
	return &Motor{name: name}, nil
}

// NewWithName creates a fake motor with a specific resource name.
func NewWithName(name resource.Name) *Motor {
	return &Motor{name: name}
}

// Name returns the motor's resource name.
func (m *Motor) Name() resource.Name {
	return m.name
}

// Reconfigure updates the motor configuration.
func (m *Motor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	// Fake motor has no configuration to update
	return nil
}

// DoCommand executes arbitrary commands for extensibility.
func (m *Motor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "set_position":
			if pos, ok := cmd["position"].(float64); ok {
				m.mu.Lock()
				m.position = pos
				m.mu.Unlock()
				return map[string]any{"status": "ok"}, nil
			}
			return nil, fmt.Errorf("invalid position value")
		case "get_state":
			m.mu.RLock()
			defer m.mu.RUnlock()
			return map[string]any{
				"power":    m.power,
				"velocity": m.velocity,
				"position": m.position,
				"moving":   m.moving,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (m *Motor) Close(ctx context.Context) error {
	return m.Stop(ctx)
}

// SetPower sets the motor power.
func (m *Motor) SetPower(ctx context.Context, power float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.power = power
	m.moving = power != 0
	return nil
}

// SetVelocity sets the target velocity.
func (m *Motor) SetVelocity(ctx context.Context, velocity float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.velocity = velocity
	m.moving = velocity != 0
	return nil
}

// GoFor moves the motor for the specified revolutions.
func (m *Motor) GoFor(ctx context.Context, rpm, revolutions float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.position += revolutions
	return nil
}

// GoTo moves the motor to the specified position.
func (m *Motor) GoTo(ctx context.Context, position, velocity float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.position = position
	m.velocity = velocity
	return nil
}

// ResetZeroPosition resets the zero position.
func (m *Motor) ResetZeroPosition(ctx context.Context, offset float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.position = offset
	return nil
}

// GetPosition returns the current position.
func (m *Motor) GetPosition(ctx context.Context) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.position, nil
}

// GetVelocity returns the current velocity.
func (m *Motor) GetVelocity(ctx context.Context) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.velocity, nil
}

// Properties returns the motor properties.
func (m *Motor) Properties(ctx context.Context) (motor.Properties, error) {
	return motor.Properties{
		PositionReporting: true,
		VelocityReporting: true,
		SupportsGoTo:      true,
	}, nil
}

// IsPowered returns the power state.
func (m *Motor) IsPowered(ctx context.Context) (bool, float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.power != 0, m.power, nil
}

// IsMoving returns whether the motor is moving.
func (m *Motor) IsMoving(ctx context.Context) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.moving, nil
}

// Stop stops the motor.
func (m *Motor) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.power = 0
	m.velocity = 0
	m.moving = false
	return nil
}

// Verify interface compliance.
var _ motor.Motor = (*Motor)(nil)
