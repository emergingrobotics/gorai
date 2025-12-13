package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/gorai/gorai/pkg/compose"
	"github.com/gorai/gorai/pkg/config"
)

func cmdStatus() error {
	// Parse flags
	var configPath string
	var showAll bool

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("--config requires a value")
			}
			i++
			configPath = args[i]
		case "-a", "--all":
			showAll = true
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

	// Get compose file path
	composePath := compose.GetComposePath(configPath, cfg.Robot.Name)
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		fmt.Printf("Robot: %s\n", cfg.Robot.Name)
		fmt.Println("Status: Not started (compose file not found)")
		fmt.Println("\nRun 'gorai start' to start the robot.")
		return nil
	}

	// Create runner
	projectDir := compose.GetProjectDir(configPath)
	runner := compose.NewRunner(composePath, cfg.Robot.Name, projectDir)

	// Print robot info
	fmt.Printf("Robot: %s\n", cfg.Robot.Name)
	if cfg.Robot.Description != "" {
		fmt.Printf("Description: %s\n", cfg.Robot.Description)
	}
	fmt.Println()

	// Get container status
	ctx := context.Background()
	output, err := runner.Ps(ctx, showAll)
	if err != nil {
		fmt.Println("Container status: Unable to retrieve")
		fmt.Printf("Error: %v\n", err)
	} else if output != "" {
		fmt.Println("CONTAINERS")
		fmt.Println(output)
	} else {
		fmt.Println("No containers running.")
	}

	// Show component mapping
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
  -a, --all               Show all containers (including stopped)
  -h, --help              Show this help message

Examples:
  gorai status --config robot.json
  gorai status -c robot.json --all`)
	return nil
}
