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

	// Runtime commands
	case "run":
		return cmdRun()
	case "start":
		return cmdStart()
	case "stop":
		return cmdStop()
	case "status":
		return cmdStatus()
	case "logs":
		return cmdLogs()

	// Build commands
	case "build":
		return cmdBuild()
	case "migrate":
		return cmdMigrate()

	// Topic/service commands
	case "version":
		return cmdVersion()
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

Runtime Commands (Recommended):
  run               Run robot directly (foreground, monolithic mode)
  start             Start robot as native systemd service
  stop              Stop robot systemd service
  status            Show robot process status
  logs              View robot logs

Build Commands:
  build             Build container images for external services
  migrate           Migrate RDL v1 config to v2 format

Topic/Service Commands:
  topic             Topic operations (list, echo, pub)
  service           Service operations (list, call)

Other Commands:
  version           Print version information
  help              Print this help message

Use "gorai <command> -h" for more information about a command.

Examples:
  # Run robot directly (recommended):
  gorai run --config robot.json

  # Deploy as systemd service:
  gorai start --config robot.json --enable
  gorai status --config robot.json
  gorai logs --config robot.json --follow
  gorai stop --config robot.json

  # Project management:
  gorai init my-robot              Initialize project from my-robot.json
  gorai validate my-robot          Validate my-robot.json
  gorai add component my_sensor    Add a custom component

  # Migrate old config:
  gorai migrate --config old-robot.json --output robot.json

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
