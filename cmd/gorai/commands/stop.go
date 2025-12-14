package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/quadlet"
)

func cmdStop() error {
	// Parse flags
	var configPath string
	var uninstall bool
	var disable bool
	var containers []string

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("--config requires a value")
			}
			i++
			configPath = args[i]
		case "--uninstall":
			uninstall = true
		case "--disable":
			disable = true
		case "--containers":
			if i+1 >= len(args) {
				return fmt.Errorf("--containers requires a value")
			}
			i++
			containers = splitComma(args[i])
		case "-h", "--help":
			return printStopUsage()
		default:
			if args[i][0] != '-' {
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

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if containers are defined
	if cfg.Containers == nil || len(cfg.Containers) == 0 {
		return fmt.Errorf("no containers defined in %s", configPath)
	}

	// Get quadlet directory
	workspaceDir := filepath.Dir(configPath)
	if !filepath.IsAbs(workspaceDir) {
		workspaceDir, _ = filepath.Abs(workspaceDir)
	}
	quadletDir := filepath.Join(workspaceDir, ".gorai")

	// Create runner
	runner := quadlet.NewRunner(cfg.Robot.Name, quadletDir, true)

	// Disable services if requested
	if disable {
		fmt.Println("Disabling services from auto-start...")
		if err := runner.Disable(context.Background(), containers...); err != nil {
			fmt.Printf("Warning: failed to disable services: %v\n", err)
		}
	}

	// Stop containers
	fmt.Printf("Stopping containers for robot %q...\n", cfg.Robot.Name)

	ctx := context.Background()
	if err := runner.Stop(ctx, containers...); err != nil {
		// Don't fail if services aren't running
		fmt.Printf("Note: %v\n", err)
	}

	fmt.Println("Containers stopped.")

	// Uninstall Quadlet files if requested
	if uninstall {
		fmt.Println("Uninstalling Quadlet files from systemd...")
		if err := runner.Uninstall(ctx); err != nil {
			return fmt.Errorf("failed to uninstall Quadlet files: %w", err)
		}
		fmt.Println("Quadlet files removed.")
	}

	return nil
}

func printStopUsage() error {
	fmt.Println(`gorai stop - Stop robot containers

Usage:
  gorai stop [--config robot.json] [flags]

Flags:
  -c, --config <file>     Path to robot configuration file
  --disable               Disable services from auto-start at boot
  --uninstall             Remove Quadlet files from systemd
  --containers <list>     Stop only specific containers (comma-separated)
  -h, --help              Show this help message

Examples:
  gorai stop --config robot.json
  gorai stop -c robot.json --uninstall
  gorai stop --config robot.json --disable
  gorai stop --config robot.json --containers nats,gorai-core`)
	return nil
}
