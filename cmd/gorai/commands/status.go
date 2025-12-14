package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/systemd"
)

func cmdStatus() error {
	// Parse flags
	var configPath string

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("--config requires a value")
			}
			i++
			configPath = args[i]
		case "-h", "--help":
			return printStatusUsage()
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

	// Print robot info
	fmt.Printf("Robot: %s\n", cfg.Robot.Name)
	if cfg.Robot.Description != "" {
		fmt.Printf("Description: %s\n", cfg.Robot.Description)
	}
	fmt.Println()

	// Get service directory
	workspaceDir := filepath.Dir(configPath)
	if !filepath.IsAbs(workspaceDir) {
		workspaceDir, _ = filepath.Abs(workspaceDir)
	}
	serviceDir := filepath.Join(workspaceDir, ".gorai")

	// Create runner
	runner := systemd.NewRunner(cfg.Robot.Name, serviceDir, true)

	// Get container status
	ctx := context.Background()
	statuses, err := runner.GetStatus(ctx)
	if err != nil {
		fmt.Println("Container status: Unable to retrieve")
		fmt.Printf("Error: %v\n", err)
	} else if len(statuses) > 0 {
		fmt.Println("CONTAINERS")
		fmt.Printf("%-30s %-12s %-15s\n", "SERVICE", "ACTIVE", "STATUS")
		fmt.Println("─────────────────────────────────────────────────────────")
		for _, s := range statuses {
			fmt.Printf("%-30s %-12s %-15s\n", s.Service, s.Active, s.Status)
		}
	} else {
		fmt.Println("No container services found.")
		fmt.Println("Run 'gorai start' to start the robot.")
	}

	// Show component mapping
	if len(cfg.Components) > 0 {
		fmt.Println("\nCOMPONENT → CONTAINER MAPPING")
		fmt.Printf("%-20s %-20s %-10s\n", "COMPONENT", "CONTAINER", "TYPE")
		fmt.Println("─────────────────────────────────────────────────────")
		for _, comp := range cfg.Components {
			container := comp.Container
			if container == "" {
				container = "(default)"
			}
			fmt.Printf("%-20s %-20s %-10s\n", comp.Name, container, comp.Type)
		}
	}

	// Show service mapping
	if len(cfg.Services) > 0 {
		fmt.Println("\nSERVICE → CONTAINER MAPPING")
		fmt.Printf("%-20s %-20s %-10s\n", "SERVICE", "CONTAINER", "TYPE")
		fmt.Println("─────────────────────────────────────────────────────")
		for _, svc := range cfg.Services {
			container := svc.Container
			if container == "" {
				container = "(default)"
			}
			fmt.Printf("%-20s %-20s %-10s\n", svc.Name, container, svc.Type)
		}
	}

	return nil
}

func printStatusUsage() error {
	fmt.Println(`gorai status - Show robot container status

Usage:
  gorai status [--config robot.json] [flags]

Flags:
  -c, --config <file>     Path to robot configuration file
  -h, --help              Show this help message

Examples:
  gorai status --config robot.json
  gorai status -c robot.json`)
	return nil
}
