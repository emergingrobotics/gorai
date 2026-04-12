package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

func cmdBuild() error {
	var configPath string
	var outputPath string
	var targetPlatform string

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
	fmt.Printf("  platform: %s/%s\n\n", goos, goarch)

	// Build with go build
	buildArgs := []string{"build", "-o", outputPath}

	// Set ldflags for version info
	ldflags := fmt.Sprintf("-X github.com/gorai/gorai/cmd/gorai/commands.Version=%s", Version)
	buildArgs = append(buildArgs, "-ldflags", ldflags)

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
  -h, --help              Show this help message

Examples:
  gorai build robot.json                         # Build for current platform
  gorai build robot.json -o my-robot             # Custom output name
  gorai build robot.json --target linux/arm64    # Cross-compile for Raspberry Pi

Deploy to a Raspberry Pi:
  gorai build robot.json -o robot --target linux/arm64
  scp robot pi@raspberrypi:~
  ssh pi@raspberrypi ./robot`)
	return nil
}
