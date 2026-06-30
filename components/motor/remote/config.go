package remote

import (
	"fmt"

	"github.com/emergingrobotics/gorai/pkg/resource"
)

const (
	OutputModeFirmware = "firmware"
	OutputModeNATS     = "nats"
)

type Config struct {
	// OutputMode selects how SetPower commands are delivered:
	//   "firmware" (default) - sends MOTOR_SET/MOTOR_CONFIG/MOTOR_ENABLE via GSP2
	//   "nats" - publishes {"power": float64} to MotorTopic
	OutputMode string `json:"output_mode"`

	// MotorSubject is the NATS subject to publish motor power commands to.
	// Required when OutputMode is "nats".
	MotorSubject string `json:"motor_subject"`

	// Firmware-mode fields (used when OutputMode is "firmware")
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
		OutputMode:    OutputModeFirmware,
		MaxSpeed:      1000,
		AutoConfigure: true,
		MaxRPM:        200,
		PPR:           360,
		GearRatio:     100,
	}

	if v, ok := conf.GetString("output_mode"); ok {
		cfg.OutputMode = v
	}
	if v, ok := conf.GetString("motor_subject"); ok {
		cfg.MotorSubject = v
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
	switch c.OutputMode {
	case OutputModeFirmware:
		if c.NATSSubjectPrefix == "" {
			return fmt.Errorf("nats_subject_prefix is required for firmware output_mode")
		}
		if c.DeviceID == "" {
			return fmt.Errorf("device_id is required for firmware output_mode")
		}
		if c.MotorIndex < 0 || c.MotorIndex > 3 {
			return fmt.Errorf("motor_index must be 0-3")
		}
		if c.MaxSpeed <= 0 {
			return fmt.Errorf("max_speed must be positive")
		}
	case OutputModeNATS:
		if c.MotorSubject == "" {
			return fmt.Errorf("motor_subject is required for nats output_mode")
		}
	default:
		return fmt.Errorf("output_mode must be %q or %q, got %q", OutputModeFirmware, OutputModeNATS, c.OutputMode)
	}
	return nil
}

func (c *Config) IsNATSMode() bool {
	return c.OutputMode == OutputModeNATS
}

func (c *Config) CommandSubject(command_type string) string {
	return fmt.Sprintf("%s.%s.tx.command.%s", c.NATSSubjectPrefix, c.DeviceID, command_type)
}
