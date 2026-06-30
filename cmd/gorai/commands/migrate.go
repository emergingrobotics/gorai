package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/emergingrobotics/gorai/pkg/config"
)

func cmdMigrate() error {
	// Parse flags
	var inputPath string
	var outputPath string
	var dryRun bool

	args := os.Args[2:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-c", "--config", "--input":
			if i+1 >= len(args) {
				return fmt.Errorf("--config requires a value")
			}
			i++
			inputPath = args[i]
		case "-o", "--output":
			if i+1 >= len(args) {
				return fmt.Errorf("--output requires a value")
			}
			i++
			outputPath = args[i]
		case "--dry-run":
			dryRun = true
		case "-h", "--help":
			return printMigrateUsage()
		default:
			if args[i][0] != '-' {
				// Treat as input path if not a flag
				inputPath = args[i]
			} else {
				return fmt.Errorf("unknown flag: %s", args[i])
			}
		}
	}

	if inputPath == "" {
		return fmt.Errorf("input config file required. Use --config <file>")
	}

	// Default output to input with .v2 suffix
	if outputPath == "" {
		ext := filepath.Ext(inputPath)
		base := inputPath[:len(inputPath)-len(ext)]
		outputPath = base + ".v2" + ext
	}

	// Load the old configuration
	cfg, err := config.Load(inputPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Check if migration is needed
	if !cfg.HasDeprecatedFields() && cfg.Version == "2" {
		fmt.Printf("Config %s is already in v2 format, no migration needed.\n", inputPath)
		return nil
	}

	// Perform migration
	migratedCfg, changes := migrateConfig(cfg)

	// Show changes
	fmt.Println("Migration changes:")
	for _, change := range changes {
		fmt.Printf("  - %s\n", change)
	}

	if dryRun {
		fmt.Println("\nDry run - no files written.")
		fmt.Println("\nMigrated configuration:")
		output, _ := json.MarshalIndent(migratedCfg, "", "  ")
		fmt.Println(string(output))
		return nil
	}

	// Write output
	output, err := json.MarshalIndent(migratedCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(outputPath, output, 0644); err != nil {
		return fmt.Errorf("failed to write output: %w", err)
	}

	fmt.Printf("\nMigrated config written to: %s\n", outputPath)
	return nil
}

// migrateConfig converts a v1 config to v2 format.
func migrateConfig(cfg *config.RDL) (*config.RDL, []string) {
	var changes []string

	// Create new config with v2 version
	newCfg := &config.RDL{
		Schema:     "https://gorai.dev/schemas/rdl-v2.json",
		Version:    "2",
		Robot:      cfg.Robot,
		Log:        cfg.Log,
		Dashboard:  cfg.Dashboard,
		Remotes:    cfg.Remotes,
	}

	// Update NATS config
	if cfg.NATS != nil {
		newCfg.NATS = &config.NATSConfig{
			URL:             cfg.NATS.URL,
			URLs:            cfg.NATS.URLs,
			JetStream:       cfg.NATS.JetStream,
			CredentialsFile: cfg.NATS.CredentialsFile,
			TLS:             cfg.NATS.TLS,
			ConnectTimeout:  cfg.NATS.ConnectTimeout,
			ReconnectWait:   cfg.NATS.ReconnectWait,
			MaxReconnects:   cfg.NATS.MaxReconnects,
		}

		// Update NATS URL from container address to localhost
		if cfg.NATS.URL == "nats://nats:4222" {
			newCfg.NATS.URL = "nats://localhost:4222"
			changes = append(changes, "Updated nats.url from nats://nats:4222 to nats://localhost:4222")
		}

		if cfg.NATS.Container != "" {
			changes = append(changes, fmt.Sprintf("Removed nats.container: %s", cfg.NATS.Container))
		}
	}

	// Copy components without container field
	for _, comp := range cfg.Components {
		newComp := config.ComponentConfig{
			Name:       comp.Name,
			Type:       comp.Type,
			Model:      comp.Model,
			Disabled:   comp.Disabled,
			Attributes: comp.Attributes,
			DependsOn:  comp.DependsOn,
		}
		if comp.Container != "" {
			changes = append(changes, fmt.Sprintf("Removed container field from component %s", comp.Name))
		}
		newCfg.Components = append(newCfg.Components, newComp)
	}

	// Copy services, converting container-based services to external if appropriate
	for _, svc := range cfg.Services {
		newSvc := config.ServiceConfig{
			Name:       svc.Name,
			Type:       svc.Type,
			Model:      svc.Model,
			Disabled:   svc.Disabled,
			Attributes: svc.Attributes,
			DependsOn:  svc.DependsOn,
			External:   svc.External, // Preserve if already set
		}

		// If service was in a separate container (not gorai-core), suggest making it external
		if svc.Container != "" && svc.Container != "gorai-core" {
			if newSvc.External == nil {
				newSvc.External = &config.ExternalConfig{
					Enabled: true,
					Managed: true,
					Restart: "always",
				}
				changes = append(changes, fmt.Sprintf("Converted service %s to external (was in container %s)", svc.Name, svc.Container))
			}
		} else if svc.Container != "" {
			changes = append(changes, fmt.Sprintf("Removed container field from service %s", svc.Name))
		}

		newCfg.Services = append(newCfg.Services, newSvc)
	}

	// Report removed containers
	if cfg.Containers != nil {
		for name := range cfg.Containers {
			changes = append(changes, fmt.Sprintf("Removed container definition: %s", name))
		}
	}

	changes = append(changes, "Updated version from 1 to 2")

	return newCfg, changes
}

func printMigrateUsage() error {
	fmt.Println(`gorai migrate - Migrate RDL v1 config to v2 format

Usage:
  gorai migrate --config <input.json> [--output <output.json>] [flags]

This command converts an RDL v1 configuration (with containers section) to
the v2 format (monolithic with optional external services).

Flags:
  -c, --config <file>     Input configuration file (v1 format)
  -o, --output <file>     Output file (default: <input>.v2.json)
  --dry-run               Show changes without writing file
  -h, --help              Show this help message

Migration changes:
  - Removes 'containers' section entirely
  - Updates nats.url from container hostname to localhost
  - Removes 'container' field from components (all run in main process)
  - Converts services in separate containers to 'external' services
  - Updates version from "1" to "2"

Examples:
  gorai migrate --config robot.json
  gorai migrate --config robot.json --output robot-v2.json
  gorai migrate --config robot.json --dry-run

After migration, update your deployment:
  1. Install NATS natively: sudo apt install nats-server
  2. Build external services separately (e.g., ML inference)
  3. Run with: gorai run --config robot-v2.json`)
	return nil
}
