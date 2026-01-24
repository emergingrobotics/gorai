# Gorai HAL Migration: Branch Differences

This document describes the changes made to implement the Hardware Abstraction Layer (HAL) and provides guidance for migrating the `basic-robot-modules-luca-test1` branch to use HAL.

**This document is temporary and should be deleted once the changes are understood.**

---

## High-Level Summary (For Humans)

### What Changed

We migrated two hardware components from direct hardware access to using a Hardware Abstraction Layer (HAL):

1. **PWM gpiod component** (`components/pwm/gpiod/`)
   - Previously opened GPIO chip directly via gpiocdev
   - Now gets GPIO access through HAL dependency injection
   - Pin configuration is now flexible: accepts `17`, `"GPIO17"`, `"PIN12"`, etc.
   - Removed the `chip` config field (HAL handles chip selection)

2. **I2C Bridge component** (`components/link/i2c_bridge/`)
   - Previously opened `/dev/i2c-N` device file directly
   - Now gets I2C bus through HAL dependency injection
   - Changed from `device: "/dev/i2c-1"` to `bus: 1` in config
   - Legacy device path format still works for backward compatibility

### Why This Matters

- **Board-agnostic code**: Components don't need to know which board they're running on
- **Flexible pin references**: Users can use GPIO numbers, physical pin numbers, or function names
- **Centralized hardware management**: HAL manages all hardware resources and cleans them up properly
- **Easier testing**: HAL can be mocked for unit tests

### The Key Pattern

Components now get HAL from dependencies instead of accessing hardware directly:

```go
// OLD WAY (direct access)
chip, _ := gpiocdev.NewChip("gpiochip4")
line, _ := chip.RequestLine(18, gpiocdev.AsOutput())

// NEW WAY (via HAL)
halAny, _ := deps.Get("hal")
h := halAny.(hal.HAL)
gpioDriver, _ := h.GPIO()
gpioPin, _ := gpioDriver.Pin(18)
```

### Files Changed

| File | Change |
|------|--------|
| `components/pwm/gpiod/config.go` | Pin field is now `any` type, removed Chip field |
| `components/pwm/gpiod/pwm.go` | Uses HAL for GPIO access |
| `components/pwm/pwm.go` | Properties struct: Chip -> Board |
| `components/pwm/fake/fake.go` | Updated to match interface changes |
| `components/link/i2c_bridge/config.go` | Device -> Bus field |
| `components/link/i2c_bridge/bridge.go` | Uses HAL for I2C access |
| `components/link/i2c_bridge/i2c.go` | **DELETED** (replaced by HAL) |
| `pkg/config/config.go` | Added PlatformConfig types |
| `pkg/robot/robot.go` | HAL initialization and dependency injection |

---

## Detailed Technical Explanation (For AI/Developer Reference)

### 1. HAL Interface (`driver/hal/hal.go`)

The HAL interface provides access to hardware peripherals:

```go
type HAL interface {
    Name() string
    Board() Board
    GPIO() (gpio.Driver, error)
    I2C(bus int) (i2c.Bus, error)
    SPI(bus int) (spi.Bus, error)
    PWM(chip int) (pwm.Chip, error)
    SoftwarePWM(pin int) (pwm.Channel, error)
    Serial(path string, config serial.Config) (serial.Port, error)
    ResolvePin(ref PinRef) (int, error)
    ResolvePinFromAny(v any) (int, error)
    Close(ctx context.Context) error
}
```

Key methods:
- `GPIO()` - Returns lazily-initialized GPIO driver
- `I2C(bus int)` - Returns cached I2C bus by number
- `ResolvePinFromAny(v any)` - Parses pin from config (int, float64, string)

### 2. Dependency Injection Pattern

Components receive HAL through the `registry.Dependencies` interface:

```go
// In pkg/robot/robot.go
type robotDeps struct {
    deps map[string]any
}

func (d *robotDeps) Get(name string) (any, error) {
    if v, ok := d.deps[name]; ok {
        return v, nil
    }
    return nil, fmt.Errorf("dependency %q not found", name)
}
```

HAL is injected when starting components:

```go
// In startRegistryComponent()
deps := &robotDeps{deps: make(map[string]any)}
if r.hal != nil {
    deps.deps["hal"] = r.hal
}
component, err := ctor(ctx, deps, conf)
```

### 3. PWM gpiod Migration Details

#### Config Changes (`components/pwm/gpiod/config.go`)

**Before:**
```go
type Config struct {
    Chip        string  `json:"chip"`       // e.g., "gpiochip4"
    Pin         int     `json:"pin"`        // GPIO number only
    FrequencyHz float64 `json:"frequency_hz"`
    // ...
}
```

**After:**
```go
type Config struct {
    // Pin accepts multiple formats:
    // - Integer: 17 (GPIO number)
    // - String: "GPIO17", "PIN12", "PWM0", "18"
    Pin any `json:"pin"`
    FrequencyHz float64 `json:"frequency_hz"`
    // Removed: Chip field (HAL handles this)
    // ...
}
```

#### Component Changes (`components/pwm/gpiod/pwm.go`)

**Struct changes:**
```go
// Before
type PWM struct {
    chip *gpiocdev.Chip
    line *gpiocdev.Line
    // ...
}

// After
type PWM struct {
    hal       hal.HAL
    gpioPin   gpio.Pin
    pinNumber int
    // ...
}
```

**Initialization changes:**
```go
// Before
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
    chip, err := gpiocdev.NewChip(cfg.Chip)
    line, err := chip.RequestLine(cfg.Pin, gpiocdev.AsOutput())
    // ...
}

// After
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
    // Get HAL from dependencies
    halAny, err := deps.Get("hal")
    h, ok := halAny.(hal.HAL)

    // Resolve pin using HAL (handles "GPIO17", "PIN12", 17, etc.)
    pinNumber, err := h.ResolvePinFromAny(cfg.Pin)

    // Get GPIO driver from HAL
    gpioDriver, err := h.GPIO()
    gpioPin, err := gpioDriver.Pin(pinNumber)
    gpioPin.SetDirection(ctx, gpio.Output)
    // ...
}
```

**PWM loop changes:**
```go
// Before
p.line.SetValue(highVal)

// After
p.gpioPin.Write(ctx, highVal)
```

**Close changes:**
```go
// Before
if p.line != nil {
    p.line.Close()
}
if p.chip != nil {
    p.chip.Close()
}

// After
if p.gpioPin != nil {
    p.gpioPin.Write(ctx, false)
}
// Note: GPIO driver cleanup is handled by HAL.Close()
```

### 4. I2C Bridge Migration Details

#### Config Changes (`components/link/i2c_bridge/config.go`)

**Before:**
```go
type Config struct {
    Device  string         `json:"device"`   // e.g., "/dev/i2c-1"
    Devices []DeviceConfig `json:"devices"`
}
```

**After:**
```go
type Config struct {
    Bus     int            `json:"bus"`      // e.g., 1 for /dev/i2c-1
    Devices []DeviceConfig `json:"devices"`
}
```

Legacy support added in `NewConfigFromResource`:
```go
// Legacy support: parse device path and extract bus number
if device, ok := conf.Attributes["device"].(string); ok && device != "" {
    busID, err := ParseBusID(device)  // "/dev/i2c-1" -> 1
    if err == nil {
        cfg.Bus = busID
    }
}
```

#### Component Changes (`components/link/i2c_bridge/bridge.go`)

**Struct changes:**
```go
// Before
type Bridge struct {
    bus *I2CBus  // local i2c.go implementation
    // ...
}

// After
type Bridge struct {
    hal hal.HAL
    bus i2c.Bus  // HAL-provided interface
    // ...
}
```

**Initialization changes:**
```go
// Before
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
    bus, err := NewI2CBus(cfg.Device)  // Opens /dev/i2c-N directly
    // ...
}

// After
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
    halAny, err := deps.Get("hal")
    h, ok := halAny.(hal.HAL)
    bus, err := h.I2C(cfg.Bus)  // HAL manages the bus
    // ...
}
```

**Device read changes:**
```go
// Before (using local I2CBus)
data, err := b.bus.ReadRegister(dev.Address, byte(dev.ReadRegister), dev.ReadLength)

// After (using HAL's i2c.Device interface)
device := b.bus.Device(uint16(dev.Address))
data, err := device.ReadReg(ctx, byte(dev.ReadRegister), dev.ReadLength)
```

#### Deleted File

`components/link/i2c_bridge/i2c.go` was **deleted entirely**. It contained:
- `I2CBus` struct with file descriptor
- Direct syscall-based I2C operations
- `NewI2CBus()`, `Close()`, `ReadRegister()`, `WriteRegister()`, `Scan()` methods

All this functionality is now provided by the HAL's I2C interface.

### 5. Robot Runtime Changes (`pkg/robot/robot.go`)

**New HAL field:**
```go
type Robot struct {
    // ...
    hal hal.HAL  // Hardware Abstraction Layer
    // ...
}
```

**HAL initialization:**
```go
func (r *Robot) initHAL(ctx context.Context) error {
    halConfig := hal.Config{}

    if r.cfg.Platform != nil {
        halConfig.Board = r.cfg.Platform.Board
        if r.cfg.Platform.GPIO != nil {
            halConfig.GPIO.Chip = r.cfg.Platform.GPIO.Chip
        }
        if r.cfg.Platform.I2C != nil {
            halConfig.I2C.Buses = r.cfg.Platform.I2C.Buses
        }
        // ... similar for SPI, PWM
    }

    h, err := hal.New(halConfig)
    r.hal = h
    return nil
}
```

**HAL cleanup in Stop():**
```go
func (r *Robot) Stop(ctx context.Context) error {
    // ... stop components ...

    // Close HAL (releases GPIO, I2C, etc.)
    if r.hal != nil {
        if err := r.hal.Close(ctx); err != nil {
            r.logger.Warn("Error closing HAL", "error", err)
        }
    }
    // ...
}
```

### 6. Platform Config Types (`pkg/config/config.go`)

New types added to RDL:

```go
type PlatformConfig struct {
    Board string               `json:"board,omitempty"`
    GPIO  *PlatformGPIOConfig  `json:"gpio,omitempty"`
    I2C   *PlatformI2CConfig   `json:"i2c,omitempty"`
    SPI   *PlatformSPIConfig   `json:"spi,omitempty"`
    PWM   *PlatformPWMConfig   `json:"pwm,omitempty"`
}

type PlatformGPIOConfig struct {
    Chip int `json:"chip,omitempty"`  // e.g., 4 for gpiochip4
}

type PlatformI2CConfig struct {
    Buses []int `json:"buses,omitempty"`  // e.g., [1] for /dev/i2c-1
}
```

Example RDL with platform config:
```json
{
  "robot": {"name": "my-robot"},
  "platform": {
    "board": "raspberrypi5",
    "gpio": {"chip": 4},
    "i2c": {"buses": [1]}
  },
  "components": [...]
}
```

### 7. Test Updates

Tests were updated to use new config structures:

**PWM tests** - Removed `Chip` field, use `Pin` as `any`:
```go
// Before
config: Config{Chip: "gpiochip4", Pin: 18, ...}

// After
config: Config{Pin: 18, ...}
config: Config{Pin: "GPIO17", ...}
```

**I2C Bridge tests** - Changed `Device` to `Bus`:
```go
// Before
attrs := map[string]any{"device": "/dev/i2c-1", ...}

// After
attrs := map[string]any{"bus": float64(1), ...}
```

---

## Migration Checklist for basic-robot-modules-luca-test1

To migrate a component from direct hardware access to HAL:

1. **Update imports:**
   ```go
   import (
       "github.com/gorai/gorai/driver/hal"
       "github.com/gorai/gorai/driver/gpio"  // or i2c, spi, etc.
   )
   ```

2. **Get HAL from dependencies in New():**
   ```go
   halAny, err := deps.Get("hal")
   if err != nil {
       return nil, fmt.Errorf("HAL not available: %w", err)
   }
   h, ok := halAny.(hal.HAL)
   ```

3. **Use HAL methods for hardware access:**
   - GPIO: `h.GPIO()` -> `driver.Pin(n)` -> `pin.Write(ctx, val)`
   - I2C: `h.I2C(bus)` -> `bus.Device(addr)` -> `device.ReadReg(ctx, reg, len)`
   - Pin resolution: `h.ResolvePinFromAny(cfg.Pin)`

4. **Update config struct:**
   - Remove board/chip-specific fields
   - Change pin fields to `any` type for flexible input

5. **Update cleanup:**
   - Don't close drivers directly; HAL manages lifecycle
   - Just set pins to safe state (e.g., LOW)

6. **Update tests:**
   - Remove chip/device path from test configs
   - Use bus numbers instead of device paths

---

## Questions?

If you need clarification on any of these changes, the relevant source files are:
- `driver/hal/hal.go` - HAL interface definition
- `driver/gpio/gpio.go` - GPIO driver interface
- `driver/i2c/i2c.go` - I2C bus/device interfaces
- `docs/hardware-abstraction.md` - Full HAL design documentation
