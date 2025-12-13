package commands

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gorai/gorai/pkg/compose"
	"github.com/gorai/gorai/pkg/config"
)

func cmdStart() error {
	// Parse flags
	var configPath string
	var detach, build, forceRecreate bool
	var containers []string

	// Simple flag parsing
	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("--config requires a value")
			}
			i++
			configPath = args[i]
		case "-d", "--detach":
			detach = true
		case "--build":
			build = true
		case "--force-recreate":
			forceRecreate = true
		case "--containers":
			if i+1 >= len(args) {
				return fmt.Errorf("--containers requires a value")
			}
			i++
			containers = splitComma(args[i])
		case "-h", "--help":
			return printStartUsage()
		default:
			if args[i][0] != '-' {
				// Treat as config path if not a flag
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

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Check if containers are defined
	if cfg.Containers == nil || len(cfg.Containers) == 0 {
		return fmt.Errorf("no containers defined in %s. Add a 'containers' section to use gorai start", configPath)
	}

	// Generate compose file
	composePath := compose.GetComposePath(configPath, cfg.Robot.Name)
	composeDir := filepath.Dir(composePath)
	if err := os.MkdirAll(composeDir, 0755); err != nil {
		return fmt.Errorf("failed to create compose directory: %w", err)
	}

	fmt.Printf("Generating compose file: %s\n", composePath)
	gen := compose.NewGenerator(cfg)
	if err := gen.WriteYAML(composePath); err != nil {
		return fmt.Errorf("failed to generate compose file: %w", err)
	}

	// Create runner
	projectDir := compose.GetProjectDir(configPath)
	runner := compose.NewRunner(composePath, cfg.Robot.Name, projectDir)

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, stopping containers...")
		cancel()
	}()

	// Run podman-compose up
	fmt.Printf("Starting containers for robot %q...\n", cfg.Robot.Name)

	opts := compose.UpOptions{
		Detach:        detach,
		Build:         build,
		ForceRecreate: forceRecreate,
		Services:      containers,
	}

	if err := runner.Up(ctx, opts); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}

	if detach {
		fmt.Printf("\nContainers started in background.\n")
		fmt.Printf("Use 'gorai status --config %s' to check status.\n", configPath)
		fmt.Printf("Use 'gorai logs --config %s' to view logs.\n", configPath)
		fmt.Printf("Use 'gorai stop --config %s' to stop.\n", configPath)
	}

	return nil
}

func printStartUsage() error {
	fmt.Println(`gorai start - Start robot containers

Usage:
  gorai start [--config robot.json] [flags]

Flags:
  -c, --config <file>     Path to robot configuration file
  -d, --detach            Run containers in background
  --build                 Build images before starting
  --force-recreate        Force recreate containers
  --containers <list>     Start only specific containers (comma-separated)
  -h, --help              Show this help message

Examples:
  gorai start --config robot.json --detach
  gorai start -c robot.json --build -d
  gorai start --config robot.json --containers nats,gorai-core`)
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
