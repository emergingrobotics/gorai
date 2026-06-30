package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	pwm_input "github.com/emergingrobotics/gorai/components/pwm/input"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterComponent("pwm_input", "remote", New)
}

// gpioConfigPayload matches the JSON format expected by gorai-nats-gw for GPIO_CONFIG.
type gpioConfigPayload struct {
	Pin     uint8 `json:"pin"`
	Mode    uint8 `json:"mode"`
	Pull    uint8 `json:"pull"`
	Options uint8 `json:"options"`
}

// pwmInputDataPayload represents device-reported PWM input data.
type pwmInputDataPayload struct {
	Channels []pwmInputChannelPayload `json:"channels"`
}

type pwmInputChannelPayload struct {
	Channel  uint8  `json:"channel"`
	PulseUs  uint32 `json:"pulse_us"`
	PeriodUs uint32 `json:"period_us"`
	Flags    uint8  `json:"flags"`
}

// resetPayload matches the JSON format for RESET.
type resetPayload struct {
	Subsystem uint8 `json:"subsystem"`
}

const stale_threshold = 2 * time.Second

// RemotePWMInput implements pwm_input.PWMInput by subscribing to
// PWM_INPUT_DATA messages from NATS, bridged from a physical device via GSP/2.
type RemotePWMInput struct {
	name   resource.Name
	config *Config
	logger *slog.Logger
	nc     *nats.Conn

	mu        sync.RWMutex
	pulse_us  float64
	period_us float64
	last_seen time.Time
	state_sub *nats.Subscription
}

// New creates a new Remote PWM Input component.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	res_conf := resource.NewConfig(conf)

	cfg, err := NewConfigFromResource(res_conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	name := "remote_pwm_input"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	logger := slog.Default()
	if logger_res, err := deps.Get("logger"); err == nil {
		if l, ok := logger_res.(*slog.Logger); ok {
			logger = l
		}
	}

	r := &RemotePWMInput{
		name:   resource.NewComponentName("gorai", "pwm_input", name),
		config: cfg,
		logger: logger.With("component", "remote_pwm_input", "name", name),
	}

	if nats_res, err := deps.Get("nats"); err == nil {
		if nc, ok := nats_res.(*nats.Conn); ok {
			r.nc = nc
		}
	}

	if r.nc == nil {
		return nil, fmt.Errorf("NATS connection not available")
	}

	r.logger.Info("remote PWM input component created",
		"device_id", cfg.DeviceID,
		"pin", cfg.Pin,
	)

	return r, nil
}

// Name returns the resource name.
func (r *RemotePWMInput) Name() resource.Name {
	return r.name
}

// Start provisions the pin on the remote device as PWM input and subscribes
// to PWM_INPUT_DATA messages.
func (r *RemotePWMInput) Start(ctx context.Context) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	r.subscribeState()

	r.logger.Info("provisioning PWM input pin",
		"pin", cfg.Pin,
		"pull", cfg.Pull,
	)

	gpio_cfg := gpioConfigPayload{
		Pin:  uint8(cfg.Pin),
		Mode: gpio_mode_pwm_input,
		Pull: cfg.PullID(),
	}
	if err := r.publishCommand("gpio_config", gpio_cfg); err != nil {
		return fmt.Errorf("failed to send GPIO_CONFIG: %w", err)
	}

	return nil
}

// subscribeState subscribes to PWM_INPUT_DATA messages from the device.
func (r *RemotePWMInput) subscribeState() {
	cfg := r.config
	subject := fmt.Sprintf("%s.%s.rx.sensor.pwm_input_data", cfg.NATSSubjectPrefix, cfg.DeviceID)
	r.logger.Info("subscribing to PWM_INPUT_DATA", "subject", subject)
	sub, err := r.nc.Subscribe(subject, func(msg *nats.Msg) {
		var env struct {
			Data json.RawMessage `json:"data"`
		}
		if err := json.Unmarshal(msg.Data, &env); err != nil {
			r.logger.Warn("failed to unmarshal PWM_INPUT_DATA envelope", "error", err)
			return
		}
		var data pwmInputDataPayload
		if err := json.Unmarshal(env.Data, &data); err != nil {
			r.logger.Warn("failed to unmarshal PWM_INPUT_DATA payload", "error", err)
			return
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, ch := range data.Channels {
			if int(ch.Channel) == cfg.Pin {
				r.pulse_us = float64(ch.PulseUs)
				r.period_us = float64(ch.PeriodUs)
				r.last_seen = time.Now()
				freq_hz := 0.0
				if ch.PeriodUs > 0 {
					freq_hz = 1_000_000.0 / float64(ch.PeriodUs)
				}
				r.logger.Debug("PWM_INPUT_DATA",
					"pin", cfg.Pin,
					"pulse_us", ch.PulseUs,
					"period_us", ch.PeriodUs,
					"freq_hz", fmt.Sprintf("%.1f", freq_hz),
				)
			}
		}
	})
	if err != nil {
		r.logger.Warn("failed to subscribe to PWM_INPUT_DATA", "error", err)
		return
	}
	r.mu.Lock()
	r.state_sub = sub
	r.mu.Unlock()
}

// GetPulse returns the most recently measured pulse width in microseconds.
func (r *RemotePWMInput) GetPulse(ctx context.Context) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.pulse_us, nil
}

// GetPeriod returns the most recently measured period in microseconds.
func (r *RemotePWMInput) GetPeriod(ctx context.Context) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.period_us, nil
}

// GetFrequency returns the derived frequency in Hz.
func (r *RemotePWMInput) GetFrequency(ctx context.Context) (float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.period_us == 0 {
		return 0, nil
	}
	return 1_000_000.0 / r.period_us, nil
}

// IsActive returns true if a valid measurement was received recently.
func (r *RemotePWMInput) IsActive(ctx context.Context) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.last_seen.IsZero() {
		return false, nil
	}
	return time.Since(r.last_seen) < stale_threshold, nil
}

// Properties returns the PWM input configuration.
func (r *RemotePWMInput) Properties(ctx context.Context) (pwm_input.Properties, error) {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	return pwm_input.Properties{
		Pin:      cfg.Pin,
		DeviceID: cfg.DeviceID,
		Mode:     "remote",
	}, nil
}

// Reconfigure updates the component configuration.
func (r *RemotePWMInput) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
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

// ResetDevice sends a full RESET command to the remote device, returning it
// to listening state with all subsystems cleared.
func (r *RemotePWMInput) ResetDevice(ctx context.Context) error {
	payload := resetPayload{Subsystem: 0}
	if err := r.publishCommand("reset", payload); err != nil {
		return fmt.Errorf("failed to send RESET: %w", err)
	}
	r.logger.Info("device reset sent", "device_id", r.config.DeviceID)
	return nil
}

// DoCommand handles arbitrary commands.
func (r *RemotePWMInput) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmd_name, _ := cmd["command"].(string)

	switch cmd_name {
	case "get_state":
		r.mu.RLock()
		defer r.mu.RUnlock()
		is_active := !r.last_seen.IsZero() && time.Since(r.last_seen) < stale_threshold
		return map[string]any{
			"pin":       r.config.Pin,
			"device_id": r.config.DeviceID,
			"pulse_us":  r.pulse_us,
			"period_us": r.period_us,
			"is_active": is_active,
		}, nil

	case "reset_device":
		return nil, r.ResetDevice(ctx)

	default:
		return nil, fmt.Errorf("unknown command: %s", cmd_name)
	}
}

// Close releases all resources.
func (r *RemotePWMInput) Close(ctx context.Context) error {
	r.mu.Lock()
	sub := r.state_sub
	r.state_sub = nil
	r.mu.Unlock()

	if sub != nil {
		_ = sub.Unsubscribe()
	}

	r.logger.Info("remote PWM input component closed")
	return nil
}

// publishCommand marshals the payload and publishes to the correct NATS subject.
func (r *RemotePWMInput) publishCommand(command_type string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	subject := r.config.CommandSubject(command_type)
	return r.nc.Publish(subject, data)
}

// Verify interface compliance at compile time.
var _ pwm_input.PWMInput = (*RemotePWMInput)(nil)
