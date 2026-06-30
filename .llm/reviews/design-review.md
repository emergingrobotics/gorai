# Design/Architecture Review: Caddy Model Core Repo Cleanup

**Date:** 2026-04-12
**Reviewer:** Claude Opus 4.6 (automated)
**Scope:** Interface boundaries, registry API, dependency injection, Run() entrypoint, CLI commands, build command, code quality
**Design docs:** `/er/gorai/REQUIREMENTS.md`, `/er/gorai/docs/package-dev-approach.md`

---

## Summary

The Caddy model implementation is structurally sound. The core patterns -- init()-based registration, topological sort for dependency ordering, `gorai.Run()` entrypoint, and the `componentregistry` package for main.go editing -- are correctly implemented and align with the design intent. Below are findings grouped by severity.

---

## Critical Findings

### C-1: robot.go hard-codes V4L2 camera driver (REQ-CADDY-2 / REQ-CORE-2 violation)

**File:** `pkg/robot/robot.go` lines 19, 23, 49, 457-530

`pkg/robot/robot.go` directly imports and uses the V4L2 camera driver:

```go
import (
    "github.com/emergingrobotics/gorai/driver/camera/v4l2"
    hwv4l2 "github.com/emergingrobotics/gorai/pkg/hardware/v4l2"
)
```

The `Robot` struct holds `cameras map[string]*v4l2.Camera` and `startCamera()` constructs V4L2 cameras directly. This means **every binary built with gorai core includes the V4L2 driver**, whether the robot has a camera or not. This violates REQ-CADDY-2 ("Components MUST NOT live in the core gorai/gorai repo except fakes for testing") and defeats the Caddy model for cameras.

Additionally, `Start()` has a special-case branch for `comp.Type == "camera"` (lines 180-196) that bypasses `startRegistryComponent()`. This means camera components cannot participate in the standard registry pattern.

**Fix:** Camera should be a registry-registered component like everything else. Remove `startCamera()`, the V4L2 imports, and the camera special case. V4L2 becomes an external module with init()-based registration.

### C-2: build command missing go.mod check (REQ-CLI-BUILD-3 violation)

**File:** `cmd/gorai/commands/build.go`

The build command does not check for the presence of `go.mod` before running `go build`. REQ-CLI-BUILD-3 explicitly requires:

> If the current directory does not contain a go.mod, gorai build MUST fail with:
> `Error: not in a Go module. Run 'gorai build' from your robot project directory.`

The command will fail with a generic `go build failed` error instead of the actionable error the spec requires.

---

## High Findings

### H-1: GetByType() stub in componentDeps returns nil, nil

**File:** `pkg/robot/deps.go` lines 43-45

```go
func (d *componentDeps) GetByType(subtype string) ([]any, error) {
    return nil, nil
}
```

This is a silent no-op. Any component calling `deps.GetByType("motor")` gets an empty result with no error, making it impossible to distinguish "no motors exist" from "not implemented." The data to implement this exists -- `d.components` contains all created components -- but the method does not filter or return them.

**Fix:** Either implement filtering (would need type metadata stored alongside components) or return an explicit error indicating the method is not yet implemented.

### H-2: Component shutdown ordering is not reverse-dependency-order

**File:** `pkg/robot/robot.go` lines 1036-1047

Components are shut down by iterating `r.components` (a `map[string]any`), which has random iteration order in Go. Per REQ-AUTHOR-4 and dependency injection practice, components should be closed in **reverse** topological order so that dependents release their references before dependencies are closed. A component may attempt to use a dependency that has already been closed.

**Fix:** Store the sorted component order from startup and iterate it in reverse during `Stop()`.

### H-3: Topological sort does not detect self-dependency

**File:** `pkg/robot/topo.go`

A component that lists itself in `depends_on` (e.g., `{name: "a", depends_on: ["a"]}`) is treated as a valid dependency in the lookup phase (line 19-24, `byName[dep]` succeeds since "a" is in the map). It increments its own in-degree to 1, never enters the queue as a zero-in-degree node, and is eventually reported as a cycle. The error message "circular dependency involving: a" is correct but not specific.

More importantly, there is no test for this edge case. There are also no tests for empty input or duplicate component names.

### H-4: `go mod tidy` error silently ignored in component add

**File:** `cmd/gorai/commands/component.go` line 118

```go
goTidy.Run()
```

The return value of `goTidy.Run()` is discarded. If `go mod tidy` fails (e.g., network error, syntax issue in go.mod), the user gets no feedback. Per CLAUDE.md: "Always handle errors explicitly."

**Fix:** At minimum log a warning. The operation has already succeeded (go get + import edit), so a tidy failure should warn but not fail the command.

### H-5: `gorai.Run()` package location differs from REQUIREMENTS.md

**File:** `pkg/gorai/run.go`, `cmd/gorai/main.go`

REQUIREMENTS.md (REQ-CORE-1) specifies the entrypoint at `github.com/emergingrobotics/gorai/cmd/gorai`. The actual implementation places it at `github.com/emergingrobotics/gorai/pkg/gorai`. Two discrepancies:

1. **Module path**: `gorai/gorai` vs `emergingrobotics/gorai` -- Acceptable if the module has not been published to the final path yet.
2. **Package location**: `pkg/gorai/` vs `cmd/gorai/` as the Run() export point. The requirements doc and the `package-dev-approach.md` both show `import "github.com/emergingrobotics/gorai/cmd/gorai"` with `gorai.Run()`. The current structure puts Run() in `pkg/gorai` which imports `cmd/gorai/commands`. This adds an indirection layer and means the template main.go imports a different path than the documentation shows.

**Fix:** Either move `Run()` to `cmd/gorai/` (making the package name match) or update all documentation to reference `pkg/gorai`.

---

## Medium Findings

### M-1: Variable name `reset_count` uses snake_case

**File:** `pkg/robot/robot.go` lines 323, 338, 341, 346

Go convention and CLAUDE.md require camelCase: `resetCount`.

### M-2: Magic number 4222 repeated for NATS default port

**File:** `pkg/robot/robot.go` lines 272, 280, 292, 604

The literal `4222` appears four times. Per CLAUDE.md ("No magic numbers: use named constants"), extract to a constant like `defaultNATSPort`.

### M-3: Magic number 500ms for device reset delay

**File:** `pkg/robot/robot.go` line 345

```go
time.Sleep(500 * time.Millisecond)
```

Unexplained magic number with no comment explaining the rationale.

### M-4: Magic number 100 for frame counter logging interval

**File:** `pkg/robot/robot.go` line 753

```go
if count%100 == 0 {
```

Hardcoded logging interval. Should be a named constant with a comment.

### M-5: `findMainGo()` only checks current directory

**File:** `cmd/gorai/commands/component.go` lines 168-172

Only checks `main.go` in CWD. If the user runs `gorai component add` from a subdirectory, the function silently fails. Consider using `go env GOMOD` to locate the module root, or walking parent directories.

### M-6: Two separate "registry" packages with confusing names

- `pkg/registry/` -- Runtime component registration (init()-time, in-memory constructors)
- `pkg/componentregistry/` -- JSON file catalog for `gorai component search/add/info`

These serve entirely different purposes but share the word "registry." The type `componentregistry.Registry` and function `registry.RegisterComponent()` are easily confused. Consider renaming `pkg/componentregistry/` to `pkg/catalog/` or `pkg/componentcatalog/`.

### M-7: No registry caching or remote fetch (REQ-REGISTRY-2 gap)

**File:** `pkg/componentregistry/registry.go`

REQ-REGISTRY-2 requires local caching with periodic refresh (e.g., every 24 hours or on `--refresh`). The current `LoadFromFile` reads from disk every time. There is no remote fetch, no cache TTL, and no `--refresh` flag.

### M-8: Potential panic on empty arg string in CLI commands

**Files:** `cmd/gorai/commands/build.go` line 43, `cmd/gorai/commands/run.go` line 39

```go
if args[i][0] != '-' {
```

If `args[i]` is an empty string, this panics with index-out-of-range. Use `len(args[i]) > 0 && args[i][0] != '-'` or `strings.HasPrefix(args[i], "-")`.

### M-9: Validate command uses manual path parsing instead of filepath.Dir

**File:** `cmd/gorai/commands/validate.go` lines 116-118

```go
configDir := "."
if strings.Contains(configPath, "/") {
    configDir = configPath[:strings.LastIndex(configPath, "/")]
}
```

This manual parsing does not handle Windows paths and ignores `filepath.Dir()` which handles these cases correctly.

---

## Low Findings

### L-1: `IsRegistered` only checks components, not services

**File:** `pkg/registry/registry.go`

No `IsServiceRegistered` equivalent exists. The asymmetry is minor since validation currently only checks components, but it limits future use.

### L-2: `ListComponents` and `ListServices` return non-deterministic ordering

Model lists within each subtype come from map iteration. Output of `gorai components` varies across runs. Sorting would make it deterministic.

### L-3: Test coverage gap: empty and single-element inputs to topoSort

**File:** `pkg/robot/topo_test.go`

No test for `topoSortComponents(nil)`, `topoSortComponents([]config.ComponentConfig{})`, or a single component with no deps. Also no test for duplicate component names.

### L-4: `configPath` absolutization ignores error

**File:** `cmd/gorai/commands/build.go` line 60

```go
configPath, _ = filepath.Abs(configPath)
```

The error is silently discarded.

### L-5: `defaultRegistryPath` is CWD-relative

**File:** `cmd/gorai/commands/component.go` line 13

`"registry.json"` makes the primary registry path dependent on where the user runs the command. The `~/.gorai/registry.json` fallback mitigates this.

---

## Interface Boundaries Assessment

**Component interfaces in `components/*/`**: Clean. Each subpackage defines an interface (Motor, Servo, Sensor subtypes, etc.) that embeds `component.Component` -> `resource.Resource`. No hardware dependencies in the interface packages. Fake implementations correctly live in `components/*/fake/` and depend only on the interface package + `pkg/registry`. External modules can import `components/motor` without pulling in hardware deps.

**Exception**: `pkg/robot/robot.go` pulling in `driver/camera/v4l2` is the one boundary violation (see C-1).

**Driver interfaces in `driver/`**: Clean. `driver/driver.go` defines base interfaces. Subdirectories define transport-specific interfaces with no platform-specific implementations.

---

## Registry API Assessment

`pkg/registry/registry.go` is clean and usable by external modules. The `Constructor` signature matches REQ-AUTHOR-2:

```go
type Constructor func(ctx context.Context, deps Dependencies, conf Config) (any, error)
```

The `Dependencies` interface is minimal (`Get` + `GetByType`). The `Config` type alias (`map[string]any`) is simple. External modules only need `import "github.com/emergingrobotics/gorai/pkg/registry"` to register and look up components.

The use of `any` return types in both Constructor and Dependencies.Get is intentional (components define local interfaces for their dependencies, per REQ-AUTHOR-4), but means type errors are caught at runtime, not compile time.

Thread safety is correct: all registry operations hold a `sync.RWMutex`.

---

## Dependency Injection Assessment

Topological sort (`pkg/robot/topo.go`) correctly implements Kahn's algorithm:
- No dependencies: all components independent -- works
- Linear chains: A -> B -> C -- works, tested
- Diamond: D -> {B, C} -> A -- works, tested
- Missing deps: error with component name -- works, tested
- Cycles: detected via remaining in-degree count -- works, tested

`pkg/robot/deps.go` correctly accumulates created components for downstream access. The "nats" and "logger" magic strings are documented. The dependency bag is created once per robot startup and shared across all component constructors.

Edge cases not handled or tested:
- Self-dependency (caught as cycle, no specific message)
- Empty input (works correctly, not tested)
- Duplicate component names (last-write-wins in byName map, no error raised)

---

## Run() Entrypoint Assessment

`pkg/gorai/run.go` is clean: 18 lines, delegates to `commands.Execute()`. The package name `gorai` means external callers write `gorai.Run()` which reads naturally. The `cmd/gorai/main.go` is 19 lines (under the 20-line REQ-TEMPLATE-4 requirement) and demonstrates the blank import pattern with remote proxy components.

---

## Component Command Assessment

`cmd/gorai/commands/component.go` correctly implements search/add/info:
- **search**: Loads registry, calls `reg.Search(query)`, formats output
- **add**: Supports both registry name lookup and direct Go module paths, runs `go get`, adds blank import via AST manipulation (`componentregistry.AddBlankImport`), runs `go mod tidy`
- **info**: Looks up by key, prints formatted details

The `AddBlankImport` implementation (`pkg/componentregistry/mainfile.go`) correctly uses `go/parser` + `go/format` for safe AST-level editing, handles existing imports (no duplicates), and creates import blocks when none exist. Tests cover all three cases.

---

## Build Command Assessment

`cmd/gorai/commands/build.go` correctly:
- Parses `--config`, `-o`, `--target` flags
- Loads and validates config before building
- Sets `GOOS`/`GOARCH` for cross-compilation
- Runs `go build .` (the user's module, not the core)
- Sets ldflags for version info
- Contains no docker/podman/container references (REQ-CLI-BUILD-2 satisfied)

Missing: go.mod existence check (C-2 above).

---

## Overall

**Counts:** 2 critical, 5 high, 9 medium, 5 low

The implementation faithfully implements the Caddy model design. The critical issues (V4L2 hardcoding in robot.go, missing go.mod check in build) should be fixed before the template repo is usable. The high issues (shutdown ordering, GetByType stub, ignored errors, Run() path mismatch) represent correctness risks or documentation mismatches that will surface during integration testing with external component modules.
