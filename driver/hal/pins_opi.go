package hal

import (
	"fmt"
	"strconv"
	"strings"
)

// RK3588 GPIO calculation helper.
// GPIO number = bank*32 + group*8 + line
// Groups: A=0, B=1, C=2, D=3
// Example: GPIO4_B3 = 4*32 + 1*8 + 3 = 139
func rk3588Pin(bank, group, line int) int {
	return bank*32 + group*8 + line
}

// Orange Pi 5/5B physical pin to RK3588 GPIO mapping.
// Note: Orange Pi 5B has a 26-pin header, not 40-pin like RPi.
var opi5PhysicalPins = map[int]int{
	3:  rk3588Pin(4, 1, 3), // GPIO4_B3 / I2C2_SDA_M0 = 139
	5:  rk3588Pin(4, 1, 4), // GPIO4_B4 / I2C2_SCL_M0 = 140
	7:  rk3588Pin(1, 0, 4), // GPIO1_A4 = 36
	8:  rk3588Pin(0, 1, 5), // GPIO0_B5 / UART2_TX_M0 = 13
	10: rk3588Pin(0, 1, 6), // GPIO0_B6 / UART2_RX_M0 = 14
	11: rk3588Pin(1, 0, 3), // GPIO1_A3 = 35
	12: rk3588Pin(1, 1, 2), // GPIO1_B2 / PWM14_M0 = 42
	13: rk3588Pin(4, 2, 6), // GPIO4_C6 = 150
	15: rk3588Pin(1, 3, 7), // GPIO1_D7 = 63
	16: rk3588Pin(4, 1, 2), // GPIO4_B2 = 138
	18: rk3588Pin(1, 1, 3), // GPIO1_B3 = 43
	19: rk3588Pin(4, 1, 2), // GPIO4_B2 / SPI0_MOSI = 138
	21: rk3588Pin(1, 1, 1), // GPIO1_B1 / SPI0_MISO = 41
	22: rk3588Pin(1, 1, 3), // GPIO1_B3 = 43
	23: rk3588Pin(1, 1, 4), // GPIO1_B4 / SPI0_CLK = 44
	24: rk3588Pin(1, 1, 2), // GPIO1_B2 / SPI0_CS0 = 42
	26: rk3588Pin(4, 2, 5), // GPIO4_C5 = 149
}

// Orange Pi 5/5B named pins.
var opiNamedPins = map[string]int{
	// I2C2
	"SDA":      rk3588Pin(4, 1, 3), // GPIO4_B3 = 139
	"SCL":      rk3588Pin(4, 1, 4), // GPIO4_B4 = 140
	"SDA2":     rk3588Pin(4, 1, 3),
	"SCL2":     rk3588Pin(4, 1, 4),
	"I2C2_SDA": rk3588Pin(4, 1, 3),
	"I2C2_SCL": rk3588Pin(4, 1, 4),

	// UART2
	"TX":        rk3588Pin(0, 1, 5), // GPIO0_B5 = 13
	"RX":        rk3588Pin(0, 1, 6), // GPIO0_B6 = 14
	"UART2_TX":  rk3588Pin(0, 1, 5),
	"UART2_RX":  rk3588Pin(0, 1, 6),
	"UART2_TXD": rk3588Pin(0, 1, 5),
	"UART2_RXD": rk3588Pin(0, 1, 6),

	// SPI0
	"MOSI":      rk3588Pin(4, 1, 2), // GPIO4_B2 = 138
	"MISO":      rk3588Pin(1, 1, 1), // GPIO1_B1 = 41
	"SCLK":      rk3588Pin(1, 1, 4), // GPIO1_B4 = 44
	"CS0":       rk3588Pin(1, 1, 2), // GPIO1_B2 = 42
	"SPI0_MOSI": rk3588Pin(4, 1, 2),
	"SPI0_MISO": rk3588Pin(1, 1, 1),
	"SPI0_CLK":  rk3588Pin(1, 1, 4),
	"SPI0_CS0":  rk3588Pin(1, 1, 2),

	// PWM
	"PWM14": rk3588Pin(1, 1, 2), // GPIO1_B2 = 42
}

// opi5bPinMapper handles pin mapping for Orange Pi 5B.
type opi5bPinMapper struct{}

func (m *opi5bPinMapper) Resolve(ref PinRef) (int, error) {
	if ref.Physical > 0 {
		return m.PhysicalToGPIO(ref.Physical)
	}
	if ref.Name != "" {
		return m.NameToGPIO(ref.Name)
	}
	return ref.GPIO, nil
}

func (m *opi5bPinMapper) PhysicalToGPIO(physical int) (int, error) {
	if gpio, ok := opi5PhysicalPins[physical]; ok {
		return gpio, nil
	}
	return 0, fmt.Errorf("%w: invalid physical pin %d for Orange Pi 5B", ErrPinNotFound, physical)
}

func (m *opi5bPinMapper) NameToGPIO(name string) (int, error) {
	upper := strings.ToUpper(name)

	// Check named pins first
	if gpio, ok := opiNamedPins[upper]; ok {
		return gpio, nil
	}

	// Try parsing as GPIO number "GPIO139"
	if strings.HasPrefix(upper, "GPIO") {
		n, err := strconv.Atoi(name[4:])
		if err != nil {
			return 0, fmt.Errorf("%w: invalid GPIO name %q", ErrPinNotFound, name)
		}
		return n, nil
	}

	// Try parsing RK3588 format "GPIO4_B3"
	if strings.HasPrefix(upper, "GPIO") && strings.Contains(upper, "_") {
		parts := strings.Split(upper[4:], "_")
		if len(parts) == 2 && len(parts[1]) >= 2 {
			bank, err := strconv.Atoi(parts[0])
			if err != nil {
				return 0, fmt.Errorf("%w: invalid GPIO bank in %q", ErrPinNotFound, name)
			}

			groupChar := parts[1][0]
			var group int
			switch groupChar {
			case 'A':
				group = 0
			case 'B':
				group = 1
			case 'C':
				group = 2
			case 'D':
				group = 3
			default:
				return 0, fmt.Errorf("%w: invalid GPIO group in %q", ErrPinNotFound, name)
			}

			line, err := strconv.Atoi(parts[1][1:])
			if err != nil {
				return 0, fmt.Errorf("%w: invalid GPIO line in %q", ErrPinNotFound, name)
			}

			return rk3588Pin(bank, group, line), nil
		}
	}

	return 0, fmt.Errorf("%w: unknown pin name %q for Orange Pi 5B", ErrPinNotFound, name)
}

// opi5PinMapper handles pin mapping for Orange Pi 5.
// Uses the same mappings as Orange Pi 5B.
type opi5PinMapper struct {
	opi5bPinMapper
}

func (m *opi5PinMapper) PhysicalToGPIO(physical int) (int, error) {
	if gpio, ok := opi5PhysicalPins[physical]; ok {
		return gpio, nil
	}
	return 0, fmt.Errorf("%w: invalid physical pin %d for Orange Pi 5", ErrPinNotFound, physical)
}

func (m *opi5PinMapper) NameToGPIO(name string) (int, error) {
	gpio, err := m.opi5bPinMapper.NameToGPIO(name)
	if err != nil {
		return 0, fmt.Errorf("%w: unknown pin name %q for Orange Pi 5", ErrPinNotFound, name)
	}
	return gpio, nil
}
