package hal

import (
	"os"
	"testing"
)

func TestGetPWMPinMapping_RaspberryPi5(t *testing.T) {
	// Check if we have any PWM chips available (required for this test)
	_, err := os.ReadDir("/sys/class/pwm")
	hasPWMChips := err == nil && getRPi5PWMChip() >= 0

	tests := []struct {
		gpio        int
		wantChannel int
		wantOk      bool // only when PWM chips are available
	}{
		{12, 0, true},  // PWM0
		{13, 1, true},  // PWM1
		{18, 2, true},  // PWM2
		{19, 3, true},  // PWM3
		{17, 0, false}, // Not a PWM pin
		{0, 0, false},  // Not a PWM pin
	}

	for _, tc := range tests {
		chip, channel, ok := GetPWMPinMapping(BoardRaspberryPi5, tc.gpio)

		// If no PWM chips available, all calls return ok=false
		if !hasPWMChips {
			if tc.wantOk && ok {
				t.Errorf("GetPWMPinMapping(RPi5, GPIO%d) ok = true, want false (no PWM chips available)",
					tc.gpio)
			}
			continue
		}

		// With PWM chips available, check the mapping
		if ok != tc.wantOk {
			t.Errorf("GetPWMPinMapping(RPi5, GPIO%d) ok = %v, want %v",
				tc.gpio, ok, tc.wantOk)
			continue
		}
		if ok {
			if channel != tc.wantChannel {
				t.Errorf("GetPWMPinMapping(RPi5, GPIO%d) channel = %d, want %d",
					tc.gpio, channel, tc.wantChannel)
			}
			// Chip number is auto-detected, just verify it's valid
			if chip < 0 {
				t.Errorf("GetPWMPinMapping(RPi5, GPIO%d) chip = %d, want >= 0",
					tc.gpio, chip)
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
