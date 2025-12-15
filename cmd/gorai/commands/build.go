package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

// BuildableService represents a service that can be built as a container.
type BuildableService struct {
	Name          string
	Image         string
	Context       string
	Containerfile string
	Args          map[string]string
	Target        string
	NoCache       bool
}

func cmdBuild() error {
	// Parse flags
	var configPath string
	var noCache bool
	var pull bool
	var servicesOnly bool
	var serviceNames []string

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
		case "--services":
			servicesOnly = true
		case "--service":
			if i+1 >= len(args) {
				return fmt.Errorf("--service requires a value")
			}
			i++
			serviceNames = append(serviceNames, args[i])
		case "-h", "--help":
			return printBuildUsage()
		default:
			if args[i][0] != '-' {
				if configPath == "" {
					configPath = args[i]
				} else {
					serviceNames = append(serviceNames, args[i])
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
	configPath, _ = filepath.Abs(configPath)
	configDir := filepath.Dir(configPath)

	// Load configuration with Service RDL support
	cfg, err := config.LoadWithServiceRDL(configPath)
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

	fmt.Printf("Building robot %q...\n", cfg.Robot.Name)
	fmt.Printf("Config: %s\n\n", configPath)

	// Find buildable container services
	buildables := getBuildableServices(cfg, configDir, serviceNames)

	if len(buildables) == 0 {
		fmt.Println("No buildable container services found.")
		if !servicesOnly {
			fmt.Println("\nTip: To add a buildable service, add 'build' config to external.container:")
			fmt.Println(`  "external": {`)
			fmt.Println(`    "container": {`)
			fmt.Println(`      "image": "localhost/my-service:latest",`)
			fmt.Println(`      "build": {`)
			fmt.Println(`        "context": "./services/my-service"`)
			fmt.Println(`      }`)
			fmt.Println(`    }`)
			fmt.Println(`  }`)
		}
		return nil
	}

	// Build each container
	fmt.Printf("Found %d buildable service(s):\n", len(buildables))
	for _, svc := range buildables {
		fmt.Printf("  - %s → %s\n", svc.Name, svc.Image)
	}
	fmt.Println()

	for i, svc := range buildables {
		fmt.Printf("[%d/%d] Building %s...\n", i+1, len(buildables), svc.Name)

		if err := buildContainer(svc, noCache, pull); err != nil {
			return fmt.Errorf("failed to build %s: %w", svc.Name, err)
		}

		fmt.Printf("  ✓ Built %s\n\n", svc.Image)
	}

	fmt.Println("Build completed successfully.")
	fmt.Println("\nNext steps:")
	fmt.Printf("  gorai run --config %s     # Run robot (foreground)\n", configPath)
	fmt.Printf("  gorai start --config %s   # Start as systemd service\n", configPath)

	return nil
}

// getBuildableServices finds all services that can be built as containers.
func getBuildableServices(cfg *config.RDL, configDir string, filterNames []string) []BuildableService {
	var result []BuildableService

	for _, svc := range cfg.Services {
		// Check if we should filter by name
		if len(filterNames) > 0 {
			found := false
			for _, name := range filterNames {
				if name == svc.Name {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		// Skip if not external or not a container
		if !svc.IsExternal() || svc.External.Container == nil {
			continue
		}

		container := svc.External.Container

		// Skip non-local images (they're pulled, not built)
		if !strings.HasPrefix(container.Image, "localhost/") {
			continue
		}

		bs := BuildableService{
			Name:  svc.Name,
			Image: container.Image,
		}

		// Use explicit build config if provided
		if container.Build != nil && container.Build.Context != "" {
			bs.Context = container.GetBuildContext(configDir)
			bs.Containerfile = container.GetContainerfile()
			bs.Args = container.Build.Args
			bs.Target = container.Build.Target
			bs.NoCache = container.Build.NoCache
		} else {
			// Check if Service RDL has build context
			if svc.HasServiceRDL() && svc.RDL != "" {
				// The build context might be relative to the Service RDL file
				rdlDir := filepath.Dir(filepath.Join(configDir, svc.RDL))
				bs.Context, bs.Containerfile = discoverBuildContext(rdlDir, svc.Name)
				if bs.Context == "" {
					// Also try with service name from image
					imageName := strings.TrimPrefix(container.Image, "localhost/")
					imageName = strings.Split(imageName, ":")[0]
					bs.Context, bs.Containerfile = discoverBuildContext(rdlDir, imageName)
				}
			}

			// Convention-based discovery relative to robot config
			if bs.Context == "" {
				bs.Context, bs.Containerfile = discoverBuildContext(configDir, svc.Name)
			}
		}

		// Skip if no build context found
		if bs.Context == "" {
			fmt.Printf("  Note: Skipping %s - no build context found\n", svc.Name)
			if svc.HasServiceRDL() {
				fmt.Printf("        Add 'build.context' to Service RDL or robot config\n")
			} else {
				fmt.Printf("        Add 'build.context' to config or create services/%s/Containerfile\n", svc.Name)
			}
			continue
		}

		// Verify context exists
		if _, err := os.Stat(bs.Context); os.IsNotExist(err) {
			fmt.Printf("  Note: Skipping %s - build context not found: %s\n", svc.Name, bs.Context)
			continue
		}

		// Verify Containerfile exists
		containerfilePath := filepath.Join(bs.Context, bs.Containerfile)
		if _, err := os.Stat(containerfilePath); os.IsNotExist(err) {
			// Try Dockerfile as fallback
			dockerfilePath := filepath.Join(bs.Context, "Dockerfile")
			if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
				fmt.Printf("  Note: Skipping %s - no Containerfile found in %s\n", svc.Name, bs.Context)
				continue
			}
			bs.Containerfile = "Dockerfile"
		}

		result = append(result, bs)
	}

	return result
}

// discoverBuildContext tries to find a build context using conventions.
func discoverBuildContext(configDir, serviceName string) (context, containerfile string) {
	// Try common locations
	candidates := []string{
		filepath.Join(configDir, "services", serviceName),
		filepath.Join(configDir, serviceName),
		filepath.Join(configDir, "..", "services", serviceName),
		filepath.Join(configDir, "..", "..", "services", serviceName),
	}

	for _, candidate := range candidates {
		// Check for Containerfile
		if _, err := os.Stat(filepath.Join(candidate, "Containerfile")); err == nil {
			return candidate, "Containerfile"
		}
		// Check for Dockerfile
		if _, err := os.Stat(filepath.Join(candidate, "Dockerfile")); err == nil {
			return candidate, "Dockerfile"
		}
	}

	return "", ""
}

// buildContainer builds a container image using podman.
func buildContainer(svc BuildableService, noCache, pull bool) error {
	buildArgs := []string{"build"}

	// Cache options
	if noCache || svc.NoCache {
		buildArgs = append(buildArgs, "--no-cache")
	}
	if pull {
		buildArgs = append(buildArgs, "--pull=always")
	}

	// Tag
	buildArgs = append(buildArgs, "-t", svc.Image)

	// Containerfile
	buildArgs = append(buildArgs, "-f", filepath.Join(svc.Context, svc.Containerfile))

	// Build args
	for key, value := range svc.Args {
		buildArgs = append(buildArgs, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Target (multi-stage builds)
	if svc.Target != "" {
		buildArgs = append(buildArgs, "--target", svc.Target)
	}

	// Context
	buildArgs = append(buildArgs, svc.Context)

	// Run podman build
	cmd := exec.Command("podman", buildArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

func printBuildUsage() error {
	fmt.Println(`gorai build - Build container images for external services

Usage:
  gorai build [--config robot.json] [flags] [service...]

This command builds container images for services that have 'external.container'
configuration with a 'build' section, or that can be auto-discovered using
convention-based paths.

Flags:
  -c, --config <file>     Path to robot configuration file
  --no-cache              Do not use cache when building
  --pull                  Always attempt to pull newer base images
  --services              Build container services only (default behavior)
  --service <name>        Build specific service by name
  -h, --help              Show this help message

Convention-based discovery:
  If a service has image "localhost/<name>:latest" without explicit build config,
  gorai will look for Containerfile in these locations:
    - ./services/<service-name>/Containerfile
    - ./<service-name>/Containerfile

Examples:
  gorai build --config robot.json              # Build all container services
  gorai build -c robot.json --no-cache         # Build without cache
  gorai build --config robot.json person_detector  # Build specific service

RDL configuration example:
  {
    "services": [{
      "name": "person_detector",
      "external": {
        "enabled": true,
        "container": {
          "image": "localhost/hailo-detector:latest",
          "build": {
            "context": "./services/hailo-detector",
            "args": {"VERSION": "1.0"}
          }
        }
      }
    }]
  }`)
	return nil
}
