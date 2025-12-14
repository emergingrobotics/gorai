package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

func cmdStart() error {
	// Parse flags
	var configPath string
	var enable bool
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
		case "--enable":
			enable = true
		case "--system":
			userMode = false
		case "-h", "--help":
			return printStartUsage()
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

	// Make paths absolute
	configPath, _ = filepath.Abs(configPath)

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Check for deprecation warnings
	warnings := cfg.DeprecationWarnings()
	for _, w := range warnings {
		fmt.Fprintf(os.Stderr, "WARNING: %s\n", w)
	}

	// Find gorai binary
	goraiBin, err := findGoraiBinary()
	if err != nil {
		return fmt.Errorf("cannot find gorai binary: %w", err)
	}

	// Generate native systemd service file
	serviceName := cfg.Robot.Name
	serviceFile := generateNativeServiceFile(serviceName, goraiBin, configPath, userMode)

	// Determine target directory
	var targetDir string
	if userMode {
		home, _ := os.UserHomeDir()
		targetDir = filepath.Join(home, ".config", "systemd", "user")
	} else {
		targetDir = "/etc/systemd/system"
	}

	// Ensure directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create systemd directory: %w", err)
	}

	// Write service file
	serviceFilePath := filepath.Join(targetDir, serviceName+".service")
	if err := os.WriteFile(serviceFilePath, []byte(serviceFile), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}
	fmt.Printf("Created service file: %s\n", serviceFilePath)

	// Reload systemd
	fmt.Println("Reloading systemd daemon...")
	if err := systemctlCmd(userMode, "daemon-reload"); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable if requested
	if enable {
		fmt.Println("Enabling service for auto-start at boot...")
		if err := systemctlCmd(userMode, "enable", serviceName+".service"); err != nil {
			fmt.Printf("Warning: failed to enable service: %v\n", err)
		}

		// For user mode, enable lingering so service runs without login
		if userMode {
			user := os.Getenv("USER")
			if user != "" {
				cmd := exec.Command("loginctl", "enable-linger", user)
				if err := cmd.Run(); err != nil {
					fmt.Printf("Warning: failed to enable linger (service may not start at boot): %v\n", err)
				}
			}
		}
	}

	// Start service
	fmt.Printf("Starting robot %q...\n", cfg.Robot.Name)
	if err := systemctlCmd(userMode, "start", serviceName+".service"); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	fmt.Printf("\nRobot started successfully.\n")
	fmt.Printf("Use 'gorai status --config %s' to check status.\n", configPath)
	fmt.Printf("Use 'gorai logs --config %s -f' to view logs.\n", configPath)
	fmt.Printf("Use 'gorai stop --config %s' to stop.\n", configPath)

	return nil
}

func generateNativeServiceFile(name, binary, configPath string, userMode bool) string {
	var sb strings.Builder

	sb.WriteString("[Unit]\n")
	sb.WriteString(fmt.Sprintf("Description=Gorai Robot: %s\n", name))
	sb.WriteString("After=network-online.target nats-server.service\n")
	sb.WriteString("Wants=network-online.target nats-server.service\n")
	sb.WriteString("\n")

	sb.WriteString("[Service]\n")
	sb.WriteString("Type=simple\n")
	sb.WriteString(fmt.Sprintf("ExecStart=%s run --config %s\n", binary, configPath))
	sb.WriteString("Restart=always\n")
	sb.WriteString("RestartSec=5\n")
	sb.WriteString("StandardOutput=journal\n")
	sb.WriteString("StandardError=journal\n")
	sb.WriteString(fmt.Sprintf("SyslogIdentifier=%s\n", name))
	sb.WriteString("\n")

	// Environment
	sb.WriteString(fmt.Sprintf("Environment=\"GORAI_ROBOT_NAME=%s\"\n", name))
	sb.WriteString("Environment=\"NATS_URL=nats://localhost:4222\"\n")
	sb.WriteString("\n")

	// Hardware access groups (for user mode)
	if userMode {
		sb.WriteString("# Hardware access - ensure user is in these groups\n")
	} else {
		sb.WriteString("# Hardware access groups\n")
		sb.WriteString("SupplementaryGroups=gpio i2c spi video dialout plugdev input\n")
	}
	sb.WriteString("\n")

	sb.WriteString("[Install]\n")
	if userMode {
		sb.WriteString("WantedBy=default.target\n")
	} else {
		sb.WriteString("WantedBy=multi-user.target\n")
	}

	return sb.String()
}

func systemctlCmd(userMode bool, args ...string) error {
	cmdArgs := args
	if userMode {
		cmdArgs = append([]string{"--user"}, args...)
	}
	cmd := exec.Command("systemctl", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func findGoraiBinary() (string, error) {
	// First try to find ourselves
	exe, err := os.Executable()
	if err == nil {
		abs, err := filepath.Abs(exe)
		if err == nil {
			return abs, nil
		}
		return exe, nil
	}

	// Try common locations
	candidates := []string{
		"/usr/local/bin/gorai",
		"/usr/bin/gorai",
		filepath.Join(os.Getenv("GOPATH"), "bin", "gorai"),
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("gorai binary not found")
}

func printStartUsage() error {
	fmt.Println(`gorai start - Start robot as native systemd service

Usage:
  gorai start [--config robot.json] [flags]

This command installs and starts the robot as a native systemd service.
The robot runs as a single process using 'gorai run' internally.

Flags:
  -c, --config <file>     Path to robot configuration file
  --enable                Enable service for auto-start at boot
  --system                Use system mode (requires root, default: user mode)
  -h, --help              Show this help message

Examples:
  gorai start --config robot.json
  gorai start --config robot.json --enable
  sudo gorai start --config robot.json --system --enable

Prerequisites:
  - NATS server must be running (install: sudo apt install nats-server)
  - For hardware access, ensure your user is in appropriate groups:
    sudo usermod -aG gpio,i2c,spi,video,dialout,plugdev $USER

Service Management:
  View status:  systemctl --user status <robot-name>
  View logs:    journalctl --user -u <robot-name> -f
  Stop:         gorai stop --config robot.json
  Restart:      systemctl --user restart <robot-name>`)
	return nil
}

func splitComma(s string) []string {
	if s == "" {
		return nil
	}
	var result []string
	for _, part := range splitString(s, ',') {
		part = trimSpace(part)
		if part != "" {
			result = append(result, part)
		}
	}
	return result
}

func splitString(s string, sep rune) []string {
	var result []string
	var current []rune
	for _, r := range s {
		if r == sep {
			result = append(result, string(current))
			current = nil
		} else {
			current = append(current, r)
		}
	}
	result = append(result, string(current))
	return result
}

func trimSpace(s string) string {
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t') {
		end--
	}
	return s[start:end]
}

func findConfigFile() string {
	// Check environment variable
	if name := os.Getenv("GORAI_ROBOT_NAME"); name != "" {
		path := name + ".json"
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// Check common names
	candidates := []string{"robot.json", "gorai.json"}
	for _, name := range candidates {
		if _, err := os.Stat(name); err == nil {
			return name
		}
	}

	return ""
}
