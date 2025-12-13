package commands

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/gorai/gorai/pkg/compose"
	"github.com/gorai/gorai/pkg/config"
)

func cmdStop() error {
	// Parse flags
	var configPath string
	var removeVolumes bool
	var timeout int

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("--config requires a value")
			}
			i++
			configPath = args[i]
		case "-v", "--volumes":
			removeVolumes = true
		case "-t", "--timeout":
			if i+1 >= len(args) {
				return fmt.Errorf("--timeout requires a value")
			}
			i++
			t, err := strconv.Atoi(args[i])
			if err != nil {
				return fmt.Errorf("invalid timeout value: %s", args[i])
			}
			timeout = t
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

	// Check if containers are defined
	if cfg.Containers == nil || len(cfg.Containers) == 0 {
		return fmt.Errorf("no containers defined in %s", configPath)
	}

	// Get compose file path
	composePath := compose.GetComposePath(configPath, cfg.Robot.Name)
	if _, err := os.Stat(composePath); os.IsNotExist(err) {
		return fmt.Errorf("compose file not found: %s\nRun 'gorai start' first to generate it", composePath)
	}

	// Create runner
	projectDir := compose.GetProjectDir(configPath)
	runner := compose.NewRunner(composePath, cfg.Robot.Name, projectDir)

	// Run podman-compose down
	fmt.Printf("Stopping containers for robot %q...\n", cfg.Robot.Name)

	ctx := context.Background()
	opts := compose.DownOptions{
		Volumes: removeVolumes,
		Timeout: timeout,
	}

	if err := runner.Down(ctx, opts); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	fmt.Println("Containers stopped.")
	return nil
}

func printStopUsage() error {
	fmt.Println(`gorai stop - Stop robot containers

Usage:
  gorai stop [--config robot.json] [flags]

Flags:
  -c, --config <file>     Path to robot configuration file
  -v, --volumes           Remove named volumes
  -t, --timeout <secs>    Timeout in seconds for container shutdown
  -h, --help              Show this help message

Examples:
  gorai stop --config robot.json
  gorai stop -c robot.json --volumes
  gorai stop --config robot.json --timeout 30`)
	return nil
}
