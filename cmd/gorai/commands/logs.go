package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/quadlet"
)

func cmdLogs() error {
	// Parse flags
	var configPath string
	var follow bool
	var tail int
	var timestamps bool
	var containers []string

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
		case "-n", "--tail":
			if i+1 >= len(args) {
				return fmt.Errorf("--tail requires a value")
			}
			i++
			n, err := strconv.Atoi(args[i])
			if err != nil {
				return fmt.Errorf("invalid tail value: %s", args[i])
			}
			tail = n
		case "-t", "--timestamps":
			timestamps = true
		case "--container":
			if i+1 >= len(args) {
				return fmt.Errorf("--container requires a value")
			}
			i++
			containers = append(containers, args[i])
		case "-h", "--help":
			return printLogsUsage()
		default:
			if args[i][0] != '-' {
				if configPath == "" {
					configPath = args[i]
				} else {
					// Treat additional args as container names
					containers = append(containers, args[i])
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

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if containers are defined
	if cfg.Containers == nil || len(cfg.Containers) == 0 {
		return fmt.Errorf("no containers defined in %s", configPath)
	}

	// Get quadlet directory
	workspaceDir := filepath.Dir(configPath)
	if !filepath.IsAbs(workspaceDir) {
		workspaceDir, _ = filepath.Abs(workspaceDir)
	}
	quadletDir := filepath.Join(workspaceDir, ".gorai")

	// Create runner
	runner := quadlet.NewRunner(cfg.Robot.Name, quadletDir, true)

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	// Get logs using journalctl
	opts := quadlet.LogsOptions{
		Containers: containers,
		Follow:     follow,
		Tail:       tail,
		Timestamps: timestamps,
	}

	if err := runner.Logs(ctx, opts); err != nil {
		// Ignore context canceled errors (user pressed Ctrl+C)
		if ctx.Err() != nil {
			return nil
		}
		return fmt.Errorf("failed to get logs: %w", err)
	}

	return nil
}

func printLogsUsage() error {
	fmt.Println(`gorai logs - View container logs (via journalctl)

Usage:
  gorai logs [--config robot.json] [flags] [container...]

Flags:
  -c, --config <file>     Path to robot configuration file
  -f, --follow            Follow log output
  -n, --tail <lines>      Number of lines to show from end of logs
  -t, --timestamps        Show timestamps in ISO format
  --container <name>      Show logs for specific container
  -h, --help              Show this help message

Examples:
  gorai logs --config robot.json
  gorai logs -c robot.json --follow
  gorai logs --config robot.json --tail 100 --container gorai-hailo
  gorai logs -c robot.json -f gorai-core`)
	return nil
}
