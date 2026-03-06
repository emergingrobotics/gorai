// Package l298n provides a motor controller service for L298N H-bridge drivers.
//
// The service subscribes to motor power commands on NATS topics and translates
// them into raw GPIO (direction) and PWM (speed) signals sent to the Pico via
// the existing gorai-nats-gw bridge. Motor control intelligence lives here in
// Go rather than in Pico firmware.
package l298n

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gorai/gorai/components/pwm"
	pwm_remote "github.com/gorai/gorai/components/pwm/remote"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterService("control", "l298n", New)
}

type gpioConfigPayload struct {
	Pin     uint8 `json:"pin"`
	Mode    uint8 `json:"mode"`
	Pull    uint8 `json:"pull"`
	Options uint8 `json:"options"`
}

type gpioSetPayload struct {
	Pins []gpioSetEntry `json:"pins"`
}

type gpioSetEntry struct {
	Pin   uint8 `json:"pin"`
	Value uint8 `json:"value"`
}

type motorPowerMessage struct {
	Power float64 `json:"power"`
}

// simpleDeps adapts a NATS connection and logger into the registry.Dependencies
// interface so we can call pwm remote.New() internally.
type simpleDeps struct {
	nc     *nats.Conn
	logger *slog.Logger
}

func (d *simpleDeps) Get(name string) (any, error) {
	switch name {
	case "nats":
		return d.nc, nil
	case "logger":
		return d.logger, nil
	default:
		return nil, fmt.Errorf("unknown dependency: %s", name)
	}
}

func (d *simpleDeps) GetByType(subtype string) ([]any, error) {
	return nil, fmt.Errorf("not supported")
}

type motorBinding struct {
	def          MotorDef
	pwm_instance pwm.PWM
	sub          *nats.Subscription

	last_in1  uint8
	last_in2  uint8
	last_duty float64
	has_state bool
}

type Controller struct {
	name   resource.Name
	config *Config
	logger *slog.Logger
	nc     *nats.Conn

	mu       sync.RWMutex
	bindings map[string]*motorBinding
	running  bool
}

func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	res_conf := resource.NewConfig(conf)

	cfg, err := NewConfigFromResource(res_conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	name := "l298n_controller"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	logger := slog.Default()
	if deps != nil {
		if logger_val, err := deps.Get("logger"); err == nil {
			if l, ok := logger_val.(*slog.Logger); ok && l != nil {
				logger = l
			}
		}
	}
	logger = logger.With("service", "l298n_controller", "name", name)

	c := &Controller{
		name:     resource.NewServiceName("gorai", "control", name),
		config:   cfg,
		logger:   logger,
		bindings: make(map[string]*motorBinding),
	}

	if deps != nil {
		if nats_val, err := deps.Get("nats"); err == nil {
			if nc, ok := nats_val.(*nats.Conn); ok {
				c.nc = nc
			}
		}
	}

	motor_names := make([]string, len(cfg.Motors))
	for i, m := range cfg.Motors {
		motor_names[i] = m.Name
	}
	c.logger.Info("l298n controller created", "motors", motor_names)

	return c, nil
}

func (c *Controller) Name() resource.Name {
	return c.name
}

func (c *Controller) Start(ctx context.Context) error {
	return nil
}

func (c *Controller) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	c.stopAll()

	bindings := make(map[string]*motorBinding)
	for _, motor_def := range cfg.Motors {
		pwm_inst, err := c.createPWMInstance(ctx, cfg, motor_def)
		if err != nil {
			return fmt.Errorf("motor %q: failed to create PWM instance: %w",
				motor_def.Name, err)
		}
		bindings[motor_def.Name] = &motorBinding{
			def:          motor_def,
			pwm_instance: pwm_inst,
		}
	}

	c.mu.Lock()
	c.config = cfg
	c.bindings = bindings
	c.mu.Unlock()

	if err := c.configureDirectionPins(); err != nil {
		return fmt.Errorf("failed to configure direction pins: %w", err)
	}

	if err := c.subscribeAll(); err != nil {
		return fmt.Errorf("failed to subscribe to motor topics: %w", err)
	}

	c.logger.Info("l298n controller reconfigured")
	return nil
}

// createPWMInstance builds a remote.RemotePWM internally for a motor's speed pin.
func (c *Controller) createPWMInstance(ctx context.Context, cfg *Config, motor_def MotorDef) (pwm.PWM, error) {
	period_us := cfg.PeriodUs()
	pwm_conf := registry.Config{
		"name":                fmt.Sprintf("%s_speed", motor_def.Name),
		"nats_subject_prefix": cfg.NATSSubjectPrefix,
		"device_id":           cfg.DeviceID,
		"pin":                 float64(motor_def.SpeedPin),
		"frequency_hz":        float64(cfg.PWMFrequencyHz),
		"min_pulse_us":        float64(0),
		"max_pulse_us":        period_us,
		"initial_pulse_us":    float64(0),
		"failsafe_pulse_us":   float64(0),
		"auto_configure":      true,
	}

	deps := &simpleDeps{nc: c.nc, logger: c.logger}

	instance, err := pwm_remote.New(ctx, deps, pwm_conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create PWM for pin %d: %w", motor_def.SpeedPin, err)
	}

	pwm_comp, ok := instance.(pwm.PWM)
	if !ok {
		return nil, fmt.Errorf("PWM instance for pin %d does not implement pwm.PWM", motor_def.SpeedPin)
	}

	type startable interface {
		Start(context.Context) error
	}
	if s, ok := instance.(startable); ok {
		if err := s.Start(ctx); err != nil {
			return nil, fmt.Errorf("failed to start PWM for pin %d: %w", motor_def.SpeedPin, err)
		}
	}

	c.logger.Debug("created internal PWM instance",
		"motor", motor_def.Name, "speed_pin", motor_def.SpeedPin,
		"in1_pin", motor_def.IN1Pin, "in2_pin", motor_def.IN2Pin,
		"invert", motor_def.Invert, "brake_on_stop", motor_def.BrakeOnStop,
		"frequency_hz", cfg.PWMFrequencyHz, "period_us", period_us)
	return pwm_comp, nil
}

// configureDirectionPins sends GPIO_CONFIG (output mode) for all IN1/IN2 pins
// and sets them to low (motor off).
func (c *Controller) configureDirectionPins() error {
	c.mu.RLock()
	cfg := c.config
	bindings := c.bindings
	c.mu.RUnlock()

	if c.nc == nil {
		return fmt.Errorf("NATS connection not available")
	}

	for _, b := range bindings {
		for _, pin := range []uint8{b.def.IN1Pin, b.def.IN2Pin} {
			gpio_cfg := gpioConfigPayload{
				Pin:  pin,
				Mode: 0x01, // output
				Pull: 0x00, // none
			}
			if err := c.publishCommand(cfg.CommandSubject("gpio_config"), gpio_cfg); err != nil {
				return fmt.Errorf("motor %q: failed to configure pin %d: %w",
					b.def.Name, pin, err)
			}
		}

		if err := c.setDirectionPins(b.def, 0, 0); err != nil {
			return fmt.Errorf("motor %q: failed to set initial pin state: %w",
				b.def.Name, err)
		}
	}

	return nil
}

func (c *Controller) subscribeAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.nc == nil {
		return fmt.Errorf("NATS connection not available")
	}

	for name, b := range c.bindings {
		binding := b
		sub, err := c.nc.Subscribe(binding.def.MotorTopic, func(msg *nats.Msg) {
			c.handleMotorCommand(binding, msg)
		})
		if err != nil {
			return fmt.Errorf("motor %q: failed to subscribe to %s: %w",
				name, binding.def.MotorTopic, err)
		}
		binding.sub = sub
		c.logger.Debug("subscribed to motor topic",
			"motor", name, "topic", binding.def.MotorTopic)
	}

	c.running = true
	return nil
}

func (c *Controller) handleMotorCommand(binding *motorBinding, msg *nats.Msg) {
	var cmd motorPowerMessage
	if err := json.Unmarshal(msg.Data, &cmd); err != nil {
		c.logger.Warn("failed to unmarshal motor command",
			"motor", binding.def.Name, "error", err)
		return
	}

	power := clamp(cmd.Power, -1.0, 1.0)
	def := binding.def

	if def.Invert {
		power = -power
	}

	var in1, in2 uint8
	switch {
	case power > 0:
		in1 = 1
		in2 = 0
	case power < 0:
		in1 = 0
		in2 = 1
	default:
		if def.BrakeOnStop {
			in1 = 1
			in2 = 1
		} else {
			in1 = 0
			in2 = 0
		}
	}

	abs_power := power
	if abs_power < 0 {
		abs_power = -abs_power
	}

	dir_changed := !binding.has_state || binding.last_in1 != in1 || binding.last_in2 != in2
	duty_changed := !binding.has_state || binding.last_duty != abs_power

	if !dir_changed && !duty_changed {
		return
	}

	if dir_changed {
		if err := c.setDirectionPins(def, in1, in2); err != nil {
			c.logger.Error("failed to set direction",
				"motor", def.Name, "error", err)
			return
		}
	}

	if duty_changed {
		ctx := context.Background()
		if err := binding.pwm_instance.SetDuty(ctx, abs_power); err != nil {
			c.logger.Error("failed to set speed",
				"motor", def.Name, "error", err)
			return
		}
	}

	binding.last_in1 = in1
	binding.last_in2 = in2
	binding.last_duty = abs_power
	binding.has_state = true

	c.logger.Debug("motor command applied",
		"motor", def.Name, "power", cmd.Power, "invert", def.Invert,
		"in1", in1, "in2", in2, "duty", abs_power)
}

func (c *Controller) setDirectionPins(def MotorDef, in1, in2 uint8) error {
	c.mu.RLock()
	cfg := c.config
	c.mu.RUnlock()

	payload := gpioSetPayload{
		Pins: []gpioSetEntry{
			{Pin: def.IN1Pin, Value: in1},
			{Pin: def.IN2Pin, Value: in2},
		},
	}
	return c.publishCommand(cfg.CommandSubject("gpio_set"), payload)
}

func (c *Controller) stopAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for _, b := range c.bindings {
		if b.sub != nil {
			_ = b.sub.Unsubscribe()
			b.sub = nil
		}
	}

	ctx := context.Background()
	if c.running && c.nc != nil {
		for _, b := range c.bindings {
			_ = c.setDirectionPinsLocked(b.def, 0, 0)
			_ = b.pwm_instance.SetDuty(ctx, 0.0)
		}
	}

	for _, b := range c.bindings {
		if b.pwm_instance != nil {
			_ = b.pwm_instance.Close(ctx)
			b.pwm_instance = nil
		}
	}

	c.running = false
}

func (c *Controller) setDirectionPinsLocked(def MotorDef, in1, in2 uint8) error {
	payload := gpioSetPayload{
		Pins: []gpioSetEntry{
			{Pin: def.IN1Pin, Value: in1},
			{Pin: def.IN2Pin, Value: in2},
		},
	}
	return c.publishCommand(c.config.CommandSubject("gpio_set"), payload)
}

func (c *Controller) publishCommand(subject string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	return c.nc.Publish(subject, data)
}

func (c *Controller) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmd_name, _ := cmd["command"].(string)
	switch cmd_name {
	case "stop":
		c.stopAll()
		return map[string]any{"success": true}, nil
	default:
		return nil, fmt.Errorf("unknown command: %s", cmd_name)
	}
}

func (c *Controller) Close(ctx context.Context) error {
	c.stopAll()
	c.logger.Info("l298n controller closed")
	return nil
}

func clamp(value, min_val, max_val float64) float64 {
	if value < min_val {
		return min_val
	}
	if value > max_val {
		return max_val
	}
	return value
}
