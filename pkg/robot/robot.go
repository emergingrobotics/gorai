// Package robot provides the main robot runtime for Gorai.
package robot

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/hardware/v4l2"
	gorainats "github.com/gorai/gorai/pkg/nats"
	"github.com/gorai/gorai/pkg/topics"
)

// Robot represents a running robot instance.
type Robot struct {
	cfg    *config.RDL
	logger *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc

	// NATS client for messaging
	nats   *gorainats.Client
	topics *topics.Builder
}

// Option configures a Robot.
type Option func(*Robot)

// WithLogger sets the logger for the robot.
func WithLogger(logger *slog.Logger) Option {
	return func(r *Robot) {
		r.logger = logger
	}
}

// New creates a new Robot from the given configuration.
func New(ctx context.Context, cfg *config.RDL, opts ...Option) (*Robot, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}

	rCtx, cancel := context.WithCancel(ctx)
	r := &Robot{
		cfg:    cfg,
		logger: slog.Default(),
		ctx:    rCtx,
		cancel: cancel,
		topics: topics.NewBuilder(cfg.Robot.Name),
	}

	for _, opt := range opts {
		opt(r)
	}

	r.logger.Info("Robot created", "name", cfg.Robot.Name)

	return r, nil
}

// Start initializes and starts all robot components and services.
func (r *Robot) Start(ctx context.Context) error {
	r.logger.Info("Starting robot", "name", r.cfg.Robot.Name)

	// Connect to NATS
	if err := r.connectNATS(ctx); err != nil {
		return fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Publish robot started event
	r.publishStartupEvent(topics.EventRobotStarted, "", "", "Robot starting initialization", true, nil)

	// Detect and validate hardware components
	if err := r.detectHardware(ctx); err != nil {
		r.publishStartupEvent(topics.EventComponentError, "", "", fmt.Sprintf("Hardware detection failed: %v", err), false, nil)
		return err
	}

	// Initialize components
	for _, comp := range r.cfg.Components {
		if comp.Disabled {
			r.logger.Info("Skipping disabled component", "name", comp.Name)
			continue
		}
		r.logger.Info("Initializing component", "name", comp.Name, "type", comp.Type, "model", comp.Model)
		// TODO: Actually initialize component from registry
	}

	// Initialize services
	for _, svc := range r.cfg.Services {
		if svc.Disabled {
			r.logger.Info("Skipping disabled service", "name", svc.Name)
			continue
		}
		r.logger.Info("Initializing service", "name", svc.Name, "type", svc.Type)
		// TODO: Actually initialize service from registry
	}

	// Publish robot ready event
	r.publishStartupEvent(topics.EventRobotReady, "", "", "Robot initialization complete", true, map[string]any{
		"components": len(r.cfg.Components),
		"services":   len(r.cfg.Services),
	})

	r.logger.Info("Robot started", "components", len(r.cfg.Components), "services", len(r.cfg.Services))
	return nil
}

// connectNATS establishes connection to the NATS server.
func (r *Robot) connectNATS(ctx context.Context) error {
	natsURL := "nats://localhost:4222"
	if r.cfg.NATS != nil && r.cfg.NATS.URL != "" {
		natsURL = r.cfg.NATS.URL
	}

	natsCfg := &gorainats.Config{
		URL:            natsURL,
		Name:           fmt.Sprintf("gorai-%s", r.cfg.Robot.Name),
		ConnectTimeout: 10 * time.Second,
		ReconnectWait:  2 * time.Second,
		MaxReconnects:  -1,
	}

	client, err := gorainats.Connect(ctx, natsCfg, gorainats.WithLogger(r.logger))
	if err != nil {
		return err
	}

	r.nats = client
	r.logger.Info("Connected to NATS", "url", natsURL)
	return nil
}

// detectHardware detects and validates all hardware components.
func (r *Robot) detectHardware(ctx context.Context) error {
	for _, comp := range r.cfg.Components {
		if comp.Disabled {
			continue
		}

		switch comp.Type {
		case "camera":
			if err := r.detectCamera(comp); err != nil {
				return err
			}
		// Add other component types as needed
		default:
			r.logger.Debug("No hardware detection for component type", "type", comp.Type, "name", comp.Name)
		}
	}
	return nil
}

// detectCamera checks if a camera device is present.
func (r *Robot) detectCamera(comp config.ComponentConfig) error {
	// Get device path from attributes
	devicePath := "/dev/video0" // default
	if comp.Attributes != nil {
		if dev, ok := comp.Attributes["device"].(string); ok {
			devicePath = dev
		}
	}

	r.logger.Info("Detecting camera", "name", comp.Name, "device", devicePath)

	// Check if device exists
	result := v4l2.DetectDevice(devicePath)

	if !result.Found {
		errMsg := fmt.Sprintf("Camera %q not found at %s: %s", comp.Name, devicePath, result.Error)
		r.logger.Error("Camera detection failed", "name", comp.Name, "device", devicePath, "error", result.Error)

		// Publish failure event
		r.publishStartupEvent(topics.EventComponentMissing, comp.Name, comp.Type, errMsg, false, map[string]any{
			"device": devicePath,
			"error":  result.Error,
		})

		return fmt.Errorf(errMsg)
	}

	// Camera found
	deviceInfo := result.Device
	summary := v4l2.GetDeviceSummary(deviceInfo)

	r.logger.Info("Camera detected",
		"name", comp.Name,
		"device", devicePath,
		"device_name", deviceInfo.Name,
		"driver", deviceInfo.Driver,
	)

	// Publish success event
	r.publishStartupEvent(topics.EventComponentDetected, comp.Name, comp.Type,
		fmt.Sprintf("Camera %q detected: %s", comp.Name, summary), true, map[string]any{
			"device":      devicePath,
			"device_name": deviceInfo.Name,
			"driver":      deviceInfo.Driver,
			"index":       deviceInfo.Index,
		})

	return nil
}

// publishStartupEvent publishes a startup event to the NATS system topic.
func (r *Robot) publishStartupEvent(eventType, component, componentType, message string, success bool, details map[string]any) {
	if r.nats == nil {
		return
	}

	event := topics.StartupEvent{
		EventType:     eventType,
		Component:     component,
		ComponentType: componentType,
		Message:       message,
		Details:       details,
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		Success:       success,
	}

	subject := r.topics.SystemStartup()
	if err := r.nats.PublishJSON(subject, event); err != nil {
		r.logger.Warn("Failed to publish startup event", "error", err, "subject", subject)
	} else {
		r.logger.Debug("Published startup event", "subject", subject, "event_type", eventType)
	}
}

// Run runs the robot until the context is cancelled.
func (r *Robot) Run(ctx context.Context) error {
	r.logger.Info("Robot running", "name", r.cfg.Robot.Name)

	// Wait for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-r.ctx.Done():
		return r.ctx.Err()
	}
}

// Stop gracefully stops the robot.
func (r *Robot) Stop(ctx context.Context) error {
	r.logger.Info("Stopping robot", "name", r.cfg.Robot.Name)

	// Publish shutdown event
	r.publishStartupEvent(topics.EventRobotShutdown, "", "", "Robot shutting down", true, nil)

	// Cancel internal context
	r.cancel()

	// TODO: Stop all components and services gracefully

	// Close NATS connection
	if r.nats != nil {
		r.nats.Close()
	}

	r.logger.Info("Robot stopped", "name", r.cfg.Robot.Name)
	return nil
}

// Reconfigure updates the robot configuration at runtime.
func (r *Robot) Reconfigure(ctx context.Context, cfg *config.RDL) error {
	r.logger.Info("Reconfiguring robot", "name", cfg.Robot.Name)

	// TODO: Implement hot reconfiguration
	r.cfg = cfg

	return nil
}

// Config returns the current robot configuration.
func (r *Robot) Config() *config.RDL {
	return r.cfg
}

// NATS returns the NATS client.
func (r *Robot) NATS() *gorainats.Client {
	return r.nats
}

// Topics returns the topic builder for this robot.
func (r *Robot) Topics() *topics.Builder {
	return r.topics
}
