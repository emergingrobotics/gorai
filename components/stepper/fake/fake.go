// Package fake provides a fake stepper motor implementation for testing.
package fake

import (
	"context"
	"fmt"

	"github.com/gorai/gorai/components/stepper"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("stepper", "fake", New)
}

// Stepper is a fake stepper motor for testing.
type Stepper struct {
	name resource.Name

	// Current state
	position     int64
	moving       bool
	microstepping int
	runCurrent   int
	holdCurrent  int

	// Properties
	stepsPerRevolution int
	maxMicrostepping   int
	maxCurrent         int
	hasStallDetection  bool
	driver             string
}

// New creates a new fake stepper.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "stepper", nameStr)

	stepsPerRev := 200
	if v, ok := conf["steps_per_revolution"].(float64); ok {
		stepsPerRev = int(v)
	}

	return &Stepper{
		name:               name,
		microstepping:      1,
		runCurrent:         1000,
		holdCurrent:        500,
		stepsPerRevolution: stepsPerRev,
		maxMicrostepping:   256,
		maxCurrent:         2000,
		hasStallDetection:  true,
		driver:             "tmc2209",
	}, nil
}

// NewWithName creates a fake stepper with a specific resource name.
func NewWithName(name resource.Name) *Stepper {
	return &Stepper{
		name:               name,
		microstepping:      1,
		runCurrent:         1000,
		holdCurrent:        500,
		stepsPerRevolution: 200,
		maxMicrostepping:   256,
		maxCurrent:         2000,
		hasStallDetection:  true,
		driver:             "tmc2209",
	}
}

// Name returns the resource name.
func (s *Stepper) Name() resource.Name {
	return s.name
}

// Reconfigure updates the configuration.
func (s *Stepper) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (s *Stepper) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_state":
			return map[string]any{
				"position":      s.position,
				"microstepping": s.microstepping,
				"run_current":   s.runCurrent,
				"hold_current":  s.holdCurrent,
				"moving":        s.moving,
			}, nil
		case "set_position":
			if pos, ok := cmd["position"].(float64); ok {
				s.position = int64(pos)
				return map[string]any{"status": "ok"}, nil
			}
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (s *Stepper) Close(ctx context.Context) error {
	return s.Stop(ctx)
}

// IsMoving returns whether the stepper is moving.
func (s *Stepper) IsMoving(ctx context.Context) (bool, error) {
	return s.moving, nil
}

// Stop stops the stepper.
func (s *Stepper) Stop(ctx context.Context) error {
	s.moving = false
	return nil
}

// Step moves the motor by the specified number of steps.
func (s *Stepper) Step(ctx context.Context, steps int64) error {
	s.position += steps
	return nil
}

// SetMicrostepping sets the microstepping divisor.
func (s *Stepper) SetMicrostepping(ctx context.Context, divisor int) error {
	if divisor < 1 || divisor > s.maxMicrostepping {
		return fmt.Errorf("microstepping divisor %d out of range [1, %d]", divisor, s.maxMicrostepping)
	}
	// Validate it's a power of 2
	if divisor&(divisor-1) != 0 {
		return fmt.Errorf("microstepping divisor must be a power of 2, got %d", divisor)
	}
	s.microstepping = divisor
	return nil
}

// SetCurrent sets the run and hold current.
func (s *Stepper) SetCurrent(ctx context.Context, runMA, holdMA int) error {
	if runMA < 0 || runMA > s.maxCurrent {
		return fmt.Errorf("run current %d out of range [0, %d]", runMA, s.maxCurrent)
	}
	if holdMA < 0 || holdMA > s.maxCurrent {
		return fmt.Errorf("hold current %d out of range [0, %d]", holdMA, s.maxCurrent)
	}
	s.runCurrent = runMA
	s.holdCurrent = holdMA
	return nil
}

// GetPosition returns the current position in steps.
func (s *Stepper) GetPosition(ctx context.Context) (int64, error) {
	return s.position, nil
}

// ResetPosition sets the current position as zero.
func (s *Stepper) ResetPosition(ctx context.Context) error {
	s.position = 0
	return nil
}

// Home performs a homing operation.
func (s *Stepper) Home(ctx context.Context, direction bool) error {
	// Fake homing just resets position
	s.position = 0
	return nil
}

// GetProperties returns the stepper properties.
func (s *Stepper) GetProperties(ctx context.Context) (stepper.Properties, error) {
	return stepper.Properties{
		StepsPerRevolution: s.stepsPerRevolution,
		MaxMicrostepping:   s.maxMicrostepping,
		MaxCurrent:         s.maxCurrent,
		HasStallDetection:  s.hasStallDetection,
		Driver:             s.driver,
	}, nil
}

// SetPosition sets the position for testing.
func (s *Stepper) SetPosition(pos int64) {
	s.position = pos
}

// SetMoving sets the moving state for testing.
func (s *Stepper) SetMoving(moving bool) {
	s.moving = moving
}

// SetProperties sets the stepper properties for testing.
func (s *Stepper) SetProperties(stepsPerRev, maxMicro, maxCurrent int, hasStall bool, driver string) {
	s.stepsPerRevolution = stepsPerRev
	s.maxMicrostepping = maxMicro
	s.maxCurrent = maxCurrent
	s.hasStallDetection = hasStall
	s.driver = driver
}

// Verify interface compliance.
var _ stepper.Stepper = (*Stepper)(nil)
