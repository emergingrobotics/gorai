# Sensors

Sensors are how robots perceive the world—their eyes, ears, and proprioception. They transform physical phenomena—light, temperature, acceleration, distance—into data structures your software can reason about.

In Gorai, all sensors implement a common interface that returns readings as key-value maps. This simple abstraction handles everything from temperature sensors to IMUs to GPS receivers. This chapter covers the Sensor interface in depth, surveys the built-in sensor types, explains the Protocol Buffer data types for sensor data, and introduces the fake pattern that makes sensors testable without hardware.

## The Sensor Interface

Every sensor in Gorai implements a simple interface from `pkg/resource/resource.go`:

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

## Built-in Sensor Types

Gorai provides interfaces and implementations for common sensor types. Each builds on the base Sensor interface with domain-specific methods and data structures.

### Temperature Sensor

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

The hello-sensor example in Chapter 12 implements a complete temperature sensor.

### IMU (Inertial Measurement Unit)

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

**Coordinate frames** matter for IMUs. Gorai follows REP 103 conventions:

- X: Forward
- Y: Left
- Z: Up

Document your IMU's native frame and any transformations applied.

**Calibration considerations**:

- Accelerometer bias: Subtract offset measured at rest
- Gyroscope drift: Integrate error accumulates over time
- Magnetometer hard/soft iron: Requires figure-8 calibration routine

### GPS

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

Gorai GPS implementations parse these into structured data.

**Integration with navigation**: GPS alone isn't sufficient for robot localization—it's too slow and inaccurate. Combine with IMU and wheel odometry for sensor fusion.

### Encoder

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

### Range Sensors

Distance sensors come in several technologies. Gorai provides the `RangeSensor` interface:

```go
type RangeSensor interface {
    component.Component

    GetRange(ctx context.Context) (float64, error)        // meters
    GetRanges(ctx context.Context) ([]float64, error)     // for array sensors
    GetMinRange(ctx context.Context) (float64, error)
    GetMaxRange(ctx context.Context) (float64, error)
}
```

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

**Time-of-Flight** (e.g., VL53L0X, VL53L1X):

```go
map[string]any{
    "distance":   1.23,       // Meters
    "min_range":  0.03,
    "max_range":  4.0,
    "ambient":    123,        // Ambient light level
}
```

Pros: Fast, accurate, works in various lighting
Cons: Limited range compared to ultrasonic

### AHRS (Attitude and Heading Reference System)

AHRS extends IMU with onboard sensor fusion. Devices like the BNO055 perform orientation calculations internally, providing ready-to-use Euler angles and quaternions:

```go
type AHRS interface {
    IMU  // Inherits all IMU methods

    GetEulerAngles(ctx context.Context) (roll, pitch, yaw float64, err error)
    GetQuaternion(ctx context.Context) (x, y, z, w float64, err error)
    GetLinearAccelerationWithoutGravity(ctx context.Context) (x, y, z float64, err error)
    GetGravityVector(ctx context.Context) (x, y, z float64, err error)
    GetCalibrationStatus(ctx context.Context) (sys, gyro, accel, mag uint8, err error)
}
```

**Standard readings:**

```go
map[string]any{
    // From IMU
    "accel_x": 0.01, "accel_y": -0.02, "accel_z": 9.81,
    "gyro_x": 0.001, "gyro_y": 0.002, "gyro_z": 0.000,
    "mag_x": 25.3, "mag_y": 5.1, "mag_z": 42.7,

    // AHRS-specific
    "euler_roll": 0.5, "euler_pitch": -1.2, "euler_yaw": 45.0,
    "quat_x": 0.0, "quat_y": 0.0, "quat_z": 0.38, "quat_w": 0.92,
    "linear_accel_x": 0.01, "linear_accel_y": -0.02, "linear_accel_z": 0.0,
    "gravity_x": 0.0, "gravity_y": 0.0, "gravity_z": 9.81,
    "calib_sys": 3, "calib_gyro": 3, "calib_accel": 2, "calib_mag": 3,
}
```

**Why AHRS vs raw IMU?**

- **Easier integration**: Get orientation directly without implementing sensor fusion
- **Better accuracy**: Onboard algorithms tuned for the specific hardware
- **Lower CPU load**: Processing happens on the sensor chip

Common AHRS sensors: BNO055, BNO085, ICM-20948

### LiDAR

LiDAR sensors provide 2D or 3D point cloud data for mapping and obstacle detection:

```go
type LiDAR interface {
    component.Component

    GetScan(ctx context.Context) (*LaserScan, error)
    GetPointCloud(ctx context.Context) (*PointCloud, error)  // for 3D
    GetScanRate(ctx context.Context) (float64, error)        // Hz
    SetScanMode(ctx context.Context, mode string) error
    GetProperties(ctx context.Context) (LiDARProperties, error)
}

type LaserScan struct {
    AngleMin       float64   // rad
    AngleMax       float64   // rad
    AngleIncrement float64   // rad
    Ranges         []float64 // meters
    Intensities    []float64 // optional
    Timestamp      int64     // nanoseconds
}

type LiDARProperties struct {
    MinRange          float64
    MaxRange          float64
    AngularResolution float64 // degrees
    SampleRate        int     // points/second
    Is3D              bool
}
```

**2D LiDAR** (e.g., RPLIDAR, Hokuyo):

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

**3D LiDAR** returns point clouds with x, y, z coordinates for each point. Used in autonomous vehicles and advanced robotics.

### Presence Sensors

Presence sensors detect whether something (usually a person) is in a given area:

```go
type PresenceSensor interface {
    component.Component

    IsPresenceDetected(ctx context.Context) (bool, error)
    GetDistance(ctx context.Context) (float64, error)        // meters, if supported
    GetMotionState(ctx context.Context) (MotionState, error)
}

type MotionState int
const (
    MotionUnknown MotionState = iota
    MotionStatic
    MotionMoving
)
```

**PIR (Passive Infrared)** sensors detect motion via infrared changes:

```go
map[string]any{
    "presence": true,
    "motion":   "moving",  // or "static", "unknown"
}
```

**mmWave radar** (e.g., LD2410, LD2450) provides distance and motion state:

```go
map[string]any{
    "presence": true,
    "distance": 1.5,       // meters
    "motion":   "static",
    "energy":   45,        // detection energy/confidence
}
```

### Thermal Arrays

Thermal imaging sensors like the AMG8833 (8x8) or MLX90640 (32x24) provide temperature grids:

```go
type ThermalArray interface {
    component.Component

    GetTemperatureGrid(ctx context.Context) ([][]float64, error)  // °C
    GetAmbientTemperature(ctx context.Context) (float64, error)
    GetMinMaxTemperature(ctx context.Context) (min, max float64, err error)
    GetResolution(ctx context.Context) (width, height int, err error)
}
```

**Standard readings:**

```go
map[string]any{
    "grid": [][]float64{
        {24.5, 25.0, 25.2, 26.1, ...},
        {24.8, 25.1, 36.5, 37.2, ...},  // Person detected!
        ...
    },
    "ambient":  24.0,
    "min_temp": 24.0,
    "max_temp": 37.2,
    "width":    8,
    "height":   8,
}
```

**Applications**: People detection, hotspot identification, thermal monitoring

### Force Sensors

Force sensors measure mechanical force, essential for manipulation tasks:

```go
type ForceSensor interface {
    component.Component

    GetForce(ctx context.Context) (float64, error)  // Newtons
    Tare(ctx context.Context) error
}
```

**Load cells** provide single-axis force measurement:

```go
map[string]any{
    "force":    5.2,      // Newtons
    "raw":      12345,    // ADC value
    "tared":    true,
}
```

### Force6DOF (6-Axis Force/Torque)

For advanced manipulation, 6-DOF force/torque sensors measure forces and torques in all directions:

```go
type Force6DOF interface {
    ForceSensor

    GetWrench(ctx context.Context) (*Wrench, error)
}

type Wrench struct {
    ForceX, ForceY, ForceZ    float64  // Newtons
    TorqueX, TorqueY, TorqueZ float64  // Nm
}
```

**Standard readings:**

```go
map[string]any{
    "force_x": 0.5, "force_y": -0.2, "force_z": 10.1,
    "torque_x": 0.01, "torque_y": 0.02, "torque_z": 0.0,
}
```

**Applications**: Robot arms, haptic feedback, contact detection

### Current Sensors

Current sensors monitor electrical power consumption:

```go
type CurrentSensor interface {
    component.Component

    GetCurrent(ctx context.Context) (float64, error)  // Amps
    GetVoltage(ctx context.Context) (float64, error)  // Volts, if supported
    GetPower(ctx context.Context) (float64, error)    // Watts, if supported
}
```

**Standard readings:**

```go
map[string]any{
    "current": 2.5,       // Amps
    "voltage": 12.1,      // Volts
    "power":   30.25,     // Watts
}
```

Common sensors: INA219, INA260 (I2C-based power monitors)

**Applications**: Battery monitoring, motor current sensing, power budgeting

### Reflectance Sensors

Reflectance sensors detect surface reflectivity, commonly used for line following:

```go
type ReflectanceSensor interface {
    component.Component

    GetReflectances(ctx context.Context) ([]float64, error)  // 0.0-1.0 per channel
    GetLinePosition(ctx context.Context) (float64, error)    // weighted average
    Calibrate(ctx context.Context) error
}
```

**Standard readings** (e.g., QTR-8RC 8-channel sensor):

```go
map[string]any{
    "reflectances": []float64{0.9, 0.8, 0.2, 0.1, 0.1, 0.2, 0.8, 0.9},
    "line_position": 0.5,  // Center = 0.5, Left = 0.0, Right = 1.0
}
```

**Calibration** is essential—sensors must learn what "white" and "black" look like for the specific surface.

### Sensor Type Summary

| Sensor Type | Key Interface Methods | Common Hardware |
|-------------|----------------------|-----------------|
| **IMU** | LinearAcceleration, AngularVelocity, Orientation | MPU6050, LSM6DS3 |
| **AHRS** | GetEulerAngles, GetQuaternion, GetCalibrationStatus | BNO055, BNO085 |
| **GPS** | Position, Accuracy, Fix, GetSatellitesUsed | NEO-6M, NEO-M8N |
| **Encoder** | Position, GetVelocity, GetResolution | Magnetic, optical |
| **RangeSensor** | GetRange, GetMinRange, GetMaxRange | HC-SR04, VL53L0X |
| **LiDAR** | GetScan, GetPointCloud, GetProperties | RPLIDAR, Hokuyo |
| **PresenceSensor** | IsPresenceDetected, GetMotionState | PIR, LD2410 |
| **ThermalArray** | GetTemperatureGrid, GetMinMaxTemperature | AMG8833, MLX90640 |
| **ForceSensor** | GetForce, Tare | Load cells, FSRs |
| **Force6DOF** | GetWrench | ATI F/T sensors |
| **CurrentSensor** | GetCurrent, GetVoltage, GetPower | INA219, INA260 |
| **ReflectanceSensor** | GetReflectances, GetLinePosition | QTR-8RC |

## Sensor Data Types (Protocol Buffers)

While `Readings()` returns dynamic maps, structured sensor data uses Protocol Buffers for efficient serialization and type safety.

### The sensor.proto Definitions

Gorai defines standard sensor messages in `api/proto/gorai/sensor/sensor.proto`:

```protobuf
syntax = "proto3";
package gorai.sensor;

import "gorai/std/std.proto";
import "gorai/geometry/geometry.proto";

// Imu - Inertial Measurement Unit data
message Imu {
    std.Header header = 1;

    geometry.Quaternion orientation = 2;
    repeated double orientation_covariance = 3;

    geometry.Vector3 angular_velocity = 4;
    repeated double angular_velocity_covariance = 5;

    geometry.Vector3 linear_acceleration = 6;
    repeated double linear_acceleration_covariance = 7;
}

// Image - Raw camera image
message Image {
    std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;
    string encoding = 4;    // "rgb8", "bgr8", "mono8", etc.
    uint32 step = 5;        // Row length in bytes
    bytes data = 6;
}

// LaserScan - 2D laser scan
message LaserScan {
    std.Header header = 1;
    float angle_min = 2;
    float angle_max = 3;
    float angle_increment = 4;
    float time_increment = 5;
    float scan_time = 6;
    float range_min = 7;
    float range_max = 8;
    repeated float ranges = 9;
    repeated float intensities = 10;
}

// Range - Single distance measurement
message Range {
    std.Header header = 1;
    uint32 radiation_type = 2;   // ULTRASOUND=0, INFRARED=1
    float field_of_view = 3;
    float min_range = 4;
    float max_range = 5;
    float range = 6;
}

// NavSatFix - GPS position
message NavSatFix {
    std.Header header = 1;

    int32 status = 2;           // STATUS_NO_FIX=-1, FIX=0, SBAS=1, GBAS=2
    uint32 service = 3;         // SERVICE_GPS=1, GLONASS=2, ...

    double latitude = 4;
    double longitude = 5;
    double altitude = 6;

    repeated double position_covariance = 7;
    uint32 position_covariance_type = 8;
}

// BatteryState - Power source status
message BatteryState {
    std.Header header = 1;
    float voltage = 2;
    float current = 3;
    float charge = 4;
    float capacity = 5;
    float design_capacity = 6;
    float percentage = 7;
    uint32 power_supply_status = 8;
    uint32 power_supply_health = 9;
    uint32 power_supply_technology = 10;
    bool present = 11;
}
```

### Timestamps and Headers

Every sensor message includes a Header:

```protobuf
message Header {
    Timestamp stamp = 1;
    string frame_id = 2;
    uint32 seq = 3;
}

message Timestamp {
    int64 seconds = 1;
    int32 nanos = 2;
}
```

- **stamp**: When the measurement was taken (not when it was published)
- **frame_id**: Coordinate frame reference (e.g., "imu_link", "camera_optical")
- **seq**: Sequence number for ordering and gap detection

Usage in Go:

```go
import "google.golang.org/protobuf/types/known/timestamppb"

msg := &sensor.Imu{
    Header: &std.Header{
        Stamp:   timestamppb.Now(),
        FrameId: "imu_link",
        Seq:     atomic.AddUint32(&seq, 1),
    },
    LinearAcceleration: &geometry.Vector3{
        X: accel.X,
        Y: accel.Y,
        Z: accel.Z,
    },
    // ...
}
```

### Covariance Matrices for Uncertainty

Sensor data is uncertain. Covariance matrices express this uncertainty:

```go
// 3x3 covariance matrix as 9 elements, row-major
// [0 1 2]
// [3 4 5]
// [6 7 8]

msg.OrientationCovariance = []float64{
    0.01, 0,    0,     // Roll variance and correlations
    0,    0.01, 0,     // Pitch variance and correlations
    0,    0,    0.02,  // Yaw variance and correlations
}
```

- **Diagonal elements**: Variance in each dimension
- **Off-diagonal elements**: Correlation between dimensions

For uncorrelated sensors, use a diagonal matrix. For unknown covariance, use -1 in the first element as a flag.

## Fake Sensors for Testing

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
    readCount   int
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
    f.mu.Lock()
    defer f.mu.Unlock()

    f.readCount++

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

---

With sensors understood, you're ready to explore how robots act on the world. Chapter 6 covers actuators—motors, servos, and other devices that create motion.
