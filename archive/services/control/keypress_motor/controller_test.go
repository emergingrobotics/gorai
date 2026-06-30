package keypress_motor

import (
	"testing"

	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAngleToPulse(t *testing.T) {
	tests := []struct {
		name       string
		angle      float64
		minAngle   float64
		maxAngle   float64
		minPulseUs float64
		maxPulseUs float64
		want       float64
	}{
		{
			name:       "center angle standard range",
			angle:      0.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       1500.0,
		},
		{
			name:       "min angle standard range",
			angle:      -90.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       1000.0,
		},
		{
			name:       "max angle standard range",
			angle:      90.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       2000.0,
		},
		{
			name:       "positive 45 degrees standard range",
			angle:      45.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       1750.0,
		},
		{
			name:       "negative 45 degrees standard range",
			angle:      -45.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       1250.0,
		},
		{
			name:       "asymmetric range",
			angle:      0.0,
			minAngle:   -45.0,
			maxAngle:   45.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       1500.0,
		},
		{
			name:       "positive only range center",
			angle:      45.0,
			minAngle:   0.0,
			maxAngle:   90.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       1500.0,
		},
		{
			name:       "custom pulse range 900-2100",
			angle:      0.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 900.0,
			maxPulseUs: 2100.0,
			want:       1500.0,
		},
		{
			name:       "custom pulse range min angle",
			angle:      -90.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 900.0,
			maxPulseUs: 2100.0,
			want:       900.0,
		},
		{
			name:       "custom pulse range max angle",
			angle:      90.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 900.0,
			maxPulseUs: 2100.0,
			want:       2100.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AngleToPulse(tt.angle, tt.minAngle, tt.maxAngle, tt.minPulseUs, tt.maxPulseUs)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestPulseToAngle(t *testing.T) {
	tests := []struct {
		name       string
		pulseUs    float64
		minAngle   float64
		maxAngle   float64
		minPulseUs float64
		maxPulseUs float64
		want       float64
	}{
		{
			name:       "center pulse standard range",
			pulseUs:    1500.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       0.0,
		},
		{
			name:       "min pulse standard range",
			pulseUs:    1000.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       -90.0,
		},
		{
			name:       "max pulse standard range",
			pulseUs:    2000.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       90.0,
		},
		{
			name:       "custom pulse range center",
			pulseUs:    1500.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 900.0,
			maxPulseUs: 2100.0,
			want:       0.0,
		},
		{
			name:       "custom pulse range min",
			pulseUs:    900.0,
			minAngle:   -90.0,
			maxAngle:   90.0,
			minPulseUs: 900.0,
			maxPulseUs: 2100.0,
			want:       -90.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PulseToAngle(tt.pulseUs, tt.minAngle, tt.maxAngle, tt.minPulseUs, tt.maxPulseUs)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestSpeedToPulse(t *testing.T) {
	tests := []struct {
		name       string
		speed      float64
		minPulseUs float64
		maxPulseUs float64
		want       float64
	}{
		{
			name:       "stop standard range",
			speed:      0.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       1500.0,
		},
		{
			name:       "full forward standard range",
			speed:      1.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       2000.0,
		},
		{
			name:       "full reverse standard range",
			speed:      -1.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       1000.0,
		},
		{
			name:       "half forward standard range",
			speed:      0.5,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       1750.0,
		},
		{
			name:       "half reverse standard range",
			speed:      -0.5,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       1250.0,
		},
		{
			name:       "stop custom range",
			speed:      0.0,
			minPulseUs: 900.0,
			maxPulseUs: 2100.0,
			want:       1500.0,
		},
		{
			name:       "full forward custom range",
			speed:      1.0,
			minPulseUs: 900.0,
			maxPulseUs: 2100.0,
			want:       2100.0,
		},
		{
			name:       "full reverse custom range",
			speed:      -1.0,
			minPulseUs: 900.0,
			maxPulseUs: 2100.0,
			want:       900.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SpeedToPulse(tt.speed, tt.minPulseUs, tt.maxPulseUs)
			assert.InDelta(t, tt.want, got, 0.001)
		})
	}
}

func TestPulseToSpeed(t *testing.T) {
	tests := []struct {
		name       string
		pulseUs    float64
		minPulseUs float64
		maxPulseUs float64
		want       float64
	}{
		{
			name:       "center pulse standard range",
			pulseUs:    1500.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       0.0,
		},
		{
			name:       "max pulse standard range",
			pulseUs:    2000.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       1.0,
		},
		{
			name:       "min pulse standard range",
			pulseUs:    1000.0,
			minPulseUs: 1000.0,
			maxPulseUs: 2000.0,
			want:       -1.0,
		},
		{
			name:       "center pulse custom range",
			pulseUs:    1500.0,
			minPulseUs: 900.0,
			maxPulseUs: 2100.0,
			want:       0.0,
		},
		{
			name:       "max pulse custom range",
			pulseUs:    2100.0,
			minPulseUs: 900.0,
			maxPulseUs: 2100.0,
			want:       1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := PulseToSpeed(tt.pulseUs, tt.minPulseUs, tt.maxPulseUs)
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
			name: "valid angle behavior",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:                "pan",
						Type:                DriveTypePWM,
						ControlledComponent: "pan_pwm",
						ForwardKey:          "d",
						ReverseKey:          "a",
						Behavior: BehaviorConfig{
							Type: BehaviorTypeAngle,
							Angle: &AngleBehaviorConfig{
								AngleStep:    5.0,
								MinAngle:     -90.0,
								MaxAngle:     90.0,
								InitialAngle: 0.0,
								Speed:        1.0,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid continuous behavior",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:                "drive",
						Type:                DriveTypePWM,
						ControlledComponent: "drive_pwm",
						ForwardKey:          "w",
						ReverseKey:          "s",
						StopOnRelease:       true,
						Behavior: BehaviorConfig{
							Type: BehaviorTypeContinuous,
							Continuous: &ContinuousBehaviorConfig{
								Speed:        0.5,
								InitialSpeed: 0.0,
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "valid dc behavior",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:                "wheel",
						Type:                DriveTypePWM,
						ControlledComponent: "wheel_pwm",
						ForwardKey:          "w",
						ReverseKey:          "s",
						StopOnRelease:       true,
						Behavior: BehaviorConfig{
							Type: BehaviorTypeDC,
							DC: &DCBehaviorConfig{
								Speed:        1.0,
								InitialSpeed: 0.0,
							},
						},
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
						Name:                "",
						Type:                DriveTypePWM,
						ControlledComponent: "pwm",
						ForwardKey:          "d",
						ReverseKey:          "a",
						Behavior: BehaviorConfig{
							Type: BehaviorTypeAngle,
							Angle: &AngleBehaviorConfig{
								AngleStep:    5.0,
								MinAngle:     -90.0,
								MaxAngle:     90.0,
								InitialAngle: 0.0,
								Speed:        1.0,
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "name is required",
		},
		{
			name: "invalid drive type",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:                "motor",
						Type:                "invalid",
						ControlledComponent: "pwm",
						ForwardKey:          "d",
						ReverseKey:          "a",
						Behavior: BehaviorConfig{
							Type: BehaviorTypeAngle,
							Angle: &AngleBehaviorConfig{
								AngleStep:    5.0,
								MinAngle:     -90.0,
								MaxAngle:     90.0,
								InitialAngle: 0.0,
								Speed:        1.0,
							},
						},
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
						Name:                "motor",
						Type:                DriveTypePWM,
						ControlledComponent: "pwm1",
						ForwardKey:          "d",
						ReverseKey:          "a",
						Behavior: BehaviorConfig{
							Type: BehaviorTypeAngle,
							Angle: &AngleBehaviorConfig{
								AngleStep:    5.0,
								MinAngle:     -90.0,
								MaxAngle:     90.0,
								InitialAngle: 0.0,
								Speed:        1.0,
							},
						},
					},
					{
						Name:                "motor",
						Type:                DriveTypePWM,
						ControlledComponent: "pwm2",
						ForwardKey:          "w",
						ReverseKey:          "s",
						Behavior: BehaviorConfig{
							Type: BehaviorTypeAngle,
							Angle: &AngleBehaviorConfig{
								AngleStep:    5.0,
								MinAngle:     -90.0,
								MaxAngle:     90.0,
								InitialAngle: 0.0,
								Speed:        1.0,
							},
						},
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
						Name:                "motor1",
						Type:                DriveTypePWM,
						ControlledComponent: "pwm1",
						ForwardKey:          "d",
						ReverseKey:          "a",
						Behavior: BehaviorConfig{
							Type: BehaviorTypeAngle,
							Angle: &AngleBehaviorConfig{
								AngleStep:    5.0,
								MinAngle:     -90.0,
								MaxAngle:     90.0,
								InitialAngle: 0.0,
								Speed:        1.0,
							},
						},
					},
					{
						Name:                "motor2",
						Type:                DriveTypePWM,
						ControlledComponent: "pwm2",
						ForwardKey:          "d", // Duplicate!
						ReverseKey:          "s",
						Behavior: BehaviorConfig{
							Type: BehaviorTypeAngle,
							Angle: &AngleBehaviorConfig{
								AngleStep:    5.0,
								MinAngle:     -90.0,
								MaxAngle:     90.0,
								InitialAngle: 0.0,
								Speed:        1.0,
							},
						},
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
						Name:                "motor",
						Type:                DriveTypePWM,
						ControlledComponent: "pwm",
						ForwardKey:          "d",
						ReverseKey:          "a",
						Behavior: BehaviorConfig{
							Type: BehaviorTypeAngle,
							Angle: &AngleBehaviorConfig{
								AngleStep:    5.0,
								MinAngle:     90.0, // min > max
								MaxAngle:     -90.0,
								InitialAngle: 0.0,
								Speed:        1.0,
							},
						},
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
						Name:                "motor",
						Type:                DriveTypePWM,
						ControlledComponent: "pwm",
						ForwardKey:          "d",
						ReverseKey:          "a",
						Behavior: BehaviorConfig{
							Type: BehaviorTypeAngle,
							Angle: &AngleBehaviorConfig{
								AngleStep:    5.0,
								MinAngle:     -45.0,
								MaxAngle:     45.0,
								InitialAngle: 90.0, // Out of range
								Speed:        1.0,
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "initial_angle must be within",
		},
		{
			name: "invalid speed in angle behavior",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:                "motor",
						Type:                DriveTypePWM,
						ControlledComponent: "pwm",
						ForwardKey:          "d",
						ReverseKey:          "a",
						Behavior: BehaviorConfig{
							Type: BehaviorTypeAngle,
							Angle: &AngleBehaviorConfig{
								AngleStep:    5.0,
								MinAngle:     -90.0,
								MaxAngle:     90.0,
								InitialAngle: 0.0,
								Speed:        1.5, // > 1.0
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "speed must be between",
		},
		{
			name: "invalid speed in continuous behavior",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:                "motor",
						Type:                DriveTypePWM,
						ControlledComponent: "pwm",
						ForwardKey:          "d",
						ReverseKey:          "a",
						Behavior: BehaviorConfig{
							Type: BehaviorTypeContinuous,
							Continuous: &ContinuousBehaviorConfig{
								Speed:        1.5, // > 1.0
								InitialSpeed: 0.0,
							},
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "speed must be between",
		},
		{
			name: "missing angle config for angle behavior",
			config: &Config{
				KeyboardComponent: "keyboard",
				Motors: []MotorConfig{
					{
						Name:                "motor",
						Type:                DriveTypePWM,
						ControlledComponent: "pwm",
						ForwardKey:          "d",
						ReverseKey:          "a",
						Behavior: BehaviorConfig{
							Type:  BehaviorTypeAngle,
							Angle: nil, // Missing!
						},
					},
				},
			},
			wantErr: true,
			errMsg:  "behavior.angle is required",
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
				"name":                 "pan",
				"type":                 "pwm",
				"controlled_component": "pan_pwm",
				"forward_key":          "d",
				"reverse_key":          "a",
				"behavior": map[string]any{
					"type": "angle",
					"angle": map[string]any{
						"angle_step":    10.0,
						"min_angle":     -45.0,
						"max_angle":     45.0,
						"initial_angle": 0.0,
						"speed":         1.0,
					},
				},
			},
			map[string]any{
				"name":                 "drive",
				"type":                 "pwm",
				"controlled_component": "drive_pwm",
				"forward_key":          "w",
				"reverse_key":          "s",
				"stop_on_release":      true,
				"behavior": map[string]any{
					"type": "continuous",
					"continuous": map[string]any{
						"speed":         0.7,
						"initial_speed": 0.0,
					},
				},
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
	assert.Equal(t, DriveTypePWM, cfg.Motors[0].Type)
	assert.Equal(t, "pan_pwm", cfg.Motors[0].ControlledComponent)
	assert.Equal(t, "d", cfg.Motors[0].ForwardKey)
	assert.Equal(t, "a", cfg.Motors[0].ReverseKey)
	assert.Equal(t, BehaviorTypeAngle, cfg.Motors[0].Behavior.Type)
	require.NotNil(t, cfg.Motors[0].Behavior.Angle)
	assert.Equal(t, 10.0, cfg.Motors[0].Behavior.Angle.AngleStep)
	assert.Equal(t, -45.0, cfg.Motors[0].Behavior.Angle.MinAngle)
	assert.Equal(t, 45.0, cfg.Motors[0].Behavior.Angle.MaxAngle)

	// Check continuous servo
	assert.Equal(t, "drive", cfg.Motors[1].Name)
	assert.Equal(t, DriveTypePWM, cfg.Motors[1].Type)
	assert.Equal(t, "drive_pwm", cfg.Motors[1].ControlledComponent)
	assert.True(t, cfg.Motors[1].StopOnRelease)
	assert.Equal(t, BehaviorTypeContinuous, cfg.Motors[1].Behavior.Type)
	require.NotNil(t, cfg.Motors[1].Behavior.Continuous)
	assert.Equal(t, 0.7, cfg.Motors[1].Behavior.Continuous.Speed)
}

func TestConfigDefaults(t *testing.T) {
	attrs := map[string]any{
		"motors": []any{
			map[string]any{
				"name":                 "motor",
				"type":                 "pwm",
				"controlled_component": "pwm",
				"forward_key":          "d",
				"reverse_key":          "a",
				"behavior": map[string]any{
					"type":  "angle",
					"angle": map[string]any{},
				},
			},
		},
	}

	conf := resource.NewConfig(attrs)
	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	// Check defaults
	assert.Equal(t, "keyboard", cfg.KeyboardComponent)
	assert.True(t, cfg.Motors[0].StopOnRelease)
	assert.False(t, cfg.Motors[0].HoldEnabled)

	// Check angle behavior defaults
	require.NotNil(t, cfg.Motors[0].Behavior.Angle)
	assert.Equal(t, 5.0, cfg.Motors[0].Behavior.Angle.AngleStep)
	assert.Equal(t, -90.0, cfg.Motors[0].Behavior.Angle.MinAngle)
	assert.Equal(t, 90.0, cfg.Motors[0].Behavior.Angle.MaxAngle)
	assert.Equal(t, 0.0, cfg.Motors[0].Behavior.Angle.InitialAngle)
	assert.Equal(t, 1.0, cfg.Motors[0].Behavior.Angle.Speed)
}

func TestHoldEnabledParsing(t *testing.T) {
	attrs := map[string]any{
		"motors": []any{
			map[string]any{
				"name":                 "pan",
				"type":                 "pwm",
				"controlled_component": "pan_pwm",
				"forward_key":          "l",
				"reverse_key":          "j",
				"hold_enabled":         true,
				"behavior": map[string]any{
					"type": "angle",
					"angle": map[string]any{
						"angle_step": 2.5,
					},
				},
			},
			map[string]any{
				"name":                 "drive",
				"type":                 "pwm",
				"controlled_component": "drive_pwm",
				"forward_key":          "w",
				"reverse_key":          "s",
				"behavior": map[string]any{
					"type": "continuous",
					"continuous": map[string]any{
						"speed": 0.5,
					},
				},
			},
		},
	}

	conf := resource.NewConfig(attrs)
	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)
	require.Len(t, cfg.Motors, 2)

	assert.True(t, cfg.Motors[0].HoldEnabled, "pan should have hold_enabled=true")
	assert.False(t, cfg.Motors[1].HoldEnabled, "drive should default to hold_enabled=false")
}

func TestHoldEnabledValidation(t *testing.T) {
	cfg := &Config{
		KeyboardComponent: "keyboard",
		Motors: []MotorConfig{
			{
				Name:                "pan",
				Type:                DriveTypePWM,
				ControlledComponent: "pan_pwm",
				ForwardKey:          "l",
				ReverseKey:          "j",
				HoldEnabled:         true,
				Behavior: BehaviorConfig{
					Type: BehaviorTypeAngle,
					Angle: &AngleBehaviorConfig{
						AngleStep:    2.5,
						MinAngle:     -90.0,
						MaxAngle:     90.0,
						InitialAngle: 0.0,
						Speed:        1.0,
					},
				},
			},
		},
	}
	assert.NoError(t, cfg.Validate())
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
		pulse := AngleToPulse(angle, -90.0, 90.0, 1000.0, 2000.0)
		recovered := PulseToAngle(pulse, -90.0, 90.0, 1000.0, 2000.0)
		assert.InDelta(t, angle, recovered, 0.001, "angle round trip failed for %v", angle)
	}

	// Test round trip with custom pulse range
	for _, angle := range angles {
		pulse := AngleToPulse(angle, -90.0, 90.0, 900.0, 2100.0)
		recovered := PulseToAngle(pulse, -90.0, 90.0, 900.0, 2100.0)
		assert.InDelta(t, angle, recovered, 0.001, "angle round trip (custom range) failed for %v", angle)
	}

	// Test that speed -> pulse -> speed is identity
	speeds := []float64{-1.0, -0.5, 0.0, 0.5, 1.0}
	for _, speed := range speeds {
		pulse := SpeedToPulse(speed, 1000.0, 2000.0)
		recovered := PulseToSpeed(pulse, 1000.0, 2000.0)
		assert.InDelta(t, speed, recovered, 0.001, "speed round trip failed for %v", speed)
	}

	// Test speed round trip with custom pulse range
	for _, speed := range speeds {
		pulse := SpeedToPulse(speed, 900.0, 2100.0)
		recovered := PulseToSpeed(pulse, 900.0, 2100.0)
		assert.InDelta(t, speed, recovered, 0.001, "speed round trip (custom range) failed for %v", speed)
	}
}
