package hal

// PWMPinMapping maps a GPIO pin to a PWM chip and channel.
type PWMPinMapping struct {
	Chip    int // PWM chip number (e.g., 0 for pwmchip0)
	Channel int // Channel number within the chip
}

// Raspberry Pi 3/4 GPIO to PWM mapping.
// These GPIOs can be configured for hardware PWM via device tree overlays.
// Note: GPIO 12/18 share PWM channel 0, GPIO 13/19 share PWM channel 1.
// Only one GPIO per channel can be active at a time.
var rpiPWMPins = map[int]PWMPinMapping{
	12: {Chip: 0, Channel: 0}, // PWM0 on ALT0
	13: {Chip: 0, Channel: 1}, // PWM1 on ALT0
	18: {Chip: 0, Channel: 0}, // PWM0 on ALT5
	19: {Chip: 0, Channel: 1}, // PWM1 on ALT5
}

// Raspberry Pi 5 GPIO to PWM mapping.
// The RP1 chip provides 4 PWM channels via pwmchip2.
var rpi5PWMPins = map[int]PWMPinMapping{
	12: {Chip: 2, Channel: 0}, // PWM0
	13: {Chip: 2, Channel: 1}, // PWM1
	18: {Chip: 2, Channel: 2}, // PWM2
	19: {Chip: 2, Channel: 3}, // PWM3
}

// Orange Pi 5 Plus GPIO to PWM mapping.
// The RK3588 provides multiple PWM controllers.
// The Orange Pi 5 Plus has a 40-pin header with 4 PWM pins available.
var opi5PlusPWMPins = map[int]PWMPinMapping{
	44:  {Chip: 0, Channel: 0}, // PWM0 (GPIO1_B4, physical pin 33)
	45:  {Chip: 0, Channel: 1}, // PWM1 (GPIO1_B5, physical pin 32)
	42:  {Chip: 3, Channel: 2}, // PWM14 (GPIO1_B2, physical pin 12)
	150: {Chip: 3, Channel: 1}, // PWM13 (GPIO4_C6, physical pin 13)
}

// Orange Pi 5/5B GPIO to PWM mapping.
// The RK3588 provides multiple PWM controllers.
// The Orange Pi 5/5B has a 26-pin header with limited PWM pins exposed.
var opi5PWMPins = map[int]PWMPinMapping{
	42:  {Chip: 3, Channel: 2}, // PWM14 (GPIO1_B2, physical pin 12)
	150: {Chip: 3, Channel: 1}, // PWM13 (GPIO4_C6, physical pin 13)
}

// GetPWMPinMapping returns the hardware PWM chip and channel for a GPIO pin.
// Returns ok=false if the pin doesn't support hardware PWM on the given board.
func GetPWMPinMapping(board Board, gpio int) (chip, channel int, ok bool) {
	var mapping map[int]PWMPinMapping

	switch board {
	case BoardRaspberryPi5:
		mapping = rpi5PWMPins
	case BoardRaspberryPi4, BoardRaspberryPi3:
		mapping = rpiPWMPins
	case BoardOrangePi5Plus:
		mapping = opi5PlusPWMPins
	case BoardOrangePi5B, BoardOrangePi5:
		mapping = opi5PWMPins
	default:
		// Generic Linux: no predefined PWM mappings
		return 0, 0, false
	}

	if m, found := mapping[gpio]; found {
		return m.Chip, m.Channel, true
	}
	return 0, 0, false
}

// GetHardwarePWMPins returns a list of GPIO pins that support hardware PWM
// on the given board.
func GetHardwarePWMPins(board Board) []int {
	var mapping map[int]PWMPinMapping

	switch board {
	case BoardRaspberryPi5:
		mapping = rpi5PWMPins
	case BoardRaspberryPi4, BoardRaspberryPi3:
		mapping = rpiPWMPins
	case BoardOrangePi5Plus:
		mapping = opi5PlusPWMPins
	case BoardOrangePi5B, BoardOrangePi5:
		mapping = opi5PWMPins
	default:
		return nil
	}

	pins := make([]int, 0, len(mapping))
	for pin := range mapping {
		pins = append(pins, pin)
	}
	return pins
}
