## 4.4 Fake Sensors for Testing

Every sensor needs a fake—a test double that simulates sensor behavior without real hardware. Fakes are essential for:

- **Unit testing**: Test logic without hardware dependencies
- **Simulation**: Run the full system on a development laptop
- **CI/CD**: Automated tests in environments without robots
- **Debugging**: Reproduce specific scenarios on demand

### Why Fake Implementations Matter

Consider testing a temperature monitoring system:

```go
func TestOverheatDetection(t *testing.T) {
    // With a real sensor, you can't control the temperature
    sensor := realSensor.New()
    // How do you trigger the overheat condition?

    // With a fake, you have complete control
    fake := fake.New()
    fake.SetTemperature(95.0)  // Simulate overheating

    monitor := NewMonitor(fake)
    status := monitor.Check()

    assert.True(t, status.Overheating)
}
```

Without fakes, you'd need:
- Real hardware connected
- A way to actually heat the sensor
- Tests that take minutes instead of milliseconds
- Flaky results from environmental variation

### Fake Implementation Pattern

A good fake implements the same interface as the real sensor plus control methods:

```go
// fake/fake.go
package fake

import (
    "context"
    "sync"

    "github.com/gorai/gorai/pkg/resource"
)

// TemperatureSensor is a fake temperature sensor for testing.
type TemperatureSensor struct {
    name        resource.Name
    mu          sync.RWMutex
    temperature float64
    zone        string
    shouldError bool
    errorMsg    string
}

// New creates a new fake temperature sensor.
func New() *TemperatureSensor {
    return &TemperatureSensor{
        name:        resource.NewComponentName("test", "sensor", "fake_temp"),
        temperature: 42.0,
        zone:        "fake_zone",
    }
}

// Implement the Sensor interface
func (f *TemperatureSensor) Name() resource.Name {
    return f.name
}

func (f *TemperatureSensor) Readings(ctx context.Context) (map[string]any, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()

    if f.shouldError {
        return nil, fmt.Errorf(f.errorMsg)
    }

    return map[string]any{
        "temperature_celsius":    f.temperature,
        "temperature_fahrenheit": f.temperature*9/5 + 32,
        "zone":                   f.zone,
    }, nil
}

func (f *TemperatureSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
    return nil
}

func (f *TemperatureSensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
    return nil, nil
}

func (f *TemperatureSensor) Close(ctx context.Context) error {
    return nil
}

// Control methods for testing

// SetTemperature sets the temperature the fake sensor will return.
func (f *TemperatureSensor) SetTemperature(celsius float64) {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.temperature = celsius
}

// SetError makes the sensor return an error on next reading.
func (f *TemperatureSensor) SetError(msg string) {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.shouldError = true
    f.errorMsg = msg
}

// ClearError stops the sensor from returning errors.
func (f *TemperatureSensor) ClearError() {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.shouldError = false
}

// GetReadCount returns how many times Readings was called (for verification).
func (f *TemperatureSensor) GetReadCount() int {
    f.mu.RLock()
    defer f.mu.RUnlock()
    return f.readCount
}
```

### Configurable Behavior

Fakes should support various test scenarios:

```go
// Configure for specific test scenarios
fake := fake.New()
fake.SetTemperature(25.0)  // Normal temperature
fake.SetZone("cpu_thermal")
fake.SetUpdateRate(100 * time.Millisecond)

// Simulate sensor failure
fake.SetError("I2C read timeout")

// Simulate noisy readings
fake.SetNoise(0.5)  // ±0.5°C random variation

// Simulate gradual change
fake.SetDrift(0.1)  // +0.1°C per reading
```

### Error Injection

Testing error handling requires controlled failures:

```go
func TestSensorRecovery(t *testing.T) {
    fake := fake.New()
    monitor := NewMonitor(fake)

    // Normal operation
    fake.SetTemperature(25.0)
    status, err := monitor.Check()
    assert.NoError(t, err)

    // Simulate sensor failure
    fake.SetError("hardware disconnected")
    status, err = monitor.Check()
    assert.Error(t, err)
    assert.True(t, status.SensorFailed)

    // Simulate recovery
    fake.ClearError()
    fake.SetTemperature(26.0)
    status, err = monitor.Check()
    assert.NoError(t, err)
    assert.False(t, status.SensorFailed)
}
```

### Testing Fakes Themselves

Fakes should have their own tests:

```go
// fake/fake_test.go
func TestFakeSensor_ReturnsConfiguredTemperature(t *testing.T) {
    fake := New()
    fake.SetTemperature(50.0)

    readings, err := fake.Readings(context.Background())
    require.NoError(t, err)

    assert.Equal(t, 50.0, readings["temperature_celsius"])
    assert.Equal(t, 122.0, readings["temperature_fahrenheit"])
}

func TestFakeSensor_ReturnsErrorWhenConfigured(t *testing.T) {
    fake := New()
    fake.SetError("test error")

    _, err := fake.Readings(context.Background())
    assert.Error(t, err)
    assert.Contains(t, err.Error(), "test error")
}
```

*Cross-reference: See Chapter 11 for comprehensive testing strategies.*
