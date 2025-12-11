package commands

import (
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

// Component type to import path mapping
var componentImports = map[string]string{
	// Sensors
	"imu":                "github.com/gorai/gorai/component/sensor/imu",
	"ahrs":               "github.com/gorai/gorai/component/sensor/ahrs",
	"gps":                "github.com/gorai/gorai/component/sensor/gps",
	"encoder":            "github.com/gorai/gorai/component/sensor/encoder",
	"range_sensor":       "github.com/gorai/gorai/component/sensor/range",
	"lidar":              "github.com/gorai/gorai/component/sensor/lidar",
	"presence_sensor":    "github.com/gorai/gorai/component/sensor/presence",
	"thermal_array":      "github.com/gorai/gorai/component/sensor/thermal",
	"force_sensor":       "github.com/gorai/gorai/component/sensor/force",
	"force_6dof":         "github.com/gorai/gorai/component/sensor/force6dof",
	"current_sensor":     "github.com/gorai/gorai/component/sensor/current",
	"reflectance_sensor": "github.com/gorai/gorai/component/sensor/reflectance",
	"camera":             "github.com/gorai/gorai/component/camera",
	"temperature":        "github.com/gorai/gorai/component/sensor/temperature",

	// Actuators
	"motor":    "github.com/gorai/gorai/component/motor",
	"servo":    "github.com/gorai/gorai/component/servo",
	"stepper":  "github.com/gorai/gorai/component/stepper",
	"thruster": "github.com/gorai/gorai/component/thruster",
	"valve":    "github.com/gorai/gorai/component/valve",
	"gripper":  "github.com/gorai/gorai/component/gripper",
	"arm":      "github.com/gorai/gorai/component/arm",
	"base":     "github.com/gorai/gorai/component/base",

	// Infrastructure
	"power": "github.com/gorai/gorai/component/power",
	"space": "github.com/gorai/gorai/component/space",
	"link":  "github.com/gorai/gorai/component/link",
}

// Service type to import path mapping
var serviceImports = map[string]string{
	"vision":      "github.com/gorai/gorai/service/vision",
	"slam":        "github.com/gorai/gorai/service/slam",
	"navigation":  "github.com/gorai/gorai/service/navigation",
	"motion":      "github.com/gorai/gorai/service/motion",
	"behavior":    "github.com/gorai/gorai/service/behavior",
	"coordinator": "github.com/gorai/gorai/service/coordinator",
	"mlmodel":     "github.com/gorai/gorai/service/mlmodel",
}

// cmdGenerate handles the 'gorai generate' command.
func cmdGenerate() error {
	args := os.Args[2:]

	// Parse flags
	var robotName string
	dryRun := false
	verbose := false
	validateOnly := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return printGenerateUsage()
		case arg == "--dry-run":
			dryRun = true
		case arg == "--verbose":
			verbose = true
		case arg == "--validate-only":
			validateOnly = true
		case !strings.HasPrefix(arg, "-"):
			if robotName == "" {
				robotName = arg
			}
		default:
			return fmt.Errorf("unknown flag: %s", arg)
		}
	}

	// Get robot name from args or environment
	if robotName == "" {
		robotName = os.Getenv("GORAI_ROBOT_NAME")
	}

	if robotName == "" {
		return fmt.Errorf("robot name required\n\nUsage: gorai generate <robot-name> [flags]\n\nOr set GORAI_ROBOT_NAME environment variable")
	}

	// Derive config file from robot name
	configPath := robotName + ".json"

	if verbose {
		fmt.Printf("Reading %s...\n", configPath)
	}

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load %s: %w", configPath, err)
	}

	// Validate configuration
	fmt.Println("Validating configuration...")
	if err := cfg.Validate(); err != nil {
		return err
	}

	fmt.Printf("  + Robot name: %s\n", cfg.Robot.Name)
	fmt.Printf("  + %d components defined\n", len(cfg.Components))
	fmt.Printf("  + %d services defined\n", len(cfg.Services))
	fmt.Println("  + All validation checks passed")

	if validateOnly {
		fmt.Println("\nValidation complete!")
		return nil
	}

	fmt.Println("\nGenerating code...")

	// Collect imports
	imports, unknownTypes := collectImports(cfg)
	if len(unknownTypes) > 0 && verbose {
		fmt.Println("  ! Warning: Unknown types (may be custom components):")
		for _, t := range unknownTypes {
			fmt.Printf("    - %s\n", t)
		}
	}

	// Get project module name from go.mod
	moduleName, err := getModuleName()
	if err != nil {
		// Fall back to robot name
		moduleName = robotName
	}

	// Generate imports.go
	importsContent := generateImportsGo(cfg, imports, moduleName, robotName)
	importsPath := "internal/generated/imports.go"

	if dryRun {
		fmt.Printf("  Would generate %s\n", importsPath)
		if verbose {
			fmt.Println("--- imports.go ---")
			fmt.Println(importsContent)
			fmt.Println("--- end ---")
		}
	} else {
		// Ensure directory exists
		if err := os.MkdirAll("internal/generated", 0755); err != nil {
			return fmt.Errorf("failed to create internal/generated: %w", err)
		}

		if err := os.WriteFile(importsPath, []byte(importsContent), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", importsPath, err)
		}
		fmt.Printf("  + %s (updated)\n", importsPath)
	}

	// Generate validate.go (compile-time type assertions)
	validateContent := generateValidateGo(cfg, imports)
	validatePath := "internal/generated/validate.go"

	if dryRun {
		fmt.Printf("  Would generate %s\n", validatePath)
		if verbose {
			fmt.Println("--- validate.go ---")
			fmt.Println(validateContent)
			fmt.Println("--- end ---")
		}
	} else {
		if err := os.WriteFile(validatePath, []byte(validateContent), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", validatePath, err)
		}
		fmt.Printf("  + %s (updated)\n", validatePath)
	}

	// Generate/update systemd service if deploy directory exists
	if _, err := os.Stat("deploy"); err == nil {
		servicePath := fmt.Sprintf("deploy/%s.service", robotName)
		serviceContent := generateSystemdService(robotName)

		if dryRun {
			fmt.Printf("  Would generate %s\n", servicePath)
		} else {
			if err := os.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", servicePath, err)
			}
			fmt.Printf("  + %s (updated)\n", servicePath)
		}
	}

	fmt.Println("\nGeneration complete!")
	return nil
}

func printGenerateUsage() error {
	fmt.Println(`gorai generate - Generate code from robot configuration

Usage:
  gorai generate <robot-name> [flags]

The command reads <robot-name>.json and generates code in internal/generated/.

Flags:
  --dry-run            Show what would be generated without writing
  --verbose            Show detailed output
  --validate-only      Only validate, don't generate
  -h, --help           Show this help

Environment:
  GORAI_ROBOT_NAME     Default robot name if not specified

Example:
  gorai generate my-robot
  gorai generate my-robot --dry-run --verbose

  # Or with environment variable:
  export GORAI_ROBOT_NAME=my-robot
  gorai generate`)
	return nil
}

type importInfo struct {
	Path    string
	Comment string
}

func collectImports(cfg *config.RDL) ([]importInfo, []string) {
	var imports []importInfo
	var unknownTypes []string
	seen := make(map[string]bool)

	// Collect component imports
	for _, comp := range cfg.Components {
		if comp.Disabled {
			continue
		}

		if path, ok := componentImports[comp.Type]; ok {
			if !seen[path] {
				seen[path] = true
				imports = append(imports, importInfo{
					Path:    path,
					Comment: fmt.Sprintf("%s: %s", comp.Type, comp.Name),
				})
			}
		} else {
			unknownTypes = append(unknownTypes, fmt.Sprintf("component type %q (%s)", comp.Type, comp.Name))
		}
	}

	// Collect service imports
	for _, svc := range cfg.Services {
		if svc.Disabled {
			continue
		}

		if path, ok := serviceImports[svc.Type]; ok {
			if !seen[path] {
				seen[path] = true
				imports = append(imports, importInfo{
					Path:    path,
					Comment: fmt.Sprintf("%s: %s", svc.Type, svc.Name),
				})
			}
		} else {
			unknownTypes = append(unknownTypes, fmt.Sprintf("service type %q (%s)", svc.Type, svc.Name))
		}
	}

	// Sort imports for consistent output
	sort.Slice(imports, func(i, j int) bool {
		return imports[i].Path < imports[j].Path
	})

	return imports, unknownTypes
}

func generateImportsGo(cfg *config.RDL, imports []importInfo, moduleName, robotName string) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("// Code generated by gorai generate from %s.json. DO NOT EDIT.\n\n", robotName))
	sb.WriteString("package generated\n\n")
	sb.WriteString("import (\n")

	// Group imports by category
	var sensorImports, actuatorImports, serviceImportsList []importInfo

	for _, imp := range imports {
		switch {
		case strings.Contains(imp.Path, "/sensor/") || strings.Contains(imp.Path, "/camera"):
			sensorImports = append(sensorImports, imp)
		case strings.Contains(imp.Path, "/service/"):
			serviceImportsList = append(serviceImportsList, imp)
		default:
			actuatorImports = append(actuatorImports, imp)
		}
	}

	if len(sensorImports) > 0 {
		sb.WriteString("\t// ============================================\n")
		sb.WriteString(fmt.Sprintf("\t// Sensors (from %s.json)\n", robotName))
		sb.WriteString("\t// ============================================\n\n")
		for _, imp := range sensorImports {
			sb.WriteString(fmt.Sprintf("\t_ %q // %s\n", imp.Path, imp.Comment))
		}
		sb.WriteString("\n")
	}

	if len(actuatorImports) > 0 {
		sb.WriteString("\t// ============================================\n")
		sb.WriteString(fmt.Sprintf("\t// Actuators (from %s.json)\n", robotName))
		sb.WriteString("\t// ============================================\n\n")
		for _, imp := range actuatorImports {
			sb.WriteString(fmt.Sprintf("\t_ %q // %s\n", imp.Path, imp.Comment))
		}
		sb.WriteString("\n")
	}

	if len(serviceImportsList) > 0 {
		sb.WriteString("\t// ============================================\n")
		sb.WriteString(fmt.Sprintf("\t// Services (from %s.json)\n", robotName))
		sb.WriteString("\t// ============================================\n\n")
		for _, imp := range serviceImportsList {
			sb.WriteString(fmt.Sprintf("\t_ %q // %s\n", imp.Path, imp.Comment))
		}
		sb.WriteString("\n")
	}

	// Custom components placeholder
	sb.WriteString("\t// ============================================\n")
	sb.WriteString("\t// Custom Components (from components/)\n")
	sb.WriteString("\t// ============================================\n\n")

	// Check for custom components
	customComps := findCustomComponents(moduleName)
	if len(customComps) > 0 {
		for _, comp := range customComps {
			sb.WriteString(fmt.Sprintf("\t_ %q\n", comp))
		}
	} else {
		sb.WriteString("\t// (none defined yet - add with: gorai add component <name>)\n")
	}
	sb.WriteString("\n")

	// Custom services placeholder
	sb.WriteString("\t// ============================================\n")
	sb.WriteString("\t// Custom Services (from services/)\n")
	sb.WriteString("\t// ============================================\n\n")

	customSvcs := findCustomServices(moduleName)
	if len(customSvcs) > 0 {
		for _, svc := range customSvcs {
			sb.WriteString(fmt.Sprintf("\t_ %q\n", svc))
		}
	} else {
		sb.WriteString("\t// (none defined yet - add with: gorai add service <name>)\n")
	}

	sb.WriteString(")\n\n")

	// Constants
	sb.WriteString("// RobotConfig is the path to the configuration file\n")
	sb.WriteString(fmt.Sprintf("const RobotConfig = %q\n\n", robotName+".json"))
	sb.WriteString("// ComponentCount is the number of components defined\n")
	sb.WriteString(fmt.Sprintf("const ComponentCount = %d\n\n", len(cfg.Components)))
	sb.WriteString("// ServiceCount is the number of services defined\n")
	sb.WriteString(fmt.Sprintf("const ServiceCount = %d\n", len(cfg.Services)))

	// Format the source
	result := sb.String()
	formatted, err := format.Source([]byte(result))
	if err != nil {
		// Return unformatted if formatting fails
		return result
	}
	return string(formatted)
}

func generateValidateGo(cfg *config.RDL, imports []importInfo) string {
	var sb strings.Builder

	sb.WriteString("// Code generated by gorai generate. DO NOT EDIT.\n\n")
	sb.WriteString("package generated\n\n")

	// Only generate if we have imports to validate
	if len(imports) == 0 {
		sb.WriteString("// No components or services to validate\n")
		return sb.String()
	}

	sb.WriteString("// Build will fail if any referenced types don't exist.\n")
	sb.WriteString("// This provides compile-time validation of configuration references.\n\n")

	sb.WriteString("import (\n")
	for _, imp := range imports {
		// Extract package name from path
		parts := strings.Split(imp.Path, "/")
		pkgName := parts[len(parts)-1]
		sb.WriteString(fmt.Sprintf("\t%s %q\n", pkgName, imp.Path))
	}
	sb.WriteString(")\n\n")

	sb.WriteString("// Type assertions to verify imports are valid\n")
	sb.WriteString("var (\n")
	for _, imp := range imports {
		parts := strings.Split(imp.Path, "/")
		pkgName := parts[len(parts)-1]
		// Use a simple assertion that the package exists
		sb.WriteString(fmt.Sprintf("\t_ = %s.Name // %s exists\n", pkgName, imp.Comment))
	}
	sb.WriteString(")\n")

	return sb.String()
}

func findCustomComponents(moduleName string) []string {
	var results []string

	entries, err := os.ReadDir("components")
	if err != nil {
		return results
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check if there's a .go file in the directory
			goFiles, err := filepath.Glob(filepath.Join("components", entry.Name(), "*.go"))
			if err == nil && len(goFiles) > 0 {
				results = append(results, fmt.Sprintf("%s/components/%s", moduleName, entry.Name()))
			}
		}
	}

	return results
}

func findCustomServices(moduleName string) []string {
	var results []string

	entries, err := os.ReadDir("services")
	if err != nil {
		return results
	}

	for _, entry := range entries {
		if entry.IsDir() {
			// Check if there's a .go file in the directory
			goFiles, err := filepath.Glob(filepath.Join("services", entry.Name(), "*.go"))
			if err == nil && len(goFiles) > 0 {
				results = append(results, fmt.Sprintf("%s/services/%s", moduleName, entry.Name()))
			}
		}
	}

	return results
}

func getModuleName() (string, error) {
	data, err := os.ReadFile("go.mod")
	if err != nil {
		return "", err
	}

	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module")), nil
		}
	}

	return "", fmt.Errorf("module name not found in go.mod")
}
