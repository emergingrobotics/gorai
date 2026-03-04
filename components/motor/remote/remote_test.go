package remote

import (
	"testing"

	"github.com/gorai/gorai/pkg/resource"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		want_ok bool
	}{
		{
			name: "valid firmware config",
			cfg: Config{
				OutputMode:        OutputModeFirmware,
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				MotorIndex:        0,
				MaxSpeed:          1000,
			},
			want_ok: true,
		},
		{
			name: "firmware default when empty",
			cfg: Config{
				OutputMode:        "",
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				MaxSpeed:          1000,
			},
			want_ok: false,
		},
		{
			name: "firmware missing prefix",
			cfg: Config{
				OutputMode: OutputModeFirmware,
				DeviceID:   "gsp-pico",
				MotorIndex: 0,
				MaxSpeed:   1000,
			},
			want_ok: false,
		},
		{
			name: "firmware missing device id",
			cfg: Config{
				OutputMode:        OutputModeFirmware,
				NATSSubjectPrefix: "gsp",
				MotorIndex:        0,
				MaxSpeed:          1000,
			},
			want_ok: false,
		},
		{
			name: "firmware invalid motor index high",
			cfg: Config{
				OutputMode:        OutputModeFirmware,
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				MotorIndex:        5,
				MaxSpeed:          1000,
			},
			want_ok: false,
		},
		{
			name: "firmware invalid motor index negative",
			cfg: Config{
				OutputMode:        OutputModeFirmware,
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				MotorIndex:        -1,
				MaxSpeed:          1000,
			},
			want_ok: false,
		},
		{
			name: "firmware zero max speed",
			cfg: Config{
				OutputMode:        OutputModeFirmware,
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				MotorIndex:        0,
				MaxSpeed:          0,
			},
			want_ok: false,
		},
		{
			name: "firmware max motor index",
			cfg: Config{
				OutputMode:        OutputModeFirmware,
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				MotorIndex:        3,
				MaxSpeed:          1000,
			},
			want_ok: true,
		},
		{
			name: "valid nats config",
			cfg: Config{
				OutputMode: OutputModeNATS,
				MotorTopic: "gorai.main-robot.motor.motor_fl.command",
			},
			want_ok: true,
		},
		{
			name: "nats missing motor_topic",
			cfg: Config{
				OutputMode: OutputModeNATS,
			},
			want_ok: false,
		},
		{
			name: "nats does not require firmware fields",
			cfg: Config{
				OutputMode: OutputModeNATS,
				MotorTopic: "gorai.main-robot.motor.motor_fl.command",
			},
			want_ok: true,
		},
		{
			name: "invalid output_mode",
			cfg: Config{
				OutputMode: "unknown",
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

func TestCommandSubject(t *testing.T) {
	cfg := &Config{
		NATSSubjectPrefix: "gsp",
		DeviceID:          "gsp-pico",
	}

	tests := []struct {
		command string
		want    string
	}{
		{"motor_set", "gsp.gsp-pico.tx.command.motor_set"},
		{"motor_config", "gsp.gsp-pico.tx.command.motor_config"},
		{"motor_enable", "gsp.gsp-pico.tx.command.motor_enable"},
	}

	for _, tt := range tests {
		t.Run(tt.command, func(t *testing.T) {
			got := cfg.CommandSubject(tt.command)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestNewConfigFromResourceFirmware(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"output_mode":         "firmware",
			"nats_subject_prefix": "gsp",
			"device_id":          "gsp-pico",
			"motor_index":        float64(2),
			"max_speed":          float64(500),
			"auto_configure":     true,
			"max_rpm":            float64(300),
			"ppr":                float64(720),
			"gear_ratio":         float64(200),
			"invert":             true,
			"brake_on_stop":      true,
			"pwm_pin":            float64(10),
			"in1_pin":            float64(11),
			"in2_pin":            float64(12),
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.OutputMode != OutputModeFirmware {
		t.Errorf("OutputMode = %q, want %q", cfg.OutputMode, OutputModeFirmware)
	}
	if cfg.NATSSubjectPrefix != "gsp" {
		t.Errorf("NATSSubjectPrefix = %q, want %q", cfg.NATSSubjectPrefix, "gsp")
	}
	if cfg.DeviceID != "gsp-pico" {
		t.Errorf("DeviceID = %q, want %q", cfg.DeviceID, "gsp-pico")
	}
	if cfg.MotorIndex != 2 {
		t.Errorf("MotorIndex = %d, want 2", cfg.MotorIndex)
	}
	if cfg.MaxSpeed != 500 {
		t.Errorf("MaxSpeed = %d, want 500", cfg.MaxSpeed)
	}
	if cfg.MaxRPM != 300 {
		t.Errorf("MaxRPM = %d, want 300", cfg.MaxRPM)
	}
	if cfg.PPR != 720 {
		t.Errorf("PPR = %d, want 720", cfg.PPR)
	}
	if cfg.GearRatio != 200 {
		t.Errorf("GearRatio = %d, want 200", cfg.GearRatio)
	}
	if !cfg.Invert {
		t.Error("Invert should be true")
	}
	if !cfg.BrakeOnStop {
		t.Error("BrakeOnStop should be true")
	}
	if cfg.PWMPin != 10 {
		t.Errorf("PWMPin = %d, want 10", cfg.PWMPin)
	}
	if cfg.IN1Pin != 11 {
		t.Errorf("IN1Pin = %d, want 11", cfg.IN1Pin)
	}
	if cfg.IN2Pin != 12 {
		t.Errorf("IN2Pin = %d, want 12", cfg.IN2Pin)
	}
}

func TestNewConfigFromResourceNATS(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"output_mode": "nats",
			"motor_topic": "gorai.main-robot.motor.motor_fl.command",
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.OutputMode != OutputModeNATS {
		t.Errorf("OutputMode = %q, want %q", cfg.OutputMode, OutputModeNATS)
	}
	if cfg.MotorTopic != "gorai.main-robot.motor.motor_fl.command" {
		t.Errorf("MotorTopic = %q, want %q", cfg.MotorTopic, "gorai.main-robot.motor.motor_fl.command")
	}
	if !cfg.IsNATSMode() {
		t.Error("IsNATSMode() should return true")
	}
}

func TestNewConfigFromResourceDefaults(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"nats_subject_prefix": "gsp",
			"device_id":          "gsp-pico",
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.OutputMode != OutputModeFirmware {
		t.Errorf("OutputMode default = %q, want %q", cfg.OutputMode, OutputModeFirmware)
	}
	if cfg.MaxSpeed != 1000 {
		t.Errorf("MaxSpeed default = %d, want 1000", cfg.MaxSpeed)
	}
	if !cfg.AutoConfigure {
		t.Error("AutoConfigure default should be true")
	}
	if cfg.MaxRPM != 200 {
		t.Errorf("MaxRPM default = %d, want 200", cfg.MaxRPM)
	}
	if cfg.PPR != 360 {
		t.Errorf("PPR default = %d, want 360", cfg.PPR)
	}
	if cfg.GearRatio != 100 {
		t.Errorf("GearRatio default = %d, want 100", cfg.GearRatio)
	}
	if cfg.IsNATSMode() {
		t.Error("IsNATSMode() should return false for default config")
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		value, min_val, max_val, want float64
	}{
		{0.5, -1.0, 1.0, 0.5},
		{-2.0, -1.0, 1.0, -1.0},
		{3.0, -1.0, 1.0, 1.0},
		{0.0, -1.0, 1.0, 0.0},
		{-1.0, -1.0, 1.0, -1.0},
		{1.0, -1.0, 1.0, 1.0},
	}

	for _, tt := range tests {
		got := clamp(tt.value, tt.min_val, tt.max_val)
		if got != tt.want {
			t.Errorf("clamp(%f, %f, %f) = %f, want %f",
				tt.value, tt.min_val, tt.max_val, got, tt.want)
		}
	}
}

func TestSpeedCalculation(t *testing.T) {
	tests := []struct {
		name      string
		power     float64
		max_speed int16
		want      int16
	}{
		{"full forward", 1.0, 1000, 1000},
		{"full reverse", -1.0, 1000, -1000},
		{"half forward", 0.5, 1000, 500},
		{"quarter reverse", -0.25, 1000, -250},
		{"zero", 0.0, 1000, 0},
		{"clamped high", 2.0, 1000, 1000},
		{"clamped low", -5.0, 1000, -1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			power := clamp(tt.power, -1.0, 1.0)
			speed := int16(power * float64(tt.max_speed))
			if speed != tt.want {
				t.Errorf("power=%f max_speed=%d: got speed %d, want %d",
					tt.power, tt.max_speed, speed, tt.want)
			}
		})
	}
}
