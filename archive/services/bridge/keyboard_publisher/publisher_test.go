package keyboard_publisher

import (
	"testing"
	"time"

	"github.com/gorai/gorai/pkg/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name      string
		config    *Config
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid config",
			config: &Config{
				Keyboard:            "main_keyboard",
				HeartbeatIntervalMs: 1000,
			},
			expectErr: false,
		},
		{
			name: "missing keyboard",
			config: &Config{
				Keyboard:            "",
				HeartbeatIntervalMs: 1000,
			},
			expectErr: true,
			errMsg:    "keyboard component name is required",
		},
		{
			name: "heartbeat too low",
			config: &Config{
				Keyboard:            "main_keyboard",
				HeartbeatIntervalMs: 50,
			},
			expectErr: true,
			errMsg:    "heartbeat_interval_ms must be at least 100",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.expectErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigParsing(t *testing.T) {
	resConf := resource.Config{
		Attributes: map[string]any{
			"keyboard":             "my_keyboard",
			"topic":                "gorai.test.keyboard.events",
			"publish_repeat":       true,
			"heartbeat_interval_ms": 2000.0,
			"include_modifiers":    false,
			"robot_name":           "test-robot",
		},
	}

	cfg, err := NewConfigFromResource(resConf)
	require.NoError(t, err)

	assert.Equal(t, "my_keyboard", cfg.Keyboard)
	assert.Equal(t, "gorai.test.keyboard.events", cfg.Topic)
	assert.True(t, cfg.PublishRepeat)
	assert.Equal(t, 2000, cfg.HeartbeatIntervalMs)
	assert.False(t, cfg.IncludeModifiers)
	assert.Equal(t, "test-robot", cfg.RobotName)
}

func TestConfigDefaults(t *testing.T) {
	resConf := resource.Config{
		Attributes: map[string]any{
			"keyboard": "test_keyboard",
		},
	}

	cfg, err := NewConfigFromResource(resConf)
	require.NoError(t, err)

	assert.Equal(t, "test_keyboard", cfg.Keyboard)
	assert.Equal(t, "", cfg.Topic)
	assert.False(t, cfg.PublishRepeat)
	assert.Equal(t, 1000, cfg.HeartbeatIntervalMs)
	assert.True(t, cfg.IncludeModifiers)
}

func TestHeartbeatInterval(t *testing.T) {
	cfg := &Config{HeartbeatIntervalMs: 500}
	assert.Equal(t, 500*time.Millisecond, cfg.HeartbeatInterval())
}

func TestGetTopic(t *testing.T) {
	tests := []struct {
		name          string
		config        *Config
		expectedTopic string
	}{
		{
			name: "explicit topic",
			config: &Config{
				Topic:     "custom.topic.events",
				RobotName: "robot",
			},
			expectedTopic: "custom.topic.events",
		},
		{
			name: "default with robot name",
			config: &Config{
				Topic:     "",
				RobotName: "my-robot",
			},
			expectedTopic: "gorai.my-robot.keyboard.events",
		},
		{
			name: "default without robot name",
			config: &Config{
				Topic:     "",
				RobotName: "",
			},
			expectedTopic: "gorai.keyboard.events",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedTopic, tt.config.GetTopic())
		})
	}
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state    State
		expected string
	}{
		{StateStopped, "stopped"},
		{StateStarting, "starting"},
		{StateRunning, "running"},
		{StateDisconnected, "disconnected"},
		{StateError, "error"},
		{State(100), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.state.String())
		})
	}
}

func TestEncodeJSON(t *testing.T) {
	msg := KeyEventMessage{
		Timestamp: 1234567890,
		Seq:       42,
		Key:       "W",
		Code:      17,
		Pressed:   true,
		Repeat:    false,
		Modifiers: ModifiersData{
			Shift: true,
			Ctrl:  false,
			Alt:   true,
			Meta:  false,
		},
	}

	data, err := encodeJSON(msg)
	require.NoError(t, err)

	// Verify it's valid JSON-ish
	json := string(data)
	assert.Contains(t, json, `"key":"W"`)
	assert.Contains(t, json, `"code":17`)
	assert.Contains(t, json, `"pressed":true`)
	assert.Contains(t, json, `"repeat":false`)
	assert.Contains(t, json, `"shift":true`)
	assert.Contains(t, json, `"alt":true`)
}

func TestModifiersData(t *testing.T) {
	mods := ModifiersData{
		Shift:    true,
		Ctrl:     true,
		Alt:      false,
		Meta:     false,
		CapsLock: true,
		NumLock:  false,
	}

	assert.True(t, mods.Shift)
	assert.True(t, mods.Ctrl)
	assert.False(t, mods.Alt)
	assert.False(t, mods.Meta)
	assert.True(t, mods.CapsLock)
	assert.False(t, mods.NumLock)
}

