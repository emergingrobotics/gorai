package gpiod

import (
	"testing"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "valid custom config",
			config: Config{
				Pin:            18,
				FrequencyHz:    50,
				MinPulseUs:     1000,
				MaxPulseUs:     2000,
				InitialPulseUs: 1500,
			},
			wantErr: false,
		},
		{
			name: "nil pin",
			config: Config{
				Pin:            nil,
				FrequencyHz:    50,
				MinPulseUs:     1000,
				MaxPulseUs:     2000,
				InitialPulseUs: 1500,
			},
			wantErr: true,
			errMsg:  "pin is required",
		},
		{
			name: "zero frequency",
			config: Config{
				Pin:            18,
				FrequencyHz:    0,
				MinPulseUs:     1000,
				MaxPulseUs:     2000,
				InitialPulseUs: 1500,
			},
			wantErr: true,
			errMsg:  "frequency_hz must be positive",
		},
		{
			name: "frequency too high",
			config: Config{
				Pin:            18,
				FrequencyHz:    1001,
				MinPulseUs:     1000,
				MaxPulseUs:     2000,
				InitialPulseUs: 1500,
			},
			wantErr: true,
			errMsg:  "frequency_hz must be <= 1000 Hz",
		},
		{
			name: "negative min pulse",
			config: Config{
				Pin:            18,
				FrequencyHz:    50,
				MinPulseUs:     -100,
				MaxPulseUs:     2000,
				InitialPulseUs: 1500,
			},
			wantErr: true,
			errMsg:  "min_pulse_us must be non-negative",
		},
		{
			name: "max pulse less than min pulse",
			config: Config{
				Pin:            18,
				FrequencyHz:    50,
				MinPulseUs:     2000,
				MaxPulseUs:     1000,
				InitialPulseUs: 1500,
			},
			wantErr: true,
			errMsg:  "max_pulse_us",
		},
		{
			name: "initial pulse below min",
			config: Config{
				Pin:            18,
				FrequencyHz:    50,
				MinPulseUs:     1000,
				MaxPulseUs:     2000,
				InitialPulseUs: 500,
			},
			wantErr: true,
			errMsg:  "initial_pulse_us",
		},
		{
			name: "initial pulse above max",
			config: Config{
				Pin:            18,
				FrequencyHz:    50,
				MinPulseUs:     1000,
				MaxPulseUs:     2000,
				InitialPulseUs: 2500,
			},
			wantErr: true,
			errMsg:  "initial_pulse_us",
		},
		{
			name: "max pulse exceeds period",
			config: Config{
				Pin:            18,
				FrequencyHz:    100, // period = 10,000 µs
				MinPulseUs:     1000,
				MaxPulseUs:     15000, // exceeds 10,000 µs period
				InitialPulseUs: 5000,
			},
			wantErr: true,
			errMsg:  "exceeds period",
		},
		{
			name: "pin as string GPIO17",
			config: Config{
				Pin:            "GPIO17",
				FrequencyHz:    50,
				MinPulseUs:     1000,
				MaxPulseUs:     2000,
				InitialPulseUs: 1500,
			},
			wantErr: false,
		},
		{
			name: "pin as string number",
			config: Config{
				Pin:            "18",
				FrequencyHz:    50,
				MinPulseUs:     1000,
				MaxPulseUs:     2000,
				InitialPulseUs: 1500,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
					return
				}
				if tt.errMsg != "" && !containsSubstring(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Validate() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestNormalizedConversion(t *testing.T) {
	cfg := DefaultConfig() // min=1000, max=2000, center=1500, range=500

	// Create a PWM instance for testing conversion functions
	p := &PWM{config: cfg}

	tests := []struct {
		name       string
		normalized float64
		wantPulse  float64
	}{
		{"center", 0.0, 1500.0},
		{"max", 1.0, 2000.0},
		{"min", -1.0, 1000.0},
		{"half positive", 0.5, 1750.0},
		{"half negative", -0.5, 1250.0},
		{"quarter positive", 0.25, 1625.0},
		{"quarter negative", -0.25, 1375.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPulse := p.normalizedToPulse(tt.normalized)
			if !floatEquals(gotPulse, tt.wantPulse, 0.001) {
				t.Errorf("normalizedToPulse(%f) = %f, want %f", tt.normalized, gotPulse, tt.wantPulse)
			}

			// Test reverse conversion
			gotNormalized := p.pulseToNormalized(tt.wantPulse)
			if !floatEquals(gotNormalized, tt.normalized, 0.001) {
				t.Errorf("pulseToNormalized(%f) = %f, want %f", tt.wantPulse, gotNormalized, tt.normalized)
			}
		})
	}
}

func TestDutyConversion(t *testing.T) {
	cfg := Config{
		Pin:         18,
		FrequencyHz: 50.0, // period = 20,000 µs
		MinPulseUs:  1000,
		MaxPulseUs:  2000,
	}

	p := &PWM{config: cfg}

	tests := []struct {
		name      string
		duty      float64
		wantPulse float64
	}{
		{"zero duty", 0.0, 0.0},
		{"full duty", 1.0, 20000.0},
		{"5% duty (servo min)", 0.05, 1000.0},
		{"10% duty (servo max)", 0.10, 2000.0},
		{"7.5% duty (servo center)", 0.075, 1500.0},
		{"50% duty", 0.5, 10000.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPulse := p.dutyToPulse(tt.duty)
			if !floatEquals(gotPulse, tt.wantPulse, 0.001) {
				t.Errorf("dutyToPulse(%f) = %f, want %f", tt.duty, gotPulse, tt.wantPulse)
			}

			// Test reverse conversion (only for non-zero)
			if tt.wantPulse > 0 {
				gotDuty := p.pulseToDuty(tt.wantPulse)
				if !floatEquals(gotDuty, tt.duty, 0.001) {
					t.Errorf("pulseToDuty(%f) = %f, want %f", tt.wantPulse, gotDuty, tt.duty)
				}
			}
		})
	}
}

func TestConfigParsing(t *testing.T) {
	tests := []struct {
		name      string
		conf      map[string]any
		wantPin   any
		wantFreq  float64
		wantErr   bool
	}{
		{
			name: "full config with int pin",
			conf: map[string]any{
				"pin":              float64(18),
				"frequency_hz":    float64(50),
				"min_pulse_us":    float64(1000),
				"max_pulse_us":    float64(2000),
				"initial_pulse_us": float64(1500),
				"invert":          false,
			},
			wantPin:  float64(18),
			wantFreq: 50,
			wantErr:  false,
		},
		{
			name: "config with string pin GPIO18",
			conf: map[string]any{
				"pin":          "GPIO18",
				"frequency_hz": float64(50),
			},
			wantPin:  "GPIO18",
			wantFreq: 50,
			wantErr:  false,
		},
		{
			name: "config with string pin PIN12",
			conf: map[string]any{
				"pin":          "PIN12",
				"frequency_hz": float64(50),
			},
			wantPin:  "PIN12",
			wantFreq: 50,
			wantErr:  false,
		},
		{
			name: "minimal config with defaults",
			conf: map[string]any{
				"pin": float64(12),
			},
			wantPin:  float64(12),
			wantFreq: 50, // default
			wantErr:  false,
		},
		{
			name: "missing pin",
			conf: map[string]any{
				"frequency_hz": float64(50),
			},
			wantErr: true,
		},
		{
			name: "pin as int",
			conf: map[string]any{
				"pin": 18,
			},
			wantPin:  18,
			wantFreq: 50, // default
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseConfig(tt.conf)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseConfig() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ParseConfig() unexpected error = %v", err)
				return
			}
			if got.Pin != tt.wantPin {
				t.Errorf("ParseConfig().Pin = %v, want %v", got.Pin, tt.wantPin)
			}
			if got.FrequencyHz != tt.wantFreq {
				t.Errorf("ParseConfig().FrequencyHz = %v, want %v", got.FrequencyHz, tt.wantFreq)
			}
		})
	}
}

func TestConfigHelpers(t *testing.T) {
	cfg := Config{
		Pin:         18,
		FrequencyHz: 50.0,
		MinPulseUs:  1000,
		MaxPulseUs:  2000,
	}

	t.Run("PeriodUs", func(t *testing.T) {
		want := 20000.0 // 1,000,000 / 50
		got := cfg.PeriodUs()
		if !floatEquals(got, want, 0.001) {
			t.Errorf("PeriodUs() = %f, want %f", got, want)
		}
	})

	t.Run("PeriodNs", func(t *testing.T) {
		want := int64(20_000_000) // 1,000,000,000 / 50
		got := cfg.PeriodNs()
		if got != want {
			t.Errorf("PeriodNs() = %d, want %d", got, want)
		}
	})

	t.Run("CenterPulseUs", func(t *testing.T) {
		want := 1500.0
		got := cfg.CenterPulseUs()
		if !floatEquals(got, want, 0.001) {
			t.Errorf("CenterPulseUs() = %f, want %f", got, want)
		}
	})

	t.Run("PulseRangeUs", func(t *testing.T) {
		want := 500.0
		got := cfg.PulseRangeUs()
		if !floatEquals(got, want, 0.001) {
			t.Errorf("PulseRangeUs() = %f, want %f", got, want)
		}
	})
}

// Helper functions

func floatEquals(a, b, epsilon float64) bool {
	diff := a - b
	if diff < 0 {
		diff = -diff
	}
	return diff < epsilon
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
