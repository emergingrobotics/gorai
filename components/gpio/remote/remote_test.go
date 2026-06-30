package remote

import (
	"testing"

	"github.com/emergingrobotics/gorai/pkg/resource"
)

func TestGPIOConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
	}{
		{
			name: "valid output config",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				Pin:               15,
				Mode:              "output",
				Pull:              "none",
			},
			wantErr: false,
		},
		{
			name: "valid input config with pull-up",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				Pin:               14,
				Mode:              "input",
				Pull:              "up",
			},
			wantErr: false,
		},
		{
			name: "missing prefix",
			config: Config{
				DeviceID: "gsp-pico",
				Pin:      15,
				Mode:     "output",
				Pull:     "none",
			},
			wantErr: true,
		},
		{
			name: "missing device_id",
			config: Config{
				NATSSubjectPrefix: "gsp",
				Pin:               15,
				Mode:              "output",
				Pull:              "none",
			},
			wantErr: true,
		},
		{
			name: "pin out of range high",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				Pin:               30,
				Mode:              "output",
				Pull:              "none",
			},
			wantErr: true,
		},
		{
			name: "pin out of range negative",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				Pin:               -1,
				Mode:              "output",
				Pull:              "none",
			},
			wantErr: true,
		},
		{
			name: "invalid mode",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				Pin:               15,
				Mode:              "pwm",
				Pull:              "none",
			},
			wantErr: true,
		},
		{
			name: "invalid pull",
			config: Config{
				NATSSubjectPrefix: "gsp",
				DeviceID:          "gsp-pico",
				Pin:               15,
				Mode:              "output",
				Pull:              "both",
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

func TestGPIOCommandSubject(t *testing.T) {
	cfg := &Config{
		NATSSubjectPrefix: "gsp",
		DeviceID:          "gsp-pico",
		Pin:               15,
	}

	tests := []struct {
		command_type string
		want         string
	}{
		{"gpio_config", "gsp.gsp-pico.tx.command.gpio_config"},
		{"gpio_set", "gsp.gsp-pico.tx.command.gpio_set"},
		{"gpio_query", "gsp.gsp-pico.tx.command.gpio_query"},
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

func TestGPIOModeID(t *testing.T) {
	tests := []struct {
		mode string
		want uint8
	}{
		{"input", 0x00},
		{"output", 0x01},
	}

	for _, tt := range tests {
		t.Run(tt.mode, func(t *testing.T) {
			cfg := &Config{Mode: tt.mode}
			got := cfg.ModeID()
			if got != tt.want {
				t.Errorf("ModeID() = 0x%02X, want 0x%02X", got, tt.want)
			}
		})
	}
}

func TestGPIOPullID(t *testing.T) {
	tests := []struct {
		pull string
		want uint8
	}{
		{"none", 0x00},
		{"up", 0x01},
		{"down", 0x02},
	}

	for _, tt := range tests {
		t.Run(tt.pull, func(t *testing.T) {
			cfg := &Config{Pull: tt.pull}
			got := cfg.PullID()
			if got != tt.want {
				t.Errorf("PullID() = 0x%02X, want 0x%02X", got, tt.want)
			}
		})
	}
}

func TestGPIONewConfigFromResource(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"nats_subject_prefix": "gsp",
			"device_id":           "gsp-pico",
			"pin":                 float64(15),
			"mode":                "output",
			"pull":                "up",
			"initial_value":       float64(1),
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("NewConfigFromResource() error = %v", err)
	}

	if cfg.NATSSubjectPrefix != "gsp" {
		t.Errorf("NATSSubjectPrefix = %q, want %q", cfg.NATSSubjectPrefix, "gsp")
	}
	if cfg.DeviceID != "gsp-pico" {
		t.Errorf("DeviceID = %q, want %q", cfg.DeviceID, "gsp-pico")
	}
	if cfg.Pin != 15 {
		t.Errorf("Pin = %d, want %d", cfg.Pin, 15)
	}
	if cfg.Mode != "output" {
		t.Errorf("Mode = %q, want %q", cfg.Mode, "output")
	}
	if cfg.Pull != "up" {
		t.Errorf("Pull = %q, want %q", cfg.Pull, "up")
	}
	if cfg.InitialValue != 1 {
		t.Errorf("InitialValue = %d, want %d", cfg.InitialValue, 1)
	}
}

func TestGPIONewConfigFromResourceDefaults(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"nats_subject_prefix": "gsp",
			"device_id":           "gsp-pico",
			"pin":                 float64(14),
			"mode":                "input",
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("NewConfigFromResource() error = %v", err)
	}

	if cfg.Pull != "none" {
		t.Errorf("Pull default = %q, want %q", cfg.Pull, "none")
	}
	if cfg.InitialValue != 0 {
		t.Errorf("InitialValue default = %d, want 0", cfg.InitialValue)
	}
}

func TestGPIOMaxPin(t *testing.T) {
	cfg := Config{
		NATSSubjectPrefix: "gsp",
		DeviceID:          "gsp-pico",
		Pin:               28,
		Mode:              "input",
		Pull:              "none",
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("pin 28 should be valid, got error: %v", err)
	}
}
