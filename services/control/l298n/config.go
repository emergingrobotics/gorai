package l298n

import (
	"fmt"

	"github.com/gorai/gorai/pkg/resource"
)

// MotorDef describes a single motor controlled by an L298N board.
type MotorDef struct {
	Name         string `json:"name"`
	MotorTopic   string `json:"motor_topic"`
	PWMComponent string `json:"pwm_component"`
	IN1Pin       uint8  `json:"in1_pin"`
	IN2Pin       uint8  `json:"in2_pin"`
	Invert       bool   `json:"invert"`
	BrakeOnStop  bool   `json:"brake_on_stop"`
}

// Config holds the configuration for the L298N motor controller service.
type Config struct {
	NATSSubjectPrefix string     `json:"nats_subject_prefix"`
	DeviceID          string     `json:"device_id"`
	Motors            []MotorDef `json:"motors"`
}

func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{}

	if v, ok := conf.GetString("nats_subject_prefix"); ok {
		cfg.NATSSubjectPrefix = v
	}
	if v, ok := conf.GetString("device_id"); ok {
		cfg.DeviceID = v
	}

	motors_raw, ok := conf.Attributes["motors"]
	if !ok {
		return cfg, nil
	}

	motors_arr, ok := motors_raw.([]any)
	if !ok {
		return nil, fmt.Errorf("motors must be an array")
	}

	for i, entry := range motors_arr {
		m, ok := entry.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("motors[%d]: expected object", i)
		}
		def, err := parseMotorDef(m, i)
		if err != nil {
			return nil, err
		}
		cfg.Motors = append(cfg.Motors, *def)
	}

	return cfg, nil
}

func parseMotorDef(m map[string]any, idx int) (*MotorDef, error) {
	def := &MotorDef{}

	name, ok := m["name"].(string)
	if !ok || name == "" {
		return nil, fmt.Errorf("motors[%d]: name is required", idx)
	}
	def.Name = name

	if v, ok := m["motor_topic"].(string); ok {
		def.MotorTopic = v
	}
	if v, ok := m["pwm_component"].(string); ok {
		def.PWMComponent = v
	}
	if v, ok := m["in1_pin"].(float64); ok {
		def.IN1Pin = uint8(v)
	}
	if v, ok := m["in2_pin"].(float64); ok {
		def.IN2Pin = uint8(v)
	}
	if v, ok := m["invert"].(bool); ok {
		def.Invert = v
	}
	if v, ok := m["brake_on_stop"].(bool); ok {
		def.BrakeOnStop = v
	}

	return def, nil
}

func (c *Config) Validate() error {
	if c.NATSSubjectPrefix == "" {
		return fmt.Errorf("nats_subject_prefix is required")
	}
	if c.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if len(c.Motors) == 0 {
		return fmt.Errorf("at least one motor definition is required")
	}

	seen := make(map[string]bool)
	for i, m := range c.Motors {
		if m.Name == "" {
			return fmt.Errorf("motors[%d]: name is required", i)
		}
		if seen[m.Name] {
			return fmt.Errorf("motors[%d]: duplicate name %q", i, m.Name)
		}
		seen[m.Name] = true

		if m.MotorTopic == "" {
			return fmt.Errorf("motors[%d] (%s): motor_topic is required", i, m.Name)
		}
		if m.PWMComponent == "" {
			return fmt.Errorf("motors[%d] (%s): pwm_component is required", i, m.Name)
		}
		if m.IN1Pin > 28 {
			return fmt.Errorf("motors[%d] (%s): in1_pin must be 0-28", i, m.Name)
		}
		if m.IN2Pin > 28 {
			return fmt.Errorf("motors[%d] (%s): in2_pin must be 0-28", i, m.Name)
		}
		if m.IN1Pin == m.IN2Pin {
			return fmt.Errorf("motors[%d] (%s): in1_pin and in2_pin must be different", i, m.Name)
		}
	}

	return nil
}

func (c *Config) CommandSubject(command_type string) string {
	return fmt.Sprintf("%s.%s.tx.command.%s", c.NATSSubjectPrefix, c.DeviceID, command_type)
}
