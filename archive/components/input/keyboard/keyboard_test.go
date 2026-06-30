package keyboard

import (
	"testing"

	"github.com/emergingrobotics/gorai/components/input"
	"github.com/emergingrobotics/gorai/pkg/resource"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
	}{
		{
			name: "valid config",
			config: &Config{
				Device: "/dev/input/event0",
			},
			wantErr: false,
		},
		{
			name: "missing device",
			config: &Config{
				Device: "",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestConfigParsing(t *testing.T) {
	attrs := map[string]any{
		"device":         "/dev/input/event0",
		"grab_device":    true,
		"publish_repeat": true,
	}

	conf := resource.NewConfig(attrs)
	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	assert.Equal(t, "/dev/input/event0", cfg.Device)
	assert.True(t, cfg.GrabDevice)
	assert.True(t, cfg.PublishRepeat)
}

func TestConfigDefaults(t *testing.T) {
	attrs := map[string]any{
		"device": "/dev/input/event0",
	}

	conf := resource.NewConfig(attrs)
	cfg, err := NewConfigFromResource(conf)
	require.NoError(t, err)

	assert.Equal(t, "/dev/input/event0", cfg.Device)
	assert.False(t, cfg.GrabDevice)
	assert.False(t, cfg.PublishRepeat)
}

func TestKeyCodeToName(t *testing.T) {
	tests := []struct {
		code uint16
		want string
	}{
		{KeyW, "W"},
		{KeyA, "A"},
		{KeyS, "S"},
		{KeyD, "D"},
		{KeySpace, "SPACE"},
		{KeyEnter, "ENTER"},
		{KeyLeftShift, "LSHIFT"},
		{KeyRightCtrl, "RCTRL"},
		{KeyUp, "UP"},
		{KeyDown, "DOWN"},
		{KeyLeft, "LEFT"},
		{KeyRight, "RIGHT"},
		{KeyF1, "F1"},
		{KeyF12, "F12"},
		{KeyEsc, "ESC"},
		{12345, "KEY_12345"}, // Unknown key
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := KeyCodeToName(tt.code)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestKeyNameToCode(t *testing.T) {
	tests := []struct {
		name    string
		wantOK  bool
		wantCode uint16
	}{
		{"W", true, KeyW},
		{"SPACE", true, KeySpace},
		{"UP", true, KeyUp},
		{"LSHIFT", true, KeyLeftShift},
		{"UNKNOWN", false, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, ok := KeyNameToCode(tt.name)
			assert.Equal(t, tt.wantOK, ok)
			if tt.wantOK {
				assert.Equal(t, tt.wantCode, code)
			}
		})
	}
}

func TestIsModifierKey(t *testing.T) {
	modifiers := []uint16{
		KeyLeftShift, KeyRightShift,
		KeyLeftCtrl, KeyRightCtrl,
		KeyLeftAlt, KeyRightAlt,
		KeyLeftMeta, KeyRightMeta,
		KeyCapsLock, KeyNumLock,
	}

	nonModifiers := []uint16{
		KeyW, KeyA, KeyS, KeyD,
		KeySpace, KeyEnter,
		KeyUp, KeyDown,
	}

	for _, code := range modifiers {
		t.Run(KeyCodeToName(code)+"_is_modifier", func(t *testing.T) {
			assert.True(t, IsModifierKey(code))
		})
	}

	for _, code := range nonModifiers {
		t.Run(KeyCodeToName(code)+"_not_modifier", func(t *testing.T) {
			assert.False(t, IsModifierKey(code))
		})
	}
}

func TestParseInputEvent(t *testing.T) {
	// Create a valid input_event buffer (24 bytes on 64-bit)
	// TimeSec: 1234567890 (0x499602D2)
	// TimeUsec: 123456 (0x1E240)
	// Type: 1 (EV_KEY)
	// Code: 17 (KEY_W)
	// Value: 1 (press)
	buf := []byte{
		// TimeSec (8 bytes, little-endian)
		0xD2, 0x02, 0x96, 0x49, 0x00, 0x00, 0x00, 0x00,
		// TimeUsec (8 bytes, little-endian)
		0x40, 0xE2, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
		// Type (2 bytes, little-endian)
		0x01, 0x00,
		// Code (2 bytes, little-endian)
		0x11, 0x00,
		// Value (4 bytes, little-endian)
		0x01, 0x00, 0x00, 0x00,
	}

	ev, err := parseInputEvent(buf)
	require.NoError(t, err)

	assert.Equal(t, int64(1234567890), ev.TimeSec)
	assert.Equal(t, int64(123456), ev.TimeUsec)
	assert.Equal(t, uint16(1), ev.Type)
	assert.Equal(t, uint16(17), ev.Code)
	assert.Equal(t, int32(1), ev.Value)
	assert.True(t, ev.isKeyEvent())
	assert.True(t, ev.isPress())
	assert.False(t, ev.isRelease())
	assert.False(t, ev.isRepeat())
}

func TestParseInputEventRelease(t *testing.T) {
	buf := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // TimeSec
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // TimeUsec
		0x01, 0x00, // Type: EV_KEY
		0x11, 0x00, // Code: KEY_W
		0x00, 0x00, 0x00, 0x00, // Value: 0 (release)
	}

	ev, err := parseInputEvent(buf)
	require.NoError(t, err)

	assert.True(t, ev.isRelease())
	assert.False(t, ev.isPress())
}

func TestParseInputEventRepeat(t *testing.T) {
	buf := []byte{
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // TimeSec
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // TimeUsec
		0x01, 0x00, // Type: EV_KEY
		0x11, 0x00, // Code: KEY_W
		0x02, 0x00, 0x00, 0x00, // Value: 2 (repeat)
	}

	ev, err := parseInputEvent(buf)
	require.NoError(t, err)

	assert.True(t, ev.isRepeat())
	assert.False(t, ev.isPress())
	assert.False(t, ev.isRelease())
}

func TestParseInputEventBufferTooSmall(t *testing.T) {
	buf := []byte{0x00, 0x01, 0x02} // Only 3 bytes

	_, err := parseInputEvent(buf)
	assert.Error(t, err)
}

func TestModifiersUpdate(t *testing.T) {
	k := &Keyboard{
		pressedKeys: make(map[uint16]bool),
	}

	// Press left shift
	k.updateModifiers(KeyLeftShift, true)
	assert.True(t, k.modifiers.Shift)
	assert.True(t, k.leftShiftPressed)

	// Press right shift too
	k.updateModifiers(KeyRightShift, true)
	assert.True(t, k.modifiers.Shift)
	assert.True(t, k.rightShiftPressed)

	// Release left shift - right still pressed
	k.updateModifiers(KeyLeftShift, false)
	assert.True(t, k.modifiers.Shift)

	// Release right shift - both released
	k.updateModifiers(KeyRightShift, false)
	assert.False(t, k.modifiers.Shift)
}

func TestModifiersToggle(t *testing.T) {
	k := &Keyboard{
		pressedKeys: make(map[uint16]bool),
	}

	// Caps lock toggles
	assert.False(t, k.modifiers.CapsLock)
	k.updateModifiers(KeyCapsLock, true) // Press
	assert.True(t, k.modifiers.CapsLock)
	k.updateModifiers(KeyCapsLock, false) // Release (no change)
	assert.True(t, k.modifiers.CapsLock)
	k.updateModifiers(KeyCapsLock, true) // Press again
	assert.False(t, k.modifiers.CapsLock)
}

func TestStateString(t *testing.T) {
	tests := []struct {
		state State
		want  string
	}{
		{StateClosed, "closed"},
		{StateOpening, "opening"},
		{StateRunning, "running"},
		{StateDisconnected, "disconnected"},
		{StateError, "error"},
		{State(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.state.String())
		})
	}
}

func TestKeyboardInterface(t *testing.T) {
	// Verify the interface is correctly implemented
	var _ input.Keyboard = (*Keyboard)(nil)
}

