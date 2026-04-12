# Design/Architecture Review

**Date:** 2026-04-11
**Reviewer:** Claude Opus 4.6 (automated)
**Scope:** Process-compose runtime (embedded NATS, health server, compose compiler, CLI commands, robot lifecycle)
**Design doc:** `/gorai/docs/DESIGN.md`

---

## Critical

None.

---

## High

### H1. Design doc specifies `embeddednats.Start(ctx context.Context) error` but implementation uses `Start() error`

**File:** `pkg/embeddednats/server.go:99`
**Design ref:** Section 3.1, line ~139

The design specifies `Start(ctx context.Context) error` so callers can pass a cancellation context. The implementation creates its own internal context with a hardcoded 10-second deadline. This means the caller cannot cancel a slow start or propagate a parent context timeout. The `Shutdown` method also takes no context (design specifies `Shutdown(ctx context.Context) error`), so there is no way to bound shutdown time from the caller.

**Recommendation:** Accept a `context.Context` parameter in both `Start` and `Shutdown` to match the design and allow callers to control timeouts.

### H2. Design doc specifies functional options pattern for `embeddednats.New` and `health.New`; implementation uses config structs

**File:** `pkg/embeddednats/server.go:51`, `pkg/health/server.go:40`
**Design ref:** Sections 3.1 and 3.2

The design shows `New(cfg Config, opts ...Option)` with `WithLogger` option. The implementation puts `Logger` directly in the `Config` struct and has no `Option` type. This is arguably simpler (KISS), but diverges from the design. The rest of the codebase (`pkg/robot`) uses the options pattern. Inconsistency makes the API surface harder to learn.

**Recommendation:** Either update the design to match the implementation (config struct with Logger) or add the options pattern. Given KISS preference and that the config approach works, updating the design doc is the pragmatic fix.

### H3. `health.Server` handlers ignore `json.Encode` errors

**File:** `pkg/health/server.go:124,128,133`

The `json.NewEncoder(writer).Encode(...)` return value is discarded. While encoding a small static struct is extremely unlikely to fail, the CLAUDE.md code quality rules say "always handle errors explicitly." If the write to the response writer fails (e.g., client disconnected), the error is silently dropped.

**Recommendation:** Log the error at debug level, or assign and check:
```go
if err := json.NewEncoder(writer).Encode(...); err != nil {
    server.logger.Debug("failed to write health response", "error", err)
}
```

### H4. Design specifies `ConfigFromRDL` and `DetectNATSMode` functions; neither exists

**File:** `pkg/embeddednats/server.go` (missing), `pkg/compose/compiler.go` (missing)
**Design ref:** Sections 4.1, 4.3

The design specifies `embeddednats.ConfigFromRDL()` to encapsulate NATS mode selection as "the single place where the decision tree is evaluated." Instead, this logic is split between `config.RDL.ShouldEmbedNATS()` and `compose.Compiler.shouldEmitExternalNATS()`. Similarly, `DetectNATSMode()` with a `NATSMode` enum type does not exist; the logic is inline.

The current implementation works correctly and the logic is not duplicated in a dangerous way, but the design's goal of a single decision point is not achieved -- mode selection is evaluated in two different packages.

**Recommendation:** Either implement the design's `ConfigFromRDL`/`DetectNATSMode` pattern or update the design to reflect the current split. The current split is reasonable since `config` and `compose` have different concerns.

---

## Medium

### M1. `compose/compiler.go` -- file layout diverges from design

**Design ref:** Section 3.3 file table

The design specifies 7 files (`compiler.go`, `model.go`, `nats.go`, `services.go`, `devices.go`, `infra.go`, `dependencies.go`). The implementation puts everything in `compiler.go` (~625 lines) with types in `types.go` (not `model.go`). The single-file approach works but makes the file large and harder to navigate. As more features are added, this will become unwieldy.

**Recommendation:** Consider splitting per the design in a future refactor. At minimum, rename `types.go` to `model.go` to match the design, or update the design.

### M2. `extractNATSPort` function is dead code

**File:** `pkg/compose/compiler.go:614-624`

The function `extractNATSPort` is defined but never called. Dead code should be removed per CLAUDE.md rules ("Do not comment out code -- remove it" applies to dead code too).

**Recommendation:** Remove `extractNATSPort`.

### M3. `reset_count` variable uses snake_case instead of camelCase

**File:** `pkg/robot/robot.go:387,401,404`

```go
reset_count := 0
```

Go convention requires `resetCount`. CLAUDE.md says "Go: camelCase exported/unexported."

**Recommendation:** Rename to `resetCount`.

### M4. `device.go` -- `cmdDeviceReset` publishes `nil` payload instead of `{"subsystem":0}` per design

**File:** `cmd/gorai/commands/device.go:91`
**Design ref:** Section 4.4

The design specifies publishing `{"subsystem":0}` to the reset topic. The CLI implementation publishes `nil`. The `robot.go` implementation at line 394 correctly publishes `[]byte(`{"subsystem":0}")`. This inconsistency means device reset behaves differently depending on whether it's triggered by the CLI or the robot lifecycle.

**Recommendation:** Change `device.go:91` from `client.Publish(topic, nil)` to `client.Publish(topic, []byte(`{"subsystem":0}"))`.

### M5. `run.go` -- `healthListen` variable is suppressed with `_ = healthListen`

**File:** `cmd/gorai/commands/run.go:109`

The `--health-listen` flag is parsed but then explicitly suppressed. The design says the health server should be created in `cmdRun()` with this address. However, the health server is actually created inside `robot.Start()` using a different default. The flag is effectively dead code.

**Recommendation:** Pass `healthListen` to the robot via `robot.WithHealthListen(healthListen)` when creating the robot instance (the option already exists at `robot.go:119`). Remove the `_ = healthListen` suppression.

### M6. Design specifies health server created in `cmdRun()` before `robot.Start()`; implementation creates it inside `robot.Start()`

**File:** `cmd/gorai/commands/run.go`, `pkg/robot/robot.go:167`
**Design ref:** Section 6.2

The design says the health server should be created in `cmdRun()` and readiness set after `robot.Start()` succeeds. The implementation creates the health server inside `robot.Start()`. This means the health server's lifecycle is coupled to the robot, not the CLI command. Both approaches work, but the implementation diverges from the design. The robot-internal approach is arguably cleaner since it keeps lifecycle management in one place.

**Recommendation:** Update the design to reflect the current implementation.

### M7. `compose/compiler.go` -- magic number 64 * 1024 * 1024 for MaxPending

**File:** `pkg/embeddednats/server.go:70`

```go
MaxPending: 64 * 1024 * 1024,
```

This is a magic number. CLAUDE.md says "No magic numbers: use named constants."

**Recommendation:** Extract to a named constant:
```go
const defaultMaxPending = 64 * 1024 * 1024 // 64 MiB
```

### M8. `run.go` -- `runCompose` leaks temp file on successful exec

**File:** `cmd/gorai/commands/run.go:197-215`

When `syscall.Exec` succeeds, the temp file is never deleted because the current process is replaced. This leaves orphaned YAML files in the temp directory. On failure, the file is also not cleaned up.

**Recommendation:** This is inherent to the `syscall.Exec` approach. Document this as a known limitation, or use a well-known path (e.g., `.gorai/process-compose.yaml`) that gets overwritten each time instead of a temp file.

### M9. Thread safety: `Robot.underProcessCompose` is written in `Start()` and could be read concurrently

**File:** `pkg/robot/robot.go:161`

`underProcessCompose` is a plain `bool` set during `Start()`. While in practice it's only read during the same `Start()` call, it lacks synchronization. Other fields like `ready` in health use `atomic.Bool`.

**Recommendation:** Use `atomic.Bool` or document that this field is only accessed during sequential startup.

---

## Low

### L1. `health.Server` default port 4180 is hardcoded in three places

**Files:** `pkg/health/server.go:15`, `pkg/robot/robot.go:278`, `cmd/gorai/commands/run.go:34`

The default health listen address `127.0.0.1:4180` appears in three separate locations. DRY principle suggests a single exported constant.

**Recommendation:** Export `health.DefaultListenAddress` and reference it from `robot.go` and `run.go`.

### L2. `compile.go` argument parsing -- potential panic on empty flag

**File:** `cmd/gorai/commands/compile.go:33`

```go
if args[i][0] != '-' {
```

If `args[i]` is an empty string, this panics with an index-out-of-range. While unlikely from CLI input, it's not defensive.

**Recommendation:** Check `len(args[i]) > 0` first, or use `strings.HasPrefix`.

### L3. `run.go` has the same potential panic on empty arg

**File:** `cmd/gorai/commands/run.go:63`

Same pattern as L2.

### L4. `compose/types.go` vs `compose/model.go` naming

**File:** `pkg/compose/types.go`

The design calls the file `model.go`. The implementation names it `types.go`. Both are valid Go conventions, but diverging from the design without documenting the change adds confusion.

**Recommendation:** Either rename to `model.go` or update the design.

### L5. `WaitReady` busy-polls on context cancellation

**File:** `pkg/embeddednats/server.go:119-130`

The `WaitReady` loop calls `ReadyForConnections(readyPollInterval)` which blocks for 50ms, then checks `ctx.Done()`. This is a reasonable approach, but the `select` with `default` means if `ReadyForConnections` returns false, it immediately re-enters without respecting the poll interval on the context-check path. This is fine because `ReadyForConnections` itself sleeps for the poll interval, but the code structure is slightly misleading.

### L6. Test coverage: no test for TLS configuration path in embeddednats

**File:** `pkg/embeddednats/server_test.go`

No test exercises the TLS code path (`config.TLS != nil`). While TLS testing requires certificates, the lack of coverage means TLS configuration bugs would not be caught.

### L7. `compose/compiler.go` container command building does not shell-escape values

**File:** `pkg/compose/compiler.go:317-350`

Environment variable values, volume paths, and device paths are concatenated into the podman command without shell escaping. Values containing spaces, quotes, or special characters would break the command. This is acceptable for controlled RDL input but fragile.

**Recommendation:** Note as a known limitation or add shell quoting for values.

---

## Summary

The implementation is well-structured and functionally correct. The major deviations from the design are primarily API-level (config struct vs options pattern, method signatures missing context parameters) rather than architectural. The core architecture (embedded NATS, health probes, compose compiler, CLI commands, robot lifecycle integration) matches the design's intent. Thread safety is generally good with proper mutex usage. The code is clean and readable with meaningful names and explicit error handling throughout.

**Counts:** 0 critical, 4 high, 9 medium, 7 low
