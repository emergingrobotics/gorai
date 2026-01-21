// Package keypress_motor provides a service that translates keyboard input
// events into motor control commands.
package keypress_motor

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/gorai/gorai/component/input"
	"github.com/gorai/gorai/component/pwm"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterService("control", "keypress_motor_controller", New)
}

// State represents the operational state of the controller.
type State int

const (
	StateStopped State = iota
	StateRunning
	StateError
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateStopped:
		return "stopped"
	case StateRunning:
		return "running"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// MotorState holds the current state of a motor binding.
type MotorState struct {
	Name              string
	Type              MotorType
	PWMComponent      string
	Enabled           bool
	CurrentAngle      float64 // For angle servo
	CurrentSpeed      float64 // For continuous/DC motor
	CurrentPulseUs    float64 // Actual pulse width
	ForwardKeyPressed bool
	ReverseKeyPressed bool
}

// Controller implements the keypress motor controller service.
type Controller struct {
	name   resource.Name
	config *Config
	logger *slog.Logger

	// Dependencies
	keyboard input.Keyboard
	pwmMap   map[string]pwm.PWM

	// Configuration lookup
	motorConfigs  map[string]*MotorConfig
	forwardKeyMap map[string]*MotorConfig
	reverseKeyMap map[string]*MotorConfig

	// State
	mu          sync.RWMutex
	state       State
	motorStates map[string]*MotorState
	errorMsg    string

	// Control
	stopCh chan struct{}
	doneCh chan struct{}

	// Metrics
	keyEventsProcessed atomic.Uint64
	pwmCommandsSent    atomic.Uint64
}

// New creates a new keypress motor controller service.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	// Convert registry.Config to resource.Config
	resConf := resource.NewConfig(conf)

	cfg, err := NewConfigFromResource(resConf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Get name from config
	name := "motor_controller"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	c := &Controller{
		name:          resource.NewServiceName("gorai", "control", name),
		config:        cfg,
		logger:        slog.Default().With("service", "keypress_motor_controller", "name", name),
		pwmMap:        make(map[string]pwm.PWM),
		motorConfigs:  make(map[string]*MotorConfig),
		forwardKeyMap: make(map[string]*MotorConfig),
		reverseKeyMap: make(map[string]*MotorConfig),
		motorStates:   make(map[string]*MotorState),
		state:         StateStopped,
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
	}

	// Note: Dependencies (keyboard, PWM components) are resolved later
	// when the robot runtime calls Reconfigure with actual dependencies.
	// For now, we just store the config and prepare the lookup tables.

	// Build lookup tables
	for i := range cfg.Motors {
		motor := &cfg.Motors[i]
		c.motorConfigs[motor.Name] = motor
		c.forwardKeyMap[motor.ForwardKey] = motor
		c.reverseKeyMap[motor.ReverseKey] = motor

		// Initialize motor state
		c.motorStates[motor.Name] = &MotorState{
			Name:         motor.Name,
			Type:         motor.Type,
			PWMComponent: motor.PWMComponent,
			Enabled:      true,
			CurrentAngle: motor.InitialAngle,
			CurrentSpeed: 0.0,
		}
	}

	c.logger.Info("keypress motor controller created",
		"keyboard", cfg.KeyboardComponent,
		"motor_count", len(cfg.Motors))

	return c, nil
}

// Name returns the resource name.
func (c *Controller) Name() resource.Name {
	return c.name
}

// Reconfigure updates the controller configuration and resolves dependencies.
func (c *Controller) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// Stop current operation if running
	c.mu.Lock()
	wasRunning := c.state == StateRunning
	c.mu.Unlock()

	if wasRunning {
		if err := c.stop(ctx); err != nil {
			c.logger.Warn("failed to stop during reconfigure", "error", err)
		}
	}

	// Resolve keyboard dependency
	kbName := resource.NewComponentName("gorai", "input", cfg.KeyboardComponent)
	kbRes, err := deps.Get(kbName)
	if err != nil {
		return fmt.Errorf("keyboard component %q not found: %w", cfg.KeyboardComponent, err)
	}

	keyboard, ok := kbRes.(input.Keyboard)
	if !ok {
		return fmt.Errorf("component %q is not a keyboard", cfg.KeyboardComponent)
	}

	// Resolve PWM dependencies
	pwmMap := make(map[string]pwm.PWM)
	for _, motor := range cfg.Motors {
		pwmName := resource.NewComponentName("gorai", "pwm", motor.PWMComponent)
		pwmRes, err := deps.Get(pwmName)
		if err != nil {
			return fmt.Errorf("PWM component %q not found: %w", motor.PWMComponent, err)
		}

		pwmComp, ok := pwmRes.(pwm.PWM)
		if !ok {
			return fmt.Errorf("component %q is not a PWM", motor.PWMComponent)
		}
		pwmMap[motor.Name] = pwmComp
	}

	// Update controller
	c.mu.Lock()
	c.config = cfg
	c.keyboard = keyboard
	c.pwmMap = pwmMap

	// Rebuild lookup tables
	c.motorConfigs = make(map[string]*MotorConfig)
	c.forwardKeyMap = make(map[string]*MotorConfig)
	c.reverseKeyMap = make(map[string]*MotorConfig)

	for i := range cfg.Motors {
		motor := &cfg.Motors[i]
		c.motorConfigs[motor.Name] = motor
		c.forwardKeyMap[motor.ForwardKey] = motor
		c.reverseKeyMap[motor.ReverseKey] = motor

		// Initialize or update motor state
		if _, exists := c.motorStates[motor.Name]; !exists {
			c.motorStates[motor.Name] = &MotorState{
				Name:         motor.Name,
				Type:         motor.Type,
				PWMComponent: motor.PWMComponent,
				Enabled:      true,
				CurrentAngle: motor.InitialAngle,
				CurrentSpeed: 0.0,
			}
		}
	}
	c.mu.Unlock()

	// Start operation
	if err := c.start(ctx); err != nil {
		return fmt.Errorf("failed to start: %w", err)
	}

	c.logger.Info("keypress motor controller reconfigured",
		"keyboard", cfg.KeyboardComponent,
		"motor_count", len(cfg.Motors))

	return nil
}

// start begins the event processing loop.
func (c *Controller) start(ctx context.Context) error {
	c.mu.Lock()
	if c.state == StateRunning {
		c.mu.Unlock()
		return nil
	}

	if c.keyboard == nil {
		c.mu.Unlock()
		return fmt.Errorf("keyboard not configured")
	}

	c.stopCh = make(chan struct{})
	c.doneCh = make(chan struct{})
	c.state = StateRunning
	c.mu.Unlock()

	// Set initial positions for angle servos
	for _, motor := range c.config.Motors {
		if motor.Type == MotorTypeAngleServo {
			if err := c.setAngle(ctx, motor.Name, motor.InitialAngle); err != nil {
				c.logger.Warn("failed to set initial angle", "motor", motor.Name, "error", err)
			}
		}
	}

	// Get event channel from keyboard
	eventsCh, err := c.keyboard.Events(ctx)
	if err != nil {
		c.mu.Lock()
		c.state = StateError
		c.errorMsg = fmt.Sprintf("failed to get keyboard events: %v", err)
		c.mu.Unlock()
		return err
	}

	// Start event processing goroutine
	go c.eventLoop(eventsCh)

	c.logger.Info("keypress motor controller started")
	return nil
}

// stop halts the event processing loop and stops all motors.
func (c *Controller) stop(ctx context.Context) error {
	c.mu.Lock()
	if c.state != StateRunning {
		c.mu.Unlock()
		return nil
	}
	c.state = StateStopped
	c.mu.Unlock()

	// Signal stop
	close(c.stopCh)

	// Wait for event loop to finish
	<-c.doneCh

	// Stop all motors
	c.stopAllMotors(ctx)

	c.logger.Info("keypress motor controller stopped")
	return nil
}

// eventLoop processes keyboard events.
func (c *Controller) eventLoop(eventsCh <-chan input.KeyEvent) {
	defer close(c.doneCh)

	ctx := context.Background()

	for {
		select {
		case <-c.stopCh:
			return

		case event, ok := <-eventsCh:
			if !ok {
				// Channel closed, keyboard disconnected
				c.mu.Lock()
				c.state = StateError
				c.errorMsg = "keyboard disconnected"
				c.mu.Unlock()
				c.logger.Error("keyboard event channel closed")
				return
			}

			// Skip repeat events
			if event.Repeat {
				continue
			}

			c.processKeyEvent(ctx, event)
		}
	}
}

// processKeyEvent handles a single key event.
func (c *Controller) processKeyEvent(ctx context.Context, event input.KeyEvent) {
	c.keyEventsProcessed.Add(1)

	c.mu.RLock()
	forwardMotor := c.forwardKeyMap[event.Key]
	reverseMotor := c.reverseKeyMap[event.Key]
	c.mu.RUnlock()

	if forwardMotor != nil {
		c.handleForwardKey(ctx, forwardMotor, event.Pressed)
	} else if reverseMotor != nil {
		c.handleReverseKey(ctx, reverseMotor, event.Pressed)
	}
	// Key not bound to any motor - ignore
}

// handleForwardKey processes a forward key event.
func (c *Controller) handleForwardKey(ctx context.Context, motor *MotorConfig, pressed bool) {
	c.mu.Lock()
	state := c.motorStates[motor.Name]
	if state == nil || !state.Enabled {
		c.mu.Unlock()
		return
	}
	state.ForwardKeyPressed = pressed
	c.mu.Unlock()

	switch motor.Type {
	case MotorTypeAngleServo:
		if pressed {
			c.mu.RLock()
			currentAngle := state.CurrentAngle
			c.mu.RUnlock()

			newAngle := Clamp(currentAngle+motor.AngleStep, motor.MinAngle, motor.MaxAngle)
			if err := c.setAngle(ctx, motor.Name, newAngle); err != nil {
				c.logger.Error("failed to set angle", "motor", motor.Name, "error", err)
			}
		}

	case MotorTypeContinuousServo, MotorTypeDCMotor:
		if pressed {
			if err := c.setSpeed(ctx, motor.Name, motor.Speed); err != nil {
				c.logger.Error("failed to set speed", "motor", motor.Name, "error", err)
			}
		} else if motor.StopOnRelease {
			c.mu.RLock()
			reversePressed := state.ReverseKeyPressed
			c.mu.RUnlock()

			if !reversePressed {
				if err := c.setSpeed(ctx, motor.Name, 0.0); err != nil {
					c.logger.Error("failed to stop motor", "motor", motor.Name, "error", err)
				}
			}
		}
	}
}

// handleReverseKey processes a reverse key event.
func (c *Controller) handleReverseKey(ctx context.Context, motor *MotorConfig, pressed bool) {
	c.mu.Lock()
	state := c.motorStates[motor.Name]
	if state == nil || !state.Enabled {
		c.mu.Unlock()
		return
	}
	state.ReverseKeyPressed = pressed
	c.mu.Unlock()

	switch motor.Type {
	case MotorTypeAngleServo:
		if pressed {
			c.mu.RLock()
			currentAngle := state.CurrentAngle
			c.mu.RUnlock()

			newAngle := Clamp(currentAngle-motor.AngleStep, motor.MinAngle, motor.MaxAngle)
			if err := c.setAngle(ctx, motor.Name, newAngle); err != nil {
				c.logger.Error("failed to set angle", "motor", motor.Name, "error", err)
			}
		}

	case MotorTypeContinuousServo, MotorTypeDCMotor:
		if pressed {
			if err := c.setSpeed(ctx, motor.Name, -motor.Speed); err != nil {
				c.logger.Error("failed to set speed", "motor", motor.Name, "error", err)
			}
		} else if motor.StopOnRelease {
			c.mu.RLock()
			forwardPressed := state.ForwardKeyPressed
			c.mu.RUnlock()

			if !forwardPressed {
				if err := c.setSpeed(ctx, motor.Name, 0.0); err != nil {
					c.logger.Error("failed to stop motor", "motor", motor.Name, "error", err)
				}
			}
		}
	}
}

// setAngle sets the angle for an angle servo.
func (c *Controller) setAngle(ctx context.Context, motorName string, angle float64) error {
	c.mu.RLock()
	motor := c.motorConfigs[motorName]
	state := c.motorStates[motorName]
	pwmComp := c.pwmMap[motorName]
	c.mu.RUnlock()

	if motor == nil || state == nil || pwmComp == nil {
		return fmt.Errorf("motor %q not found", motorName)
	}

	// Clamp angle
	angle = Clamp(angle, motor.MinAngle, motor.MaxAngle)

	// Convert to pulse width
	pulseUs := AngleToPulse(angle, motor.MinAngle, motor.MaxAngle)

	// Send to PWM
	if err := pwmComp.SetPulse(ctx, pulseUs); err != nil {
		return fmt.Errorf("PWM SetPulse failed: %w", err)
	}

	// Update state
	c.mu.Lock()
	state.CurrentAngle = angle
	state.CurrentPulseUs = pulseUs
	c.mu.Unlock()

	c.pwmCommandsSent.Add(1)
	c.logger.Debug("angle set", "motor", motorName, "angle", angle, "pulse_us", pulseUs)

	return nil
}

// setSpeed sets the speed for a continuous servo or DC motor.
func (c *Controller) setSpeed(ctx context.Context, motorName string, speed float64) error {
	c.mu.RLock()
	motor := c.motorConfigs[motorName]
	state := c.motorStates[motorName]
	pwmComp := c.pwmMap[motorName]
	c.mu.RUnlock()

	if motor == nil || state == nil || pwmComp == nil {
		return fmt.Errorf("motor %q not found", motorName)
	}

	// Clamp speed
	speed = Clamp(speed, -1.0, 1.0)

	var err error
	var pulseUs float64

	switch motor.Type {
	case MotorTypeContinuousServo:
		pulseUs = SpeedToPulse(speed)
		err = pwmComp.SetPulse(ctx, pulseUs)

	case MotorTypeDCMotor:
		err = pwmComp.SetNormalized(ctx, speed)
	}

	if err != nil {
		return fmt.Errorf("PWM command failed: %w", err)
	}

	// Update state
	c.mu.Lock()
	state.CurrentSpeed = speed
	state.CurrentPulseUs = pulseUs
	c.mu.Unlock()

	c.pwmCommandsSent.Add(1)
	c.logger.Debug("speed set", "motor", motorName, "speed", speed)

	return nil
}

// stopMotor stops a single motor.
func (c *Controller) stopMotor(ctx context.Context, motorName string) error {
	c.mu.RLock()
	motor := c.motorConfigs[motorName]
	state := c.motorStates[motorName]
	c.mu.RUnlock()

	if motor == nil || state == nil {
		return fmt.Errorf("motor %q not found", motorName)
	}

	// Clear key states
	c.mu.Lock()
	state.ForwardKeyPressed = false
	state.ReverseKeyPressed = false
	c.mu.Unlock()

	switch motor.Type {
	case MotorTypeAngleServo:
		// Angle servos maintain position, nothing to do
		return nil
	case MotorTypeContinuousServo, MotorTypeDCMotor:
		return c.setSpeed(ctx, motorName, 0.0)
	}

	return nil
}

// stopAllMotors stops all motors.
func (c *Controller) stopAllMotors(ctx context.Context) {
	c.mu.RLock()
	motorNames := make([]string, 0, len(c.motorStates))
	for name := range c.motorStates {
		motorNames = append(motorNames, name)
	}
	c.mu.RUnlock()

	for _, name := range motorNames {
		if err := c.stopMotor(ctx, name); err != nil {
			c.logger.Warn("failed to stop motor", "motor", name, "error", err)
		}
	}
}

// DoCommand handles arbitrary commands.
func (c *Controller) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)

	switch cmdName {
	case "set_angle":
		motorName, _ := cmd["motor"].(string)
		angle, _ := cmd["angle"].(float64)

		c.mu.RLock()
		motor := c.motorConfigs[motorName]
		c.mu.RUnlock()

		if motor == nil {
			return nil, fmt.Errorf("motor %q not found", motorName)
		}
		if motor.Type != MotorTypeAngleServo {
			return nil, fmt.Errorf("motor %q is not an angle servo", motorName)
		}

		if err := c.setAngle(ctx, motorName, angle); err != nil {
			return nil, err
		}
		return map[string]any{"success": true}, nil

	case "set_speed":
		motorName, _ := cmd["motor"].(string)
		speed, _ := cmd["speed"].(float64)

		c.mu.RLock()
		motor := c.motorConfigs[motorName]
		c.mu.RUnlock()

		if motor == nil {
			return nil, fmt.Errorf("motor %q not found", motorName)
		}
		if motor.Type == MotorTypeAngleServo {
			return nil, fmt.Errorf("motor %q is an angle servo, use set_angle", motorName)
		}

		if err := c.setSpeed(ctx, motorName, speed); err != nil {
			return nil, err
		}
		return map[string]any{"success": true}, nil

	case "stop":
		motorName, hasMotor := cmd["motor"].(string)
		if hasMotor && motorName != "" {
			if err := c.stopMotor(ctx, motorName); err != nil {
				return nil, err
			}
		} else {
			c.stopAllMotors(ctx)
		}
		return map[string]any{"success": true}, nil

	case "get_state":
		motorName, hasMotor := cmd["motor"].(string)

		c.mu.RLock()
		defer c.mu.RUnlock()

		if hasMotor && motorName != "" {
			state := c.motorStates[motorName]
			if state == nil {
				return nil, fmt.Errorf("motor %q not found", motorName)
			}
			return map[string]any{
				"name":                 state.Name,
				"type":                 string(state.Type),
				"enabled":              state.Enabled,
				"current_angle":        state.CurrentAngle,
				"current_speed":        state.CurrentSpeed,
				"current_pulse_us":     state.CurrentPulseUs,
				"forward_key_pressed":  state.ForwardKeyPressed,
				"reverse_key_pressed":  state.ReverseKeyPressed,
			}, nil
		}

		// Return all motor states
		states := make(map[string]any)
		for name, state := range c.motorStates {
			states[name] = map[string]any{
				"type":                string(state.Type),
				"enabled":             state.Enabled,
				"current_angle":       state.CurrentAngle,
				"current_speed":       state.CurrentSpeed,
				"current_pulse_us":    state.CurrentPulseUs,
				"forward_key_pressed": state.ForwardKeyPressed,
				"reverse_key_pressed": state.ReverseKeyPressed,
			}
		}
		return map[string]any{
			"state":  c.state.String(),
			"motors": states,
		}, nil

	case "enable":
		motorName, hasMotor := cmd["motor"].(string)

		c.mu.Lock()
		defer c.mu.Unlock()

		if hasMotor && motorName != "" {
			state := c.motorStates[motorName]
			if state == nil {
				return nil, fmt.Errorf("motor %q not found", motorName)
			}
			state.Enabled = true
		} else {
			for _, state := range c.motorStates {
				state.Enabled = true
			}
		}
		return map[string]any{"success": true}, nil

	case "disable":
		motorName, hasMotor := cmd["motor"].(string)

		c.mu.Lock()
		defer c.mu.Unlock()

		if hasMotor && motorName != "" {
			state := c.motorStates[motorName]
			if state == nil {
				return nil, fmt.Errorf("motor %q not found", motorName)
			}
			state.Enabled = false
		} else {
			for _, state := range c.motorStates {
				state.Enabled = false
			}
		}
		return map[string]any{"success": true}, nil

	case "get_stats":
		return map[string]any{
			"key_events_processed": c.keyEventsProcessed.Load(),
			"pwm_commands_sent":    c.pwmCommandsSent.Load(),
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close releases all resources.
func (c *Controller) Close(ctx context.Context) error {
	if err := c.stop(ctx); err != nil {
		c.logger.Warn("error during stop", "error", err)
	}

	c.logger.Info("keypress motor controller closed")
	return nil
}

// GetState returns the current operational state.
func (c *Controller) GetState() State {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.state
}

// GetMotorState returns the state of a specific motor.
func (c *Controller) GetMotorState(motorName string) (*MotorState, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	state := c.motorStates[motorName]
	if state == nil {
		return nil, fmt.Errorf("motor %q not found", motorName)
	}

	// Return a copy
	copy := *state
	return &copy, nil
}

