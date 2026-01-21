package keypress_motor

import (
	"fmt"

	"github.com/gorai/gorai/pkg/resource"
)

// MotorType defines the type of motor being controlled.
type MotorType string

const (
	MotorTypeAngleServo      MotorType = "angle_servo"
	MotorTypeContinuousServo MotorType = "continuous_servo"
	MotorTypeDCMotor         MotorType = "dc_motor"
)

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

	// Type is the motor type: "angle_servo", "continuous_servo", or "dc_motor".
	Type MotorType `json:"type"`

	// PWMComponent is the name of the PWM component to control.
	PWMComponent string `json:"pwm_component"`

	// ForwardKey is the key for forward/increase action.
	ForwardKey string `json:"forward_key"`

	// ReverseKey is the key for reverse/decrease action.
	ReverseKey string `json:"reverse_key"`

	// StopOnRelease stops the motor when the key is released (continuous/DC only).
	StopOnRelease bool `json:"stop_on_release"`

	// Speed is the speed multiplier (0.0 to 1.0).
	Speed float64 `json:"speed"`

	// AngleStep is the degrees per keypress (angle servo only).
	AngleStep float64 `json:"angle_step"`

	// MinAngle is the minimum angle in degrees (angle servo only).
	MinAngle float64 `json:"min_angle"`

	// MaxAngle is the maximum angle in degrees (angle servo only).
	MaxAngle float64 `json:"max_angle"`

	// InitialAngle is the starting angle in degrees (angle servo only).
	InitialAngle float64 `json:"initial_angle"`
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
		Speed:         1.0,
		AngleStep:     5.0,
		MinAngle:      -90.0,
		MaxAngle:      90.0,
		InitialAngle:  0.0,
	}

	// Required fields
	if name, ok := m["name"].(string); ok {
		cfg.Name = name
	} else {
		return nil, fmt.Errorf("motor[%d]: name is required", index)
	}

	if typeStr, ok := m["type"].(string); ok {
		cfg.Type = MotorType(typeStr)
	} else {
		return nil, fmt.Errorf("motor[%d]: type is required", index)
	}

	if pwm, ok := m["pwm_component"].(string); ok {
		cfg.PWMComponent = pwm
	} else {
		return nil, fmt.Errorf("motor[%d]: pwm_component is required", index)
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

	if speed, ok := m["speed"].(float64); ok {
		cfg.Speed = speed
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
		if motor.PWMComponent == "" {
			return fmt.Errorf("motor[%d]: pwm_component is required", i)
		}
		if motor.ForwardKey == "" {
			return fmt.Errorf("motor[%d]: forward_key is required", i)
		}
		if motor.ReverseKey == "" {
			return fmt.Errorf("motor[%d]: reverse_key is required", i)
		}

		// Validate motor type
		switch motor.Type {
		case MotorTypeAngleServo, MotorTypeContinuousServo, MotorTypeDCMotor:
			// Valid
		default:
			return fmt.Errorf("motor[%d]: invalid type %q (must be angle_servo, continuous_servo, or dc_motor)", i, motor.Type)
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

		// Validate angle servo specific fields
		if motor.Type == MotorTypeAngleServo {
			if motor.MinAngle >= motor.MaxAngle {
				return fmt.Errorf("motor[%d]: min_angle must be less than max_angle", i)
			}
			if motor.InitialAngle < motor.MinAngle || motor.InitialAngle > motor.MaxAngle {
				return fmt.Errorf("motor[%d]: initial_angle must be within min/max range", i)
			}
			if motor.AngleStep <= 0 {
				return fmt.Errorf("motor[%d]: angle_step must be positive", i)
			}
		}

		// Validate speed
		if motor.Speed < 0 || motor.Speed > 1 {
			return fmt.Errorf("motor[%d]: speed must be between 0.0 and 1.0", i)
		}
	}

	return nil
}

