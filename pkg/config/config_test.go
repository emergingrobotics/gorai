package config

import (
	"strings"
	"testing"
)

func TestDeviceConfigParsing(t *testing.T) {
	json := `{
		"version": "2",
		"robot": {"name": "test-robot"},
		"devices": [
			{
				"id": "gsp-pico",
				"nats_prefix": "gsp",
				"reset_on_startup": true
			},
			{
				"id": "other-device",
				"nats_prefix": "dev",
				"reset_on_startup": false
			}
		]
	}`

	cfg, err := LoadFromBytes([]byte(json))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(cfg.Devices) != 2 {
		t.Fatalf("Devices count = %d, want 2", len(cfg.Devices))
	}

	d0 := cfg.Devices[0]
	if d0.ID != "gsp-pico" {
		t.Errorf("Devices[0].ID = %q, want %q", d0.ID, "gsp-pico")
	}
	if d0.NATSPrefix != "gsp" {
		t.Errorf("Devices[0].NATSPrefix = %q, want %q", d0.NATSPrefix, "gsp")
	}
	if !d0.ResetOnStartup {
		t.Error("Devices[0].ResetOnStartup = false, want true")
	}

	d1 := cfg.Devices[1]
	if d1.ID != "other-device" {
		t.Errorf("Devices[1].ID = %q, want %q", d1.ID, "other-device")
	}
	if d1.ResetOnStartup {
		t.Error("Devices[1].ResetOnStartup = true, want false")
	}
}

func TestDeviceConfigParsingEmpty(t *testing.T) {
	json := `{
		"version": "2",
		"robot": {"name": "test-robot"}
	}`

	cfg, err := LoadFromBytes([]byte(json))
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(cfg.Devices) != 0 {
		t.Errorf("Devices count = %d, want 0", len(cfg.Devices))
	}

	if err := cfg.Validate(); err != nil {
		t.Errorf("validation should pass with no devices: %v", err)
	}
}

func TestDeviceConfigValidation(t *testing.T) {
	tests := []struct {
		name       string
		devices    []DeviceConfig
		want_error string
	}{
		{
			name: "valid",
			devices: []DeviceConfig{
				{ID: "gsp-pico", NATSPrefix: "gsp", ResetOnStartup: true},
			},
			want_error: "",
		},
		{
			name: "empty id",
			devices: []DeviceConfig{
				{ID: "", NATSPrefix: "gsp"},
			},
			want_error: "devices[0].id: required",
		},
		{
			name: "empty nats_prefix",
			devices: []DeviceConfig{
				{ID: "pico", NATSPrefix: ""},
			},
			want_error: "devices[0].nats_prefix: required",
		},
		{
			name: "duplicate ids",
			devices: []DeviceConfig{
				{ID: "pico", NATSPrefix: "gsp"},
				{ID: "pico", NATSPrefix: "gsp2"},
			},
			want_error: "duplicate device ID",
		},
		{
			name: "multiple valid",
			devices: []DeviceConfig{
				{ID: "pico-1", NATSPrefix: "gsp"},
				{ID: "pico-2", NATSPrefix: "gsp"},
			},
			want_error: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RDL{
				Version: "2",
				Robot:   RobotConfig{Name: "test"},
				Devices: tt.devices,
			}

			err := cfg.Validate()
			if tt.want_error == "" {
				if err != nil {
					t.Errorf("expected valid, got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.want_error)
				} else if !strings.Contains(err.Error(), tt.want_error) {
					t.Errorf("expected error containing %q, got: %v", tt.want_error, err)
				}
			}
		})
	}
}

func TestShouldEmbedNATS(t *testing.T) {
	tests := []struct {
		name string
		nats *NATSConfig
		want bool
	}{
		{"nil defaults to embed", nil, true},
		{"local url embeds", &NATSConfig{URL: "nats://localhost:4222"}, true},
		{"remote url does not embed", &NATSConfig{URL: "nats://192.168.1.50:4222"}, false},
		{"external disables embed", &NATSConfig{URL: "nats://localhost:4222", External: true}, false},
		{"listen forces embed even with remote url", &NATSConfig{URL: "nats://192.168.1.50:4222", Listen: "0.0.0.0:4222"}, true},
		{"listen forces embed", &NATSConfig{Listen: "0.0.0.0:4222", URL: "nats://127.0.0.1:4222"}, true},
		{"external beats listen", &NATSConfig{Listen: "0.0.0.0:4222", External: true}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &RDL{Version: "2", Robot: RobotConfig{Name: "test"}, NATS: tt.nats}
			if got := cfg.ShouldEmbedNATS(); got != tt.want {
				t.Errorf("ShouldEmbedNATS() = %v, want %v", got, tt.want)
			}
		})
	}
}
