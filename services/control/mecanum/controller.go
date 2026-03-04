package mecanum

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gorai/gorai/components/motor"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterService("control", "mecanum_controller", New)
}

type VelocityCommand struct {
	VX    float64 `json:"vx"`
	VY    float64 `json:"vy"`
	Omega float64 `json:"omega"`
}

type Config struct {
	VelocityTopic string  `json:"velocity_topic"`
	MotorFLName   string  `json:"motor_fl"`
	MotorFRName   string  `json:"motor_fr"`
	MotorRLName   string  `json:"motor_rl"`
	MotorRRName   string  `json:"motor_rr"`
	WheelBaseX    float64 `json:"wheel_base_x"`
	WheelBaseY    float64 `json:"wheel_base_y"`
}

func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		WheelBaseX: 0.1,
		WheelBaseY: 0.075,
	}

	if v, ok := conf.GetString("velocity_topic"); ok {
		cfg.VelocityTopic = v
	}
	if v, ok := conf.GetString("motor_fl"); ok {
		cfg.MotorFLName = v
	}
	if v, ok := conf.GetString("motor_fr"); ok {
		cfg.MotorFRName = v
	}
	if v, ok := conf.GetString("motor_rl"); ok {
		cfg.MotorRLName = v
	}
	if v, ok := conf.GetString("motor_rr"); ok {
		cfg.MotorRRName = v
	}
	if v, ok := conf.GetFloat("wheel_base_x"); ok {
		cfg.WheelBaseX = v
	}
	if v, ok := conf.GetFloat("wheel_base_y"); ok {
		cfg.WheelBaseY = v
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.VelocityTopic == "" {
		return fmt.Errorf("velocity_topic is required")
	}
	if c.MotorFLName == "" || c.MotorFRName == "" ||
		c.MotorRLName == "" || c.MotorRRName == "" {
		return fmt.Errorf("all four motor names are required (motor_fl, motor_fr, motor_rl, motor_rr)")
	}
	if c.WheelBaseX <= 0 || c.WheelBaseY <= 0 {
		return fmt.Errorf("wheel_base_x and wheel_base_y must be positive")
	}
	return nil
}

type Controller struct {
	name   resource.Name
	config *Config
	logger *slog.Logger
	nc     *nats.Conn

	mu       sync.RWMutex
	motor_fl motor.Motor
	motor_fr motor.Motor
	motor_rl motor.Motor
	motor_rr motor.Motor

	vel_sub *nats.Subscription
	stop_ch chan struct{}
	done_ch chan struct{}
	running bool
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

	name := "mecanum_controller"
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
	logger = logger.With("service", "mecanum_controller", "name", name)

	c := &Controller{
		name:   resource.NewServiceName("gorai", "control", name),
		config: cfg,
		logger: logger,
	}

	if deps != nil {
		if nats_val, err := deps.Get("nats"); err == nil {
			if nc, ok := nats_val.(*nats.Conn); ok {
				c.nc = nc
			}
		}
	}

	c.logger.Info("mecanum controller created",
		"velocity_topic", cfg.VelocityTopic,
		"motors", fmt.Sprintf("[%s, %s, %s, %s]",
			cfg.MotorFLName, cfg.MotorFRName, cfg.MotorRLName, cfg.MotorRRName),
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

	c.stopSubscription()

	motor_fl, err := resolveMotor(deps, cfg.MotorFLName)
	if err != nil {
		return fmt.Errorf("motor_fl: %w", err)
	}
	motor_fr, err := resolveMotor(deps, cfg.MotorFRName)
	if err != nil {
		return fmt.Errorf("motor_fr: %w", err)
	}
	motor_rl, err := resolveMotor(deps, cfg.MotorRLName)
	if err != nil {
		return fmt.Errorf("motor_rl: %w", err)
	}
	motor_rr, err := resolveMotor(deps, cfg.MotorRRName)
	if err != nil {
		return fmt.Errorf("motor_rr: %w", err)
	}

	c.mu.Lock()
	c.config = cfg
	c.motor_fl = motor_fl
	c.motor_fr = motor_fr
	c.motor_rl = motor_rl
	c.motor_rr = motor_rr
	c.mu.Unlock()

	if c.nc != nil {
		if err := c.startSubscription(); err != nil {
			return fmt.Errorf("failed to start NATS subscription: %w", err)
		}
	}

	c.logger.Info("mecanum controller reconfigured")
	return nil
}

func resolveMotor(deps resource.Dependencies, name string) (motor.Motor, error) {
	motor_name := resource.NewComponentName("gorai", "motor", name)
	res, err := deps.Get(motor_name)
	if err != nil {
		return nil, fmt.Errorf("motor %q not found: %w", name, err)
	}
	m, ok := res.(motor.Motor)
	if !ok {
		return nil, fmt.Errorf("component %q is not a motor", name)
	}
	return m, nil
}

func (c *Controller) startSubscription() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.running {
		return nil
	}

	topic := c.config.VelocityTopic
	sub, err := c.nc.Subscribe(topic, func(msg *nats.Msg) {
		var cmd VelocityCommand
		if err := json.Unmarshal(msg.Data, &cmd); err != nil {
			c.logger.Warn("failed to unmarshal velocity command", "error", err)
			return
		}
		c.handleVelocityCommand(cmd)
	})
	if err != nil {
		return err
	}

	c.vel_sub = sub
	c.running = true
	c.logger.Info("subscribed to velocity commands", "topic", topic)
	return nil
}

func (c *Controller) stopSubscription() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.vel_sub != nil {
		_ = c.vel_sub.Unsubscribe()
		c.vel_sub = nil
	}
	c.running = false
}

func (c *Controller) handleVelocityCommand(cmd VelocityCommand) {
	c.mu.RLock()
	cfg := c.config
	fl := c.motor_fl
	fr := c.motor_fr
	rl := c.motor_rl
	rr := c.motor_rr
	c.mu.RUnlock()

	if fl == nil || fr == nil || rl == nil || rr == nil {
		c.logger.Warn("motors not configured, ignoring velocity command")
		return
	}

	ws := InverseKinematics(cmd.VX, cmd.VY, cmd.Omega, cfg.WheelBaseX, cfg.WheelBaseY)

	ctx := context.Background()

	c.logger.Debug("velocity command received",
		"vx", cmd.VX, "vy", cmd.VY, "omega", cmd.Omega,
		"fl", ws.FL, "fr", ws.FR, "rl", ws.RL, "rr", ws.RR,
	)

	if err := fl.SetPower(ctx, ws.FL); err != nil {
		c.logger.Error("failed to set FL motor power", "error", err)
	}
	if err := fr.SetPower(ctx, ws.FR); err != nil {
		c.logger.Error("failed to set FR motor power", "error", err)
	}
	if err := rl.SetPower(ctx, ws.RL); err != nil {
		c.logger.Error("failed to set RL motor power", "error", err)
	}
	if err := rr.SetPower(ctx, ws.RR); err != nil {
		c.logger.Error("failed to set RR motor power", "error", err)
	}
}

func (c *Controller) stopAllMotors() {
	c.mu.RLock()
	motors := []motor.Motor{c.motor_fl, c.motor_fr, c.motor_rl, c.motor_rr}
	c.mu.RUnlock()

	ctx := context.Background()
	for _, m := range motors {
		if m != nil {
			_ = m.SetPower(ctx, 0)
		}
	}
}

func (c *Controller) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmd_name, _ := cmd["command"].(string)
	switch cmd_name {
	case "stop":
		c.stopAllMotors()
		return map[string]any{"success": true}, nil
	case "set_velocity":
		vx, _ := cmd["vx"].(float64)
		vy, _ := cmd["vy"].(float64)
		omega, _ := cmd["omega"].(float64)
		c.handleVelocityCommand(VelocityCommand{VX: vx, VY: vy, Omega: omega})
		return map[string]any{"success": true}, nil
	default:
		return nil, fmt.Errorf("unknown command: %s", cmd_name)
	}
}

func (c *Controller) Close(ctx context.Context) error {
	c.stopSubscription()
	c.stopAllMotors()
	c.logger.Info("mecanum controller closed")
	return nil
}
