// Package robot provides the main robot runtime for Gorai.
package robot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/gorai/gorai/pkg/config"
)

// Robot represents a running robot instance.
type Robot struct {
	cfg    *config.RDL
	logger *slog.Logger
	ctx    context.Context
	cancel context.CancelFunc
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

	r.logger.Info("Robot started", "components", len(r.cfg.Components), "services", len(r.cfg.Services))
	return nil
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

	// Cancel internal context
	r.cancel()

	// TODO: Stop all components and services gracefully

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
