// Command gorai-robot runs a robot from an RDL configuration file.
//
// This is the main entrypoint for containerized robot deployments.
// It loads the configuration, initializes all components and services,
// and runs until interrupted.
//
// Usage:
//
//	gorai-robot --config /etc/gorai/robot.json
package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/robot"
)

var (
	Version = "0.1.0"
	Commit  = "unknown"
	Date    = "unknown"
)

func main() {
	// Parse flags
	configPath := flag.String("config", "/etc/gorai/robot.json", "Path to robot configuration file")
	logLevel := flag.String("log-level", "", "Log level (debug, info, warn, error)")
	showVersion := flag.Bool("version", false, "Show version information")
	flag.Parse()

	if *showVersion {
		fmt.Printf("gorai-robot %s (commit: %s, built: %s)\n", Version, Commit, Date)
		os.Exit(0)
	}

	// Setup logging
	level := slog.LevelInfo
	if *logLevel != "" {
		switch *logLevel {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	} else if envLevel := os.Getenv("LOG_LEVEL"); envLevel != "" {
		switch envLevel {
		case "debug":
			level = slog.LevelDebug
		case "info":
			level = slog.LevelInfo
		case "warn":
			level = slog.LevelWarn
		case "error":
			level = slog.LevelError
		}
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})
	logger := slog.New(handler)
	slog.SetDefault(logger)

	// Load configuration
	logger.Info("Loading configuration", "path", *configPath)
	cfg, err := config.Load(*configPath)
	if err != nil {
		logger.Error("Failed to load configuration", "error", err)
		os.Exit(1)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		logger.Error("Invalid configuration", "error", err)
		os.Exit(1)
	}

	logger.Info("Configuration loaded",
		"robot", cfg.Robot.Name,
		"components", len(cfg.Components),
		"services", len(cfg.Services),
	)

	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		logger.Info("Received signal, shutting down", "signal", sig)
		cancel()
	}()

	// Create and start robot
	r, err := robot.New(ctx, cfg, robot.WithLogger(logger))
	if err != nil {
		logger.Error("Failed to create robot", "error", err)
		os.Exit(1)
	}

	if err := r.Start(ctx); err != nil {
		logger.Error("Failed to start robot", "error", err)
		os.Exit(1)
	}

	// Run until context is cancelled
	logger.Info("Robot running", "name", cfg.Robot.Name)
	if err := r.Run(ctx); err != nil && err != context.Canceled {
		logger.Error("Robot error", "error", err)
	}

	// Graceful shutdown
	logger.Info("Shutting down robot")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*1000*1000*1000) // 30s
	defer shutdownCancel()

	if err := r.Stop(shutdownCtx); err != nil {
		logger.Error("Error during shutdown", "error", err)
		os.Exit(1)
	}

	logger.Info("Robot stopped")
}
