// Package commands implements the gorai CLI commands.
package commands

import (
	"fmt"
	"os"
)

// Version information (set by ldflags at build time)
var (
	Version = "0.1.0"
	Commit  = "unknown"
	Date    = "unknown"
)

// Execute runs the CLI.
func Execute() error {
	if len(os.Args) < 2 {
		return printUsage()
	}

	switch os.Args[1] {
	// Core commands
	case "validate":
		return cmdValidate()
	case "run":
		return cmdRun()
	case "build":
		return cmdBuild()

	// Component management
	case "components":
		return cmdList()
	case "list":
		// Alias for backwards compatibility
		return cmdList()
	case "component":
		return cmdComponent()

	// Mesh/service discovery
	case "mesh":
		return cmdMesh()

	// Utility commands
	case "version":
		return cmdVersion()
	case "migrate":
		return cmdMigrate()

	// Help
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown command: %s\n\nRun 'gorai help' for usage.", os.Args[1])
	}
}

func printUsage() error {
	fmt.Printf(`gorai - Gorai Robotics Framework CLI (v%s)

The robotics platform for the AI era

Usage:
  gorai <command> [flags]

Core Commands:
  validate <config>   Validate RDL configuration file
  run <config>        Run robot in development mode (foreground)
  build <config>      Build standalone binary for deployment

Component Commands:
  components          List available component types
  component search    Search for third-party components
  component info      Show component information
  component add       Add component to project

Mesh Commands (Service Discovery):
  mesh services       List running services in the mesh
  mesh channels       List registered NATS channels
  mesh schemas        List or show message schemas
  mesh watch          Watch for services joining/leaving
  mesh summary        Show mesh state summary
  mesh robots         List robots with registered services
  mesh init           Initialize mesh with predefined schemas
  mesh reset          Reset mesh (delete all data)

Utility Commands:
  version             Print version information
  migrate             Migrate RDL v1 config to v2 format
  help                Print this help message

Use "gorai <command> -h" for more information about a command.

Examples:
  # Validate configuration
  gorai validate robot.json

  # Run robot in development mode
  gorai run robot.json

  # Build standalone binary
  gorai build robot.json -o my-robot --target linux/arm64

  # List available components
  gorai components

  # Discover running services
  gorai mesh services robot-alpha

  # Watch mesh in real-time
  gorai mesh watch

Quick Start:
  1. Create robot.json with your components
  2. gorai validate robot.json
  3. gorai run robot.json

Documentation:
  https://gorai.dev/docs

`, Version)
	return nil
}

func cmdVersion() error {
	fmt.Printf("gorai version %s\n", Version)
	fmt.Printf("  commit: %s\n", Commit)
	fmt.Printf("  built:  %s\n", Date)
	return nil
}

// cmdComponent bridges to the new cobra-based component commands
func cmdComponent() error {
	cmd := NewComponentCmd()
	// Set args to skip "gorai component" and pass the rest
	cmd.SetArgs(os.Args[2:])
	return cmd.Execute()
}

// findConfigFile looks for common config file names in the current directory.
func findConfigFile() string {
	// Check environment variable
	if name := os.Getenv("GORAI_ROBOT_NAME"); name != "" {
		path := name + ".json"
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Check common names
	candidates := []string{"robot.json", "gorai.json", "robot.rdl.json"}
	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}

	return ""
}
