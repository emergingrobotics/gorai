package keyboard

import "fmt"

// Linux key codes from input-event-codes.h
const (
	KeyEsc        uint16 = 1
	Key1          uint16 = 2
	Key2          uint16 = 3
	Key3          uint16 = 4
	Key4          uint16 = 5
	Key5          uint16 = 6
	Key6          uint16 = 7
	Key7          uint16 = 8
	Key8          uint16 = 9
	Key9          uint16 = 10
	Key0          uint16 = 11
	KeyMinus      uint16 = 12
	KeyEqual      uint16 = 13
	KeyBackspace  uint16 = 14
	KeyTab        uint16 = 15
	KeyQ          uint16 = 16
	KeyW          uint16 = 17
	KeyE          uint16 = 18
	KeyR          uint16 = 19
	KeyT          uint16 = 20
	KeyY          uint16 = 21
	KeyU          uint16 = 22
	KeyI          uint16 = 23
	KeyO          uint16 = 24
	KeyP          uint16 = 25
	KeyLeftBrace  uint16 = 26
	KeyRightBrace uint16 = 27
	KeyEnter      uint16 = 28
	KeyLeftCtrl   uint16 = 29
	KeyA          uint16 = 30
	KeyS          uint16 = 31
	KeyD          uint16 = 32
	KeyF          uint16 = 33
	KeyG          uint16 = 34
	KeyH          uint16 = 35
	KeyJ          uint16 = 36
	KeyK          uint16 = 37
	KeyL          uint16 = 38
	KeySemicolon  uint16 = 39
	KeyApostrophe uint16 = 40
	KeyGrave      uint16 = 41
	KeyLeftShift  uint16 = 42
	KeyBackslash  uint16 = 43
	KeyZ          uint16 = 44
	KeyX          uint16 = 45
	KeyC          uint16 = 46
	KeyV          uint16 = 47
	KeyB          uint16 = 48
	KeyN          uint16 = 49
	KeyM          uint16 = 50
	KeyComma      uint16 = 51
	KeyDot        uint16 = 52
	KeySlash      uint16 = 53
	KeyRightShift uint16 = 54
	KeyKpAsterisk uint16 = 55
	KeyLeftAlt    uint16 = 56
	KeySpace      uint16 = 57
	KeyCapsLock   uint16 = 58
	KeyF1         uint16 = 59
	KeyF2         uint16 = 60
	KeyF3         uint16 = 61
	KeyF4         uint16 = 62
	KeyF5         uint16 = 63
	KeyF6         uint16 = 64
	KeyF7         uint16 = 65
	KeyF8         uint16 = 66
	KeyF9         uint16 = 67
	KeyF10        uint16 = 68
	KeyNumLock    uint16 = 69
	KeyScrollLock uint16 = 70
	KeyKp7        uint16 = 71
	KeyKp8        uint16 = 72
	KeyKp9        uint16 = 73
	KeyKpMinus    uint16 = 74
	KeyKp4        uint16 = 75
	KeyKp5        uint16 = 76
	KeyKp6        uint16 = 77
	KeyKpPlus     uint16 = 78
	KeyKp1        uint16 = 79
	KeyKp2        uint16 = 80
	KeyKp3        uint16 = 81
	KeyKp0        uint16 = 82
	KeyKpDot      uint16 = 83
	KeyF11        uint16 = 87
	KeyF12        uint16 = 88
	KeyKpEnter    uint16 = 96
	KeyRightCtrl  uint16 = 97
	KeyKpSlash    uint16 = 98
	KeySysRq      uint16 = 99
	KeyRightAlt   uint16 = 100
	KeyHome       uint16 = 102
	KeyUp         uint16 = 103
	KeyPageUp     uint16 = 104
	KeyLeft       uint16 = 105
	KeyRight      uint16 = 106
	KeyEnd        uint16 = 107
	KeyDown       uint16 = 108
	KeyPageDown   uint16 = 109
	KeyInsert     uint16 = 110
	KeyDelete     uint16 = 111
	KeyPause      uint16 = 119
	KeyLeftMeta   uint16 = 125
	KeyRightMeta  uint16 = 126
	KeyCompose    uint16 = 127
)

// keyCodeToName maps Linux key codes to human-readable names.
var keyCodeToName = map[uint16]string{
	// Escape and function keys
	KeyEsc: "ESC",
	KeyF1:  "F1",
	KeyF2:  "F2",
	KeyF3:  "F3",
	KeyF4:  "F4",
	KeyF5:  "F5",
	KeyF6:  "F6",
	KeyF7:  "F7",
	KeyF8:  "F8",
	KeyF9:  "F9",
	KeyF10: "F10",
	KeyF11: "F11",
	KeyF12: "F12",

	// Number row
	Key1:     "1",
	Key2:     "2",
	Key3:     "3",
	Key4:     "4",
	Key5:     "5",
	Key6:     "6",
	Key7:     "7",
	Key8:     "8",
	Key9:     "9",
	Key0:     "0",
	KeyMinus: "MINUS",
	KeyEqual: "EQUAL",

	// Top letter row
	KeyQ: "Q",
	KeyW: "W",
	KeyE: "E",
	KeyR: "R",
	KeyT: "T",
	KeyY: "Y",
	KeyU: "U",
	KeyI: "I",
	KeyO: "O",
	KeyP: "P",

	// Middle letter row
	KeyA: "A",
	KeyS: "S",
	KeyD: "D",
	KeyF: "F",
	KeyG: "G",
	KeyH: "H",
	KeyJ: "J",
	KeyK: "K",
	KeyL: "L",

	// Bottom letter row
	KeyZ: "Z",
	KeyX: "X",
	KeyC: "C",
	KeyV: "V",
	KeyB: "B",
	KeyN: "N",
	KeyM: "M",

	// Special keys
	KeyBackspace:  "BACKSPACE",
	KeyTab:        "TAB",
	KeyEnter:      "ENTER",
	KeySpace:      "SPACE",
	KeyLeftBrace:  "LBRACE",
	KeyRightBrace: "RBRACE",
	KeySemicolon:  "SEMICOLON",
	KeyApostrophe: "APOSTROPHE",
	KeyGrave:      "GRAVE",
	KeyBackslash:  "BACKSLASH",
	KeyComma:      "COMMA",
	KeyDot:        "DOT",
	KeySlash:      "SLASH",

	// Modifiers
	KeyLeftShift:  "LSHIFT",
	KeyRightShift: "RSHIFT",
	KeyLeftCtrl:   "LCTRL",
	KeyRightCtrl:  "RCTRL",
	KeyLeftAlt:    "LALT",
	KeyRightAlt:   "RALT",
	KeyLeftMeta:   "LMETA",
	KeyRightMeta:  "RMETA",
	KeyCapsLock:   "CAPSLOCK",
	KeyNumLock:    "NUMLOCK",
	KeyScrollLock: "SCROLLLOCK",

	// Arrow keys
	KeyUp:    "UP",
	KeyDown:  "DOWN",
	KeyLeft:  "LEFT",
	KeyRight: "RIGHT",

	// Navigation
	KeyHome:     "HOME",
	KeyEnd:      "END",
	KeyPageUp:   "PAGEUP",
	KeyPageDown: "PAGEDOWN",
	KeyInsert:   "INSERT",
	KeyDelete:   "DELETE",

	// Numpad
	KeyKp0:        "KP0",
	KeyKp1:        "KP1",
	KeyKp2:        "KP2",
	KeyKp3:        "KP3",
	KeyKp4:        "KP4",
	KeyKp5:        "KP5",
	KeyKp6:        "KP6",
	KeyKp7:        "KP7",
	KeyKp8:        "KP8",
	KeyKp9:        "KP9",
	KeyKpDot:      "KPDOT",
	KeyKpEnter:    "KPENTER",
	KeyKpPlus:     "KPPLUS",
	KeyKpMinus:    "KPMINUS",
	KeyKpAsterisk: "KPASTERISK",
	KeyKpSlash:    "KPSLASH",

	// Misc
	KeySysRq:   "SYSRQ",
	KeyPause:   "PAUSE",
	KeyCompose: "COMPOSE",
}

// keyNameToCode maps human-readable key names to Linux key codes.
var keyNameToCode map[string]uint16

func init() {
	keyNameToCode = make(map[string]uint16, len(keyCodeToName))
	for code, name := range keyCodeToName {
		keyNameToCode[name] = code
	}
}

// KeyCodeToName converts a Linux key code to a human-readable name.
// Returns "KEY_<code>" for unknown key codes.
func KeyCodeToName(code uint16) string {
	if name, ok := keyCodeToName[code]; ok {
		return name
	}
	return fmt.Sprintf("KEY_%d", code)
}

// KeyNameToCode converts a human-readable key name to a Linux key code.
// Returns 0 and false if the key name is not found.
func KeyNameToCode(name string) (uint16, bool) {
	code, ok := keyNameToCode[name]
	return code, ok
}

// IsModifierKey returns true if the key code is a modifier key.
func IsModifierKey(code uint16) bool {
	switch code {
	case KeyLeftShift, KeyRightShift,
		KeyLeftCtrl, KeyRightCtrl,
		KeyLeftAlt, KeyRightAlt,
		KeyLeftMeta, KeyRightMeta,
		KeyCapsLock, KeyNumLock:
		return true
	}
	return false
}

