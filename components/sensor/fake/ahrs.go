package fake

import (
	"context"
	"math"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("ahrs", "fake", NewAHRS)
}

// AHRS is a fake AHRS sensor for testing.
type AHRS struct {
	IMU // Embed IMU for base functionality

	// Additional AHRS-specific values
	Roll, Pitch, Yaw       float64
	LinearAccelNoGravX     float64
	LinearAccelNoGravY     float64
	LinearAccelNoGravZ     float64
	GravityX, GravityY     float64
	GravityZ               float64
	CalibSys, CalibGyro    uint8
	CalibAccel, CalibMag   uint8
}

// NewAHRS creates a new fake AHRS.
func NewAHRS(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "ahrs", nameStr)
	return &AHRS{
		IMU: IMU{
			name:    name,
			OrientW: 1.0,
		},
		GravityZ:   9.81, // Default gravity pointing down
		CalibSys:   3,    // Fully calibrated
		CalibGyro:  3,
		CalibAccel: 3,
		CalibMag:   3,
	}, nil
}

// NewAHRSWithName creates a fake AHRS with a specific resource name.
func NewAHRSWithName(name resource.Name) *AHRS {
	return &AHRS{
		IMU:        IMU{name: name, OrientW: 1.0},
		GravityZ:   9.81,
		CalibSys:   3,
		CalibGyro:  3,
		CalibAccel: 3,
		CalibMag:   3,
	}
}

// DoCommand executes arbitrary commands.
func (a *AHRS) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_calibration":
			return map[string]any{
				"sys":   a.CalibSys,
				"gyro":  a.CalibGyro,
				"accel": a.CalibAccel,
				"mag":   a.CalibMag,
			}, nil
		}
	}
	return a.IMU.DoCommand(ctx, cmd)
}

// Readings returns all sensor readings as a map.
func (a *AHRS) Readings(ctx context.Context) (map[string]any, error) {
	readings, _ := a.IMU.Readings(ctx)
	readings["euler_angles"] = map[string]float64{"roll": a.Roll, "pitch": a.Pitch, "yaw": a.Yaw}
	readings["gravity"] = map[string]float64{"x": a.GravityX, "y": a.GravityY, "z": a.GravityZ}
	readings["calibration"] = map[string]uint8{
		"sys": a.CalibSys, "gyro": a.CalibGyro, "accel": a.CalibAccel, "mag": a.CalibMag,
	}
	return readings, nil
}

// GetEulerAngles returns orientation as Euler angles in radians.
func (a *AHRS) GetEulerAngles(ctx context.Context) (roll, pitch, yaw float64, err error) {
	return a.Roll, a.Pitch, a.Yaw, nil
}

// GetQuaternion returns orientation as quaternion.
func (a *AHRS) GetQuaternion(ctx context.Context) (x, y, z, w float64, err error) {
	return a.OrientX, a.OrientY, a.OrientZ, a.OrientW, nil
}

// GetLinearAccelerationWithoutGravity returns acceleration with gravity removed.
func (a *AHRS) GetLinearAccelerationWithoutGravity(ctx context.Context) (x, y, z float64, err error) {
	return a.LinearAccelNoGravX, a.LinearAccelNoGravY, a.LinearAccelNoGravZ, nil
}

// GetGravityVector returns the gravity vector.
func (a *AHRS) GetGravityVector(ctx context.Context) (x, y, z float64, err error) {
	return a.GravityX, a.GravityY, a.GravityZ, nil
}

// GetCalibrationStatus returns calibration status (0-3) for each sensor.
func (a *AHRS) GetCalibrationStatus(ctx context.Context) (sys, gyro, accel, mag uint8, err error) {
	return a.CalibSys, a.CalibGyro, a.CalibAccel, a.CalibMag, nil
}

// SetEulerAngles sets the Euler angles for testing.
func (a *AHRS) SetEulerAngles(roll, pitch, yaw float64) {
	a.Roll, a.Pitch, a.Yaw = roll, pitch, yaw
	// Also update quaternion for consistency
	a.updateQuaternionFromEuler()
}

// SetCalibrationStatus sets calibration values for testing.
func (a *AHRS) SetCalibrationStatus(sys, gyro, accel, mag uint8) {
	a.CalibSys, a.CalibGyro, a.CalibAccel, a.CalibMag = sys, gyro, accel, mag
}

// SetGravityVector sets the gravity vector for testing.
func (a *AHRS) SetGravityVector(x, y, z float64) {
	a.GravityX, a.GravityY, a.GravityZ = x, y, z
}

// updateQuaternionFromEuler converts Euler angles to quaternion.
func (a *AHRS) updateQuaternionFromEuler() {
	// ZYX Euler angle to quaternion conversion
	cr := math.Cos(a.Roll / 2)
	sr := math.Sin(a.Roll / 2)
	cp := math.Cos(a.Pitch / 2)
	sp := math.Sin(a.Pitch / 2)
	cy := math.Cos(a.Yaw / 2)
	sy := math.Sin(a.Yaw / 2)

	a.OrientW = cr*cp*cy + sr*sp*sy
	a.OrientX = sr*cp*cy - cr*sp*sy
	a.OrientY = cr*sp*cy + sr*cp*sy
	a.OrientZ = cr*cp*sy - sr*sp*cy
}

// Verify interface compliance.
var _ sensor.AHRS = (*AHRS)(nil)
