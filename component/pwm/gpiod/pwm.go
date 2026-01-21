// Package gpiod provides a software PWM implementation using libgpiod.
//
// This package implements the pwm.PWM interface for Raspberry Pi 5 and other
// Linux systems with GPIO support via the gpiod character device interface.
//
// Software PWM is generated in a dedicated goroutine with timing achieved
// through a combination of time.Sleep and busy-waiting for precision.
package gpiod

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorai/gorai/component/pwm"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/warthog618/go-gpiocdev"
)

func init() {
	registry.RegisterComponent("pwm", "gpiod", New)
}

// PWM implements software PWM using libgpiod.
type PWM struct {
	name   resource.Name
	config Config
	logger *slog.Logger

	chip *gpiocdev.Chip
	line *gpiocdev.Line

	mu       sync.RWMutex
	pulseUs  float64
	enabled  bool
	periodNs int64

	stopCh chan struct{}
	doneCh chan struct{}

	// Metrics
	cmdCount   atomic.Uint64
	cycleCount atomic.Uint64
}

// New creates a new gpiod PWM component.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "pwm", nameStr)

	cfg, err := ParseConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	p := &PWM{
		name:     name,
		config:   cfg,
		logger:   slog.Default().With("component", nameStr),
		pulseUs:  cfg.InitialPulseUs,
		periodNs: cfg.PeriodNs(),
		stopCh:   make(chan struct{}),
		doneCh:   make(chan struct{}),
	}

	// Open GPIO chip
	chip, err := gpiocdev.NewChip(cfg.Chip)
	if err != nil {
		return nil, fmt.Errorf("failed to open GPIO chip %s: %w", cfg.Chip, err)
	}
	p.chip = chip

	// Request line as output, initially LOW
	line, err := chip.RequestLine(cfg.Pin, gpiocdev.AsOutput(0))
	if err != nil {
		chip.Close()
		return nil, fmt.Errorf("failed to request GPIO pin %d: %w", cfg.Pin, err)
	}
	p.line = line

	// Start PWM goroutine (initially disabled)
	go p.pwmLoop()

	p.logger.Info("PWM component initialized",
		"chip", cfg.Chip,
		"pin", cfg.Pin,
		"frequency_hz", cfg.FrequencyHz,
		"min_pulse_us", cfg.MinPulseUs,
		"max_pulse_us", cfg.MaxPulseUs,
	)

	return p, nil
}

// Name returns the resource name.
func (p *PWM) Name() resource.Name {
	return p.name
}

// Reconfigure updates the component configuration.
func (p *PWM) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	// For now, reconfiguration is not supported without restart
	return fmt.Errorf("reconfiguration not supported, restart component instead")
}

// DoCommand executes arbitrary commands.
func (p *PWM) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	p.cmdCount.Add(1)

	cmdName, _ := cmd["command"].(string)
	switch cmdName {
	case "set_pulse":
		pulseUs, ok := cmd["pulse_us"].(float64)
		if !ok {
			return nil, fmt.Errorf("pulse_us required for set_pulse command")
		}
		if err := p.SetPulse(ctx, pulseUs); err != nil {
			return nil, err
		}
		return map[string]any{"status": "ok", "pulse_us": pulseUs}, nil

	case "set_normalized":
		value, ok := cmd["value"].(float64)
		if !ok {
			return nil, fmt.Errorf("value required for set_normalized command")
		}
		if err := p.SetNormalized(ctx, value); err != nil {
			return nil, err
		}
		return map[string]any{"status": "ok", "normalized": value}, nil

	case "set_duty":
		duty, ok := cmd["duty"].(float64)
		if !ok {
			return nil, fmt.Errorf("duty required for set_duty command")
		}
		if err := p.SetDuty(ctx, duty); err != nil {
			return nil, err
		}
		return map[string]any{"status": "ok", "duty": duty}, nil

	case "enable":
		if err := p.Enable(ctx); err != nil {
			return nil, err
		}
		return map[string]any{"status": "ok", "enabled": true}, nil

	case "disable":
		if err := p.Disable(ctx); err != nil {
			return nil, err
		}
		return map[string]any{"status": "ok", "enabled": false}, nil

	case "get_state":
		p.mu.RLock()
		state := map[string]any{
			"pulse_us":       p.pulseUs,
			"normalized":     p.pulseToNormalized(p.pulseUs),
			"duty_cycle":     p.pulseToDuty(p.pulseUs),
			"enabled":        p.enabled,
			"frequency_hz":   p.config.FrequencyHz,
			"min_pulse_us":   p.config.MinPulseUs,
			"max_pulse_us":   p.config.MaxPulseUs,
			"cmd_count":      p.cmdCount.Load(),
			"cycle_count":    p.cycleCount.Load(),
		}
		p.mu.RUnlock()
		return state, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close releases resources.
func (p *PWM) Close(ctx context.Context) error {
	p.logger.Info("Closing PWM component")

	// Stop PWM loop
	close(p.stopCh)

	// Wait for loop to finish with timeout
	select {
	case <-p.doneCh:
	case <-time.After(time.Second):
		p.logger.Warn("PWM loop did not stop cleanly")
	}

	// Set pin LOW
	if p.line != nil {
		p.line.SetValue(0)
		p.line.Close()
	}

	// Close chip
	if p.chip != nil {
		p.chip.Close()
	}

	return nil
}

// SetPulse sets the pulse width in microseconds.
func (p *PWM) SetPulse(ctx context.Context, pulseUs float64) error {
	// Clamp to configured limits
	if pulseUs < p.config.MinPulseUs {
		pulseUs = p.config.MinPulseUs
	}
	if pulseUs > p.config.MaxPulseUs {
		pulseUs = p.config.MaxPulseUs
	}

	p.mu.Lock()
	p.pulseUs = pulseUs
	p.mu.Unlock()

	p.logger.Debug("Set pulse", "pulse_us", pulseUs)
	return nil
}

// SetNormalized sets the output using a normalized value from -1.0 to 1.0.
func (p *PWM) SetNormalized(ctx context.Context, value float64) error {
	// Clamp normalized value
	if value < -1.0 {
		value = -1.0
	}
	if value > 1.0 {
		value = 1.0
	}

	pulseUs := p.normalizedToPulse(value)
	return p.SetPulse(ctx, pulseUs)
}

// SetDuty sets the duty cycle as a fraction from 0.0 to 1.0.
func (p *PWM) SetDuty(ctx context.Context, duty float64) error {
	// Clamp duty cycle
	if duty < 0.0 {
		duty = 0.0
	}
	if duty > 1.0 {
		duty = 1.0
	}

	pulseUs := p.dutyToPulse(duty)
	return p.SetPulse(ctx, pulseUs)
}

// Enable starts PWM signal generation.
func (p *PWM) Enable(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.enabled {
		return nil
	}

	p.enabled = true
	p.logger.Info("PWM enabled", "pulse_us", p.pulseUs)
	return nil
}

// Disable stops PWM signal generation.
func (p *PWM) Disable(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.enabled {
		return nil
	}

	p.enabled = false

	// Set pin LOW immediately
	if p.line != nil {
		p.line.SetValue(0)
	}

	p.logger.Info("PWM disabled")
	return nil
}

// IsEnabled returns true if PWM is currently generating a signal.
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
	return p.pulseToNormalized(p.pulseUs), nil
}

// Properties returns the PWM configuration and capabilities.
func (p *PWM) Properties(ctx context.Context) (pwm.Properties, error) {
	return pwm.Properties{
		FrequencyHz: p.config.FrequencyHz,
		MinPulseUs:  p.config.MinPulseUs,
		MaxPulseUs:  p.config.MaxPulseUs,
		Pin:         p.config.Pin,
		Chip:        p.config.Chip,
		Inverted:    p.config.Invert,
	}, nil
}

// pwmLoop is the main PWM generation goroutine.
func (p *PWM) pwmLoop() {
	defer close(p.doneCh)

	for {
		select {
		case <-p.stopCh:
			return
		default:
		}

		p.mu.RLock()
		enabled := p.enabled
		pulseUs := p.pulseUs
		invert := p.config.Invert
		p.mu.RUnlock()

		if !enabled {
			// Sleep briefly when disabled to avoid busy-waiting
			time.Sleep(10 * time.Millisecond)
			continue
		}

		// Calculate timing for this cycle
		pulseNs := int64(pulseUs * 1000)

		// High phase
		highVal := 1
		lowVal := 0
		if invert {
			highVal = 0
			lowVal = 1
		}

		cycleStart := time.Now()

		// Set HIGH
		p.line.SetValue(highVal)

		// Wait for pulse duration
		p.precisionSleep(pulseNs)

		// Set LOW
		p.line.SetValue(lowVal)

		// Wait for remaining period
		elapsed := time.Since(cycleStart).Nanoseconds()
		remaining := p.periodNs - elapsed
		if remaining > 0 {
			time.Sleep(time.Duration(remaining))
		}

		p.cycleCount.Add(1)
	}
}

// precisionSleep provides more accurate timing for short durations.
// For durations < 100µs, uses busy-waiting. For longer durations, combines
// sleep with busy-waiting for the final portion.
func (p *PWM) precisionSleep(ns int64) {
	if ns <= 0 {
		return
	}

	start := time.Now()

	// For short durations, busy-wait entirely
	if ns < 100_000 { // < 100µs
		for time.Since(start).Nanoseconds() < ns {
			// Busy wait
		}
		return
	}

	// For longer durations, sleep most of it then busy-wait
	sleepNs := ns - 50_000 // Sleep all but last 50µs
	if sleepNs > 0 {
		time.Sleep(time.Duration(sleepNs))
	}

	// Busy-wait for remaining time
	for time.Since(start).Nanoseconds() < ns {
		// Busy wait
	}
}

// Value conversion functions

// normalizedToPulse converts a normalized value (-1.0 to 1.0) to pulse width.
func (p *PWM) normalizedToPulse(normalized float64) float64 {
	center := p.config.CenterPulseUs()
	rangeUs := p.config.PulseRangeUs()
	return center + (normalized * rangeUs)
}

// pulseToNormalized converts a pulse width to normalized value (-1.0 to 1.0).
func (p *PWM) pulseToNormalized(pulseUs float64) float64 {
	center := p.config.CenterPulseUs()
	rangeUs := p.config.PulseRangeUs()
	if rangeUs == 0 {
		return 0
	}
	return (pulseUs - center) / rangeUs
}

// dutyToPulse converts a duty cycle (0.0 to 1.0) to pulse width.
func (p *PWM) dutyToPulse(duty float64) float64 {
	periodUs := p.config.PeriodUs()
	return duty * periodUs
}

// pulseToDuty converts a pulse width to duty cycle (0.0 to 1.0).
func (p *PWM) pulseToDuty(pulseUs float64) float64 {
	periodUs := p.config.PeriodUs()
	if periodUs == 0 {
		return 0
	}
	return pulseUs / periodUs
}

// Verify interface compliance at compile time.
var _ pwm.PWM = (*PWM)(nil)

