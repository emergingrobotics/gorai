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

	// BoardRaspberryPi4 is Raspberry Pi 4.
	BoardRaspberryPi4 Board = "rpi4"

	// BoardRaspberryPi3 is Raspberry Pi 3.
	BoardRaspberryPi3 Board = "rpi3"

	// BoardOrangePi5B is Orange Pi 5B.
	BoardOrangePi5B Board = "opi5b"

	// BoardOrangePi5 is Orange Pi 5.
	BoardOrangePi5 Board = "opi5"

	// BoardGenericLinux is a generic Linux system.
	BoardGenericLinux Board = "linux"
)

// String returns the board identifier string.
func (b Board) String() string {
	return string(b)
}

// IsRaspberryPi returns true for any Raspberry Pi board.
func (b Board) IsRaspberryPi() bool {
	switch b {
	case BoardRaspberryPi5, BoardRaspberryPi4, BoardRaspberryPi3:
		return true
	}
	return false
}

// IsOrangePi returns true for any Orange Pi board.
func (b Board) IsOrangePi() bool {
	switch b {
	case BoardOrangePi5B, BoardOrangePi5:
		return true
	}
	return false
}

// SupportedBoards returns all boards with built-in support.
func SupportedBoards() []Board {
	return []Board{
		BoardRaspberryPi5,
		BoardRaspberryPi4,
		BoardRaspberryPi3,
		BoardOrangePi5B,
		BoardOrangePi5,
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

		switch {
		case strings.Contains(modelStr, "raspberry pi 5"):
			return BoardRaspberryPi5
		case strings.Contains(modelStr, "raspberry pi 4"):
			return BoardRaspberryPi4
		case strings.Contains(modelStr, "raspberry pi 3"):
			return BoardRaspberryPi3
		case strings.Contains(modelStr, "orange pi 5b"):
			return BoardOrangePi5B
		case strings.Contains(modelStr, "orange pi 5"):
			return BoardOrangePi5
		}
	}

	// Try compatible strings
	if compat, err := os.ReadFile("/proc/device-tree/compatible"); err == nil {
		compatStr := strings.ToLower(string(compat))

		switch {
		case strings.Contains(compatStr, "brcm,bcm2712"):
			return BoardRaspberryPi5
		case strings.Contains(compatStr, "brcm,bcm2711"):
			return BoardRaspberryPi4
		case strings.Contains(compatStr, "brcm,bcm2837"):
			return BoardRaspberryPi3
		case strings.Contains(compatStr, "rockchip,rk3588"):
			// Could be OPi5 or 5B, check more specifically
			if model, _ := os.ReadFile("/proc/device-tree/model"); model != nil {
				if strings.Contains(strings.ToLower(string(model)), "5b") {
					return BoardOrangePi5B
				}
			}
			return BoardOrangePi5
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
func DefaultGPIOChip(board Board) int {
	switch board {
	case BoardRaspberryPi5:
		return 4 // RP1 chip
	case BoardRaspberryPi4, BoardRaspberryPi3:
		return 0 // BCM chip
	case BoardOrangePi5B, BoardOrangePi5:
		return 0 // RK3588 main GPIO
	default:
		return 0
	}
}

// DefaultI2CBus returns the default I2C bus number for a board.
func DefaultI2CBus(board Board) int {
	switch board {
	case BoardRaspberryPi5, BoardRaspberryPi4, BoardRaspberryPi3:
		return 1 // User-accessible I2C bus
	case BoardOrangePi5B, BoardOrangePi5:
		return 2 // Commonly used I2C bus
	default:
		return 1
	}
}

// DefaultPWMChip returns the default PWM chip number for a board.
func DefaultPWMChip(board Board) int {
	switch board {
	case BoardRaspberryPi5:
		return 2
	case BoardRaspberryPi4, BoardRaspberryPi3:
		return 0
	case BoardOrangePi5B, BoardOrangePi5:
		return 0
	default:
		return 0
	}
}
