package velocity_input

import (
	"math"
	"testing"

	"github.com/gorai/gorai/components/input"
	"github.com/gorai/gorai/pkg/resource"
)

func validConfig() Config {
	return Config{
		KeyboardComponent: "remote_keyboard",
		VelocityTopic:     "gorai.robot.velocity_input.command",
		SpeedScale:        0.5,
		MinSpeed:          0.3,
		RampSteps:         10,
		KeyBindings: map[string]VelocityBinding{
			"I": {VX: 0, VY: 1, Omega: 0},
		},
	}
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		want_ok bool
	}{
		{
			name:    "valid",
			cfg:     validConfig(),
			want_ok: true,
		},
		{
			name: "missing keyboard",
			cfg: Config{
				VelocityTopic: "topic",
				SpeedScale:    1.0,
				MinSpeed:      0.3,
				RampSteps:     10,
				KeyBindings:   map[string]VelocityBinding{"I": {VY: 1}},
			},
			want_ok: false,
		},
		{
			name: "missing topic",
			cfg: Config{
				KeyboardComponent: "kb",
				SpeedScale:        1.0,
				MinSpeed:          0.3,
				RampSteps:         10,
				KeyBindings:       map[string]VelocityBinding{"I": {VY: 1}},
			},
			want_ok: false,
		},
		{
			name: "zero speed scale",
			cfg: Config{
				KeyboardComponent: "kb",
				VelocityTopic:     "topic",
				SpeedScale:        0,
				MinSpeed:          0.3,
				RampSteps:         10,
				KeyBindings:       map[string]VelocityBinding{"I": {VY: 1}},
			},
			want_ok: false,
		},
		{
			name: "no bindings",
			cfg: Config{
				KeyboardComponent: "kb",
				VelocityTopic:     "topic",
				SpeedScale:        1.0,
				MinSpeed:          0.3,
				RampSteps:         10,
				KeyBindings:       map[string]VelocityBinding{},
			},
			want_ok: false,
		},
		{
			name: "min_speed zero",
			cfg: Config{
				KeyboardComponent: "kb",
				VelocityTopic:     "topic",
				SpeedScale:        1.0,
				MinSpeed:          0,
				RampSteps:         10,
				KeyBindings:       map[string]VelocityBinding{"I": {VY: 1}},
			},
			want_ok: false,
		},
		{
			name: "min_speed negative",
			cfg: Config{
				KeyboardComponent: "kb",
				VelocityTopic:     "topic",
				SpeedScale:        1.0,
				MinSpeed:          -0.1,
				RampSteps:         10,
				KeyBindings:       map[string]VelocityBinding{"I": {VY: 1}},
			},
			want_ok: false,
		},
		{
			name: "min_speed above 1",
			cfg: Config{
				KeyboardComponent: "kb",
				VelocityTopic:     "topic",
				SpeedScale:        1.0,
				MinSpeed:          1.1,
				RampSteps:         10,
				KeyBindings:       map[string]VelocityBinding{"I": {VY: 1}},
			},
			want_ok: false,
		},
		{
			name: "min_speed exactly 1",
			cfg: Config{
				KeyboardComponent: "kb",
				VelocityTopic:     "topic",
				SpeedScale:        1.0,
				MinSpeed:          1.0,
				RampSteps:         1,
				KeyBindings:       map[string]VelocityBinding{"I": {VY: 1}},
			},
			want_ok: true,
		},
		{
			name: "ramp_steps zero",
			cfg: Config{
				KeyboardComponent: "kb",
				VelocityTopic:     "topic",
				SpeedScale:        1.0,
				MinSpeed:          0.3,
				RampSteps:         0,
				KeyBindings:       map[string]VelocityBinding{"I": {VY: 1}},
			},
			want_ok: false,
		},
		{
			name: "ramp_steps negative",
			cfg: Config{
				KeyboardComponent: "kb",
				VelocityTopic:     "topic",
				SpeedScale:        1.0,
				MinSpeed:          0.3,
				RampSteps:         -1,
				KeyBindings:       map[string]VelocityBinding{"I": {VY: 1}},
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

func TestConfigFromResource(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"keyboard_component": "remote_keyboard",
			"velocity_topic":     "gorai.robot.velocity_input.command",
			"speed_scale":        0.5,
			"min_speed":          0.4,
			"ramp_steps":         float64(15),
			"key_bindings": map[string]any{
				"i": map[string]any{"vx": 0.0, "vy": 1.0, "omega": 0.0},
				"k": map[string]any{"vx": 0.0, "vy": -1.0, "omega": 0.0},
				"j": map[string]any{"vx": -1.0, "vy": 0.0, "omega": 0.0},
				"l": map[string]any{"vx": 1.0, "vy": 0.0, "omega": 0.0},
				"u": map[string]any{"vx": 0.0, "vy": 0.0, "omega": -1.0},
				"o": map[string]any{"vx": 0.0, "vy": 0.0, "omega": 1.0},
			},
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.KeyboardComponent != "remote_keyboard" {
		t.Errorf("KeyboardComponent = %q", cfg.KeyboardComponent)
	}
	if cfg.VelocityTopic != "gorai.robot.velocity_input.command" {
		t.Errorf("VelocityTopic = %q", cfg.VelocityTopic)
	}
	if cfg.SpeedScale != 0.5 {
		t.Errorf("SpeedScale = %f", cfg.SpeedScale)
	}
	if cfg.MinSpeed != 0.4 {
		t.Errorf("MinSpeed = %f, want 0.4", cfg.MinSpeed)
	}
	if cfg.RampSteps != 15 {
		t.Errorf("RampSteps = %d, want 15", cfg.RampSteps)
	}
	if len(cfg.KeyBindings) != 6 {
		t.Errorf("KeyBindings count = %d, want 6", len(cfg.KeyBindings))
	}

	if b, ok := cfg.KeyBindings["I"]; !ok {
		t.Error("expected 'I' binding (uppercase)")
	} else if b.VY != 1.0 {
		t.Errorf("I.VY = %f, want 1.0", b.VY)
	}

	if b, ok := cfg.KeyBindings["U"]; !ok {
		t.Error("expected 'U' binding")
	} else if b.Omega != -1.0 {
		t.Errorf("U.Omega = %f, want -1.0", b.Omega)
	}
}

func TestConfigDefaults(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"keyboard_component": "kb",
			"velocity_topic":     "topic",
			"key_bindings": map[string]any{
				"w": map[string]any{"vy": 1.0},
			},
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.SpeedScale != 1.0 {
		t.Errorf("SpeedScale default = %f, want 1.0", cfg.SpeedScale)
	}
	if cfg.MinSpeed != 0.3 {
		t.Errorf("MinSpeed default = %f, want 0.3", cfg.MinSpeed)
	}
	if cfg.RampSteps != 10 {
		t.Errorf("RampSteps default = %d, want 10", cfg.RampSteps)
	}
}

func TestStepSize(t *testing.T) {
	cfg := &Config{MinSpeed: 0.3, RampSteps: 10}
	expected := (1.0 - 0.3) / 10.0
	if math.Abs(cfg.StepSize()-expected) > 1e-9 {
		t.Errorf("StepSize = %f, want %f", cfg.StepSize(), expected)
	}

	cfg2 := &Config{MinSpeed: 1.0, RampSteps: 5}
	if cfg2.StepSize() != 0 {
		t.Errorf("StepSize with MinSpeed=1.0 should be 0, got %f", cfg2.StepSize())
	}

	cfg3 := &Config{MinSpeed: 0.5, RampSteps: 0}
	if cfg3.StepSize() != 0 {
		t.Errorf("StepSize with RampSteps=0 should be 0, got %f", cfg3.StepSize())
	}
}

func TestComputeVelocity(t *testing.T) {
	c := &Controller{
		config: &Config{
			SpeedScale: 0.5,
			MinSpeed:   0.3,
			RampSteps:  10,
			KeyBindings: map[string]VelocityBinding{
				"I": {VX: 0, VY: 1.0, Omega: 0},
				"K": {VX: 0, VY: -1.0, Omega: 0},
				"J": {VX: -1.0, VY: 0, Omega: 0},
				"L": {VX: 1.0, VY: 0, Omega: 0},
				"U": {VX: 0, VY: 0, Omega: -1.0},
				"O": {VX: 0, VY: 0, Omega: 1.0},
			},
		},
		active_keys: make(map[string]float64),
	}

	cmd := c.computeVelocity()
	if cmd.VX != 0 || cmd.VY != 0 || cmd.Omega != 0 {
		t.Errorf("zero: vx=%f vy=%f omega=%f", cmd.VX, cmd.VY, cmd.Omega)
	}

	// Forward key at full magnitude
	c.active_keys["I"] = 1.0
	cmd = c.computeVelocity()
	if cmd.VY != 0.5 {
		t.Errorf("forward full: vy=%f, want 0.5", cmd.VY)
	}
	if cmd.VX != 0 || cmd.Omega != 0 {
		t.Errorf("forward full: vx=%f omega=%f, want both 0", cmd.VX, cmd.Omega)
	}

	// Forward key at min_speed magnitude
	c.active_keys = make(map[string]float64)
	c.active_keys["I"] = 0.3
	cmd = c.computeVelocity()
	if math.Abs(cmd.VY-0.15) > 1e-9 {
		t.Errorf("forward min: vy=%f, want 0.15", cmd.VY)
	}

	// Forward + strafe right both at full magnitude
	c.active_keys["I"] = 1.0
	c.active_keys["L"] = 1.0
	cmd = c.computeVelocity()
	if cmd.VX != 0.5 || cmd.VY != 0.5 {
		t.Errorf("fwd+right: vx=%f vy=%f, want both 0.5", cmd.VX, cmd.VY)
	}

	// Forward + strafe right + rotate
	c.active_keys["O"] = 1.0
	cmd = c.computeVelocity()
	if cmd.Omega != 0.5 {
		t.Errorf("fwd+right+rot: omega=%f, want 0.5", cmd.Omega)
	}

	// Release all
	c.active_keys = make(map[string]float64)
	cmd = c.computeVelocity()
	if cmd.VX != 0 || cmd.VY != 0 || cmd.Omega != 0 {
		t.Errorf("released: vx=%f vy=%f omega=%f, want all 0", cmd.VX, cmd.VY, cmd.Omega)
	}
}

func TestComputeVelocityOpposingKeys(t *testing.T) {
	c := &Controller{
		config: &Config{
			SpeedScale: 1.0,
			MinSpeed:   0.3,
			RampSteps:  10,
			KeyBindings: map[string]VelocityBinding{
				"I": {VY: 1.0},
				"K": {VY: -1.0},
			},
		},
		active_keys: make(map[string]float64),
	}

	// Both at same magnitude -> cancel out
	c.active_keys["I"] = 0.5
	c.active_keys["K"] = 0.5
	cmd := c.computeVelocity()
	if cmd.VY != 0 {
		t.Errorf("opposing keys: vy=%f, want 0", cmd.VY)
	}
}

func TestProcessKeyEventRamp(t *testing.T) {
	cfg := &Config{
		SpeedScale: 1.0,
		MinSpeed:   0.2,
		RampSteps:  4,
		KeyBindings: map[string]VelocityBinding{
			"W": {VY: 1.0},
		},
	}
	step := cfg.StepSize() // (1.0 - 0.2) / 4 = 0.2

	c := &Controller{
		config:      cfg,
		active_keys: make(map[string]float64),
	}

	// Press: should set to min_speed
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: false})
	if mag, ok := c.active_keys["W"]; !ok {
		t.Fatal("W should be in active_keys after press")
	} else if math.Abs(mag-0.2) > 1e-9 {
		t.Errorf("press: magnitude=%f, want 0.2", mag)
	}

	// Repeat 1: 0.2 + 0.2 = 0.4
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: true})
	if mag := c.active_keys["W"]; math.Abs(mag-0.2-step) > 1e-9 {
		t.Errorf("repeat 1: magnitude=%f, want %f", mag, 0.2+step)
	}

	// Repeat 2: 0.4 + 0.2 = 0.6
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: true})
	if mag := c.active_keys["W"]; math.Abs(mag-0.6) > 1e-9 {
		t.Errorf("repeat 2: magnitude=%f, want 0.6", mag)
	}

	// Repeat 3: 0.6 + 0.2 = 0.8
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: true})
	if mag := c.active_keys["W"]; math.Abs(mag-0.8) > 1e-9 {
		t.Errorf("repeat 3: magnitude=%f, want 0.8", mag)
	}

	// Repeat 4: 0.8 + 0.2 = 1.0
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: true})
	if mag := c.active_keys["W"]; math.Abs(mag-1.0) > 1e-9 {
		t.Errorf("repeat 4: magnitude=%f, want 1.0", mag)
	}

	// Extra repeats should stay capped at 1.0
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: true})
	if mag := c.active_keys["W"]; mag != 1.0 {
		t.Errorf("repeat overflow: magnitude=%f, want 1.0", mag)
	}

	// Release: should remove from active_keys
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: false, Repeat: false})
	if _, ok := c.active_keys["W"]; ok {
		t.Error("W should be removed from active_keys after release")
	}
}

func TestProcessKeyEventUnboundKeyIgnored(t *testing.T) {
	c := &Controller{
		config: &Config{
			SpeedScale: 1.0,
			MinSpeed:   0.3,
			RampSteps:  10,
			KeyBindings: map[string]VelocityBinding{
				"W": {VY: 1.0},
			},
		},
		active_keys: make(map[string]float64),
	}

	c.processKeyEvent(input.KeyEvent{Key: "x", Pressed: true, Repeat: false})
	if len(c.active_keys) != 0 {
		t.Errorf("unbound key should not affect active_keys, got %d entries", len(c.active_keys))
	}
}

func TestProcessKeyEventRepeatWithoutPress(t *testing.T) {
	c := &Controller{
		config: &Config{
			SpeedScale: 1.0,
			MinSpeed:   0.3,
			RampSteps:  10,
			KeyBindings: map[string]VelocityBinding{
				"W": {VY: 1.0},
			},
		},
		active_keys: make(map[string]float64),
	}

	// Repeat for a key that was never pressed should be a no-op
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: true})
	if _, ok := c.active_keys["W"]; ok {
		t.Error("repeat without prior press should not add key to active_keys")
	}
}

func TestComputeVelocityMixedMagnitudes(t *testing.T) {
	c := &Controller{
		config: &Config{
			SpeedScale: 1.0,
			MinSpeed:   0.3,
			RampSteps:  10,
			KeyBindings: map[string]VelocityBinding{
				"W": {VY: 1.0},
				"D": {VX: 1.0},
			},
		},
		active_keys: make(map[string]float64),
	}

	// W at min_speed, D at full
	c.active_keys["W"] = 0.3
	c.active_keys["D"] = 1.0
	cmd := c.computeVelocity()
	if math.Abs(cmd.VY-0.3) > 1e-9 {
		t.Errorf("mixed: vy=%f, want 0.3", cmd.VY)
	}
	if math.Abs(cmd.VX-1.0) > 1e-9 {
		t.Errorf("mixed: vx=%f, want 1.0", cmd.VX)
	}
}
