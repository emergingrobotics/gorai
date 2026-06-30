package velocity_input

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"strings"
	"sync"

	"github.com/emergingrobotics/gorai/components/input"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterService("control", "velocity_input", New)
}

type VelocityBinding struct {
	VX    float64 `json:"vx"`
	VY    float64 `json:"vy"`
	Omega float64 `json:"omega"`
}

type VelocityCommand struct {
	VX       float64 `json:"vx"`
	VY       float64 `json:"vy"`
	Omega    float64 `json:"omega"`
	SetSpeed float64 `json:"set_speed,omitempty"`
}

// KeyboardConfig holds keyboard-specific input config.
type KeyboardConfig struct {
	InputComponent string                     `json:"input_component"`
	SetSpeed       float64                    `json:"set_speed"`
	KeyBindings   map[string]VelocityBinding `json:"key_bindings"`
}

type Config struct {
	VelocityTopic string         `json:"velocity_topic"`
	InputType     string         `json:"input_type"`
	Keyboard      KeyboardConfig `json:"keyboard"`
}

func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		InputType: "keyboard",
		Keyboard: KeyboardConfig{
			SetSpeed:    0.25,
			KeyBindings: make(map[string]VelocityBinding),
		},
	}

	if v, ok := conf.GetString("velocity_topic"); ok {
		cfg.VelocityTopic = v
	}
	if v, ok := conf.GetString("input_type"); ok {
		cfg.InputType = v
	}

	keyboard_raw, ok := conf.Get("keyboard")
	if !ok {
		return cfg, nil
	}
	keyboard_map, ok := keyboard_raw.(map[string]any)
	if !ok {
		return cfg, nil
	}

	if v, ok := keyboard_map["input_component"].(string); ok {
		cfg.Keyboard.InputComponent = v
	}
	if v, ok := keyboard_map["set_speed"].(float64); ok {
		cfg.Keyboard.SetSpeed = v
	}
	if bindings_raw, ok := keyboard_map["key_bindings"]; ok {
		if bindings_map, ok := bindings_raw.(map[string]any); ok {
			for key, val := range bindings_map {
				if binding_map, ok := val.(map[string]any); ok {
					binding := VelocityBinding{}
					if vx, ok := binding_map["vx"].(float64); ok {
						binding.VX = vx
					}
					if vy, ok := binding_map["vy"].(float64); ok {
						binding.VY = vy
					}
					if omega, ok := binding_map["omega"].(float64); ok {
						binding.Omega = omega
					}
					cfg.Keyboard.KeyBindings[strings.ToUpper(key)] = binding
				}
			}
		}
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.VelocityTopic == "" {
		return fmt.Errorf("velocity_topic is required")
	}
	if c.InputType != "keyboard" {
		return fmt.Errorf("input_type %q not supported, only keyboard for now", c.InputType)
	}
	if c.Keyboard.InputComponent == "" {
		return fmt.Errorf("keyboard.input_component is required")
	}
	if len(c.Keyboard.KeyBindings) == 0 {
		return fmt.Errorf("keyboard.key_bindings must have at least one binding")
	}
	if c.Keyboard.SetSpeed < 0 || c.Keyboard.SetSpeed > 1.0 {
		return fmt.Errorf("keyboard.set_speed must be in [0, 1], got %f", c.Keyboard.SetSpeed)
	}
	return nil
}

type Controller struct {
	name   resource.Name
	config *Config
	logger *slog.Logger
	nc     *nats.Conn

	mu             sync.RWMutex
	keyboard       input.Keyboard
	active_keys    map[string]struct{}
	last_published VelocityCommand
	has_published  bool
	stop_ch        chan struct{}
	done_ch        chan struct{}
	running        bool
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

	name := "velocity_input"
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
	logger = logger.With("service", "velocity_input", "name", name)

	c := &Controller{
		name:        resource.NewServiceName("gorai", "control", name),
		config:      cfg,
		logger:      logger,
		active_keys: make(map[string]struct{}),
	}

	if deps != nil {
		if nats_val, err := deps.Get("nats"); err == nil {
			if nc, ok := nats_val.(*nats.Conn); ok {
				c.nc = nc
			}
		}
	}

	c.logger.Info("velocity input service created",
		"input_type", cfg.InputType,
		"velocity_topic", cfg.VelocityTopic,
		"bindings", len(cfg.Keyboard.KeyBindings),
		"set_speed", cfg.Keyboard.SetSpeed,
	)

	return c, nil
}

func (c *Controller) Name() resource.Name {
	return c.name
}

func (c *Controller) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	c.stopEventLoop()

	kb_name := resource.NewComponentName("gorai", "input", cfg.Keyboard.InputComponent)
	kb_res, err := deps.Get(kb_name)
	if err != nil {
		return fmt.Errorf("keyboard component %q not found: %w", cfg.Keyboard.InputComponent, err)
	}
	keyboard, ok := kb_res.(input.Keyboard)
	if !ok {
		return fmt.Errorf("component %q is not a keyboard", cfg.Keyboard.InputComponent)
	}

	c.mu.Lock()
	c.config = cfg
	c.keyboard = keyboard
	c.active_keys = make(map[string]struct{})
	c.has_published = false
	c.mu.Unlock()

	if err := c.startEventLoop(ctx); err != nil {
		return fmt.Errorf("failed to start event loop: %w", err)
	}

	c.logger.Info("velocity input service reconfigured")
	return nil
}

func (c *Controller) startEventLoop(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return nil
	}
	if c.keyboard == nil {
		c.mu.Unlock()
		return fmt.Errorf("keyboard not configured")
	}
	c.mu.Unlock()

	events_ch, err := c.keyboard.Events(ctx)
	if err != nil {
		return fmt.Errorf("failed to get keyboard events: %w", err)
	}

	c.mu.Lock()
	c.stop_ch = make(chan struct{})
	c.done_ch = make(chan struct{})
	c.running = true
	c.mu.Unlock()

	go c.eventLoop(events_ch)
	c.logger.Info("velocity input event loop started")
	return nil
}

func (c *Controller) stopEventLoop() {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return
	}
	c.running = false
	close(c.stop_ch)
	done := c.done_ch
	c.mu.Unlock()

	if done != nil {
		<-done
	}
}

func (c *Controller) eventLoop(events_ch <-chan input.KeyEvent) {
	defer close(c.done_ch)

	for {
		select {
		case <-c.stop_ch:
			return
		case event, ok := <-events_ch:
			if !ok {
				c.logger.Error("keyboard event channel closed")
				return
			}
			c.processKeyEvent(event)
		}
	}
}

func (c *Controller) processKeyEvent(event input.KeyEvent) {
	normalized_key := strings.ToUpper(event.Key)

	c.mu.Lock()
	cfg := c.config
	_, is_bound := cfg.Keyboard.KeyBindings[normalized_key]
	if !is_bound {
		c.mu.Unlock()
		return
	}

	if event.Pressed {
		c.active_keys[normalized_key] = struct{}{}
	} else {
		delete(c.active_keys, normalized_key)
	}

	cmd := c.computeVelocity()

	if c.has_published && cmd == c.last_published {
		c.mu.Unlock()
		return
	}
	c.last_published = cmd
	c.has_published = true
	c.mu.Unlock()

	c.publishVelocity(cmd)
}

// computeVelocity sums direction from active keys, normalizes to unit vector, and
// sets set_speed from config when any keys are active (0 when none).
// Must be called while holding c.mu (at least RLock).
func (c *Controller) computeVelocity() VelocityCommand {
	var vx, vy, omega float64

	for key := range c.active_keys {
		if binding, ok := c.config.Keyboard.KeyBindings[key]; ok {
			vx += binding.VX
			vy += binding.VY
			omega += binding.Omega
		}
	}

	set_speed := 0.0
	if len(c.active_keys) > 0 {
		set_speed = c.config.Keyboard.SetSpeed
	}

	mag := vx*vx + vy*vy + omega*omega
	if mag > 0 {
		mag = math.Sqrt(mag)
		vx /= mag
		vy /= mag
		omega /= mag
	}

	return VelocityCommand{
		VX:       vx,
		VY:       vy,
		Omega:    omega,
		SetSpeed: set_speed,
	}
}

func (c *Controller) publishVelocity(cmd VelocityCommand) {
	if c.nc == nil {
		return
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		c.logger.Error("failed to marshal velocity command", "error", err)
		return
	}

	topic := c.config.VelocityTopic
	if err := c.nc.Publish(topic, data); err != nil {
		c.logger.Error("failed to publish velocity command", "error", err, "topic", topic)
		return
	}

	c.logger.Debug("velocity command published",
		"topic", topic, "vx", cmd.VX, "vy", cmd.VY, "omega", cmd.Omega, "set_speed", cmd.SetSpeed)
}

func (c *Controller) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmd_name, _ := cmd["command"].(string)
	switch cmd_name {
	case "get_state":
		c.mu.RLock()
		defer c.mu.RUnlock()
		active := make([]string, 0, len(c.active_keys))
		for k := range c.active_keys {
			active = append(active, k)
		}
		return map[string]any{
			"active_keys": active,
			"running":     c.running,
		}, nil
	case "stop":
		c.mu.Lock()
		c.active_keys = make(map[string]struct{})
		c.mu.Unlock()
		c.publishVelocity(VelocityCommand{})
		return map[string]any{"success": true}, nil
	default:
		return nil, fmt.Errorf("unknown command: %s", cmd_name)
	}
}

func (c *Controller) Close(ctx context.Context) error {
	c.stopEventLoop()
	c.publishVelocity(VelocityCommand{})
	c.logger.Info("velocity input service closed")
	return nil
}
