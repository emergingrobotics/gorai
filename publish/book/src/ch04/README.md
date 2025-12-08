# Components: Sensors

Sensors are components that observe the physical world. This chapter covers the sensor interfaces in Gorai and how to work with different sensor types.

## The Sensor Interface

All sensors implement the base `Sensor` interface:

```go
type Sensor interface {
    resource.Resource

    // Readings returns a map of sensor readings
    // Keys are reading names, values are the measurements
    Readings(ctx context.Context) (map[string]any, error)
}
```

The `Readings()` method returns a flexible map that can contain any sensor data:

```go
// Example IMU readings
readings, _ := imu.Readings(ctx)
// Returns:
// {
//   "linear_acceleration_x": 0.1,
//   "linear_acceleration_y": 0.0,
//   "linear_acceleration_z": 9.8,
//   "angular_velocity_x": 0.0,
//   "angular_velocity_y": 0.01,
//   "angular_velocity_z": 0.0,
// }

// Example temperature sensor
readings, _ := tempSensor.Readings(ctx)
// Returns:
// {
//   "temperature_celsius": 42.5,
//   "humidity_percent": 65.0,
// }
```

## Specialized Sensor Interfaces

While `Readings()` works for any sensor, Gorai provides specialized interfaces for common sensor types with strongly-typed methods.

### IMU (Inertial Measurement Unit)

IMUs measure acceleration, rotation, and orientation:

```go
type IMU interface {
    Sensor

    // LinearAcceleration returns acceleration in m/s² for X, Y, Z axes
    LinearAcceleration(ctx context.Context) (x, y, z float64, err error)

    // AngularVelocity returns rotation rate in rad/s for X, Y, Z axes
    AngularVelocity(ctx context.Context) (x, y, z float64, err error)

    // Orientation returns quaternion (x, y, z, w) representing orientation
    // Returns error if IMU doesn't support orientation
    Orientation(ctx context.Context) (x, y, z, w float64, err error)

    // CompassHeading returns heading in degrees (0-360) relative to magnetic north
    // Returns error if IMU doesn't have a magnetometer
    CompassHeading(ctx context.Context) (float64, error)

    // Properties returns IMU capabilities
    Properties(ctx context.Context) (IMUProperties, error)
}

type IMUProperties struct {
    SupportsOrientation   bool
    SupportsCompassHeading bool
    AccelerationRange     float64  // Max acceleration in m/s²
    GyroRange             float64  // Max rotation rate in rad/s
}
```

**Example: Reading from an IMU**

```go
func monitorIMU(imu sensor.IMU) {
    ctx := context.Background()

    // Check capabilities
    props, _ := imu.Properties(ctx)

    // Read acceleration
    ax, ay, az, err := imu.LinearAcceleration(ctx)
    if err != nil {
        log.Printf("Acceleration error: %v", err)
        return
    }
    fmt.Printf("Acceleration: (%.2f, %.2f, %.2f) m/s²\n", ax, ay, az)

    // Read angular velocity
    gx, gy, gz, _ := imu.AngularVelocity(ctx)
    fmt.Printf("Angular velocity: (%.2f, %.2f, %.2f) rad/s\n", gx, gy, gz)

    // Read orientation if supported
    if props.SupportsOrientation {
        ox, oy, oz, ow, _ := imu.Orientation(ctx)
        fmt.Printf("Orientation: (%.2f, %.2f, %.2f, %.2f)\n", ox, oy, oz, ow)
    }
}
```

### GPS (Global Positioning System)

GPS sensors provide location and velocity:

```go
type GPS interface {
    Sensor

    // Position returns latitude, longitude in degrees and altitude in meters
    Position(ctx context.Context) (lat, lng, alt float64, err error)

    // LinearVelocity returns velocity in m/s for each axis
    LinearVelocity(ctx context.Context) (vx, vy, vz float64, err error)

    // Accuracy returns horizontal and vertical accuracy in meters
    Accuracy(ctx context.Context) (horizontal, vertical float64, err error)

    // Fix returns current fix type
    Fix(ctx context.Context) (GPSFix, error)

    // Satellites returns number of satellites used and visible
    Satellites(ctx context.Context) (used, visible int, err error)
}

type GPSFix int

const (
    GPSFixNone GPSFix = iota  // No fix
    GPSFix2D                   // 2D fix (lat/lng only)
    GPSFix3D                   // 3D fix (lat/lng/alt)
    GPSFixDGPS                 // Differential GPS
    GPSFixRTK                  // Real-Time Kinematic (centimeter accuracy)
)
```

**Example: Monitoring GPS**

```go
func monitorGPS(gps sensor.GPS) {
    ctx := context.Background()

    // Check fix quality
    fix, _ := gps.Fix(ctx)
    if fix == sensor.GPSFixNone {
        fmt.Println("No GPS fix")
        return
    }

    // Get position
    lat, lng, alt, _ := gps.Position(ctx)
    fmt.Printf("Position: %.6f, %.6f @ %.1fm\n", lat, lng, alt)

    // Get accuracy
    hAcc, vAcc, _ := gps.Accuracy(ctx)
    fmt.Printf("Accuracy: ±%.1fm horizontal, ±%.1fm vertical\n", hAcc, vAcc)

    // Get velocity
    vx, vy, vz, _ := gps.LinearVelocity(ctx)
    speed := math.Sqrt(vx*vx + vy*vy)
    fmt.Printf("Speed: %.1f m/s\n", speed)
}
```

### Encoder

Encoders measure rotational or linear position:

```go
type Encoder interface {
    Sensor

    // Position returns the current position
    // For rotary encoders: revolutions (can be fractional)
    // For linear encoders: distance in meters
    Position(ctx context.Context) (float64, error)

    // ResetPosition sets the current position as the new zero
    ResetPosition(ctx context.Context, offset float64) error

    // Properties returns encoder capabilities
    Properties(ctx context.Context) (EncoderProperties, error)
}

type EncoderProperties struct {
    TicksPerRevolution int     // For rotary encoders
    SupportsDirection  bool    // Can detect direction
    SupportsVelocity   bool    // Can calculate velocity
    IsAbsolute         bool    // Absolute vs incremental
}
```

**Example: Reading encoder position**

```go
func trackPosition(enc sensor.Encoder) {
    ctx := context.Background()

    // Reset to zero at startup
    enc.ResetPosition(ctx, 0)

    // Track position changes
    var lastPos float64
    for {
        pos, _ := enc.Position(ctx)
        if pos != lastPos {
            delta := pos - lastPos
            fmt.Printf("Position: %.3f revs (delta: %.3f)\n", pos, delta)
            lastPos = pos
        }
        time.Sleep(10 * time.Millisecond)
    }
}
```

### RangeFinder

RangeFinders measure distance to objects:

```go
type RangeFinder interface {
    Sensor

    // Range returns distance in meters, or error if out of range
    Range(ctx context.Context) (float64, error)

    // Properties returns sensor capabilities
    Properties(ctx context.Context) (RangeFinderProperties, error)
}

type RangeFinderProperties struct {
    MinRange   float64  // Minimum detectable distance (meters)
    MaxRange   float64  // Maximum detectable distance (meters)
    FieldOfView float64 // FOV in radians (for ultrasonic)
    Type       string   // "ultrasonic", "infrared", "lidar", "tof"
}
```

**Example: Obstacle detection**

```go
func detectObstacles(rf sensor.RangeFinder) {
    ctx := context.Background()

    props, _ := rf.Properties(ctx)

    distance, err := rf.Range(ctx)
    if err != nil {
        // Out of range or sensor error
        fmt.Println("No obstacle detected")
        return
    }

    if distance < 0.5 {
        fmt.Printf("WARNING: Obstacle at %.2fm!\n", distance)
    } else if distance < props.MaxRange {
        fmt.Printf("Object at %.2fm\n", distance)
    }
}
```

## Sensor Data Types (Protocol Buffers)

Gorai defines Protocol Buffer messages for common sensor data:

### Vector3

```protobuf
message Vector3 {
    double x = 1;
    double y = 2;
    double z = 3;
}
```

### IMU Reading

```protobuf
message IMUReading {
    google.protobuf.Timestamp timestamp = 1;
    Vector3 linear_acceleration = 2;  // m/s²
    Vector3 angular_velocity = 3;     // rad/s
    Quaternion orientation = 4;        // Optional
    double compass_heading = 5;        // Degrees, optional
}
```

### GPS Reading

```protobuf
message GPSReading {
    google.protobuf.Timestamp timestamp = 1;
    double latitude = 2;   // Degrees
    double longitude = 3;  // Degrees
    double altitude = 4;   // Meters above sea level
    Vector3 velocity = 5;  // m/s
    double horizontal_accuracy = 6;  // Meters
    double vertical_accuracy = 7;    // Meters
    int32 fix_type = 8;
    int32 satellites_used = 9;
}
```

### Temperature Reading

```protobuf
message TemperatureReading {
    google.protobuf.Timestamp timestamp = 1;
    double celsius = 2;
    double fahrenheit = 3;
    string sensor_name = 4;
    int32 thermal_zone = 5;  // Optional, for multi-zone sensors
}
```

### Distance Reading

```protobuf
message DistanceReading {
    google.protobuf.Timestamp timestamp = 1;
    double distance_meters = 2;
    double min_range = 3;
    double max_range = 4;
    bool in_range = 5;
}
```

## Publishing Sensor Data

Sensors publish data to NATS topics:

```go
func (s *IMUSensor) publishLoop(ctx context.Context) {
    ticker := time.NewTicker(10 * time.Millisecond)  // 100Hz
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            ax, ay, az, _ := s.reader.ReadAcceleration()
            gx, gy, gz, _ := s.reader.ReadGyro()

            reading := &pb.IMUReading{
                Timestamp:          timestamppb.Now(),
                LinearAcceleration: &pb.Vector3{X: ax, Y: ay, Z: az},
                AngularVelocity:    &pb.Vector3{X: gx, Y: gy, Z: gz},
            }

            data, _ := proto.Marshal(reading)
            s.nc.Publish("gorai.myrobot.component.sensor.imu.readings", data)
        }
    }
}
```

### Topic Conventions for Sensors

| Sensor Type | Topic Pattern | Example |
|-------------|---------------|---------|
| IMU | `gorai.{robot}.component.sensor.imu.readings` | IMU data stream |
| GPS | `gorai.{robot}.component.sensor.gps.position` | Position updates |
| Encoder | `gorai.{robot}.component.sensor.encoder.{name}.position` | Encoder position |
| Range | `gorai.{robot}.component.sensor.range.{name}.distance` | Distance reading |
| Temperature | `gorai.{robot}.component.sensor.temp.{name}.reading` | Temperature |

## Testing with Fakes

Every sensor type should have a fake implementation for testing:

```go
// fake/fake.go
type FakeIMU struct {
    resource.Named

    mu           sync.Mutex
    acceleration Vector3
    angularVel   Vector3
    orientation  Quaternion
}

func New(name resource.Name) *FakeIMU {
    return &FakeIMU{
        Named: resource.NewNamed(name),
        acceleration: Vector3{X: 0, Y: 0, Z: 9.8},
        angularVel:   Vector3{X: 0, Y: 0, Z: 0},
        orientation:  Quaternion{X: 0, Y: 0, Z: 0, W: 1},
    }
}

func (f *FakeIMU) LinearAcceleration(ctx context.Context) (x, y, z float64, err error) {
    f.mu.Lock()
    defer f.mu.Unlock()
    return f.acceleration.X, f.acceleration.Y, f.acceleration.Z, nil
}

// SetAcceleration allows tests to control the sensor
func (f *FakeIMU) SetAcceleration(x, y, z float64) {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.acceleration = Vector3{X: x, Y: y, Z: z}
}

func (f *FakeIMU) Readings(ctx context.Context) (map[string]any, error) {
    ax, ay, az, _ := f.LinearAcceleration(ctx)
    gx, gy, gz, _ := f.AngularVelocity(ctx)

    return map[string]any{
        "linear_acceleration_x": ax,
        "linear_acceleration_y": ay,
        "linear_acceleration_z": az,
        "angular_velocity_x": gx,
        "angular_velocity_y": gy,
        "angular_velocity_z": gz,
    }, nil
}
```

### Using Fakes in Tests

```go
func TestNavigationWithIMU(t *testing.T) {
    // Create fake IMU
    imu := fake.New(resource.NewName("gorai", "component", "sensor", "imu"))

    // Set initial state
    imu.SetAcceleration(0, 0, 9.8)

    // Create component under test
    nav := NewNavigator(imu)

    // Simulate tilt
    imu.SetAcceleration(2.0, 0, 9.6)  // Tilted forward

    // Verify navigation responds to tilt
    state, _ := nav.GetState(context.Background())
    assert.True(t, state.TiltDetected)
}
```

## Configuration

Sensors are configured via JSON:

```json
{
  "name": "imu",
  "type": "component",
  "subtype": "sensor",
  "model": "gorai:driver:mpu6050",
  "config": {
    "i2c_bus": 1,
    "i2c_address": "0x68",
    "sample_rate_hz": 100,
    "accel_range": "4g",
    "gyro_range": "500dps"
  }
}
```

Common configuration options:

| Option | Description | Example Values |
|--------|-------------|----------------|
| `sample_rate_hz` | Reading frequency | 10, 50, 100, 200 |
| `i2c_bus` | I2C bus number | 0, 1 |
| `i2c_address` | I2C device address | "0x68", "0x1E" |
| `spi_bus` | SPI bus number | 0, 1 |
| `gpio_pin` | GPIO pin for digital sensors | 17, 27 |
| `uart_port` | Serial port | "/dev/ttyAMA0" |
| `baud_rate` | Serial baud rate | 9600, 115200 |

## Best Practices

### 1. Use Appropriate Data Types

```go
// Good: Use floats for continuous measurements
func (s *TempSensor) Temperature(ctx context.Context) (float64, error)

// Good: Use integers for discrete counts
func (s *Counter) Count(ctx context.Context) (int64, error)

// Good: Use enums for categorical data
func (s *GPS) Fix(ctx context.Context) (GPSFix, error)
```

### 2. Handle Errors Gracefully

```go
func readSensor(s sensor.Sensor) {
    ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
    defer cancel()

    readings, err := s.Readings(ctx)
    if err != nil {
        if errors.Is(err, context.DeadlineExceeded) {
            log.Println("Sensor read timed out")
        } else {
            log.Printf("Sensor error: %v", err)
        }
        return
    }

    // Process readings...
}
```

### 3. Validate Readings

```go
func (s *IMUSensor) LinearAcceleration(ctx context.Context) (x, y, z float64, err error) {
    x, y, z, err = s.reader.ReadAcceleration()
    if err != nil {
        return 0, 0, 0, err
    }

    // Sanity check: acceleration shouldn't be too high
    magnitude := math.Sqrt(x*x + y*y + z*z)
    if magnitude > 100 {  // > 10g
        return 0, 0, 0, fmt.Errorf("acceleration out of range: %.1f m/s²", magnitude)
    }

    return x, y, z, nil
}
```

### 4. Document Units

Always document the units for measurements:

```go
// Temperature returns the temperature in degrees Celsius.
func (s *TempSensor) Temperature(ctx context.Context) (float64, error)

// Distance returns the distance in meters.
func (s *RangeFinder) Distance(ctx context.Context) (float64, error)

// LinearAcceleration returns acceleration in m/s² for each axis.
// Positive X is forward, positive Y is left, positive Z is up.
func (s *IMU) LinearAcceleration(ctx context.Context) (x, y, z float64, err error)
```
