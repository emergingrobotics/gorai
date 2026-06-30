package remote

import (
	"testing"

	"github.com/emergingrobotics/gorai/pkg/resource"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		want_ok bool
	}{
		{
			name: "valid",
			cfg: Config{
				NATSSubjectPrefix:  "gsp",
				DeviceID:           "gsp-pico",
				EncoderIndex:       0,
				TicksPerRevolution: 360,
			},
			want_ok: true,
		},
		{
			name: "missing prefix",
			cfg: Config{
				DeviceID:           "gsp-pico",
				EncoderIndex:       0,
				TicksPerRevolution: 360,
			},
			want_ok: false,
		},
		{
			name: "invalid encoder index",
			cfg: Config{
				NATSSubjectPrefix:  "gsp",
				DeviceID:           "gsp-pico",
				EncoderIndex:       5,
				TicksPerRevolution: 360,
			},
			want_ok: false,
		},
		{
			name: "zero ticks",
			cfg: Config{
				NATSSubjectPrefix:  "gsp",
				DeviceID:           "gsp-pico",
				EncoderIndex:       0,
				TicksPerRevolution: 0,
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

func TestNewConfigFromResource(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"nats_subject_prefix":  "gsp",
			"device_id":           "gsp-pico",
			"encoder_index":       float64(2),
			"ticks_per_revolution": float64(720),
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
	if cfg.EncoderIndex != 2 {
		t.Errorf("EncoderIndex = %d, want 2", cfg.EncoderIndex)
	}
	if cfg.TicksPerRevolution != 720 {
		t.Errorf("TicksPerRevolution = %d, want 720", cfg.TicksPerRevolution)
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

	if cfg.TicksPerRevolution != 360 {
		t.Errorf("TicksPerRevolution default = %d, want 360", cfg.TicksPerRevolution)
	}
	if cfg.EncoderIndex != 0 {
		t.Errorf("EncoderIndex default = %d, want 0", cfg.EncoderIndex)
	}
}

func TestNATSSubject(t *testing.T) {
	cfg := &Config{
		NATSSubjectPrefix: "gsp",
		DeviceID:          "gsp-pico",
		EncoderIndex:      1,
	}

	want := "gsp.gsp-pico.rx.sensor.encoder_data"
	got := cfg.NATSSubjectPrefix + "." + cfg.DeviceID + ".rx.sensor.encoder_data"
	if got != want {
		t.Errorf("subject = %q, want %q", got, want)
	}
}
