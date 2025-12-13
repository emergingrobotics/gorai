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
	// Project commands
	case "init":
		return cmdInit()
	case "generate":
		return cmdGenerate()
	case "validate":
		return cmdValidate()
	case "add":
		return cmdAdd()
	case "list":
		return cmdList()

	// Container orchestration commands
	case "start":
		return cmdStart()
	case "stop":
		return cmdStop()
	case "status":
		return cmdStatus()
	case "logs":
		return cmdLogs()
	case "build":
		return cmdBuild()

	// Runtime commands
	case "version":
		return cmdVersion()
	case "run":
		return cmdRun()
	case "topic":
		return cmdTopic()
	case "service":
		return cmdService()

	// Help
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown command: %s\n\nRun 'gorai help' for usage.", os.Args[1])
	}
}

func printUsage() error {
	fmt.Printf(`gorai - Gorai Robotics Framework CLI (v%s)

Usage:
  gorai <command> [flags]

Project Commands:
  init <name>       Initialize project from <name>.json (must exist)
  generate <name>   Generate code from <name>.json
  validate <name>   Validate <name>.json configuration
  add               Add custom components or services
  list              List available component and service types

Container Orchestration (Podman):
  start             Start robot containers from RDL configuration
  stop              Stop robot containers
  status            Show robot and container status
  logs              View container logs
  build             Build container images

Runtime Commands:
  run               Run a robot from configuration (monolithic mode)
  topic             Topic operations (list, echo, pub)
  service           Service operations (list, call)

Other Commands:
  version           Print version information
  help              Print this help message

Use "gorai <command> -h" for more information about a command.

Examples:
  # Container orchestration (recommended):
  gorai start --config robot.json --build --detach
  gorai status --config robot.json
  gorai logs --config robot.json --follow
  gorai stop --config robot.json

  # Project management:
  gorai init my-robot              Initialize project from my-robot.json
  gorai validate my-robot          Validate my-robot.json
  gorai add component my_sensor    Add a custom component

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

func cmdRun() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: gorai run <config.json>")
	}
	configPath := os.Args[2]
	fmt.Printf("Running robot from %s...\n", configPath)
	// TODO: Load config and start robot
	return nil
}

func cmdTopic() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: gorai topic <list|echo|pub> [args]")
	}
	subCmd := os.Args[2]
	switch subCmd {
	case "list":
		fmt.Println("Listing topics...")
		// TODO: List active topics
	case "echo":
		if len(os.Args) < 4 {
			return fmt.Errorf("usage: gorai topic echo <topic>")
		}
		fmt.Printf("Echoing topic %s...\n", os.Args[3])
		// TODO: Subscribe and print messages
	case "pub":
		if len(os.Args) < 5 {
			return fmt.Errorf("usage: gorai topic pub <topic> <message>")
		}
		fmt.Printf("Publishing to %s...\n", os.Args[3])
		// TODO: Publish message
	default:
		return fmt.Errorf("unknown topic command: %s", subCmd)
	}
	return nil
}

func cmdService() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: gorai service <list|call> [args]")
	}
	subCmd := os.Args[2]
	switch subCmd {
	case "list":
		fmt.Println("Listing services...")
		// TODO: List registered services
	case "call":
		if len(os.Args) < 5 {
			return fmt.Errorf("usage: gorai service call <service> <request>")
		}
		fmt.Printf("Calling service %s...\n", os.Args[3])
		// TODO: Call service
	default:
		return fmt.Errorf("unknown service command: %s", subCmd)
	}
	return nil
}
