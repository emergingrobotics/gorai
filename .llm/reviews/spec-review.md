# Spec Compliance Review
Date: 2026-04-12

Scope: Phase 1 requirements from REQUIREMENTS.md v2.0 -- sections 3 (Caddy Model), 5 (Core Repo Changes), 6 (CLI Commands), 8 (Component Authoring Contract), and 9 (Dependency Injection).

## Passed

- REQ-CADDY-1: Blank imports are the component manifest. -- `cmd/gorai/main.go` uses blank imports (lines 8-14) as the sole mechanism for declaring components. No separate manifest file exists.

- REQ-CADDY-3: Self-registration via init(). -- `pkg/registry/registry.go` exports `RegisterComponent()` and `RegisterService()`. All component packages (fake, remote) call these in `init()` functions.

- REQ-CADDY-4: gorai.Run() entrypoint. -- `pkg/gorai/run.go` exports `Run()` which calls `commands.Execute()` and exits with code 1 on error. Matches the spec code example exactly.

- REQ-CADDY-5: go build is the build system. -- `cmd/gorai/commands/build.go` runs `exec.Command("go", buildArgs...)` with GOOS/GOARCH for cross-compilation. No container tooling.

- REQ-CADDY-6: Component interfaces stay in core. -- `components/servo/servo.go`, `components/motor/motor.go`, etc. are interface-only packages. `pkg/registry/registry.go` defines `Constructor`, `Dependencies`, `Config` types for external callers.

- REQ-CORE-1: Export a Run() function. -- `pkg/gorai/run.go` exports `Run()`. CLI dispatches validate, run, build, component, version, help via `commands.Execute()` in `root.go`.

- REQ-CORE-2: Clean component interfaces. -- Component interface packages contain only interfaces and types. Fakes in `*/fake/` sub-packages. No hardware implementations in interface packages.

- REQ-CORE-4: Registry validates at startup. -- `robot.go` `startRegistryComponent()` (line 543) calls `registry.LookupComponent()` and on failure returns: "type %q model %q not found in registry; this component may need to be installed -- try: gorai component search %s". Actionable error with search suggestion.

- REQ-CORE-5: Remove container build logic. -- `build.go` contains zero references to docker, podman, or containers. `go build .` is the only build mechanism.

- REQ-CLI-BUILD-1: gorai build = go build. -- Loads/validates RDL, runs `go build -o <output> .` with GOOS/GOARCH from `--target`. Supports `-o` for output path. Defaults to robot name from config.

- REQ-CLI-BUILD-2: No container tooling in build. -- Confirmed no docker/podman invocations anywhere in `build.go`.

- REQ-CLI-SEARCH-1: Search the registry. -- `cmdComponentSearch()` loads registry JSON, calls `reg.Search(query)` which matches against name, type, model, description, and tags (case-insensitive). Prints results with name, description, module path, and version.

- REQ-CLI-ADD-1: go get + blank import. -- `cmdComponentAdd()` looks up registry, runs `go get <module>@latest`, calls `AddBlankImport()` to edit main.go, then runs `go mod tidy`.

- REQ-CLI-ADD-2: Edit main.go safely. -- `pkg/componentregistry/mainfile.go` `AddBlankImport()` uses Go AST parser to find/create grouped import declaration, adds blank import spec, reformats with `go/format`.

- REQ-CLI-ADD-3: Support direct module paths. -- `cmdComponentAdd()` line 67: if arg contains a dot, treated as direct Go module path bypassing registry. Supports `@version` suffix parsing.

- REQ-CLI-INFO-1: Show component details. -- `cmdComponentInfo()` displays type/model, description, module, version, hardware, tags, and installation command.

- REQ-DI-1: Topological sort. -- `pkg/robot/topo.go` implements Kahn's algorithm. Detects missing dependencies (line 21) and circular dependencies (line 59-67, "circular dependency involving: ...").

- REQ-DI-2: deps.Get() returns the component instance. -- `componentDeps.Get(name)` in `deps.go` returns the `any` value from the components map. Consuming components type-assert to needed interface.

- REQ-DI-3: NATS and logger always available. -- `componentDeps.Get()` has hardcoded switch cases for "nats" (returns `*nats.Conn`) and "logger" (returns `*slog.Logger`), available to all components regardless of `depends_on`.

- REQ-DI-4: Missing dependency is a startup error. -- `topo.go` line 20-23 checks every `depends_on` name exists and returns error. `config.go` line 631-633 validates deps in `Validate()`. When a component fails to start, `robot.go` returns immediately (lines 184-195), aborting all subsequent component creation.

- REQ-AUTHOR-2: Standard constructor signature. -- `registry.Constructor` is `func(ctx context.Context, deps Dependencies, conf Config) (any, error)` with `Config = map[string]any` and `Dependencies` interface with `Get(name)`. Matches spec exactly.

- REQ-REGISTRY-3: Registry is optional. -- `cmdComponentAdd()` works with direct Go module paths (containing a dot) without consulting the registry.

## Issues

- REQ-CLI-BUILD-3: Error if not in a Go module. -- FIXED. `build.go` now checks `os.Stat("go.mod")` before running `go build` and returns: `not in a Go module. Run 'gorai build' from your robot project directory. See: https://gorai.dev/docs/getting-started`.

- REQ-CLI-VALIDATE-1: Registry-aware validation (partial). -- `validate.go` lines 82-102 check all component type+model pairs and suggest `gorai component search`. However, missing components produce only a WARNING to stderr; the command still exits with code 0 reporting the config as valid. The requirement says missing components "MUST produce an actionable error." The check should either return a non-zero exit code or at minimum clearly indicate validation failure.

- REQ-CADDY-2: Components are standalone Go modules (partial, Phase 1 scope). -- `components/camera/v4l2/` remains in the core repo, registering as `("camera", "v4l2", New)`. Section 5.1 explicitly lists `components/camera/v4l2` as needing to move to an external module. PARTIALLY FIXED: `robot.go` no longer directly imports `driver/camera/v4l2` or `pkg/hardware/v4l2` — the hardware-specific camera detection, creation, and frame publishing code has been removed from robot.go. The `components/camera/v4l2/` package still exists in the repo but is no longer a hard dependency of the runtime. Moving it to an external module is Phase 3 work.

- REQ-REGISTRY-1: Static JSON file (partial). -- Registry loads from local JSON files only (`registry.json` in CWD or `~/.gorai/registry.json`). There is no HTTP fetch from a GitHub raw URL. The requirement says "a JSON file in a GitHub repo, fetchable via raw URL." While local loading works, remote fetch is absent.

- REQ-REGISTRY-2: Local caching. -- Not implemented. No caching mechanism, no TTL refresh, no `--refresh` flag. The `loadRegistry()` function only checks local paths. The requirement says search "MUST cache the registry locally and refresh periodically (every 24 hours or on explicit --refresh)."

## Not Applicable (deferred/out of scope)

- REQ-CADDY-2 (external module creation): Creating actual external module repos (gorai-picarx, gorai-driver-hcsr04, etc.) is Phase 3 work. Only the core cleanup is Phase 1.

- REQ-TEMPLATE-1 through REQ-TEMPLATE-4: Template repo creation is Phase 2 work.

- REQ-CORE-3: Clean driver interfaces. -- Driver interface definitions in `driver/` directory. Platform-specific implementations under `driver/*/linux/` are permitted per spec. Not specifically audited in this review.

- REQ-AUTHOR-1: init() registration is mandatory. -- Contract for external module authors, not a core repo enforcement mechanism. Core fakes and remotes follow the pattern.

- REQ-AUTHOR-3: Implement the correct interface. -- Contract for external authors. Runtime type-assertion is the enforcement mechanism.

- REQ-AUTHOR-4: Accept interfaces, not concrete types. -- Design guidance for external authors. Not enforceable via core code review.

- REQ-RDL-1: RDL format unchanged. -- Config format was not modified; existing robot.json files continue to parse correctly.

- REQ-EXTERNAL-SVC-1 through REQ-EXTERNAL-SVC-3: Non-Go external services are Phase 2+ per section 13. Runtime support exists in `robot.go` but is pre-existing.

- REQ-REGISTRY-1 / REQ-REGISTRY-2 (full implementation): Registry infrastructure (GitHub repo, HTTP fetch, caching) is Phase 4 per the implementation phases section. Phase 1 only needs the core-side plumbing, which exists.
