package fake

import (
	"context"
	"fmt"

	"github.com/gorai/gorai/components/sensor"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("force_sensor", "fake", NewForceSensor)
	registry.RegisterComponent("force_6dof", "fake", NewForce6DOF)
}

// ForceSensor is a fake force sensor for testing.
type ForceSensor struct {
	name resource.Name

	// Current reading
	force      float64
	tareOffset float64
}

// NewForceSensor creates a new fake force sensor.
func NewForceSensor(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "force_sensor", nameStr)
	return &ForceSensor{name: name}, nil
}

// NewForceSensorWithName creates a fake force sensor with a specific resource name.
func NewForceSensorWithName(name resource.Name) *ForceSensor {
	return &ForceSensor{name: name}
}

// Name returns the resource name.
func (f *ForceSensor) Name() resource.Name {
	return f.name
}

// Reconfigure updates the configuration.
func (f *ForceSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (f *ForceSensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "set_force":
			if force, ok := cmd["force"].(float64); ok {
				f.force = force
				return map[string]any{"status": "ok"}, nil
			}
		case "get_state":
			return map[string]any{
				"force":       f.force,
				"tare_offset": f.tareOffset,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (f *ForceSensor) Close(ctx context.Context) error {
	return nil
}

// Readings returns all sensor readings as a map.
func (f *ForceSensor) Readings(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"force": f.force - f.tareOffset,
	}, nil
}

// GetForce returns the measured force in Newtons.
func (f *ForceSensor) GetForce(ctx context.Context) (float64, error) {
	return f.force - f.tareOffset, nil
}

// Tare zeros the sensor.
func (f *ForceSensor) Tare(ctx context.Context) error {
	f.tareOffset = f.force
	return nil
}

// SetForce sets the force value for testing.
func (f *ForceSensor) SetForce(force float64) {
	f.force = force
}

// Verify interface compliance.
var _ sensor.ForceSensor = (*ForceSensor)(nil)

// Force6DOF is a fake 6-axis force/torque sensor for testing.
type Force6DOF struct {
	ForceSensor

	// 6-DOF wrench
	wrench sensor.Wrench
}

// NewForce6DOF creates a new fake 6-DOF force sensor.
func NewForce6DOF(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "force_6dof", nameStr)
	return &Force6DOF{
		ForceSensor: ForceSensor{name: name},
	}, nil
}

// NewForce6DOFWithName creates a fake 6-DOF force sensor with a specific resource name.
func NewForce6DOFWithName(name resource.Name) *Force6DOF {
	return &Force6DOF{
		ForceSensor: ForceSensor{name: name},
	}
}

// GetWrench returns the full 6-DOF force/torque measurement.
func (f *Force6DOF) GetWrench(ctx context.Context) (*sensor.Wrench, error) {
	return &f.wrench, nil
}

// GetForce returns the Z-axis force (for compatibility with ForceSensor interface).
func (f *Force6DOF) GetForce(ctx context.Context) (float64, error) {
	return f.wrench.ForceZ - f.tareOffset, nil
}

// Readings returns all sensor readings as a map.
func (f *Force6DOF) Readings(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"force_x":  f.wrench.ForceX,
		"force_y":  f.wrench.ForceY,
		"force_z":  f.wrench.ForceZ,
		"torque_x": f.wrench.TorqueX,
		"torque_y": f.wrench.TorqueY,
		"torque_z": f.wrench.TorqueZ,
	}, nil
}

// SetWrench sets the wrench values for testing.
func (f *Force6DOF) SetWrench(fx, fy, fz, tx, ty, tz float64) {
	f.wrench = sensor.Wrench{
		ForceX:  fx,
		ForceY:  fy,
		ForceZ:  fz,
		TorqueX: tx,
		TorqueY: ty,
		TorqueZ: tz,
	}
	f.force = fz // Set Z-force as the primary force
}

// Verify interface compliance.
var _ sensor.Force6DOF = (*Force6DOF)(nil)
