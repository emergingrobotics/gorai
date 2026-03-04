package remote

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/gorai/gorai/components/motor"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

func init() {
	registry.RegisterComponent("motor", "remote", New)
}

type motorSetPayload struct {
	Motors []motorSetEntry `json:"motors"`
}

type motorSetEntry struct {
	Motor uint8 `json:"motor"`
	Speed int16 `json:"speed"`
}

type motorEnablePayload struct {
	Motors []motorEnableEntry `json:"motors"`
}

type motorEnableEntry struct {
	Motor   uint8 `json:"motor"`
	Enabled bool  `json:"enabled"`
}

type motorConfigPayload struct {
	Motor    uint8  `json:"motor"`
	MaxRPM   uint16 `json:"max_rpm"`
	MaxAccel uint16 `json:"max_accel"`
	PPR      uint16 `json:"ppr"`
	GearRatio uint16 `json:"gear_ratio"`
	Flags    uint8  `json:"flags"`
}

type RemoteMotor struct {
	name   resource.Name
	config *Config
	logger *slog.Logger
	nc     *nats.Conn

	mu            sync.RWMutex
	is_powered    bool
	current_power float64
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

	name := "remote_motor"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	logger := slog.Default()
	if logger_res, err := deps.Get("logger"); err == nil {
		if l, ok := logger_res.(*slog.Logger); ok {
			logger = l
		}
	}

	r := &RemoteMotor{
		name:   resource.NewComponentName("gorai", "motor", name),
		config: cfg,
		logger: logger.With("component", "remote_motor", "name", name),
	}

	if nats_res, err := deps.Get("nats"); err == nil {
		if nc, ok := nats_res.(*nats.Conn); ok {
			r.nc = nc
		}
	}

	if r.nc == nil {
		return nil, fmt.Errorf("NATS connection not available")
	}

	r.logger.Info("remote motor component created",
		"device_id", cfg.DeviceID,
		"motor_index", cfg.MotorIndex,
	)

	return r, nil
}

func (r *RemoteMotor) Name() resource.Name {
	return r.name
}

func (r *RemoteMotor) Start(ctx context.Context) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	if !cfg.AutoConfigure {
		r.logger.Info("auto_configure disabled, skipping motor provisioning")
		return nil
	}

	r.logger.Info("provisioning motor",
		"motor_index", cfg.MotorIndex,
		"pwm_pin", cfg.PWMPin,
		"in1_pin", cfg.IN1Pin,
		"in2_pin", cfg.IN2Pin,
	)

	var flags uint8
	if cfg.Invert {
		flags |= 0x01
	}
	if cfg.BrakeOnStop {
		flags |= 0x02
	}

	// Encode pin mapping into ppr/gear_ratio fields:
	// ppr = (in1_pin << 8) | pwm_pin
	// gear_ratio = in2_pin
	ppr := (uint16(cfg.IN1Pin) << 8) | uint16(cfg.PWMPin)

	motor_cfg := motorConfigPayload{
		Motor:     uint8(cfg.MotorIndex),
		MaxRPM:    cfg.MaxRPM,
		MaxAccel:  0,
		PPR:       ppr,
		GearRatio: uint16(cfg.IN2Pin),
		Flags:     flags,
	}

	if err := r.publishCommand("motor_config", motor_cfg); err != nil {
		return fmt.Errorf("failed to send MOTOR_CONFIG: %w", err)
	}

	enable_payload := motorEnablePayload{
		Motors: []motorEnableEntry{
			{Motor: uint8(cfg.MotorIndex), Enabled: true},
		},
	}
	if err := r.publishCommand("motor_enable", enable_payload); err != nil {
		return fmt.Errorf("failed to send MOTOR_ENABLE: %w", err)
	}

	return nil
}

func (r *RemoteMotor) SetPower(ctx context.Context, power float64) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	power = clamp(power, -1.0, 1.0)
	speed := int16(power * float64(cfg.MaxSpeed))

	payload := motorSetPayload{
		Motors: []motorSetEntry{
			{Motor: uint8(cfg.MotorIndex), Speed: speed},
		},
	}

	if err := r.publishCommand("motor_set", payload); err != nil {
		return fmt.Errorf("failed to publish MOTOR_SET: %w", err)
	}

	r.mu.Lock()
	r.current_power = power
	r.is_powered = power != 0
	r.mu.Unlock()

	r.logger.Debug("motor power set", "motor_index", cfg.MotorIndex, "power", power, "speed", speed)
	return nil
}

func (r *RemoteMotor) IsMoving(ctx context.Context) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.is_powered, nil
}

func (r *RemoteMotor) Stop(ctx context.Context) error {
	return r.SetPower(ctx, 0)
}

func (r *RemoteMotor) SetVelocity(ctx context.Context, velocity float64) error {
	return fmt.Errorf("SetVelocity not supported in open-loop mode")
}

func (r *RemoteMotor) GoTo(ctx context.Context, position, velocity float64) error {
	return fmt.Errorf("GoTo not supported in open-loop mode")
}

func (r *RemoteMotor) GoFor(ctx context.Context, rpm, revolutions float64) error {
	return fmt.Errorf("GoFor not supported in open-loop mode")
}

func (r *RemoteMotor) GetPosition(ctx context.Context) (float64, error) {
	return 0, nil
}

func (r *RemoteMotor) GetVelocity(ctx context.Context) (float64, error) {
	return 0, nil
}

func (r *RemoteMotor) ResetZeroPosition(ctx context.Context, offset float64) error {
	return nil
}

func (r *RemoteMotor) IsPowered(ctx context.Context) (bool, float64, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.is_powered, math.Abs(r.current_power), nil
}

func (r *RemoteMotor) Properties(ctx context.Context) (motor.Properties, error) {
	return motor.Properties{
		PositionReporting: false,
		VelocityReporting: false,
		SupportsGoTo:      false,
	}, nil
}

func (r *RemoteMotor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
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

func (r *RemoteMotor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmd_name, _ := cmd["command"].(string)
	switch cmd_name {
	case "get_state":
		r.mu.RLock()
		defer r.mu.RUnlock()
		return map[string]any{
			"motor_index":   r.config.MotorIndex,
			"device_id":     r.config.DeviceID,
			"current_power": r.current_power,
			"is_powered":    r.is_powered,
		}, nil
	default:
		return nil, fmt.Errorf("unknown command: %s", cmd_name)
	}
}

func (r *RemoteMotor) Close(ctx context.Context) error {
	r.mu.RLock()
	cfg := r.config
	r.mu.RUnlock()

	stop_payload := motorSetPayload{
		Motors: []motorSetEntry{
			{Motor: uint8(cfg.MotorIndex), Speed: 0},
		},
	}
	_ = r.publishCommand("motor_set", stop_payload)

	disable_payload := motorEnablePayload{
		Motors: []motorEnableEntry{
			{Motor: uint8(cfg.MotorIndex), Enabled: false},
		},
	}
	_ = r.publishCommand("motor_enable", disable_payload)

	r.logger.Info("remote motor component closed")
	return nil
}

func (r *RemoteMotor) publishCommand(command_type string, payload any) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal payload: %w", err)
	}
	subject := r.config.CommandSubject(command_type)
	return r.nc.Publish(subject, data)
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

var _ motor.Motor = (*RemoteMotor)(nil)
