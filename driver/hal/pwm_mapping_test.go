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
