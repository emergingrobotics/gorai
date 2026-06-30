package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

// cmdAdd handles the 'gorai add' command.
func cmdAdd() error {
	args := os.Args[2:]

	if len(args) == 0 {
		return printAddUsage()
	}

	subCmd := args[0]
	switch subCmd {
	case "component":
		return addComponent(args[1:])
	case "service":
		return addService(args[1:])
	case "-h", "--help":
		return printAddUsage()
	default:
		return fmt.Errorf("unknown subcommand: %s\n\nUse 'gorai add -h' for help.", subCmd)
	}
}

func printAddUsage() error {
	fmt.Println(`gorai add - Add custom components or services

Usage:
  gorai add <subcommand> <name> [flags]

Subcommands:
  component <name>    Create a custom component
  service <name>      Create a custom service

Flags:
  --type <type>       Component/service type (e.g., sensor, motor, vision)
  -h, --help          Show this help

Examples:
  gorai add component my_sensor --type sensor
  gorai add component custom_motor --type motor
  gorai add service my_detector --type vision`)
	return nil
}

func addComponent(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("component name required\n\nUsage: gorai add component <name> [--type <type>]")
	}

	// Parse arguments
	name := ""
	compType := "sensor" // default

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return printAddComponentUsage()
		case arg == "--type" && i+1 < len(args):
			compType = args[i+1]
			i++
		case strings.HasPrefix(arg, "--type="):
			compType = strings.TrimPrefix(arg, "--type=")
		case !strings.HasPrefix(arg, "-"):
			if name == "" {
				name = arg
			}
		}
	}

	if name == "" {
		return fmt.Errorf("component name required")
	}

	// Validate name (must be valid Go package name)
	if err := validatePackageName(name); err != nil {
		return fmt.Errorf("invalid component name: %w", err)
	}

	// Create component directory
	componentDir := filepath.Join("components", name)
	if _, err := os.Stat(componentDir); err == nil {
		return fmt.Errorf("component %q already exists at %s", name, componentDir)
	}

	fmt.Printf("Creating custom component: %s\n", name)

	if err := os.MkdirAll(componentDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate component files
	structName := toCamelCase(name)

	// Main component file
	componentGo := generateComponentGo(name, compType, structName)
	if err := os.WriteFile(filepath.Join(componentDir, "component.go"), []byte(componentGo), 0644); err != nil {
		return fmt.Errorf("failed to create component.go: %w", err)
	}
	fmt.Printf("  + Created components/%s/component.go\n", name)

	// Test file
	testGo := generateComponentTestGo(name, structName)
	if err := os.WriteFile(filepath.Join(componentDir, "component_test.go"), []byte(testGo), 0644); err != nil {
		return fmt.Errorf("failed to create component_test.go: %w", err)
	}
	fmt.Printf("  + Created components/%s/component_test.go\n", name)

	// Fake directory and file
	fakeDir := filepath.Join(componentDir, "fake")
	if err := os.MkdirAll(fakeDir, 0755); err != nil {
		return fmt.Errorf("failed to create fake directory: %w", err)
	}

	fakeGo := generateFakeComponentGo(name, compType, structName)
	if err := os.WriteFile(filepath.Join(fakeDir, "fake.go"), []byte(fakeGo), 0644); err != nil {
		return fmt.Errorf("failed to create fake.go: %w", err)
	}
	fmt.Printf("  + Created components/%s/fake/fake.go\n", name)

	fmt.Printf(`
Next steps:
  1. Edit components/%s/component.go to implement your component
  2. Add to robot.json:
     { "name": "%s_instance", "type": "%s", "model": "%s" }
  3. Run: gorai generate
`, name, name, compType, name)

	return nil
}

func printAddComponentUsage() error {
	fmt.Println(`gorai add component - Create a custom component

Usage:
  gorai add component <name> [flags]

Flags:
  --type <type>       Component type: sensor, motor, servo, etc. (default: sensor)
  -h, --help          Show this help

Examples:
  gorai add component my_sensor --type sensor
  gorai add component custom_motor --type motor
  gorai add component my_gripper --type gripper`)
	return nil
}

func addService(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("service name required\n\nUsage: gorai add service <name> [--type <type>]")
	}

	// Parse arguments
	name := ""
	svcType := "behavior" // default

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-h" || arg == "--help":
			return printAddServiceUsage()
		case arg == "--type" && i+1 < len(args):
			svcType = args[i+1]
			i++
		case strings.HasPrefix(arg, "--type="):
			svcType = strings.TrimPrefix(arg, "--type=")
		case !strings.HasPrefix(arg, "-"):
			if name == "" {
				name = arg
			}
		}
	}

	if name == "" {
		return fmt.Errorf("service name required")
	}

	// Validate name
	if err := validatePackageName(name); err != nil {
		return fmt.Errorf("invalid service name: %w", err)
	}

	// Create service directory
	serviceDir := filepath.Join("services", name)
	if _, err := os.Stat(serviceDir); err == nil {
		return fmt.Errorf("service %q already exists at %s", name, serviceDir)
	}

	fmt.Printf("Creating custom service: %s\n", name)

	if err := os.MkdirAll(serviceDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Generate service files
	structName := toCamelCase(name)

	// Main service file
	serviceGo := generateServiceGo(name, svcType, structName)
	if err := os.WriteFile(filepath.Join(serviceDir, "service.go"), []byte(serviceGo), 0644); err != nil {
		return fmt.Errorf("failed to create service.go: %w", err)
	}
	fmt.Printf("  + Created services/%s/service.go\n", name)

	// Test file
	testGo := generateServiceTestGo(name, structName)
	if err := os.WriteFile(filepath.Join(serviceDir, "service_test.go"), []byte(testGo), 0644); err != nil {
		return fmt.Errorf("failed to create service_test.go: %w", err)
	}
	fmt.Printf("  + Created services/%s/service_test.go\n", name)

	fmt.Printf(`
Next steps:
  1. Edit services/%s/service.go to implement your service
  2. Add to robot.json:
     { "name": "%s_instance", "type": "%s", "model": "%s" }
  3. Run: gorai generate
`, name, name, svcType, name)

	return nil
}

func printAddServiceUsage() error {
	fmt.Println(`gorai add service - Create a custom service

Usage:
  gorai add service <name> [flags]

Flags:
  --type <type>       Service type: vision, behavior, etc. (default: behavior)
  -h, --help          Show this help

Examples:
  gorai add service my_detector --type vision
  gorai add service patrol --type behavior
  gorai add service custom_slam --type slam`)
	return nil
}

func validatePackageName(name string) error {
	if name == "" {
		return fmt.Errorf("name cannot be empty")
	}

	// Must start with letter
	if !unicode.IsLetter(rune(name[0])) {
		return fmt.Errorf("must start with a letter")
	}

	// Only letters, numbers, underscores
	for _, c := range name {
		if !unicode.IsLetter(c) && !unicode.IsDigit(c) && c != '_' {
			return fmt.Errorf("may only contain letters, numbers, and underscores")
		}
	}

	// Can't be a Go keyword
	goKeywords := map[string]bool{
		"break": true, "case": true, "chan": true, "const": true, "continue": true,
		"default": true, "defer": true, "else": true, "fallthrough": true, "for": true,
		"func": true, "go": true, "goto": true, "if": true, "import": true,
		"interface": true, "map": true, "package": true, "range": true, "return": true,
		"select": true, "struct": true, "switch": true, "type": true, "var": true,
	}
	if goKeywords[name] {
		return fmt.Errorf("%q is a Go keyword", name)
	}

	return nil
}

func toCamelCase(s string) string {
	parts := strings.Split(s, "_")
	for i := range parts {
		if len(parts[i]) > 0 {
			parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
		}
	}
	return strings.Join(parts, "")
}

func generateComponentGo(pkgName, compType, structName string) string {
	return fmt.Sprintf(`// Package %s implements a custom %s component.
package %s

import (
	"context"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent(%q, %q, New)
}

// Config holds configuration for %s.
type Config struct {
	// TODO: Add your configuration fields here
	SampleRate int `+"`json:\"sample_rate\"`"+`
}

// %s is a custom %s implementation.
type %s struct {
	name   resource.Name
	config Config
	// TODO: Add your fields here
}

// New creates a new %s from configuration.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	var cfg Config
	if err := mapToStruct(conf, &cfg); err != nil {
		return nil, err
	}

	return &%s{
		name:   resource.NewComponentName("", %q, conf["name"].(string)),
		config: cfg,
	}, nil
}

// Name returns the resource name.
func (c *%s) Name() resource.Name {
	return c.name
}

// Readings returns component readings.
func (c *%s) Readings(ctx context.Context) (map[string]any, error) {
	// TODO: Implement your reading logic
	return map[string]any{
		"value": 0.0,
	}, nil
}

// Reconfigure updates the component configuration.
func (c *%s) Reconfigure(ctx context.Context, deps registry.Dependencies, conf registry.Config) error {
	var cfg Config
	if err := mapToStruct(conf, &cfg); err != nil {
		return err
	}
	c.config = cfg
	return nil
}

// DoCommand handles arbitrary commands.
func (c *%s) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, nil
}

// Close cleans up the component.
func (c *%s) Close(ctx context.Context) error {
	return nil
}

// mapToStruct converts a map to a struct using JSON tags.
func mapToStruct(m registry.Config, v any) error {
	// Simple implementation - in production, use encoding/json or mapstructure
	// This is a placeholder
	return nil
}
`, pkgName, compType, pkgName, compType, pkgName, structName, structName, compType, structName,
		structName, structName, compType, structName, structName, structName, structName, structName)
}

func generateComponentTestGo(pkgName, structName string) string {
	return fmt.Sprintf(`package %s

import (
	"context"
	"testing"

	"github.com/emergingrobotics/gorai/pkg/registry"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	conf := registry.Config{
		"name":        "test_%s",
		"sample_rate": 100,
	}

	comp, err := New(ctx, nil, conf)
	if err != nil {
		t.Fatalf("New() error = %%v", err)
	}

	c, ok := comp.(*%s)
	if !ok {
		t.Fatalf("New() returned wrong type")
	}

	if c.name.Name != "test_%s" {
		t.Errorf("Name = %%v, want test_%s", c.name.Name)
	}
}

func TestReadings(t *testing.T) {
	ctx := context.Background()
	c := &%s{
		config: Config{SampleRate: 100},
	}

	readings, err := c.Readings(ctx)
	if err != nil {
		t.Fatalf("Readings() error = %%v", err)
	}

	if readings == nil {
		t.Error("Readings() returned nil")
	}
}

func TestClose(t *testing.T) {
	ctx := context.Background()
	c := &%s{}

	if err := c.Close(ctx); err != nil {
		t.Errorf("Close() error = %%v", err)
	}
}
`, pkgName, pkgName, structName, pkgName, pkgName, structName, structName)
}

func generateFakeComponentGo(pkgName, compType, structName string) string {
	return fmt.Sprintf(`// Package fake provides a fake implementation of %s for testing.
package fake

import (
	"context"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent(%q, %q, New)
}

// Fake%s is a fake implementation for testing.
type Fake%s struct {
	name resource.Name
}

// New creates a new fake component.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	return &Fake%s{
		name: resource.NewComponentName("", %q, conf["name"].(string)),
	}, nil
}

// Name returns the resource name.
func (f *Fake%s) Name() resource.Name {
	return f.name
}

// Readings returns fake readings.
func (f *Fake%s) Readings(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"value": 0.0,
		"fake":  true,
	}, nil
}

// Reconfigure does nothing for fake component.
func (f *Fake%s) Reconfigure(ctx context.Context, deps registry.Dependencies, conf registry.Config) error {
	return nil
}

// DoCommand does nothing for fake component.
func (f *Fake%s) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, nil
}

// Close does nothing for fake component.
func (f *Fake%s) Close(ctx context.Context) error {
	return nil
}
`, pkgName, compType, "fake_"+pkgName, structName, structName, structName, compType, structName, structName, structName, structName, structName)
}

func generateServiceGo(pkgName, svcType, structName string) string {
	return fmt.Sprintf(`// Package %s implements a custom %s service.
package %s

import (
	"context"

	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterService(%q, %q, New)
}

// Config holds configuration for %s.
type Config struct {
	// TODO: Add your configuration fields here
}

// %s is a custom %s service implementation.
type %s struct {
	name   resource.Name
	config Config
	deps   registry.Dependencies
	// TODO: Add your fields here
}

// New creates a new %s from configuration.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	var cfg Config
	if err := mapToStruct(conf, &cfg); err != nil {
		return nil, err
	}

	return &%s{
		name:   resource.NewServiceName("", %q, conf["name"].(string)),
		config: cfg,
		deps:   deps,
	}, nil
}

// Name returns the resource name.
func (s *%s) Name() resource.Name {
	return s.name
}

// Run executes the service's main logic.
func (s *%s) Run(ctx context.Context) error {
	// TODO: Implement your service logic
	// This is called periodically or runs continuously
	return nil
}

// Reconfigure updates the service configuration.
func (s *%s) Reconfigure(ctx context.Context, deps registry.Dependencies, conf registry.Config) error {
	var cfg Config
	if err := mapToStruct(conf, &cfg); err != nil {
		return err
	}
	s.config = cfg
	s.deps = deps
	return nil
}

// DoCommand handles arbitrary commands.
func (s *%s) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, nil
}

// Close cleans up the service.
func (s *%s) Close(ctx context.Context) error {
	return nil
}

// mapToStruct converts a map to a struct using JSON tags.
func mapToStruct(m registry.Config, v any) error {
	// Simple implementation - in production, use encoding/json or mapstructure
	return nil
}
`, pkgName, svcType, pkgName, svcType, pkgName, structName, structName, svcType, structName,
		structName, structName, svcType, structName, structName, structName, structName, structName)
}

func generateServiceTestGo(pkgName, structName string) string {
	return fmt.Sprintf(`package %s

import (
	"context"
	"testing"

	"github.com/emergingrobotics/gorai/pkg/registry"
)

func TestNew(t *testing.T) {
	ctx := context.Background()
	conf := registry.Config{
		"name": "test_%s",
	}

	svc, err := New(ctx, nil, conf)
	if err != nil {
		t.Fatalf("New() error = %%v", err)
	}

	s, ok := svc.(*%s)
	if !ok {
		t.Fatalf("New() returned wrong type")
	}

	if s.name.Name != "test_%s" {
		t.Errorf("Name = %%v, want test_%s", s.name.Name)
	}
}

func TestRun(t *testing.T) {
	ctx := context.Background()
	s := &%s{
		config: Config{},
	}

	if err := s.Run(ctx); err != nil {
		t.Errorf("Run() error = %%v", err)
	}
}

func TestClose(t *testing.T) {
	ctx := context.Background()
	s := &%s{}

	if err := s.Close(ctx); err != nil {
		t.Errorf("Close() error = %%v", err)
	}
}
`, pkgName, pkgName, structName, pkgName, pkgName, structName, structName)
}
