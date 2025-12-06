// Package fake provides a fake motor implementation for testing.
package fake

import (
	"context"
	"sync"

	"github.com/gorai/gorai/component/motor"
	"github.com/gorai/gorai/pkg/registry"
)

func init() {
	registry.RegisterComponent("motor", "fake", New)
}

// Motor is a fake motor for testing.
type Motor struct {
	name     string
	mu       sync.RWMutex
	power    float64
	position float64
	moving   bool
}

// New creates a new fake motor.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	name, _ := conf["name"].(string)
	return &Motor{name: name}, nil
}

// Name returns the motor's name.
func (m *Motor) Name() string {
	return m.name
}

// Reconfigure updates the motor configuration.
func (m *Motor) Reconfigure(ctx context.Context, conf map[string]any) error {
	return nil
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

// GoFor moves the motor for the specified revolutions.
func (m *Motor) GoFor(ctx context.Context, rpm, revolutions float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.position += revolutions
	return nil
}

// GoTo moves the motor to the specified position.
func (m *Motor) GoTo(ctx context.Context, rpm, position float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.position = position
	return nil
}

// ResetZeroPosition resets the zero position.
func (m *Motor) ResetZeroPosition(ctx context.Context, offset float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.position = offset
	return nil
}

// Position returns the current position.
func (m *Motor) Position(ctx context.Context) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.position, nil
}

// Properties returns the motor properties.
func (m *Motor) Properties(ctx context.Context) (motor.Properties, error) {
	return motor.Properties{PositionReporting: true}, nil
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
	m.moving = false
	return nil
}

// Verify interface compliance.
var _ motor.Motor = (*Motor)(nil)
