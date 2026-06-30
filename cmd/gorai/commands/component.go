package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/emergingrobotics/gorai/pkg/componentregistry"
)

const defaultRegistryPath = "registry.json"

func cmdComponentDispatch() error {
	if len(os.Args) < 3 {
		return printComponentUsage()
	}
	switch os.Args[2] {
	case "search":
		return cmdComponentSearch()
	case "add":
		return cmdComponentAdd()
	case "info":
		return cmdComponentInfo()
	case "help", "-h", "--help":
		return printComponentUsage()
	default:
		return fmt.Errorf("unknown component subcommand: %s", os.Args[2])
	}
}

func cmdComponentSearch() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: gorai component search <query>")
	}
	query := strings.Join(os.Args[3:], " ")

	reg, err := loadRegistry()
	if err != nil {
		return err
	}

	results := reg.Search(query)
	if len(results) == 0 {
		fmt.Println("No components found.")
		return nil
	}

	fmt.Printf("Found %d component(s):\n\n", len(results))
	for _, c := range results {
		fmt.Printf("  %-35s %s\n", c.Type+"/"+c.Model, c.Description)
		fmt.Printf("  %-35s %s\n", "", c.Module+" "+c.Version)
		fmt.Println()
	}
	return nil
}

func cmdComponentAdd() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: gorai component add <name|module-path>")
	}
	arg := os.Args[3]

	// If it looks like a Go module path (contains a dot), use directly
	modulePath := arg
	if !strings.Contains(arg, ".") {
		reg, err := loadRegistry()
		if err != nil {
			return err
		}
		c, ok := reg.Lookup(arg)
		if !ok {
			return fmt.Errorf("component %q not found in registry.\n"+
				"  Try: gorai component search %s\n"+
				"  Or use a direct module path: gorai component add github.com/org/module", arg, arg)
		}
		modulePath = c.Module
		fmt.Printf("Found: %s (%s)\n", arg, c.Description)
	}

	// Split @version if present
	version := "@latest"
	if idx := strings.Index(modulePath, "@"); idx >= 0 {
		version = modulePath[idx:]
		modulePath = modulePath[:idx]
	}

	if err := validateModulePath(modulePath); err != nil {
		return fmt.Errorf("invalid module path: %w", err)
	}

	// Run go get
	getArg := modulePath + version
	fmt.Printf("Running: go get %s\n", getArg)
	goGet := exec.Command("go", "get", getArg)
	goGet.Stdout = os.Stdout
	goGet.Stderr = os.Stderr
	if err := goGet.Run(); err != nil {
		return fmt.Errorf("go get failed: %w", err)
	}

	// Add blank import to main.go
	mainPath := findMainGo()
	if mainPath == "" {
		fmt.Printf("Added %s to go.mod.\n", modulePath)
		fmt.Printf("Add this to your main.go:\n  _ %q\n", modulePath)
		return nil
	}

	if err := componentregistry.AddBlankImport(mainPath, modulePath); err != nil {
		fmt.Printf("Added %s to go.mod.\n", modulePath)
		fmt.Printf("Could not auto-edit main.go: %v\n", err)
		fmt.Printf("Add this manually:\n  _ %q\n", modulePath)
		return nil
	}

	// Run go mod tidy
	goTidy := exec.Command("go", "mod", "tidy")
	goTidy.Stdout = os.Stdout
	goTidy.Stderr = os.Stderr
	if err := goTidy.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "warning: go mod tidy failed: %v\n", err)
	}

	fmt.Printf("Added %s to go.mod and main.go.\n", modulePath)
	return nil
}

func cmdComponentInfo() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: gorai component info <name>")
	}
	name := os.Args[3]

	reg, err := loadRegistry()
	if err != nil {
		return err
	}

	c, ok := reg.Lookup(name)
	if !ok {
		return fmt.Errorf("component %q not found in registry", name)
	}

	fmt.Printf("\n  %s/%s\n", c.Type, c.Model)
	fmt.Printf("  %s\n\n", c.Description)
	fmt.Printf("  Module:   %s\n", c.Module)
	fmt.Printf("  Version:  %s\n", c.Version)
	if c.Hardware != "" {
		fmt.Printf("  Hardware: %s\n", c.Hardware)
	}
	if len(c.Tags) > 0 {
		fmt.Printf("  Tags:     %s\n", strings.Join(c.Tags, ", "))
	}
	fmt.Printf("\n  Install:\n    gorai component add %s\n\n", name)
	return nil
}

func loadRegistry() (*componentregistry.Registry, error) {
	candidates := []string{
		defaultRegistryPath,
		filepath.Join(os.Getenv("HOME"), ".gorai", "registry.json"),
	}
	for _, path := range candidates {
		reg, err := componentregistry.LoadFromFile(path)
		if err == nil {
			return reg, nil
		}
	}
	return nil, fmt.Errorf("no registry file found. Create registry.json or place one at ~/.gorai/registry.json")
}

func findMainGo() string {
	if _, err := os.Stat("main.go"); err == nil {
		return "main.go"
	}
	return ""
}

// validateModulePath checks that a module path is safe to pass to go get.
func validateModulePath(path string) error {
	if path == "" {
		return fmt.Errorf("empty module path")
	}
	for _, ch := range path {
		if ch == ';' || ch == '&' || ch == '|' || ch == '$' || ch == '`' || ch == '\'' || ch == '"' || ch == '\\' || ch == '\n' || ch == '\r' {
			return fmt.Errorf("module path contains forbidden character %q", ch)
		}
	}
	if !strings.Contains(path, "/") {
		return fmt.Errorf("module path must contain at least one slash")
	}
	return nil
}

func printComponentUsage() error {
	fmt.Println(`gorai component - Manage GoRAI components

Usage:
  gorai component <subcommand> [args]

Subcommands:
  search <query>     Search for components in the registry
  add <name|module>  Add a component (go get + blank import)
  info <name>        Show component details

Examples:
  gorai component search servo
  gorai component add sensor/hc-sr04
  gorai component add github.com/acme/gorai-driver-custom@v1.0.0
  gorai component info picarx`)
	return nil
}
