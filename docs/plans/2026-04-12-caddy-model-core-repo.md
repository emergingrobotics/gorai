# Caddy Model Core Repo Cleanup — Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor the gorai core repo so external Go modules can import it and build robot binaries using the Caddy model (blank imports + `gorai.Run()`).

**Architecture:** Export a `Run()` entrypoint from `cmd/gorai`. Add dependency-ordered component creation to `pkg/robot`. Replace the old discovery/metadata packages with a simple JSON registry client. Move hardware-specific components to archive. Core keeps only interfaces, fakes, remote proxies, and the runtime.

**Tech Stack:** Go 1.25, embedded NATS, cobra (for component subcommands)

**Spec:** `/er/gorai/REQUIREMENTS.md` sections 3, 5, 6, 8, 9

---

## Scope

This plan covers **REQUIREMENTS.md Phase 1: Core Repo Cleanup** only. It does NOT create external repos (template, PiCar-X drivers). Those are separate plans that depend on this one completing first.

**What this plan produces:**
1. An exported `Run()` function that external `main.go` files can call
2. Dependency-ordered component creation (topological sort on `depends_on`)
3. Component instances available via `deps.Get(name)` to downstream components
4. Registry-aware validation (missing type+model produces actionable error)
5. Simplified `gorai component search/add` backed by a static JSON registry
6. Hardware-specific components moved to `archive/` — core has only interfaces + fakes + remote proxies
7. All existing tests pass

**What it does NOT produce:**
- External component repos (separate plan)
- Template repo (separate plan)
- PiCar-X drivers (separate plan)

---

## File Map

### New Files

| File | Responsibility |
|------|---------------|
| `pkg/gorai/run.go` | Exports `Run()` for external main.go callers |
| `pkg/robot/deps.go` | Dependency-aware `robotDeps` that exposes created component instances |
| `pkg/robot/topo.go` | Topological sort of components by `depends_on` |
| `pkg/robot/topo_test.go` | Tests for topological sort |
| `pkg/componentregistry/registry.go` | JSON registry client (fetch, cache, search) |
| `pkg/componentregistry/registry_test.go` | Tests for registry client |
| `pkg/componentregistry/types.go` | Registry JSON types |
| `pkg/componentregistry/mainfile.go` | Reads and edits main.go blank imports |
| `pkg/componentregistry/mainfile_test.go` | Tests for main.go editing |

### Modified Files

| File | What Changes |
|------|-------------|
| `cmd/gorai/main.go` | Calls `Run()` internally; remote proxies stay, hardware components removed |
| `cmd/gorai/commands/root.go` | No changes needed (already dispatches correctly) |
| `cmd/gorai/commands/component.go` | Rewrite: use `pkg/componentregistry` instead of `pkg/discovery` + `pkg/components/metadata` |
| `cmd/gorai/commands/validate.go` | Add registry-aware check for unregistered type+model pairs |
| `pkg/robot/robot.go` | Use topo sort for component creation; pass created components into deps |

### Archived Files (moved, not deleted)

| File | Destination |
|------|------------|
| `components/serial/` | `archive/components/serial/` |
| `components/input/keyboard/` | `archive/components/input/keyboard/` |
| `components/pwm/gpiod/` | `archive/components/pwm/gpiod/` |
| `services/control/keypress_motor/` | `archive/services/control/keypress_motor/` |
| `services/control/l298n/` | `archive/services/control/l298n/` |
| `services/control/mecanum/` | `archive/services/control/mecanum/` |
| `services/control/velocity_input/` | `archive/services/control/velocity_input/` |
| `services/bridge/keyboard_publisher/` | `archive/services/bridge/keyboard_publisher/` |
| `pkg/discovery/` | `archive/pkg/discovery/` |
| `pkg/components/metadata/` | `archive/pkg/components/metadata/` |
| `pkg/systemd/` | `archive/pkg/systemd/` |

### Unchanged Files (for reference)

| File | Why It Matters |
|------|---------------|
| `pkg/registry/registry.go` | Already correct. `RegisterComponent`, `LookupComponent`, `ListComponents` are the API external modules use. |
| `pkg/config/config.go` | RDL structs including `ComponentConfig.DependsOn`. Already parsed, just not used at runtime. |
| `pkg/embeddednats/server.go` | Already implemented. No changes needed. |
| `components/*/remote/` | Remote proxy components stay in core. They are infrastructure, not hardware-specific. |
| `components/*/fake/` | Fake components stay in core. Used for testing. |

---

## Task 1: Export Run() Entrypoint

**Files:**
- Create: `cmd/gorai/entrypoint.go`
- Modify: `cmd/gorai/main.go`

**Spec:** REQ-CADDY-4

- [ ] **Step 1: Create entrypoint.go with Run()**

```go
// cmd/gorai/entrypoint.go
package main

import (
	"fmt"
	"os"

	"github.com/gorai/gorai/cmd/gorai/commands"
)

// Run is the main entrypoint for GoRAI robot projects.
// External robot projects call this from their main.go after
// blank-importing their component packages.
//
// Usage in a robot project's main.go:
//
//	package main
//
//	import (
//	    gorai "github.com/gorai/gorai/cmd/gorai"
//	    _ "github.com/emergingrobotics/gorai-picarx"
//	)
//
//	func main() {
//	    gorai.Run()
//	}
func Run() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

Note: This lives in `package main` alongside the existing `main.go`. For external callers, the package path is `github.com/gorai/gorai/cmd/gorai`. The problem: Go doesn't allow importing a `package main`. We need a different approach.

**Revised approach:** Create a new importable package.

- [ ] **Step 1 (revised): Create an importable entrypoint package**

Create `pkg/gorai/run.go`:

```go
// Package gorai provides the main entrypoint for GoRAI robot projects.
package gorai

import (
	"fmt"
	"os"

	"github.com/gorai/gorai/cmd/gorai/commands"
)

// Run is the main entrypoint for GoRAI robot projects.
// Call this from your main.go after blank-importing component packages.
func Run() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./pkg/gorai/`
Expected: no errors

- [ ] **Step 3: Update the core main.go to use Run()**

Modify `cmd/gorai/main.go`:

```go
package main

import (
	"github.com/gorai/gorai/pkg/gorai"

	// Remote proxy components (core infrastructure, stay in core)
	_ "github.com/gorai/gorai/components/camera/remote"
	_ "github.com/gorai/gorai/components/gpio/remote"
	_ "github.com/gorai/gorai/components/input/remote"
	_ "github.com/gorai/gorai/components/motor/remote"
	_ "github.com/gorai/gorai/components/pwm/input/remote"
	_ "github.com/gorai/gorai/components/pwm/remote"
	_ "github.com/gorai/gorai/components/sensor/encoder/remote"
)

func main() {
	gorai.Run()
}
```

Note: Hardware-specific imports (serial, keyboard, pwm/gpiod, control services) are removed. Only remote proxies remain.

- [ ] **Step 4: Verify it builds**

Run: `go build -o /tmp/gorai-test ./cmd/gorai && /tmp/gorai-test version`
Expected: `gorai version 0.1.0`

- [ ] **Step 5: Commit**

```bash
git add pkg/gorai/run.go cmd/gorai/main.go
git commit -m "add exported Run() entrypoint for Caddy model robot projects."
```

---

## Task 2: Topological Sort for Component Creation

**Files:**
- Create: `pkg/robot/topo.go`
- Create: `pkg/robot/topo_test.go`

**Spec:** REQ-DI-1

- [ ] **Step 1: Write failing tests for topological sort**

```go
// pkg/robot/topo_test.go
package robot

import (
	"testing"

	"github.com/gorai/gorai/pkg/config"
)

func TestTopoSort_NoDeps(t *testing.T) {
	components := []config.ComponentConfig{
		{Name: "a", Type: "sensor", Model: "fake"},
		{Name: "b", Type: "motor", Model: "fake"},
	}
	sorted, err := topoSortComponents(components)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sorted) != 2 {
		t.Fatalf("expected 2 components, got %d", len(sorted))
	}
}

func TestTopoSort_LinearDeps(t *testing.T) {
	components := []config.ComponentConfig{
		{Name: "servo", Type: "servo", Model: "fake", DependsOn: []string{"mcu"}},
		{Name: "mcu", Type: "i2c_device", Model: "fake"},
	}
	sorted, err := topoSortComponents(components)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sorted[0].Name != "mcu" {
		t.Fatalf("expected mcu first, got %s", sorted[0].Name)
	}
	if sorted[1].Name != "servo" {
		t.Fatalf("expected servo second, got %s", sorted[1].Name)
	}
}

func TestTopoSort_DiamondDeps(t *testing.T) {
	components := []config.ComponentConfig{
		{Name: "base", Type: "base", Model: "fake", DependsOn: []string{"left", "right"}},
		{Name: "left", Type: "motor", Model: "fake", DependsOn: []string{"mcu"}},
		{Name: "right", Type: "motor", Model: "fake", DependsOn: []string{"mcu"}},
		{Name: "mcu", Type: "i2c_device", Model: "fake"},
	}
	sorted, err := topoSortComponents(components)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// mcu must come before left and right, both must come before base
	indexOf := func(name string) int {
		for i, c := range sorted {
			if c.Name == name {
				return i
			}
		}
		return -1
	}
	if indexOf("mcu") >= indexOf("left") || indexOf("mcu") >= indexOf("right") {
		t.Fatal("mcu must come before left and right")
	}
	if indexOf("left") >= indexOf("base") || indexOf("right") >= indexOf("base") {
		t.Fatal("left and right must come before base")
	}
}

func TestTopoSort_CyclicDeps(t *testing.T) {
	components := []config.ComponentConfig{
		{Name: "a", Type: "x", Model: "fake", DependsOn: []string{"b"}},
		{Name: "b", Type: "x", Model: "fake", DependsOn: []string{"a"}},
	}
	_, err := topoSortComponents(components)
	if err == nil {
		t.Fatal("expected error for cyclic dependency")
	}
}

func TestTopoSort_MissingDep(t *testing.T) {
	components := []config.ComponentConfig{
		{Name: "a", Type: "x", Model: "fake", DependsOn: []string{"nonexistent"}},
	}
	_, err := topoSortComponents(components)
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/robot/ -run TestTopoSort -v`
Expected: compile error (topoSortComponents undefined)

- [ ] **Step 3: Implement topological sort**

```go
// pkg/robot/topo.go
package robot

import (
	"fmt"
	"strings"

	"github.com/gorai/gorai/pkg/config"
)

// topoSortComponents returns components ordered so that dependencies come
// before dependents. Returns an error for cycles or missing dependencies.
func topoSortComponents(components []config.ComponentConfig) ([]config.ComponentConfig, error) {
	byName := make(map[string]config.ComponentConfig, len(components))
	for _, c := range components {
		byName[c.Name] = c
	}

	// Check for missing dependencies
	for _, c := range components {
		for _, dep := range c.DependsOn {
			if _, ok := byName[dep]; !ok {
				return nil, fmt.Errorf("component %q depends on %q which is not defined", c.Name, dep)
			}
		}
	}

	// Kahn's algorithm
	inDegree := make(map[string]int, len(components))
	for _, c := range components {
		if _, ok := inDegree[c.Name]; !ok {
			inDegree[c.Name] = 0
		}
		for _, dep := range c.DependsOn {
			inDegree[c.Name]++
			_ = dep
		}
	}

	// Build adjacency: dep -> list of dependents
	dependents := make(map[string][]string)
	for _, c := range components {
		for _, dep := range c.DependsOn {
			dependents[dep] = append(dependents[dep], c.Name)
		}
	}

	// Start with nodes that have no dependencies
	var queue []string
	for _, c := range components {
		if inDegree[c.Name] == 0 {
			queue = append(queue, c.Name)
		}
	}

	var sorted []config.ComponentConfig
	for len(queue) > 0 {
		name := queue[0]
		queue = queue[1:]
		sorted = append(sorted, byName[name])

		for _, dependent := range dependents[name] {
			inDegree[dependent]--
			if inDegree[dependent] == 0 {
				queue = append(queue, dependent)
			}
		}
	}

	if len(sorted) != len(components) {
		// Find the cycle for a useful error message
		var stuck []string
		for name, degree := range inDegree {
			if degree > 0 {
				stuck = append(stuck, name)
			}
		}
		return nil, fmt.Errorf("circular dependency involving: %s", strings.Join(stuck, ", "))
	}

	return sorted, nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/robot/ -run TestTopoSort -v`
Expected: all 5 tests PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/robot/topo.go pkg/robot/topo_test.go
git commit -m "add topological sort for dependency-ordered component creation."
```

---

## Task 3: Dependency-Aware Component Creation

**Files:**
- Create: `pkg/robot/deps.go`
- Modify: `pkg/robot/robot.go`

**Spec:** REQ-DI-2, REQ-DI-3, REQ-DI-4

This task changes `Robot.Start()` to:
1. Topologically sort components by `depends_on`
2. Create components in sorted order
3. Make each created component available to subsequent components via `deps.Get(name)`

- [ ] **Step 1: Create componentDeps struct**

```go
// pkg/robot/deps.go
package robot

import (
	"fmt"
	"log/slog"

	"github.com/gorai/gorai/pkg/registry"
	"github.com/nats-io/nats.go"
)

// componentDeps provides access to already-created components, the NATS
// connection, and the logger. Passed to component constructors.
// It replaces the old robotDeps and satisfies registry.Dependencies.
type componentDeps struct {
	components map[string]any
	natsConn   *nats.Conn
	logger     *slog.Logger
}

// Compile-time check that componentDeps implements registry.Dependencies.
var _ registry.Dependencies = (*componentDeps)(nil)

func newComponentDeps(nc *nats.Conn, logger *slog.Logger) *componentDeps {
	return &componentDeps{
		components: make(map[string]any),
		natsConn:   nc,
		logger:     logger,
	}
}

// Get returns a named dependency. Components created earlier in the
// topological order are available. "nats" and "logger" are always available.
func (d *componentDeps) Get(name string) (any, error) {
	switch name {
	case "nats":
		return d.natsConn, nil
	case "logger":
		return d.logger, nil
	}
	if comp, ok := d.components[name]; ok {
		return comp, nil
	}
	return nil, fmt.Errorf("dependency %q not found", name)
}

// GetByType returns all components (not filtered by type yet).
func (d *componentDeps) GetByType(subtype string) ([]any, error) {
	return nil, nil
}

// Add registers a created component so downstream components can access it.
func (d *componentDeps) Add(name string, component any) {
	d.components[name] = component
}
```

- [ ] **Step 2: Write integration test for dependency injection**

```go
// pkg/robot/deps_test.go
package robot

import (
	"testing"
)

func TestComponentDeps_GetNatsAndLogger(t *testing.T) {
	deps := newComponentDeps(nil, nil)
	// nats and logger always available (even if nil)
	_, err := deps.Get("nats")
	if err != nil {
		t.Fatalf("expected nats to be available: %v", err)
	}
	_, err = deps.Get("logger")
	if err != nil {
		t.Fatalf("expected logger to be available: %v", err)
	}
}

func TestComponentDeps_GetCreatedComponent(t *testing.T) {
	deps := newComponentDeps(nil, nil)
	deps.Add("mcu", "fake-mcu-instance")

	val, err := deps.Get("mcu")
	if err != nil {
		t.Fatalf("expected mcu to be available: %v", err)
	}
	if val != "fake-mcu-instance" {
		t.Fatalf("expected fake-mcu-instance, got %v", val)
	}
}

func TestComponentDeps_GetMissing(t *testing.T) {
	deps := newComponentDeps(nil, nil)
	_, err := deps.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing dependency")
	}
}
```

- [ ] **Step 3: Run deps tests**

Run: `go test ./pkg/robot/ -run TestComponentDeps -v`
Expected: all 3 tests PASS

- [ ] **Step 4: Modify robot.go Start() to use topo sort and componentDeps**

In `pkg/robot/robot.go`, make these specific changes:

**a) Delete the old `robotDeps` struct and its methods** (lines 26-42 in current file). The new `componentDeps` in `deps.go` replaces it.

**b) Add a `sharedDeps *componentDeps` field to the `Robot` struct:**
```go
// In the Robot struct, add after the `topics` field:
sharedDeps *componentDeps
```

**c) Replace the component creation loop in `Start()`**. Find:
```go
// Initialize and start components
for _, comp := range r.cfg.Components {
```
Replace with:
```go
// Sort components by dependency order
sortedComponents, err := topoSortComponents(r.cfg.Components)
if err != nil {
    return fmt.Errorf("component dependency error: %w", err)
}

// Create shared deps that accumulates created components
r.sharedDeps = newComponentDeps(r.nats.Conn(), r.logger)

// Initialize and start components in dependency order
for _, comp := range sortedComponents {
```

**d) Modify `startRegistryComponent` signature** to accept and use `*componentDeps`:

Change the method to accept the shared deps instead of creating fresh ones:
```go
func (r *Robot) startRegistryComponent(ctx context.Context, comp config.ComponentConfig) error {
    ctor, err := registry.LookupComponent(comp.Type, comp.Model)
    // ... error handling ...

    conf := registry.Config{
        "name":  comp.Name,
        "type":  comp.Type,
        "model": comp.Model,
        "nats_url":   r.getNATSURL(),
        "namespace":  "gorai",
        "robot_name": r.cfg.Robot.Name,
    }
    for k, v := range comp.Attributes {
        conf[k] = v
    }

    // Use shared deps (includes already-created components)
    component, err := ctor(ctx, r.sharedDeps, conf)
    // ... error handling, start, track ...

    // Register this component so downstream components can access it
    r.sharedDeps.Add(comp.Name, component)
    // ... rest of method unchanged ...
}
```

**e) Apply the same pattern to `startInternalService`**: services should also receive `r.sharedDeps` so they can access components. The `serviceDeps` struct can be kept for the `resource.Dependencies` interface (used by `Reconfigure`), but the constructor should receive `r.sharedDeps`.

- [ ] **Step 5: Run full build**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add pkg/robot/deps.go pkg/robot/deps_test.go pkg/robot/robot.go
git commit -m "add dependency-ordered component creation with inter-component deps."
```

---

## Task 4: Registry-Aware Validation

**Files:**
- Modify: `cmd/gorai/commands/validate.go`
- Modify: `pkg/robot/robot.go` (the `startRegistryComponent` error path)
- Modify: `pkg/registry/registry.go` (add `IsRegistered` helper)

**Spec:** REQ-CORE-4, REQ-CLI-VALIDATE-1

- [ ] **Step 1: Add IsRegistered helper to registry**

```go
// In pkg/registry/registry.go, add:

// IsRegistered returns true if a component with the given subtype and model
// has been registered.
func IsRegistered(subtype, model string) bool {
	mu.RLock()
	defer mu.RUnlock()
	if m, ok := components[subtype]; ok {
		_, ok := m[model]
		return ok
	}
	return false
}
```

- [ ] **Step 2: Read the current validate command**

Read: `cmd/gorai/commands/validate.go`
Understand what it currently checks.

- [ ] **Step 3: Add registry check to validate command**

After the existing `cfg.Validate()` call in the validate command, add a loop that checks every component against the compiled-in registry:

```go
// Check that all referenced components are registered
var missing []string
for _, comp := range cfg.Components {
    if comp.Disabled {
        continue
    }
    if !registry.IsRegistered(comp.Type, comp.Model) {
        missing = append(missing, fmt.Sprintf(
            "  component %q: type %q model %q not registered.\n"+
            "    Try: gorai component search %s",
            comp.Name, comp.Type, comp.Model, comp.Type))
    }
}
if len(missing) > 0 {
    fmt.Fprintf(os.Stderr, "WARNING: %d component(s) not found in compiled-in registry:\n", len(missing))
    for _, m := range missing {
        fmt.Fprintln(os.Stderr, m)
    }
    fmt.Fprintln(os.Stderr, "\nThe binary may not have these components compiled in.")
    fmt.Fprintln(os.Stderr, "Add blank imports to main.go or install with: gorai component add <name>")
}
```

This is a WARNING, not an error, because the user may be validating an RDL before building a binary that includes the components.

- [ ] **Step 4: Also improve the runtime error in robot.go**

In `pkg/robot/robot.go`, in the `startRegistryComponent` method, when `registry.LookupComponent` fails, make the error actionable:

```go
ctor, err := registry.LookupComponent(comp.Type, comp.Model)
if err != nil {
    return fmt.Errorf("component %q: type %q model %q not found in registry.\n"+
        "  This component may need to be installed. Try:\n"+
        "    gorai component search %s", comp.Name, comp.Type, comp.Model, comp.Type)
}
```

- [ ] **Step 5: Build and verify**

Run: `go build ./...`
Expected: no errors

- [ ] **Step 6: Commit**

```bash
git add pkg/registry/registry.go cmd/gorai/commands/validate.go pkg/robot/robot.go
git commit -m "add registry-aware validation with actionable missing component errors."
```

---

## Task 5: Simple JSON Registry Client

**Files:**
- Create: `pkg/componentregistry/types.go`
- Create: `pkg/componentregistry/registry.go`
- Create: `pkg/componentregistry/registry_test.go`

**Spec:** REQ-REGISTRY-1, REQ-REGISTRY-3 (REQ-REGISTRY-2 local caching deferred to Phase 4)

This replaces the old `pkg/discovery/` and `pkg/components/metadata/` packages with a much simpler implementation backed by a static JSON file. HTTP fetch and local caching (REQ-REGISTRY-2) are deferred to the Component CLI phase -- this task provides the core data structures and file-based loading that the CLI will build on.

- [ ] **Step 1: Write the types**

```go
// pkg/componentregistry/types.go
package componentregistry

// Registry is the top-level registry JSON structure.
type Registry struct {
	Version    string               `json:"version"`
	Components map[string]Component `json:"components"`
}

// Component is a single entry in the registry.
type Component struct {
	Module      string   `json:"module"`
	Type        string   `json:"type"`
	Model       string   `json:"model"`
	Description string   `json:"description"`
	Hardware    string   `json:"hardware,omitempty"`
	Version     string   `json:"version"`
	Tags        []string `json:"tags,omitempty"`
}
```

- [ ] **Step 2: Write failing tests**

```go
// pkg/componentregistry/registry_test.go
package componentregistry

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "registry.json")
	os.WriteFile(path, []byte(`{
		"version": "1",
		"components": {
			"sensor/hc-sr04": {
				"module": "github.com/emergingrobotics/gorai-driver-hcsr04",
				"type": "sensor",
				"model": "hc-sr04",
				"description": "HC-SR04 ultrasonic",
				"version": "v0.1.0",
				"tags": ["ultrasonic", "gpio"]
			}
		}
	}`), 0644)

	reg, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reg.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(reg.Components))
	}
	c := reg.Components["sensor/hc-sr04"]
	if c.Module != "github.com/emergingrobotics/gorai-driver-hcsr04" {
		t.Fatalf("wrong module: %s", c.Module)
	}
}

func TestSearch(t *testing.T) {
	reg := &Registry{
		Components: map[string]Component{
			"sensor/hc-sr04":  {Type: "sensor", Model: "hc-sr04", Description: "HC-SR04 ultrasonic", Tags: []string{"ultrasonic"}},
			"servo/dynamixel": {Type: "servo", Model: "dynamixel", Description: "Dynamixel smart servo", Tags: []string{"serial"}},
		},
	}
	results := reg.Search("ultrasonic")
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Model != "hc-sr04" {
		t.Fatalf("wrong result: %s", results[0].Model)
	}
}

func TestSearchByType(t *testing.T) {
	reg := &Registry{
		Components: map[string]Component{
			"sensor/hc-sr04":  {Type: "sensor", Model: "hc-sr04", Description: "ultrasonic"},
			"sensor/bno055":   {Type: "sensor", Model: "bno055", Description: "IMU"},
			"servo/dynamixel": {Type: "servo", Model: "dynamixel", Description: "servo"},
		},
	}
	results := reg.Search("sensor")
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
}

func TestLookup(t *testing.T) {
	reg := &Registry{
		Components: map[string]Component{
			"sensor/hc-sr04": {Module: "github.com/example/hcsr04", Type: "sensor", Model: "hc-sr04"},
		},
	}
	c, ok := reg.Lookup("sensor/hc-sr04")
	if !ok {
		t.Fatal("expected to find sensor/hc-sr04")
	}
	if c.Module != "github.com/example/hcsr04" {
		t.Fatalf("wrong module: %s", c.Module)
	}

	_, ok = reg.Lookup("sensor/nonexistent")
	if ok {
		t.Fatal("expected not to find sensor/nonexistent")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail**

Run: `go test ./pkg/componentregistry/ -v`
Expected: compile error (package does not exist yet)

- [ ] **Step 4: Implement registry client**

```go
// pkg/componentregistry/registry.go
package componentregistry

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// LoadFromFile loads a registry from a local JSON file.
func LoadFromFile(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read registry file: %w", err)
	}
	var reg Registry
	if err := json.Unmarshal(data, &reg); err != nil {
		return nil, fmt.Errorf("parse registry JSON: %w", err)
	}
	return &reg, nil
}

// Search finds components matching a query string. Matches against name,
// type, model, description, and tags. Case-insensitive.
func (r *Registry) Search(query string) []Component {
	q := strings.ToLower(query)
	var results []Component
	for name, c := range r.Components {
		if matches(q, name, c) {
			results = append(results, c)
		}
	}
	return results
}

// Lookup finds a component by its registry key (e.g., "sensor/hc-sr04").
func (r *Registry) Lookup(key string) (Component, bool) {
	c, ok := r.Components[key]
	return c, ok
}

func matches(query, name string, c Component) bool {
	fields := []string{
		strings.ToLower(name),
		strings.ToLower(c.Type),
		strings.ToLower(c.Model),
		strings.ToLower(c.Description),
	}
	for _, tag := range c.Tags {
		fields = append(fields, strings.ToLower(tag))
	}
	for _, field := range fields {
		if strings.Contains(field, query) {
			return true
		}
	}
	return false
}
```

- [ ] **Step 5: Run tests**

Run: `go test ./pkg/componentregistry/ -v`
Expected: all 4 tests PASS

- [ ] **Step 6: Commit**

```bash
git add pkg/componentregistry/
git commit -m "add simple JSON registry client for component search/lookup."
```

---

## Task 6: main.go Editor for gorai component add

**Files:**
- Create: `pkg/componentregistry/mainfile.go`
- Create: `pkg/componentregistry/mainfile_test.go`

**Spec:** REQ-CLI-ADD-1, REQ-CLI-ADD-2

- [ ] **Step 1: Write failing tests**

```go
// pkg/componentregistry/mainfile_test.go
package componentregistry

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAddBlankImport(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.go")
	os.WriteFile(mainPath, []byte(`package main

import (
	"github.com/gorai/gorai/pkg/gorai"
)

func main() {
	gorai.Run()
}
`), 0644)

	err := AddBlankImport(mainPath, "github.com/example/gorai-driver-hcsr04")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(mainPath)
	if !strings.Contains(string(content), `_ "github.com/example/gorai-driver-hcsr04"`) {
		t.Fatalf("import not added. Content:\n%s", content)
	}
}

func TestAddBlankImport_AlreadyPresent(t *testing.T) {
	dir := t.TempDir()
	mainPath := filepath.Join(dir, "main.go")
	os.WriteFile(mainPath, []byte(`package main

import (
	"github.com/gorai/gorai/pkg/gorai"
	_ "github.com/example/gorai-driver-hcsr04"
)

func main() {
	gorai.Run()
}
`), 0644)

	err := AddBlankImport(mainPath, "github.com/example/gorai-driver-hcsr04")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content, _ := os.ReadFile(mainPath)
	count := strings.Count(string(content), "gorai-driver-hcsr04")
	if count != 1 {
		t.Fatalf("expected 1 occurrence, got %d", count)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./pkg/componentregistry/ -run TestAddBlankImport -v`
Expected: compile error (AddBlankImport undefined)

- [ ] **Step 3: Implement AddBlankImport**

```go
// pkg/componentregistry/mainfile.go
package componentregistry

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"os"
	"strconv"
	"strings"
)

// AddBlankImport adds a blank import to a Go source file if not already present.
// The file is reformatted with gofmt after editing.
func AddBlankImport(filePath, modulePath string) error {
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, filePath, nil, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("parse %s: %w", filePath, err)
	}

	quotedPath := strconv.Quote(modulePath)

	// Check if already imported
	for _, imp := range node.Imports {
		if imp.Path.Value == quotedPath {
			return nil
		}
	}

	// Find or create import declaration
	var targetDecl *ast.GenDecl
	for _, decl := range node.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if ok && genDecl.Tok == token.IMPORT && genDecl.Lparen.IsValid() {
			targetDecl = genDecl
			break
		}
	}

	newSpec := &ast.ImportSpec{
		Name: &ast.Ident{Name: "_"},
		Path: &ast.BasicLit{Kind: token.STRING, Value: quotedPath},
	}

	if targetDecl != nil {
		targetDecl.Specs = append(targetDecl.Specs, newSpec)
	} else {
		newDecl := &ast.GenDecl{
			Tok:    token.IMPORT,
			Lparen: 1,
			Specs:  []ast.Spec{newSpec},
		}
		node.Decls = append([]ast.Decl{newDecl}, node.Decls...)
	}

	// Write back
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("write %s: %w", filePath, err)
	}
	defer f.Close()

	if err := format.Node(f, fset, node); err != nil {
		return fmt.Errorf("format %s: %w", filePath, err)
	}

	return nil
}

// HasBlankImport checks if a file already has a blank import for the given module.
func HasBlankImport(filePath, modulePath string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}
	return strings.Contains(string(data), strconv.Quote(modulePath)), nil
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./pkg/componentregistry/ -v`
Expected: all tests PASS

- [ ] **Step 5: Commit**

```bash
git add pkg/componentregistry/mainfile.go pkg/componentregistry/mainfile_test.go
git commit -m "add main.go blank import editor for gorai component add."
```

---

## Task 7: Rewrite gorai component Command

**Files:**
- Rewrite: `cmd/gorai/commands/component.go`

**Spec:** REQ-CLI-SEARCH-1, REQ-CLI-ADD-1, REQ-CLI-ADD-3, REQ-CLI-INFO-1

This rewrites the component command to use the new simple registry client. Drops cobra dependency (manual arg parsing like the rest of the CLI). Drops old `pkg/discovery` and `pkg/components/metadata` imports. Only 3 subcommands for launch: search, add, info.

- [ ] **Step 1: Rewrite component.go**

```go
// cmd/gorai/commands/component.go
package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gorai/gorai/pkg/componentregistry"
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

	// REQ-CLI-ADD-3: if it looks like a Go module path (contains a dot), use directly
	modulePath := arg
	if !strings.Contains(arg, ".") {
		// Look up in registry
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
	goTidy.Run()

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
	// Try local registry.json first, then ~/.gorai/registry.json
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
```

- [ ] **Step 2: Update root.go to call the new dispatch function**

In `cmd/gorai/commands/root.go`, change the `case "component":` handler:

```go
case "component":
    return cmdComponentDispatch()
```

This replaces the old `cmdComponent()` which used cobra. Remove the old `cmdComponent()` function.

- [ ] **Step 3: Verify build**

Run: `go build ./...`
Expected: no errors (the old cobra import and discovery/metadata imports are gone)

- [ ] **Step 4: Commit**

```bash
git add cmd/gorai/commands/component.go cmd/gorai/commands/root.go
git commit -m "rewrite gorai component to use simple JSON registry, drop cobra."
```

---

## Task 8: Archive Hardware-Specific Components

**Files:**
- Move multiple directories to `archive/`

**Spec:** REQUIREMENTS.md section 5.1

- [ ] **Step 1: Move hardware-specific code to archive**

Archive paths preserve the original directory structure for easy restoration.

```bash
mkdir -p archive/components/serial
mkdir -p archive/components/input/keyboard
mkdir -p archive/components/pwm/gpiod
mkdir -p archive/services/bridge/keyboard_publisher
mkdir -p archive/services/control/keypress_motor
mkdir -p archive/services/control/l298n
mkdir -p archive/services/control/mecanum
mkdir -p archive/services/control/velocity_input
mkdir -p archive/pkg/discovery
mkdir -p archive/pkg/components/metadata
mkdir -p archive/pkg/systemd

# Hardware-specific components
git mv components/serial/* archive/components/serial/
git mv components/input/keyboard/* archive/components/input/keyboard/
git mv components/pwm/gpiod/* archive/components/pwm/gpiod/

# Robot-specific control services
git mv services/bridge/keyboard_publisher/* archive/services/bridge/keyboard_publisher/
git mv services/control/keypress_motor/* archive/services/control/keypress_motor/
git mv services/control/l298n/* archive/services/control/l298n/
git mv services/control/mecanum/* archive/services/control/mecanum/
git mv services/control/velocity_input/* archive/services/control/velocity_input/

# Old discovery/metadata packages (replaced by componentregistry)
git mv pkg/discovery/* archive/pkg/discovery/
git mv pkg/components/metadata/* archive/pkg/components/metadata/

# Container management (not needed for single binary)
git mv pkg/systemd/* archive/pkg/systemd/
```

- [ ] **Step 2: Update main.go imports**

Remove the hardware-specific blank imports from `cmd/gorai/main.go` (if not already done in Task 1).

- [ ] **Step 3: Fix any broken imports in remaining code**

Run: `go build ./... 2>&1`
Fix any compilation errors from moved packages. The component.go rewrite (Task 7) should have already removed the discovery/metadata imports.

- [ ] **Step 4: Run all tests**

Run: `go test ./... 2>&1`
Expected: all tests pass. Tests in archived packages are not run (they're not in the module path).

- [ ] **Step 5: Commit**

```bash
git add archive/ components/ services/ pkg/discovery/ pkg/components/ pkg/systemd/ cmd/gorai/main.go
git commit -m "archive hardware-specific components and old discovery packages."
```

---

## Task 9: Final Integration Test

**Files:** none (verification only)

- [ ] **Step 1: Full build**

Run: `go build -o /tmp/gorai-test ./cmd/gorai`
Expected: binary produced, no errors

- [ ] **Step 2: Version check**

Run: `/tmp/gorai-test version`
Expected: `gorai version 0.1.0`

- [ ] **Step 3: Help check**

Run: `/tmp/gorai-test help`
Expected: shows all commands including `component`

- [ ] **Step 4: Validate with minimal RDL**

Create `/tmp/test-robot.json`:
```json
{"version": "2", "robot": {"name": "test"}}
```

Run: `/tmp/gorai-test validate /tmp/test-robot.json`
Expected: validation passes (no components to check)

- [ ] **Step 5: Full test suite**

Run: `go test ./... 2>&1 | grep -E "FAIL|ok"`
Expected: no FAIL lines, all `ok` lines

- [ ] **Step 6: Verify no container references in active code**

Run: `grep -r "podman\|Containerfile\|Dockerfile" cmd/ pkg/ --include="*.go" | grep -v "_test.go" | grep -v archive/`
Expected: no matches (or only in archived/deprecated code)

- [ ] **Step 7: Commit verification marker**

```bash
git commit --allow-empty -m "verified: Caddy model core repo cleanup complete."
```
