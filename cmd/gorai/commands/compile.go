package commands

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/gorai/gorai/pkg/compose"
	"github.com/gorai/gorai/pkg/config"
)

func cmdCompile() error {
	var configPath string
	var outputPath string
	var writeStdout bool

	outputPath = "process-compose.yaml"

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o", "--output":
			if i+1 >= len(args) {
				return fmt.Errorf("--output requires a value")
			}
			i++
			outputPath = args[i]
		case "--stdout":
			writeStdout = true
		case "-h", "--help":
			return printCompileUsage()
		default:
			if args[i][0] != '-' {
				configPath = args[i]
			} else {
				return fmt.Errorf("unknown flag: %s", args[i])
			}
		}
	}

	if configPath == "" {
		configPath = findConfigFile()
		if configPath == "" {
			return fmt.Errorf("no config file specified. Use 'gorai compile <config>' or create robot.json")
		}
	}

	if !filepath.IsAbs(configPath) {
		absPath, err := filepath.Abs(configPath)
		if err == nil {
			configPath = absPath
		}
	}

	cfg, err := config.LoadWithServiceRDL(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	compiler := compose.New(cfg, configPath)
	yamlBytes, err := compiler.CompileToYAML()
	if err != nil {
		return fmt.Errorf("failed to compile: %w", err)
	}

	if writeStdout {
		_, err := os.Stdout.Write(yamlBytes)
		return err
	}

	if err := os.WriteFile(outputPath, yamlBytes, 0644); err != nil {
		return fmt.Errorf("failed to write %s: %w", outputPath, err)
	}

	fmt.Fprintf(os.Stderr, "Compiled %s -> %s\n", configPath, outputPath)
	return nil
}

func printCompileUsage() error {
	fmt.Println(`gorai compile - Compile RDL to process-compose.yaml

Usage:
  gorai compile <config> [flags]

Compiles a robot configuration (RDL) into a process-compose.yaml file that
orchestrates all processes (gorai controller, NATS, external services, etc.).

Flags:
  -o, --output <file>   Output file path (default: process-compose.yaml)
  --stdout              Write to stdout instead of file
  -h, --help            Show this help message

Examples:
  gorai compile robot.json
  gorai compile robot.json -o my-compose.yaml
  gorai compile robot.json --stdout | process-compose up -f -`)
	return nil
}
