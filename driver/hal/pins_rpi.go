package hal

import (
	"fmt"
	"strconv"
	"strings"
)

// Raspberry Pi physical pin to BCM GPIO mapping.
// This mapping is the same for Pi 3, 4, and 5 (40-pin header).
var rpiPhysicalPins = map[int]int{
	3:  2,  // GPIO2 / SDA1
	5:  3,  // GPIO3 / SCL1
	7:  4,  // GPIO4 / GPCLK0
	8:  14, // GPIO14 / TXD0
	10: 15, // GPIO15 / RXD0
	11: 17, // GPIO17
	12: 18, // GPIO18 / PCM_CLK / PWM0
	13: 27, // GPIO27
	15: 22, // GPIO22
	16: 23, // GPIO23
	18: 24, // GPIO24
	19: 10, // GPIO10 / SPI0_MOSI
	21: 9,  // GPIO9 / SPI0_MISO
	22: 25, // GPIO25
	23: 11, // GPIO11 / SPI0_SCLK
	24: 8,  // GPIO8 / SPI0_CE0
	26: 7,  // GPIO7 / SPI0_CE1
	27: 0,  // GPIO0 / ID_SD (EEPROM)
	28: 1,  // GPIO1 / ID_SC (EEPROM)
	29: 5,  // GPIO5
	31: 6,  // GPIO6
	32: 12, // GPIO12 / PWM0
	33: 13, // GPIO13 / PWM1
	35: 19, // GPIO19 / PCM_FS / SPI1_MISO
	36: 16, // GPIO16
	37: 26, // GPIO26
	38: 20, // GPIO20 / PCM_DIN / SPI1_MOSI
	40: 21, // GPIO21 / PCM_DOUT / SPI1_SCLK
}

// Raspberry Pi named pins (common aliases).
var rpiNamedPins = map[string]int{
	// I2C
	"SDA":  2,
	"SCL":  3,
	"SDA1": 2,
	"SCL1": 3,

	// UART
	"TX":   14,
	"RX":   15,
	"TXD":  14,
	"RXD":  15,
	"TXD0": 14,
	"RXD0": 15,

	// SPI0
	"MOSI":      10,
	"MISO":      9,
	"SCLK":      11,
	"CE0":       8,
	"CE1":       7,
	"SPI0_MOSI": 10,
	"SPI0_MISO": 9,
	"SPI0_SCLK": 11,
	"SPI0_CE0":  8,
	"SPI0_CE1":  7,

	// SPI1
	"SPI1_MOSI": 20,
	"SPI1_MISO": 19,
	"SPI1_SCLK": 21,

	// PWM
	"PWM0": 18,
	"PWM1": 13,

	// PCM/I2S
	"PCM_CLK":  18,
	"PCM_FS":   19,
	"PCM_DIN":  20,
	"PCM_DOUT": 21,

	// EEPROM
	"ID_SD": 0,
	"ID_SC": 1,

	// Clock
	"GPCLK0": 4,
}

// rpi5PinMapper handles pin mapping for Raspberry Pi 5.
type rpi5PinMapper struct{}

func (m *rpi5PinMapper) Resolve(ref PinRef) (int, error) {
	if ref.Physical > 0 {
		return m.PhysicalToGPIO(ref.Physical)
	}
	if ref.Name != "" {
		return m.NameToGPIO(ref.Name)
	}
	return ref.GPIO, nil
}

func (m *rpi5PinMapper) PhysicalToGPIO(physical int) (int, error) {
	if gpio, ok := rpiPhysicalPins[physical]; ok {
		return gpio, nil
	}
	return 0, fmt.Errorf("%w: invalid physical pin %d for Raspberry Pi 5", ErrPinNotFound, physical)
}

func (m *rpi5PinMapper) NameToGPIO(name string) (int, error) {
	upper := strings.ToUpper(name)

	// Check named pins first
	if gpio, ok := rpiNamedPins[upper]; ok {
		return gpio, nil
	}

	// Try parsing as GPIO number "GPIO17"
	if strings.HasPrefix(upper, "GPIO") {
		n, err := strconv.Atoi(name[4:])
		if err != nil {
			return 0, fmt.Errorf("%w: invalid GPIO name %q", ErrPinNotFound, name)
		}
		// Validate BCM GPIO range (0-27 for user GPIOs)
		if n < 0 || n > 27 {
			return 0, fmt.Errorf("%w: GPIO%d out of range (0-27)", ErrPinNotFound, n)
		}
		return n, nil
	}

	return 0, fmt.Errorf("%w: unknown pin name %q for Raspberry Pi 5", ErrPinNotFound, name)
}

