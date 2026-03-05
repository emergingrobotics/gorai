package keypress_motor

import (
	"fmt"

	"github.com/gorai/gorai/pkg/resource"
)

// DriveType defines the type of drive signal used to control the component.
type DriveType string

const (
	// DriveTypePWM controls components via PWM signal.
	DriveTypePWM DriveType = "pwm"
	// Future: DriveTypeI2C, DriveTypeSerial, etc.
)

// BehaviorType defines how input maps to output.
type BehaviorType string

const (
	// BehaviorTypeAngle controls position via angle (servo).
	BehaviorTypeAngle BehaviorType = "angle"
	// BehaviorTypeContinuous controls speed bidirectionally (continuous servo).
	BehaviorTypeContinuous BehaviorType = "continuous"
	// BehaviorTypeDC controls speed bidirectionally (DC motor).
	BehaviorTypeDC BehaviorType = "dc"
)

// AngleBehaviorConfig holds parameters for angle servo behavior.
type AngleBehaviorConfig struct {
	// AngleStep is the degrees per keypress.
	AngleStep float64 `json:"angle_step"`
	// MinAngle is the minimum angle in degrees.
	MinAngle float64 `json:"min_angle"`
	// MaxAngle is the maximum angle in degrees.
	MaxAngle float64 `json:"max_angle"`
	// InitialAngle is the starting angle in degrees.
	InitialAngle float64 `json:"initial_angle"`
	// Speed is the movement speed multiplier (0.0-1.0).
	Speed float64 `json:"speed"`
}

// ContinuousBehaviorConfig holds parameters for continuous servo behavior.
type ContinuousBehaviorConfig struct {
	// Speed is the max speed (0.0-1.0).
	Speed float64 `json:"speed"`
	// InitialSpeed is the starting speed (-1.0 to 1.0, default 0.0).
	InitialSpeed float64 `json:"initial_speed"`
}

// DCBehaviorConfig holds parameters for DC motor behavior.
type DCBehaviorConfig struct {
	// Speed is the max speed (0.0-1.0).
	Speed float64 `json:"speed"`
	// InitialSpeed is the starting speed (-1.0 to 1.0, default 0.0).
	InitialSpeed float64 `json:"initial_speed"`
}

// BehaviorConfig wraps behavior-specific configuration.
type BehaviorConfig struct {
	// Type is the behavior type: "angle", "continuous", or "dc".
	Type BehaviorType `json:"type"`
	// Angle holds angle behavior configuration (required when Type is "angle").
	Angle *AngleBehaviorConfig `json:"angle,omitempty"`
	// Continuous holds continuous behavior configuration (required when Type is "continuous").
	Continuous *ContinuousBehaviorConfig `json:"continuous,omitempty"`
	// DC holds DC motor behavior configuration (required when Type is "dc").
	DC *DCBehaviorConfig `json:"dc,omitempty"`
}

// Config holds the configuration for the keypress motor controller.
type Config struct {
	// KeyboardComponent is the name of the keyboard component to subscribe to.
	KeyboardComponent string `json:"keyboard_component"`

	// Motors is the list of motor binding configurations.
	Motors []MotorConfig `json:"motors"`
}

// MotorConfig holds the configuration for a single motor binding.
type MotorConfig struct {
	// Name is the unique name for this motor binding.
	Name string `json:"name"`

	// Type is the drive signal type: "pwm" (future: "i2c", "serial", etc.).
	Type DriveType `json:"type"`

	// ControlledComponent is the name of the component to control.
	ControlledComponent string `json:"controlled_component"`

	// ForwardKey is the key for forward/increase action.
	ForwardKey string `json:"forward_key"`

	// ReverseKey is the key for reverse/decrease action.
	ReverseKey string `json:"reverse_key"`

	// StopOnRelease stops the motor when the key is released (continuous/DC only).
	StopOnRelease bool `json:"stop_on_release"`

	// HoldEnabled allows OS key-repeat events to fire additional commands while
	// a key is held down. When false (default), repeat events are discarded.
	HoldEnabled bool `json:"hold_enabled"`

	// Behavior holds the behavior-specific configuration.
	Behavior BehaviorConfig `json:"behavior"`
}

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		KeyboardComponent: "keyboard",
		Motors:            []MotorConfig{},
	}

	// Parse keyboard_component
	if kb, ok := conf.Attributes["keyboard_component"].(string); ok {
		cfg.KeyboardComponent = kb
	}

	// Parse motors array
	motorsRaw, ok := conf.Attributes["motors"].([]any)
	if !ok {
		return nil, fmt.Errorf("motors configuration is required")
	}

	for i, motorRaw := range motorsRaw {
		motorMap, ok := motorRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("motor[%d]: invalid format", i)
		}

		motor, err := parseMotorConfig(motorMap, i)
		if err != nil {
			return nil, err
		}
		cfg.Motors = append(cfg.Motors, *motor)
	}

	return cfg, nil
}

func parseMotorConfig(m map[string]any, index int) (*MotorConfig, error) {
	cfg := &MotorConfig{
		StopOnRelease: true,
	}

	// Required fields
	if name, ok := m["name"].(string); ok {
		cfg.Name = name
	} else {
		return nil, fmt.Errorf("motor[%d]: name is required", index)
	}

	if typeStr, ok := m["type"].(string); ok {
		cfg.Type = DriveType(typeStr)
	} else {
		return nil, fmt.Errorf("motor[%d]: type is required", index)
	}

	if comp, ok := m["controlled_component"].(string); ok {
		cfg.ControlledComponent = comp
	} else {
		return nil, fmt.Errorf("motor[%d]: controlled_component is required", index)
	}

	if fwd, ok := m["forward_key"].(string); ok {
		cfg.ForwardKey = fwd
	} else {
		return nil, fmt.Errorf("motor[%d]: forward_key is required", index)
	}

	if rev, ok := m["reverse_key"].(string); ok {
		cfg.ReverseKey = rev
	} else {
		return nil, fmt.Errorf("motor[%d]: reverse_key is required", index)
	}

	// Optional fields
	if stop, ok := m["stop_on_release"].(bool); ok {
		cfg.StopOnRelease = stop
	}
	if hold, ok := m["hold_enabled"].(bool); ok {
		cfg.HoldEnabled = hold
	}

	// Parse behavior
	behaviorRaw, ok := m["behavior"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("motor[%d]: behavior is required", index)
	}

	behavior, err := parseBehaviorConfig(behaviorRaw, index)
	if err != nil {
		return nil, err
	}
	cfg.Behavior = *behavior

	return cfg, nil
}

func parseBehaviorConfig(m map[string]any, motorIndex int) (*BehaviorConfig, error) {
	cfg := &BehaviorConfig{}

	// Parse behavior type
	if typeStr, ok := m["type"].(string); ok {
		cfg.Type = BehaviorType(typeStr)
	} else {
		return nil, fmt.Errorf("motor[%d]: behavior.type is required", motorIndex)
	}

	// Parse behavior-specific config based on type
	switch cfg.Type {
	case BehaviorTypeAngle:
		angleRaw, ok := m["angle"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("motor[%d]: behavior.angle is required when type is 'angle'", motorIndex)
		}
		angleCfg, err := parseAngleBehaviorConfig(angleRaw, motorIndex)
		if err != nil {
			return nil, err
		}
		cfg.Angle = angleCfg

	case BehaviorTypeContinuous:
		contRaw, ok := m["continuous"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("motor[%d]: behavior.continuous is required when type is 'continuous'", motorIndex)
		}
		contCfg, err := parseContinuousBehaviorConfig(contRaw, motorIndex)
		if err != nil {
			return nil, err
		}
		cfg.Continuous = contCfg

	case BehaviorTypeDC:
		dcRaw, ok := m["dc"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("motor[%d]: behavior.dc is required when type is 'dc'", motorIndex)
		}
		dcCfg, err := parseDCBehaviorConfig(dcRaw, motorIndex)
		if err != nil {
			return nil, err
		}
		cfg.DC = dcCfg

	default:
		return nil, fmt.Errorf("motor[%d]: invalid behavior.type %q (must be 'angle', 'continuous', or 'dc')", motorIndex, cfg.Type)
	}

	return cfg, nil
}

func parseAngleBehaviorConfig(m map[string]any, motorIndex int) (*AngleBehaviorConfig, error) {
	cfg := &AngleBehaviorConfig{
		AngleStep:    5.0,
		MinAngle:     -90.0,
		MaxAngle:     90.0,
		InitialAngle: 0.0,
		Speed:        1.0,
	}

	if step, ok := m["angle_step"].(float64); ok {
		cfg.AngleStep = step
	}
	if minAngle, ok := m["min_angle"].(float64); ok {
		cfg.MinAngle = minAngle
	}
	if maxAngle, ok := m["max_angle"].(float64); ok {
		cfg.MaxAngle = maxAngle
	}
	if initial, ok := m["initial_angle"].(float64); ok {
		cfg.InitialAngle = initial
	}
	if speed, ok := m["speed"].(float64); ok {
		cfg.Speed = speed
	}

	return cfg, nil
}

func parseContinuousBehaviorConfig(m map[string]any, motorIndex int) (*ContinuousBehaviorConfig, error) {
	cfg := &ContinuousBehaviorConfig{
		Speed:        1.0,
		InitialSpeed: 0.0,
	}

	if speed, ok := m["speed"].(float64); ok {
		cfg.Speed = speed
	}
	if initial, ok := m["initial_speed"].(float64); ok {
		cfg.InitialSpeed = initial
	}

	return cfg, nil
}

func parseDCBehaviorConfig(m map[string]any, motorIndex int) (*DCBehaviorConfig, error) {
	cfg := &DCBehaviorConfig{
		Speed:        1.0,
		InitialSpeed: 0.0,
	}

	if speed, ok := m["speed"].(float64); ok {
		cfg.Speed = speed
	}
	if initial, ok := m["initial_speed"].(float64); ok {
		cfg.InitialSpeed = initial
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if len(c.Motors) == 0 {
		return fmt.Errorf("at least one motor binding is required")
	}

	motorNames := make(map[string]bool)
	keyBindings := make(map[string]string) // key -> motor name

	for i, motor := range c.Motors {
		// Validate required fields
		if motor.Name == "" {
			return fmt.Errorf("motor[%d]: name is required", i)
		}
		if motor.ControlledComponent == "" {
			return fmt.Errorf("motor[%d]: controlled_component is required", i)
		}
		if motor.ForwardKey == "" {
			return fmt.Errorf("motor[%d]: forward_key is required", i)
		}
		if motor.ReverseKey == "" {
			return fmt.Errorf("motor[%d]: reverse_key is required", i)
		}

		// Validate drive type
		switch motor.Type {
		case DriveTypePWM:
			// Valid
		default:
			return fmt.Errorf("motor[%d]: invalid type %q (must be 'pwm')", i, motor.Type)
		}

		// Check for duplicate motor names
		if motorNames[motor.Name] {
			return fmt.Errorf("motor[%d]: duplicate motor name %q", i, motor.Name)
		}
		motorNames[motor.Name] = true

		// Check for duplicate key bindings
		if existing, ok := keyBindings[motor.ForwardKey]; ok {
			return fmt.Errorf("motor[%d]: forward_key %q already bound to motor %q", i, motor.ForwardKey, existing)
		}
		keyBindings[motor.ForwardKey] = motor.Name

		if existing, ok := keyBindings[motor.ReverseKey]; ok {
			return fmt.Errorf("motor[%d]: reverse_key %q already bound to motor %q", i, motor.ReverseKey, existing)
		}
		keyBindings[motor.ReverseKey] = motor.Name

		// Validate behavior
		if err := validateBehavior(&motor, i); err != nil {
			return err
		}
	}

	return nil
}

func validateBehavior(motor *MotorConfig, index int) error {
	switch motor.Behavior.Type {
	case BehaviorTypeAngle:
		if motor.Behavior.Angle == nil {
			return fmt.Errorf("motor[%d]: behavior.angle is required when type is 'angle'", index)
		}
		cfg := motor.Behavior.Angle
		if cfg.MinAngle >= cfg.MaxAngle {
			return fmt.Errorf("motor[%d]: min_angle must be less than max_angle", index)
		}
		if cfg.InitialAngle < cfg.MinAngle || cfg.InitialAngle > cfg.MaxAngle {
			return fmt.Errorf("motor[%d]: initial_angle must be within min/max range", index)
		}
		if cfg.AngleStep <= 0 {
			return fmt.Errorf("motor[%d]: angle_step must be positive", index)
		}
		if cfg.Speed < 0 || cfg.Speed > 1 {
			return fmt.Errorf("motor[%d]: speed must be between 0.0 and 1.0", index)
		}

	case BehaviorTypeContinuous:
		if motor.Behavior.Continuous == nil {
			return fmt.Errorf("motor[%d]: behavior.continuous is required when type is 'continuous'", index)
		}
		cfg := motor.Behavior.Continuous
		if cfg.Speed < 0 || cfg.Speed > 1 {
			return fmt.Errorf("motor[%d]: speed must be between 0.0 and 1.0", index)
		}
		if cfg.InitialSpeed < -1 || cfg.InitialSpeed > 1 {
			return fmt.Errorf("motor[%d]: initial_speed must be between -1.0 and 1.0", index)
		}

	case BehaviorTypeDC:
		if motor.Behavior.DC == nil {
			return fmt.Errorf("motor[%d]: behavior.dc is required when type is 'dc'", index)
		}
		cfg := motor.Behavior.DC
		if cfg.Speed < 0 || cfg.Speed > 1 {
			return fmt.Errorf("motor[%d]: speed must be between 0.0 and 1.0", index)
		}
		if cfg.InitialSpeed < -1 || cfg.InitialSpeed > 1 {
			return fmt.Errorf("motor[%d]: initial_speed must be between -1.0 and 1.0", index)
		}

	default:
		return fmt.Errorf("motor[%d]: invalid behavior.type %q", index, motor.Behavior.Type)
	}

	return nil
}
