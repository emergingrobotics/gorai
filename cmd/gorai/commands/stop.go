package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gorai/gorai/pkg/config"
)

func cmdStop() error {
	// Parse flags
	var configPath string
	var uninstall bool
	var disable bool
	var userMode bool = true

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
		case "--system":
			userMode = false
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

	serviceName := cfg.Robot.Name

	// Stop service
	fmt.Printf("Stopping robot %q...\n", serviceName)
	if err := systemctlCmd(userMode, "stop", serviceName+".service"); err != nil {
		fmt.Printf("Note: service may not be running: %v\n", err)
	} else {
		fmt.Println("Robot stopped.")
	}

	// Disable service if requested
	if disable {
		fmt.Println("Disabling service from auto-start...")
		if err := systemctlCmd(userMode, "disable", serviceName+".service"); err != nil {
			fmt.Printf("Warning: failed to disable service: %v\n", err)
		}
	}

	// Uninstall service file if requested
	if uninstall {
		fmt.Println("Removing service file...")

		var serviceFilePath string
		if userMode {
			home, _ := os.UserHomeDir()
			serviceFilePath = filepath.Join(home, ".config", "systemd", "user", serviceName+".service")
		} else {
			serviceFilePath = filepath.Join("/etc/systemd/system", serviceName+".service")
		}

		if err := os.Remove(serviceFilePath); err != nil {
			if !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove service file: %w", err)
			}
			fmt.Println("Service file not found (already removed).")
		} else {
			fmt.Printf("Removed: %s\n", serviceFilePath)
		}

		// Reload systemd
		if err := systemctlCmd(userMode, "daemon-reload"); err != nil {
			fmt.Printf("Warning: failed to reload systemd: %v\n", err)
		}
	}

	return nil
}

func printStopUsage() error {
	fmt.Println(`gorai stop - Stop robot systemd service

Usage:
  gorai stop [--config robot.json] [flags]

Flags:
  -c, --config <file>     Path to robot configuration file
  --disable               Disable service from auto-start at boot
  --uninstall             Remove service file from systemd
  --system                Use system mode (default: user mode)
  -h, --help              Show this help message

Examples:
  gorai stop --config robot.json
  gorai stop -c robot.json --uninstall
  gorai stop --config robot.json --disable
  sudo gorai stop --config robot.json --system`)
	return nil
}
