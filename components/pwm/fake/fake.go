// Package fake provides a fake PWM implementation for testing.
package fake

import (
	"context"
	"sync"

	"github.com/gorai/gorai/components/pwm"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("pwm", "fake", New)
}

// PWM is a fake PWM implementation for testing.
type PWM struct {
	name   resource.Name
	config Config

	mu      sync.RWMutex
	pulseUs float64
	enabled bool

	// Call tracking for test assertions
	SetPulseCalls      []float64
	SetNormalizedCalls []float64
	SetDutyCalls       []float64
	EnableCalls        int
	DisableCalls       int
}

// Config holds configuration for the fake PWM.
type Config struct {
	FrequencyHz float64
	MinPulseUs  float64
	MaxPulseUs  float64
	Pin         int
	Chip        string
}

// DefaultConfig returns default configuration for testing.
func DefaultConfig() Config {
	return Config{
		FrequencyHz: 50.0,
		MinPulseUs:  1000.0,
		MaxPulseUs:  2000.0,
		Pin:         18,
		Chip:        "/dev/gpiochip4",
	}
}

// New creates a new fake PWM component.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "pwm", nameStr)

	cfg := DefaultConfig()

	if chip, ok := conf["chip"].(string); ok {
		cfg.Chip = chip
	}
	if pin, ok := conf["pin"].(float64); ok {
		cfg.Pin = int(pin)
	}
	if freq, ok := conf["frequency_hz"].(float64); ok {
		cfg.FrequencyHz = freq
	}
	if minPulse, ok := conf["min_pulse_us"].(float64); ok {
		cfg.MinPulseUs = minPulse
	}
	if maxPulse, ok := conf["max_pulse_us"].(float64); ok {
		cfg.MaxPulseUs = maxPulse
	}

	initialPulse := (cfg.MinPulseUs + cfg.MaxPulseUs) / 2
	if initial, ok := conf["initial_pulse_us"].(float64); ok {
		initialPulse = initial
	}

	return &PWM{
		name:    name,
		config:  cfg,
		pulseUs: initialPulse,
	}, nil
}

// NewWithName creates a fake PWM with a specific resource name for testing.
func NewWithName(name resource.Name) *PWM {
	cfg := DefaultConfig()
	return &PWM{
		name:    name,
		config:  cfg,
		pulseUs: (cfg.MinPulseUs + cfg.MaxPulseUs) / 2,
	}
}

// NewWithConfig creates a fake PWM with specific configuration for testing.
func NewWithConfig(name resource.Name, cfg Config) *PWM {
	return &PWM{
		name:    name,
		config:  cfg,
		pulseUs: (cfg.MinPulseUs + cfg.MaxPulseUs) / 2,
	}
}

// Name returns the resource name.
func (p *PWM) Name() resource.Name {
	return p.name
}

// Reconfigure updates the configuration.
func (p *PWM) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (p *PWM) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)
	switch cmdName {
	case "set_pulse":
		pulseUs, _ := cmd["pulse_us"].(float64)
		p.SetPulse(ctx, pulseUs)
		return map[string]any{"status": "ok"}, nil

	case "set_normalized":
		value, _ := cmd["value"].(float64)
		p.SetNormalized(ctx, value)
		return map[string]any{"status": "ok"}, nil

	case "set_duty":
		duty, _ := cmd["duty"].(float64)
		p.SetDuty(ctx, duty)
		return map[string]any{"status": "ok"}, nil

	case "enable":
		p.Enable(ctx)
		return map[string]any{"status": "ok"}, nil

	case "disable":
		p.Disable(ctx)
		return map[string]any{"status": "ok"}, nil

	case "get_state":
		p.mu.RLock()
		defer p.mu.RUnlock()
		return map[string]any{
			"pulse_us":     p.pulseUs,
			"enabled":      p.enabled,
			"frequency_hz": p.config.FrequencyHz,
		}, nil

	default:
		return nil, nil
	}
}

// Close releases resources.
func (p *PWM) Close(ctx context.Context) error {
	return nil
}

// SetPulse sets the pulse width in microseconds.
func (p *PWM) SetPulse(ctx context.Context, pulseUs float64) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Clamp to limits
	if pulseUs < p.config.MinPulseUs {
		pulseUs = p.config.MinPulseUs
	}
	if pulseUs > p.config.MaxPulseUs {
		pulseUs = p.config.MaxPulseUs
	}

	p.pulseUs = pulseUs
	p.SetPulseCalls = append(p.SetPulseCalls, pulseUs)
	return nil
}

// SetNormalized sets the output using a normalized value from -1.0 to 1.0.
func (p *PWM) SetNormalized(ctx context.Context, value float64) error {
	p.mu.Lock()
	p.SetNormalizedCalls = append(p.SetNormalizedCalls, value)
	p.mu.Unlock()

	// Clamp
	if value < -1.0 {
		value = -1.0
	}
	if value > 1.0 {
		value = 1.0
	}

	center := (p.config.MinPulseUs + p.config.MaxPulseUs) / 2
	rangeUs := (p.config.MaxPulseUs - p.config.MinPulseUs) / 2
	pulseUs := center + (value * rangeUs)

	return p.SetPulse(ctx, pulseUs)
}

// SetDuty sets the duty cycle as a fraction from 0.0 to 1.0.
func (p *PWM) SetDuty(ctx context.Context, duty float64) error {
	p.mu.Lock()
	p.SetDutyCalls = append(p.SetDutyCalls, duty)
	p.mu.Unlock()

	// Clamp
	if duty < 0.0 {
		duty = 0.0
	}
	if duty > 1.0 {
		duty = 1.0
	}

	periodUs := 1_000_000.0 / p.config.FrequencyHz
	pulseUs := duty * periodUs

	return p.SetPulse(ctx, pulseUs)
}

// Enable starts PWM signal generation.
func (p *PWM) Enable(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = true
	p.EnableCalls++
	return nil
}

// Disable stops PWM signal generation.
func (p *PWM) Disable(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.enabled = false
	p.DisableCalls++
	return nil
}

// IsEnabled returns true if PWM is currently enabled.
func (p *PWM) IsEnabled(ctx context.Context) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.enabled, nil
}

// GetPulse returns the current pulse width in microseconds.
func (p *PWM) GetPulse(ctx context.Context) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.pulseUs, nil
}

// GetNormalized returns the current value as normalized (-1.0 to 1.0).
func (p *PWM) GetNormalized(ctx context.Context) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	center := (p.config.MinPulseUs + p.config.MaxPulseUs) / 2
	rangeUs := (p.config.MaxPulseUs - p.config.MinPulseUs) / 2
	if rangeUs == 0 {
		return 0, nil
	}
	return (p.pulseUs - center) / rangeUs, nil
}

// Properties returns the PWM configuration and capabilities.
func (p *PWM) Properties(ctx context.Context) (pwm.Properties, error) {
	return pwm.Properties{
		FrequencyHz: p.config.FrequencyHz,
		MinPulseUs:  p.config.MinPulseUs,
		MaxPulseUs:  p.config.MaxPulseUs,
		Pin:         p.config.Pin,
		Chip:        p.config.Chip,
	}, nil
}

// Reset clears call tracking for tests.
func (p *PWM) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.SetPulseCalls = nil
	p.SetNormalizedCalls = nil
	p.SetDutyCalls = nil
	p.EnableCalls = 0
	p.DisableCalls = 0
}

// Verify interface compliance at compile time.
var _ pwm.PWM = (*PWM)(nil)

