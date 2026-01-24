//go:build linux

package hal

import (
	"fmt"

	"github.com/gorai/gorai/driver/gpio"
	gpiolinux "github.com/gorai/gorai/driver/gpio/linux"
	"github.com/gorai/gorai/driver/i2c"
	i2clinux "github.com/gorai/gorai/driver/i2c/linux"
	"github.com/gorai/gorai/driver/pwm"
	"github.com/gorai/gorai/driver/serial"
	"github.com/gorai/gorai/driver/spi"
)

// createGPIODriver creates a GPIO driver for the given chip.
func createGPIODriver(chip int) (gpio.Driver, error) {
	chipPath := fmt.Sprintf("/dev/gpiochip%d", chip)
	return gpiolinux.NewDriver(chipPath)
}

// createI2CBus creates an I2C bus.
func createI2CBus(bus int) (i2c.Bus, error) {
	devicePath := fmt.Sprintf("/dev/i2c-%d", bus)
	return i2clinux.Open(devicePath)
}

// createSPIBus creates an SPI bus.
func createSPIBus(bus int) (spi.Bus, error) {
	// TODO: Implement SPI bus creation
	return nil, fmt.Errorf("%w: SPI not yet implemented", ErrFeatureNotSupported)
}

// createPWMChip creates a hardware PWM chip.
func createPWMChip(chip int) (pwm.Chip, error) {
	// TODO: Implement hardware PWM chip creation
	return nil, fmt.Errorf("%w: hardware PWM not yet implemented", ErrFeatureNotSupported)
}

// createSoftwarePWM creates a software PWM channel.
func createSoftwarePWM(pin gpio.Pin) (pwm.Channel, error) {
	// TODO: Implement software PWM creation
	return nil, fmt.Errorf("%w: software PWM not yet implemented", ErrFeatureNotSupported)
}

// createSerialPort opens a serial port.
func createSerialPort(path string, config serial.Config) (serial.Port, error) {
	// TODO: Implement serial port creation
	return nil, fmt.Errorf("%w: serial not yet implemented", ErrFeatureNotSupported)
}
