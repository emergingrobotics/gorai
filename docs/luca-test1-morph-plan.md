# Migration Plan: luca-test1 Components to HAL Architecture

This document provides a detailed, step-by-step plan to migrate components from the `basic-robot-modules-luca-test1` branch to use the new Hardware Abstraction Layer (HAL). An AI agent can execute these changes unsupervised.

## Overview

### Components Requiring Migration

| Component | Current Hardware Access | HAL Migration Scope |
|-----------|------------------------|---------------------|
| `components/pwm/gpiod/` | Direct gpiocdev (GPIO chip path) | Use `HAL.GPIO()` + `HAL.ResolvePin()` |
| `components/link/i2c_bridge/` | Direct `/dev/i2c-*` access | Use `HAL.I2C()` |
| `components/input/keyboard/` | Direct `/dev/input/event*` | Out of scope (evdev is not GPIO/I2C/SPI) |
| `components/camera/v4l2/` | Direct V4L2 via webcam lib | Out of scope (V4L2 is not GPIO/I2C/SPI) |
| `driver/camera/v4l2/` | Direct V4L2 via webcam lib | Out of scope (V4L2 is not GPIO/I2C/SPI) |

### Migration Priority

1. **PWM gpiod** - High priority (directly uses GPIO, benefits most from HAL)
2. **I2C Bridge** - High priority (directly uses I2C bus)
3. **Keyboard** - Skip (evdev input devices are not HAL scope)
4. **Camera V4L2** - Skip (V4L2 is not HAL scope)

---

## Task 1: Migrate PWM gpiod Component

### Current State Analysis

**File:** `components/pwm/gpiod/pwm.go`

**Current hardware access pattern:**
```go
// Direct gpiocdev usage
import "github.com/warthog618/go-gpiocdev"

// Opens chip directly by path
chip, err := gpiocdev.NewChip(cfg.Chip)  // e.g., "/dev/gpiochip4"

// Requests line directly by pin number
line, err := chip.RequestLine(cfg.Pin, gpiocdev.AsOutput(0))
```

**Current config (`components/pwm/gpiod/config.go`):**
```go
type Config struct {
    Chip           string  // "/dev/gpiochip4" - board-specific!
    Pin            int     // BCM GPIO number
    FrequencyHz    float64
    MinPulseUs     float64
    MaxPulseUs     float64
    InitialPulseUs float64
    Invert         bool
}
```

### Target State

The component should:
1. Obtain HAL from dependencies
2. Use HAL to resolve pin references (supporting "GPIO17", "PIN12", "PWM0", etc.)
3. Use HAL.GPIO() to get GPIO driver
4. Use the GPIO driver to get pin objects

### Step-by-Step Migration

#### Step 1.1: Update Config struct

**File:** `components/pwm/gpiod/config.go`

Replace the `Chip` and `Pin` fields with a single `pin` field that accepts multiple formats:

```go
// BEFORE
type Config struct {
    Chip           string  `json:"chip"`
    Pin            int     `json:"pin"`
    // ... other fields
}

// AFTER
type Config struct {
    // Pin accepts multiple formats:
    // - Integer: 17 (GPIO number)
    // - String: "GPIO17", "PIN12", "PWM0", "18"
    // The HAL will resolve this to the correct GPIO number for the board.
    Pin            any     `json:"pin"`
    FrequencyHz    float64 `json:"frequency_hz"`
    MinPulseUs     float64 `json:"min_pulse_us"`
    MaxPulseUs     float64 `json:"max_pulse_us"`
    InitialPulseUs float64 `json:"initial_pulse_us"`
    Invert         bool    `json:"invert"`
}
```

#### Step 1.2: Update ParseConfig function

**File:** `components/pwm/gpiod/config.go`

```go
func ParseConfig(conf registry.Config) (Config, error) {
    cfg := Config{
        FrequencyHz:    50.0,
        MinPulseUs:     1000.0,
        MaxPulseUs:     2000.0,
        InitialPulseUs: 1500.0,
        Invert:         false,
    }

    // Pin is required - can be int, float64, or string
    if pin, ok := conf["pin"]; ok {
        cfg.Pin = pin
    } else {
        return cfg, fmt.Errorf("pin is required")
    }

    // Optional fields (unchanged)
    if freq, ok := conf["frequency_hz"].(float64); ok {
        cfg.FrequencyHz = freq
    }
    if minPulse, ok := conf["min_pulse_us"].(float64); ok {
        cfg.MinPulseUs = minPulse
    }
    if maxPulse, ok := conf["max_pulse_us"].(float64); ok {
        cfg.MaxPulseUs = maxPulse
    }
    if initialPulse, ok := conf["initial_pulse_us"].(float64); ok {
        cfg.InitialPulseUs = initialPulse
    }
    if invert, ok := conf["invert"].(bool); ok {
        cfg.Invert = invert
    }

    return cfg, nil
}
```

#### Step 1.3: Update Validate function

**File:** `components/pwm/gpiod/config.go`

Remove chip validation, update pin validation:

```go
func (c Config) Validate() error {
    // Pin validation is deferred to HAL resolution
    if c.Pin == nil {
        return fmt.Errorf("pin is required")
    }

    if c.FrequencyHz <= 0 {
        return fmt.Errorf("frequency_hz must be positive, got %f", c.FrequencyHz)
    }
    if c.FrequencyHz > 1000 {
        return fmt.Errorf("frequency_hz must be <= 1000 Hz for software PWM, got %f", c.FrequencyHz)
    }
    // ... rest of validation unchanged
    return nil
}
```

#### Step 1.4: Update PWM struct

**File:** `components/pwm/gpiod/pwm.go`

```go
// BEFORE
import (
    "github.com/warthog618/go-gpiocdev"
)

type PWM struct {
    // ...
    chip *gpiocdev.Chip
    line *gpiocdev.Line
    // ...
}

// AFTER
import (
    "github.com/gorai/gorai/driver/gpio"
    "github.com/gorai/gorai/driver/hal"
)

type PWM struct {
    name   resource.Name
    config Config
    logger *slog.Logger

    hal       hal.HAL
    gpioPin   gpio.Pin
    pinNumber int

    mu       sync.RWMutex
    pulseUs  float64
    enabled  bool
    periodNs int64

    stopCh chan struct{}
    doneCh chan struct{}

    cmdCount   atomic.Uint64
    cycleCount atomic.Uint64
}
```

#### Step 1.5: Update New function

**File:** `components/pwm/gpiod/pwm.go`

```go
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
    nameStr, _ := conf["name"].(string)
    name := resource.NewComponentName("gorai", "pwm", nameStr)

    cfg, err := ParseConfig(conf)
    if err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    // Get HAL from dependencies
    halAny, err := deps.Get("hal")
    if err != nil {
        return nil, fmt.Errorf("HAL not available: %w (ensure platform section is configured)", err)
    }
    h, ok := halAny.(hal.HAL)
    if !ok {
        return nil, fmt.Errorf("invalid HAL type: %T", halAny)
    }

    // Resolve pin using HAL
    pinNumber, err := h.ResolvePinFromAny(cfg.Pin)
    if err != nil {
        return nil, fmt.Errorf("failed to resolve pin %v: %w", cfg.Pin, err)
    }

    // Get GPIO driver from HAL
    gpioDriver, err := h.GPIO()
    if err != nil {
        return nil, fmt.Errorf("failed to get GPIO driver: %w", err)
    }

    // Get the pin
    gpioPin, err := gpioDriver.Pin(pinNumber)
    if err != nil {
        return nil, fmt.Errorf("failed to get GPIO pin %d: %w", pinNumber, err)
    }

    // Set pin as output
    if err := gpioPin.SetDirection(ctx, gpio.Output); err != nil {
        return nil, fmt.Errorf("failed to set pin %d as output: %w", pinNumber, err)
    }

    p := &PWM{
        name:      name,
        config:    cfg,
        logger:    slog.Default().With("component", nameStr),
        hal:       h,
        gpioPin:   gpioPin,
        pinNumber: pinNumber,
        pulseUs:   cfg.InitialPulseUs,
        periodNs:  cfg.PeriodNs(),
        stopCh:    make(chan struct{}),
        doneCh:    make(chan struct{}),
    }

    // Start PWM goroutine (initially disabled)
    go p.pwmLoop()

    p.logger.Info("PWM component initialized",
        "board", h.Board(),
        "pin", pinNumber,
        "pin_ref", cfg.Pin,
        "frequency_hz", cfg.FrequencyHz,
        "min_pulse_us", cfg.MinPulseUs,
        "max_pulse_us", cfg.MaxPulseUs,
    )

    return p, nil
}
```

#### Step 1.6: Update pwmLoop function

**File:** `components/pwm/gpiod/pwm.go`

Replace `p.line.SetValue()` with `p.gpioPin.Write()`:

```go
func (p *PWM) pwmLoop() {
    defer close(p.doneCh)

    for {
        select {
        case <-p.stopCh:
            return
        default:
        }

        p.mu.RLock()
        enabled := p.enabled
        pulseUs := p.pulseUs
        invert := p.config.Invert
        p.mu.RUnlock()

        if !enabled {
            time.Sleep(10 * time.Millisecond)
            continue
        }

        pulseNs := int64(pulseUs * 1000)

        highVal := true
        lowVal := false
        if invert {
            highVal = false
            lowVal = true
        }

        cycleStart := time.Now()

        // Set HIGH using gpio.Pin interface
        ctx := context.Background()
        p.gpioPin.Write(ctx, highVal)

        // Wait for pulse duration
        p.precisionSleep(pulseNs)

        // Set LOW
        p.gpioPin.Write(ctx, lowVal)

        // Wait for remaining period
        elapsed := time.Since(cycleStart).Nanoseconds()
        remaining := p.periodNs - elapsed
        if remaining > 0 {
            time.Sleep(time.Duration(remaining))
        }

        p.cycleCount.Add(1)
    }
}
```

#### Step 1.7: Update Close function

**File:** `components/pwm/gpiod/pwm.go`

```go
func (p *PWM) Close(ctx context.Context) error {
    p.logger.Info("Closing PWM component")

    // Stop PWM loop
    close(p.stopCh)

    // Wait for loop to finish with timeout
    select {
    case <-p.doneCh:
    case <-time.After(time.Second):
        p.logger.Warn("PWM loop did not stop cleanly")
    }

    // Set pin LOW
    if p.gpioPin != nil {
        p.gpioPin.Write(ctx, false)
    }

    // Note: GPIO driver cleanup is handled by HAL.Close()
    // Don't close the pin here as other components may share the GPIO driver

    return nil
}
```

#### Step 1.8: Update Disable function

**File:** `components/pwm/gpiod/pwm.go`

```go
func (p *PWM) Disable(ctx context.Context) error {
    p.mu.Lock()
    defer p.mu.Unlock()

    if !p.enabled {
        return nil
    }

    p.enabled = false

    // Set pin LOW immediately
    if p.gpioPin != nil {
        p.gpioPin.Write(ctx, false)
    }

    p.logger.Info("PWM disabled")
    return nil
}
```

#### Step 1.9: Update Properties function

**File:** `components/pwm/gpiod/pwm.go`

Remove `Chip` from properties, use board info instead:

```go
func (p *PWM) Properties(ctx context.Context) (pwm.Properties, error) {
    return pwm.Properties{
        FrequencyHz: p.config.FrequencyHz,
        MinPulseUs:  p.config.MinPulseUs,
        MaxPulseUs:  p.config.MaxPulseUs,
        Pin:         p.pinNumber,
        Board:       string(p.hal.Board()),
        Inverted:    p.config.Invert,
    }, nil
}
```

#### Step 1.10: Update imports

**File:** `components/pwm/gpiod/pwm.go`

Remove:
```go
"github.com/warthog618/go-gpiocdev"
```

Add:
```go
"github.com/gorai/gorai/driver/gpio"
"github.com/gorai/gorai/driver/hal"
```

#### Step 1.11: Update pwm.Properties struct

**File:** `components/pwm/pwm.go`

```go
// BEFORE
type Properties struct {
    FrequencyHz float64
    MinPulseUs  float64
    MaxPulseUs  float64
    Pin         int
    Chip        string
    Inverted    bool
}

// AFTER
type Properties struct {
    FrequencyHz float64
    MinPulseUs  float64
    MaxPulseUs  float64
    Pin         int
    Board       string  // Board name from HAL (e.g., "raspberrypi5")
    Inverted    bool
}
```

---

## Task 2: Migrate I2C Bridge Component

### Current State Analysis

**Files:**
- `components/link/i2c_bridge/bridge.go` - Main component
- `components/link/i2c_bridge/i2c.go` - Direct I2C bus access
- `components/link/i2c_bridge/config.go` - Configuration

**Current hardware access pattern:**
```go
// Direct I2C device access
file, err := os.OpenFile(devicePath, os.O_RDWR, 0)  // "/dev/i2c-1"

// Direct ioctl syscall
unix.Syscall(unix.SYS_IOCTL, uintptr(b.fd), I2C_SLAVE, uintptr(address))
```

**Current config:**
```go
type Config struct {
    Device  string          `json:"device"`   // "/dev/i2c-1" - board specific!
    Devices []DeviceConfig  `json:"devices"`
}
```

### Target State

The component should:
1. Obtain HAL from dependencies
2. Use HAL.I2C(bus) to get I2C bus
3. Use the i2c.Bus interface methods

### Step-by-Step Migration

#### Step 2.1: Update Config struct

**File:** `components/link/i2c_bridge/config.go`

```go
// BEFORE
type Config struct {
    Device  string         `json:"device"`   // "/dev/i2c-1"
    Devices []DeviceConfig `json:"devices"`
}

// AFTER
type Config struct {
    // Bus is the I2C bus number (e.g., 1 for /dev/i2c-1)
    // Replaces the old "device" field which was board-specific
    Bus     int            `json:"bus"`
    Devices []DeviceConfig `json:"devices"`
}
```

#### Step 2.2: Update NewConfigFromResource

**File:** `components/link/i2c_bridge/config.go`

```go
func NewConfigFromResource(conf resource.Config) (*Config, error) {
    cfg := &Config{
        Bus:     1, // Default to bus 1
        Devices: []DeviceConfig{},
    }

    // Parse bus number (new field)
    if bus, ok := conf.Attributes["bus"].(float64); ok {
        cfg.Bus = int(bus)
    } else if bus, ok := conf.Attributes["bus"].(int); ok {
        cfg.Bus = bus
    }

    // Legacy support: parse device path and extract bus number
    if device, ok := conf.Attributes["device"].(string); ok && device != "" {
        busID, err := ParseBusID(device)
        if err == nil {
            cfg.Bus = busID
        }
    }

    // Parse devices array (unchanged)
    devicesRaw, ok := conf.Attributes["devices"].([]any)
    if !ok {
        return nil, fmt.Errorf("devices configuration is required")
    }

    for i, devRaw := range devicesRaw {
        devMap, ok := devRaw.(map[string]any)
        if !ok {
            return nil, fmt.Errorf("device[%d]: invalid format", i)
        }

        dev, err := parseDeviceConfig(devMap, i)
        if err != nil {
            return nil, err
        }
        cfg.Devices = append(cfg.Devices, *dev)
    }

    return cfg, nil
}
```

#### Step 2.3: Update Validate function

**File:** `components/link/i2c_bridge/config.go`

```go
func (c *Config) Validate() error {
    if c.Bus < 0 {
        return fmt.Errorf("bus number must be non-negative")
    }

    if len(c.Devices) == 0 {
        return fmt.Errorf("at least one device is required")
    }

    // ... rest of validation unchanged
    return nil
}
```

#### Step 2.4: Delete i2c.go file

**File:** `components/link/i2c_bridge/i2c.go`

This file contains the direct I2C access code (`I2CBus` struct, `OpenBus`, `ReadRegister`, etc.).

**Action:** Delete this file entirely. The HAL's `i2c.Bus` interface provides the same functionality.

#### Step 2.5: Update Bridge struct

**File:** `components/link/i2c_bridge/bridge.go`

```go
// BEFORE
import (
    "golang.org/x/sys/unix"
)

type Bridge struct {
    // ...
    bus   *I2CBus  // Local I2CBus type
    busID int
    // ...
}

// AFTER
import (
    "github.com/gorai/gorai/driver/hal"
    "github.com/gorai/gorai/driver/i2c"
)

type Bridge struct {
    name   resource.Name
    config *Config
    logger *slog.Logger

    hal hal.HAL
    bus i2c.Bus  // HAL-provided I2C bus interface

    mu           sync.RWMutex
    state        State
    errorMsg     string
    deviceStates map[string]*DeviceState

    stopCh chan struct{}
    wg     sync.WaitGroup

    totalReads        atomic.Uint64
    totalErrors       atomic.Uint64
    messagesPublished atomic.Uint64

    onData func(deviceName string, data []byte, address uint8, register int)
}
```

#### Step 2.6: Update New function

**File:** `components/link/i2c_bridge/bridge.go`

```go
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
    resConf := resource.NewConfig(conf)

    cfg, err := NewConfigFromResource(resConf)
    if err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    if err := cfg.Validate(); err != nil {
        return nil, fmt.Errorf("invalid config: %w", err)
    }

    name := "i2c_bridge"
    if n, ok := conf["name"].(string); ok {
        name = n
    }

    // Get HAL from dependencies
    halAny, err := deps.Get("hal")
    if err != nil {
        return nil, fmt.Errorf("HAL not available: %w (ensure platform section is configured)", err)
    }
    h, ok := halAny.(hal.HAL)
    if !ok {
        return nil, fmt.Errorf("invalid HAL type: %T", halAny)
    }

    // Get I2C bus from HAL
    bus, err := h.I2C(cfg.Bus)
    if err != nil {
        return nil, fmt.Errorf("failed to open I2C bus %d: %w", cfg.Bus, err)
    }

    b := &Bridge{
        name:         resource.NewComponentName("gorai", "link", name),
        config:       cfg,
        logger:       slog.Default().With("component", "i2c_bridge", "name", name),
        hal:          h,
        bus:          bus,
        state:        StateClosed,
        deviceStates: make(map[string]*DeviceState),
        stopCh:       make(chan struct{}),
    }

    // Initialize device states
    for _, dev := range cfg.Devices {
        b.deviceStates[dev.Name] = &DeviceState{
            Name:    dev.Name,
            Address: dev.Address,
            Enabled: dev.Enabled,
        }
    }

    // Start the bridge
    if err := b.open(ctx); err != nil {
        return nil, fmt.Errorf("failed to open I2C bridge: %w", err)
    }

    return b, nil
}
```

#### Step 2.7: Update open function

**File:** `components/link/i2c_bridge/bridge.go`

```go
func (b *Bridge) open(ctx context.Context) error {
    b.mu.Lock()
    defer b.mu.Unlock()

    b.state = StateOpening

    // Probe devices using HAL's I2C bus
    for _, dev := range b.config.Devices {
        state := b.deviceStates[dev.Name]

        // Try to read one byte to probe device
        device := b.bus.Device(uint16(dev.Address))
        _, err := device.Read(ctx, 1)
        if err == nil {
            state.Connected = true
            b.logger.Info("device found", "name", dev.Name, "address", fmt.Sprintf("0x%02X", dev.Address))
        } else {
            state.Connected = false
            b.logger.Warn("device not found", "name", dev.Name, "address", fmt.Sprintf("0x%02X", dev.Address))
        }
    }

    b.state = StateRunning
    b.errorMsg = ""

    // Start polling goroutines for each enabled device
    for i := range b.config.Devices {
        dev := &b.config.Devices[i]
        if dev.Enabled {
            b.wg.Add(1)
            go b.pollDevice(ctx, dev)
        }
    }

    b.logger.Info("I2C bridge started", "bus", b.config.Bus, "board", b.hal.Board())
    return nil
}
```

#### Step 2.8: Update pollDevice function

**File:** `components/link/i2c_bridge/bridge.go`

```go
func (b *Bridge) pollDevice(ctx context.Context, dev *DeviceConfig) {
    defer b.wg.Done()

    interval := time.Duration(float64(time.Second) / dev.PollRateHz)
    ticker := time.NewTicker(interval)
    defer ticker.Stop()

    // Get device handle from bus
    device := b.bus.Device(uint16(dev.Address))

    var lastRead time.Time

    for {
        select {
        case <-b.stopCh:
            return

        case <-ticker.C:
            b.mu.RLock()
            state := b.deviceStates[dev.Name]
            enabled := state.Enabled
            b.mu.RUnlock()

            if !enabled {
                continue
            }

            // Read from device using HAL's i2c.Device interface
            var data []byte
            var err error

            if dev.ReadRegister >= 0 {
                // Register read
                data, err = device.ReadReg(ctx, byte(dev.ReadRegister), dev.ReadLength)
            } else {
                // Raw read
                data, err = device.Read(ctx, dev.ReadLength)
            }

            b.mu.Lock()
            if err != nil {
                state.ReadsFailed++
                state.Connected = false
                b.totalErrors.Add(1)
                b.mu.Unlock()

                b.logger.Debug("read failed", "device", dev.Name, "error", err)
                continue
            }

            state.ReadsSuccessful++
            state.Connected = true
            state.LastReadTime = time.Now()

            if !lastRead.IsZero() {
                elapsed := time.Since(lastRead)
                state.ActualRateHz = float64(time.Second) / float64(elapsed)
            }
            lastRead = time.Now()
            b.mu.Unlock()

            b.totalReads.Add(1)

            if b.onData != nil {
                b.onData(dev.Name, data, dev.Address, dev.ReadRegister)
                b.messagesPublished.Add(1)
            }
        }
    }
}
```

#### Step 2.9: Update probe_bus command

**File:** `components/link/i2c_bridge/bridge.go`

In `DoCommand`, update the `probe_bus` case:

```go
case "probe_bus":
    ctx := context.Background()
    found := []string{}
    for addr := uint8(0x08); addr <= 0x77; addr++ {
        device := b.bus.Device(uint16(addr))
        _, err := device.Read(ctx, 1)
        if err == nil {
            found = append(found, fmt.Sprintf("0x%02X", addr))
        }
    }
    return map[string]any{"devices": found}, nil
```

#### Step 2.10: Update read_once command

**File:** `components/link/i2c_bridge/bridge.go`

```go
case "read_once":
    deviceName, _ := cmd["device"].(string)

    b.mu.RLock()
    var dev *DeviceConfig
    for i := range b.config.Devices {
        if b.config.Devices[i].Name == deviceName {
            dev = &b.config.Devices[i]
            break
        }
    }
    b.mu.RUnlock()

    if dev == nil {
        return nil, fmt.Errorf("device %q not found", deviceName)
    }

    device := b.bus.Device(uint16(dev.Address))
    var data []byte
    var err error

    if dev.ReadRegister >= 0 {
        data, err = device.ReadReg(ctx, byte(dev.ReadRegister), dev.ReadLength)
    } else {
        data, err = device.Read(ctx, dev.ReadLength)
    }

    if err != nil {
        return nil, fmt.Errorf("read failed: %w", err)
    }

    return map[string]any{
        "device": deviceName,
        "data":   data,
        "length": len(data),
    }, nil
```

#### Step 2.11: Update Close function

**File:** `components/link/i2c_bridge/bridge.go`

```go
func (b *Bridge) Close(ctx context.Context) error {
    b.mu.Lock()
    if b.state == StateClosed {
        b.mu.Unlock()
        return nil
    }
    b.state = StateClosed
    b.mu.Unlock()

    // Signal all goroutines to stop
    close(b.stopCh)

    // Wait for all goroutines to finish
    b.wg.Wait()

    // Note: I2C bus cleanup is handled by HAL.Close()
    // Don't close the bus here as other components may share it

    b.logger.Info("I2C bridge closed")
    return nil
}
```

#### Step 2.12: Update GetBusID function

**File:** `components/link/i2c_bridge/bridge.go`

```go
func (b *Bridge) GetBusID() int {
    return b.config.Bus
}
```

#### Step 2.13: Remove ParseBusID (optional)

The `ParseBusID` function in `config.go` can be kept for legacy config support, or removed if not needed.

---

## Task 3: Update RDL Examples

### Step 3.1: Update PWM example configs

**Before (board-specific):**
```json
{
  "name": "servo1",
  "type": "pwm/gpiod",
  "config": {
    "chip": "/dev/gpiochip4",
    "pin": 18,
    "frequency_hz": 50
  }
}
```

**After (board-agnostic):**
```json
{
  "name": "servo1",
  "type": "pwm/gpiod",
  "config": {
    "pin": "GPIO18",
    "frequency_hz": 50
  }
}
```

Alternative pin formats:
```json
{"pin": 18}           // GPIO number
{"pin": "18"}         // GPIO number as string
{"pin": "GPIO18"}     // Named GPIO
{"pin": "PIN12"}      // Physical pin number
{"pin": "PWM0"}       // Named function
```

### Step 3.2: Update I2C example configs

**Before (board-specific):**
```json
{
  "name": "sensors",
  "type": "link/i2c_bridge",
  "config": {
    "device": "/dev/i2c-1",
    "devices": [...]
  }
}
```

**After (board-agnostic):**
```json
{
  "name": "sensors",
  "type": "link/i2c_bridge",
  "config": {
    "bus": 1,
    "devices": [...]
  }
}
```

### Step 3.3: Update platform section documentation

Add/update platform section in example RDL files:

```json
{
  "name": "my-robot",
  "platform": {
    "board": "auto",
    "gpio": {
      "chip": 0
    },
    "i2c": {
      "buses": [1]
    }
  },
  "components": [...]
}
```

---

## Task 4: Ensure HAL is Provided to Components

### Step 4.1: Update runtime to inject HAL

The runtime must create HAL from the platform config and inject it into the dependencies map.

**File to update:** `pkg/runtime/runtime.go` (or wherever component initialization occurs)

```go
import (
    "github.com/gorai/gorai/driver/hal"
)

func (r *Runtime) initializeComponents(ctx context.Context) error {
    // Create HAL from platform config
    halConfig := hal.Config{
        Board: r.config.Platform.Board,
        GPIO: hal.GPIOConfig{
            Chip: r.config.Platform.GPIO.Chip,
        },
        I2C: hal.I2CConfig{
            Buses: r.config.Platform.I2C.Buses,
        },
    }

    h, err := hal.New(halConfig)
    if err != nil {
        return fmt.Errorf("failed to create HAL: %w", err)
    }

    // Register HAL in dependencies
    r.deps.Set("hal", h)

    // Store for cleanup
    r.hal = h

    // Continue with component initialization...
}

func (r *Runtime) Close(ctx context.Context) error {
    // Close HAL after all components
    if r.hal != nil {
        if err := r.hal.Close(ctx); err != nil {
            r.logger.Warn("failed to close HAL", "error", err)
        }
    }
    return nil
}
```

### Step 4.2: Update RDL config types

**File:** `pkg/config/rdl.go`

```go
type RobotDefinition struct {
    Name        string                 `json:"name"`
    Description string                 `json:"description,omitempty"`
    Platform    PlatformConfig         `json:"platform,omitempty"`
    NATS        NATSConfig             `json:"nats,omitempty"`
    Components  []ComponentConfig      `json:"components"`
    Behaviors   []BehaviorConfig       `json:"behaviors,omitempty"`
    Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type PlatformConfig struct {
    Board string         `json:"board,omitempty"` // "auto", "raspberrypi5", "orangepi5b", etc.
    GPIO  GPIOConfig     `json:"gpio,omitempty"`
    I2C   I2CConfig      `json:"i2c,omitempty"`
    SPI   SPIConfig      `json:"spi,omitempty"`
    PWM   PWMConfig      `json:"pwm,omitempty"`
}

type GPIOConfig struct {
    Chip int `json:"chip,omitempty"` // GPIO chip number (0 for auto)
}

type I2CConfig struct {
    Buses []int `json:"buses,omitempty"` // Enabled I2C bus numbers
}

type SPIConfig struct {
    Buses []int `json:"buses,omitempty"` // Enabled SPI bus numbers
}

type PWMConfig struct {
    Chips []int `json:"chips,omitempty"` // Enabled PWM chip numbers
}
```

---

## Task 5: Update Tests

### Step 5.1: Update PWM tests

**File:** `components/pwm/gpiod/pwm_test.go`

Update tests to use mock HAL:

```go
func TestPWM(t *testing.T) {
    // Create mock HAL
    mockHAL := &MockHAL{
        board: hal.BoardRaspberryPi5,
        gpio:  &MockGPIODriver{},
    }
    hal.SetTestHAL(mockHAL)
    defer hal.SetTestHAL(nil)

    // Create component with new config format
    conf := registry.Config{
        "name": "test-pwm",
        "pin":  18,  // No longer need chip path
        "frequency_hz": 50.0,
    }

    deps := registry.NewDependencies()
    deps.Set("hal", mockHAL)

    comp, err := New(context.Background(), deps, conf)
    require.NoError(t, err)
    // ... rest of test
}

type MockHAL struct {
    board hal.Board
    gpio  gpio.Driver
}

func (m *MockHAL) Board() hal.Board { return m.board }
func (m *MockHAL) GPIO() (gpio.Driver, error) { return m.gpio, nil }
func (m *MockHAL) ResolvePinFromAny(v any) (int, error) {
    switch pin := v.(type) {
    case int:
        return pin, nil
    case float64:
        return int(pin), nil
    case string:
        // Simple mock resolution
        var n int
        fmt.Sscanf(pin, "GPIO%d", &n)
        return n, nil
    }
    return 0, fmt.Errorf("invalid pin")
}
// ... other interface methods
```

### Step 5.2: Update I2C Bridge tests

**File:** `components/link/i2c_bridge/bridge_test.go`

```go
func TestI2CBridge(t *testing.T) {
    // Create mock HAL with mock I2C bus
    mockBus := &MockI2CBus{}
    mockHAL := &MockHAL{
        i2cBuses: map[int]i2c.Bus{1: mockBus},
    }
    hal.SetTestHAL(mockHAL)
    defer hal.SetTestHAL(nil)

    conf := registry.Config{
        "name": "test-bridge",
        "bus":  1,  // No longer need device path
        "devices": []any{
            map[string]any{
                "name":        "sensor1",
                "address":     0x27,
                "read_length": 2,
            },
        },
    }

    deps := registry.NewDependencies()
    deps.Set("hal", mockHAL)

    comp, err := New(context.Background(), deps, conf)
    require.NoError(t, err)
    // ... rest of test
}
```

---

## Task 6: Documentation Updates

### Step 6.1: Update hardware-abstraction.md

Add migration examples and update component documentation.

### Step 6.2: Update example RDL files

Ensure all example RDL files use the new HAL-compatible format.

### Step 6.3: Update component README files

If components have individual READMEs, update configuration examples.

---

## Verification Checklist

After completing all migrations:

- [ ] `go build ./...` passes
- [ ] `go test ./...` passes
- [ ] PWM component works with `"pin": "GPIO18"` format
- [ ] PWM component works with `"pin": 18` format
- [ ] PWM component works with `"pin": "PIN12"` format
- [ ] I2C bridge works with `"bus": 1` format
- [ ] Legacy `"device": "/dev/i2c-1"` still works (backward compatibility)
- [ ] Components fail gracefully when HAL is not available
- [ ] HAL auto-detects board correctly on RPi5 and OPi5B
- [ ] Example RDL files are updated and work on both boards

---

## Out of Scope

The following components do NOT require HAL migration:

### Keyboard Input (`components/input/keyboard/`)
- Uses Linux evdev (`/dev/input/event*`)
- evdev is not GPIO/I2C/SPI - it's a separate input subsystem
- Device path is already user-specified in config
- No board-specific differences

### Camera V4L2 (`components/camera/v4l2/`, `driver/camera/v4l2/`)
- Uses V4L2 (`/dev/video*`) via blackjack/webcam library
- V4L2 is not GPIO/I2C/SPI - it's a separate video subsystem
- Device path is already user-specified in config
- No board-specific differences

### Serial/GPS (`components/serial/`)
- Uses external gorai-gps library
- Serial port path is user-specified
- No board-specific differences beyond device naming

---

## Future Considerations

### SPI Components
When SPI components are added, they should follow the same pattern:
- Use `HAL.SPI(bus)` instead of direct spidev access
- Config uses `bus` number instead of device path

### Hardware PWM
When hardware PWM support is added:
- Implement `driver/pwm/linux/sysfs.go` using `/sys/class/pwm/`
- Components can check `HAL.PWM(chip)` availability
- Fall back to software PWM via `HAL.SoftwarePWM(pin)` if hardware unavailable
