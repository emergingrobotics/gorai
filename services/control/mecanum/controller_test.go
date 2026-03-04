package mecanum

import (
	"testing"

	"github.com/gorai/gorai/pkg/resource"
)

func TestControllerConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		want_ok bool
	}{
		{
			name: "valid",
			cfg: Config{
				VelocityTopic: "gorai.robot.velocity_input.command",
				MotorFLName:   "motor_fl",
				MotorFRName:   "motor_fr",
				MotorRLName:   "motor_rl",
				MotorRRName:   "motor_rr",
				WheelBaseX:    0.1,
				WheelBaseY:    0.075,
			},
			want_ok: true,
		},
		{
			name: "missing velocity topic",
			cfg: Config{
				MotorFLName: "motor_fl",
				MotorFRName: "motor_fr",
				MotorRLName: "motor_rl",
				MotorRRName: "motor_rr",
				WheelBaseX:  0.1,
				WheelBaseY:  0.075,
			},
			want_ok: false,
		},
		{
			name: "missing motor",
			cfg: Config{
				VelocityTopic: "topic",
				MotorFLName:   "motor_fl",
				MotorFRName:   "",
				MotorRLName:   "motor_rl",
				MotorRRName:   "motor_rr",
				WheelBaseX:    0.1,
				WheelBaseY:    0.075,
			},
			want_ok: false,
		},
		{
			name: "zero wheel base",
			cfg: Config{
				VelocityTopic: "topic",
				MotorFLName:   "motor_fl",
				MotorFRName:   "motor_fr",
				MotorRLName:   "motor_rl",
				MotorRRName:   "motor_rr",
				WheelBaseX:    0,
				WheelBaseY:    0.075,
			},
			want_ok: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.cfg.Validate()
			if tt.want_ok && err != nil {
				t.Errorf("expected valid, got error: %v", err)
			}
			if !tt.want_ok && err == nil {
				t.Errorf("expected error, got nil")
			}
		})
	}
}

func TestControllerConfigFromResource(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"velocity_topic": "gorai.test.velocity_input.command",
			"motor_fl":       "m_fl",
			"motor_fr":       "m_fr",
			"motor_rl":       "m_rl",
			"motor_rr":       "m_rr",
			"wheel_base_x":   0.15,
			"wheel_base_y":   0.1,
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.VelocityTopic != "gorai.test.velocity_input.command" {
		t.Errorf("VelocityTopic = %q", cfg.VelocityTopic)
	}
	if cfg.MotorFLName != "m_fl" {
		t.Errorf("MotorFLName = %q", cfg.MotorFLName)
	}
	if cfg.MotorFRName != "m_fr" {
		t.Errorf("MotorFRName = %q", cfg.MotorFRName)
	}
	if cfg.MotorRLName != "m_rl" {
		t.Errorf("MotorRLName = %q", cfg.MotorRLName)
	}
	if cfg.MotorRRName != "m_rr" {
		t.Errorf("MotorRRName = %q", cfg.MotorRRName)
	}
	if cfg.WheelBaseX != 0.15 {
		t.Errorf("WheelBaseX = %f", cfg.WheelBaseX)
	}
	if cfg.WheelBaseY != 0.1 {
		t.Errorf("WheelBaseY = %f", cfg.WheelBaseY)
	}
}

func TestControllerConfigDefaults(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"velocity_topic": "topic",
			"motor_fl":       "fl",
			"motor_fr":       "fr",
			"motor_rl":       "rl",
			"motor_rr":       "rr",
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.WheelBaseX != 0.1 {
		t.Errorf("WheelBaseX default = %f, want 0.1", cfg.WheelBaseX)
	}
	if cfg.WheelBaseY != 0.075 {
		t.Errorf("WheelBaseY default = %f, want 0.075", cfg.WheelBaseY)
	}
}

func TestVelocityCommandJSON(t *testing.T) {
	cmd := VelocityCommand{VX: 1.0, VY: -0.5, Omega: 0.3}
	if cmd.VX != 1.0 || cmd.VY != -0.5 || cmd.Omega != 0.3 {
		t.Errorf("VelocityCommand fields mismatch")
	}
}
