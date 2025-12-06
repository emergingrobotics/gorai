// Package commands implements the gorai CLI commands.
package commands

import (
	"fmt"
	"os"
)

// Execute runs the CLI.
func Execute() error {
	if len(os.Args) < 2 {
		return printUsage()
	}

	switch os.Args[1] {
	case "version":
		return cmdVersion()
	case "run":
		return cmdRun()
	case "topic":
		return cmdTopic()
	case "service":
		return cmdService()
	case "help", "-h", "--help":
		return printUsage()
	default:
		return fmt.Errorf("unknown command: %s", os.Args[1])
	}
}

func printUsage() error {
	fmt.Println(`gorai - Gorai Robotics Framework CLI

Usage:
  gorai <command> [arguments]

Commands:
  version     Print version information
  run         Run a robot from configuration
  topic       Topic operations (list, echo, pub)
  service     Service operations (list, call)
  help        Print this help message

Use "gorai <command> -h" for more information about a command.`)
	return nil
}

func cmdVersion() error {
	fmt.Println("gorai version 0.1.0")
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
