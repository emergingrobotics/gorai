# Plan: Update Sensors and Motors to Match Specification

## Overview

This plan updates the Gorai codebase to align with the expanded sensor and actuator specifications documented in `specs/gorai-framework-specification.md`. The specification was updated based on comprehensive research in `docs/sensor-analysis.md` and `docs/motor-analysis.md`.

## Current State Analysis

### Existing Code Structure

```
component/
├── sensor/sensor.go          # IMU, GPS, Encoder, RangeFinder interfaces
├── motor/motor.go            # Generic Motor interface
│   └── fake/fake.go          # Fake motor implementation
├── camera/camera.go          # Camera interface
├── gripper/gripper.go        # Gripper interface
├── arm/arm.go                # Arm interface
├── base/base.go              # Mobile base interface
├── power/power.go            # Power interface
├── space/space.go            # Space interface
└── link/link.go              # Link interface (Serial, Radio)

pkg/resource/resource.go      # Base interfaces (Resource, Sensor, Actuator, etc.)
```

### Gap Analysis: Sensors

| Spec Interface | Current Code | Status | Action |
|----------------|--------------|--------|--------|
| `IMU` | `sensor.IMU` | Partial | Add magnetometer, enhance |
| `AHRS` | Missing | **NEW** | Create interface |
| `GPS` | `sensor.GPS` | Partial | Add satellites, fix quality |
| `Encoder` | `sensor.Encoder` | Partial | Add velocity, resolution |
| `RangeSensor` | `sensor.RangeFinder` | Partial | Rename, add min/max |
| `LiDAR` | Missing | **NEW** | Create interface |
| `PresenceSensor` | Missing | **NEW** | Create interface |
| `ThermalArray` | Missing | **NEW** | Create interface |
| `ForceSensor` | Missing | **NEW** | Create interface |
| `Force6DOF` | Missing | **NEW** | Create interface |
| `CurrentSensor` | Missing | **NEW** | Create interface |
| `ReflectanceSensor` | Missing | **NEW** | Create interface |

### Gap Analysis: Actuators

| Spec Interface | Current Code | Status | Action |
|----------------|--------------|--------|--------|
| `Motor` | `motor.Motor` | OK | Minor updates |
| `Servo` | Missing | **NEW** | Create interface + package |
| `Stepper` | Missing | **NEW** | Create interface + package |
| `Thruster` | Missing | **NEW** | Create interface + package |
| `Valve` | Missing | **NEW** | Create interface + package |
| `Gripper` | `gripper.Gripper` | OK | No changes |
| `Arm` | `arm.Arm` | OK | No changes |
| `Base` | `base.Base` | OK | No changes |

---

## Implementation Plan

### Phase 1: Update pkg/resource Interfaces

**Goal**: Ensure base interfaces in `pkg/resource/resource.go` match the specification.

#### 1.1 Review and Update Base Interfaces

The base `Sensor` and `Actuator` interfaces are already correct:

```go
// Already correct in pkg/resource/resource.go
type Sensor interface {
    Resource
    Readings(ctx context.Context) (map[string]any, error)
}

type Actuator interface {
    Resource
    IsMoving(ctx context.Context) (bool, error)
    Stop(ctx context.Context) error
}
```

**No changes needed** to base interfaces.

---

### Phase 2: Expand Sensor Interfaces

**Goal**: Update `components/sensor/sensor.go` with all new sensor types from the specification.

#### 2.1 Enhance Existing Interfaces

**IMU** - Add magnetometer support:
```go
type IMU interface {
    component.Component

    // LinearAcceleration returns acceleration in m/s² (x, y, z).
    LinearAcceleration(ctx context.Context) (x, y, z float64, err error)

    // AngularVelocity returns rotation rate in rad/s (x, y, z).
    AngularVelocity(ctx context.Context) (x, y, z float64, err error)

    // Orientation returns orientation as quaternion (x, y, z, w).
    Orientation(ctx context.Context) (x, y, z, w float64, err error)

    // NEW: GetMagneticField returns magnetic field in µT (x, y, z).
    GetMagneticField(ctx context.Context) (x, y, z float64, err error)
}
```

**GPS** - Add fix quality and satellite count:
```go
type GPS interface {
    component.Component

    Position(ctx context.Context) (lat, lng, alt float64, err error)
    LinearVelocity(ctx context.Context) (x, y, z float64, err error)
    Accuracy(ctx context.Context) (horizontal, vertical float64, err error)
    Fix(ctx context.Context) (FixType, error)

    // NEW methods
    GetHeading(ctx context.Context) (float64, error)        // degrees
    GetSatellitesUsed(ctx context.Context) (int, error)
}
```

**Encoder** - Add velocity and resolution:
```go
type Encoder interface {
    component.Component

    Position(ctx context.Context) (float64, error)
    ResetPosition(ctx context.Context) error
    Properties(ctx context.Context) (EncoderProperties, error)

    // NEW methods
    GetVelocity(ctx context.Context) (float64, error)       // counts/sec or rad/s
    GetResolution(ctx context.Context) (int, error)         // PPR or bits
}

type EncoderProperties struct {
    TicksPerRevolution    int
    AngleDegreesSupported bool
    IsAbsolute            bool    // NEW: absolute vs incremental
}
```

**RangeSensor** (rename from RangeFinder):
```go
type RangeSensor interface {
    component.Component

    GetRange(ctx context.Context) (float64, error)          // meters
    GetRanges(ctx context.Context) ([]float64, error)       // for array sensors
    GetMinRange(ctx context.Context) (float64, error)
    GetMaxRange(ctx context.Context) (float64, error)
}
```

#### 2.2 Add New Sensor Interfaces

**AHRS** (Attitude and Heading Reference System):
```go
// AHRS extends IMU with onboard sensor fusion (e.g., BNO055).
type AHRS interface {
    IMU

    GetEulerAngles(ctx context.Context) (roll, pitch, yaw float64, err error)
    GetQuaternion(ctx context.Context) (x, y, z, w float64, err error)
    GetLinearAccelerationWithoutGravity(ctx context.Context) (x, y, z float64, err error)
    GetGravityVector(ctx context.Context) (x, y, z float64, err error)
    GetCalibrationStatus(ctx context.Context) (sys, gyro, accel, mag uint8, err error)
}
```

**LiDAR**:
```go
// LiDAR for 2D/3D laser scanning (RPLIDAR, etc.).
type LiDAR interface {
    component.Component

    GetScan(ctx context.Context) (*LaserScan, error)
    GetPointCloud(ctx context.Context) (*PointCloud, error)  // for 3D
    GetScanRate(ctx context.Context) (float64, error)        // Hz
    SetScanMode(ctx context.Context, mode string) error
    GetProperties(ctx context.Context) (LiDARProperties, error)
}

type LaserScan struct {
    AngleMin      float64   // rad
    AngleMax      float64   // rad
    AngleIncrement float64  // rad
    Ranges        []float64 // meters
    Intensities   []float64 // optional
    Timestamp     int64     // nanoseconds
}

type LiDARProperties struct {
    MinRange          float64
    MaxRange          float64
    AngularResolution float64 // degrees
    SampleRate        int     // points/second
    Is3D              bool
}
```

**PresenceSensor**:
```go
// PresenceSensor for PIR and mmWave presence detection.
type PresenceSensor interface {
    component.Component

    IsPresenceDetected(ctx context.Context) (bool, error)
    GetDistance(ctx context.Context) (float64, error)           // meters, if supported
    GetMotionState(ctx context.Context) (MotionState, error)
}

type MotionState int
const (
    MotionUnknown MotionState = iota
    MotionStatic
    MotionMoving
)
```

**ThermalArray**:
```go
// ThermalArray for thermal imaging (AMG8833, MLX90640).
type ThermalArray interface {
    component.Component

    GetTemperatureGrid(ctx context.Context) ([][]float64, error)  // °C
    GetAmbientTemperature(ctx context.Context) (float64, error)
    GetMinMaxTemperature(ctx context.Context) (min, max float64, err error)
    GetResolution(ctx context.Context) (width, height int, err error)
}
```

**ForceSensor / Force6DOF**:
```go
// ForceSensor for force/torque measurement.
type ForceSensor interface {
    component.Component

    GetForce(ctx context.Context) (float64, error)  // Newtons
    Tare(ctx context.Context) error
}

// Force6DOF for 6-axis force/torque sensors.
type Force6DOF interface {
    ForceSensor

    GetWrench(ctx context.Context) (*Wrench, error)
}

type Wrench struct {
    ForceX, ForceY, ForceZ    float64  // Newtons
    TorqueX, TorqueY, TorqueZ float64  // Nm
}
```

**CurrentSensor**:
```go
// CurrentSensor for electrical current monitoring.
type CurrentSensor interface {
    component.Component

    GetCurrent(ctx context.Context) (float64, error)   // Amps
    GetVoltage(ctx context.Context) (float64, error)   // Volts, if supported
    GetPower(ctx context.Context) (float64, error)     // Watts, if supported
}
```

**ReflectanceSensor**:
```go
// ReflectanceSensor for line following (QTR-8RC, etc.).
type ReflectanceSensor interface {
    component.Component

    GetReflectances(ctx context.Context) ([]float64, error)  // 0.0-1.0 per channel
    GetLinePosition(ctx context.Context) (float64, error)    // weighted average
    Calibrate(ctx context.Context) error
}
```

---

### Phase 3: Create New Actuator Packages

**Goal**: Create new packages for specialized actuators.

#### 3.1 Create `components/servo/servo.go`

```go
package servo

import (
    "context"
    "github.com/gorai/gorai/components"
)

// Servo for position-controlled motors (RC servos, Dynamixel, etc.).
type Servo interface {
    component.Actuator

    SetAngle(ctx context.Context, degrees float64) error
    GetAngle(ctx context.Context) (float64, error)
    SetSpeed(ctx context.Context, speed float64) error
    SetTorqueLimit(ctx context.Context, limit float64) error
    GetProperties(ctx context.Context) (Properties, error)
}

type Properties struct {
    MinAngle     float64
    MaxAngle     float64
    IsContinuous bool
    HasFeedback  bool
    Protocol     string  // "pwm", "dynamixel", "lx16a", "feetech"
}
```

#### 3.2 Create `components/stepper/stepper.go`

```go
package stepper

import (
    "context"
    "github.com/gorai/gorai/components"
)

// Stepper for discrete-step motors (NEMA 17, etc.).
type Stepper interface {
    component.Actuator

    Step(ctx context.Context, steps int64, direction bool) error
    SetMicrostepping(ctx context.Context, divisor int) error
    SetCurrent(ctx context.Context, runMA, holdMA int) error
    GetPosition(ctx context.Context) (int64, error)
    ResetPosition(ctx context.Context) error
    Home(ctx context.Context, direction bool) error
    GetProperties(ctx context.Context) (Properties, error)
}

type Properties struct {
    StepsPerRevolution int
    MaxMicrostepping   int
    MaxCurrent         int
    HasStallDetection  bool
    Driver             string  // "a4988", "drv8825", "tmc2209", "tmc5160"
}
```

#### 3.3 Create `components/thruster/thruster.go`

```go
package thruster

import (
    "context"
    "github.com/gorai/gorai/components"
)

// Thruster for underwater propulsion (BlueRobotics, etc.).
type Thruster interface {
    component.Actuator

    SetThrust(ctx context.Context, thrust float64) error  // -1.0 to 1.0
    GetRPM(ctx context.Context) (int, error)
    GetTemperature(ctx context.Context) (float64, error)
    GetCurrent(ctx context.Context) (float64, error)
    GetProperties(ctx context.Context) (Properties, error)
}

type Properties struct {
    MaxThrustForward  float64
    MaxThrustReverse  float64
    DeadbandWidth     float64
    IsBidirectional   bool
    HasTelemetry      bool
    Protocol          string  // "pwm", "i2c", "can"
}
```

#### 3.4 Create `components/valve/valve.go`

```go
package valve

import (
    "context"
    "github.com/gorai/gorai/components"
)

// Valve for fluid control actuators.
type Valve interface {
    component.Actuator

    Open(ctx context.Context) error
    Close(ctx context.Context) error
    SetPosition(ctx context.Context, position float64) error  // 0.0-1.0
    GetPosition(ctx context.Context) (float64, error)
    IsOpen(ctx context.Context) (bool, error)
    IsClosed(ctx context.Context) (bool, error)
}
```

---

### Phase 4: Create Fake Implementations

**Goal**: Create fake implementations for testing all new interfaces.

#### Directory Structure
```
component/
├── sensor/
│   └── fake/
│       ├── imu.go
│       ├── ahrs.go
│       ├── gps.go
│       ├── encoder.go
│       ├── lidar.go
│       ├── presence.go
│       ├── thermal.go
│       ├── force.go
│       ├── current.go
│       ├── reflectance.go
│       └── range.go
├── servo/
│   └── fake/
│       └── fake.go
├── stepper/
│   └── fake/
│       └── fake.go
├── thruster/
│   └── fake/
│       └── fake.go
└── valve/
    └── fake/
        └── fake.go
```

Each fake implementation should follow these patterns:
- **Single-owner model** (no mutex needed) - components are owned by one goroutine
- If cross-goroutine coordination is needed, use NATS messaging instead of shared memory
- Configurable via `registry.Config`
- Test helper methods (setters)
- `DoCommand()` support for extensibility
- Interface compliance verification: `var _ Interface = (*Implementation)(nil)`

---

### Phase 5: Update Tests

**Goal**: Comprehensive test coverage for all new interfaces.

#### 5.1 Sensor Tests (`components/sensor/sensor_test.go`)

- Test all new interface method signatures
- Test fake implementations
- Test configuration and reconfiguration
- Test edge cases (nil values, error conditions)

#### 5.2 Actuator Tests

Create test files for each new actuator package:
- `components/servo/servo_test.go`
- `components/stepper/stepper_test.go`
- `components/thruster/thruster_test.go`
- `components/valve/valve_test.go`

---

### Phase 6: Update Registry

**Goal**: Register all new component types.

#### 6.1 Update `pkg/registry/registry.go`

Ensure all new component subtypes are registrable:
```go
// Sensor subtypes
registry.RegisterComponent("imu", "fake", fake.NewIMU)
registry.RegisterComponent("ahrs", "fake", fake.NewAHRS)
registry.RegisterComponent("gps", "fake", fake.NewGPS)
registry.RegisterComponent("lidar", "fake", fake.NewLiDAR)
registry.RegisterComponent("presence_sensor", "fake", fake.NewPresence)
registry.RegisterComponent("thermal_array", "fake", fake.NewThermal)
registry.RegisterComponent("force_sensor", "fake", fake.NewForce)
registry.RegisterComponent("current_sensor", "fake", fake.NewCurrent)
registry.RegisterComponent("reflectance_sensor", "fake", fake.NewReflectance)
registry.RegisterComponent("range_sensor", "fake", fake.NewRange)
registry.RegisterComponent("encoder", "fake", fake.NewEncoder)

// Actuator subtypes
registry.RegisterComponent("servo", "fake", fake.NewServo)
registry.RegisterComponent("stepper", "fake", fake.NewStepper)
registry.RegisterComponent("thruster", "fake", fake.NewThruster)
registry.RegisterComponent("valve", "fake", fake.NewValve)
```

---

## Implementation Order

### Recommended Sequence

1. **Phase 2.1**: Update existing sensor interfaces (IMU, GPS, Encoder, RangeSensor)
2. **Phase 2.2**: Add new sensor interfaces (AHRS, LiDAR, etc.)
3. **Phase 4 (partial)**: Create sensor fake implementations
4. **Phase 5 (partial)**: Create sensor tests
5. **Phase 3**: Create new actuator packages (servo, stepper, thruster, valve)
6. **Phase 4 (continued)**: Create actuator fake implementations
7. **Phase 5 (continued)**: Create actuator tests
8. **Phase 6**: Update registry

### Rationale

- Start with sensors since they are more numerous and have more interface changes
- Build fake implementations alongside interfaces for immediate testability
- Complete one category before moving to the next to maintain focus
- Registry updates last since they depend on all fake implementations existing

---

## Concurrency Model

Gorai uses a **single-owner model** for components:
- Each component instance is owned by one goroutine
- No mutex protection needed for component state
- If cross-goroutine coordination is required, use NATS messaging

This simplifies implementations and aligns with Go's "share memory by communicating" philosophy.

**Note**: Existing fake implementations (`components/motor/fake/fake.go`, `components/link/fake/fake.go`, etc.) currently use `sync.RWMutex`. These should be refactored to remove the mutex boilerplate, either as part of this work or as a separate cleanup task.

---

## Backward Compatibility Considerations

### Breaking Changes

1. **Rename `RangeFinder` → `RangeSensor`**
   - Search and replace all usages
   - Update imports

2. **`Encoder.Properties` struct changes**
   - Add `IsAbsolute` field (non-breaking, zero value is false)

3. **`GPS` interface additions**
   - New methods `GetHeading()`, `GetSatellitesUsed()` require implementation
   - Existing implementations will need updates

### Non-Breaking Changes

- All new interfaces are additive
- All new packages are additive
- New fake implementations don't affect existing code

---

## Files to Create/Modify

### New Files (Create)

| File | Description |
|------|-------------|
| `components/servo/servo.go` | Servo interface definition |
| `components/servo/fake/fake.go` | Fake servo implementation |
| `components/servo/servo_test.go` | Servo tests |
| `components/stepper/stepper.go` | Stepper interface definition |
| `components/stepper/fake/fake.go` | Fake stepper implementation |
| `components/stepper/stepper_test.go` | Stepper tests |
| `components/thruster/thruster.go` | Thruster interface definition |
| `components/thruster/fake/fake.go` | Fake thruster implementation |
| `components/thruster/thruster_test.go` | Thruster tests |
| `components/valve/valve.go` | Valve interface definition |
| `components/valve/fake/fake.go` | Fake valve implementation |
| `components/valve/valve_test.go` | Valve tests |
| `components/sensor/fake/*.go` | Multiple fake sensor files |

### Existing Files (Modify)

| File | Changes |
|------|---------|
| `components/sensor/sensor.go` | Add new interfaces, enhance existing |
| `components/sensor/sensor_test.go` | Add tests for new interfaces |
| `pkg/registry/registry.go` | Register new component types |

---

## Estimated Effort

| Phase | Complexity | Estimated Effort |
|-------|------------|------------------|
| Phase 1 | Low | Review only |
| Phase 2.1 | Low | ~100 lines |
| Phase 2.2 | Medium | ~300 lines |
| Phase 3 | Medium | ~200 lines (4 packages) |
| Phase 4 | High | ~800 lines (all fakes) |
| Phase 5 | Medium | ~400 lines (all tests) |
| Phase 6 | Low | ~50 lines |
| **Total** | | ~1850 lines |

---

## Verification Checklist

- [ ] All interfaces from spec exist in code
- [ ] All interfaces have fake implementations
- [ ] All fake implementations pass interface compliance checks
- [ ] All new code has test coverage
- [ ] Registry can instantiate all fake components
- [ ] Existing tests still pass
- [ ] Documentation updated (godoc comments)

---

## Open Questions for Review

1. Should we keep `RangeFinder` as an alias for backward compatibility?
2. Should sensor fake implementations be in separate files or one consolidated `fake.go`?
3. Should `PointCloud` be a shared type in a geometry package or defined per-sensor?
4. Do we need Properties structs for all sensors, or only where hardware varies significantly?
