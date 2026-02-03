package hal

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// PWMPinMapping maps a GPIO pin to a PWM chip and channel.
type PWMPinMapping struct {
	Chip    int // PWM chip number (e.g., 0 for pwmchip0)
	Channel int // Channel number within the chip
}

// Raspberry Pi 5 GPIO to PWM channel mapping.
// The chip number is detected at runtime since it varies:
// - RP1 native: pwmchip2
// - pwm-2chan overlay: pwmchip0
//
// Channel assignments for pwm-2chan overlay:
// - GPIO12 → channel 0
// - GPIO13 → channel 1
var rpi5PWMChannels = map[int]int{
	12: 0, // PWM0
	13: 1, // PWM1
	18: 2, // PWM2 (4-chan overlay only)
	19: 3, // PWM3 (4-chan overlay only)
}

// Cached PWM chip number for RPi5
var (
	rpi5PWMChip     int = -1
	rpi5PWMChipOnce sync.Once
)

// detectRPi5PWMChip finds the correct PWM chip number for Raspberry Pi 5.
// Returns the chip number or -1 if no PWM chip is found.
func detectRPi5PWMChip() int {
	// Check chips in order of preference:
	// - pwmchip2: RP1 native PWM (some kernels/configs)
	// - pwmchip0: pwm-2chan overlay (most common with dtoverlay=pwm-2chan)
	for _, chip := range []int{2, 0, 1} {
		chipPath := filepath.Join("/sys/class/pwm", formatPWMChip(chip))
		if _, err := os.Stat(chipPath); err == nil {
			return chip
		}
	}
	return -1
}

// formatPWMChip returns the pwmchip directory name for a chip number.
func formatPWMChip(chip int) string {
	return fmt.Sprintf("pwmchip%d", chip)
}

// getRPi5PWMChip returns the detected PWM chip for RPi5, caching the result.
func getRPi5PWMChip() int {
	rpi5PWMChipOnce.Do(func() {
		rpi5PWMChip = detectRPi5PWMChip()
	})
	return rpi5PWMChip
}

// GetPWMPinMapping returns the hardware PWM chip and channel for a GPIO pin.
// Returns ok=false if the pin doesn't support hardware PWM on the given board.
func GetPWMPinMapping(board Board, gpio int) (chip, channel int, ok bool) {
	switch board {
	case BoardRaspberryPi5:
		ch, found := rpi5PWMChannels[gpio]
		if !found {
			return 0, 0, false
		}

		// Detect the actual PWM chip number
		detectedChip := getRPi5PWMChip()
		if detectedChip < 0 {
			return 0, 0, false
		}

		return detectedChip, ch, true

	default:
		// Generic Linux: no predefined PWM mappings
		return 0, 0, false
	}
}

// GetHardwarePWMPins returns a list of GPIO pins that support hardware PWM
// on the given board.
func GetHardwarePWMPins(board Board) []int {
	switch board {
	case BoardRaspberryPi5:
		// Return pins that the pwm-2chan overlay supports (12, 13)
		// Full RP1 support would include 18, 19 as well
		return []int{12, 13, 18, 19}
	default:
		return nil
	}
}
