// Package gpiod provides a PWM implementation supporting both hardware and software modes.
//
// NOTE: This package is currently DISABLED because the HAL (Hardware Abstraction Layer)
// has been removed. The HAL provided hardware access for GPIO, PWM, I2C, etc.
// This component will be re-enabled when HAL is reimplemented.
//
// This package implements the pwm.PWM interface for Raspberry Pi 5 and other
// Linux systems with GPIO support via the HAL (Hardware Abstraction Layer).
//
// Hardware PWM uses the Linux sysfs interface (/sys/class/pwm) for precise timing.
// Software PWM is generated in a dedicated goroutine with timing achieved
// through a combination of time.Sleep and busy-waiting (less precise).
//
// The component automatically detects hardware PWM support based on the pin
// and falls back to software PWM with a warning when hardware is not available.
package gpiod

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorai/gorai/components/pwm"
	"github.com/gorai/gorai/driver/gpio"
	driverpwm "github.com/gorai/gorai/driver/pwm"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

// HAL interface stub - will be provided by reimplemented HAL package
type halInterface interface {
	ResolvePinFromAny(v any) (int, error)
	GetPWMMapping(pin int) (chip, channel int, hasHardware bool)
	Board() string
	GPIO() (gpio.Driver, error)
	PWM(chip int) (driverpwm.Chip, error)
}

// NOTE: Registration disabled - HAL package removed
// func init() {
// 	registry.RegisterComponent("pwm", "gpiod", New)
// }

// PWM implements hardware or software PWM using HAL.
type PWM struct {
	name   resource.Name
	config Config
	logger *slog.Logger

	hal       halInterface
	pinNumber int

	// Hardware PWM (used when isHardware is true)
	hwChannel  driverpwm.Channel
	isHardware bool

	// Software PWM (used when isHardware is false)
	gpioPin gpio.Pin
	stopCh  chan struct{}
	doneCh  chan struct{}

	mu       sync.RWMutex
	pulseUs  float64
	enabled  bool
	periodNs int64

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

	// Get HAL from dependencies
	halAny, err := deps.Get("hal")
	if err != nil {
		return nil, fmt.Errorf("HAL not available: %w (ensure platform section is configured)", err)
	}
	h, ok := halAny.(halInterface)
	if !ok {
		return nil, fmt.Errorf("invalid HAL type: %T", halAny)
	}

	// Resolve pin using HAL
	pinNumber, err := h.ResolvePinFromAny(cfg.Pin)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve pin %v: %w", cfg.Pin, err)
	}

	// Get logger from dependencies or use default
	logger := slog.Default()
	if loggerRes, err := deps.Get("logger"); err == nil {
		if l, ok := loggerRes.(*slog.Logger); ok {
			logger = l
		}
	}
	logger = logger.With("component", nameStr)

	// Check for hardware PWM support
	chip, channel, hasHardwarePWM := h.GetPWMMapping(pinNumber)

	// Determine whether to use hardware or software PWM
	useHardware := false
	hwMode := cfg.HWMode
	if hwMode == "" {
		hwMode = HWModeAuto
	}

	switch hwMode {
	case HWModeHardware:
		if !hasHardwarePWM {
			hwPins := []int{12, 13, 18, 19} // Default PWM pins - TODO: get from reimplemented HAL
			return nil, fmt.Errorf("hardware PWM not available on GPIO %d. "+
				"Hardware PWM pins on %s: %v", pinNumber, h.Board(), formatPinList(hwPins))
		}
		useHardware = true

	case HWModeSoftware:
		useHardware = false

	case HWModeAuto:
		if hasHardwarePWM {
			useHardware = true
		} else {
			// Emit warning about software PWM
			hwPins := []int{12, 13, 18, 19} // Default PWM pins - TODO: get from reimplemented HAL
			logger.Warn("PWM using software mode - timing may be inaccurate",
				"pin", pinNumber,
				"reason", "pin does not support hardware PWM",
				"hardware_pwm_pins", formatPinList(hwPins),
				"recommendation", fmt.Sprintf("use GPIO %v for precise servo control", hwPins),
			)
			useHardware = false
		}
	}

	p := &PWM{
		name:       name,
		config:     cfg,
		logger:     logger,
		hal:        h,
		pinNumber:  pinNumber,
		pulseUs:    cfg.InitialPulseUs,
		periodNs:   cfg.PeriodNs(),
		isHardware: useHardware,
	}

	if useHardware {
		// Initialize hardware PWM
		if err := p.initHardwarePWM(ctx, chip, channel); err != nil {
			return nil, fmt.Errorf("failed to initialize hardware PWM: %w", err)
		}
	} else {
		// Initialize software PWM
		if err := p.initSoftwarePWM(ctx); err != nil {
			return nil, fmt.Errorf("failed to initialize software PWM: %w", err)
		}
	}

	modeStr := "software"
	if useHardware {
		modeStr = "hardware"
	}
	p.logger.Info("PWM component initialized",
		"mode", modeStr,
		"board", h.Board(),
		"pin", pinNumber,
		"pin_ref", cfg.Pin,
		"frequency_hz", cfg.FrequencyHz,
		"min_pulse_us", cfg.MinPulseUs,
		"max_pulse_us", cfg.MaxPulseUs,
	)

	return p, nil
}

// initHardwarePWM sets up hardware PWM via sysfs.
func (p *PWM) initHardwarePWM(ctx context.Context, chip, channel int) error {
	// Get PWM chip from HAL
	pwmChip, err := p.hal.PWM(chip)
	if err != nil {
		return fmt.Errorf("failed to open PWM chip %d: %w", chip, err)
	}

	// Get the channel
	hwChan, err := pwmChip.Channel(channel)
	if err != nil {
		return fmt.Errorf("failed to get PWM channel %d: %w", channel, err)
	}

	p.hwChannel = hwChan

	// Set frequency (period)
	if err := hwChan.SetFrequency(ctx, p.config.FrequencyHz); err != nil {
		return fmt.Errorf("failed to set frequency: %w", err)
	}

	// Set initial pulse width as duty cycle
	pulseNs := uint64(p.config.InitialPulseUs * 1000)
	if err := hwChan.SetDuty(ctx, pulseNs); err != nil {
		return fmt.Errorf("failed to set initial duty: %w", err)
	}

	p.logger.Debug("Hardware PWM initialized",
		"chip", chip,
		"channel", channel,
		"frequency_hz", p.config.FrequencyHz,
		"initial_pulse_us", p.config.InitialPulseUs,
	)

	return nil
}

// initSoftwarePWM sets up software PWM via GPIO bit-banging.
func (p *PWM) initSoftwarePWM(ctx context.Context) error {
	// Get GPIO driver from HAL
	gpioDriver, err := p.hal.GPIO()
	if err != nil {
		return fmt.Errorf("failed to get GPIO driver: %w", err)
	}

	// Get the pin
	gpioPin, err := gpioDriver.Pin(p.pinNumber)
	if err != nil {
		return fmt.Errorf("failed to get GPIO pin %d: %w", p.pinNumber, err)
	}

	// Set pin as output
	if err := gpioPin.SetDirection(ctx, gpio.Output); err != nil {
		return fmt.Errorf("failed to set pin %d as output: %w", p.pinNumber, err)
	}

	p.gpioPin = gpioPin
	p.stopCh = make(chan struct{})
	p.doneCh = make(chan struct{})

	// Start PWM goroutine (initially disabled)
	go p.pwmLoop()

	p.logger.Debug("Software PWM initialized",
		"pin", p.pinNumber,
		"frequency_hz", p.config.FrequencyHz,
	)

	return nil
}

// formatPinList formats a list of pins for display.
func formatPinList(pins []int) string {
	if len(pins) == 0 {
		return "none"
	}
	sort.Ints(pins)
	result := ""
	for i, pin := range pins {
		if i > 0 {
			result += ", "
		}
		result += fmt.Sprintf("%d", pin)
	}
	return result
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
		mode := "software"
		if p.isHardware {
			mode = "hardware"
		}
		state := map[string]any{
			"pulse_us":     p.pulseUs,
			"normalized":   p.pulseToNormalized(p.pulseUs),
			"duty_cycle":   p.pulseToDuty(p.pulseUs),
			"enabled":      p.enabled,
			"frequency_hz": p.config.FrequencyHz,
			"min_pulse_us": p.config.MinPulseUs,
			"max_pulse_us": p.config.MaxPulseUs,
			"cmd_count":    p.cmdCount.Load(),
			"cycle_count":  p.cycleCount.Load(),
			"mode":         mode,
			"pin":          p.pinNumber,
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

	if p.isHardware {
		// Disable hardware PWM
		if p.hwChannel != nil {
			if err := p.hwChannel.Disable(ctx); err != nil {
				p.logger.Warn("Failed to disable hardware PWM on close", "error", err)
			}
		}
		// Note: PWM chip cleanup is handled by HAL.Close()
	} else {
		// Stop software PWM loop
		if p.stopCh != nil {
			close(p.stopCh)

			// Wait for loop to finish with timeout
			select {
			case <-p.doneCh:
			case <-time.After(time.Second):
				p.logger.Warn("PWM loop did not stop cleanly")
			}
		}

		// Set pin LOW
		if p.gpioPin != nil {
			p.gpioPin.Write(ctx, false)
		}
		// Note: GPIO driver cleanup is handled by HAL.Close()
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

	// For hardware PWM, update immediately
	if p.isHardware && p.hwChannel != nil {
		pulseNs := uint64(pulseUs * 1000)
		if err := p.hwChannel.SetDuty(ctx, pulseNs); err != nil {
			return fmt.Errorf("failed to set hardware PWM duty: %w", err)
		}
	}
	// For software PWM, the pwmLoop will pick up the new value

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

	// For hardware PWM, enable the channel
	if p.isHardware && p.hwChannel != nil {
		// Set the current pulse before enabling
		pulseNs := uint64(p.pulseUs * 1000)
		if err := p.hwChannel.SetDuty(ctx, pulseNs); err != nil {
			return fmt.Errorf("failed to set duty before enable: %w", err)
		}
		if err := p.hwChannel.Enable(ctx); err != nil {
			return fmt.Errorf("failed to enable hardware PWM: %w", err)
		}
	}
	// For software PWM, the pwmLoop will start generating signal

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

	// For hardware PWM, disable the channel
	if p.isHardware && p.hwChannel != nil {
		if err := p.hwChannel.Disable(ctx); err != nil {
			return fmt.Errorf("failed to disable hardware PWM: %w", err)
		}
	} else if p.gpioPin != nil {
		// For software PWM, set pin LOW immediately
		p.gpioPin.Write(ctx, false)
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
	mode := "software"
	if p.isHardware {
		mode = "hardware"
	}
	return pwm.Properties{
		FrequencyHz: p.config.FrequencyHz,
		MinPulseUs:  p.config.MinPulseUs,
		MaxPulseUs:  p.config.MaxPulseUs,
		Pin:         p.pinNumber,
		Board:       string(p.hal.Board()),
		Inverted:    p.config.Invert,
		Mode:        mode,
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
		highVal := true
		lowVal := false
		if invert {
			highVal = false
			lowVal = true
		}

		cycleStart := time.Now()

		// Set HIGH using gpio.Pin interface
		ctx := context.Background()
		p.gpioPin.Write(ctx, highVal)

		// Wait for pulse duration
		p.precisionSleep(pulseNs)

		// Set LOW
		p.gpioPin.Write(ctx, lowVal)

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
