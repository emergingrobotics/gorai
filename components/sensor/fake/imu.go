// Package fake provides fake sensor implementations for testing.
package fake

import (
	"context"
	"fmt"

	"github.com/gorai/gorai/components/sensor"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("imu", "fake", NewIMU)
}

// IMU is a fake IMU sensor for testing.
type IMU struct {
	name resource.Name

	// Configurable values for testing
	AccelX, AccelY, AccelZ float64
	GyroX, GyroY, GyroZ    float64
	OrientX, OrientY       float64
	OrientZ, OrientW       float64
	MagX, MagY, MagZ       float64
}

// NewIMU creates a new fake IMU.
func NewIMU(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "imu", nameStr)
	return &IMU{
		name:    name,
		OrientW: 1.0, // Identity quaternion
	}, nil
}

// NewIMUWithName creates a fake IMU with a specific resource name.
func NewIMUWithName(name resource.Name) *IMU {
	return &IMU{name: name, OrientW: 1.0}
}

// Name returns the resource name.
func (i *IMU) Name() resource.Name {
	return i.name
}

// Reconfigure updates the configuration.
func (i *IMU) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (i *IMU) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_state":
			return map[string]any{
				"accel": []float64{i.AccelX, i.AccelY, i.AccelZ},
				"gyro":  []float64{i.GyroX, i.GyroY, i.GyroZ},
				"mag":   []float64{i.MagX, i.MagY, i.MagZ},
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (i *IMU) Close(ctx context.Context) error {
	return nil
}

// Readings returns all sensor readings as a map.
func (i *IMU) Readings(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"linear_acceleration": map[string]float64{"x": i.AccelX, "y": i.AccelY, "z": i.AccelZ},
		"angular_velocity":    map[string]float64{"x": i.GyroX, "y": i.GyroY, "z": i.GyroZ},
		"orientation":         map[string]float64{"x": i.OrientX, "y": i.OrientY, "z": i.OrientZ, "w": i.OrientW},
		"magnetic_field":      map[string]float64{"x": i.MagX, "y": i.MagY, "z": i.MagZ},
	}, nil
}

// LinearAcceleration returns acceleration in m/s².
func (i *IMU) LinearAcceleration(ctx context.Context) (x, y, z float64, err error) {
	return i.AccelX, i.AccelY, i.AccelZ, nil
}

// AngularVelocity returns rotation rate in rad/s.
func (i *IMU) AngularVelocity(ctx context.Context) (x, y, z float64, err error) {
	return i.GyroX, i.GyroY, i.GyroZ, nil
}

// Orientation returns orientation as a quaternion.
func (i *IMU) Orientation(ctx context.Context) (x, y, z, w float64, err error) {
	return i.OrientX, i.OrientY, i.OrientZ, i.OrientW, nil
}

// GetMagneticField returns magnetic field in µT.
func (i *IMU) GetMagneticField(ctx context.Context) (x, y, z float64, err error) {
	return i.MagX, i.MagY, i.MagZ, nil
}

// SetAcceleration sets the acceleration values for testing.
func (i *IMU) SetAcceleration(x, y, z float64) {
	i.AccelX, i.AccelY, i.AccelZ = x, y, z
}

// SetAngularVelocity sets the angular velocity values for testing.
func (i *IMU) SetAngularVelocity(x, y, z float64) {
	i.GyroX, i.GyroY, i.GyroZ = x, y, z
}

// SetOrientation sets the orientation values for testing.
func (i *IMU) SetOrientation(x, y, z, w float64) {
	i.OrientX, i.OrientY, i.OrientZ, i.OrientW = x, y, z, w
}

// SetMagneticField sets the magnetic field values for testing.
func (i *IMU) SetMagneticField(x, y, z float64) {
	i.MagX, i.MagY, i.MagZ = x, y, z
}

// Verify interface compliance.
var _ sensor.IMU = (*IMU)(nil)
