package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/emergingrobotics/gorai/pkg/config"
)

func cmdBuild() error {
	var configPath string
	var outputPath string
	var targetPlatform string
	var buildTags string

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config":
			if i+1 >= len(args) {
				return fmt.Errorf("--config requires a value")
			}
			i++
			configPath = args[i]
		case "-o", "--output":
			if i+1 >= len(args) {
				return fmt.Errorf("--output requires a value")
			}
			i++
			outputPath = args[i]
		case "--target":
			if i+1 >= len(args) {
				return fmt.Errorf("--target requires a value")
			}
			i++
			targetPlatform = args[i]
		case "--tags":
			if i+1 >= len(args) {
				return fmt.Errorf("--tags requires a value")
			}
			i++
			buildTags = args[i]
		case "-h", "--help":
			return printBuildUsage()
		default:
			if args[i][0] != '-' {
				if configPath == "" {
					configPath = args[i]
				}
			} else {
				return fmt.Errorf("unknown flag: %s", args[i])
			}
		}
	}

	if configPath == "" {
		configPath = findConfigFile()
		if configPath == "" {
			return fmt.Errorf("no config file specified. Use --config or create robot.json")
		}
	}

	configPath, _ = filepath.Abs(configPath)

	// Check that we're in a Go module
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		return fmt.Errorf("not in a Go module. Run 'gorai build' from your robot project directory. See: https://gorai.dev/docs/getting-started")
	}

	// Load and validate config
	cfg, err := config.LoadWithServiceRDL(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	// Default output name from robot name
	if outputPath == "" {
		outputPath = cfg.Robot.Name
	}

	// Prevent writing to system directories
	absOutput, err := filepath.Abs(outputPath)
	if err != nil {
		return fmt.Errorf("invalid output path: %w", err)
	}
	for _, prefix := range []string{"/etc", "/usr", "/sys", "/proc", "/dev"} {
		if strings.HasPrefix(absOutput, prefix+"/") || absOutput == prefix {
			return fmt.Errorf("refusing to write binary to system directory: %s", absOutput)
		}
	}

	// Parse target platform
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	if targetPlatform != "" {
		parts := strings.SplitN(targetPlatform, "/", 2)
		if len(parts) != 2 {
			return fmt.Errorf("--target must be os/arch (e.g., linux/arm64), got: %s", targetPlatform)
		}
		goos = parts[0]
		goarch = parts[1]
	}

	fmt.Printf("Building robot %q...\n", cfg.Robot.Name)
	fmt.Printf("  config:   %s\n", configPath)
	fmt.Printf("  output:   %s\n", outputPath)
	fmt.Printf("  platform: %s/%s\n", goos, goarch)
	if buildTags != "" {
		fmt.Printf("  tags:     %s\n", buildTags)
	}
	fmt.Println()

	// Build with go build
	buildArgs := []string{"build", "-o", outputPath}

	// Set ldflags for version info
	ldflags := fmt.Sprintf("-X github.com/emergingrobotics/gorai/cmd/gorai/commands.Version=%s", Version)
	buildArgs = append(buildArgs, "-ldflags", ldflags)

	// Build constraints (e.g. v4l2 for the real camera source on the Pi)
	if buildTags != "" {
		buildArgs = append(buildArgs, "-tags", buildTags)
	}

	// Build the current module (the user's robot project)
	buildArgs = append(buildArgs, ".")

	cmd := exec.Command("go", buildArgs...)
	cmd.Env = append(os.Environ(),
		"GOOS="+goos,
		"GOARCH="+goarch,
	)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go build failed: %w", err)
	}

	fmt.Printf("\nBuilt: %s (%s/%s)\n", outputPath, goos, goarch)
	fmt.Println("\nDeploy:")
	if goos != runtime.GOOS || goarch != runtime.GOARCH {
		fmt.Printf("  scp %s pi@raspberrypi:~\n", outputPath)
		fmt.Printf("  ssh pi@raspberrypi ./%s\n", outputPath)
	} else {
		fmt.Printf("  ./%s\n", outputPath)
	}

	return nil
}

func printBuildUsage() error {
	fmt.Println(`gorai build - Build standalone Go binary for deployment

Usage:
  gorai build [--config robot.json] [flags]

Builds the current robot project into a single Go binary with embedded NATS.
The binary contains exactly the components declared in main.go via blank imports.

Flags:
  -c, --config <file>     Path to robot configuration file (validates before build)
  -o, --output <path>     Output binary path (default: robot name from config)
  --target <os/arch>      Cross-compile target (e.g., linux/arm64)
  --tags <tags>           Go build tags, comma-separated (e.g., v4l2)
  -h, --help              Show this help message

Examples:
  gorai build robot.json                         # Build for current platform
  gorai build robot.json -o my-robot             # Custom output name
  gorai build robot.json --target linux/arm64    # Cross-compile for Raspberry Pi
  gorai build robot.json --tags v4l2             # Include the real V4L2 camera source

Deploy to a Raspberry Pi:
  gorai build robot.json -o robot --target linux/arm64
  scp robot pi@raspberrypi:~
  ssh pi@raspberrypi ./robot`)
	return nil
}
