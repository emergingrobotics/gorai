package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gorai/gorai/components/gpio"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterComponent("gpio", "remote", New)
}

// gpioConfigPayload matches the JSON format for GPIO_CONFIG.
type gpioConfigPayload struct {
	Pin     uint8 `json:"pin"`
	Mode    uint8 `json:"mode"`
	Pull    uint8 `json:"pull"`
	Options uint8 `json:"options"`
}

// gpioSetPayload matches the JSON format for GPIO_SET.
type gpioSetPayload struct {
	Pins []gpioSetEntry `json:"pins"`
}

type gpioSetEntry struct {
	Pin   uint8 `json:"pin"`
	Value uint8 `json:"value"`
}

// gpioQueryPayload matches the JSON format for GPIO_QUERY.
type gpioQueryPayload struct {
	Pin uint8 `json:"pin"`
}

// gpioStatePayload represents the device-reported GPIO state.
type gpioStatePayload struct {
	Pins []gpioStateEntry `json:"pins"`
}

type gpioStateEntry struct {
	Pin   uint8 `json:"pin"`
	Value uint8 `json:"value"`
	Mode  uint8 `json:"mode"`
}

// RemoteGPIO implements gpio.GPIO by publishing commands to NATS,
// which gorai-nats-gw bridges to a physical device via GSP/2.
type RemoteGPIO struct {
	name   resource.Name
	config *Config
	logger *slog.Logger
	nc     *nats.Conn

	mu            sync.RWMutex
	is_configured bool
	current_value uint8
	state_sub     *nats.Subscription
}

// New creates a new Remote GPIO component.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	res_conf := resource.NewConfig(conf)

	cfg, err := NewConfigFromResource(res_conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	name := "remote_gpio"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	logger := slog.Default()
	if logger_res, err := deps.Get("logger"); err == nil {
		if l, ok := logger_res.(*slog.Logger); ok {
			logger = l
		}
	}

	r := &RemoteGPIO{
		name:   resource.NewComponentName("gorai", "gpio", name),
		config: cfg,
		logger: logger.With("component", "remote_gpio", "name", name),
	}

	if nats_res, err := deps.Get("nats"); err == nil {
		if nc, ok := nats_res.(*nats.Conn); ok {
			r.nc = nc
		}
	}

	if r.nc == nil {
		return nil, fmt.Errorf("NATS connection not available")
	}

	r.logger.Info("remote GPIO component created",
		"device_id", cfg.DeviceID,
		"pin", cfg.Pin,
		"mode", cfg.Mode,
	)

	return r, nil
}

// Name returns the resource name.
func (r *RemoteGPIO) Name() resource.Name {
	return r.name
}

// Start provisions the pin on the remote device and subscribes to state changes.
func (r *RemoteGPIO) Start(ctx context.Context) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	r.subscribeState()

	r.logger.Info("provisioning GPIO pin",
		"pin", cfg.Pin,
		"mode", cfg.Mode,
		"pull", cfg.Pull,
	)

	gpio_cfg := gpioConfigPayload{
		Pin:  uint8(cfg.Pin),
		Mode: cfg.ModeID(),
		Pull: cfg.PullID(),
	}
	if err := r.publishCommand("gpio_config", gpio_cfg); err != nil {
		return fmt.Errorf("failed to send GPIO_CONFIG: %w", err)
	}

	r.mu.Lock()
	r.is_configured = true
	r.mu.Unlock()

	if cfg.Mode == "output" && cfg.InitialValue != 0 {
		if err := r.Set(ctx, cfg.InitialValue); err != nil {
			return fmt.Errorf("failed to set initial value: %w", err)
		}
	}

	return nil
}

// Set sets the output pin value.
func (r *RemoteGPIO) Set(ctx context.Context, value uint8) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	if cfg.Mode != "output" {
		return fmt.Errorf("cannot set value on input pin %d", cfg.Pin)
	}

	payload := gpioSetPayload{
		Pins: []gpioSetEntry{
			{Pin: uint8(cfg.Pin), Value: value},
		},
	}

	if err := r.publishCommand("gpio_set", payload); err != nil {
		return fmt.Errorf("failed to publish GPIO_SET: %w", err)
	}

	r.mu.Lock()
	if value <= 1 {
		r.current_value = value
	}
	r.mu.Unlock()

	r.logger.Debug("pin set", "pin", cfg.Pin, "value", value)
	return nil
}

// Get returns the current pin value.
func (r *RemoteGPIO) Get(ctx context.Context) (uint8, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.current_value, nil
}

// IsConfigured returns true if the pin has been provisioned.
func (r *RemoteGPIO) IsConfigured(ctx context.Context) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.is_configured, nil
}

// Properties returns the GPIO configuration.
func (r *RemoteGPIO) Properties(ctx context.Context) (gpio.Properties, error) {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	return gpio.Properties{
		Pin:      cfg.Pin,
		Mode:     cfg.Mode,
		Pull:     cfg.Pull,
		DeviceID: cfg.DeviceID,
	}, nil
}

// Reconfigure updates the component configuration.
func (r *RemoteGPIO) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
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

// DoCommand handles arbitrary commands.
func (r *RemoteGPIO) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmd_name, _ := cmd["command"].(string)

	switch cmd_name {
	case "get_state":
		r.mu.RLock()
		defer r.mu.RUnlock()
		return map[string]any{
			"pin":           r.config.Pin,
			"device_id":     r.config.DeviceID,
			"mode":          r.config.Mode,
			"current_value": r.current_value,
			"is_configured": r.is_configured,
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmd_name)
	}
}

// Close releases all resources.
func (r *RemoteGPIO) Close(ctx context.Context) error {
	r.mu.Lock()
	sub := r.state_sub
	r.state_sub = nil
	r.mu.Unlock()

	if sub != nil {
		_ = sub.Unsubscribe()
	}

	r.logger.Info("remote GPIO component closed")
	return nil
}

// subscribeState subscribes to GPIO_STATE messages from the device.
func (r *RemoteGPIO) subscribeState() {
	cfg := r.config
	subject := fmt.Sprintf("%s.%s.rx.response.gpio_state", cfg.NATSSubjectPrefix, cfg.DeviceID)
	r.logger.Info("subscribing to GPIO_STATE", "subject", subject)
	sub, err := r.nc.Subscribe(subject, func(msg *nats.Msg) {
		var state gpioStatePayload
		if err := json.Unmarshal(msg.Data, &state); err != nil {
			r.logger.Warn("failed to unmarshal GPIO_STATE", "error", err)
			return
		}
		r.mu.Lock()
		defer r.mu.Unlock()
		for _, entry := range state.Pins {
			if int(entry.Pin) == cfg.Pin {
				prev := r.current_value
				r.current_value = entry.Value
				if entry.Value != prev {
					r.logger.Info("GPIO_STATE changed",
						"pin", cfg.Pin,
						"value", entry.Value,
						"prev", prev,
					)
				} else {
					r.logger.Debug("GPIO_STATE",
						"pin", cfg.Pin,
						"value", entry.Value,
					)
				}
			}
		}
	})
	if err != nil {
		r.logger.Warn("failed to subscribe to GPIO_STATE", "error", err)
		return
	}
	r.mu.Lock()
	r.state_sub = sub
	r.mu.Unlock()
}

// publishCommand marshals the payload and publishes to the correct NATS subject.
func (r *RemoteGPIO) publishCommand(command_type string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}

	subject := r.config.CommandSubject(command_type)
	return r.nc.Publish(subject, data)
}

// Verify interface compliance at compile time.
var _ gpio.GPIO = (*RemoteGPIO)(nil)
