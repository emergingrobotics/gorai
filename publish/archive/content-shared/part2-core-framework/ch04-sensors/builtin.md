## 4.2 Built-in Sensor Types

GoRAI provides interfaces and implementations for common sensor types. Each builds on the base Sensor interface with domain-specific methods and data structures.

### 4.2.1 Temperature Sensor

The simplest sensor type—a single scalar measurement:

```go
// Standard readings
map[string]any{
    "temperature_celsius":    42.5,
    "temperature_fahrenheit": 108.5,
    "zone":                   "thermal_zone0",
    "critical_celsius":       105.0,  // Optional: thermal limits
    "warning_celsius":        85.0,
}
```

Common sources:
- Linux thermal zones (`/sys/class/thermal/thermal_zone*/temp`)
- I2C sensors (TMP102, BME280, DS18B20)
- ADC-based thermistors

The hello-sensor example in Chapter 9 implements a complete temperature sensor.

*Cross-reference: See Chapter 9 for complete implementation details.*

### 4.2.2 IMU (Inertial Measurement Unit)

IMUs combine multiple sensors measuring motion:

```go
// Standard readings
map[string]any{
    // Accelerometer (m/s²)
    "accel_x": 0.01,
    "accel_y": -0.02,
    "accel_z": 9.81,  // Gravity

    // Gyroscope (rad/s)
    "gyro_x": 0.001,
    "gyro_y": 0.002,
    "gyro_z": 0.000,

    // Magnetometer (µT) - if available
    "mag_x": 25.3,
    "mag_y": 5.1,
    "mag_z": 42.7,
}
```

**Coordinate frames** matter for IMUs. GoRAI follows REP 103 conventions:
- X: Forward
- Y: Left
- Z: Up

Document your IMU's native frame and any transformations applied.

**Protocol Buffer representation** from `sensor.proto`:

```protobuf
message Imu {
    std.Header header = 1;

    geometry.Quaternion orientation = 2;
    double orientation_covariance = 3;

    geometry.Vector3 angular_velocity = 4;
    double angular_velocity_covariance = 5;

    geometry.Vector3 linear_acceleration = 6;
    double linear_acceleration_covariance = 7;
}
```

**Calibration considerations**:
- Accelerometer bias: Subtract offset measured at rest
- Gyroscope drift: Integrate error accumulates over time
- Magnetometer hard/soft iron: Requires figure-8 calibration routine

### 4.2.3 GPS

GPS sensors provide position on Earth:

```go
map[string]any{
    "latitude":            37.4220,      // Degrees
    "longitude":          -122.0841,     // Degrees
    "altitude":            10.5,         // Meters above sea level
    "horizontal_accuracy": 2.5,          // Meters (CEP)
    "vertical_accuracy":   4.0,          // Meters
    "speed":               1.2,          // m/s
    "heading":             45.0,         // Degrees from north
    "satellites":          12,           // Satellites in view
    "fix_type":           "3d",          // "none", "2d", "3d", "rtk"
}
```

**NMEA parsing**: Most GPS modules output NMEA sentences over serial:
```
$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,47.0,M,,*47
```

GoRAI GPS implementations parse these into structured data.

**Integration with navigation**: GPS alone isn't sufficient for robot localization—it's too slow and inaccurate. Combine with IMU and wheel odometry for sensor fusion.

### 4.2.4 Encoder

Encoders measure rotational position, typically for motors:

```go
map[string]any{
    "position":       1234.5,     // Ticks or radians
    "velocity":       10.2,       // Ticks/s or rad/s
    "ticks_per_rev":  1200,       // Encoder resolution
    "direction":      1,          // 1 = forward, -1 = reverse
}
```

**Quadrature encoding**: Most encoders output two signals (A and B) 90° out of phase, allowing direction detection and 4x resolution.

**Velocity calculation**: Differentiate position over time, but filter to reduce noise:
```go
func (e *Encoder) calculateVelocity() float64 {
    now := time.Now()
    dt := now.Sub(e.lastTime).Seconds()
    dp := e.position - e.lastPosition

    e.lastTime = now
    e.lastPosition = e.position

    // Low-pass filter
    rawVelocity := dp / dt
    e.filteredVelocity = 0.8*e.filteredVelocity + 0.2*rawVelocity

    return e.filteredVelocity
}
```

### 4.2.5 Range Finders

Distance sensors come in several technologies:

**Ultrasonic** (e.g., HC-SR04):
```go
map[string]any{
    "distance":   0.42,       // Meters
    "min_range":  0.02,       // Minimum detectable
    "max_range":  4.0,        // Maximum range
    "field_of_view": 0.26,    // Radians (~15°)
}
```
Pros: Cheap, works with any surface
Cons: Slow (limited update rate), wide beam, temperature sensitive

**Infrared** (e.g., Sharp GP2Y0A21):
```go
map[string]any{
    "distance":   0.25,
    "min_range":  0.10,
    "max_range":  0.80,
}
```
Pros: Fast, narrow beam
Cons: Surface-dependent (dark surfaces absorb IR)

**LiDAR** (e.g., RPLIDAR, Hokuyo):
```go
map[string]any{
    "ranges":     []float64{...},  // Array of distances
    "angles":     []float64{...},  // Corresponding angles
    "min_range":  0.15,
    "max_range":  12.0,
    "angle_min":  -3.14159,        // -180°
    "angle_max":  3.14159,         // +180°
    "scan_time":  0.1,             // Seconds per scan
}
```

**Point cloud generation**: For 3D sensing (RGB-D cameras, rotating LiDAR):
```protobuf
message PointCloud2 {
    std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;
    repeated PointField fields = 4;
    bool is_bigendian = 5;
    uint32 point_step = 6;
    uint32 row_step = 7;
    bytes data = 8;
    bool is_dense = 9;
}
```
