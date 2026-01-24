package fake

import (
	"context"
	"fmt"

	"github.com/gorai/gorai/components/sensor"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("current_sensor", "fake", NewCurrentSensor)
}

// CurrentSensor is a fake current sensor for testing.
type CurrentSensor struct {
	name resource.Name

	// Current readings
	current float64
	voltage float64

	// Capabilities
	supportsVoltage bool
}

// NewCurrentSensor creates a new fake current sensor.
func NewCurrentSensor(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "current_sensor", nameStr)

	supportsVoltage := true
	if v, ok := conf["supports_voltage"].(bool); ok {
		supportsVoltage = v
	}

	return &CurrentSensor{
		name:            name,
		supportsVoltage: supportsVoltage,
	}, nil
}

// NewCurrentSensorWithName creates a fake current sensor with a specific resource name.
func NewCurrentSensorWithName(name resource.Name) *CurrentSensor {
	return &CurrentSensor{
		name:            name,
		supportsVoltage: true,
	}
}

// Name returns the resource name.
func (c *CurrentSensor) Name() resource.Name {
	return c.name
}

// Reconfigure updates the configuration.
func (c *CurrentSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (c *CurrentSensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "set_current":
			if current, ok := cmd["current"].(float64); ok {
				c.current = current
				return map[string]any{"status": "ok"}, nil
			}
		case "get_state":
			return map[string]any{
				"current": c.current,
				"voltage": c.voltage,
				"power":   c.current * c.voltage,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (c *CurrentSensor) Close(ctx context.Context) error {
	return nil
}

// Readings returns all sensor readings as a map.
func (c *CurrentSensor) Readings(ctx context.Context) (map[string]any, error) {
	readings := map[string]any{
		"current": c.current,
	}
	if c.supportsVoltage {
		readings["voltage"] = c.voltage
		readings["power"] = c.current * c.voltage
	}
	return readings, nil
}

// GetCurrent returns the current in Amps.
func (c *CurrentSensor) GetCurrent(ctx context.Context) (float64, error) {
	return c.current, nil
}

// GetVoltage returns the voltage in Volts.
func (c *CurrentSensor) GetVoltage(ctx context.Context) (float64, error) {
	if !c.supportsVoltage {
		return 0, fmt.Errorf("voltage measurement not supported")
	}
	return c.voltage, nil
}

// GetPower returns the power in Watts.
func (c *CurrentSensor) GetPower(ctx context.Context) (float64, error) {
	if !c.supportsVoltage {
		return 0, fmt.Errorf("power calculation not supported (no voltage)")
	}
	return c.current * c.voltage, nil
}

// SetCurrent sets the current value for testing.
func (c *CurrentSensor) SetCurrent(current float64) {
	c.current = current
}

// SetVoltage sets the voltage value for testing.
func (c *CurrentSensor) SetVoltage(voltage float64) {
	c.voltage = voltage
	c.supportsVoltage = true
}

// SetMeasurements sets both current and voltage for testing.
func (c *CurrentSensor) SetMeasurements(current, voltage float64) {
	c.current = current
	c.voltage = voltage
	c.supportsVoltage = true
}

// Verify interface compliance.
var _ sensor.CurrentSensor = (*CurrentSensor)(nil)
