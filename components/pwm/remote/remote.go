package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gorai/gorai/components/pwm"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterComponent("pwm", "remote", New)
}

// pwmSetPayload matches the JSON format expected by gorai-nats-gw for PWM_SET.
type pwmSetPayload struct {
	Channels []pwmSetChannel `json:"channels"`
}

type pwmSetChannel struct {
	Channel uint8  `json:"channel"`
	PulseUS uint16 `json:"pulse_us"`
}

// pwmEnablePayload matches the JSON format expected by gorai-nats-gw for PWM_ENABLE.
type pwmEnablePayload struct {
	Channels []pwmEnableChannel `json:"channels"`
}

type pwmEnableChannel struct {
	Channel uint8 `json:"channel"`
	Enabled bool  `json:"enabled"`
}

// RemotePWM implements pwm.PWM by publishing commands to NATS,
// which gorai-nats-gw bridges to a physical device via GSP/2.
type RemotePWM struct {
	name   resource.Name
	config *Config
	logger *slog.Logger
	nc     *nats.Conn

	mu            sync.RWMutex
	is_enabled    bool
	current_pulse float64
}

// New creates a new Remote PWM component.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	resConf := resource.NewConfig(conf)

	cfg, err := NewConfigFromResource(resConf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	name := "remote_pwm"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	logger := slog.Default()
	if loggerRes, err := deps.Get("logger"); err == nil {
		if l, ok := loggerRes.(*slog.Logger); ok {
			logger = l
		}
	}

	r := &RemotePWM{
		name:          resource.NewComponentName("gorai", "pwm", name),
		config:        cfg,
		logger:        logger.With("component", "remote_pwm", "name", name),
		current_pulse: cfg.InitialPulseUs,
	}

	if natsRes, err := deps.Get("nats"); err == nil {
		if nc, ok := natsRes.(*nats.Conn); ok {
			r.nc = nc
		}
	}

	if r.nc == nil {
		return nil, fmt.Errorf("NATS connection not available")
	}

	r.logger.Info("remote PWM component created",
		"device_id", cfg.DeviceID,
		"channel", cfg.Channel,
		"prefix", cfg.NATSSubjectPrefix,
	)

	return r, nil
}

// Name returns the resource name.
func (r *RemotePWM) Name() resource.Name {
	return r.name
}

// Reconfigure updates the component configuration.
func (r *RemotePWM) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	r.mu.Lock()
	r.config = cfg
	r.mu.Unlock()

	return nil
}

// SetPulse sets the pulse width in microseconds.
func (r *RemotePWM) SetPulse(ctx context.Context, pulseUs float64) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	pulseUs = clamp(pulseUs, cfg.MinPulseUs, cfg.MaxPulseUs)

	payload := pwmSetPayload{
		Channels: []pwmSetChannel{
			{Channel: uint8(cfg.Channel), PulseUS: uint16(pulseUs)},
		},
	}

	if err := r.publishCommand("pwm_set", payload); err != nil {
		return fmt.Errorf("failed to publish PWM_SET: %w", err)
	}

	r.mu.Lock()
	r.current_pulse = pulseUs
	r.mu.Unlock()

	r.logger.Debug("pulse set", "channel", cfg.Channel, "pulse_us", pulseUs)
	return nil
}

// SetNormalized sets the output using a normalized value from -1.0 to 1.0.
func (r *RemotePWM) SetNormalized(ctx context.Context, value float64) error {
	value = clamp(value, -1.0, 1.0)

	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	center := (cfg.MinPulseUs + cfg.MaxPulseUs) / 2.0
	half_range := (cfg.MaxPulseUs - cfg.MinPulseUs) / 2.0
	pulseUs := center + value*half_range

	return r.SetPulse(ctx, pulseUs)
}

// SetDuty sets the duty cycle as a fraction from 0.0 to 1.0.
func (r *RemotePWM) SetDuty(ctx context.Context, duty float64) error {
	duty = clamp(duty, 0.0, 1.0)

	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	period_us := 1_000_000.0 / cfg.FrequencyHz
	pulseUs := duty * period_us

	return r.SetPulse(ctx, pulseUs)
}

// Enable starts PWM signal generation on the remote device.
func (r *RemotePWM) Enable(ctx context.Context) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	payload := pwmEnablePayload{
		Channels: []pwmEnableChannel{
			{Channel: uint8(cfg.Channel), Enabled: true},
		},
	}

	if err := r.publishCommand("pwm_enable", payload); err != nil {
		return fmt.Errorf("failed to publish PWM_ENABLE: %w", err)
	}

	r.mu.Lock()
	r.is_enabled = true
	r.mu.Unlock()

	r.logger.Debug("channel enabled", "channel", cfg.Channel)
	return nil
}

// Disable stops PWM signal generation on the remote device.
func (r *RemotePWM) Disable(ctx context.Context) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	payload := pwmEnablePayload{
		Channels: []pwmEnableChannel{
			{Channel: uint8(cfg.Channel), Enabled: false},
		},
	}

	if err := r.publishCommand("pwm_enable", payload); err != nil {
		return fmt.Errorf("failed to publish PWM_ENABLE: %w", err)
	}

	r.mu.Lock()
	r.is_enabled = false
	r.mu.Unlock()

	r.logger.Debug("channel disabled", "channel", cfg.Channel)
	return nil
}

// IsEnabled returns whether PWM is currently generating a signal.
func (r *RemotePWM) IsEnabled(ctx context.Context) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.is_enabled, nil
}

// GetPulse returns the current pulse width in microseconds.
func (r *RemotePWM) GetPulse(ctx context.Context) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.current_pulse, nil
}

// GetNormalized returns the current value as normalized (-1.0 to 1.0).
func (r *RemotePWM) GetNormalized(ctx context.Context) (float64, error) {
	r.mu.RLock()
	pulse := r.current_pulse
	cfg := r.config
	r.mu.RUnlock()

	center := (cfg.MinPulseUs + cfg.MaxPulseUs) / 2.0
	half_range := (cfg.MaxPulseUs - cfg.MinPulseUs) / 2.0
	if half_range == 0 {
		return 0, nil
	}
	return (pulse - center) / half_range, nil
}

// Properties returns the PWM configuration and capabilities.
func (r *RemotePWM) Properties(ctx context.Context) (pwm.Properties, error) {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	return pwm.Properties{
		FrequencyHz: cfg.FrequencyHz,
		MinPulseUs:  cfg.MinPulseUs,
		MaxPulseUs:  cfg.MaxPulseUs,
		Pin:         cfg.Channel,
		Mode:        "remote",
	}, nil
}

// DoCommand handles arbitrary commands.
func (r *RemotePWM) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)

	switch cmdName {
	case "get_state":
		r.mu.RLock()
		defer r.mu.RUnlock()
		return map[string]any{
			"channel":       r.config.Channel,
			"device_id":     r.config.DeviceID,
			"current_pulse": r.current_pulse,
			"is_enabled":    r.is_enabled,
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close releases all resources.
func (r *RemotePWM) Close(ctx context.Context) error {
	r.logger.Info("remote PWM component closed")
	return nil
}

// publishCommand marshals the payload and publishes to the correct NATS subject.
func (r *RemotePWM) publishCommand(commandType string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	subject := r.config.CommandSubject(commandType)
	return r.nc.Publish(subject, data)
}

func clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// Verify interface compliance at compile time.
var _ pwm.PWM = (*RemotePWM)(nil)
