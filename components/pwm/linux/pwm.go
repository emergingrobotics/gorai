package linux

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/components/pwm"
	driverpwm "github.com/emergingrobotics/gorai/driver/pwm"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("pwm", "linux", New)
}

// channelOpener opens a hardware PWM channel for a given chip and channel index.
// It is injectable so the component logic can be unit-tested without hardware.
type channelOpener func(chip, channel int) (driverpwm.Channel, error)

// polaritySetter is an optional capability for channels that support inversion.
type polaritySetter interface {
	SetPolarity(ctx context.Context, inverted bool) error
}

// PWM drives a hardware PWM channel via the Linux sysfs driver.
type PWM struct {
	name    resource.Name
	config  Config
	logger  *slog.Logger
	channel driverpwm.Channel

	mu      sync.RWMutex
	pulseUs float64
	enabled bool
	arming  bool
}

// New creates a native sysfs PWM component.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)

	cfg, err := ParseConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger := slog.Default()
	if loggerRes, err := deps.Get("logger"); err == nil {
		if l, ok := loggerRes.(*slog.Logger); ok {
			logger = l
		}
	}

	return newPWM(ctx, nameStr, cfg, defaultOpener, logger)
}

// newPWM constructs the component using the supplied channel opener. It opens
// the channel, sets frequency and the initial (neutral) pulse, and optionally
// enables output.
func newPWM(ctx context.Context, name string, cfg Config, opener channelOpener, logger *slog.Logger) (*PWM, error) {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "pwm/linux", "name", name)

	channel, err := opener(cfg.Chip, cfg.Channel)
	if err != nil {
		return nil, fmt.Errorf("failed to open PWM chip %d channel %d: %w", cfg.Chip, cfg.Channel, err)
	}

	p := &PWM{
		name:    resource.NewComponentName("gorai", "pwm", name),
		config:  cfg,
		logger:  logger,
		channel: channel,
		pulseUs: cfg.InitialPulseUs,
	}

	// Frequency (period) must be set before duty so the kernel accepts the write.
	if err := channel.SetFrequency(ctx, cfg.FrequencyHz); err != nil {
		return nil, fmt.Errorf("failed to set frequency: %w", err)
	}

	if cfg.Invert {
		if ps, ok := channel.(polaritySetter); ok {
			if err := ps.SetPolarity(ctx, true); err != nil {
				return nil, fmt.Errorf("failed to set polarity: %w", err)
			}
		} else {
			logger.Warn("invert requested but controller does not support polarity")
		}
	}

	if err := channel.SetDuty(ctx, pulseToNs(cfg.InitialPulseUs)); err != nil {
		return nil, fmt.Errorf("failed to set initial pulse: %w", err)
	}

	if cfg.EnableOnStart {
		if err := channel.Enable(ctx); err != nil {
			return nil, fmt.Errorf("failed to enable PWM: %w", err)
		}
		p.enabled = true
	}

	logger.Info("native PWM initialized",
		"chip", cfg.Chip,
		"channel", cfg.Channel,
		"frequency_hz", cfg.FrequencyHz,
		"initial_pulse_us", cfg.InitialPulseUs,
		"enabled", p.enabled,
	)

	return p, nil
}

// pulseToNs converts a pulse width in microseconds to nanoseconds.
func pulseToNs(pulseUs float64) uint64 {
	if pulseUs < 0 {
		return 0
	}
	return uint64(pulseUs * 1000)
}

// Name returns the resource name.
func (p *PWM) Name() resource.Name {
	return p.name
}

// Reconfigure is not supported; restart the component instead.
func (p *PWM) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return fmt.Errorf("reconfiguration not supported, restart component instead")
}

// DoCommand exposes actuator methods through the generic command interface.
func (p *PWM) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
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

	case "arm":
		if err := p.Arm(ctx); err != nil {
			return nil, err
		}
		return map[string]any{"status": "ok", "armed": true}, nil

	case "get_state":
		p.mu.RLock()
		defer p.mu.RUnlock()
		return map[string]any{
			"pulse_us":     p.pulseUs,
			"enabled":      p.enabled,
			"frequency_hz": p.config.FrequencyHz,
			"min_pulse_us": p.config.MinPulseUs,
			"max_pulse_us": p.config.MaxPulseUs,
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close disables output and releases the channel.
func (p *PWM) Close(ctx context.Context) error {
	if p.channel != nil {
		if err := p.channel.Disable(ctx); err != nil {
			p.logger.Warn("failed to disable PWM on close", "error", err)
		}
	}
	return nil
}

// SetPulse sets the pulse width in microseconds, clamped to the configured range.
func (p *PWM) SetPulse(ctx context.Context, pulseUs float64) error {
	if pulseUs < p.config.MinPulseUs {
		pulseUs = p.config.MinPulseUs
	}
	if pulseUs > p.config.MaxPulseUs {
		pulseUs = p.config.MaxPulseUs
	}

	if err := p.channel.SetDuty(ctx, pulseToNs(pulseUs)); err != nil {
		return fmt.Errorf("failed to set duty: %w", err)
	}

	p.mu.Lock()
	p.pulseUs = pulseUs
	p.mu.Unlock()

	p.logger.Info("set pulse", "pulse_us", pulseUs)
	return nil
}

// SetNormalized sets the output from a -1.0..1.0 value mapped to min..max pulse.
func (p *PWM) SetNormalized(ctx context.Context, value float64) error {
	if value < -1.0 {
		value = -1.0
	}
	if value > 1.0 {
		value = 1.0
	}
	pulseUs := p.config.CenterPulseUs() + value*p.config.PulseRangeUs()
	return p.SetPulse(ctx, pulseUs)
}

// SetDuty sets the duty cycle as a fraction (0.0..1.0) of the period.
func (p *PWM) SetDuty(ctx context.Context, duty float64) error {
	if duty < 0.0 {
		duty = 0.0
	}
	if duty > 1.0 {
		duty = 1.0
	}
	return p.SetPulse(ctx, duty*p.config.PeriodUs())
}

// Enable starts signal generation.
func (p *PWM) Enable(ctx context.Context) error {
	if err := p.channel.Enable(ctx); err != nil {
		return fmt.Errorf("failed to enable PWM: %w", err)
	}
	p.mu.Lock()
	p.enabled = true
	p.mu.Unlock()
	return nil
}

// Disable stops signal generation.
func (p *PWM) Disable(ctx context.Context) error {
	if err := p.channel.Disable(ctx); err != nil {
		return fmt.Errorf("failed to disable PWM: %w", err)
	}
	p.mu.Lock()
	p.enabled = false
	p.mu.Unlock()
	return nil
}

// Arm runs the configured arming sequence, holding each pulse for its duration.
// It ensures output is enabled first and rejects concurrent arm attempts. The
// sequence is interrupted if the context is cancelled.
func (p *PWM) Arm(ctx context.Context) error {
	p.mu.Lock()
	if p.arming {
		p.mu.Unlock()
		return fmt.Errorf("arm already in progress")
	}
	p.arming = true
	p.mu.Unlock()
	defer func() {
		p.mu.Lock()
		p.arming = false
		p.mu.Unlock()
	}()

	if len(p.config.ArmSequence) == 0 {
		return fmt.Errorf("no arm sequence configured")
	}

	if err := p.Enable(ctx); err != nil {
		return fmt.Errorf("failed to enable before arming: %w", err)
	}

	p.logger.Info("arming ESC", "steps", len(p.config.ArmSequence))
	for i, step := range p.config.ArmSequence {
		if err := p.SetPulse(ctx, step.PulseUs); err != nil {
			return fmt.Errorf("arm step %d failed: %w", i, err)
		}
		select {
		case <-time.After(time.Duration(step.HoldMs) * time.Millisecond):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	p.logger.Info("ESC armed")
	return nil
}

// IsEnabled reports whether output is currently generating a signal.
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

// GetNormalized returns the current pulse as a -1.0..1.0 value.
func (p *PWM) GetNormalized(ctx context.Context) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	rangeUs := p.config.PulseRangeUs()
	if rangeUs == 0 {
		return 0, nil
	}
	return (p.pulseUs - p.config.CenterPulseUs()) / rangeUs, nil
}

// Properties returns the PWM configuration and capabilities.
func (p *PWM) Properties(ctx context.Context) (pwm.Properties, error) {
	return pwm.Properties{
		FrequencyHz: p.config.FrequencyHz,
		MinPulseUs:  p.config.MinPulseUs,
		MaxPulseUs:  p.config.MaxPulseUs,
		Inverted:    p.config.Invert,
		Mode:        "hardware",
	}, nil
}

// Verify interface compliance at compile time.
var _ pwm.PWM = (*PWM)(nil)
