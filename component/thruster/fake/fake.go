// Package fake provides a fake thruster implementation for testing.
package fake

import (
	"context"
	"fmt"

	"github.com/gorai/gorai/component/thruster"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("thruster", "fake", New)
}

// Thruster is a fake thruster for testing.
type Thruster struct {
	name resource.Name

	// Current state
	thrust      float64
	rpm         int
	temperature float64
	current     float64
	moving      bool

	// Properties
	maxThrustForward float64
	maxThrustReverse float64
	deadbandWidth    float64
	isBidirectional  bool
	hasTelemetry     bool
	protocol         string
}

// New creates a new fake thruster.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "thruster", nameStr)

	maxForward := 5.0 // kg-force
	if v, ok := conf["max_thrust_forward"].(float64); ok {
		maxForward = v
	}

	maxReverse := 3.5
	if v, ok := conf["max_thrust_reverse"].(float64); ok {
		maxReverse = v
	}

	return &Thruster{
		name:             name,
		temperature:      25.0, // Ambient
		maxThrustForward: maxForward,
		maxThrustReverse: maxReverse,
		deadbandWidth:    0.05, // 5% deadband
		isBidirectional:  true,
		hasTelemetry:     true,
		protocol:         "pwm",
	}, nil
}

// NewWithName creates a fake thruster with a specific resource name.
func NewWithName(name resource.Name) *Thruster {
	return &Thruster{
		name:             name,
		temperature:      25.0,
		maxThrustForward: 5.0,
		maxThrustReverse: 3.5,
		deadbandWidth:    0.05,
		isBidirectional:  true,
		hasTelemetry:     true,
		protocol:         "pwm",
	}
}

// Name returns the resource name.
func (t *Thruster) Name() resource.Name {
	return t.name
}

// Reconfigure updates the configuration.
func (t *Thruster) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (t *Thruster) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_state":
			return map[string]any{
				"thrust":      t.thrust,
				"rpm":         t.rpm,
				"temperature": t.temperature,
				"current":     t.current,
				"moving":      t.moving,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (t *Thruster) Close(ctx context.Context) error {
	return t.Stop(ctx)
}

// IsMoving returns whether the thruster is running.
func (t *Thruster) IsMoving(ctx context.Context) (bool, error) {
	return t.moving, nil
}

// Stop stops the thruster.
func (t *Thruster) Stop(ctx context.Context) error {
	t.thrust = 0
	t.rpm = 0
	t.current = 0
	t.moving = false
	return nil
}

// SetThrust sets the thrust level (-1.0 to 1.0).
func (t *Thruster) SetThrust(ctx context.Context, thrust float64) error {
	if thrust < -1.0 || thrust > 1.0 {
		return fmt.Errorf("thrust must be -1.0 to 1.0, got %v", thrust)
	}

	// Apply deadband
	if thrust > -t.deadbandWidth && thrust < t.deadbandWidth {
		thrust = 0
	}

	t.thrust = thrust
	t.moving = thrust != 0

	// Simulate RPM and current based on thrust
	if thrust > 0 {
		t.rpm = int(thrust * 3000) // Max ~3000 RPM
	} else {
		t.rpm = int(-thrust * 2500) // Slightly lower reverse
	}
	t.current = absFloat(thrust) * 15.0 // Max ~15A

	return nil
}

// GetRPM returns the current motor RPM.
func (t *Thruster) GetRPM(ctx context.Context) (int, error) {
	if !t.hasTelemetry {
		return 0, fmt.Errorf("RPM telemetry not supported")
	}
	return t.rpm, nil
}

// GetTemperature returns the motor/ESC temperature.
func (t *Thruster) GetTemperature(ctx context.Context) (float64, error) {
	if !t.hasTelemetry {
		return 0, fmt.Errorf("temperature telemetry not supported")
	}
	return t.temperature, nil
}

// GetCurrent returns the current draw.
func (t *Thruster) GetCurrent(ctx context.Context) (float64, error) {
	if !t.hasTelemetry {
		return 0, fmt.Errorf("current telemetry not supported")
	}
	return t.current, nil
}

// GetProperties returns the thruster properties.
func (t *Thruster) GetProperties(ctx context.Context) (thruster.Properties, error) {
	return thruster.Properties{
		MaxThrustForward: t.maxThrustForward,
		MaxThrustReverse: t.maxThrustReverse,
		DeadbandWidth:    t.deadbandWidth,
		IsBidirectional:  t.isBidirectional,
		HasTelemetry:     t.hasTelemetry,
		Protocol:         t.protocol,
	}, nil
}

// SetThrustDirectly sets thrust without validation (for testing).
func (t *Thruster) SetThrustDirectly(thrust float64) {
	t.thrust = thrust
	t.moving = thrust != 0
}

// SetTelemetry sets telemetry values for testing.
func (t *Thruster) SetTelemetry(rpm int, temperature, current float64) {
	t.rpm = rpm
	t.temperature = temperature
	t.current = current
}

// SetProperties sets the thruster properties for testing.
func (t *Thruster) SetProperties(maxFwd, maxRev, deadband float64, bidir, telem bool, protocol string) {
	t.maxThrustForward = maxFwd
	t.maxThrustReverse = maxRev
	t.deadbandWidth = deadband
	t.isBidirectional = bidir
	t.hasTelemetry = telem
	t.protocol = protocol
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Verify interface compliance.
var _ thruster.Thruster = (*Thruster)(nil)
