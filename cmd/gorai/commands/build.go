package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/quadlet"
)

func cmdBuild() error {
	// Parse flags
	var configPath string
	var noCache bool
	var pull bool
	var install bool
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
		case "--install":
			install = true
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

	// Generate Quadlet files
	workspaceDir := filepath.Dir(configPath)
	if !filepath.IsAbs(workspaceDir) {
		workspaceDir, _ = filepath.Abs(workspaceDir)
	}

	gen := quadlet.NewGenerator(cfg,
		quadlet.WithWorkspaceDir(workspaceDir),
		quadlet.WithUserMode(true),
	)

	quadletDir := gen.GetLocalDir()
	fmt.Printf("Generating Quadlet files in: %s\n", quadletDir)

	if err := gen.WriteFiles(); err != nil {
		return fmt.Errorf("failed to generate Quadlet files: %w", err)
	}

	// Build container images if any have build configs
	if len(buildableContainers) > 0 {
		containersToBuild := containers
		if len(containersToBuild) == 0 {
			containersToBuild = buildableContainers
		}

		fmt.Printf("Building containers for robot %q...\n", cfg.Robot.Name)
		fmt.Printf("Building: %v\n", containersToBuild)

		for _, containerName := range containersToBuild {
			container, exists := cfg.Containers[containerName]
			if !exists {
				return fmt.Errorf("container %q not found in configuration", containerName)
			}
			if container.Build == nil {
				fmt.Printf("Skipping %s (no build configuration)\n", containerName)
				continue
			}

			fmt.Printf("Building %s...\n", containerName)

			// Build using podman build
			buildArgs := []string{"build"}

			if noCache {
				buildArgs = append(buildArgs, "--no-cache")
			}
			if pull {
				buildArgs = append(buildArgs, "--pull=always")
			}

			// Tag
			tag := container.Image
			if tag == "" {
				tag = fmt.Sprintf("%s-%s:latest", cfg.Robot.Name, containerName)
			}
			buildArgs = append(buildArgs, "-t", tag)

			// Dockerfile
			dockerfile := container.Build.Dockerfile
			if dockerfile == "" {
				dockerfile = "Containerfile"
			}
			buildArgs = append(buildArgs, "-f", dockerfile)

			// Build args
			for key, value := range container.Build.Args {
				buildArgs = append(buildArgs, "--build-arg", fmt.Sprintf("%s=%s", key, value))
			}

			// Target
			if container.Build.Target != "" {
				buildArgs = append(buildArgs, "--target", container.Build.Target)
			}

			// Context
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
				return fmt.Errorf("failed to build %s: %w", containerName, err)
			}
		}

		fmt.Println("Build completed successfully.")
	} else {
		fmt.Println("No containers with build configuration found.")
	}

	// Install Quadlet files to systemd if requested
	if install {
		fmt.Println("\nInstalling Quadlet files to systemd...")
		if err := gen.InstallFiles(); err != nil {
			return fmt.Errorf("failed to install Quadlet files: %w", err)
		}

		// Reload systemd
		runner := quadlet.NewRunner(cfg.Robot.Name, quadletDir, true)
		if err := runner.DaemonReload(context.Background()); err != nil {
			return fmt.Errorf("failed to reload systemd: %w", err)
		}

		fmt.Println("Quadlet files installed. Services are now available.")
		fmt.Printf("Use 'gorai start --config %s' to start the robot.\n", configPath)
	}

	return nil
}

func printBuildUsage() error {
	fmt.Println(`gorai build - Build container images and generate Quadlet files

Usage:
  gorai build [--config robot.json] [flags] [container...]

Flags:
  -c, --config <file>     Path to robot configuration file
  --no-cache              Do not use cache when building
  --pull                  Always attempt to pull newer base images
  --install               Install Quadlet files to systemd
  --container <name>      Build specific container
  -h, --help              Show this help message

Examples:
  gorai build --config robot.json
  gorai build -c robot.json --no-cache
  gorai build --config robot.json --install
  gorai build --config robot.json --container gorai-hailo`)
	return nil
}
