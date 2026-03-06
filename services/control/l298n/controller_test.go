package l298n

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
			name: "valid config with one motor",
			cfg: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				PWMFrequencyHz:    5000,
				Motors: []MotorDef{
					{
						Name:       "motor_fl",
						MotorTopic: "gorai.main-robot.motor.motor_fl.command",
						SpeedPin:   2,
						IN1Pin:     3,
						IN2Pin:     4,
					},
				},
			},
			want_ok: true,
		},
		{
			name: "valid config with four motors",
			cfg: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				PWMFrequencyHz:    5000,
				Motors: []MotorDef{
					{Name: "fl", MotorTopic: "t.fl", SpeedPin: 2, IN1Pin: 3, IN2Pin: 4},
					{Name: "fr", MotorTopic: "t.fr", SpeedPin: 10, IN1Pin: 11, IN2Pin: 12},
					{Name: "rl", MotorTopic: "t.rl", SpeedPin: 13, IN1Pin: 14, IN2Pin: 15},
					{Name: "rr", MotorTopic: "t.rr", SpeedPin: 16, IN1Pin: 17, IN2Pin: 18},
				},
			},
			want_ok: true,
		},
		{
			name: "missing nats_subject_prefix",
			cfg: Config{
				DeviceID: "gsp-pico",
				Motors: []MotorDef{
					{Name: "m", MotorTopic: "t", SpeedPin: 2, IN1Pin: 3, IN2Pin: 4},
				},
			},
			want_ok: false,
		},
		{
			name: "missing device_id",
			cfg: Config{
				NATSSubjectPrefix: "gsp",
				Motors: []MotorDef{
					{Name: "m", MotorTopic: "t", SpeedPin: 2, IN1Pin: 3, IN2Pin: 4},
				},
			},
			want_ok: false,
		},
		{
			name:    "no motors",
			cfg:     Config{NATSSubjectPrefix: "gsp", DeviceID: "gsp-pico"},
			want_ok: false,
		},
		{
			name: "motor missing motor_topic",
			cfg: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				Motors: []MotorDef{
					{Name: "m", SpeedPin: 2, IN1Pin: 3, IN2Pin: 4},
				},
			},
			want_ok: false,
		},
		{
			name: "motor same in1 and in2 pins",
			cfg: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				Motors: []MotorDef{
					{Name: "m", MotorTopic: "t", SpeedPin: 2, IN1Pin: 3, IN2Pin: 3},
				},
			},
			want_ok: false,
		},
		{
			name: "motor speed_pin same as in1_pin",
			cfg: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				Motors: []MotorDef{
					{Name: "m", MotorTopic: "t", SpeedPin: 3, IN1Pin: 3, IN2Pin: 4},
				},
			},
			want_ok: false,
		},
		{
			name: "motor speed_pin same as in2_pin",
			cfg: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				Motors: []MotorDef{
					{Name: "m", MotorTopic: "t", SpeedPin: 4, IN1Pin: 3, IN2Pin: 4},
				},
			},
			want_ok: false,
		},
		{
			name: "duplicate motor names",
			cfg: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				Motors: []MotorDef{
					{Name: "m", MotorTopic: "t1", SpeedPin: 2, IN1Pin: 3, IN2Pin: 4},
					{Name: "m", MotorTopic: "t2", SpeedPin: 5, IN1Pin: 6, IN2Pin: 7},
				},
			},
			want_ok: false,
		},
		{
			name: "motor with invert and brake_on_stop",
			cfg: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				PWMFrequencyHz:    5000,
				Motors: []MotorDef{
					{
						Name: "m", MotorTopic: "t", SpeedPin: 2,
						IN1Pin: 3, IN2Pin: 4, Invert: true, BrakeOnStop: true,
					},
				},
			},
			want_ok: true,
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
		{"gpio_config", "gsp.gsp-pico.tx.command.gpio_config"},
		{"gpio_set", "gsp.gsp-pico.tx.command.gpio_set"},
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

func TestNewConfigFromResource(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"nats_subject_prefix": "gsp",
			"device_id":          "gsp-pico",
			"motors": []any{
				map[string]any{
					"name":          "motor_fl",
					"motor_topic":   "gorai.main-robot.motor.motor_fl.command",
					"speed_pin":     float64(2),
					"in1_pin":       float64(3),
					"in2_pin":       float64(4),
					"invert":        false,
					"brake_on_stop": true,
				},
				map[string]any{
					"name":          "motor_fr",
					"motor_topic":   "gorai.main-robot.motor.motor_fr.command",
					"speed_pin":     float64(10),
					"in1_pin":       float64(11),
					"in2_pin":       float64(12),
					"invert":        true,
					"brake_on_stop": false,
				},
			},
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.NATSSubjectPrefix != "gsp" {
		t.Errorf("NATSSubjectPrefix = %q, want %q", cfg.NATSSubjectPrefix, "gsp")
	}
	if cfg.DeviceID != "gsp-pico" {
		t.Errorf("DeviceID = %q, want %q", cfg.DeviceID, "gsp-pico")
	}
	if len(cfg.Motors) != 2 {
		t.Fatalf("Motors count = %d, want 2", len(cfg.Motors))
	}

	fl := cfg.Motors[0]
	if fl.Name != "motor_fl" {
		t.Errorf("Motors[0].Name = %q, want %q", fl.Name, "motor_fl")
	}
	if fl.MotorTopic != "gorai.main-robot.motor.motor_fl.command" {
		t.Errorf("Motors[0].MotorTopic = %q", fl.MotorTopic)
	}
	if fl.SpeedPin != 2 {
		t.Errorf("Motors[0].SpeedPin = %d, want 2", fl.SpeedPin)
	}
	if fl.IN1Pin != 3 {
		t.Errorf("Motors[0].IN1Pin = %d, want 3", fl.IN1Pin)
	}
	if fl.IN2Pin != 4 {
		t.Errorf("Motors[0].IN2Pin = %d, want 4", fl.IN2Pin)
	}
	if fl.BrakeOnStop != true {
		t.Error("Motors[0].BrakeOnStop should be true")
	}

	fr := cfg.Motors[1]
	if fr.Name != "motor_fr" {
		t.Errorf("Motors[1].Name = %q, want %q", fr.Name, "motor_fr")
	}
	if fr.SpeedPin != 10 {
		t.Errorf("Motors[1].SpeedPin = %d, want 10", fr.SpeedPin)
	}
	if !fr.Invert {
		t.Error("Motors[1].Invert should be true")
	}
	if fr.BrakeOnStop {
		t.Error("Motors[1].BrakeOnStop should be false")
	}
}

func TestTruthTable(t *testing.T) {
	tests := []struct {
		name         string
		power        float64
		invert       bool
		brake_on_stop bool
		want_in1     uint8
		want_in2     uint8
		want_speed   float64
	}{
		{"forward", 0.5, false, false, 1, 0, 0.5},
		{"backward", -0.5, false, false, 0, 1, 0.5},
		{"full forward", 1.0, false, false, 1, 0, 1.0},
		{"full backward", -1.0, false, false, 0, 1, 1.0},
		{"stop coast", 0.0, false, false, 0, 0, 0.0},
		{"stop brake", 0.0, false, true, 1, 1, 0.0},
		{"inverted forward", 0.5, true, false, 0, 1, 0.5},
		{"inverted backward", -0.5, true, false, 1, 0, 0.5},
		{"inverted stop brake", 0.0, true, true, 1, 1, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			power := clamp(tt.power, -1.0, 1.0)
			if tt.invert {
				power = -power
			}

			var in1, in2 uint8
			switch {
			case power > 0:
				in1 = 1
				in2 = 0
			case power < 0:
				in1 = 0
				in2 = 1
			default:
				if tt.brake_on_stop {
					in1 = 1
					in2 = 1
				} else {
					in1 = 0
					in2 = 0
				}
			}

			abs_power := power
			if abs_power < 0 {
				abs_power = -abs_power
			}

			if in1 != tt.want_in1 {
				t.Errorf("in1 = %d, want %d", in1, tt.want_in1)
			}
			if in2 != tt.want_in2 {
				t.Errorf("in2 = %d, want %d", in2, tt.want_in2)
			}
			if abs_power != tt.want_speed {
				t.Errorf("abs_power = %f, want %f", abs_power, tt.want_speed)
			}
		})
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
	}

	for _, tt := range tests {
		got := clamp(tt.value, tt.min_val, tt.max_val)
		if got != tt.want {
			t.Errorf("clamp(%f, %f, %f) = %f, want %f",
				tt.value, tt.min_val, tt.max_val, got, tt.want)
		}
	}
}
