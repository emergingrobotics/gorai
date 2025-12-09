# Chapter 4: Components - Sensors

Sensors are the robot's eyes, ears, and proprioception. They transform physical phenomena—light, temperature, acceleration, distance—into data structures your software can reason about.

## 4.1 The Sensor Interface

Every sensor in GoRAI implements a simple interface from `pkg/resource/resource.go`:

```go
type Sensor interface {
    Resource

    // Readings returns the current sensor readings as key-value pairs.
    // The keys and value types depend on the specific sensor implementation.
    Readings(ctx context.Context) (map[string]any, error)
}
```

That's it. One method beyond the base Resource interface. This simplicity is intentional.

### Why map[string]any for Readings

Different sensors produce radically different data:
- Temperature sensor: Single float (degrees Celsius)
- IMU: Nine floats (3-axis acceleration, gyroscope, magnetometer)
- GPS: Latitude, longitude, altitude, accuracy, satellite count
- LiDAR: Thousands of range measurements

A fixed return type would either be too restrictive or require sensor-specific interfaces for every sensor type. The `map[string]any` approach provides:

- **Flexibility**: Any sensor can return whatever data it produces
- **Discoverability**: Print the map to see what's available
- **Forward compatibility**: New readings can be added without interface changes

```go
readings, _ := tempSensor.Readings(ctx)
// Output: map[temperature_celsius:42.5 temperature_fahrenheit:108.5 zone:thermal_zone0]

readings, _ := imu.Readings(ctx)
// Output: map[accel_x:0.01 accel_y:-0.02 accel_z:9.81 gyro_x:0.001 ...]
```

### Standard Reading Keys

While sensors can return any keys, conventions enable interoperability:

| Key Pattern | Type | Description |
|-------------|------|-------------|
| `temperature_celsius` | float64 | Temperature in Celsius |
| `temperature_fahrenheit` | float64 | Temperature in Fahrenheit |
| `accel_x`, `accel_y`, `accel_z` | float64 | Acceleration (m/s²) |
| `gyro_x`, `gyro_y`, `gyro_z` | float64 | Angular velocity (rad/s) |
| `latitude`, `longitude` | float64 | GPS coordinates (degrees) |
| `altitude` | float64 | Altitude (meters) |
| `distance` | float64 | Range measurement (meters) |
| `battery_percent` | float64 | Battery level (0-100) |

Following conventions enables generic processing:
```go
// Works with any temperature sensor
temp, ok := readings["temperature_celsius"].(float64)
if ok && temp > 80 {
    log.Warn("High temperature detected")
}
```

### Timestamp Handling

Readings represent a point in time. The sensor implementation should include when the measurement was taken:

```go
func (s *TemperatureSensor) Readings(ctx context.Context) (map[string]any, error) {
    reading := readHardware()

    return map[string]any{
        "temperature_celsius":    reading.Value,
        "timestamp":              time.Now(),
        "measurement_duration":   reading.Duration,
    }, nil
}
```

For Protocol Buffer messages, timestamps are explicit:

```go
msg := &sensor.Temperature{
    Header: &std.Header{
        Stamp: timestamppb.Now(),
        FrameId: "thermal_zone0",
    },
    Temperature: reading.Value,
    Variance:    reading.Variance,
}
```

### Implementing the Sensor Interface

A minimal sensor implementation:

```go
type SimpleSensor struct {
    name resource.Name
}

func (s *SimpleSensor) Name() resource.Name {
    return s.name
}

func (s *SimpleSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
    // Update configuration if needed
    return nil
}

func (s *SimpleSensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
    return nil, fmt.Errorf("no commands supported")
}

func (s *SimpleSensor) Close(ctx context.Context) error {
    return nil
}

func (s *SimpleSensor) Readings(ctx context.Context) (map[string]any, error) {
    value := readFromHardware()
    return map[string]any{
        "value": value,
    }, nil
}
```

The real complexity lies in:
- Hardware communication (I2C, SPI, GPIO, serial)
- Parsing hardware data formats
- Calibration and unit conversion
- Error handling and recovery
