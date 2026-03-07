# Design/Architecture Review: Configuration Hot-Reload

**Reviewer:** Claude Opus 4.6 (automated)
**Date:** 2026-02-22
**Files Reviewed:**
- `/er/gorai/docs/DESIGN-config-hot-reload.md`
- `/er/gorai/pkg/config/diff.go`
- `/er/gorai/pkg/config/watcher.go`
- `/er/gorai/pkg/robot/robot.go`
- `/er/gorai/pkg/robot/reload_test.go`
- `/er/gorai/pkg/topics/topics.go`
- `/er/gorai/pkg/resource/resource.go`
- `/er/gorai/CLAUDE.md`

---

## Executive Summary

The configuration hot-reload feature is well-architected with a clean separation between diff logic (`pkg/config`), orchestration (`pkg/robot`), and event notification (`pkg/topics`). The watcher/debounce/diff/reconfigure pipeline follows established patterns. Three HIGH findings relate to incomplete structural diff coverage and unprotected config access. Seven MEDIUM findings cover design-spec drift, missing panic recovery, and silent config acceptance. Five LOW findings address efficiency, test gaps, and edge-case concurrency.

---

## HIGH Issues

### HIGH-1: Incomplete NATS structural diff -- missing fields

**Severity:** HIGH
**Location:** `/er/gorai/pkg/config/diff.go`, `natsStructuralDiff()`

The design document (section 4) specifies that these NATS fields are structural: `URL, URLs, JetStream, CredentialsFile, TLS, ConnectTimeout, ReconnectWait, MaxReconnects`. The implementation only checks `URL`, `JetStream`, and `CredentialsFile`. It omits:

- `URLs` (alternate server list)
- `TLS` (entire TLS configuration struct)
- `ConnectTimeout`
- `ReconnectWait`
- `MaxReconnects`

The `NATSConfig` struct defines all these fields:

```go
type NATSConfig struct {
    URL             string     `json:"url,omitempty"`
    URLs            []string   `json:"urls,omitempty"`
    JetStream       bool       `json:"jetstream,omitempty"`
    CredentialsFile string     `json:"credentials_file,omitempty"`
    TLS             *TLSConfig `json:"tls,omitempty"`
    ConnectTimeout  string     `json:"connect_timeout,omitempty"`
    ReconnectWait   string     `json:"reconnect_wait,omitempty"`
    MaxReconnects   int        `json:"max_reconnects,omitempty"`
}
```

A user changing TLS config or connection timeouts would have the change silently accepted as parameter-only and stored as the new `r.cfg`, but the running NATS connection would use the old settings. This creates a divergence between stored config and actual runtime state.

---

### HIGH-2: Incomplete dashboard structural diff -- missing fields

**Severity:** HIGH
**Location:** `/er/gorai/pkg/config/diff.go`, `dashboardStructuralDiff()`

The design document specifies that `dashboard.websocket` and `dashboard.video` are structural. The `DashboardConfig` struct has these fields:

```go
type DashboardConfig struct {
    Enabled   *bool            `json:"enabled,omitempty"`
    Listen    string           `json:"listen,omitempty"`
    WebSocket *WebSocketConfig `json:"websocket,omitempty"`
    Video     *VideoConfig     `json:"video,omitempty"`
}
```

The implementation only checks `Enabled` and `Listen`. Changes to `WebSocket` or `Video` sub-configs would be silently accepted and stored without affecting the running dashboard, creating the same stored-vs-actual divergence as HIGH-1.

---

### HIGH-3: `r.cfg` accessed without `cfgMu` lock in multiple locations

**Severity:** HIGH
**Location:** `/er/gorai/pkg/robot/robot.go`

The `cfgMu` mutex is correctly added to the `Robot` struct and used in `handleConfigReload()` (lines 1019-1021 for read, 1092-1094 for write) and the `Config()` accessor (lines 1124-1127). However, `r.cfg` is read without the lock in many other methods:

- `Start()` (lines 144, 166, 192, 224-228) -- safe because watcher not yet active
- `connectNATS()` (lines 235-236, 241) -- safe, runs before watcher
- `startDashboard()` (lines 259, 264) -- safe, runs before watcher
- `detectHardware()` (line 294) -- safe, runs before watcher
- `getNATSURL()` (lines 513-514) -- **UNSAFE**: called during `startRegistryComponent` and `startInternalService`; if a service constructor is slow, a config reload could fire concurrently
- `startExternalService()` (lines 771, 773-774) -- could race if called from `monitorExternalService` restart path
- `Run()` (line 915) -- reads `r.cfg.Robot.Name` while watcher is running
- `Stop()` (lines 928, 993) -- reads `r.cfg.Robot.Name` during shutdown, which races with `handleConfigReload`

Most pre-watcher accesses are safe in practice, but `getNATSURL()`, `Run()`, and `Stop()` can race with the watcher goroutine.

---

## MEDIUM Issues

### MEDIUM-1: Design document vs implementation mismatch -- index-based vs name-based matching

**Severity:** MEDIUM
**Location:** `/er/gorai/docs/DESIGN-config-hot-reload.md` (section 4), `/er/gorai/pkg/config/diff.go`

The design document explicitly states: "Components and services are matched by index position in the JSON arrays. This is deliberate: reordering components in the file is treated as a structural change because the framework has no way to distinguish 'component A was renamed' from 'component A was replaced' if we match by name alone. Index-based matching is simple and predictable."

The implementation uses name-based matching via maps:

```go
oldByName := make(map[string]ComponentConfig, len(old))
for _, c := range old {
    oldByName[c.Name] = c
}
```

Name-based matching is arguably better (more intuitive for users), but the spec and code disagree. Per project rules in CLAUDE.md: "When code and spec disagree, fix the code." The spec should be updated to match the implementation, or vice versa.

---

### MEDIUM-2: Design document vs implementation mismatch -- NewWatcher signature

**Severity:** MEDIUM
**Location:** `/er/gorai/docs/DESIGN-config-hot-reload.md` (section 5), `/er/gorai/pkg/config/watcher.go`

The design specifies `NewWatcher` returns `*Watcher` (no error):

```go
// Design doc:
func NewWatcher(path string, onChange func(*RDL), logger *slog.Logger) *Watcher

// Implementation:
func NewWatcher(path string, onChange func(*RDL), logger *slog.Logger) (*Watcher, error)
```

The implementation returning an error is correct (it calls `filepath.Abs` which can fail). The design document needs updating.

---

### MEDIUM-3: Debounce timer fires on separate goroutine -- TOCTOU race with context

**Severity:** MEDIUM
**Location:** `/er/gorai/pkg/config/watcher.go`, `run()` and `loadAndNotify()`

`time.AfterFunc` fires the callback on a new goroutine, separate from the `run()` event loop. The `loadAndNotify` method does a non-blocking context check:

```go
func (w *Watcher) loadAndNotify(ctx context.Context) {
    select {
    case <-ctx.Done():
        return
    default:
    }
    // ...
    w.onChange(cfg)
}
```

There is a TOCTOU window: after the context check passes, the context could be cancelled before `w.onChange(cfg)` is called, invoking `handleConfigReload` on a Robot that is shutting down. The design document (section 12) says "handleConfigReload checks r.ctx.Done() before applying" but the actual `handleConfigReload` implementation does not have this check.

---

### MEDIUM-4: Config updated even when all Reconfigure calls fail

**Severity:** MEDIUM
**Location:** `/er/gorai/pkg/robot/robot.go`, `handleConfigReload()`, lines 1091-1094

After iterating through component and service changes, `r.cfg` is unconditionally updated to `newCfg`:

```go
r.cfgMu.Lock()
r.cfg = newCfg
r.cfgMu.Unlock()
```

This happens even if every single `Reconfigure()` call failed. The stored config then reflects the desired state rather than the actual running state. On the next file change, the diff will be computed against the failed config, potentially missing that those components still need reconfiguring.

The design document states "robot logs the error but does not roll back other successful reconfigurations" which implies this is intentional. However, if all reconfigures fail, the entire stored config diverges from actual running state with no mechanism to detect or recover from this.

---

### MEDIUM-5: No panic recovery in handleConfigReload

**Severity:** MEDIUM
**Location:** `/er/gorai/pkg/robot/robot.go`, `handleConfigReload()`

The design document (section 12, Risks) explicitly states: "Component Reconfigure panics -- Recover in the loop; log and continue with other components." The implementation does not include any `recover()` call. A panic in any component's `Reconfigure()` method would crash the entire robot process. This is particularly important for a system intended to run on remote embedded devices (Raspberry Pi, Orange Pi) where physical access for restart may not be available.

---

### MEDIUM-6: `depends_on` changes silently accepted but never applied

**Severity:** MEDIUM
**Location:** `/er/gorai/pkg/config/diff.go`, design doc section 4

The design lists `depends_on` as non-structural ("Dependency graph is only used at startup"). However, `depends_on` changes are not detected by `AttributeDiff` either -- `AttributeDiff` only compares the `Attributes` map, and `depends_on` is a separate struct field:

```go
type ComponentConfig struct {
    Name       string         `json:"name"`
    Type       string         `json:"type"`
    Model      string         `json:"model"`
    Disabled   bool           `json:"disabled,omitempty"`
    Attributes map[string]any `json:"attributes,omitempty"`
    DependsOn  []string       `json:"depends_on,omitempty"`  // Not in Attributes
    // ...
}
```

A change to `depends_on` would be silently accepted and stored in `r.cfg` without any notification, event, or log message. While low-risk (only matters at next startup), it could confuse operators who expect feedback when their config change is processed.

---

### MEDIUM-7: `log_level` per-component changes silently accepted but not applied

**Severity:** MEDIUM
**Location:** `/er/gorai/pkg/config/diff.go`, design doc section 4

The design lists per-component `log_level` as non-structural and "operational." However, like `depends_on`, `log_level` is not part of the `Attributes` map and is not detected by `AttributeDiff`. No `Reconfigure` call is triggered and no mechanism exists to change a running component's log level. The change is stored in `r.cfg` silently. Users may reasonably expect log level changes to take effect immediately since the design calls them "operational."

---

## LOW Issues

### LOW-1: `attributesEqual` uses JSON marshal byte comparison

**Severity:** LOW
**Location:** `/er/gorai/pkg/config/diff.go`, `attributesEqual()`

The function marshals both maps to JSON and compares the byte strings:

```go
func attributesEqual(a, b map[string]any) bool {
    if len(a) == 0 && len(b) == 0 {
        return true
    }
    aJSON, errA := json.Marshal(a)
    bJSON, errB := json.Marshal(b)
    if errA != nil || errB != nil {
        return false
    }
    return string(aJSON) == string(bJSON)
}
```

Go's `json.Marshal` sorts map keys alphabetically, so this is deterministic and correct. However, it allocates two JSON byte slices per comparison. For configs with many components this could be optimized with `reflect.DeepEqual` or a recursive comparison. This is a minor efficiency concern given typical config sizes (under 50 components).

---

### LOW-2: Design document says Start() blocks; implementation returns immediately

**Severity:** LOW
**Location:** `/er/gorai/docs/DESIGN-config-hot-reload.md` (section 5), `/er/gorai/pkg/config/watcher.go`

The design says: "Start begins watching the config file for changes. It blocks until the context is cancelled." The implementation spawns a goroutine via `go w.run(ctx, fsw)` and returns immediately. The non-blocking behavior is correct for integration into `Robot.Start()`. The design document comment is misleading.

---

### LOW-3: Test coverage gaps

**Severity:** LOW
**Location:** `/er/gorai/pkg/robot/reload_test.go`

The test file covers the core scenarios well (8 tests: attribute changes on components and services, structural rejection, non-resource skip, reconfigure error with partial failure, config update verification, no-change detection, multi-resource changes). Missing test cases:

1. **Concurrent reload**: Two rapid config changes overlapping while the first `Reconfigure` is executing. No serialization prevents this.
2. **Context cancellation during reload**: Verifying safe abandonment when the robot shuts down mid-reload.
3. **NATS event publication**: No test verifies `publishConfigReloadEvent` arguments. The test robot has `nats == nil` so publish is a no-op.
4. **`TestReload_NonResourceComponentSkipped`**: Does not assert that the component name is absent from both `updatedComponents` and `failedComponents` in the NATS event.
5. **Missing component in map**: When `AttributeDiff` returns a component name that does not exist in `r.components` (e.g., if the component failed to start initially). This path is tested indirectly through the existing tests but not explicitly.

---

### LOW-4: `componentAdapter.Reconfigure` silently succeeds

**Severity:** LOW
**Location:** `/er/gorai/pkg/robot/robot.go`, lines 580-582

The `componentAdapter` wraps non-Resource components and its `Reconfigure()` returns nil (success). During hot-reload, if a component stored via the adapter receives a config change, the cast at line 1048 (`comp.(resource.Resource)`) would succeed through the adapter, and `Reconfigure` would silently succeed without applying anything. The component would be listed in `updatedComponents` despite nothing changing.

Currently, `componentAdapter` is only used in `serviceDeps.Get()` (line 542), not stored directly in `r.components`, so this path is not reachable. However, this is fragile if the adapter is ever stored in the components map.

---

### LOW-5: No serialization of reload operations

**Severity:** LOW
**Location:** `/er/gorai/pkg/robot/robot.go`, `handleConfigReload()`

There is no mutex or channel ensuring only one `handleConfigReload` executes at a time. While the 500ms debounce makes concurrent invocations unlikely, it is theoretically possible: the `AfterFunc` timer fires on a new goroutine, and if `handleConfigReload` takes longer than 500ms (slow `Reconfigure` calls), a second debounce timer could fire and start a second concurrent reload. Two concurrent reloads could interleave `Reconfigure` calls and produce inconsistent `r.cfg` updates.

---

## Interface Correctness and Data Flow

The data flow matches the design document (section 3) with minor deviations:

```
fsnotify event -> debounce (500ms) -> config.Load() + Validate() ->
StructuralDiff() -> AttributeDiff() -> Reconfigure() per changed resource ->
update r.cfg -> publish NATS event
```

Key correctness observations:

- **`resource.Resource` interface**: Well-designed with `Reconfigure(ctx, deps, conf)` providing all necessary context. The `Dependencies` interface allows services to resolve components by name during reconfiguration.
- **`resource.Config`**: `NewConfig()` correctly wraps the attributes map with JSON serialization for `Unmarshal()` support.
- **`config.AttributeChange`**: Clean struct carrying name + new attributes. Only attributes are passed, not structural fields -- matching the Reconfigure contract.
- **`topics.ConfigReloadEvent`**: Includes both updated and failed lists, plus rejection reason. Well-structured for operator observability.
- **`topics.SystemConfigReloaded()`**: Correctly integrated into the topic builder pattern.

---

## Concurrency Safety Assessment

| Component | Lock | Assessment |
|-----------|------|-----------|
| `r.cfg` read in `handleConfigReload` | `cfgMu.RLock` | Correct |
| `r.cfg` write in `handleConfigReload` | `cfgMu.Lock` | Correct |
| `r.cfg` read in `Config()` accessor | `cfgMu.RLock` | Correct |
| `r.cfg` read in `Start()`, `Stop()`, `Run()`, `getNATSURL()` | **No lock** | See HIGH-3 |
| `r.components` lookup in reload | `componentsMu.RLock` | Correct; lock released before `Reconfigure` call |
| `r.services` lookup in reload | `servicesMu.RLock` | Correct; lock released before `Reconfigure` call |
| Watcher `run()` goroutine | Single goroutine | Safe; event loop is single-threaded |
| `loadAndNotify` via `AfterFunc` | Separate goroutine per timer | See MEDIUM-3; no serialization with `run()` |
| `mockReconfigurable` in tests | `sync.Mutex` on `reconfigureCalls` | Correct for test assertions |

---

## Deployment Target Compatibility

| Target | Compatible | Notes |
|--------|-----------|-------|
| Raspberry Pi (Linux/arm) | Yes | fsnotify uses inotify on Linux |
| Orange Pi (Linux/arm64) | Yes | Same inotify support |
| Cloud server (Linux/amd64) | Yes | Standard inotify |
| K8s pod (Linux/amd64) | Yes | Local filesystem (emptyDir, hostPath, PVC) works; ConfigMap volumes may not trigger fsnotify reliably |
| NFS/network filesystem | No | Design document correctly documents this limitation |
| macOS (development) | Yes | fsnotify uses kqueue |

No platform-specific issues identified. The `fsnotify` dependency (v1.8+) is well-maintained and supports all target platforms.

---

## Compliance with CLAUDE.md Code Quality Rules

| Rule | Status | Notes |
|------|--------|-------|
| Meaningful names | PASS | `StructuralDiff`, `AttributeDiff`, `handleConfigReload`, `loadAndNotify` are clear |
| No abbreviations | PASS | No abbreviated names in new code |
| No magic numbers | PASS | `debounceDelay` is a named constant |
| Functions do one thing | PASS | Each diff function checks one structural category; `handleConfigReload` orchestrates the full flow |
| Handle errors explicitly | PARTIAL | Reconfigure errors logged but config still updated (MEDIUM-4); panic not recovered (MEDIUM-5) |
| DRY | PASS | `componentStructuralDiff` and `serviceStructuralDiff` share the same pattern but the types differ, making extraction non-trivial without generics |
| KISS | PASS | Minimal complexity for the task |
| Comments explain WHY | PASS | Package-level and function-level comments explain purpose |
| No commented-out code | PASS | Clean |
| No emojis in code | PASS | Clean |
| Interfaces small and composable | PASS | `resource.Resource` is 4 methods; `resource.Dependencies` is 3 methods |

---

## Summary

| Severity | Count | Key Themes |
|----------|-------|------------|
| CRITICAL | 0 | -- |
| HIGH | 3 | Incomplete structural diff checks (NATS, dashboard), unprotected `r.cfg` access |
| MEDIUM | 7 | Design-vs-code mismatches (matching strategy, signatures), missing panic recovery, silent config acceptance for non-attribute fields |
| LOW | 5 | JSON marshal efficiency, test gaps, adapter fragility, missing reload serialization |

### Recommended Priority Fixes

1. **HIGH (before shipping):**
   - HIGH-1: Add missing NATS structural diff checks (URLs, TLS, ConnectTimeout, ReconnectWait, MaxReconnects)
   - HIGH-2: Add missing dashboard structural diff checks (WebSocket, Video)
   - HIGH-3: Audit and add `cfgMu` locking to `getNATSURL()`, `Stop()`, `Run()`, or copy needed values before watcher starts

2. **MEDIUM (soon after):**
   - MEDIUM-1: Update design doc section 4 to reflect name-based matching
   - MEDIUM-2: Update design doc section 5 for `(*Watcher, error)` return
   - MEDIUM-3: Add `r.ctx.Done()` check at top of `handleConfigReload` as documented in design
   - MEDIUM-5: Add `defer recover()` in the Reconfigure loop as specified in design doc section 12

3. **LOW (backlog):**
   - LOW-3: Add concurrent reload and NATS event tests
   - LOW-5: Add a reload mutex to serialize `handleConfigReload` invocations

**Total: 0 critical, 3 high, 7 medium, 5 low findings.**
