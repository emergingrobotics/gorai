package remote

import (
	"fmt"

	"github.com/gorai/gorai/pkg/resource"
)

type Config struct {
	NATSSubjectPrefix string `json:"nats_subject_prefix"`
	DeviceID          string `json:"device_id"`
	MotorIndex        int    `json:"motor_index"`
	MaxSpeed          int16  `json:"max_speed"`
	AutoConfigure     bool   `json:"auto_configure"`
	MaxRPM            uint16 `json:"max_rpm"`
	PPR               uint16 `json:"ppr"`
	GearRatio         uint16 `json:"gear_ratio"`
	Invert            bool   `json:"invert"`
	BrakeOnStop       bool   `json:"brake_on_stop"`

	PWMPin uint8 `json:"pwm_pin"`
	IN1Pin uint8 `json:"in1_pin"`
	IN2Pin uint8 `json:"in2_pin"`
}

func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		MaxSpeed:      1000,
		AutoConfigure: true,
		MaxRPM:        200,
		PPR:           360,
		GearRatio:     100,
	}

	if v, ok := conf.GetString("nats_subject_prefix"); ok {
		cfg.NATSSubjectPrefix = v
	}
	if v, ok := conf.GetString("device_id"); ok {
		cfg.DeviceID = v
	}
	if v, ok := conf.GetInt("motor_index"); ok {
		cfg.MotorIndex = v
	}
	if v, ok := conf.GetFloat("max_speed"); ok {
		cfg.MaxSpeed = int16(v)
	}
	if v, ok := conf.GetBool("auto_configure"); ok {
		cfg.AutoConfigure = v
	}
	if v, ok := conf.GetFloat("max_rpm"); ok {
		cfg.MaxRPM = uint16(v)
	}
	if v, ok := conf.GetFloat("ppr"); ok {
		cfg.PPR = uint16(v)
	}
	if v, ok := conf.GetFloat("gear_ratio"); ok {
		cfg.GearRatio = uint16(v)
	}
	if v, ok := conf.GetBool("invert"); ok {
		cfg.Invert = v
	}
	if v, ok := conf.GetBool("brake_on_stop"); ok {
		cfg.BrakeOnStop = v
	}
	if v, ok := conf.GetFloat("pwm_pin"); ok {
		cfg.PWMPin = uint8(v)
	}
	if v, ok := conf.GetFloat("in1_pin"); ok {
		cfg.IN1Pin = uint8(v)
	}
	if v, ok := conf.GetFloat("in2_pin"); ok {
		cfg.IN2Pin = uint8(v)
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	if c.NATSSubjectPrefix == "" {
		return fmt.Errorf("nats_subject_prefix is required")
	}
	if c.DeviceID == "" {
		return fmt.Errorf("device_id is required")
	}
	if c.MotorIndex < 0 || c.MotorIndex > 3 {
		return fmt.Errorf("motor_index must be 0-3")
	}
	if c.MaxSpeed <= 0 {
		return fmt.Errorf("max_speed must be positive")
	}
	return nil
}

func (c *Config) CommandSubject(command_type string) string {
	return fmt.Sprintf("%s.%s.tx.command.%s", c.NATSSubjectPrefix, c.DeviceID, command_type)
}
