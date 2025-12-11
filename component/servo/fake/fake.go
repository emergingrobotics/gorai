// Package fake provides a fake servo implementation for testing.
package fake

import (
	"context"
	"fmt"

	"github.com/gorai/gorai/component/servo"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("servo", "fake", New)
}

// Servo is a fake servo for testing.
type Servo struct {
	name resource.Name

	// Current state
	angle       float64
	speed       float64
	torqueLimit float64
	moving      bool

	// Properties
	minAngle     float64
	maxAngle     float64
	isContinuous bool
	hasFeedback  bool
	protocol     string
}

// New creates a new fake servo.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "servo", nameStr)

	minAngle := -90.0
	if v, ok := conf["min_angle"].(float64); ok {
		minAngle = v
	}

	maxAngle := 90.0
	if v, ok := conf["max_angle"].(float64); ok {
		maxAngle = v
	}

	return &Servo{
		name:        name,
		torqueLimit: 1.0,
		minAngle:    minAngle,
		maxAngle:    maxAngle,
		hasFeedback: true,
		protocol:    "pwm",
	}, nil
}

// NewWithName creates a fake servo with a specific resource name.
func NewWithName(name resource.Name) *Servo {
	return &Servo{
		name:        name,
		torqueLimit: 1.0,
		minAngle:    -90.0,
		maxAngle:    90.0,
		hasFeedback: true,
		protocol:    "pwm",
	}
}

// Name returns the resource name.
func (s *Servo) Name() resource.Name {
	return s.name
}

// Reconfigure updates the configuration.
func (s *Servo) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (s *Servo) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_state":
			return map[string]any{
				"angle":        s.angle,
				"speed":        s.speed,
				"torque_limit": s.torqueLimit,
				"moving":       s.moving,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (s *Servo) Close(ctx context.Context) error {
	return s.Stop(ctx)
}

// IsMoving returns whether the servo is moving.
func (s *Servo) IsMoving(ctx context.Context) (bool, error) {
	return s.moving, nil
}

// Stop stops the servo.
func (s *Servo) Stop(ctx context.Context) error {
	s.moving = false
	return nil
}

// SetAngle moves the servo to the specified angle.
func (s *Servo) SetAngle(ctx context.Context, degrees float64) error {
	if degrees < s.minAngle || degrees > s.maxAngle {
		return fmt.Errorf("angle %v out of range [%v, %v]", degrees, s.minAngle, s.maxAngle)
	}
	s.angle = degrees
	s.moving = false // Fake servo moves instantly
	return nil
}

// GetAngle returns the current angle.
func (s *Servo) GetAngle(ctx context.Context) (float64, error) {
	return s.angle, nil
}

// SetSpeed sets the movement speed.
func (s *Servo) SetSpeed(ctx context.Context, speed float64) error {
	s.speed = speed
	return nil
}

// SetTorqueLimit sets the torque limit.
func (s *Servo) SetTorqueLimit(ctx context.Context, limit float64) error {
	if limit < 0 || limit > 1 {
		return fmt.Errorf("torque limit must be 0.0-1.0, got %v", limit)
	}
	s.torqueLimit = limit
	return nil
}

// GetProperties returns the servo properties.
func (s *Servo) GetProperties(ctx context.Context) (servo.Properties, error) {
	return servo.Properties{
		MinAngle:     s.minAngle,
		MaxAngle:     s.maxAngle,
		IsContinuous: s.isContinuous,
		HasFeedback:  s.hasFeedback,
		Protocol:     s.protocol,
	}, nil
}

// SetAngleDirectly sets the angle without validation (for testing).
func (s *Servo) SetAngleDirectly(angle float64) {
	s.angle = angle
}

// SetMoving sets the moving state for testing.
func (s *Servo) SetMoving(moving bool) {
	s.moving = moving
}

// SetProperties sets the servo properties for testing.
func (s *Servo) SetProperties(minAngle, maxAngle float64, isContinuous, hasFeedback bool, protocol string) {
	s.minAngle = minAngle
	s.maxAngle = maxAngle
	s.isContinuous = isContinuous
	s.hasFeedback = hasFeedback
	s.protocol = protocol
}

// Verify interface compliance.
var _ servo.Servo = (*Servo)(nil)
