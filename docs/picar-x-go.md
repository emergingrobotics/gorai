# PiCar-X Go Port Design

Concrete Go implementation plan for porting the SunFounder PiCar-X Python stack to GoRAI. Every Python class maps to a GoRAI component with a registered constructor, conforming to existing interfaces.

Based on the reverse engineering in [picar-x-implementation-review.md](./picar-x-implementation-review.md).

---

## 1. Caddy Model: Separate Module, Not Core Repo

GoRAI uses the [Caddy model](./package-dev-approach.md) for component distribution: each hardware driver/component family is a **standalone Go module** that users import via blank imports in their robot project's `main.go`. PiCar-X components do NOT go in the `gorai/gorai` core repo.

### Why

- The core repo stays hardware-agnostic. It contains interfaces and the runtime, not hardware-specific drivers.
- PiCar-X users don't need ORCA's motor drivers. ORCA users don't need PiCar-X's Robot HAT driver. The Caddy model means each binary contains exactly what its RDL references.
- Third-party contributors (including SunFounder themselves) could maintain PiCar-X components independently.
- The HC-SR04 ultrasonic driver is reusable beyond PiCar-X. It belongs in its own module so other robots can import it without pulling in Robot HAT code.

### Repository Layout

Three new repos:

**`emergingrobotics/gorai-driver-robothat`** -- Robot HAT MCU I2C driver (reusable across all SunFounder Robot HAT products, not just PiCar-X):

```
gorai-driver-robothat/
  go.mod                   # module github.com/emergingrobotics/gorai-driver-robothat
  mcu.go                   # Core MCU communication (PWM + ADC over I2C)
  mcu_test.go
  board.go                 # Board version detection + pin mapping
  board_test.go
  calibration.go           # File-based calibration store (fileDB compat)
  calibration_test.go
```

**`emergingrobotics/gorai-driver-hcsr04`** -- HC-SR04 ultrasonic sensor (generic, used by many robot kits):

```
gorai-driver-hcsr04/
  go.mod                   # module github.com/emergingrobotics/gorai-driver-hcsr04
  hcsr04.go                # GPIO time-of-flight driver
  hcsr04_test.go
  component.go             # GoRAI sensor.RangeSensor component wrapper
  component_test.go
```

**`emergingrobotics/gorai-picarx`** -- PiCar-X component package (composes the drivers above into GoRAI components):

```
gorai-picarx/
  go.mod                   # module github.com/emergingrobotics/gorai-picarx
  servo/
    servo.go               # Robot HAT PWM servo component
    servo_test.go
  motor/
    motor.go               # Robot HAT DC motor component (GPIO dir + MCU PWM)
    motor_test.go
  base/
    base.go                # Ackermann-differential base
    base_test.go
  grayscale/
    sensor.go              # Robot HAT ADC grayscale line sensor
    sensor_test.go
  picarx.go                # Convenience: blank-imports all PiCar-X components
  examples/
    robot.json             # Complete PiCar-X RDL
    README.md
```

### User Experience

A PiCar-X user starts from the robot template:

```bash
git clone https://github.com/emergingrobotics/gorai-robot-template my-picarx
cd my-picarx
```

Their `main.go` is the component manifest:

```go
package main

import (
    "github.com/emergingrobotics/gorai/cmd/gorai"

    // PiCar-X components (imports all: servo, motor, base, grayscale)
    _ "github.com/emergingrobotics/gorai-picarx"

    // Or import individually:
    // _ "github.com/emergingrobotics/gorai-picarx/servo"
    // _ "github.com/emergingrobotics/gorai-picarx/motor"
    // _ "github.com/emergingrobotics/gorai-picarx/base"
    // _ "github.com/emergingrobotics/gorai-picarx/grayscale"

    // HC-SR04 is a separate module (not PiCar-X specific)
    _ "github.com/emergingrobotics/gorai-driver-hcsr04"
)

func main() {
    gorai.Run()
}
```

Install:

```bash
gorai component add picarx
# Equivalent to: go get github.com/emergingrobotics/gorai-picarx@latest
# Adds blank import to main.go
```

Build:

```bash
gorai build robot.json -o picarx --target linux/arm64
scp picarx pi@raspberrypi:~
```

The compiled binary contains the GoRAI core, embedded NATS, the Robot HAT MCU driver, HC-SR04 driver, and the PiCar-X components. Nothing else.

### How Dependencies Work Between Modules

The `gorai-picarx` module depends on `gorai-driver-robothat` via Go module dependency (`go.mod`). At runtime, the MCU driver registers as a component. Servos and motors declare `depends_on: ["robot-hat-mcu"]` in the RDL, and the GoRAI runtime passes the MCU instance through the `resource.Dependencies` interface -- not via direct Go struct references.

This means the servo component code doesn't `import "github.com/emergingrobotics/gorai-driver-robothat"` directly in its constructor. Instead, it receives the MCU through the dependency injection system:

```go
// Instead of:  mcu, ok := mcuDep.(*robothat.MCU)
// Use:         mcu, ok := mcuDep.(MCUDriver)

// MCUDriver is the interface the servo needs -- defined in the picarx module
type MCUDriver interface {
    SetPWMFrequency(ctx context.Context, channel int, hz float64) (float64, error)
    SetPWMPulseWidth(ctx context.Context, channel int, value uint16) error
    TimerPeriod(channel int) uint16
}
```

This follows Go best practice: accept interfaces, return structs. The servo component depends on a behavior (MCUDriver interface), not a concrete type. The Robot HAT MCU driver satisfies this interface. A future different MCU board could too.

---

## 2. Driver Layer

### 2.1 Robot HAT MCU Driver (`gorai-driver-robothat/mcu.go`)

This is the foundation. Every servo, motor, and ADC sensor depends on it. The Python `I2C`, `PWM`, and `ADC` classes collapse into a single Go struct because they all talk to the same MCU at `0x14`. This lives in its own module (`gorai-driver-robothat`) because the Robot HAT board is used by many SunFounder products beyond PiCar-X.

```go
package robothat

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/gorai/gorai/driver/gpio"
	"github.com/gorai/gorai/driver/i2c"
)

const (
	DefaultAddress = 0x14
	FallbackAddress = 0x15
	MCUClock = 72_000_000

	regADCBase  = 0x10 // ADC channel registers (0x10-0x17)
	regPWMDuty  = 0x20 // PWM pulse width base (0x20 + channel)
	regPWMFreq  = 0x30 // PWM frequency base (0x30 + channel)
	regPrescaler = 0x40 // Timer prescaler base (0x40 + timer)
	regPeriod   = 0x44 // Timer period/ARR base (0x44 + timer)

	maxChannels = 14
	maxTimers   = 4
	channelsPerTimer = 4

	maxRetries = 5
)

// TimerConfig holds the current prescaler and period for one hardware timer.
type TimerConfig struct {
	Prescaler uint16
	Period    uint16
}

// MCU communicates with the Robot HAT on-board MCU over I2C.
// All PWM generation and ADC reading goes through this struct.
// Thread-safe: all methods may be called from multiple goroutines.
type MCU struct {
	device i2c.Device
	mu     sync.Mutex

	timers [maxTimers]TimerConfig
}

// NewMCU creates an MCU driver on the given I2C bus.
// It probes the default address (0x14) and falls back to 0x15.
func NewMCU(bus i2c.Bus) (*MCU, error) {
	device := bus.Device(DefaultAddress)

	// Probe: try reading a byte. If it fails, try fallback address.
	_, err := device.ReadByteCtx(context.Background())
	if err != nil {
		device = bus.Device(FallbackAddress)
		_, err = device.ReadByteCtx(context.Background())
		if err != nil {
			return nil, fmt.Errorf("robot hat MCU not found at 0x%02X or 0x%02X: %w",
				DefaultAddress, FallbackAddress, err)
		}
	}

	return &MCU{device: device}, nil
}

// Reset performs the MCU hardware reset sequence via the MCURST GPIO pin.
// The caller must provide the reset pin (MCURST from the board pin map).
// After reset, the MCU needs ~210ms to initialize.
func (m *MCU) Reset(ctx context.Context, resetPin gpio.Pin) error {
	if err := resetPin.SetDirection(ctx, gpio.Output); err != nil {
		return fmt.Errorf("set MCURST direction: %w", err)
	}
	if err := resetPin.Write(ctx, false); err != nil {
		return fmt.Errorf("pull MCURST low: %w", err)
	}
	time.Sleep(1 * time.Millisecond)
	if err := resetPin.Write(ctx, true); err != nil {
		return fmt.Errorf("release MCURST: %w", err)
	}
	time.Sleep(210 * time.Millisecond)
	return nil
}

// writeReg16 writes a 16-bit value to a register, MSB first.
// Retries up to maxRetries on I2C errors (mirrors Python _retry_wrapper).
func (m *MCU) writeReg16(ctx context.Context, reg byte, value uint16) error {
	data := []byte{byte(value >> 8), byte(value & 0xFF)}
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		err = m.device.WriteReg(ctx, reg, data)
		if err == nil {
			return nil
		}
		time.Sleep(1 * time.Millisecond)
	}
	return fmt.Errorf("I2C write reg 0x%02X after %d retries: %w", reg, maxRetries, err)
}

// readReg16 reads a 16-bit value from the MCU. For ADC reads, the protocol
// is: write the channel register, then read two bytes (MSB, LSB).
func (m *MCU) readReg16(ctx context.Context, reg byte) (uint16, error) {
	var err error
	for attempt := 0; attempt < maxRetries; attempt++ {
		// Write the register address with a zero data byte to select the channel
		err = m.device.WriteByteReg(ctx, reg, 0)
		if err != nil {
			time.Sleep(1 * time.Millisecond)
			continue
		}
		// Read MSB
		msb, readErr := m.device.ReadByteCtx(ctx)
		if readErr != nil {
			err = readErr
			time.Sleep(1 * time.Millisecond)
			continue
		}
		// Read LSB
		lsb, readErr := m.device.ReadByteCtx(ctx)
		if readErr != nil {
			err = readErr
			time.Sleep(1 * time.Millisecond)
			continue
		}
		return uint16(msb)<<8 | uint16(lsb), nil
	}
	return 0, fmt.Errorf("I2C read reg 0x%02X after %d retries: %w", reg, maxRetries, err)
}

// --- PWM Operations ---

// SetPWMFrequency configures the timer frequency for the given channel.
// All channels sharing the same timer will be affected.
// Returns the actual frequency achieved after prescaler/period rounding.
func (m *MCU) SetPWMFrequency(ctx context.Context, channel int, hz float64) (float64, error) {
	if channel < 0 || channel >= maxChannels {
		return 0, fmt.Errorf("channel %d out of range 0-%d", channel, maxChannels-1)
	}
	if hz <= 0 {
		return 0, fmt.Errorf("frequency must be positive, got %f", hz)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	timer := channel / channelsPerTimer
	prescaler, period := optimalPrescalerPeriod(hz)

	if err := m.writeReg16(ctx, regPrescaler+byte(timer), prescaler-1); err != nil {
		return 0, fmt.Errorf("set prescaler for timer %d: %w", timer, err)
	}
	if err := m.writeReg16(ctx, regPeriod+byte(timer), period); err != nil {
		return 0, fmt.Errorf("set period for timer %d: %w", timer, err)
	}

	m.timers[timer] = TimerConfig{Prescaler: prescaler, Period: period}

	actualHz := float64(MCUClock) / float64(prescaler) / float64(period)
	return actualHz, nil
}

// SetPWMPulseWidth sets the raw pulse width register for a channel (0 to period).
func (m *MCU) SetPWMPulseWidth(ctx context.Context, channel int, value uint16) error {
	if channel < 0 || channel >= maxChannels {
		return fmt.Errorf("channel %d out of range 0-%d", channel, maxChannels-1)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	return m.writeReg16(ctx, regPWMDuty+byte(channel), value)
}

// SetPWMDutyCycle sets the duty cycle as a percentage (0-100) for a channel.
func (m *MCU) SetPWMDutyCycle(ctx context.Context, channel int, percent float64) error {
	if percent < 0 || percent > 100 {
		return fmt.Errorf("duty cycle %f out of range 0-100", percent)
	}

	m.mu.Lock()
	timer := channel / channelsPerTimer
	period := m.timers[timer].Period
	m.mu.Unlock()

	if period == 0 {
		return fmt.Errorf("timer %d not configured (period is 0)", timer)
	}

	value := uint16(percent / 100.0 * float64(period))
	return m.SetPWMPulseWidth(ctx, channel, value)
}

// TimerPeriod returns the current period for the timer that owns this channel.
func (m *MCU) TimerPeriod(channel int) uint16 {
	m.mu.Lock()
	defer m.mu.Unlock()
	timer := channel / channelsPerTimer
	return m.timers[timer].Period
}

// --- ADC Operations ---

// ReadADC reads a 12-bit analog value (0-4095) from the given channel (0-3).
func (m *MCU) ReadADC(ctx context.Context, channel int) (uint16, error) {
	if channel < 0 || channel > 3 {
		return 0, fmt.Errorf("ADC channel %d out of range 0-3", channel)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Channel register encoding: invert channel, set bit 4
	// A0 -> 0x17, A1 -> 0x16, A2 -> 0x15, A3 -> 0x14
	reg := byte((7 - channel) | regADCBase)
	return m.readReg16(ctx, reg)
}

// ReadADCVoltage reads the ADC and converts to voltage (assuming 3.3V reference).
func (m *MCU) ReadADCVoltage(ctx context.Context, channel int) (float64, error) {
	raw, err := m.ReadADC(ctx, channel)
	if err != nil {
		return 0, err
	}
	return float64(raw) / 4095.0 * 3.3, nil
}

// --- Prescaler/Period Calculation ---

// optimalPrescalerPeriod finds the prescaler and period pair that produces
// the closest frequency to the target. This is a port of the Python
// square-root search from robot_hat/pwm.py.
func optimalPrescalerPeriod(targetHz float64) (prescaler, period uint16) {
	// Start near the geometric mean of prescaler*period
	sqrtTotal := math.Sqrt(float64(MCUClock) / targetHz)
	startPsc := int(sqrtTotal) - 5

	bestError := math.MaxFloat64
	bestPsc := uint16(352)
	bestArr := uint16(4095)

	for psc := startPsc; psc < startPsc+10; psc++ {
		if psc < 1 {
			continue
		}
		arr := int(float64(MCUClock) / targetHz / float64(psc))
		if arr < 1 || arr > 65535 {
			continue
		}
		actualHz := float64(MCUClock) / float64(psc) / float64(arr)
		freqError := math.Abs(targetHz - actualHz)
		if freqError < bestError {
			bestError = freqError
			bestPsc = uint16(psc)
			bestArr = uint16(arr)
		}
	}

	return bestPsc, bestArr
}
```

### 2.2 Board Version Detection (`gorai-driver-robothat/board.go`)

The Robot HAT has two board revisions with different GPIO mappings. Detected by reading GPIO12.

```go
package robothat

import (
	"context"
	"fmt"

	"github.com/gorai/gorai/driver/gpio"
)

// BoardVersion identifies the Robot HAT board revision.
type BoardVersion int

const (
	BoardV1 BoardVersion = 1
	BoardV2 BoardVersion = 2
)

// PinMap maps Robot HAT abstract pin names to BCM GPIO numbers.
type PinMap struct {
	Digital   map[string]int // "D0" -> BCM number
	MCURST    int
	UserButton int
	LED       int
}

var pinMapV1 = PinMap{
	Digital: map[string]int{
		"D0": 17, "D1": 18, "D2": 27, "D3": 22,
		"D4": 23, "D5": 24, "D6": 25, "D7": 4,
		"D8": 5, "D9": 6, "D10": 12, "D11": 13,
		"D12": 19, "D13": 16, "D14": 26, "D15": 20, "D16": 21,
	},
	MCURST:    21,
	UserButton: 19,
	LED:       26,
}

var pinMapV2 = PinMap{
	Digital: map[string]int{
		"D0": 17, "D1": 4, "D2": 27, "D3": 22,
		"D4": 23, "D5": 24, "D6": 25, "D7": 4,
		"D8": 5, "D9": 6, "D10": 12, "D11": 13,
		"D12": 19, "D13": 16, "D14": 26, "D15": 20, "D16": 21,
	},
	MCURST:    5,
	UserButton: 25,
	LED:       26,
}

// DetectBoardVersion reads GPIO12 to determine the board revision.
// GPIO12 HIGH = V2, LOW = V1.
func DetectBoardVersion(gpioDriver gpio.Driver) (BoardVersion, error) {
	pin, err := gpioDriver.Pin(12)
	if err != nil {
		return 0, fmt.Errorf("open GPIO12 for board detection: %w", err)
	}
	if err := pin.SetDirection(context.Background(), gpio.Input); err != nil {
		return 0, fmt.Errorf("set GPIO12 as input: %w", err)
	}
	high, err := pin.Read(context.Background())
	if err != nil {
		return 0, fmt.Errorf("read GPIO12: %w", err)
	}
	if high {
		return BoardV2, nil
	}
	return BoardV1, nil
}

// GetPinMap returns the pin mapping for the detected board version.
func GetPinMap(version BoardVersion) PinMap {
	if version == BoardV2 {
		return pinMapV2
	}
	return pinMapV1
}

// BCMPin resolves a Robot HAT pin name ("D4", "MCURST") to a BCM GPIO number.
func (pm PinMap) BCMPin(name string) (int, error) {
	if name == "MCURST" {
		return pm.MCURST, nil
	}
	if name == "LED" {
		return pm.LED, nil
	}
	if name == "SW" || name == "USER" {
		return pm.UserButton, nil
	}
	bcm, ok := pm.Digital[name]
	if !ok {
		return 0, fmt.Errorf("unknown Robot HAT pin %q", name)
	}
	return bcm, nil
}
```

### 2.3 HC-SR04 Ultrasonic Driver (`gorai-driver-hcsr04/hcsr04.go`)

GPIO time-of-flight measurement. Uses edge detection rather than polling for better precision on Linux.

```go
package hcsr04

import (
	"context"
	"fmt"
	"time"

	"github.com/gorai/gorai/driver/gpio"
)

const (
	speedOfSoundCMPerSec = 34300.0 // cm/s at ~20C
	triggerPulseWidth    = 10 * time.Microsecond
	defaultTimeout       = 20 * time.Millisecond
	defaultRetries       = 10
	settleDelay          = 10 * time.Millisecond
)

// Driver reads distance from an HC-SR04 ultrasonic sensor.
type Driver struct {
	trig    gpio.Pin
	echo    gpio.Pin
	timeout time.Duration
	retries int
}

// Config configures the HC-SR04 driver.
type Config struct {
	TrigPin gpio.Pin
	EchoPin gpio.Pin
	Timeout time.Duration // Default: 20ms
	Retries int           // Default: 10
}

// New creates a new HC-SR04 driver.
func New(ctx context.Context, cfg Config) (*Driver, error) {
	if err := cfg.TrigPin.SetDirection(ctx, gpio.Output); err != nil {
		return nil, fmt.Errorf("set TRIG as output: %w", err)
	}
	if err := cfg.EchoPin.SetDirection(ctx, gpio.Input); err != nil {
		return nil, fmt.Errorf("set ECHO as input: %w", err)
	}
	// Ensure TRIG starts low
	if err := cfg.TrigPin.Write(ctx, false); err != nil {
		return nil, fmt.Errorf("set TRIG low: %w", err)
	}

	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = defaultTimeout
	}
	retries := cfg.Retries
	if retries == 0 {
		retries = defaultRetries
	}

	return &Driver{
		trig:    cfg.TrigPin,
		echo:    cfg.EchoPin,
		timeout: timeout,
		retries: retries,
	}, nil
}

// ReadDistance returns the distance in centimeters.
// Retries up to the configured number of times on timeout.
// Returns -1 if all retries fail.
func (d *Driver) ReadDistance(ctx context.Context) (float64, error) {
	for i := 0; i < d.retries; i++ {
		dist, err := d.readOnce(ctx)
		if err == nil {
			return dist, nil
		}
		select {
		case <-ctx.Done():
			return -1, ctx.Err()
		default:
		}
	}
	return -1, nil
}

func (d *Driver) readOnce(ctx context.Context) (float64, error) {
	// Let sensor settle
	d.trig.Write(ctx, false)
	time.Sleep(settleDelay)

	// Send 10us trigger pulse
	d.trig.Write(ctx, true)
	time.Sleep(triggerPulseWidth)
	d.trig.Write(ctx, false)

	// Wait for echo rising edge
	deadline := time.Now().Add(d.timeout)
	for {
		high, err := d.echo.Read(ctx)
		if err != nil {
			return 0, err
		}
		if high {
			break
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("timeout waiting for echo start")
		}
	}

	// Measure echo high duration
	echoStart := time.Now()
	for {
		high, err := d.echo.Read(ctx)
		if err != nil {
			return 0, err
		}
		if !high {
			break
		}
		if time.Now().After(deadline) {
			return 0, fmt.Errorf("timeout waiting for echo end")
		}
	}

	duration := time.Since(echoStart)
	distanceCM := duration.Seconds() * speedOfSoundCMPerSec / 2.0
	return distanceCM, nil
}
```

**Production improvement:** Replace the polling loop with `gpio.EdgeWaiter.WaitForEdge()` if the pin supports it. The `gpiocdev` driver supports edge events via the kernel, which avoids busy-waiting and gives sub-microsecond edge timestamps:

```go
// Better version using edge detection (if pin implements gpio.EdgeWaiter)
func (d *Driver) readOnceEdge(ctx context.Context) (float64, error) {
	edgePin, ok := d.echo.(gpio.EdgeWaiter)
	if !ok {
		return d.readOnce(ctx) // fall back to polling
	}

	d.trig.Write(ctx, false)
	time.Sleep(settleDelay)
	d.trig.Write(ctx, true)
	time.Sleep(triggerPulseWidth)
	d.trig.Write(ctx, false)

	// Wait for rising edge (echo start)
	if ok, _ := edgePin.WaitForEdge(ctx, gpio.EdgeRising, d.timeout); !ok {
		return 0, fmt.Errorf("timeout waiting for echo rising edge")
	}
	echoStart := time.Now()

	// Wait for falling edge (echo end)
	if ok, _ := edgePin.WaitForEdge(ctx, gpio.EdgeFalling, d.timeout); !ok {
		return 0, fmt.Errorf("timeout waiting for echo falling edge")
	}
	duration := time.Since(echoStart)

	return duration.Seconds() * speedOfSoundCMPerSec / 2.0, nil
}
```

---

## 3. Component Layer

### 3.1 Robot HAT Servo (`gorai-picarx/servo/servo.go`)

Maps the Python `Servo` class. One instance per servo channel.

```go
package robothat

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/gorai/gorai/components/servo"
	"github.com/gorai/gorai/driver/robothat"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("servo", "robot-hat-pwm-servo", New)
}

const (
	defaultFrequencyHz = 50
	defaultMinPulseUs  = 500.0
	defaultMaxPulseUs  = 2500.0
	defaultMinAngle    = -90.0
	defaultMaxAngle    = 90.0
	periodUs           = 20000.0 // 1/50Hz = 20ms
)

// Servo controls a single RC servo via the Robot HAT MCU's PWM output.
type Servo struct {
	name         resource.Name
	mcu          *robothat.MCU
	channel      int
	minPulseUs   float64
	maxPulseUs   float64
	minAngle     float64
	maxAngle     float64
	calibOffset  float64
	currentAngle float64
	mu           sync.Mutex
	logger       *slog.Logger
}

func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)

	channel, ok := conf["pwm_channel"].(float64)
	if !ok {
		return nil, fmt.Errorf("pwm_channel is required")
	}

	// Get MCU dependency
	mcuDep, err := deps.Get("robot_hat_mcu")
	if err != nil {
		// If not passed as dep, look it up by the name in attributes
		mcuName, _ := conf["mcu"].(string)
		if mcuName != "" {
			mcuDep, err = deps.Get(mcuName)
		}
		if err != nil {
			return nil, fmt.Errorf("robot_hat_mcu dependency required: %w", err)
		}
	}
	mcu, ok := mcuDep.(*robothat.MCU)
	if !ok {
		return nil, fmt.Errorf("robot_hat_mcu dependency is not a *robothat.MCU")
	}

	logger, _ := deps.Get("logger")
	log, _ := logger.(*slog.Logger)
	if log == nil {
		log = slog.Default()
	}

	minPulse := floatOr(conf, "min_pulse_us", defaultMinPulseUs)
	maxPulse := floatOr(conf, "max_pulse_us", defaultMaxPulseUs)
	minAngle := floatOr(conf, "min_angle", defaultMinAngle)
	maxAngle := floatOr(conf, "max_angle", defaultMaxAngle)
	calibOffset := floatOr(conf, "calibration_offset", 0)

	s := &Servo{
		name:        resource.NewComponentName("gorai", "servo", nameStr),
		mcu:         mcu,
		channel:     int(channel),
		minPulseUs:  minPulse,
		maxPulseUs:  maxPulse,
		minAngle:    minAngle,
		maxAngle:    maxAngle,
		calibOffset: calibOffset,
		logger:      log,
	}

	// Configure PWM frequency for this channel's timer
	actualHz, err := mcu.SetPWMFrequency(ctx, s.channel, defaultFrequencyHz)
	if err != nil {
		return nil, fmt.Errorf("set servo PWM frequency: %w", err)
	}
	s.logger.Debug("servo PWM configured",
		"channel", s.channel, "target_hz", defaultFrequencyHz, "actual_hz", actualHz)

	return s, nil
}

// SetAngle sets the servo to the given angle in degrees.
// The angle is clamped to [minAngle, maxAngle] and the calibration offset is applied.
func (s *Servo) SetAngle(ctx context.Context, degrees float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Clamp to configured range
	if degrees < s.minAngle {
		degrees = s.minAngle
	}
	if degrees > s.maxAngle {
		degrees = s.maxAngle
	}

	// Apply calibration offset
	adjusted := degrees + s.calibOffset

	// Map angle to pulse width in microseconds
	// Linear interpolation: minAngle -> minPulseUs, maxAngle -> maxPulseUs
	fraction := (adjusted - defaultMinAngle) / (defaultMaxAngle - defaultMinAngle)
	pulseUs := s.minPulseUs + fraction*(s.maxPulseUs-s.minPulseUs)

	// Convert pulse width to timer counts
	// pulseUs / periodUs gives the duty cycle fraction
	// Multiply by the timer period register value to get counts
	period := s.mcu.TimerPeriod(s.channel)
	if period == 0 {
		return fmt.Errorf("timer period not configured for channel %d", s.channel)
	}
	value := uint16(pulseUs / periodUs * float64(period))

	if err := s.mcu.SetPWMPulseWidth(ctx, s.channel, value); err != nil {
		return fmt.Errorf("set servo pulse width: %w", err)
	}

	s.currentAngle = degrees
	return nil
}

func (s *Servo) GetAngle(ctx context.Context) (float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.currentAngle, nil
}

func (s *Servo) SetSpeed(ctx context.Context, speed float64) error {
	return nil // Standard RC servos don't support speed control
}

func (s *Servo) SetTorqueLimit(ctx context.Context, limit float64) error {
	return nil // Standard RC servos don't support torque limiting
}

func (s *Servo) GetProperties(ctx context.Context) (servo.Properties, error) {
	return servo.Properties{
		MinAngle:     s.minAngle,
		MaxAngle:     s.maxAngle,
		IsContinuous: false,
		HasFeedback:  false,
		Protocol:     "pwm",
	}, nil
}

func (s *Servo) IsMoving(ctx context.Context) (bool, error) { return false, nil }
func (s *Servo) Stop(ctx context.Context) error             { return nil }
func (s *Servo) Name() resource.Name                        { return s.name }
func (s *Servo) Close(ctx context.Context) error            { return nil }

func (s *Servo) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

func (s *Servo) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("no custom commands")
}

func floatOr(conf registry.Config, key string, defaultValue float64) float64 {
	if v, ok := conf[key].(float64); ok {
		return v
	}
	return defaultValue
}
```

### 3.2 Robot HAT DC Motor (`gorai-picarx/motor/motor.go`)

Maps the Python motor control: GPIO direction pin + MCU PWM speed. Includes the `speed/2 + 50` mapping.

```go
package robothat

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/gorai/gorai/components/motor"
	"github.com/gorai/gorai/driver/gpio"
	"github.com/gorai/gorai/driver/robothat"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("motor", "robot-hat-dc-motor", New)
}

const (
	motorPeriod    = 4095
	motorPrescaler = 10
)

// Motor controls a single DC motor via a Robot HAT GPIO direction pin
// and MCU PWM speed channel.
type Motor struct {
	name         resource.Name
	mcu          *robothat.MCU
	channel      int
	dirPin       gpio.Pin
	invertDir    bool
	calibDir     int // +1 or -1, from calibration
	power        float64
	moving       bool
	mu           sync.Mutex
	logger       *slog.Logger
}

func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)

	channel, ok := conf["pwm_channel"].(float64)
	if !ok {
		return nil, fmt.Errorf("pwm_channel is required")
	}

	// Get MCU
	mcuDep, err := deps.Get("robot_hat_mcu")
	if err != nil {
		return nil, fmt.Errorf("robot_hat_mcu dependency required: %w", err)
	}
	mcu, ok := mcuDep.(*robothat.MCU)
	if !ok {
		return nil, fmt.Errorf("robot_hat_mcu is not a *robothat.MCU")
	}

	// Direction GPIO pin must be provided as a resolved gpio.Pin dependency
	dirPinDep, err := deps.Get(conf["direction_pin"].(string))
	if err != nil {
		return nil, fmt.Errorf("direction_pin dependency required: %w", err)
	}
	dirPin, ok := dirPinDep.(gpio.Pin)
	if !ok {
		return nil, fmt.Errorf("direction_pin is not a gpio.Pin")
	}
	if err := dirPin.SetDirection(ctx, gpio.Output); err != nil {
		return nil, fmt.Errorf("set direction pin as output: %w", err)
	}

	invertDir := false
	if v, ok := conf["invert_direction"].(bool); ok {
		invertDir = v
	}

	calibDir := 1
	if v, ok := conf["calibration_direction"].(float64); ok && v < 0 {
		calibDir = -1
	}

	logger, _ := deps.Get("logger")
	log, _ := logger.(*slog.Logger)
	if log == nil {
		log = slog.Default()
	}

	m := &Motor{
		name:      resource.NewComponentName("gorai", "motor", nameStr),
		mcu:       mcu,
		channel:   int(channel),
		dirPin:    dirPin,
		invertDir: invertDir,
		calibDir:  calibDir,
		logger:    log,
	}

	// Configure motor PWM timer
	if _, err := mcu.SetPWMFrequency(ctx, m.channel, float64(robothat.MCUClock)/float64(motorPrescaler)/float64(motorPeriod)); err != nil {
		return nil, fmt.Errorf("set motor PWM frequency: %w", err)
	}

	return m, nil
}

// SetPower sets motor power from -1.0 (full reverse) to 1.0 (full forward).
// Applies the PiCar-X speed mapping: speed/2 + 50 for non-zero speeds,
// which maps 1-100% input to 50-100% duty cycle.
func (m *Motor) SetPower(ctx context.Context, power float64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clamp to [-1, 1]
	if power > 1 {
		power = 1
	}
	if power < -1 {
		power = -1
	}

	// Convert -1..1 to -100..100 percentage
	speedPercent := power * 100.0

	// Apply calibration direction
	effectiveDir := 1
	if speedPercent < 0 {
		effectiveDir = -1
	}
	effectiveDir *= m.calibDir
	if m.invertDir {
		effectiveDir *= -1
	}

	absSpeed := math.Abs(speedPercent)

	// PiCar-X speed mapping: non-zero speeds map to 50-100% duty
	// This provides minimum torque to actually move the motor
	var duty float64
	if absSpeed > 0 {
		duty = absSpeed/2.0 + 50.0
	}

	// Set direction pin
	dirHigh := effectiveDir < 0 // HIGH = reverse in PiCar-X
	if err := m.dirPin.Write(ctx, dirHigh); err != nil {
		return fmt.Errorf("set direction: %w", err)
	}

	// Set PWM duty cycle
	if err := m.mcu.SetPWMDutyCycle(ctx, m.channel, duty); err != nil {
		return fmt.Errorf("set motor PWM: %w", err)
	}

	m.power = power
	m.moving = power != 0
	return nil
}

func (m *Motor) Stop(ctx context.Context) error {
	return m.SetPower(ctx, 0)
}

func (m *Motor) IsMoving(ctx context.Context) (bool, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.moving, nil
}

func (m *Motor) IsPowered(ctx context.Context) (bool, float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.moving, m.power, nil
}

func (m *Motor) SetVelocity(ctx context.Context, velocity float64) error {
	// No encoder feedback, so velocity = power
	return m.SetPower(ctx, velocity)
}

func (m *Motor) GoTo(ctx context.Context, position, velocity float64) error {
	return fmt.Errorf("GoTo not supported: no encoder feedback")
}

func (m *Motor) GoFor(ctx context.Context, rpm, revolutions float64) error {
	return fmt.Errorf("GoFor not supported: no encoder feedback")
}

func (m *Motor) GetPosition(ctx context.Context) (float64, error) {
	return 0, fmt.Errorf("position not available: no encoder")
}

func (m *Motor) GetVelocity(ctx context.Context) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.power, nil // Best we can do without an encoder
}

func (m *Motor) ResetZeroPosition(ctx context.Context, offset float64) error {
	return fmt.Errorf("position not available: no encoder")
}

func (m *Motor) Properties(ctx context.Context) (motor.Properties, error) {
	return motor.Properties{
		PositionReporting: false,
		VelocityReporting: false,
		SupportsGoTo:      false,
	}, nil
}

func (m *Motor) Name() resource.Name { return m.name }
func (m *Motor) Close(ctx context.Context) error {
	return m.Stop(ctx)
}
func (m *Motor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}
func (m *Motor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("no custom commands")
}
```

### 3.3 Ackermann-Differential Base (`gorai-picarx/base/base.go`)

Composes two motors + one steering servo into a `base.Base`. Ports the Python `forward()`, `backward()`, `set_dir_servo_angle()` logic including the differential speed scaling.

```go
package ackermann

import (
	"context"
	"fmt"
	"math"
	"sync"

	"github.com/gorai/gorai/components/base"
	servocomp "github.com/gorai/gorai/components/servo"
	motorcomp "github.com/gorai/gorai/components/motor"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("base", "ackermann-differential", New)
}

// Base implements Ackermann-like steering with differential rear drive.
// When turning, the inner wheel slows proportionally to steering angle.
type Base struct {
	name           resource.Name
	leftMotor      motorcomp.Motor
	rightMotor     motorcomp.Motor
	steerServo     servocomp.Servo
	maxSteerAngle  float64 // degrees
	currentAngle   float64
	currentSpeed   float64
	mu             sync.Mutex
}

func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)

	leftName, _ := conf["left_motor"].(string)
	rightName, _ := conf["right_motor"].(string)
	steerName, _ := conf["steering_servo"].(string)

	leftDep, err := deps.Get(leftName)
	if err != nil {
		return nil, fmt.Errorf("left_motor %q: %w", leftName, err)
	}
	rightDep, err := deps.Get(rightName)
	if err != nil {
		return nil, fmt.Errorf("right_motor %q: %w", rightName, err)
	}
	steerDep, err := deps.Get(steerName)
	if err != nil {
		return nil, fmt.Errorf("steering_servo %q: %w", steerName, err)
	}

	leftMotor, ok := leftDep.(motorcomp.Motor)
	if !ok {
		return nil, fmt.Errorf("left_motor is not a Motor")
	}
	rightMotor, ok := rightDep.(motorcomp.Motor)
	if !ok {
		return nil, fmt.Errorf("right_motor is not a Motor")
	}
	steerServo, ok := steerDep.(servocomp.Servo)
	if !ok {
		return nil, fmt.Errorf("steering_servo is not a Servo")
	}

	maxSteer := 30.0
	if v, ok := conf["max_steering_angle"].(float64); ok {
		maxSteer = v
	}

	return &Base{
		name:          resource.NewComponentName("gorai", "base", nameStr),
		leftMotor:     leftMotor,
		rightMotor:    rightMotor,
		steerServo:    steerServo,
		maxSteerAngle: maxSteer,
	}, nil
}

// SetVelocity sets linear speed (m/s mapped to -1..1 power) and angular
// velocity (mapped to steering angle). This is the primary control method.
func (b *Base) SetVelocity(ctx context.Context, linear, angular float64) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Convert angular velocity to steering angle
	// Normalize: angular in [-1, 1] maps to [-maxSteerAngle, +maxSteerAngle]
	steerAngle := angular * b.maxSteerAngle
	if steerAngle > b.maxSteerAngle {
		steerAngle = b.maxSteerAngle
	}
	if steerAngle < -b.maxSteerAngle {
		steerAngle = -b.maxSteerAngle
	}

	if err := b.steerServo.SetAngle(ctx, steerAngle); err != nil {
		return fmt.Errorf("set steering: %w", err)
	}
	b.currentAngle = steerAngle

	// Apply differential speed scaling based on steering angle.
	// Port of the Python Picarx.forward() logic:
	// Inner wheel slows by (100 - abs(angle)) / 100
	leftPower := linear
	rightPower := -linear // Right motor is physically mirrored

	if steerAngle != 0 {
		absAngle := math.Abs(steerAngle)
		if absAngle > b.maxSteerAngle {
			absAngle = b.maxSteerAngle
		}
		scale := (100.0 - absAngle) / 100.0

		if steerAngle > 0 {
			// Turning right: slow left (inner) wheel
			leftPower = linear * scale
		} else {
			// Turning left: slow right (inner) wheel
			rightPower = -linear * scale
		}
	}

	if err := b.leftMotor.SetPower(ctx, leftPower); err != nil {
		return fmt.Errorf("set left motor: %w", err)
	}
	if err := b.rightMotor.SetPower(ctx, rightPower); err != nil {
		return fmt.Errorf("set right motor: %w", err)
	}

	b.currentSpeed = linear
	return nil
}

func (b *Base) SetPower(ctx context.Context, linear, angular float64) error {
	return b.SetVelocity(ctx, linear, angular)
}

func (b *Base) MoveStraight(ctx context.Context, distanceM, speedMPerS float64) error {
	// No encoder feedback, so we can't measure distance.
	// Best effort: drive for estimated time then stop.
	return fmt.Errorf("MoveStraight not supported: no encoder feedback")
}

func (b *Base) Spin(ctx context.Context, angleRad, speedRadPerS float64) error {
	return fmt.Errorf("Spin not supported: Ackermann steering cannot spin in place")
}

func (b *Base) Stop(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.currentSpeed = 0
	_ = b.leftMotor.Stop(ctx)
	_ = b.rightMotor.Stop(ctx)
	return nil
}

func (b *Base) IsMoving(ctx context.Context) (bool, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.currentSpeed != 0, nil
}

func (b *Base) Properties(ctx context.Context) (base.Properties, error) {
	return base.Properties{
		MaxLinearVelocity:  1.0,  // Power units, not m/s
		MaxAngularVelocity: 1.0,  // Maps to max steering angle
	}, nil
}

func (b *Base) Name() resource.Name { return b.name }
func (b *Base) Close(ctx context.Context) error {
	return b.Stop(ctx)
}
func (b *Base) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}
func (b *Base) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("no custom commands")
}
```

### 3.4 Robot HAT Grayscale Sensor (`gorai-picarx/grayscale/sensor.go`)

Reads 3 ADC channels from the MCU and provides line-following status.

```go
package robothat

import (
	"context"
	"fmt"
	"sync"

	"github.com/gorai/gorai/components/sensor"
	"github.com/gorai/gorai/driver/robothat"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("sensor", "robot-hat-grayscale", New)
}

// GrayscaleSensor reads 3-channel reflectance data from the Robot HAT MCU ADC.
type GrayscaleSensor struct {
	name          resource.Name
	mcu           *robothat.MCU
	channels      [3]int      // ADC channel numbers (typically 0, 1, 2)
	lineRef       [3]float64  // Line-following reference thresholds
	cliffRef      [3]float64  // Cliff detection thresholds
	mu            sync.Mutex
}

func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)

	mcuDep, err := deps.Get("robot_hat_mcu")
	if err != nil {
		return nil, fmt.Errorf("robot_hat_mcu dependency required: %w", err)
	}
	mcu, ok := mcuDep.(*robothat.MCU)
	if !ok {
		return nil, fmt.Errorf("robot_hat_mcu is not a *robothat.MCU")
	}

	// Default channels A0, A1, A2
	channels := [3]int{0, 1, 2}
	lineRef := [3]float64{1000, 1000, 1000}
	cliffRef := [3]float64{500, 500, 500}

	if refs, ok := conf["line_reference"].([]any); ok && len(refs) == 3 {
		for i, r := range refs {
			if v, ok := r.(float64); ok {
				lineRef[i] = v
			}
		}
	}
	if refs, ok := conf["cliff_reference"].([]any); ok && len(refs) == 3 {
		for i, r := range refs {
			if v, ok := r.(float64); ok {
				cliffRef[i] = v
			}
		}
	}

	return &GrayscaleSensor{
		name:     resource.NewComponentName("gorai", "sensor", nameStr),
		mcu:      mcu,
		channels: channels,
		lineRef:  lineRef,
		cliffRef: cliffRef,
	}, nil
}

// GetReflectances returns normalized reflectance values (0.0-1.0) for each channel.
func (s *GrayscaleSensor) GetReflectances(ctx context.Context) ([]float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]float64, 3)
	for i, ch := range s.channels {
		raw, err := s.mcu.ReadADC(ctx, ch)
		if err != nil {
			return nil, fmt.Errorf("read ADC channel %d: %w", ch, err)
		}
		result[i] = float64(raw) / 4095.0
	}
	return result, nil
}

// GetLinePosition returns a weighted line position from -1.0 (left) to +1.0 (right).
// Returns 0.0 when centered.
func (s *GrayscaleSensor) GetLinePosition(ctx context.Context) (float64, error) {
	raw, err := s.readRaw(ctx)
	if err != nil {
		return 0, err
	}

	// Determine which sensors detect the line (above reference threshold)
	left := raw[0] > s.lineRef[0]
	center := raw[1] > s.lineRef[1]
	right := raw[2] > s.lineRef[2]

	// Weighted position: -1 = left, 0 = center, +1 = right
	switch {
	case left && !center && !right:
		return -1.0, nil
	case left && center && !right:
		return -0.5, nil
	case !left && center && !right:
		return 0.0, nil
	case !left && center && right:
		return 0.5, nil
	case !left && !center && right:
		return 1.0, nil
	default:
		return 0.0, nil
	}
}

// Calibrate is a no-op; calibration values are set via RDL configuration.
func (s *GrayscaleSensor) Calibrate(ctx context.Context) error {
	return nil
}

// Readings returns the sensor data as a map for the generic Sensor interface.
func (s *GrayscaleSensor) Readings(ctx context.Context) (map[string]any, error) {
	raw, err := s.readRaw(ctx)
	if err != nil {
		return nil, err
	}
	pos, _ := s.GetLinePosition(ctx)

	return map[string]any{
		"left":          raw[0],
		"center":        raw[1],
		"right":         raw[2],
		"line_position": pos,
	}, nil
}

func (s *GrayscaleSensor) readRaw(ctx context.Context) ([3]float64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var result [3]float64
	for i, ch := range s.channels {
		raw, err := s.mcu.ReadADC(ctx, ch)
		if err != nil {
			return result, fmt.Errorf("read ADC channel %d: %w", ch, err)
		}
		result[i] = float64(raw)
	}
	return result, nil
}

func (s *GrayscaleSensor) Name() resource.Name { return s.name }
func (s *GrayscaleSensor) Close(ctx context.Context) error { return nil }
func (s *GrayscaleSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}
func (s *GrayscaleSensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("no custom commands")
}
```

### 3.5 HC-SR04 Range Sensor (`gorai-driver-hcsr04/component.go`)

Wraps the HC-SR04 driver as a GoRAI `RangeSensor` component.

```go
package hcsr04

import (
	"context"
	"fmt"

	"github.com/gorai/gorai/components/sensor"
	"github.com/gorai/gorai/driver/gpio"
	hcsr04driver "github.com/gorai/gorai/driver/hcsr04"
	"github.com/gorai/gorai/driver/robothat"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("sensor", "hc-sr04", New)
}

type RangeSensor struct {
	name   resource.Name
	driver *hcsr04driver.Driver
}

func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)

	// Resolve trigger and echo GPIO pins via the Robot HAT pin map
	trigName, _ := conf["trigger_pin"].(string) // "D2"
	echoName, _ := conf["echo_pin"].(string)    // "D3"

	gpioDep, err := deps.Get("gpio")
	if err != nil {
		return nil, fmt.Errorf("gpio dependency required: %w", err)
	}
	gpioDriver, ok := gpioDep.(gpio.Driver)
	if !ok {
		return nil, fmt.Errorf("gpio dependency is not a gpio.Driver")
	}

	// Look up BCM pin numbers from Robot HAT pin map
	pinMapDep, _ := deps.Get("pin_map")
	pinMap, _ := pinMapDep.(*robothat.PinMap)
	if pinMap == nil {
		return nil, fmt.Errorf("pin_map dependency required for Robot HAT pin resolution")
	}

	trigBCM, err := pinMap.BCMPin(trigName)
	if err != nil {
		return nil, fmt.Errorf("resolve trigger pin %q: %w", trigName, err)
	}
	echoBCM, err := pinMap.BCMPin(echoName)
	if err != nil {
		return nil, fmt.Errorf("resolve echo pin %q: %w", echoName, err)
	}

	trigPin, err := gpioDriver.Pin(trigBCM)
	if err != nil {
		return nil, fmt.Errorf("open trigger GPIO %d: %w", trigBCM, err)
	}
	echoPin, err := gpioDriver.Pin(echoBCM)
	if err != nil {
		return nil, fmt.Errorf("open echo GPIO %d: %w", echoBCM, err)
	}

	driver, err := hcsr04driver.New(ctx, hcsr04driver.Config{
		TrigPin: trigPin,
		EchoPin: echoPin,
	})
	if err != nil {
		return nil, err
	}

	return &RangeSensor{
		name:   resource.NewComponentName("gorai", "sensor", nameStr),
		driver: driver,
	}, nil
}

// GetRange returns distance in meters.
func (s *RangeSensor) GetRange(ctx context.Context) (float64, error) {
	cm, err := s.driver.ReadDistance(ctx)
	if err != nil {
		return 0, err
	}
	if cm < 0 {
		return 0, fmt.Errorf("no echo received")
	}
	return cm / 100.0, nil // Convert cm to meters
}

func (s *RangeSensor) GetRanges(ctx context.Context) ([]float64, error) {
	d, err := s.GetRange(ctx)
	if err != nil {
		return nil, err
	}
	return []float64{d}, nil
}

func (s *RangeSensor) GetMinRange(ctx context.Context) (float64, error) { return 0.02, nil }
func (s *RangeSensor) GetMaxRange(ctx context.Context) (float64, error) { return 4.0, nil }

func (s *RangeSensor) Properties(ctx context.Context) (sensor.RangeSensorProperties, error) {
	return sensor.RangeSensorProperties{
		MinRange:    0.02,  // 2 cm
		MaxRange:    4.0,   // 400 cm
		FieldOfView: 0.26,  // ~15 degrees
		NumPoints:   1,
	}, nil
}

func (s *RangeSensor) Readings(ctx context.Context) (map[string]any, error) {
	d, err := s.GetRange(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]any{"distance_m": d}, nil
}

func (s *RangeSensor) Name() resource.Name { return s.name }
func (s *RangeSensor) Close(ctx context.Context) error { return nil }
func (s *RangeSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}
func (s *RangeSensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, fmt.Errorf("no custom commands")
}
```

---

## 4. Calibration Store (`gorai-driver-robothat/calibration.go`)

Ports the Python `fileDB` class. Simple text key-value file compatible with the existing `/opt/picar-x/picar-x.conf` format so calibration data can be shared between the Python and Go stacks during transition.

```go
package calibration

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
)

// FileDB is a simple file-based key-value store, compatible with the
// SunFounder robot_hat fileDB format.
//
// File format:
//   key = value
//
// Values are stored as strings. Typed accessors parse on read.
type FileDB struct {
	path string
	data map[string]string
	mu   sync.RWMutex
}

// Open loads a calibration file. Creates the file if it doesn't exist.
func Open(path string) (*FileDB, error) {
	db := &FileDB{
		path: path,
		data: make(map[string]string),
	}
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return db, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open calibration file: %w", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		db.data[key] = value
	}
	return db, scanner.Err()
}

// GetFloat returns a float64 value, or the default if the key is missing.
func (db *FileDB) GetFloat(key string, defaultValue float64) float64 {
	db.mu.RLock()
	defer db.mu.RUnlock()
	if v, ok := db.data[key]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return defaultValue
}

// GetFloatSlice returns a []float64 parsed from "[1.0, 2.0, 3.0]" format.
func (db *FileDB) GetFloatSlice(key string, defaultValue []float64) []float64 {
	db.mu.RLock()
	defer db.mu.RUnlock()
	v, ok := db.data[key]
	if !ok {
		return defaultValue
	}
	v = strings.Trim(v, "[]")
	parts := strings.Split(v, ",")
	result := make([]float64, 0, len(parts))
	for _, p := range parts {
		f, err := strconv.ParseFloat(strings.TrimSpace(p), 64)
		if err != nil {
			return defaultValue
		}
		result = append(result, f)
	}
	return result
}

// GetIntSlice returns a []int parsed from "[1, -1]" format.
func (db *FileDB) GetIntSlice(key string, defaultValue []int) []int {
	db.mu.RLock()
	defer db.mu.RUnlock()
	v, ok := db.data[key]
	if !ok {
		return defaultValue
	}
	v = strings.Trim(v, "[]")
	parts := strings.Split(v, ",")
	result := make([]int, 0, len(parts))
	for _, p := range parts {
		i, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return defaultValue
		}
		result = append(result, i)
	}
	return result
}

// Set stores a value and writes the file.
func (db *FileDB) Set(key string, value any) error {
	db.mu.Lock()
	defer db.mu.Unlock()
	db.data[key] = fmt.Sprintf("%v", value)
	return db.save()
}

func (db *FileDB) save() error {
	f, err := os.Create(db.path)
	if err != nil {
		return fmt.Errorf("write calibration file: %w", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	for k, v := range db.data {
		fmt.Fprintf(w, "%s = %s\n", k, v)
	}
	return w.Flush()
}
```

---

## 5. Example RDL (`gorai-picarx/examples/robot.json`)

```json
{
  "version": "2",

  "robot": {
    "name": "picar-x",
    "description": "SunFounder PiCar-X on GoRAI"
  },

  "components": [
    {
      "name": "robot-hat-mcu",
      "type": "i2c_device",
      "model": "sunfounder-robot-hat",
      "attributes": {
        "i2c_bus": "/dev/i2c-1",
        "address": "0x14",
        "reset_pin": "MCURST"
      }
    },
    {
      "name": "steering",
      "type": "servo",
      "model": "robot-hat-pwm-servo",
      "depends_on": ["robot-hat-mcu"],
      "attributes": {
        "pwm_channel": 2,
        "min_angle": -30,
        "max_angle": 30,
        "calibration_key": "picarx_dir_servo"
      }
    },
    {
      "name": "cam-pan",
      "type": "servo",
      "model": "robot-hat-pwm-servo",
      "depends_on": ["robot-hat-mcu"],
      "attributes": {
        "pwm_channel": 0,
        "min_angle": -90,
        "max_angle": 90,
        "calibration_key": "picarx_cam_pan_servo"
      }
    },
    {
      "name": "cam-tilt",
      "type": "servo",
      "model": "robot-hat-pwm-servo",
      "depends_on": ["robot-hat-mcu"],
      "attributes": {
        "pwm_channel": 1,
        "min_angle": -35,
        "max_angle": 65,
        "calibration_key": "picarx_cam_tilt_servo"
      }
    },
    {
      "name": "left-motor",
      "type": "motor",
      "model": "robot-hat-dc-motor",
      "depends_on": ["robot-hat-mcu"],
      "attributes": {
        "pwm_channel": 13,
        "direction_pin": "D4",
        "invert_direction": false
      }
    },
    {
      "name": "right-motor",
      "type": "motor",
      "model": "robot-hat-dc-motor",
      "depends_on": ["robot-hat-mcu"],
      "attributes": {
        "pwm_channel": 12,
        "direction_pin": "D5",
        "invert_direction": true
      }
    },
    {
      "name": "drive",
      "type": "base",
      "model": "ackermann-differential",
      "depends_on": ["left-motor", "right-motor", "steering"],
      "attributes": {
        "left_motor": "left-motor",
        "right_motor": "right-motor",
        "steering_servo": "steering",
        "max_steering_angle": 30
      }
    },
    {
      "name": "ultrasonic",
      "type": "sensor",
      "model": "hc-sr04",
      "attributes": {
        "trigger_pin": "D2",
        "echo_pin": "D3"
      }
    },
    {
      "name": "line-sensor",
      "type": "sensor",
      "model": "robot-hat-grayscale",
      "depends_on": ["robot-hat-mcu"],
      "attributes": {
        "channels": [0, 1, 2],
        "line_reference": [1000, 1000, 1000],
        "cliff_reference": [500, 500, 500]
      }
    },
    {
      "name": "camera",
      "type": "camera",
      "model": "v4l2",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480,
        "frame_rate": 30
      }
    }
  ],

  "dashboard": {
    "enabled": true,
    "listen": "127.0.0.1:8080"
  }
}
```

---

## 6. Python-to-Go Mapping Reference

| Python Class | Python File | Go Module | Go Type | Registration |
|---|---|---|---|---|
| `I2C` | `robot_hat/i2c.py` | `gorai-driver-robothat` | `MCU` | (driver, registered as component) |
| `PWM` | `robot_hat/pwm.py` | `gorai-driver-robothat` | `MCU.SetPWM*` methods | (methods on MCU) |
| `ADC` | `robot_hat/adc.py` | `gorai-driver-robothat` | `MCU.ReadADC` method | (methods on MCU) |
| `Pin` | `robot_hat/pin.py` | `gorai/driver/gpio` | `gpio.Pin` | (existing GoRAI core) |
| `Servo` | `robot_hat/servo.py` | `gorai-picarx/servo` | `Servo` | `"servo", "robot-hat-pwm-servo"` |
| `Motor` (internal) | `picarx/picarx.py` | `gorai-picarx/motor` | `Motor` | `"motor", "robot-hat-dc-motor"` |
| `Picarx.forward/backward` | `picarx/picarx.py` | `gorai-picarx/base` | `Base` | `"base", "ackermann-differential"` |
| `Ultrasonic` | `robot_hat/modules/ultrasonic.py` | `gorai-driver-hcsr04` | `Driver` + `RangeSensor` | `"sensor", "hc-sr04"` |
| `Grayscale_Module` | `robot_hat/modules/grayscale_module.py` | `gorai-picarx/grayscale` | `GrayscaleSensor` | `"sensor", "robot-hat-grayscale"` |
| `fileDB` | `robot_hat/filedb.py` | `gorai-driver-robothat` | `FileDB` | (utility, loaded by MCU component) |
| `Vilib` | `vilib/vilib.py` | satellite repo (future) | (future) | (future) |

---

## 7. What Is NOT Ported

These Python features are out of scope for the initial Go port:

| Feature | Python Module | Reason |
|---|---|---|
| Text-to-speech | `robot_hat/tts.py` | Not robotics-critical; shell out to `espeak` if needed |
| Music playback | `robot_hat/music.py` | pygame dependency; not needed for autonomy |
| RGB LED control | `robot_hat/modules/rgb_led.py` | Trivial addition later via MCU PWM |
| Buzzer | `robot_hat/modules/buzzer.py` | Trivial addition later via MCU PWM |
| Joystick | `robot_hat/modules/joystick.py` | ADC-based; trivial addition later |
| ADXL345 accelerometer | `robot_hat/modules/adxl345.py` | Not present on PiCar-X; future if needed |
| Sound sensor | `robot_hat/modules/sound.py` | ADC microphone; low priority |
| Vision (OpenCV/TFLite) | `vilib/` | CGo dependency; satellite repo per GoRAI convention |
| Remote control server | `sunfounder_controller/` | GoRAI dashboard replaces this |
| I2S audio setup | `i2samp.sh` | Device tree config, not a Go concern |

---

## 8. Implementation Phases

### Phase 1: Robot HAT MCU Driver Module
- Create `gorai-driver-robothat` repo
- `mcu.go` -- I2C communication, PWM, ADC
- `board.go` -- Board version detection, pin maps
- `calibration.go` -- File-based calibration store (fileDB compat)
- Register MCU as a GoRAI component (`"i2c_device", "sunfounder-robot-hat"`)
- Tests: mock I2C bus, verify register writes match Python behavior
- Validation on hardware: `i2cdetect` confirms 0x14, read/write PWM registers

### Phase 2: HC-SR04 Driver Module
- Create `gorai-driver-hcsr04` repo
- `hcsr04.go` -- GPIO time-of-flight driver
- `component.go` -- GoRAI `sensor.RangeSensor` wrapper with `init()` registration
- Tests: mock GPIO, verify timing calculations
- Validation: read distance on real HC-SR04

### Phase 3: PiCar-X Component Module
- Create `gorai-picarx` repo (depends on `gorai-driver-robothat`, `gorai-driver-hcsr04`)
- `servo/servo.go` -- PWM servo (depends on MCU via `resource.Dependencies`)
- `motor/motor.go` -- DC motor with speed mapping (depends on MCU + GPIO)
- `base/base.go` -- Ackermann-differential (depends on motors + steering servo)
- `grayscale/sensor.go` -- ADC line sensor (depends on MCU)
- `picarx.go` -- Convenience blank import that registers all PiCar-X components
- Tests: verify speed mapping formula matches Python `int(speed/2) + 50`
- Validation on hardware: move servos, drive motors, read grayscale

### Phase 4: Integration + Example
- Create example robot project (from `gorai-robot-template`)
- `main.go` with `_ "github.com/emergingrobotics/gorai-picarx"` import
- `robot.json` -- Complete PiCar-X RDL (the one in section 5)
- Build for ARM64, deploy to Pi, run with `gorai run robot.json`
- Validate: all servos, motors, sensors, camera, dashboard working
- Submit component entries to the GoRAI registry JSON

### Phase 5: Behaviors
- Line-following service (uses grayscale sensor + base)
- Obstacle avoidance service (uses ultrasonic sensor + base)
- Camera tracking service (uses camera + pan/tilt servos)
- These compose existing components via RDL services -- no new drivers needed
- Could live in `gorai-picarx` or in the user's robot project
