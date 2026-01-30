package hal

// PWMPinMapping maps a GPIO pin to a PWM chip and channel.
type PWMPinMapping struct {
	Chip    int // PWM chip number (e.g., 0 for pwmchip0)
	Channel int // Channel number within the chip
}

// Raspberry Pi 5 GPIO to PWM mapping.
// The RP1 chip provides 4 PWM channels via pwmchip2.
var rpi5PWMPins = map[int]PWMPinMapping{
	12: {Chip: 2, Channel: 0}, // PWM0
	13: {Chip: 2, Channel: 1}, // PWM1
	18: {Chip: 2, Channel: 2}, // PWM2
	19: {Chip: 2, Channel: 3}, // PWM3
}

// GetPWMPinMapping returns the hardware PWM chip and channel for a GPIO pin.
// Returns ok=false if the pin doesn't support hardware PWM on the given board.
func GetPWMPinMapping(board Board, gpio int) (chip, channel int, ok bool) {
	var mapping map[int]PWMPinMapping

	switch board {
	case BoardRaspberryPi5:
		mapping = rpi5PWMPins
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
	default:
		return nil
	}

	pins := make([]int, 0, len(mapping))
	for pin := range mapping {
		pins = append(pins, pin)
	}
	return pins
}
