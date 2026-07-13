package remote

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/components/pwm"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterComponent("pwm", "remote", New)
}

// gpioConfigPayload matches the JSON format expected by gorai-nats-gw for GPIO_CONFIG.
type gpioConfigPayload struct {
	Pin     uint8 `json:"pin"`
	Mode    uint8 `json:"mode"`
	Pull    uint8 `json:"pull"`
	Options uint8 `json:"options"`
}

// pwmConfigPayload matches the JSON format expected by gorai-nats-gw for PWM_CONFIG.
type pwmConfigPayload struct {
	Channel    uint8  `json:"channel"`
	MinUs      uint16 `json:"min_us"`
	MaxUs      uint16 `json:"max_us"`
	CenterUs   uint16 `json:"center_us"`
	FailsafeUs uint16 `json:"failsafe_us"`
	Flags      uint8  `json:"flags"`
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

// pwmStatePayload represents device-reported PWM state.
type pwmStatePayload struct {
	Channels []pwmStateChannel `json:"channels"`
}

type pwmStateChannel struct {
	Channel uint8  `json:"channel"`
	PulseUS uint16 `json:"pulse_us"`
	Flags   uint8  `json:"flags"`
}

// configSetPayload matches the JSON format expected by gorai-nats-gw for CONFIG_SET.
type configSetPayload struct {
	Key   uint8  `json:"key"`
	Value []byte `json:"value"`
}

// resetPayload matches the JSON format expected by gorai-nats-gw for RESET.
type resetPayload struct {
	Subsystem uint8 `json:"subsystem"`
}

const (
	gpioModePWM           = 0x02
	configKeyPWMFrequency = 0x10
)

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
	state_sub     *nats.Subscription
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
		"pin", cfg.Pin,
		"prefix", cfg.NATSSubjectPrefix,
	)

	return r, nil
}

// Name returns the resource name.
func (r *RemotePWM) Name() resource.Name {
	return r.name
}

// Start sends the provisioning sequence to configure the pin on the remote
// device (gsp-pico-fw). Skipped when auto_configure is false.
func (r *RemotePWM) Start(ctx context.Context) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	r.subscribeState()

	if !cfg.AutoConfigure {
		r.logger.Info("auto_configure disabled, skipping pin provisioning")
		return nil
	}

	r.logger.Info("provisioning PWM pin",
		"pin", cfg.Pin,
		"frequency_hz", cfg.FrequencyHz,
		"min_us", cfg.MinPulseUs,
		"max_us", cfg.MaxPulseUs,
	)

	// 1. GPIO_CONFIG -- claim pin as PWM
	gpio_cfg := gpioConfigPayload{
		Pin:  uint8(cfg.Pin),
		Mode: gpioModePWM,
	}
	if err := r.publishCommand("gpio_config", gpio_cfg); err != nil {
		return fmt.Errorf("failed to send GPIO_CONFIG: %w", err)
	}

	// 2. CONFIG_SET PWM_FREQUENCY -- set slice frequency
	freq_value := make([]byte, 5)
	freq_value[0] = uint8(cfg.Pin)
	binary.BigEndian.PutUint32(freq_value[1:], uint32(cfg.FrequencyHz))
	config_set := configSetPayload{
		Key:   configKeyPWMFrequency,
		Value: freq_value,
	}
	if err := r.publishCommand("config_set", config_set); err != nil {
		return fmt.Errorf("failed to send CONFIG_SET PWM_FREQUENCY: %w", err)
	}

	// 3. PWM_CONFIG -- set limits and failsafe
	center_us := uint16((cfg.MinPulseUs + cfg.MaxPulseUs) / 2.0)
	pwm_cfg := pwmConfigPayload{
		Channel:    uint8(cfg.Pin),
		MinUs:      uint16(cfg.MinPulseUs),
		MaxUs:      uint16(cfg.MaxPulseUs),
		CenterUs:   center_us,
		FailsafeUs: uint16(cfg.FailsafePulseUs),
	}
	if err := r.publishCommand("pwm_config", pwm_cfg); err != nil {
		return fmt.Errorf("failed to send PWM_CONFIG: %w", err)
	}

	// 4. PWM_ENABLE
	if err := r.Enable(ctx); err != nil {
		return fmt.Errorf("failed to enable PWM: %w", err)
	}

	// 5. PWM_SET -- initial pulse
	if err := r.SetPulse(ctx, cfg.InitialPulseUs); err != nil {
		return fmt.Errorf("failed to set initial pulse: %w", err)
	}

	return nil
}

// subscribeState subscribes to PWM_STATE messages from the device to keep
// local state in sync with actual hardware state.
func (r *RemotePWM) subscribeState() {
	cfg := r.config
	subject := fmt.Sprintf("%s.%s.rx.response.pwm_state", cfg.NATSSubjectPrefix, cfg.DeviceID)
	sub, err := r.nc.Subscribe(subject, func(msg *nats.Msg) {
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			return
		}
		var state pwmStatePayload
		if err := json.Unmarshal(env.Data, &state); err != nil {
			return
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, ch := range state.Channels {
			if int(ch.Channel) == cfg.Pin {
				r.current_pulse = float64(ch.PulseUS)
				r.is_enabled = ch.Flags&0x01 != 0
			}
		}
	})
	if err != nil {
		r.logger.Warn("failed to subscribe to PWM_STATE", "error", err)
		return
	}
	r.mu.Lock()
	r.state_sub = sub
	r.mu.Unlock()
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
			{Channel: uint8(cfg.Pin), PulseUS: uint16(pulseUs)},
		},
	}

	if err := r.publishCommand("pwm_set", payload); err != nil {
		return fmt.Errorf("failed to publish PWM_SET: %w", err)
	}

	r.mu.Lock()
	r.current_pulse = pulseUs
	r.mu.Unlock()

	r.logger.Log(ctx, slog.LevelDebug-4, "pulse set", "pin", cfg.Pin, "pulse_us", pulseUs)
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

	if err := r.SetPulse(ctx, pulseUs); err != nil {
		return err
	}

	r.logger.Debug("duty set", "pin", cfg.Pin, "duty", duty,
		"frequency_hz", cfg.FrequencyHz)
	return nil
}

// Enable starts PWM signal generation on the remote device.
func (r *RemotePWM) Enable(ctx context.Context) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	payload := pwmEnablePayload{
		Channels: []pwmEnableChannel{
			{Channel: uint8(cfg.Pin), Enabled: true},
		},
	}

	if err := r.publishCommand("pwm_enable", payload); err != nil {
		return fmt.Errorf("failed to publish PWM_ENABLE: %w", err)
	}

	r.mu.Lock()
	r.is_enabled = true
	r.mu.Unlock()

	r.logger.Debug("pin enabled", "pin", cfg.Pin)
	return nil
}

// Disable stops PWM signal generation on the remote device.
func (r *RemotePWM) Disable(ctx context.Context) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	payload := pwmEnablePayload{
		Channels: []pwmEnableChannel{
			{Channel: uint8(cfg.Pin), Enabled: false},
		},
	}

	if err := r.publishCommand("pwm_enable", payload); err != nil {
		return fmt.Errorf("failed to publish PWM_ENABLE: %w", err)
	}

	r.mu.Lock()
	r.is_enabled = false
	r.mu.Unlock()

	r.logger.Debug("pin disabled", "pin", cfg.Pin)
	return nil
}

// Arm runs a standard ESC arming sequence on the remote device: max pulse held
// for one second, then neutral held for one second.
func (r *RemotePWM) Arm(ctx context.Context) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	if err := r.Enable(ctx); err != nil {
		return fmt.Errorf("failed to enable before arming: %w", err)
	}

	center := (cfg.MinPulseUs + cfg.MaxPulseUs) / 2.0
	steps := []struct {
		pulseUs float64
		hold    time.Duration
	}{
		{cfg.MaxPulseUs, time.Second},
		{center, time.Second},
	}
	for i, step := range steps {
		if err := r.SetPulse(ctx, step.pulseUs); err != nil {
			return fmt.Errorf("arm step %d failed: %w", i, err)
		}
		select {
		case <-time.After(step.hold):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	r.logger.Info("ESC armed", "pin", cfg.Pin)
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
		Pin:         cfg.Pin,
		Mode:        "remote",
	}, nil
}

// ResetDevice sends a full RESET command to the remote device, returning it
// to listening state with all subsystems cleared.
func (r *RemotePWM) ResetDevice(ctx context.Context) error {
	payload := resetPayload{Subsystem: 0}
	if err := r.publishCommand("reset", payload); err != nil {
		return fmt.Errorf("failed to send RESET: %w", err)
	}
	r.logger.Info("device reset sent", "device_id", r.config.DeviceID)
	return nil
}

// DoCommand handles arbitrary commands.
func (r *RemotePWM) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)

	switch cmdName {
	case "get_state":
		r.mu.RLock()
		defer r.mu.RUnlock()
		return map[string]any{
			"pin":           r.config.Pin,
			"device_id":     r.config.DeviceID,
			"current_pulse": r.current_pulse,
			"is_enabled":    r.is_enabled,
		}, nil

	case "reset_device":
		return nil, r.ResetDevice(ctx)

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close releases all resources.
func (r *RemotePWM) Close(ctx context.Context) error {
	r.mu.Lock()
	sub := r.state_sub
	r.state_sub = nil
	r.mu.Unlock()

	if sub != nil {
		_ = sub.Unsubscribe()
	}

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
