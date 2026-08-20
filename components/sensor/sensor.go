// Package sensor defines generic sensor component interfaces.
package sensor

import (
	"context"
	"time"

	"github.com/emergingrobotics/gorai/components"
)

// Mounting describes how an orientation sensor is physically mounted by mapping
// each body axis to a signed sensor axis. Each field is one of "+x","-x","+y",
// "-y","+z","-z" and names the sensor axis that the given body axis points
// along. The identity mounting is {X:"+x", Y:"+y", Z:"+z"}.
type Mounting struct {
	X string `json:"x"`
	Y string `json:"y"`
	Z string `json:"z"`
}

// OrientationState is a snapshot of an orientation sensor's runtime frame
// configuration.
type OrientationState struct {
	// Mounting is the active axis remap.
	Mounting Mounting `json:"mounting"`

	// OffsetDeg is the configured hardcoded orientation offset in degrees
	// (roll, pitch, yaw). It is used as the zero reference until a calibration
	// overrides it.
	OffsetDeg [3]float64 `json:"offset_deg"`

	// Zeroed is true when a runtime calibration has captured a zero reference
	// that overrides OffsetDeg.
	Zeroed bool `json:"zeroed"`
}

// OrientationConfigurable is implemented by orientation sensors whose reference
// frame can be reconfigured at runtime: a captured zero (calibration), a
// hardcoded offset, and an axis-remap mounting. All operations are runtime-only.
type OrientationConfigurable interface {
	// Calibrate samples orientation over dur and sets the average as the zero
	// reference, overriding any configured offset.
	Calibrate(ctx context.Context, dur time.Duration) error

	// ClearZero removes a calibrated zero, reverting to the configured offset.
	ClearZero(ctx context.Context) error

	// SetMounting sets the axis-remap describing the sensor's mounting.
	SetMounting(ctx context.Context, m Mounting) error

	// SetOffset sets the hardcoded orientation offset in degrees.
	SetOffset(ctx context.Context, rollDeg, pitchDeg, yawDeg float64) error

	// OrientationConfig returns the current frame configuration.
	OrientationConfig(ctx context.Context) (OrientationState, error)
}

// Sensor is a generic sensor that returns key-value readings.
type Sensor interface {
	component.Sensor
}

// IMU represents an inertial measurement unit.
type IMU interface {
	component.Component

	// LinearAcceleration returns acceleration in m/s² (x, y, z).
	LinearAcceleration(ctx context.Context) (x, y, z float64, err error)

	// AngularVelocity returns rotation rate in rad/s (x, y, z).
	AngularVelocity(ctx context.Context) (x, y, z float64, err error)

	// Orientation returns the current orientation as a quaternion (x, y, z, w).
	Orientation(ctx context.Context) (x, y, z, w float64, err error)

	// GetMagneticField returns magnetic field in µT (x, y, z).
	// Returns ErrNotSupported if the IMU does not have a magnetometer.
	GetMagneticField(ctx context.Context) (x, y, z float64, err error)
}

// AHRS (Attitude and Heading Reference System) extends IMU with onboard sensor fusion.
// Examples: BNO055, BNO085, ICM-20948 with DMP.
type AHRS interface {
	IMU

	// GetEulerAngles returns orientation as Euler angles in radians (roll, pitch, yaw).
	GetEulerAngles(ctx context.Context) (roll, pitch, yaw float64, err error)

	// GetQuaternion returns orientation as quaternion (x, y, z, w).
	GetQuaternion(ctx context.Context) (x, y, z, w float64, err error)

	// GetLinearAccelerationWithoutGravity returns acceleration with gravity removed.
	GetLinearAccelerationWithoutGravity(ctx context.Context) (x, y, z float64, err error)

	// GetGravityVector returns the gravity vector in m/s².
	GetGravityVector(ctx context.Context) (x, y, z float64, err error)

	// GetCalibrationStatus returns calibration status (0-3) for each sensor.
	GetCalibrationStatus(ctx context.Context) (sys, gyro, accel, mag uint8, err error)
}

// GPS represents a GPS/GNSS receiver.
type GPS interface {
	component.Component

	// Position returns latitude, longitude in degrees and altitude in meters.
	Position(ctx context.Context) (lat, lng, alt float64, err error)

	// LinearVelocity returns velocity in m/s (x=east, y=north, z=up).
	LinearVelocity(ctx context.Context) (x, y, z float64, err error)

	// Accuracy returns position accuracy in meters (horizontal, vertical).
	Accuracy(ctx context.Context) (horizontal, vertical float64, err error)

	// Fix returns the current fix type.
	Fix(ctx context.Context) (FixType, error)

	// GetHeading returns the heading/course over ground in degrees (0-360).
	GetHeading(ctx context.Context) (float64, error)

	// GetSatellitesUsed returns the number of satellites used in the fix.
	GetSatellitesUsed(ctx context.Context) (int, error)
}

// FixType represents GPS fix quality.
type FixType int

const (
	FixNone FixType = iota
	FixGPS
	FixDGPS
	FixRTK
)

// Encoder represents a rotary or linear encoder.
type Encoder interface {
	component.Component

	// Position returns the current position in ticks or radians.
	Position(ctx context.Context) (float64, error)

	// ResetPosition sets the current position as zero.
	ResetPosition(ctx context.Context) error

	// Properties returns encoder properties.
	Properties(ctx context.Context) (EncoderProperties, error)

	// GetVelocity returns the velocity in counts/sec or rad/s.
	GetVelocity(ctx context.Context) (float64, error)

	// GetResolution returns the encoder resolution (PPR or bits for absolute).
	GetResolution(ctx context.Context) (int, error)
}

// EncoderProperties describes encoder capabilities.
type EncoderProperties struct {
	TicksPerRevolution    int
	AngleDegreesSupported bool
	IsAbsolute            bool // true for absolute encoders, false for incremental
}

// RangeSensor represents a distance sensor.
// Examples: Ultrasonic (HC-SR04), ToF (VL53L0X, VL53L1X), IR rangefinders.
type RangeSensor interface {
	component.Component

	// GetRange returns the measured distance in meters.
	GetRange(ctx context.Context) (float64, error)

	// GetRanges returns multiple range measurements for array sensors.
	// Single-point sensors return a slice with one element.
	GetRanges(ctx context.Context) ([]float64, error)

	// GetMinRange returns the minimum detectable range in meters.
	GetMinRange(ctx context.Context) (float64, error)

	// GetMaxRange returns the maximum detectable range in meters.
	GetMaxRange(ctx context.Context) (float64, error)

	// Properties returns range sensor properties.
	Properties(ctx context.Context) (RangeSensorProperties, error)
}

// RangeSensorProperties describes range sensor capabilities.
type RangeSensorProperties struct {
	MinRange    float64 // meters
	MaxRange    float64 // meters
	FieldOfView float64 // radians
	NumPoints   int     // number of measurement points (1 for single-point sensors)
}

// RangeFinder is an alias for RangeSensor for backward compatibility.
// Deprecated: Use RangeSensor instead.
type RangeFinder = RangeSensor

// RangeFinderProperties is an alias for RangeSensorProperties for backward compatibility.
// Deprecated: Use RangeSensorProperties instead.
type RangeFinderProperties = RangeSensorProperties

// LiDAR represents a 2D or 3D laser scanning sensor.
// Examples: RPLIDAR A1/A3/C1 (2D), Livox, Velodyne (3D).
type LiDAR interface {
	component.Component

	// GetScan returns a 2D laser scan.
	GetScan(ctx context.Context) (*LaserScan, error)

	// GetPointCloud returns a 3D point cloud (for 3D LiDARs).
	// Returns ErrNotSupported for 2D-only sensors.
	GetPointCloud(ctx context.Context) (*PointCloud, error)

	// GetScanRate returns the current scan rate in Hz.
	GetScanRate(ctx context.Context) (float64, error)

	// SetScanMode sets the scanning mode (e.g., "standard", "express", "boost").
	SetScanMode(ctx context.Context, mode string) error

	// GetProperties returns LiDAR properties.
	GetProperties(ctx context.Context) (LiDARProperties, error)
}

// LaserScan represents a 2D laser scan.
type LaserScan struct {
	AngleMin       float64   // Start angle in radians
	AngleMax       float64   // End angle in radians
	AngleIncrement float64   // Angular step in radians
	Ranges         []float64 // Distance measurements in meters
	Intensities    []float64 // Signal intensities (optional, may be nil)
	Timestamp      int64     // Unix timestamp in nanoseconds
}

// PointCloud represents a 3D point cloud.
type PointCloud struct {
	Points    [][3]float64 // (x, y, z) in meters
	Colors    [][3]uint8   // RGB colors (optional, may be nil)
	Timestamp int64        // Unix timestamp in nanoseconds
}

// LiDARProperties describes LiDAR capabilities.
type LiDARProperties struct {
	MinRange          float64 // meters
	MaxRange          float64 // meters
	AngularResolution float64 // degrees
	SampleRate        int     // points per second
	Is3D              bool    // true for 3D LiDARs
}

// PresenceSensor detects presence or motion in an area.
// Examples: PIR sensors (HC-SR501), mmWave radar (LD2410, 24GHz/60GHz modules).
type PresenceSensor interface {
	component.Component

	// IsPresenceDetected returns true if presence is detected.
	IsPresenceDetected(ctx context.Context) (bool, error)

	// GetDistance returns distance to detected target in meters.
	// Returns ErrNotSupported if the sensor cannot measure distance.
	GetDistance(ctx context.Context) (float64, error)

	// GetMotionState returns the current motion state.
	GetMotionState(ctx context.Context) (MotionState, error)
}

// MotionState represents the detected motion state.
type MotionState int

const (
	MotionUnknown MotionState = iota
	MotionStatic              // Stationary presence detected
	MotionMoving              // Moving presence detected
)

// ThermalArray represents a thermal imaging sensor.
// Examples: AMG8833 (8x8), MLX90640 (32x24).
type ThermalArray interface {
	component.Component

	// GetTemperatureGrid returns the temperature grid in °C.
	// The outer slice is rows, inner slice is columns.
	GetTemperatureGrid(ctx context.Context) ([][]float64, error)

	// GetAmbientTemperature returns the sensor's ambient temperature in °C.
	GetAmbientTemperature(ctx context.Context) (float64, error)

	// GetMinMaxTemperature returns the min and max temperatures in the current frame.
	GetMinMaxTemperature(ctx context.Context) (min, max float64, err error)

	// GetResolution returns the grid dimensions (width, height).
	GetResolution(ctx context.Context) (width, height int, err error)
}

// ForceSensor measures force or weight.
// Examples: FSR (Force Sensitive Resistor), load cells with HX711.
type ForceSensor interface {
	component.Component

	// GetForce returns the measured force in Newtons.
	GetForce(ctx context.Context) (float64, error)

	// Tare zeros the sensor (sets current reading as zero).
	Tare(ctx context.Context) error
}

// Force6DOF is a 6-axis force/torque sensor.
// Returns forces (Fx, Fy, Fz) and torques (Tx, Ty, Tz).
type Force6DOF interface {
	ForceSensor

	// GetWrench returns the full 6-DOF force/torque measurement.
	GetWrench(ctx context.Context) (*Wrench, error)
}

// Wrench represents a 6-DOF force/torque measurement.
type Wrench struct {
	ForceX, ForceY, ForceZ    float64 // Newtons
	TorqueX, TorqueY, TorqueZ float64 // Newton-meters
}

// CurrentSensor measures electrical current.
// Examples: ACS712 (Hall effect), INA219 (I2C power monitor).
type CurrentSensor interface {
	component.Component

	// GetCurrent returns the current in Amps.
	GetCurrent(ctx context.Context) (float64, error)

	// GetVoltage returns the voltage in Volts.
	// Returns ErrNotSupported if the sensor only measures current.
	GetVoltage(ctx context.Context) (float64, error)

	// GetPower returns the power in Watts.
	// Returns ErrNotSupported if the sensor cannot calculate power.
	GetPower(ctx context.Context) (float64, error)
}

// ReflectanceSensor measures surface reflectance for line following.
// Examples: QTR-8RC, TCRT5000 arrays.
type ReflectanceSensor interface {
	component.Component

	// GetReflectances returns reflectance values (0.0-1.0) for each channel.
	// Higher values indicate more reflective (lighter) surfaces.
	GetReflectances(ctx context.Context) ([]float64, error)

	// GetLinePosition returns a weighted average position of the line.
	// Returns a value typically in range [-1.0, 1.0] or [0, n-1] depending on implementation.
	GetLinePosition(ctx context.Context) (float64, error)

	// Calibrate performs calibration using current surface readings.
	Calibrate(ctx context.Context) error
}
