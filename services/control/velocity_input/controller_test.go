package velocity_input

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
			name: "valid",
			cfg: Config{
				KeyboardComponent: "remote_keyboard",
				VelocityTopic:     "gorai.robot.velocity_input.command",
				SpeedScale:        0.5,
				KeyBindings: map[string]VelocityBinding{
					"I": {VX: 0, VY: 1, Omega: 0},
				},
			},
			want_ok: true,
		},
		{
			name: "missing keyboard",
			cfg: Config{
				VelocityTopic: "topic",
				SpeedScale:    1.0,
				KeyBindings:   map[string]VelocityBinding{"I": {VY: 1}},
			},
			want_ok: false,
		},
		{
			name: "missing topic",
			cfg: Config{
				KeyboardComponent: "kb",
				SpeedScale:        1.0,
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
				KeyBindings:       map[string]VelocityBinding{},
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
	if len(cfg.KeyBindings) != 6 {
		t.Errorf("KeyBindings count = %d, want 6", len(cfg.KeyBindings))
	}

	// Keys should be normalized to uppercase
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
}

func TestComputeVelocity(t *testing.T) {
	c := &Controller{
		config: &Config{
			SpeedScale: 0.5,
			KeyBindings: map[string]VelocityBinding{
				"I": {VX: 0, VY: 1.0, Omega: 0},
				"K": {VX: 0, VY: -1.0, Omega: 0},
				"J": {VX: -1.0, VY: 0, Omega: 0},
				"L": {VX: 1.0, VY: 0, Omega: 0},
				"U": {VX: 0, VY: 0, Omega: -1.0},
				"O": {VX: 0, VY: 0, Omega: 1.0},
			},
		},
		active_keys: make(map[string]bool),
	}

	// No keys pressed -> zero velocity
	cmd := c.computeVelocity()
	if cmd.VX != 0 || cmd.VY != 0 || cmd.Omega != 0 {
		t.Errorf("zero: vx=%f vy=%f omega=%f", cmd.VX, cmd.VY, cmd.Omega)
	}

	// Forward key
	c.active_keys["I"] = true
	cmd = c.computeVelocity()
	if cmd.VY != 0.5 {
		t.Errorf("forward: vy=%f, want 0.5", cmd.VY)
	}
	if cmd.VX != 0 || cmd.Omega != 0 {
		t.Errorf("forward: vx=%f omega=%f, want both 0", cmd.VX, cmd.Omega)
	}

	// Forward + strafe right
	c.active_keys["L"] = true
	cmd = c.computeVelocity()
	if cmd.VX != 0.5 || cmd.VY != 0.5 {
		t.Errorf("fwd+right: vx=%f vy=%f, want both 0.5", cmd.VX, cmd.VY)
	}

	// Forward + strafe right + rotate
	c.active_keys["O"] = true
	cmd = c.computeVelocity()
	if cmd.Omega != 0.5 {
		t.Errorf("fwd+right+rot: omega=%f, want 0.5", cmd.Omega)
	}

	// Release all
	c.active_keys = make(map[string]bool)
	cmd = c.computeVelocity()
	if cmd.VX != 0 || cmd.VY != 0 || cmd.Omega != 0 {
		t.Errorf("released: vx=%f vy=%f omega=%f, want all 0", cmd.VX, cmd.VY, cmd.Omega)
	}
}

func TestComputeVelocityOpposingKeys(t *testing.T) {
	c := &Controller{
		config: &Config{
			SpeedScale: 1.0,
			KeyBindings: map[string]VelocityBinding{
				"I": {VY: 1.0},
				"K": {VY: -1.0},
			},
		},
		active_keys: make(map[string]bool),
	}

	// Both forward and backward pressed -> cancel out
	c.active_keys["I"] = true
	c.active_keys["K"] = true
	cmd := c.computeVelocity()
	if cmd.VY != 0 {
		t.Errorf("opposing keys: vy=%f, want 0", cmd.VY)
	}
}
