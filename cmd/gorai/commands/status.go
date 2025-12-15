package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/runtime"
)

func cmdStatus() error {
	// Parse flags
	var configPath string
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
		case "--system":
			userMode = false
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

	// Make config path absolute
	if !filepath.IsAbs(configPath) {
		configPath, _ = filepath.Abs(configPath)
	}

	// Load configuration with Service RDL support
	cfg, err := config.LoadWithServiceRDL(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	serviceName := cfg.Robot.Name

	// Print robot info
	fmt.Printf("Robot: %s\n", cfg.Robot.Name)
	if cfg.Robot.Description != "" {
		fmt.Printf("Description: %s\n", cfg.Robot.Description)
	}
	fmt.Println()

	// Show systemd service status
	fmt.Println("SERVICE STATUS")
	fmt.Println("─────────────────────────────────────────────────────────")

	if err := systemctlCmd(userMode, "status", serviceName+".service", "--no-pager"); err != nil {
		// Service might not exist or not be running
		fmt.Printf("\nService %s.service is not running or not installed.\n", serviceName)
		fmt.Println("\nTo start the robot:")
		fmt.Printf("  gorai start --config %s\n", configPath)
	}

	// Show components
	if len(cfg.Components) > 0 {
		fmt.Println("\nCOMPONENTS")
		fmt.Printf("%-20s %-15s %-10s\n", "NAME", "TYPE", "MODEL")
		fmt.Println("─────────────────────────────────────────────────────")
		for _, comp := range cfg.Components {
			status := ""
			if comp.Disabled {
				status = " (disabled)"
			}
			fmt.Printf("%-20s %-15s %-10s%s\n", comp.Name, comp.Type, comp.Model, status)
		}
	}

	// Show services
	if len(cfg.Services) > 0 {
		fmt.Println("\nSERVICES")
		fmt.Printf("%-20s %-15s %-10s %-10s\n", "NAME", "TYPE", "MODEL", "EXTERNAL")
		fmt.Println("─────────────────────────────────────────────────────────────")
		for _, svc := range cfg.Services {
			external := ""
			if svc.IsExternal() {
				if svc.IsManaged() {
					external = "managed"
				} else {
					external = "unmanaged"
				}
			} else {
				external = "internal"
			}
			status := ""
			if svc.Disabled {
				status = " (disabled)"
			}
			fmt.Printf("%-20s %-15s %-10s %-10s%s\n", svc.Name, svc.Type, svc.Model, external, status)
		}
	}

	// Show external service status
	externalServices := cfg.GetExternalServices()
	managedCount := 0
	for _, svc := range externalServices {
		if !svc.Disabled && svc.IsManaged() {
			managedCount++
		}
	}

	if managedCount > 0 {
		fmt.Println("\nEXTERNAL SERVICE STATUS")
		fmt.Printf("%-20s %-12s %-10s %-15s\n", "NAME", "RUNTIME", "STATUS", "DETAILS")
		fmt.Println("─────────────────────────────────────────────────────────────")

		for i := range externalServices {
			svc := &externalServices[i]
			if svc.Disabled || !svc.IsManaged() {
				continue
			}

			runtimeType := "process"
			if svc.External.Container != nil {
				runtimeType = "container"
			}

			// Check status directly using runtime helpers
			var status runtime.ServiceStatus
			if svc.External.Container != nil {
				status = runtime.CheckContainerStatus(svc.Name)
			} else {
				status = runtime.CheckProcessStatus(svc.Name)
			}

			statusStr := "stopped"
			if status.Running {
				statusStr = "running"
			}

			details := ""
			if status.Container != "" {
				details = status.Container
			} else if status.PID > 0 {
				details = fmt.Sprintf("pid=%d", status.PID)
			}
			if status.Error != "" {
				details = status.Error
			}

			fmt.Printf("%-20s %-12s %-10s %-15s\n", svc.Name, runtimeType, statusStr, details)
		}
	}

	return nil
}

func printStatusUsage() error {
	fmt.Println(`gorai status - Show robot service status

Usage:
  gorai status [--config robot.json] [flags]

Flags:
  -c, --config <file>     Path to robot configuration file
  --system                Use system mode (default: user mode)
  -h, --help              Show this help message

Examples:
  gorai status --config robot.json
  gorai status -c robot.json`)
	return nil
}

// systemctlStatusCmd runs systemctl status without capturing output (direct to terminal)
func systemctlStatusCmd(userMode bool, serviceName string) error {
	args := []string{"status", serviceName, "--no-pager"}
	if userMode {
		args = append([]string{"--user"}, args...)
	}
	cmd := exec.Command("systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
