// Package hal provides the Hardware Abstraction Layer for Gorai.
//
// The HAL enables board-agnostic component development by providing
// consistent interfaces to GPIO, I2C, SPI, PWM, and serial peripherals.
// Components obtain HAL via dependency injection and use it to access
// hardware without knowing the underlying board or driver.
package hal

import (
	"context"
	"fmt"
	"sync"

	"github.com/gorai/gorai/driver/gpio"
	"github.com/gorai/gorai/driver/i2c"
	"github.com/gorai/gorai/driver/pwm"
	"github.com/gorai/gorai/driver/serial"
	"github.com/gorai/gorai/driver/spi"
)

// HAL provides access to hardware peripherals.
// Components obtain HAL from dependencies and use it for all hardware access.
type HAL interface {
	// Name returns the HAL identifier.
	Name() string

	// Board returns the detected or configured board identifier.
	Board() Board

	// GPIO returns the GPIO driver for this board.
	// The driver is lazily initialized on first call.
	GPIO() (gpio.Driver, error)

	// I2C returns an I2C bus by number (e.g., 1 for /dev/i2c-1).
	// Returns a cached bus if already opened.
	I2C(bus int) (i2c.Bus, error)

	// SPI returns an SPI bus by number (e.g., 0 for /dev/spidev0.*).
	SPI(bus int) (spi.Bus, error)

	// PWM returns a PWM controller.
	// For hardware PWM, chip is the pwmchip number.
	PWM(chip int) (pwm.Chip, error)

	// SoftwarePWM creates a software PWM channel on a GPIO pin.
	// This is useful when hardware PWM is not available.
	SoftwarePWM(pin int) (pwm.Channel, error)

	// Serial opens a serial port with the given configuration.
	Serial(path string, config serial.Config) (serial.Port, error)

	// ResolvePin resolves a board-agnostic pin reference to a GPIO number.
	// Supports formats: "GPIO17", "PIN11", 17, "SDA1"
	ResolvePin(ref PinRef) (int, error)

	// ResolvePinFromAny parses and resolves a pin reference from RDL config.
	// Accepts int, float64, or string values.
	ResolvePinFromAny(v any) (int, error)

	// GetPWMMapping returns the hardware PWM chip and channel for a GPIO pin.
	// Returns ok=false if the pin doesn't support hardware PWM.
	// This allows components to check if hardware PWM is available before
	// falling back to software PWM.
	GetPWMMapping(gpio int) (chip, channel int, ok bool)

	// Close releases all hardware resources.
	Close(ctx context.Context) error
}

// Config holds HAL configuration from RDL platform section.
type Config struct {
	// Board specifies the target board. Empty or "auto" for detection.
	Board string `json:"board"`

	// GPIO configuration
	GPIO GPIOConfig `json:"gpio"`

	// I2C configuration
	I2C I2CConfig `json:"i2c"`

	// SPI configuration
	SPI SPIConfig `json:"spi"`

	// PWM configuration
	PWM PWMConfig `json:"pwm"`
}

// GPIOConfig holds GPIO-specific configuration.
type GPIOConfig struct {
	// Chip is the default GPIO chip number (e.g., 4 for /dev/gpiochip4)
	Chip int `json:"chip"`
}

// I2CConfig holds I2C-specific configuration.
type I2CConfig struct {
	// Buses lists enabled I2C bus numbers
	Buses []int `json:"buses"`
}

// SPIConfig holds SPI-specific configuration.
type SPIConfig struct {
	// Buses lists enabled SPI bus numbers
	Buses []int `json:"buses"`
}

// PWMConfig holds PWM-specific configuration.
type PWMConfig struct {
	// Chips lists enabled PWM chip numbers
	Chips []int `json:"chips"`
}

// hal is the concrete HAL implementation.
type linuxHAL struct {
	board     Board
	config    Config
	pinMapper PinMapper
	mu        sync.RWMutex

	// Lazily initialized drivers
	gpioDriver gpio.Driver
	i2cBuses   map[int]i2c.Bus
	spiBuses   map[int]spi.Bus
	pwmChips   map[int]pwm.Chip
	softPWMs   map[int]pwm.Channel

	closed bool
}

// For testing: allow injecting a mock HAL
var testHAL HAL

// SetTestHAL sets a mock HAL for testing. Pass nil to clear.
func SetTestHAL(h HAL) {
	testHAL = h
}

// New creates a new HAL with the given configuration.
// If board is empty or "auto", board detection is performed.
func New(config Config) (HAL, error) {
	// Allow test injection
	if testHAL != nil {
		return testHAL, nil
	}

	board := Board(config.Board)
	if board == "" || board == BoardAuto {
		board = Detect()
	}

	if !IsSupported(board) {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedBoard, board)
	}

	h := &linuxHAL{
		board:     board,
		config:    config,
		pinMapper: GetPinMapper(board),
		i2cBuses:  make(map[int]i2c.Bus),
		spiBuses:  make(map[int]spi.Bus),
		pwmChips:  make(map[int]pwm.Chip),
		softPWMs:  make(map[int]pwm.Channel),
	}

	return h, nil
}

// Name returns the HAL identifier.
func (h *linuxHAL) Name() string {
	return fmt.Sprintf("hal:%s", h.board)
}

// Board returns the board identifier.
func (h *linuxHAL) Board() Board {
	return h.board
}

// GPIO returns the GPIO driver, initializing if needed.
func (h *linuxHAL) GPIO() (gpio.Driver, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, ErrHALClosed
	}

	if h.gpioDriver != nil {
		return h.gpioDriver, nil
	}

	// Determine GPIO chip
	chip := h.config.GPIO.Chip
	if chip == 0 {
		chip = DefaultGPIOChip(h.board)
	}

	// Create GPIO driver
	driver, err := createGPIODriver(chip)
	if err != nil {
		return nil, fmt.Errorf("failed to create GPIO driver for chip %d: %w", chip, err)
	}

	h.gpioDriver = driver
	return driver, nil
}

// I2C returns an I2C bus, opening if needed.
func (h *linuxHAL) I2C(bus int) (i2c.Bus, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, ErrHALClosed
	}

	if existing, ok := h.i2cBuses[bus]; ok {
		return existing, nil
	}

	b, err := createI2CBus(bus)
	if err != nil {
		return nil, fmt.Errorf("failed to open I2C bus %d: %w", bus, err)
	}

	h.i2cBuses[bus] = b
	return b, nil
}

// SPI returns an SPI bus, opening if needed.
func (h *linuxHAL) SPI(bus int) (spi.Bus, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, ErrHALClosed
	}

	if existing, ok := h.spiBuses[bus]; ok {
		return existing, nil
	}

	b, err := createSPIBus(bus)
	if err != nil {
		return nil, fmt.Errorf("failed to open SPI bus %d: %w", bus, err)
	}

	h.spiBuses[bus] = b
	return b, nil
}

// PWM returns a PWM chip, opening if needed.
func (h *linuxHAL) PWM(chip int) (pwm.Chip, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, ErrHALClosed
	}

	if existing, ok := h.pwmChips[chip]; ok {
		return existing, nil
	}

	c, err := createPWMChip(chip)
	if err != nil {
		return nil, fmt.Errorf("failed to open PWM chip %d: %w", chip, err)
	}

	h.pwmChips[chip] = c
	return c, nil
}

// SoftwarePWM creates a software PWM channel on a GPIO pin.
func (h *linuxHAL) SoftwarePWM(pin int) (pwm.Channel, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil, ErrHALClosed
	}

	if existing, ok := h.softPWMs[pin]; ok {
		return existing, nil
	}

	// Ensure GPIO driver is initialized
	if h.gpioDriver == nil {
		chip := h.config.GPIO.Chip
		if chip == 0 {
			chip = DefaultGPIOChip(h.board)
		}
		driver, err := createGPIODriver(chip)
		if err != nil {
			return nil, fmt.Errorf("failed to create GPIO driver: %w", err)
		}
		h.gpioDriver = driver
	}

	// Get the GPIO pin
	gpioPin, err := h.gpioDriver.Pin(pin)
	if err != nil {
		return nil, fmt.Errorf("failed to get GPIO pin %d: %w", pin, err)
	}

	// Create software PWM
	ch, err := createSoftwarePWM(gpioPin)
	if err != nil {
		return nil, fmt.Errorf("failed to create software PWM on pin %d: %w", pin, err)
	}

	h.softPWMs[pin] = ch
	return ch, nil
}

// Serial opens a serial port.
func (h *linuxHAL) Serial(path string, config serial.Config) (serial.Port, error) {
	if h.closed {
		return nil, ErrHALClosed
	}
	return createSerialPort(path, config)
}

// ResolvePin resolves a pin reference to a GPIO number.
func (h *linuxHAL) ResolvePin(ref PinRef) (int, error) {
	return h.pinMapper.Resolve(ref)
}

// ResolvePinFromAny parses and resolves a pin from RDL config value.
func (h *linuxHAL) ResolvePinFromAny(v any) (int, error) {
	ref, err := ParsePinRef(v)
	if err != nil {
		return 0, err
	}
	return h.ResolvePin(ref)
}

// GetPWMMapping returns the hardware PWM chip and channel for a GPIO pin.
// Returns ok=false if the pin doesn't support hardware PWM.
func (h *linuxHAL) GetPWMMapping(gpio int) (chip, channel int, ok bool) {
	return GetPWMPinMapping(h.board, gpio)
}

// Close releases all resources.
func (h *linuxHAL) Close(ctx context.Context) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.closed {
		return nil
	}
	h.closed = true

	var errs []error

	// Close software PWMs
	for pin, ch := range h.softPWMs {
		if closer, ok := ch.(interface{ Close(context.Context) error }); ok {
			if err := closer.Close(ctx); err != nil {
				errs = append(errs, fmt.Errorf("close soft PWM pin %d: %w", pin, err))
			}
		}
	}
	h.softPWMs = nil

	// Close PWM chips
	for chip, c := range h.pwmChips {
		if err := c.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("close PWM chip %d: %w", chip, err))
		}
	}
	h.pwmChips = nil

	// Close SPI buses
	for bus, b := range h.spiBuses {
		if err := b.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("close SPI bus %d: %w", bus, err))
		}
	}
	h.spiBuses = nil

	// Close I2C buses
	for bus, b := range h.i2cBuses {
		if err := b.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("close I2C bus %d: %w", bus, err))
		}
	}
	h.i2cBuses = nil

	// Close GPIO driver
	if h.gpioDriver != nil {
		if err := h.gpioDriver.Close(ctx); err != nil {
			errs = append(errs, fmt.Errorf("close GPIO driver: %w", err))
		}
		h.gpioDriver = nil
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing HAL: %v", errs)
	}
	return nil
}

// Verify interface compliance
var _ HAL = (*linuxHAL)(nil)
