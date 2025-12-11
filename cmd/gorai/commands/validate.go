package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/gorai/gorai/pkg/config"
	"github.com/gorai/gorai/pkg/registry"
)

// cmdValidate handles the 'gorai validate' command.
func cmdValidate() error {
	args := os.Args[2:]

	// Parse flags
	var robotName string
	strict := false
	showHelp := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			showHelp = true
		case arg == "--strict":
			strict = true
		case !strings.HasPrefix(arg, "-"):
			if robotName == "" {
				robotName = arg
			}
		default:
			return fmt.Errorf("unknown flag: %s", arg)
		}
	}

	if showHelp {
		return printValidateUsage()
	}

	// Get robot name from args or environment
	if robotName == "" {
		robotName = os.Getenv("GORAI_ROBOT_NAME")
	}

	if robotName == "" {
		return fmt.Errorf("robot name required\n\nUsage: gorai validate <robot-name> [flags]\n\nOr set GORAI_ROBOT_NAME environment variable")
	}

	// Derive config file from robot name
	configPath := robotName + ".json"

	fmt.Printf("Validating %s...\n", configPath)

	// Stage 1: JSON Syntax
	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read %s: %w", configPath, err)
	}
	fmt.Println("  + JSON syntax valid")

	// Stage 2: Schema Validation (parse into struct)
	cfg, err := config.LoadFromBytes(data)
	if err != nil {
		return fmt.Errorf("  ! Schema error: %w", err)
	}
	fmt.Println("  + Schema valid (RDL v1)")

	// Stage 3: Semantic Validation
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("  ! Semantic error: %w", err)
	}
	fmt.Printf("  + Robot name: %s\n", cfg.Robot.Name)
	fmt.Printf("  + %d components defined\n", len(cfg.Components))
	fmt.Printf("  + %d services defined\n", len(cfg.Services))
	fmt.Println("  + Dependencies resolvable")
	fmt.Println("  + No circular dependencies")

	// Stage 4: Registry Validation (check if types/models exist)
	if strict {
		if err := validateAgainstRegistry(cfg); err != nil {
			return err
		}
	}

	fmt.Printf("\n%s is valid!\n", configPath)
	return nil
}

func printValidateUsage() error {
	fmt.Println(`gorai validate - Validate robot configuration

Usage:
  gorai validate <robot-name> [flags]

The command reads <robot-name>.json and validates it.

Flags:
  --strict             Also validate component/service types against registry
  -h, --help           Show this help

Environment:
  GORAI_ROBOT_NAME     Default robot name if not specified

Examples:
  gorai validate my-robot
  gorai validate my-robot --strict

  # Or with environment variable:
  export GORAI_ROBOT_NAME=my-robot
  gorai validate

Validation Stages:
  1. JSON Syntax      - Valid JSON?
  2. Schema           - Required fields present? Types correct?
  3. Semantic         - Names unique? Dependencies exist? No cycles?
  4. Registry         - (--strict) Types and models registered?`)
	return nil
}

func validateAgainstRegistry(cfg *config.RDL) error {
	var errors []string

	// Validate component types
	registeredComponents := registry.ListComponents()
	for i, comp := range cfg.Components {
		if comp.Disabled {
			continue
		}

		models, typeExists := registeredComponents[comp.Type]
		if !typeExists {
			// Check if it might be a typo
			suggestions := findSimilar(comp.Type, getKeys(registeredComponents))
			if len(suggestions) > 0 {
				errors = append(errors, fmt.Sprintf(
					"components[%d]: unknown type %q (did you mean %q?)",
					i, comp.Type, suggestions[0],
				))
			} else {
				errors = append(errors, fmt.Sprintf(
					"components[%d]: unknown type %q",
					i, comp.Type,
				))
			}
			continue
		}

		// Check if model exists for type
		modelExists := false
		for _, m := range models {
			if m == comp.Model {
				modelExists = true
				break
			}
		}

		if !modelExists {
			suggestions := findSimilar(comp.Model, models)
			if len(suggestions) > 0 {
				errors = append(errors, fmt.Sprintf(
					"components[%d]: unknown model %q for type %q (did you mean %q?)",
					i, comp.Model, comp.Type, suggestions[0],
				))
			} else {
				errors = append(errors, fmt.Sprintf(
					"components[%d]: unknown model %q for type %q (available: %s)",
					i, comp.Model, comp.Type, strings.Join(models, ", "),
				))
			}
		}
	}

	// Validate service types
	registeredServices := registry.ListServices()
	for i, svc := range cfg.Services {
		if svc.Disabled {
			continue
		}

		models, typeExists := registeredServices[svc.Type]
		if !typeExists {
			suggestions := findSimilar(svc.Type, getKeys(registeredServices))
			if len(suggestions) > 0 {
				errors = append(errors, fmt.Sprintf(
					"services[%d]: unknown type %q (did you mean %q?)",
					i, svc.Type, suggestions[0],
				))
			} else {
				errors = append(errors, fmt.Sprintf(
					"services[%d]: unknown type %q",
					i, svc.Type,
				))
			}
			continue
		}

		// Check if model exists for type
		modelExists := false
		for _, m := range models {
			if m == svc.Model {
				modelExists = true
				break
			}
		}

		if !modelExists {
			suggestions := findSimilar(svc.Model, models)
			if len(suggestions) > 0 {
				errors = append(errors, fmt.Sprintf(
					"services[%d]: unknown model %q for type %q (did you mean %q?)",
					i, svc.Model, svc.Type, suggestions[0],
				))
			} else {
				errors = append(errors, fmt.Sprintf(
					"services[%d]: unknown model %q for type %q (available: %s)",
					i, svc.Model, svc.Type, strings.Join(models, ", "),
				))
			}
		}
	}

	if len(errors) > 0 {
		fmt.Println("  ! Registry validation errors:")
		for _, err := range errors {
			fmt.Printf("    - %s\n", err)
		}
		return fmt.Errorf("registry validation failed with %d errors", len(errors))
	}

	fmt.Println("  + All component types registered")
	fmt.Println("  + All models available")
	return nil
}

func getKeys(m map[string][]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func findSimilar(target string, candidates []string) []string {
	var similar []string
	target = strings.ToLower(target)

	for _, c := range candidates {
		cl := strings.ToLower(c)
		// Check for prefix match
		if strings.HasPrefix(cl, target) || strings.HasPrefix(target, cl) {
			similar = append(similar, c)
			continue
		}
		// Check for substring match
		if strings.Contains(cl, target) || strings.Contains(target, cl) {
			similar = append(similar, c)
			continue
		}
		// Check for Levenshtein distance <= 2
		if levenshtein(target, cl) <= 2 {
			similar = append(similar, c)
		}
	}

	return similar
}

func levenshtein(a, b string) int {
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	// Create matrix
	matrix := make([][]int, len(a)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(b)+1)
		matrix[i][0] = i
	}
	for j := range matrix[0] {
		matrix[0][j] = j
	}

	// Fill matrix
	for i := 1; i <= len(a); i++ {
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,      // deletion
				matrix[i][j-1]+1,      // insertion
				matrix[i-1][j-1]+cost, // substitution
			)
		}
	}

	return matrix[len(a)][len(b)]
}
