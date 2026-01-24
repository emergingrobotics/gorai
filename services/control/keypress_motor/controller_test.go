package keypress_motor

import (
	"testing"

	"github.com/gorai/gorai/pkg/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAngleToPulse(t *testing.T) {
	tests := []struct {
		name     string
		angle    float64
		minAngle float64
		maxAngle float64
		want     float64
	}{
		{
			name:     "center angle",
			angle:    0.0,
			minAngle: -90.0,
			maxAngle: 90.0,
			want:     1500.0,
		},
		{
			name:     "min angle",
			angle:    -90.0,
			minAngle: -90.0,
			maxAngle: 90.0,
			want:     1000.0,
		},
		{
			name:     "max angle",
			angle:    90.0,
			minAngle: -90.0,
			maxAngle: 90.0,
			want:     2000.0,
		},
		{
			name:     "positive 45 degrees",
			angle:    45.0,
			minAngle: -90.0,
			maxAngle: 90.0,
			want:     1750.0,
		},
		{
			name:     "negative 45 degrees",
			angle:    -45.0,
			minAngle: -90.0,
			maxAngle: 90.0,
			want:     1250.0,
		},
		{
			name:     "asymmetric range",
			angle:    0.0,
			minAngle: -45.0,
			maxAngle: 45.0,
			want:     1500.0,
		},
		{
			name:     "positive only range center",
			angle:    45.0,
			minAngle: 0.0,
			maxAngle: 90.0,
			want:     1500.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AngleToPulse(tt.angle, tt.minAngle, tt.maxAngle)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestPulseToAngle(t *testing.T) {
	tests := []struct {
		name     string
		pulseUs  float64
		minAngle float64
		maxAngle float64
		want     float64
	}{
		{
			name:     "center pulse",
			pulseUs:  1500.0,
			minAngle: -90.0,
			maxAngle: 90.0,
			want:     0.0,
		},
		{
			name:     "min pulse",
			pulseUs:  1000.0,
			minAngle: -90.0,
			maxAngle: 90.0,
			want:     -90.0,
		},
		{
			name:     "max pulse",
			pulseUs:  2000.0,
			minAngle: -90.0,
			maxAngle: 90.0,
			want:     90.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PulseToAngle(tt.pulseUs, tt.minAngle, tt.maxAngle)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestSpeedToPulse(t *testing.T) {
	tests := []struct {
		name  string
		speed float64
		want  float64
	}{
		{
			name:  "stop",
			speed: 0.0,
			want:  1500.0,
		},
		{
			name:  "full forward",
			speed: 1.0,
			want:  2000.0,
		},
		{
			name:  "full reverse",
			speed: -1.0,
			want:  1000.0,
		},
		{
			name:  "half forward",
			speed: 0.5,
			want:  1750.0,
		},
		{
			name:  "half reverse",
			speed: -0.5,
			want:  1250.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SpeedToPulse(tt.speed)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestPulseToSpeed(t *testing.T) {
	tests := []struct {
		name    string
		pulseUs float64
		want    float64
	}{
		{
			name:    "center pulse",
			pulseUs: 1500.0,
			want:    0.0,
		},
		{
			name:    "max pulse",
			pulseUs: 2000.0,
			want:    1.0,
		},
		{
			name:    "min pulse",
			pulseUs: 1000.0,
			want:    -1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PulseToSpeed(tt.pulseUs)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestClamp(t *testing.T) {
	tests := []struct {
		name  string
		value float64
		min   float64
		max   float64
		want  float64
	}{
		{"in range", 0.5, 0.0, 1.0, 0.5},
		{"below min", -0.5, 0.0, 1.0, 0.0},
		{"above max", 1.5, 0.0, 1.0, 1.0},
		{"at min", 0.0, 0.0, 1.0, 0.0},
		{"at max", 1.0, 0.0, 1.0, 1.0},
		{"negative range", -45.0, -90.0, 90.0, -45.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Clamp(tt.value, tt.min, tt.max)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid angle servo",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:         "pan",
						Type:         MotorTypeAngleServo,
						PWMComponent: "pan_pwm",
						ForwardKey:   "D",
						ReverseKey:   "A",
						MinAngle:     -90.0,
						MaxAngle:     90.0,
						AngleStep:    5.0,
						Speed:        1.0,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid continuous servo",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:          "drive",
						Type:          MotorTypeContinuousServo,
						PWMComponent:  "drive_pwm",
						ForwardKey:    "W",
						ReverseKey:    "S",
						StopOnRelease: true,
						Speed:         0.5,
					},
				},
			},
			wantErr: false,
		},
		{
			name: "no motors",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors:            []MotorConfig{},
			},
			wantErr: true,
			errMsg:  "at least one motor binding is required",
		},
		{
			name: "missing motor name",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:         "",
						Type:         MotorTypeAngleServo,
						PWMComponent: "pwm",
						ForwardKey:   "D",
						ReverseKey:   "A",
					},
				},
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "invalid motor type",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:         "motor",
						Type:         "invalid",
						PWMComponent: "pwm",
						ForwardKey:   "D",
						ReverseKey:   "A",
					},
				},
			},
			wantErr: true,
			errMsg:  "invalid type",
		},
		{
			name: "duplicate motor names",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:         "motor",
						Type:         MotorTypeAngleServo,
						PWMComponent: "pwm1",
						ForwardKey:   "D",
						ReverseKey:   "A",
						MinAngle:     -90.0,
						MaxAngle:     90.0,
						AngleStep:    5.0,
						Speed:        1.0,
					},
					{
						Name:         "motor",
						Type:         MotorTypeAngleServo,
						PWMComponent: "pwm2",
						ForwardKey:   "W",
						ReverseKey:   "S",
						MinAngle:     -90.0,
						MaxAngle:     90.0,
						AngleStep:    5.0,
						Speed:        1.0,
					},
				},
			},
			wantErr: true,
			errMsg:  "duplicate motor name",
		},
		{
			name: "duplicate key bindings",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:         "motor1",
						Type:         MotorTypeAngleServo,
						PWMComponent: "pwm1",
						ForwardKey:   "D",
						ReverseKey:   "A",
						MinAngle:     -90.0,
						MaxAngle:     90.0,
						AngleStep:    5.0,
						Speed:        1.0,
					},
					{
						Name:         "motor2",
						Type:         MotorTypeAngleServo,
						PWMComponent: "pwm2",
						ForwardKey:   "D", // Duplicate!
						ReverseKey:   "S",
						MinAngle:     -90.0,
						MaxAngle:     90.0,
						AngleStep:    5.0,
						Speed:        1.0,
					},
				},
			},
			wantErr: true,
			errMsg:  "already bound",
		},
		{
			name: "invalid angle range",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:         "motor",
						Type:         MotorTypeAngleServo,
						PWMComponent: "pwm",
						ForwardKey:   "D",
						ReverseKey:   "A",
						MinAngle:     90.0,  // min > max
						MaxAngle:     -90.0,
						AngleStep:    5.0,
						Speed:        1.0,
					},
				},
			},
			wantErr: true,
			errMsg:  "min_angle must be less than max_angle",
		},
		{
			name: "initial angle out of range",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:         "motor",
						Type:         MotorTypeAngleServo,
						PWMComponent: "pwm",
						ForwardKey:   "D",
						ReverseKey:   "A",
						MinAngle:     -45.0,
						MaxAngle:     45.0,
						InitialAngle: 90.0, // Out of range
						AngleStep:    5.0,
						Speed:        1.0,
					},
				},
			},
			wantErr: true,
			errMsg:  "initial_angle must be within",
		},
		{
			name: "invalid speed",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:         "motor",
						Type:         MotorTypeContinuousServo,
						PWMComponent: "pwm",
						ForwardKey:   "D",
						ReverseKey:   "A",
						Speed:        1.5, // > 1.0
					},
				},
			},
			wantErr: true,
			errMsg:  "speed must be between",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigParsing(t *testing.T) {
	attrs := map[string]any{
		"keyboard_component": "my_keyboard",
		"motors": []any{
			map[string]any{
				"name":          "pan",
				"type":          "angle_servo",
				"pwm_component": "pan_pwm",
				"forward_key":   "D",
				"reverse_key":   "A",
				"angle_step":    10.0,
				"min_angle":     -45.0,
				"max_angle":     45.0,
				"initial_angle": 0.0,
				"speed":         1.0,
			},
			map[string]any{
				"name":            "drive",
				"type":            "continuous_servo",
				"pwm_component":   "drive_pwm",
				"forward_key":     "W",
				"reverse_key":     "S",
				"stop_on_release": true,
				"speed":           0.7,
			},
		},
	}

	conf := resource.NewConfig(attrs)
	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	assert.Equal(t, "my_keyboard", cfg.KeyboardComponent)
	require.Len(t, cfg.Motors, 2)

	// Check angle servo
	assert.Equal(t, "pan", cfg.Motors[0].Name)
	assert.Equal(t, MotorTypeAngleServo, cfg.Motors[0].Type)
	assert.Equal(t, "pan_pwm", cfg.Motors[0].PWMComponent)
	assert.Equal(t, "D", cfg.Motors[0].ForwardKey)
	assert.Equal(t, "A", cfg.Motors[0].ReverseKey)
	assert.Equal(t, 10.0, cfg.Motors[0].AngleStep)
	assert.Equal(t, -45.0, cfg.Motors[0].MinAngle)
	assert.Equal(t, 45.0, cfg.Motors[0].MaxAngle)

	// Check continuous servo
	assert.Equal(t, "drive", cfg.Motors[1].Name)
	assert.Equal(t, MotorTypeContinuousServo, cfg.Motors[1].Type)
	assert.Equal(t, "drive_pwm", cfg.Motors[1].PWMComponent)
	assert.True(t, cfg.Motors[1].StopOnRelease)
	assert.Equal(t, 0.7, cfg.Motors[1].Speed)
}

func TestConfigDefaults(t *testing.T) {
	attrs := map[string]any{
		"motors": []any{
			map[string]any{
				"name":          "motor",
				"type":          "angle_servo",
				"pwm_component": "pwm",
				"forward_key":   "D",
				"reverse_key":   "A",
			},
		},
	}

	conf := resource.NewConfig(attrs)
	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	// Check defaults
	assert.Equal(t, "keyboard", cfg.KeyboardComponent)
	assert.True(t, cfg.Motors[0].StopOnRelease)
	assert.Equal(t, 1.0, cfg.Motors[0].Speed)
	assert.Equal(t, 5.0, cfg.Motors[0].AngleStep)
	assert.Equal(t, -90.0, cfg.Motors[0].MinAngle)
	assert.Equal(t, 90.0, cfg.Motors[0].MaxAngle)
	assert.Equal(t, 0.0, cfg.Motors[0].InitialAngle)
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateStopped, "stopped"},
		{StateRunning, "running"},
		{StateError, "error"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestRoundTripConversions(t *testing.T) {
	// Test that angle -> pulse -> angle is identity (within precision)
	angles := []float64{-90.0, -45.0, 0.0, 45.0, 90.0}
	for _, angle := range angles {
		pulse := AngleToPulse(angle, -90.0, 90.0)
		recovered := PulseToAngle(pulse, -90.0, 90.0)
		assert.InDelta(t, angle, recovered, 0.001, "angle round trip failed for %v", angle)
	}

	// Test that speed -> pulse -> speed is identity
	speeds := []float64{-1.0, -0.5, 0.0, 0.5, 1.0}
	for _, speed := range speeds {
		pulse := SpeedToPulse(speed)
		recovered := PulseToSpeed(pulse)
		assert.InDelta(t, speed, recovered, 0.001, "speed round trip failed for %v", speed)
	}
}

