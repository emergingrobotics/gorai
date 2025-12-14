package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/gorai/gorai/pkg/compose"
	"github.com/gorai/gorai/pkg/config"
)

func cmdBuild() error {
	// Parse flags
	var configPath string
	var noCache bool
	var pull bool
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
		case "--no-cache":
			noCache = true
		case "--pull":
			pull = true
		case "--container":
			if i+1 >= len(args) {
				return fmt.Errorf("--container requires a value")
			}
			i++
			containers = append(containers, args[i])
		case "-h", "--help":
			return printBuildUsage()
		default:
			if args[i][0] != '-' {
				if configPath == "" {
					configPath = args[i]
				} else {
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

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Check if containers are defined
	if cfg.Containers == nil || len(cfg.Containers) == 0 {
		return fmt.Errorf("no containers defined in %s", configPath)
	}

	// Check which containers have build configs
	buildableContainers := make([]string, 0)
	for name, container := range cfg.Containers {
		if container.Build != nil {
			buildableContainers = append(buildableContainers, name)
		}
	}

	if len(buildableContainers) == 0 {
		fmt.Println("No containers with build configuration found.")
		fmt.Println("Add a 'build' section to container definitions to enable building.")
		return nil
	}

	// Generate compose file
	composePath := compose.GetComposePath(configPath, cfg.Robot.Name)
	gen := compose.NewGenerator(cfg)
	// Set workspace directory for resolving relative paths in build contexts
	workspaceDir := compose.GetProjectDir(configPath)
	gen.SetWorkspaceDir(workspaceDir)
	if err := gen.WriteJSON(composePath); err != nil {
		return fmt.Errorf("failed to generate compose file: %w", err)
	}

	// Create runner
	projectDir := compose.GetProjectDir(configPath)
	runner := compose.NewRunner(composePath, cfg.Robot.Name, projectDir)

	// Build containers
	fmt.Printf("Building containers for robot %q...\n", cfg.Robot.Name)
	if len(containers) > 0 {
		fmt.Printf("Building: %v\n", containers)
	} else {
		fmt.Printf("Building: %v\n", buildableContainers)
	}

	ctx := context.Background()
	opts := compose.BuildOptions{
		NoCache:  noCache,
		Pull:     pull,
		Services: containers,
	}

	if err := runner.Build(ctx, opts); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Println("Build completed successfully.")
	return nil
}

func printBuildUsage() error {
	fmt.Println(`gorai build - Build container images

Usage:
  gorai build [--config robot.json] [flags] [container...]

Flags:
  -c, --config <file>     Path to robot configuration file
  --no-cache              Do not use cache when building
  --pull                  Always attempt to pull newer base images
  --container <name>      Build specific container
  -h, --help              Show this help message

Examples:
  gorai build --config robot.json
  gorai build -c robot.json --no-cache
  gorai build --config robot.json --container gorai-hailo`)
	return nil
}
