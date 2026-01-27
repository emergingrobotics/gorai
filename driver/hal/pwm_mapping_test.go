package hal

import (
	"testing"
)

func TestGetPWMPinMapping_RaspberryPi5(t *testing.T) {
	tests := []struct {
		gpio        int
		wantChip    int
		wantChannel int
		wantOk      bool
	}{
		{12, 2, 0, true},  // PWM0
		{13, 2, 1, true},  // PWM1
		{18, 2, 2, true},  // PWM2
		{19, 2, 3, true},  // PWM3
		{17, 0, 0, false}, // Not a PWM pin
		{0, 0, 0, false},  // Not a PWM pin
	}

	for _, tc := range tests {
		chip, channel, ok := GetPWMPinMapping(BoardRaspberryPi5, tc.gpio)
		if ok != tc.wantOk {
			t.Errorf("GetPWMPinMapping(RPi5, GPIO%d) ok = %v, want %v",
				tc.gpio, ok, tc.wantOk)
			continue
		}
		if ok {
			if chip != tc.wantChip || channel != tc.wantChannel {
				t.Errorf("GetPWMPinMapping(RPi5, GPIO%d) = (%d, %d), want (%d, %d)",
					tc.gpio, chip, channel, tc.wantChip, tc.wantChannel)
			}
		}
	}
}

func TestGetPWMPinMapping_RaspberryPi4(t *testing.T) {
	tests := []struct {
		gpio        int
		wantChip    int
		wantChannel int
		wantOk      bool
	}{
		{12, 0, 0, true},  // PWM0 on ALT0
		{13, 0, 1, true},  // PWM1 on ALT0
		{18, 0, 0, true},  // PWM0 on ALT5
		{19, 0, 1, true},  // PWM1 on ALT5
		{17, 0, 0, false}, // Not a PWM pin
	}

	for _, tc := range tests {
		chip, channel, ok := GetPWMPinMapping(BoardRaspberryPi4, tc.gpio)
		if ok != tc.wantOk {
			t.Errorf("GetPWMPinMapping(RPi4, GPIO%d) ok = %v, want %v",
				tc.gpio, ok, tc.wantOk)
			continue
		}
		if ok {
			if chip != tc.wantChip || channel != tc.wantChannel {
				t.Errorf("GetPWMPinMapping(RPi4, GPIO%d) = (%d, %d), want (%d, %d)",
					tc.gpio, chip, channel, tc.wantChip, tc.wantChannel)
			}
		}
	}
}

func TestGetPWMPinMapping_RaspberryPi3(t *testing.T) {
	// RPi3 uses the same mapping as RPi4
	chip, channel, ok := GetPWMPinMapping(BoardRaspberryPi3, 18)
	if !ok || chip != 0 || channel != 0 {
		t.Errorf("GetPWMPinMapping(RPi3, GPIO18) = (%d, %d, %v), want (0, 0, true)",
			chip, channel, ok)
	}
}

func TestGetPWMPinMapping_OrangePi5B(t *testing.T) {
	tests := []struct {
		gpio        int
		wantChip    int
		wantChannel int
		wantOk      bool
	}{
		{42, 3, 2, true},   // PWM14 (physical pin 12)
		{150, 3, 1, true},  // PWM13 (physical pin 13)
		{47, 0, 0, false},  // PWM15 not on standard header
		{100, 0, 0, false}, // Not a PWM pin
	}

	for _, tc := range tests {
		chip, channel, ok := GetPWMPinMapping(BoardOrangePi5B, tc.gpio)
		if ok != tc.wantOk {
			t.Errorf("GetPWMPinMapping(OPi5B, GPIO%d) ok = %v, want %v",
				tc.gpio, ok, tc.wantOk)
			continue
		}
		if ok {
			if chip != tc.wantChip || channel != tc.wantChannel {
				t.Errorf("GetPWMPinMapping(OPi5B, GPIO%d) = (%d, %d), want (%d, %d)",
					tc.gpio, chip, channel, tc.wantChip, tc.wantChannel)
			}
		}
	}
}

func TestGetPWMPinMapping_OrangePi5Plus(t *testing.T) {
	tests := []struct {
		gpio        int
		wantChip    int
		wantChannel int
		wantOk      bool
	}{
		{44, 0, 0, true},   // PWM0 (physical pin 33)
		{45, 0, 1, true},   // PWM1 (physical pin 32)
		{42, 3, 2, true},   // PWM14 (physical pin 12)
		{150, 3, 1, true},  // PWM13 (physical pin 13)
		{100, 0, 0, false}, // Not a PWM pin
	}

	for _, tc := range tests {
		chip, channel, ok := GetPWMPinMapping(BoardOrangePi5Plus, tc.gpio)
		if ok != tc.wantOk {
			t.Errorf("GetPWMPinMapping(OPi5Plus, GPIO%d) ok = %v, want %v",
				tc.gpio, ok, tc.wantOk)
			continue
		}
		if ok {
			if chip != tc.wantChip || channel != tc.wantChannel {
				t.Errorf("GetPWMPinMapping(OPi5Plus, GPIO%d) = (%d, %d), want (%d, %d)",
					tc.gpio, chip, channel, tc.wantChip, tc.wantChannel)
			}
		}
	}
}

func TestGetPWMPinMapping_GenericLinux(t *testing.T) {
	// Generic Linux has no predefined mappings
	_, _, ok := GetPWMPinMapping(BoardGenericLinux, 18)
	if ok {
		t.Error("GetPWMPinMapping(GenericLinux, GPIO18) should return ok=false")
	}
}

func TestGetHardwarePWMPins(t *testing.T) {
	tests := []struct {
		board    Board
		wantPins int
	}{
		{BoardRaspberryPi5, 4},
		{BoardRaspberryPi4, 4},
		{BoardRaspberryPi3, 4},
		{BoardOrangePi5Plus, 4}, // 4 PWM pins on 40-pin header
		{BoardOrangePi5B, 2},    // 2 PWM pins on 26-pin header
		{BoardOrangePi5, 2},
		{BoardGenericLinux, 0},
	}

	for _, tc := range tests {
		pins := GetHardwarePWMPins(tc.board)
		if len(pins) != tc.wantPins {
			t.Errorf("GetHardwarePWMPins(%s) returned %d pins, want %d",
				tc.board, len(pins), tc.wantPins)
		}
	}
}
