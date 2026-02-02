package hal

import (
	"fmt"
	"os"
	"strings"
)

// Board identifies a hardware platform.
type Board string

const (
	// BoardUnknown indicates an unknown board.
	BoardUnknown Board = "unknown"

	// BoardAuto indicates auto-detection should be used.
	BoardAuto Board = "auto"

	// BoardRaspberryPi5 is Raspberry Pi 5.
	BoardRaspberryPi5 Board = "rpi5"

	// BoardGenericLinux is a generic Linux system.
	BoardGenericLinux Board = "linux"
)

// String returns the board identifier string.
func (b Board) String() string {
	return string(b)
}

// IsRaspberryPi returns true for any Raspberry Pi board.
func (b Board) IsRaspberryPi() bool {
	return b == BoardRaspberryPi5
}

// SupportedBoards returns all boards with built-in support.
func SupportedBoards() []Board {
	return []Board{
		BoardRaspberryPi5,
		BoardGenericLinux,
	}
}

// IsSupported returns true if the board has built-in support.
func IsSupported(board Board) bool {
	for _, b := range SupportedBoards() {
		if b == board {
			return true
		}
	}
	return false
}

// Detect attempts to identify the current board.
// Returns BoardGenericLinux if detection fails.
func Detect() Board {
	// Try device tree model first
	if model, err := os.ReadFile("/proc/device-tree/model"); err == nil {
		modelStr := strings.ToLower(strings.TrimRight(string(model), "\x00\n"))

		if strings.Contains(modelStr, "raspberry pi 5") {
			return BoardRaspberryPi5
		}
	}

	// Try compatible strings
	if compat, err := os.ReadFile("/proc/device-tree/compatible"); err == nil {
		compatStr := strings.ToLower(string(compat))

		if strings.Contains(compatStr, "brcm,bcm2712") {
			return BoardRaspberryPi5
		}
	}

	return BoardGenericLinux
}

// BoardInfo contains detailed information about the detected board.
type BoardInfo struct {
	Board     Board
	Model     string
	Revision  string
	Serial    string
	GPIOChips []string
	I2CBuses  []int
	SPIBuses  []int
	PWMChips  []int
}

// DetectInfo returns detailed board information.
func DetectInfo() BoardInfo {
	info := BoardInfo{
		Board: Detect(),
	}

	// Read model
	if model, err := os.ReadFile("/proc/device-tree/model"); err == nil {
		info.Model = strings.TrimRight(string(model), "\x00\n")
	}

	// Read serial
	if serial, err := os.ReadFile("/proc/device-tree/serial-number"); err == nil {
		info.Serial = strings.TrimRight(string(serial), "\x00\n")
	}

	// Enumerate GPIO chips
	if entries, err := os.ReadDir("/dev"); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "gpiochip") {
				info.GPIOChips = append(info.GPIOChips, "/dev/"+name)
			}
		}
	}

	// Enumerate I2C buses
	if entries, err := os.ReadDir("/dev"); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "i2c-") {
				var bus int
				if _, err := fmt.Sscanf(name, "i2c-%d", &bus); err == nil {
					info.I2CBuses = append(info.I2CBuses, bus)
				}
			}
		}
	}

	// Enumerate SPI buses
	if entries, err := os.ReadDir("/dev"); err == nil {
		seen := make(map[int]bool)
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "spidev") {
				var bus, cs int
				if _, err := fmt.Sscanf(name, "spidev%d.%d", &bus, &cs); err == nil {
					if !seen[bus] {
						info.SPIBuses = append(info.SPIBuses, bus)
						seen[bus] = true
					}
				}
			}
		}
	}

	// Enumerate PWM chips
	if entries, err := os.ReadDir("/sys/class/pwm"); err == nil {
		for _, e := range entries {
			name := e.Name()
			if strings.HasPrefix(name, "pwmchip") {
				var chip int
				if _, err := fmt.Sscanf(name, "pwmchip%d", &chip); err == nil {
					info.PWMChips = append(info.PWMChips, chip)
				}
			}
		}
	}

	return info
}

// DefaultGPIOChip returns the default GPIO chip number for a board.
// Deprecated: Use DetectGPIOChip for auto-detection with validation.
func DefaultGPIOChip(board Board) int {
	switch board {
	case BoardRaspberryPi5:
		return 4 // RP1 chip (may not exist on all configurations)
	default:
		return 0
	}
}

// DetectGPIOChip auto-detects the correct GPIO chip number for a board.
// It validates that the chip exists and is a valid character device.
func DetectGPIOChip(board Board) int {
	switch board {
	case BoardRaspberryPi5:
		// RPi5 RP1 chip is typically at gpiochip4, but varies by kernel/config
		// Try the expected locations in order of preference
		candidates := []int{4, 0}
		for _, chip := range candidates {
			if isValidGPIOChip(chip) {
				return chip
			}
		}
		// Fall back to 0 if nothing valid found
		return 0
	default:
		// For generic Linux, try gpiochip0
		if isValidGPIOChip(0) {
			return 0
		}
		return 0
	}
}

// isValidGPIOChip checks if the given GPIO chip number corresponds to
// a valid character device.
func isValidGPIOChip(chip int) bool {
	path := fmt.Sprintf("/dev/gpiochip%d", chip)
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	// Check if it's a character device (mode includes ModeCharDevice)
	return info.Mode()&os.ModeCharDevice != 0
}

// DefaultI2CBus returns the default I2C bus number for a board.
func DefaultI2CBus(board Board) int {
	switch board {
	case BoardRaspberryPi5:
		return 1 // User-accessible I2C bus
	default:
		return 1
	}
}

// DefaultPWMChip returns the default PWM chip number for a board.
func DefaultPWMChip(board Board) int {
	switch board {
	case BoardRaspberryPi5:
		return 2
	default:
		return 0
	}
}
