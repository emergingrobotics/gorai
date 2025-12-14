package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/systemd"
)

func cmdStart() error {
	// Parse flags
	var configPath string
	var build bool
	var enable bool
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
		case "--build":
			build = true
		case "--enable":
			enable = true
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

	// Generate systemd service files
	workspaceDir := filepath.Dir(configPath)
	if !filepath.IsAbs(workspaceDir) {
		workspaceDir, _ = filepath.Abs(workspaceDir)
	}

	gen := systemd.NewGenerator(cfg,
		systemd.WithWorkspaceDir(workspaceDir),
		systemd.WithUserMode(true),
	)

	serviceDir := gen.GetLocalDir()
	fmt.Printf("Generating systemd service files in: %s\n", serviceDir)

	if err := gen.WriteFiles(); err != nil {
		return fmt.Errorf("failed to generate systemd files: %w", err)
	}

	// Build containers if requested
	if build {
		fmt.Println("Building container images...")
		for name, container := range cfg.Containers {
			if container.Build == nil {
				continue
			}

			fmt.Printf("Building %s...\n", name)

			buildArgs := []string{"build"}

			tag := container.Image
			if tag == "" {
				tag = fmt.Sprintf("%s-%s:latest", cfg.Robot.Name, name)
			}
			buildArgs = append(buildArgs, "-t", tag)

			dockerfile := container.Build.Dockerfile
			if dockerfile == "" {
				dockerfile = "Containerfile"
			}
			buildArgs = append(buildArgs, "-f", dockerfile)

			buildContext := container.Build.Context
			if buildContext == "" {
				buildContext = "."
			}
			if !filepath.IsAbs(buildContext) {
				buildContext = filepath.Join(workspaceDir, buildContext)
			}
			buildArgs = append(buildArgs, buildContext)

			cmd := exec.CommandContext(context.Background(), "podman", buildArgs...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to build %s: %w", name, err)
			}
		}
	}

	// Install service files to systemd
	fmt.Println("Installing systemd service files...")
	if err := gen.InstallFiles(); err != nil {
		return fmt.Errorf("failed to install systemd files: %w", err)
	}

	// Create runner
	runner := systemd.NewRunner(cfg.Robot.Name, serviceDir, true)

	// Reload systemd
	fmt.Println("Reloading systemd daemon...")
	if err := runner.DaemonReload(context.Background()); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Setup context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nReceived interrupt, stopping containers...")
		stopCtx := context.Background()
		_ = runner.Stop(stopCtx, containers...)
		cancel()
	}()

	// Enable services for auto-start at boot if requested
	if enable {
		fmt.Println("Enabling services for auto-start at boot...")
		if err := runner.Enable(ctx, containers...); err != nil {
			fmt.Printf("Warning: failed to enable services: %v\n", err)
		}
	}

	// Start containers
	fmt.Printf("Starting containers for robot %q...\n", cfg.Robot.Name)

	if err := runner.Start(ctx, containers...); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}

	fmt.Printf("\nContainers started successfully.\n")
	fmt.Printf("Use 'gorai status --config %s' to check status.\n", configPath)
	fmt.Printf("Use 'gorai logs --config %s -f' to view logs.\n", configPath)
	fmt.Printf("Use 'gorai stop --config %s' to stop.\n", configPath)

	return nil
}

func printStartUsage() error {
	fmt.Println(`gorai start - Start robot containers using systemd/Quadlet

Usage:
  gorai start [--config robot.json] [flags]

Flags:
  -c, --config <file>     Path to robot configuration file
  --build                 Build images before starting
  --enable                Enable services for auto-start at boot
  --containers <list>     Start only specific containers (comma-separated)
  -h, --help              Show this help message

Examples:
  gorai start --config robot.json
  gorai start -c robot.json --build
  gorai start --config robot.json --enable
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
