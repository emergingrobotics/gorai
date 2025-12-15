package commands

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/robot"
	"github.com/gorai/gorai/pkg/runtime"
)

func cmdRun() error {
	// Parse flags
	var configPath string
	var logLevel string

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("--config requires a value")
			}
			i++
			configPath = args[i]
		case "--log-level":
			if i+1 >= len(args) {
				return fmt.Errorf("--log-level requires a value")
			}
			i++
			logLevel = args[i]
		case "-h", "--help":
			return printRunUsage()
		default:
			if args[i][0] != '-' {
				// Treat as config path if not a flag
				configPath = args[i]
			} else {
				return fmt.Errorf("unknown flag: %s", args[i])
			}
		}
	}

	// Try to find config if not specified
	if configPath == "" {
		configPath = findConfigFile()
		if configPath == "" {
			return fmt.Errorf("no config file specified. Use --config or create robot.json")
		}
	}

	// Make config path absolute
	if !filepath.IsAbs(configPath) {
		absPath, err := filepath.Abs(configPath)
		if err == nil {
			configPath = absPath
		}
	}

	// Get config directory
	configDir := filepath.Dir(configPath)

	// Load configuration with Service RDL support
	cfg, err := config.LoadWithServiceRDL(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Check for deprecation warnings
	warnings := cfg.DeprecationWarnings()
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", w)
	}

	// Setup logger
	var level slog.Level
	switch logLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	// Use log config if specified
	if cfg.Log != nil && logLevel == "" {
		switch cfg.Log.Level {
		case "debug", "trace":
			level = slog.LevelDebug
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		default:
			level = slog.LevelInfo
		}
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("Received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Create robot
	r, err := robot.New(ctx, cfg, robot.WithLogger(logger))
	if err != nil {
		return fmt.Errorf("failed to create robot: %w", err)
	}

	// Start robot
	if err := r.Start(ctx); err != nil {
		return fmt.Errorf("failed to start robot: %w", err)
	}

	logger.Info("Robot started", "name", cfg.Robot.Name)

	// Start external services
	var extManager *runtime.Manager
	externalServices := cfg.GetExternalServices()
	managedCount := 0
	for _, svc := range externalServices {
		if !svc.Disabled && svc.IsManaged() {
			managedCount++
		}
	}

	if managedCount > 0 {
		logger.Info("Starting external services", "count", managedCount)
		extManager = runtime.NewManager(cfg, configDir)

		for i := range externalServices {
			svc := &externalServices[i]
			if svc.Disabled || !svc.IsManaged() {
				continue
			}

			logger.Info("Starting external service", "name", svc.Name)
			if err := extManager.Start(ctx, svc); err != nil {
				logger.Error("Failed to start external service", "name", svc.Name, "error", err)
			} else {
				logger.Info("External service started", "name", svc.Name)
			}
		}
	}

	// Run until context is cancelled
	if err := r.Run(ctx); err != nil && err != context.Canceled {
		logger.Error("Robot error", "error", err)
	}

	// Stop external services
	if extManager != nil {
		logger.Info("Stopping external services")
		stopCtx := context.Background()
		if err := extManager.StopAll(stopCtx); err != nil {
			logger.Error("Error stopping external services", "error", err)
		}
	}

	// Stop robot
	stopCtx := context.Background()
	if err := r.Stop(stopCtx); err != nil {
		logger.Error("Error stopping robot", "error", err)
	}

	logger.Info("Robot stopped", "name", cfg.Robot.Name)
	return nil
}

func printRunUsage() error {
	fmt.Println(`gorai run - Run robot directly (monolithic mode)

Usage:
  gorai run [--config robot.json] [flags]

This command runs the robot as a single process with all components and internal
services in one binary. External services (configured with external.enabled=true)
are spawned as child processes if external.managed=true.

Flags:
  -c, --config <file>     Path to robot configuration file
  --log-level <level>     Log level: debug, info, warn, error (default: info)
  -h, --help              Show this help message

Examples:
  gorai run --config robot.json
  gorai run -c robot.json --log-level debug
  gorai run robot.json

The robot runs in the foreground and can be stopped with Ctrl+C (SIGINT) or
SIGTERM. For background/daemon operation, use 'gorai start' to run as a
systemd service.

Note: NATS must be running before starting the robot. Install NATS natively:
  sudo apt install nats-server
  sudo systemctl start nats-server`)
	return nil
}
