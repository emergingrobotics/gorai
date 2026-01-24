package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/runtime"
)

func cmdLogs() error {
	// Parse flags
	var configPath string
	var follow bool
	var serviceName string
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
		case "-f", "--follow":
			follow = true
		case "-s", "--service":
			if i+1 >= len(args) {
				return fmt.Errorf("--service requires a value")
			}
			i++
			serviceName = args[i]
		case "--system":
			userMode = false
		case "-h", "--help":
			return printLogsUsage()
		default:
			if args[i][0] != '-' {
				if configPath == "" {
					configPath = args[i]
				} else if serviceName == "" {
					serviceName = args[i]
				}
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

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// If no service specified, show main robot logs via journalctl
	if serviceName == "" {
		return showJournalLogs(ctx, cfg.Robot.Name, follow, userMode)
	}

	// Check if this is an external service
	var targetService *config.ServiceConfig
	for i := range cfg.Services {
		if cfg.Services[i].Name == serviceName {
			targetService = &cfg.Services[i]
			break
		}
	}

	if targetService == nil {
		return fmt.Errorf("service %q not found", serviceName)
	}

	// If it's an internal service, use journalctl
	if !targetService.IsExternal() || !targetService.IsManaged() {
		return showJournalLogs(ctx, cfg.Robot.Name, follow, userMode)
	}

	// Use appropriate method for external services
	var reader io.ReadCloser
	if targetService.External.Container != nil {
		reader, err = runtime.GetContainerLogs(targetService.Name, follow)
	} else {
		reader, err = runtime.GetProcessLogs(targetService.Name, follow)
	}
	if err != nil {
		return fmt.Errorf("failed to get logs for %s: %w", serviceName, err)
	}
	defer reader.Close()

	// Stream logs to stdout
	_, err = io.Copy(os.Stdout, reader)
	if err != nil && ctx.Err() == nil {
		return fmt.Errorf("error reading logs: %w", err)
	}

	return nil
}

func showJournalLogs(ctx context.Context, robotName string, follow, userMode bool) error {
	args := []string{"-u", robotName + ".service", "--no-pager"}
	if follow {
		args = append(args, "-f")
	}
	if userMode {
		args = append([]string{"--user"}, args...)
	}

	cmd := exec.CommandContext(ctx, "journalctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if ctx.Err() != nil {
		return nil // Ignore if context was cancelled (user pressed Ctrl+C)
	}
	return err
}

func printLogsUsage() error {
	fmt.Println(`gorai logs - View robot and service logs

Usage:
  gorai logs [--config robot.json] [flags] [service-name]

Flags:
  -c, --config <file>     Path to robot configuration file
  -f, --follow            Follow log output (like tail -f)
  -s, --service <name>    Show logs for specific service
  --system                Use system mode for journalctl (default: user mode)
  -h, --help              Show this help message

Without a service name, shows the main robot logs via journalctl.
For external services (containers/processes), shows their specific logs.

Examples:
  gorai logs --config robot.json              # Show main robot logs
  gorai logs -c robot.json -f                 # Follow main robot logs
  gorai logs --config robot.json person_detector  # Show service logs
  gorai logs -c robot.json -s person_detector -f  # Follow service logs`)
	return nil
}
