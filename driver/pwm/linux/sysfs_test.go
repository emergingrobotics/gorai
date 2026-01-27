//go:build linux

package linux

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAvailableChips(t *testing.T) {
	chips, err := AvailableChips()
	if err != nil {
		// Not an error if /sys/class/pwm doesn't exist
		if os.IsNotExist(err) {
			t.Skip("PWM sysfs not available")
		}
		t.Fatalf("AvailableChips() error: %v", err)
	}
	t.Logf("Found %d PWM chips: %v", len(chips), chips)
}

func TestChipExists(t *testing.T) {
	// pwmchip0 is commonly available on systems with PWM support
	exists := ChipExists(0)
	t.Logf("pwmchip0 exists: %v", exists)

	// A very high number shouldn't exist
	exists = ChipExists(9999)
	if exists {
		t.Error("pwmchip9999 should not exist")
	}
}

func TestOpenChip_NotFound(t *testing.T) {
	_, err := OpenChip(9999)
	if err == nil {
		t.Error("OpenChip(9999) should return error")
	}
}

func TestOpenChip_Success(t *testing.T) {
	// Skip if no PWM chips available
	if !ChipExists(0) {
		t.Skip("pwmchip0 not available")
	}

	chip, err := OpenChip(0)
	if err != nil {
		t.Fatalf("OpenChip(0) error: %v", err)
	}
	defer chip.Close(nil)

	t.Logf("Chip name: %s", chip.Name())
	t.Logf("Channels: %d", chip.Channels())

	if chip.Channels() < 1 {
		t.Error("Expected at least 1 channel")
	}
}

// TestSysfsPathConstants verifies the sysfs path is correct
func TestSysfsPathConstants(t *testing.T) {
	expected := "/sys/class/pwm"
	if sysfsBasePath != expected {
		t.Errorf("sysfsBasePath = %q, want %q", sysfsBasePath, expected)
	}
}

// mockSysfsChip tests the chip without actual hardware
func TestMockChipLogic(t *testing.T) {
	// Test that chip validates channel numbers
	c := &sysfsChip{
		num:      0,
		path:     "/tmp/test-pwmchip0",
		npwm:     2,
		channels: make(map[int]*sysfsChannel),
	}

	// Channel -1 should fail
	_, err := c.Channel(-1)
	if err == nil {
		t.Error("Channel(-1) should return error")
	}

	// Channel 2 should fail (only 0 and 1 valid)
	_, err = c.Channel(2)
	if err == nil {
		t.Error("Channel(2) should return error for 2-channel chip")
	}
}

// TestChannelFrequencyConversion tests the frequency to period conversion
func TestChannelFrequencyConversion(t *testing.T) {
	tests := []struct {
		freqHz   float64
		periodNs uint64
	}{
		{50, 20000000},      // 50 Hz = 20ms period (servo)
		{1000, 1000000},     // 1 kHz = 1ms period
		{20000, 50000},      // 20 kHz = 50µs period (motor)
		{1000000, 1000},     // 1 MHz = 1µs period
	}

	for _, tc := range tests {
		periodNs := uint64(1e9 / tc.freqHz)
		if periodNs != tc.periodNs {
			t.Errorf("Frequency %f Hz: got period %d ns, want %d ns",
				tc.freqHz, periodNs, tc.periodNs)
		}
	}
}

// TestDutyCycleConversion tests duty cycle percentage conversion
func TestDutyCycleConversion(t *testing.T) {
	period := uint64(20000000) // 20ms = 50 Hz

	tests := []struct {
		dutyCycle float64
		dutyNs    uint64
	}{
		{0.0, 0},
		{0.5, 10000000},
		{1.0, 20000000},
		{0.075, 1500000}, // 1.5ms pulse for servo center
	}

	for _, tc := range tests {
		dutyNs := uint64(float64(period) * tc.dutyCycle)
		if dutyNs != tc.dutyNs {
			t.Errorf("Duty cycle %f: got %d ns, want %d ns",
				tc.dutyCycle, dutyNs, tc.dutyNs)
		}
	}
}

// TestSysfsPathGeneration tests path generation for sysfs files
func TestSysfsPathGeneration(t *testing.T) {
	chipPath := filepath.Join(sysfsBasePath, "pwmchip0")
	expected := "/sys/class/pwm/pwmchip0"
	if chipPath != expected {
		t.Errorf("Chip path = %q, want %q", chipPath, expected)
	}

	channelPath := filepath.Join(chipPath, "pwm0")
	expected = "/sys/class/pwm/pwmchip0/pwm0"
	if channelPath != expected {
		t.Errorf("Channel path = %q, want %q", channelPath, expected)
	}

	periodPath := filepath.Join(channelPath, "period")
	expected = "/sys/class/pwm/pwmchip0/pwm0/period"
	if periodPath != expected {
		t.Errorf("Period path = %q, want %q", periodPath, expected)
	}
}
