# Gorai Component Reference

This document provides a complete reference for all component interfaces in Gorai. Components are hardware abstractions that implement the `Resource` interface with additional category-specific methods.

## Table of Contents

- [Base Interfaces](#base-interfaces)
- [Sensor Types](#sensor-types)
- [Actuator Types](#actuator-types)
- [Component Categories](#component-categories)
- [Concurrency Model](#concurrency-model)

---

## Base Interfaces

### Resource

All components implement the base `Resource` interface:

```go
type Resource interface {
    // Name returns the unique resource identifier
    Name() resource.Name

    // Reconfigure updates the resource with new configuration
    Reconfigure(ctx context.Context, deps Dependencies, conf Config) error

    // DoCommand executes arbitrary commands for extensibility
    DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error)

    // Close releases all resources and stops background operations
    Close(ctx context.Context) error
}
```

### Component

Components are Resources that interface with hardware:

```go
type Component interface {
    Resource
}
```

### Sensor

Sensors observe the environment without changing it:

```go
type Sensor interface {
    Resource
    Readings(ctx context.Context) (map[string]any, error)
}
```

### Actuator

Actuators change the environment:

```go
type Actuator interface {
    Resource
    IsMoving(ctx context.Context) (bool, error)
    Stop(ctx context.Context) error
}
```

---

## Sensor Types

### IMU (Inertial Measurement Unit)

**Package**: `components/sensor`

```go
type IMU interface {
    component.Component

    // LinearAcceleration returns acceleration in m/s² (x, y, z)
    LinearAcceleration(ctx context.Context) (x, y, z float64, err error)

    // AngularVelocity returns rotation rate in rad/s (x, y, z)
    AngularVelocity(ctx context.Context) (x, y, z float64, err error)

    // Orientation returns orientation as quaternion (x, y, z, w)
    Orientation(ctx context.Context) (x, y, z, w float64, err error)

    // GetMagneticField returns magnetic field in µT (x, y, z)
    GetMagneticField(ctx context.Context) (x, y, z float64, err error)
}
```

**Common hardware**: MPU6050, MPU9250, LSM6DS3, ICM-20948

### AHRS (Attitude and Heading Reference System)

**Package**: `components/sensor`

AHRS extends IMU with onboard sensor fusion:

```go
type AHRS interface {
    IMU

    // GetEulerAngles returns roll, pitch, yaw in degrees
    GetEulerAngles(ctx context.Context) (roll, pitch, yaw float64, err error)

    // GetQuaternion returns orientation as quaternion
    GetQuaternion(ctx context.Context) (x, y, z, w float64, err error)

    // GetLinearAccelerationWithoutGravity returns acceleration without gravity
    GetLinearAccelerationWithoutGravity(ctx context.Context) (x, y, z float64, err error)

    // GetGravityVector returns the gravity vector
    GetGravityVector(ctx context.Context) (x, y, z float64, err error)

    // GetCalibrationStatus returns calibration status (0-3 for each)
    GetCalibrationStatus(ctx context.Context) (sys, gyro, accel, mag uint8, err error)
}
```

**Common hardware**: BNO055, BNO085, ICM-20948

### GPS

**Package**: `components/sensor`

```go
type GPS interface {
    component.Component

    // Position returns latitude, longitude (degrees), altitude (meters)
    Position(ctx context.Context) (lat, lng, alt float64, err error)

    // LinearVelocity returns velocity in m/s (x, y, z)
    LinearVelocity(ctx context.Context) (x, y, z float64, err error)

    // Accuracy returns horizontal and vertical accuracy in meters
    Accuracy(ctx context.Context) (horizontal, vertical float64, err error)

    // Fix returns the current fix type
    Fix(ctx context.Context) (FixType, error)

    // GetHeading returns heading in degrees from north
    GetHeading(ctx context.Context) (float64, error)

    // GetSatellitesUsed returns number of satellites used in fix
    GetSatellitesUsed(ctx context.Context) (int, error)
}

type FixType int
const (
    FixNone FixType = iota
    Fix2D
    Fix3D
    FixDGPS
    FixRTK
)
```

**Common hardware**: NEO-6M, NEO-M8N, ZED-F9P (RTK)

### Encoder

**Package**: `components/sensor`

```go
type Encoder interface {
    component.Component

    // Position returns position in ticks or radians
    Position(ctx context.Context) (float64, error)

    // ResetPosition sets current position to zero
    ResetPosition(ctx context.Context) error

    // Properties returns encoder properties
    Properties(ctx context.Context) (EncoderProperties, error)

    // GetVelocity returns velocity in ticks/s or rad/s
    GetVelocity(ctx context.Context) (float64, error)

    // GetResolution returns PPR or bits
    GetResolution(ctx context.Context) (int, error)
}

type EncoderProperties struct {
    TicksPerRevolution    int
    AngleDegreesSupported bool
    IsAbsolute            bool  // true for absolute encoders
}
```

**Common hardware**: Magnetic encoders (AS5600), optical encoders, AMT10x

### RangeSensor

**Package**: `components/sensor`

```go
type RangeSensor interface {
    component.Component

    // GetRange returns distance in meters
    GetRange(ctx context.Context) (float64, error)

    // GetRanges returns multiple ranges for array sensors
    GetRanges(ctx context.Context) ([]float64, error)

    // GetMinRange returns minimum detectable range
    GetMinRange(ctx context.Context) (float64, error)

    // GetMaxRange returns maximum detectable range
    GetMaxRange(ctx context.Context) (float64, error)
}
```

**Common hardware**: HC-SR04 (ultrasonic), VL53L0X/VL53L1X (ToF), Sharp GP2Y (IR)

### LiDAR

**Package**: `components/sensor`

```go
type LiDAR interface {
    component.Component

    // GetScan returns a 2D laser scan
    GetScan(ctx context.Context) (*LaserScan, error)

    // GetPointCloud returns 3D point cloud (for 3D LiDARs)
    GetPointCloud(ctx context.Context) (*PointCloud, error)

    // GetScanRate returns scan rate in Hz
    GetScanRate(ctx context.Context) (float64, error)

    // SetScanMode sets the scanning mode
    SetScanMode(ctx context.Context, mode string) error

    // GetProperties returns LiDAR properties
    GetProperties(ctx context.Context) (LiDARProperties, error)
}

type LaserScan struct {
    AngleMin       float64   // Starting angle (rad)
    AngleMax       float64   // Ending angle (rad)
    AngleIncrement float64   // Angular resolution (rad)
    Ranges         []float64 // Distance measurements (meters)
    Intensities    []float64 // Intensity values (optional)
    Timestamp      int64     // Measurement time (nanoseconds)
}

type LiDARProperties struct {
    MinRange          float64
    MaxRange          float64
    AngularResolution float64 // degrees
    SampleRate        int     // points/second
    Is3D              bool
}
```

**Common hardware**: RPLIDAR A1/A2/A3, Hokuyo, Velodyne

### PresenceSensor

**Package**: `components/sensor`

```go
type PresenceSensor interface {
    component.Component

    // IsPresenceDetected returns true if presence is detected
    IsPresenceDetected(ctx context.Context) (bool, error)

    // GetDistance returns distance to detected object (if supported)
    GetDistance(ctx context.Context) (float64, error)

    // GetMotionState returns motion state of detected object
    GetMotionState(ctx context.Context) (MotionState, error)
}

type MotionState int
const (
    MotionUnknown MotionState = iota
    MotionStatic
    MotionMoving
)
```

**Common hardware**: PIR sensors, LD2410/LD2450 (mmWave radar)

### ThermalArray

**Package**: `components/sensor`

```go
type ThermalArray interface {
    component.Component

    // GetTemperatureGrid returns 2D temperature grid in °C
    GetTemperatureGrid(ctx context.Context) ([][]float64, error)

    // GetAmbientTemperature returns ambient/sensor temperature
    GetAmbientTemperature(ctx context.Context) (float64, error)

    // GetMinMaxTemperature returns min and max in the grid
    GetMinMaxTemperature(ctx context.Context) (min, max float64, err error)

    // GetResolution returns grid dimensions
    GetResolution(ctx context.Context) (width, height int, err error)
}
```

**Common hardware**: AMG8833 (8x8), MLX90640 (32x24)

### ForceSensor

**Package**: `components/sensor`

```go
type ForceSensor interface {
    component.Component

    // GetForce returns force in Newtons
    GetForce(ctx context.Context) (float64, error)

    // Tare zeros the sensor
    Tare(ctx context.Context) error
}
```

**Common hardware**: Load cells with HX711, FSR sensors

### Force6DOF

**Package**: `components/sensor`

6-axis force/torque sensor:

```go
type Force6DOF interface {
    ForceSensor

    // GetWrench returns full 6-DOF force/torque
    GetWrench(ctx context.Context) (*Wrench, error)
}

type Wrench struct {
    ForceX, ForceY, ForceZ    float64  // Newtons
    TorqueX, TorqueY, TorqueZ float64  // Nm
}
```

**Common hardware**: ATI F/T sensors, robotiq FT 300

### CurrentSensor

**Package**: `components/sensor`

```go
type CurrentSensor interface {
    component.Component

    // GetCurrent returns current in Amps
    GetCurrent(ctx context.Context) (float64, error)

    // GetVoltage returns voltage in Volts (if supported)
    GetVoltage(ctx context.Context) (float64, error)

    // GetPower returns power in Watts (if supported)
    GetPower(ctx context.Context) (float64, error)
}
```

**Common hardware**: INA219, INA260, ACS712

### ReflectanceSensor

**Package**: `components/sensor`

```go
type ReflectanceSensor interface {
    component.Component

    // GetReflectances returns reflectance values (0.0-1.0) per channel
    GetReflectances(ctx context.Context) ([]float64, error)

    // GetLinePosition returns weighted position (-1.0 to 1.0 or 0.0 to 1.0)
    GetLinePosition(ctx context.Context) (float64, error)

    // Calibrate runs calibration routine
    Calibrate(ctx context.Context) error
}
```

**Common hardware**: QTR-8RC, TCRT5000 arrays

---

## Actuator Types

### Motor

**Package**: `components/motor`

```go
type Motor interface {
    component.Actuator

    // SetPower sets power from -1.0 (full reverse) to 1.0 (full forward)
    SetPower(ctx context.Context, power float64) error

    // SetVelocity sets target velocity (rad/s or m/s)
    SetVelocity(ctx context.Context, velocity float64) error

    // GoTo moves to absolute position at given velocity
    GoTo(ctx context.Context, position, velocity float64) error

    // GoFor moves for revolutions at given RPM
    GoFor(ctx context.Context, rpm, revolutions float64) error

    // GetPosition returns current position in revolutions
    GetPosition(ctx context.Context) (float64, error)

    // GetVelocity returns current velocity
    GetVelocity(ctx context.Context) (float64, error)

    // ResetZeroPosition sets current position as zero
    ResetZeroPosition(ctx context.Context, offset float64) error

    // IsPowered returns power state and level
    IsPowered(ctx context.Context) (bool, float64, error)

    // Properties returns motor capabilities
    Properties(ctx context.Context) (Properties, error)
}

type Properties struct {
    PositionReporting bool
    VelocityReporting bool
    SupportsGoTo      bool
}
```

### Servo

**Package**: `components/servo`

```go
type Servo interface {
    component.Actuator

    // SetAngle moves to angle in degrees
    SetAngle(ctx context.Context, degrees float64) error

    // GetAngle returns current angle in degrees
    GetAngle(ctx context.Context) (float64, error)

    // SetSpeed sets movement speed (0.0-1.0)
    SetSpeed(ctx context.Context, speed float64) error

    // SetTorqueLimit sets maximum torque (0.0-1.0)
    SetTorqueLimit(ctx context.Context, limit float64) error

    // GetProperties returns servo properties
    GetProperties(ctx context.Context) (Properties, error)
}

type Properties struct {
    MinAngle     float64  // e.g., -90 or 0
    MaxAngle     float64  // e.g., 90 or 180
    IsContinuous bool     // Continuous rotation servo
    HasFeedback  bool     // Position feedback available
    Protocol     string   // "pwm", "dynamixel", "lx16a", "feetech"
}
```

**Common hardware**: RC servos, Dynamixel, LX-16A, Feetech

### Stepper

**Package**: `components/stepper`

```go
type Stepper interface {
    component.Actuator

    // Step moves by steps (positive=forward, negative=backward)
    Step(ctx context.Context, steps int64) error

    // SetMicrostepping sets microstepping divisor (1, 2, 4, 8, 16, ...)
    SetMicrostepping(ctx context.Context, divisor int) error

    // SetCurrent sets run and hold current in milliamps
    SetCurrent(ctx context.Context, runMA, holdMA int) error

    // GetPosition returns position in steps from zero
    GetPosition(ctx context.Context) (int64, error)

    // ResetPosition sets current position as zero
    ResetPosition(ctx context.Context) error

    // Home moves to home position
    Home(ctx context.Context, direction bool) error

    // GetProperties returns stepper properties
    GetProperties(ctx context.Context) (Properties, error)
}

type Properties struct {
    StepsPerRevolution int     // Typically 200 for 1.8° motors
    MaxMicrostepping   int     // Maximum supported (e.g., 256)
    MaxCurrent         int     // Maximum current in mA
    HasStallDetection  bool    // TMC drivers support this
    Driver             string  // "a4988", "drv8825", "tmc2209", "tmc5160"
}
```

**Common hardware**: NEMA 17/23 motors with A4988, DRV8825, TMC2209/5160 drivers

### Thruster

**Package**: `components/thruster`

```go
type Thruster interface {
    component.Actuator

    // SetThrust sets thrust from -1.0 (full reverse) to 1.0 (full forward)
    SetThrust(ctx context.Context, thrust float64) error

    // GetRPM returns current motor RPM (if telemetry available)
    GetRPM(ctx context.Context) (int, error)

    // GetTemperature returns motor/ESC temperature in °C
    GetTemperature(ctx context.Context) (float64, error)

    // GetCurrent returns motor current in amps
    GetCurrent(ctx context.Context) (float64, error)

    // GetProperties returns thruster properties
    GetProperties(ctx context.Context) (Properties, error)
}

type Properties struct {
    MaxThrustForward  float64  // Newtons or kgf
    MaxThrustReverse  float64  // May differ from forward
    DeadbandWidth     float64  // Values below this = zero thrust
    IsBidirectional   bool     // Can reverse?
    HasTelemetry      bool     // RPM/temp/current available?
    Protocol          string   // "pwm", "i2c", "can"
}
```

**Common hardware**: BlueRobotics T100/T200, generic brushless with ESC

### Valve

**Package**: `components/valve`

```go
type Valve interface {
    component.Actuator

    // Open fully opens the valve
    Open(ctx context.Context) error

    // Shut fully closes the valve (named "Shut" to avoid conflict with Close())
    Shut(ctx context.Context) error

    // SetPosition sets position (0.0=closed, 1.0=open)
    SetPosition(ctx context.Context, position float64) error

    // GetPosition returns current position (0.0-1.0)
    GetPosition(ctx context.Context) (float64, error)

    // IsOpen returns true if fully open
    IsOpen(ctx context.Context) (bool, error)

    // IsClosed returns true if fully closed
    IsClosed(ctx context.Context) (bool, error)
}
```

**Note**: Binary valves snap to 0 or 1; proportional valves support continuous positioning.

**Common hardware**: Solenoid valves, motorized ball valves

### Gripper

**Package**: `components/gripper`

```go
type Gripper interface {
    component.Actuator

    // Open fully opens the gripper
    Open(ctx context.Context) error

    // Close fully closes the gripper
    Close(ctx context.Context) error

    // Grab closes until resistance (returns true if object grasped)
    Grab(ctx context.Context) (bool, error)

    // IsOpen returns true if fully open
    IsOpen(ctx context.Context) (bool, error)
}
```

### Base (Mobile Robot)

**Package**: `components/base`

```go
type Base interface {
    component.Actuator

    // SetVelocity sets linear (m/s) and angular (rad/s) velocity
    SetVelocity(ctx context.Context, linear, angular float64) error

    // MoveStraight moves forward by distance (mm) at velocity
    MoveStraight(ctx context.Context, distanceMm int, velocity float64) error

    // Spin rotates in place by angle (degrees) at velocity
    Spin(ctx context.Context, angleDeg, velocity float64) error

    // GetVelocities returns current linear and angular velocities
    GetVelocities(ctx context.Context) (linear, angular float64, err error)
}
```

### Arm (Manipulator)

**Package**: `components/arm`

```go
type Arm interface {
    component.Actuator

    // EndPosition returns current end effector pose
    EndPosition(ctx context.Context) (*spatialmath.Pose, error)

    // MoveToPosition moves end effector to target pose
    MoveToPosition(ctx context.Context, pose *spatialmath.Pose) error

    // JointPositions returns current joint angles
    JointPositions(ctx context.Context) ([]float64, error)

    // MoveToJointPositions sets joint angles directly
    MoveToJointPositions(ctx context.Context, positions []float64) error
}
```

---

## Component Categories

| Category | Interface | Purpose |
|----------|-----------|---------|
| **Sensor** | `Readings()` | Observe the world (read-only) |
| **Actuator** | `IsMoving()`, `Stop()` | Change the world |
| **Power** | `GetCapacity()`, `GetLevel()` | Manage energy |
| **Space** | `GetVolume()`, `GetComponents()` | Virtual containers |
| **Link** | `IsConnected()`, `GetStats()` | Extra communication channels |

---

## Concurrency Model

Gorai uses a **single-owner model** for components:

1. **Each component instance is owned by one goroutine**
   - No mutex protection needed for component state
   - Simpler implementations without synchronization overhead

2. **Cross-goroutine coordination uses NATS messaging**
   - Don't share memory between goroutines
   - Publish state changes to NATS topics
   - Subscribe to receive updates from other components

3. **Benefits**
   - Simpler code (no locks, no deadlocks)
   - Better performance (no synchronization overhead)
   - Natural fit for distributed systems
   - State changes are traceable through message flow

```go
// Single-owner component - no mutex needed
type FakeMotor struct {
    name     resource.Name
    power    float64       // Only accessed by owner goroutine
    moving   bool
    position float64
}

func (m *FakeMotor) SetPower(ctx context.Context, power float64) error {
    m.power = power
    m.moving = power != 0
    return nil
}
```

### When to Use Mutexes

Use mutexes only for:
- Statistics counters accessed from multiple goroutines
- Configuration caches with rare writes
- Connection pools or shared infrastructure

Component state (position, velocity, sensor readings) should follow single-owner semantics.

---

## Registry

Components are registered with the global registry:

```go
// In init() function
func init() {
    registry.RegisterComponent("motor", "fake", func(ctx context.Context, deps resource.Dependencies, conf registry.Config) (interface{}, error) {
        return fake.New(conf), nil
    })
}

// Lookup and create
ctor, err := registry.LookupComponent("motor", "fake")
motor, err := ctor(ctx, deps, config)
```

### Registered Component Types

| Type | Models |
|------|--------|
| `imu` | fake |
| `ahrs` | fake |
| `gps` | fake |
| `encoder` | fake |
| `range_sensor` | fake |
| `lidar` | fake |
| `presence_sensor` | fake |
| `thermal_array` | fake |
| `force_sensor` | fake |
| `force_6dof` | fake |
| `current_sensor` | fake |
| `reflectance_sensor` | fake |
| `motor` | fake |
| `servo` | fake |
| `stepper` | fake |
| `thruster` | fake |
| `valve` | fake |
