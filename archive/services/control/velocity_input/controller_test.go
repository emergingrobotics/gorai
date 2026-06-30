package velocity_input

import (
	"math"
	"testing"

	"github.com/emergingrobotics/gorai/components/input"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func validConfig() Config {
	return Config{
		VelocityTopic: "gorai.robot.velocity_input.command",
		InputType:     "keyboard",
		Keyboard: KeyboardConfig{
			InputComponent: "remote_keyboard",
			SetSpeed:       0.25,
			KeyBindings: map[string]VelocityBinding{
				"W": {VX: 1, VY: 0, Omega: 0},
				"S": {VX: -1, VY: 0, Omega: 0},
				"A": {VX: 0, VY: 1, Omega: 0},
				"D": {VX: 0, VY: -1, Omega: 0},
				"Q": {VX: 0, VY: 0, Omega: -1},
				"E": {VX: 0, VY: 0, Omega: 1},
			},
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
			name: "missing velocity_topic",
			cfg: Config{
				InputType: "keyboard",
				Keyboard: KeyboardConfig{
					InputComponent: "remote_keyboard",
					SetSpeed:       0.25,
					KeyBindings:    map[string]VelocityBinding{"W": {VX: 1}},
				},
			},
			want_ok: false,
		},
		{
			name: "unsupported input_type",
			cfg: Config{
				VelocityTopic: "topic",
				InputType:     "joystick",
				Keyboard: KeyboardConfig{
					InputComponent: "kb",
					SetSpeed:       0.25,
					KeyBindings:    map[string]VelocityBinding{"W": {VX: 1}},
				},
			},
			want_ok: false,
		},
		{
			name: "missing keyboard input_component",
			cfg: Config{
				VelocityTopic: "topic",
				InputType:     "keyboard",
				Keyboard: KeyboardConfig{
					SetSpeed:    0.25,
					KeyBindings: map[string]VelocityBinding{"W": {VX: 1}},
				},
			},
			want_ok: false,
		},
		{
			name: "no key_bindings",
			cfg: Config{
				VelocityTopic: "topic",
				InputType:     "keyboard",
				Keyboard: KeyboardConfig{
					InputComponent: "kb",
					SetSpeed:       0.25,
					KeyBindings:    map[string]VelocityBinding{},
				},
			},
			want_ok: false,
		},
		{
			name: "set_speed negative",
			cfg: Config{
				VelocityTopic: "topic",
				InputType:     "keyboard",
				Keyboard: KeyboardConfig{
					InputComponent: "kb",
					SetSpeed:       -0.1,
					KeyBindings:    map[string]VelocityBinding{"W": {VX: 1}},
				},
			},
			want_ok: false,
		},
		{
			name: "set_speed above 1",
			cfg: Config{
				VelocityTopic: "topic",
				InputType:     "keyboard",
				Keyboard: KeyboardConfig{
					InputComponent: "kb",
					SetSpeed:       1.1,
					KeyBindings:    map[string]VelocityBinding{"W": {VX: 1}},
				},
			},
			want_ok: false,
		},
		{
			name: "set_speed exactly 1",
			cfg: Config{
				VelocityTopic: "topic",
				InputType:     "keyboard",
				Keyboard: KeyboardConfig{
					InputComponent: "kb",
					SetSpeed:       1.0,
					KeyBindings:    map[string]VelocityBinding{"W": {VX: 1}},
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

func TestConfigFromResource(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"velocity_topic": "gorai.robot.velocity_input.command",
			"input_type":     "keyboard",
			"keyboard": map[string]any{
				"input_component": "remote_keyboard",
				"set_speed":       0.5,
				"key_bindings": map[string]any{
					"w": map[string]any{"vx": 1.0, "vy": 0.0, "omega": 0.0},
					"s": map[string]any{"vx": -1.0, "vy": 0.0, "omega": 0.0},
					"a": map[string]any{"vx": 0.0, "vy": 1.0, "omega": 0.0},
					"d": map[string]any{"vx": 0.0, "vy": -1.0, "omega": 0.0},
					"q": map[string]any{"vx": 0.0, "vy": 0.0, "omega": -1.0},
					"e": map[string]any{"vx": 0.0, "vy": 0.0, "omega": 1.0},
				},
			},
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.VelocityTopic != "gorai.robot.velocity_input.command" {
		t.Errorf("VelocityTopic = %q", cfg.VelocityTopic)
	}
	if cfg.InputType != "keyboard" {
		t.Errorf("InputType = %q", cfg.InputType)
	}
	if cfg.Keyboard.InputComponent != "remote_keyboard" {
		t.Errorf("Keyboard.InputComponent = %q", cfg.Keyboard.InputComponent)
	}
	if cfg.Keyboard.SetSpeed != 0.5 {
		t.Errorf("Keyboard.SetSpeed = %f", cfg.Keyboard.SetSpeed)
	}
	if len(cfg.Keyboard.KeyBindings) != 6 {
		t.Errorf("KeyBindings count = %d, want 6", len(cfg.Keyboard.KeyBindings))
	}

	if b, ok := cfg.Keyboard.KeyBindings["W"]; !ok {
		t.Error("expected 'W' binding (uppercase)")
	} else if b.VX != 1.0 {
		t.Errorf("W.VX = %f, want 1.0", b.VX)
	}

	if b, ok := cfg.Keyboard.KeyBindings["A"]; !ok {
		t.Error("expected 'A' binding")
	} else if b.VY != 1.0 {
		t.Errorf("A.VY = %f, want 1.0", b.VY)
	}

	if b, ok := cfg.Keyboard.KeyBindings["Q"]; !ok {
		t.Error("expected 'Q' binding")
	} else if b.Omega != -1.0 {
		t.Errorf("Q.Omega = %f, want -1.0", b.Omega)
	}
}

func TestConfigDefaults(t *testing.T) {
	conf := resource.Config{
		Attributes: map[string]any{
			"velocity_topic": "topic",
			"keyboard": map[string]any{
				"input_component": "kb",
				"key_bindings": map[string]any{
					"w": map[string]any{"vx": 1.0},
				},
			},
		},
	}

	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg.InputType != "keyboard" {
		t.Errorf("InputType default = %q", cfg.InputType)
	}
	if cfg.Keyboard.SetSpeed != 0.25 {
		t.Errorf("Keyboard.SetSpeed default = %f, want 0.25", cfg.Keyboard.SetSpeed)
	}
}

func TestComputeVelocity(t *testing.T) {
	c := &Controller{
		config: &Config{
			Keyboard: KeyboardConfig{
				SetSpeed: 0.5,
				KeyBindings: map[string]VelocityBinding{
					"W": {VX: 1, VY: 0, Omega: 0},
					"S": {VX: -1, VY: 0, Omega: 0},
					"A": {VX: 0, VY: 1, Omega: 0},
					"D": {VX: 0, VY: -1, Omega: 0},
					"Q": {VX: 0, VY: 0, Omega: -1},
					"E": {VX: 0, VY: 0, Omega: 1},
				},
			},
		},
		active_keys: make(map[string]struct{}),
	}

	cmd := c.computeVelocity()
	if cmd.VX != 0 || cmd.VY != 0 || cmd.Omega != 0 || cmd.SetSpeed != 0 {
		t.Errorf("zero: vx=%f vy=%f omega=%f set_speed=%f", cmd.VX, cmd.VY, cmd.Omega, cmd.SetSpeed)
	}

	// Single key: forward
	c.active_keys["W"] = struct{}{}
	cmd = c.computeVelocity()
	if cmd.VX != 1.0 || cmd.VY != 0 || cmd.Omega != 0 || cmd.SetSpeed != 0.5 {
		t.Errorf("forward: vx=%f vy=%f omega=%f set_speed=%f, want 1,0,0,0.5", cmd.VX, cmd.VY, cmd.Omega, cmd.SetSpeed)
	}

	// Two keys: diagonal
	c.active_keys = make(map[string]struct{})
	c.active_keys["W"] = struct{}{}
	c.active_keys["D"] = struct{}{}
	cmd = c.computeVelocity()
	inv_sqrt2 := 1.0 / math.Sqrt(2)
	if math.Abs(cmd.VX-inv_sqrt2) > 1e-9 || math.Abs(cmd.VY+inv_sqrt2) > 1e-9 ||
		cmd.Omega != 0 || cmd.SetSpeed != 0.5 {
		t.Errorf("diagonal: vx=%f vy=%f omega=%f set_speed=%f, want vx=vy~=0.707", cmd.VX, cmd.VY, cmd.Omega, cmd.SetSpeed)
	}

	// Release all
	c.active_keys = make(map[string]struct{})
	cmd = c.computeVelocity()
	if cmd.VX != 0 || cmd.VY != 0 || cmd.Omega != 0 || cmd.SetSpeed != 0 {
		t.Errorf("released: vx=%f vy=%f omega=%f set_speed=%f", cmd.VX, cmd.VY, cmd.Omega, cmd.SetSpeed)
	}
}

func TestComputeVelocityOpposingKeys(t *testing.T) {
	c := &Controller{
		config: &Config{
			Keyboard: KeyboardConfig{
				SetSpeed: 0.25,
				KeyBindings: map[string]VelocityBinding{
					"W": {VX: 1, VY: 0, Omega: 0},
					"S": {VX: -1, VY: 0, Omega: 0},
				},
			},
		},
		active_keys: make(map[string]struct{}),
	}

	c.active_keys["W"] = struct{}{}
	c.active_keys["S"] = struct{}{}
	cmd := c.computeVelocity()
	if cmd.VX != 0 || cmd.VY != 0 || cmd.Omega != 0 {
		t.Errorf("opposing keys: vx=%f vy=%f omega=%f, want 0,0,0", cmd.VX, cmd.VY, cmd.Omega)
	}
	if cmd.SetSpeed != 0.25 {
		t.Errorf("opposing keys: set_speed=%f, want 0.25 (keys still active)", cmd.SetSpeed)
	}
}

func TestProcessKeyEvent(t *testing.T) {
	c := &Controller{
		config: &Config{
			Keyboard: KeyboardConfig{
				SetSpeed: 0.5,
				KeyBindings: map[string]VelocityBinding{
					"W": {VX: 1, VY: 0, Omega: 0},
				},
			},
		},
		active_keys: make(map[string]struct{}),
	}

	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: false})
	if _, ok := c.active_keys["W"]; !ok {
		t.Error("W should be in active_keys after press")
	}

	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: false, Repeat: false})
	if _, ok := c.active_keys["W"]; ok {
		t.Error("W should be removed from active_keys after release")
	}
}

func TestProcessKeyEventRepeatAddsKey(t *testing.T) {
	c := &Controller{
		config: &Config{
			Keyboard: KeyboardConfig{
				SetSpeed: 0.5,
				KeyBindings: map[string]VelocityBinding{
					"W": {VX: 1, VY: 0, Omega: 0},
				},
			},
		},
		active_keys: make(map[string]struct{}),
	}

	// Repeat event (key held) adds key to active set
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: true})
	if _, ok := c.active_keys["W"]; !ok {
		t.Error("repeat with Pressed=true should add W to active_keys")
	}
}

func TestProcessKeyEventUnboundKeyIgnored(t *testing.T) {
	c := &Controller{
		config: &Config{
			Keyboard: KeyboardConfig{
				SetSpeed: 0.5,
				KeyBindings: map[string]VelocityBinding{
					"W": {VX: 1, VY: 0, Omega: 0},
				},
			},
		},
		active_keys: make(map[string]struct{}),
	}

	c.processKeyEvent(input.KeyEvent{Key: "x", Pressed: true, Repeat: false})
	if len(c.active_keys) != 0 {
		t.Errorf("unbound key should not affect active_keys, got %d entries", len(c.active_keys))
	}
}

func TestProcessKeyEventDedup(t *testing.T) {
	c := &Controller{
		config: &Config{
			Keyboard: KeyboardConfig{
				SetSpeed: 0.5,
				KeyBindings: map[string]VelocityBinding{
					"W": {VX: 1, VY: 0, Omega: 0},
				},
			},
		},
		active_keys: make(map[string]struct{}),
	}

	// Press: first publish
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: false})
	if !c.has_published {
		t.Fatal("should have published after first press")
	}
	if c.last_published.VX != 1.0 || c.last_published.SetSpeed != 0.5 {
		t.Errorf("last_published = vx=%f set_speed=%f", c.last_published.VX, c.last_published.SetSpeed)
	}

	// Repeat: same cmd, no publish (dedup)
	prev := c.last_published
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: true})
	if c.last_published != prev {
		t.Error("repeat with same velocity should not change last_published (dedup)")
	}

	// Release: different cmd, publish
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: false, Repeat: false})
	if c.last_published.VX != 0 || c.last_published.SetSpeed != 0 {
		t.Errorf("after release: vx=%f set_speed=%f, want 0,0", c.last_published.VX, c.last_published.SetSpeed)
	}
}

func TestProcessKeyEventDedupResetOnReconfigure(t *testing.T) {
	c := &Controller{
		config: &Config{
			Keyboard: KeyboardConfig{
				SetSpeed: 1.0,
				KeyBindings: map[string]VelocityBinding{
					"W": {VX: 1, VY: 0, Omega: 0},
				},
			},
		},
		active_keys: make(map[string]struct{}),
	}

	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: false})
	if !c.has_published {
		t.Fatal("should have published")
	}

	c.mu.Lock()
	c.active_keys = make(map[string]struct{})
	c.has_published = false
	c.mu.Unlock()

	c.active_keys["W"] = struct{}{}
	c.processKeyEvent(input.KeyEvent{Key: "w", Pressed: true, Repeat: false})
	if c.last_published.VX != 1.0 {
		t.Errorf("after reset, press should publish, got VX=%f", c.last_published.VX)
	}
}

func TestComputeVelocityNormalizedDirection(t *testing.T) {
	c := &Controller{
		config: &Config{
			Keyboard: KeyboardConfig{
				SetSpeed: 0.5,
				KeyBindings: map[string]VelocityBinding{
					"W": {VX: 1, VY: 0, Omega: 0},
					"D": {VX: 0, VY: -1, Omega: 0},
				},
			},
		},
		active_keys: make(map[string]struct{}),
	}

	c.active_keys["W"] = struct{}{}
	c.active_keys["D"] = struct{}{}
	cmd := c.computeVelocity()

	mag := math.Sqrt(cmd.VX*cmd.VX + cmd.VY*cmd.VY + cmd.Omega*cmd.Omega)
	if math.Abs(mag-1.0) > 1e-9 {
		t.Errorf("direction magnitude = %f, want 1.0 (normalized)", mag)
	}
	if cmd.SetSpeed != 0.5 {
		t.Errorf("set_speed = %f, want 0.5", cmd.SetSpeed)
	}
}
