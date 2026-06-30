package remote

import (
	"testing"

	"github.com/emergingrobotics/gorai/pkg/resource"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "pico-pwm",
				Pin:               6,
				FrequencyHz:       50,
				MinPulseUs:        1000,
				MaxPulseUs:        2000,
				InitialPulseUs:    1500,
				FailsafePulseUs:   -1,
			},
			wantErr: false,
		},
		{
			name: "missing prefix",
			config: Config{
				DeviceID:       "pico-pwm",
				Pin:            6,
				FrequencyHz:    50,
				MinPulseUs:     1000,
				MaxPulseUs:     2000,
				InitialPulseUs: 1500,
			},
			wantErr: true,
		},
		{
			name: "missing device_id",
			config: Config{
				NATSSubjectPrefix: "gsp",
				Pin:               6,
				FrequencyHz:       50,
				MinPulseUs:        1000,
				MaxPulseUs:        2000,
				InitialPulseUs:    1500,
			},
			wantErr: true,
		},
		{
			name: "pin out of range",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "pico-pwm",
				Pin:               29,
				FrequencyHz:       50,
				MinPulseUs:        1000,
				MaxPulseUs:        2000,
				InitialPulseUs:    1500,
			},
			wantErr: true,
		},
		{
			name: "negative pin",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "pico-pwm",
				Pin:               -1,
				FrequencyHz:       50,
				MinPulseUs:        1000,
				MaxPulseUs:        2000,
				InitialPulseUs:    1500,
			},
			wantErr: true,
		},
		{
			name: "zero frequency",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "pico-pwm",
				Pin:               6,
				FrequencyHz:       0,
				MinPulseUs:        1000,
				MaxPulseUs:        2000,
				InitialPulseUs:    1500,
			},
			wantErr: true,
		},
		{
			name: "min greater than max",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "pico-pwm",
				Pin:               6,
				FrequencyHz:       50,
				MinPulseUs:        2000,
				MaxPulseUs:        1000,
				InitialPulseUs:    1500,
			},
			wantErr: true,
		},
		{
			name: "initial out of range",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "pico-pwm",
				Pin:               6,
				FrequencyHz:       50,
				MinPulseUs:        1000,
				MaxPulseUs:        2000,
				InitialPulseUs:    2500,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCommandSubject(t *testing.T) {
	cfg := &Config{
		NATSSubjectPrefix: "gsp",
		DeviceID:          "pico-pwm",
		Pin:               6,
	}

	tests := []struct {
		command_type string
		want         string
	}{
		{"pwm_set", "gsp.pico-pwm.tx.command.pwm_set"},
		{"pwm_enable", "gsp.pico-pwm.tx.command.pwm_enable"},
		{"pwm_config", "gsp.pico-pwm.tx.command.pwm_config"},
	}

	for _, tt := range tests {
		t.Run(tt.command_type, func(t *testing.T) {
			got := cfg.CommandSubject(tt.command_type)
			if got != tt.want {
				t.Errorf("CommandSubject(%q) = %q, want %q", tt.command_type, got, tt.want)
			}
		})
	}
}

func TestNewConfigFromResource(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"nats_subject_prefix": "gsp",
			"device_id":           "pico-pwm",
			"pin":                 float64(6),
			"frequency_hz":        float64(50),
			"min_pulse_us":        float64(1000),
			"max_pulse_us":        float64(2000),
			"initial_pulse_us":    float64(1500),
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("NewConfigFromResource() error = %v", err)
	}

	if cfg.NATSSubjectPrefix != "gsp" {
		t.Errorf("NATSSubjectPrefix = %q, want %q", cfg.NATSSubjectPrefix, "gsp")
	}
	if cfg.DeviceID != "pico-pwm" {
		t.Errorf("DeviceID = %q, want %q", cfg.DeviceID, "pico-pwm")
	}
	if cfg.Pin != 6 {
		t.Errorf("Pin = %d, want %d", cfg.Pin, 6)
	}
	if cfg.FrequencyHz != 50 {
		t.Errorf("FrequencyHz = %f, want %f", cfg.FrequencyHz, 50.0)
	}
	if cfg.MinPulseUs != 1000 {
		t.Errorf("MinPulseUs = %f, want %f", cfg.MinPulseUs, 1000.0)
	}
	if cfg.MaxPulseUs != 2000 {
		t.Errorf("MaxPulseUs = %f, want %f", cfg.MaxPulseUs, 2000.0)
	}
	if cfg.InitialPulseUs != 1500 {
		t.Errorf("InitialPulseUs = %f, want %f", cfg.InitialPulseUs, 1500.0)
	}
}

func TestNewConfigFromResourceDefaults(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"nats_subject_prefix": "gsp",
			"device_id":           "dev1",
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("NewConfigFromResource() error = %v", err)
	}

	if cfg.FrequencyHz != 50 {
		t.Errorf("FrequencyHz default = %f, want 50", cfg.FrequencyHz)
	}
	if cfg.MinPulseUs != 1000 {
		t.Errorf("MinPulseUs default = %f, want 1000", cfg.MinPulseUs)
	}
	if cfg.MaxPulseUs != 2000 {
		t.Errorf("MaxPulseUs default = %f, want 2000", cfg.MaxPulseUs)
	}
	if cfg.InitialPulseUs != 1500 {
		t.Errorf("InitialPulseUs default = %f, want 1500", cfg.InitialPulseUs)
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		value, min, max, want float64
	}{
		{50, 0, 100, 50},
		{-10, 0, 100, 0},
		{150, 0, 100, 100},
		{0, 0, 100, 0},
		{100, 0, 100, 100},
	}

	for _, tt := range tests {
		got := clamp(tt.value, tt.min, tt.max)
		if got != tt.want {
			t.Errorf("clamp(%f, %f, %f) = %f, want %f", tt.value, tt.min, tt.max, got, tt.want)
		}
	}
}

func TestFailsafePulseUsDefault(t *testing.T) {
	cfg := Config{
		NATSSubjectPrefix: "gsp",
		DeviceID:          "pico-pwm",
		Pin:               6,
		FrequencyHz:       50,
		MinPulseUs:        1000,
		MaxPulseUs:        2000,
		InitialPulseUs:    1500,
		FailsafePulseUs:   -1,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	want := 1500.0
	if cfg.FailsafePulseUs != want {
		t.Errorf("FailsafePulseUs = %f, want %f (center of range)", cfg.FailsafePulseUs, want)
	}
}

func TestFailsafePulseUsExplicit(t *testing.T) {
	cfg := Config{
		NATSSubjectPrefix: "gsp",
		DeviceID:          "pico-pwm",
		Pin:               6,
		FrequencyHz:       50,
		MinPulseUs:        500,
		MaxPulseUs:        2500,
		InitialPulseUs:    1500,
		FailsafePulseUs:   1000,
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if cfg.FailsafePulseUs != 1000 {
		t.Errorf("FailsafePulseUs = %f, want 1000", cfg.FailsafePulseUs)
	}
}

func TestFailsafePulseUsOutOfRange(t *testing.T) {
	cfg := Config{
		NATSSubjectPrefix: "gsp",
		DeviceID:          "pico-pwm",
		Pin:               6,
		FrequencyHz:       50,
		MinPulseUs:        1000,
		MaxPulseUs:        2000,
		InitialPulseUs:    1500,
		FailsafePulseUs:   3000,
	}

	if err := cfg.Validate(); err == nil {
		t.Error("expected error for failsafe_pulse_us out of range")
	}
}

func TestAutoConfigureDefault(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"nats_subject_prefix": "gsp",
			"device_id":           "pico",
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("NewConfigFromResource() error = %v", err)
	}

	if !cfg.AutoConfigure {
		t.Error("AutoConfigure should default to true")
	}
}

func TestAutoConfigureExplicitFalse(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"nats_subject_prefix": "gsp",
			"device_id":           "pico",
			"auto_configure":      false,
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("NewConfigFromResource() error = %v", err)
	}

	if cfg.AutoConfigure {
		t.Error("AutoConfigure should be false when explicitly set")
	}
}

func TestProvisioningSubjects(t *testing.T) {
	cfg := &Config{
		NATSSubjectPrefix: "gsp",
		DeviceID:          "gsp-pico",
		Pin:               6,
	}

	tests := []struct {
		command_type string
		want         string
	}{
		{"gpio_config", "gsp.gsp-pico.tx.command.gpio_config"},
		{"pwm_config", "gsp.gsp-pico.tx.command.pwm_config"},
		{"pwm_enable", "gsp.gsp-pico.tx.command.pwm_enable"},
		{"pwm_set", "gsp.gsp-pico.tx.command.pwm_set"},
	}

	for _, tt := range tests {
		t.Run(tt.command_type, func(t *testing.T) {
			got := cfg.CommandSubject(tt.command_type)
			if got != tt.want {
				t.Errorf("CommandSubject(%q) = %q, want %q", tt.command_type, got, tt.want)
			}
		})
	}
}

func TestNewConfigFromResourceWithNewFields(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"nats_subject_prefix": "gsp",
			"device_id":           "gsp-pico",
			"pin":                 float64(6),
			"failsafe_pulse_us":   float64(1200),
			"auto_configure":      false,
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("NewConfigFromResource() error = %v", err)
	}

	if cfg.FailsafePulseUs != 1200 {
		t.Errorf("FailsafePulseUs = %f, want 1200", cfg.FailsafePulseUs)
	}
	if cfg.AutoConfigure {
		t.Error("AutoConfigure should be false")
	}
}

func TestNormalizedToPulseConversion(t *testing.T) {
	cfg := &Config{
		MinPulseUs: 1000,
		MaxPulseUs: 2000,
	}

	center := (cfg.MinPulseUs + cfg.MaxPulseUs) / 2.0
	half_range := (cfg.MaxPulseUs - cfg.MinPulseUs) / 2.0

	tests := []struct {
		normalized float64
		want_pulse float64
	}{
		{-1.0, 1000},
		{0.0, 1500},
		{1.0, 2000},
		{0.5, 1750},
		{-0.5, 1250},
	}

	for _, tt := range tests {
		pulse := center + tt.normalized*half_range
		if pulse != tt.want_pulse {
			t.Errorf("normalized %f -> pulse %f, want %f", tt.normalized, pulse, tt.want_pulse)
		}
	}
}
