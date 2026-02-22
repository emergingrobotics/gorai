# Design: Configuration Hot-Reload

**Status:** Implemented
**Date:** 2026-02-22

## 1. Overview

The robot runtime watches `robot.json` for filesystem changes, validates the new configuration, classifies changes as structural or parameter-only, and applies parameter-only changes to running components and services via `Reconfigure()`. Structural changes are rejected without partial application.

This enables operators to adjust schedule offsets, poll intervals, brightness defaults, and similar parameters without restarting the robot process.

## 2. Architecture

### New Files

| File | Purpose |
|------|---------|
| `pkg/config/watcher.go` | Filesystem watcher with fsnotify, 500ms debounce, context-aware lifecycle |
| `pkg/config/diff.go` | Structural diff and attribute diff between two `*RDL` configs |

### Modified Files

| File | Changes |
|------|---------|
| `pkg/robot/robot.go` | Start watcher in `Start()`, handle reload events, call `Reconfigure()` on affected components/services |
| `pkg/topics/topics.go` | Add `ConfigReloaded` system category constant and `SystemConfigReloaded()` builder method |

## 3. Data Flow

```
fsnotify write/rename event
    |
    v
debounce (500ms timer, reset on each event)
    |
    v
config.Load(path) -- parse and validate
    |
    |-- FAIL --> log error, discard, keep running config
    |
    v
structuralDiff(oldCfg, newCfg)
    |
    |-- structural change --> log rejection with reason, publish rejection event, discard
    |
    v
attributeDiff(oldCfg, newCfg) --> []ChangedComponent, []ChangedService
    |
    |-- no changes --> log "config unchanged", return
    |
    v
for each changed component:
    r.components[name].(resource.Resource).Reconfigure(ctx, deps, newConf)
    |
for each changed service:
    r.services[name].(resource.Resource).Reconfigure(ctx, deps, newConf)
    |
    v
update r.cfg to new config
    |
    v
publish config_reloaded success event to NATS
```

## 4. Structural Diff Rules

`structuralDiff(old, new *config.RDL) (string, bool)` returns a reason string and `true` if a structural change is detected. Checks are evaluated in order; the first mismatch produces the rejection reason.

| Field | Rule |
|-------|------|
| `robot.name` | Must match exactly |
| `robot.namespace` | Must match exactly |
| `nats` | URL, URLs, JetStream, CredentialsFile, TLS, ConnectTimeout, ReconnectWait, MaxReconnects must all match |
| Component count | `len(old.Components)` must equal `len(new.Components)` |
| Component identity | For each index: `name`, `type`, `model`, `disabled` must match |
| Service count | `len(old.Services)` must equal `len(new.Services)` |
| Service identity | For each index: `name`, `type`, `model`, `disabled` must match |
| Dashboard config | `dashboard.enabled`, `dashboard.listen`, `dashboard.websocket`, `dashboard.video` must match |

The following fields are **not** structural and can change freely:

| Field | Rationale |
|-------|-----------|
| `log.level`, `log.format`, `log.output` | Log configuration changes are low-risk |
| `robot.description` | Cosmetic field |
| Component/service `attributes` maps | These are the target of hot-reload |
| Component/service `log_level` | Per-resource log level is operational |
| Component/service `depends_on` | Dependency graph is only used at startup |

### Identity Matching

Components and services are matched by name. The component/service count must remain the same, and every name in the new config must exist in the old config. Reordering components in the JSON file is not treated as a structural change — only adding, removing, or renaming components/services is structural. Name-based matching is more intuitive for operators editing config files by hand.

## 5. Watcher Design (`pkg/config/watcher.go`)

### Type

```go
type Watcher struct {
    path     string
    onChange func(newCfg *config.RDL)
    logger   *slog.Logger
}
```

### Constructor

```go
func NewWatcher(path string, onChange func(*RDL), logger *slog.Logger) (*Watcher, error)
```

### Lifecycle

```go
func (w *Watcher) Start(ctx context.Context) error
```

- Creates an `fsnotify.Watcher` watching the directory containing the config file (not the file itself, to handle editor rename patterns).
- Filters events to only process those affecting the target filename.
- On `Write`, `Create`, or `Rename` events for the target file:
  - Reset a 500ms debounce timer.
  - When the timer fires: call `config.Load(path)`. On success, invoke `onChange(newCfg)`. On failure, log the error and continue watching.
- On context cancellation: close the fsnotify watcher, stop the debounce timer, return.

### Editor Compatibility

Text editors commonly save files by writing to a temporary file then renaming it over the target. Watching the directory (not the file) ensures we see the `Rename` event that delivers the new content. The debounce timer coalesces the `Rename` + `Create` events that this pattern produces.

### Error Handling

- If the watched directory is deleted or becomes inaccessible, log an error and stop watching. The robot continues running with the current config.
- If `config.Load()` returns an error, log it and discard the change. The watcher continues monitoring for the next change.

## 6. Diff Logic (`pkg/config/diff.go`)

### Exported Functions

```go
// StructuralDiff returns a human-readable reason and true if the configs
// differ structurally. Returns ("", false) if structurally identical.
func StructuralDiff(old, new *RDL) (string, bool)

// AttributeDiff returns lists of component and service names whose
// attributes maps differ between old and new configs. Only called
// after StructuralDiff confirms no structural changes.
func AttributeDiff(old, new *RDL) (components []AttributeChange, services []AttributeChange)
```

### AttributeChange Type

```go
type AttributeChange struct {
    Name          string
    NewAttributes map[string]any
}
```

### Attribute Comparison

Attributes are compared by JSON-marshaling both maps and comparing the resulting bytes. This avoids false positives from Go map ordering and correctly handles nested objects (like Kauf color defaults or light-controller schedule parameters).

## 7. Reconfigure Flow in `robot.go`

### Integration Point

In `Robot.Start()`, after all components and services are initialized:

```go
if r.configPath != "" {
    watcher := config.NewWatcher(r.configPath, r.handleConfigReload, r.logger)
    if err := watcher.Start(r.ctx); err != nil {
        r.logger.Warn("Failed to start config watcher", "error", err)
    }
}
```

### Handler Method

```go
func (r *Robot) handleConfigReload(newCfg *config.RDL) {
    // 1. Structural diff
    if reason, changed := config.StructuralDiff(r.cfg, newCfg); changed {
        r.logger.Error("config reload rejected: structural change detected",
            "reason", reason)
        r.publishConfigReloadEvent(nil, nil, true, reason)
        return
    }

    // 2. Attribute diff
    compChanges, svcChanges := config.AttributeDiff(r.cfg, newCfg)
    if len(compChanges) == 0 && len(svcChanges) == 0 {
        r.logger.Info("config reload: no attribute changes detected")
        return
    }

    // 3. Reconfigure components
    var updatedComponents []string
    for _, change := range compChanges {
        r.componentsMu.RLock()
        comp, ok := r.components[change.Name]
        r.componentsMu.RUnlock()
        if !ok {
            r.logger.Warn("config reload: component not found", "name", change.Name)
            continue
        }
        if res, ok := comp.(resource.Resource); ok {
            conf := resource.NewConfig(change.NewAttributes)
            deps := &serviceDeps{robot: r}
            if err := res.Reconfigure(r.ctx, deps, conf); err != nil {
                r.logger.Error("config reload: component reconfigure failed",
                    "name", change.Name, "error", err)
                continue
            }
            r.logger.Info("config reload: component reconfigured", "name", change.Name)
            updatedComponents = append(updatedComponents, change.Name)
        }
    }

    // 4. Reconfigure services
    var updatedServices []string
    for _, change := range svcChanges {
        r.servicesMu.RLock()
        svc, ok := r.services[change.Name]
        r.servicesMu.RUnlock()
        if !ok {
            r.logger.Warn("config reload: service not found", "name", change.Name)
            continue
        }
        if res, ok := svc.(resource.Resource); ok {
            conf := resource.NewConfig(change.NewAttributes)
            deps := &serviceDeps{robot: r}
            if err := res.Reconfigure(r.ctx, deps, conf); err != nil {
                r.logger.Error("config reload: service reconfigure failed",
                    "name", change.Name, "error", err)
                continue
            }
            r.logger.Info("config reload: service reconfigured", "name", change.Name)
            updatedServices = append(updatedServices, change.Name)
        }
    }

    // 5. Update stored config
    r.cfg = newCfg

    // 6. Publish success event
    r.publishConfigReloadEvent(updatedComponents, updatedServices, false, "")
}
```

### Concurrency

- `handleConfigReload` is called from the watcher goroutine. It acquires `componentsMu` and `servicesMu` with read locks to look up running instances, then calls `Reconfigure()` outside the lock. Each component/service is responsible for its own internal locking during `Reconfigure()`.
- The `r.cfg` field is only written from this handler and read from other goroutines. A mutex (`cfgMu sync.RWMutex`) should be added to `Robot` to protect `r.cfg` access.

## 8. NATS Events

### Topic

```
gorai.<robot_name>.system.config_reloaded
```

### Success Event

Published after applying parameter changes:

```json
{
  "timestamp": "2026-02-22T10:30:00Z",
  "rejected": false,
  "updated_components": ["pool_light_a", "front_porch"],
  "updated_services": ["light-controller"],
  "failed_components": [],
  "failed_services": []
}
```

### Rejection Event

Published when a structural change is detected:

```json
{
  "timestamp": "2026-02-22T10:30:00Z",
  "rejected": true,
  "reason": "structural change: component count changed from 3 to 4"
}
```

### Event Type Definition (in `pkg/topics/topics.go`)

```go
const ConfigReloaded = "config_reloaded"

func (b *Builder) SystemConfigReloaded() string {
    return b.System(ConfigReloaded)
}
```

### Event Struct (in `pkg/robot/robot.go` or `pkg/topics/topics.go`)

```go
type ConfigReloadEvent struct {
    Timestamp         string   `json:"timestamp"`
    Rejected          bool     `json:"rejected"`
    Reason            string   `json:"reason,omitempty"`
    UpdatedComponents []string `json:"updated_components,omitempty"`
    UpdatedServices   []string `json:"updated_services,omitempty"`
    FailedComponents  []string `json:"failed_components,omitempty"`
    FailedServices    []string `json:"failed_services,omitempty"`
}
```

## 9. Component/Service Reconfigure Contract

Every component and service that supports hot-reload implements `resource.Resource.Reconfigure()` with these invariants:

| Property | Requirement |
|----------|-------------|
| **Idempotent** | Calling with identical config is a no-op |
| **Non-destructive** | Does not stop polling, drop connections, or lose state |
| **Thread-safe** | Uses appropriate locking; may be called concurrently with polling goroutines |
| **Error reporting** | Returns error if new attributes are invalid; robot logs the error but does not roll back other successful reconfigurations |

### What Reconfigure Receives

The `conf resource.Config` argument contains only the `attributes` map from the RDL component/service entry. It does not include `name`, `type`, `model`, or `disabled` -- those are structural and guaranteed unchanged.

The `deps resource.Dependencies` argument provides access to the component registry, enabling services like the light controller to resolve component references by name.

## 10. Testing Strategy

### Unit Tests

| Test | Location | Description |
|------|----------|-------------|
| `TestStructuralDiff_NoChange` | `pkg/config/diff_test.go` | Identical configs return no structural diff |
| `TestStructuralDiff_NameChanged` | `pkg/config/diff_test.go` | Robot name change detected |
| `TestStructuralDiff_ComponentAdded` | `pkg/config/diff_test.go` | Component count mismatch detected |
| `TestStructuralDiff_ComponentTypeChanged` | `pkg/config/diff_test.go` | Component type change detected |
| `TestStructuralDiff_DisabledChanged` | `pkg/config/diff_test.go` | Disabled flag change treated as structural |
| `TestStructuralDiff_NATSChanged` | `pkg/config/diff_test.go` | NATS URL change detected |
| `TestStructuralDiff_DashboardChanged` | `pkg/config/diff_test.go` | Dashboard listen address change detected |
| `TestStructuralDiff_LogNotStructural` | `pkg/config/diff_test.go` | Log level change is not structural |
| `TestAttributeDiff_NoChange` | `pkg/config/diff_test.go` | Identical attributes return empty diff |
| `TestAttributeDiff_ComponentChanged` | `pkg/config/diff_test.go` | Changed component attributes detected |
| `TestAttributeDiff_ServiceChanged` | `pkg/config/diff_test.go` | Changed service attributes detected |
| `TestAttributeDiff_NestedChange` | `pkg/config/diff_test.go` | Nested attribute change (color object) detected |
| `TestAttributeDiff_MultipleChanges` | `pkg/config/diff_test.go` | Multiple components and services changed |

### Watcher Tests

| Test | Location | Description |
|------|----------|-------------|
| `TestWatcher_DetectsWrite` | `pkg/config/watcher_test.go` | File write triggers callback |
| `TestWatcher_Debounce` | `pkg/config/watcher_test.go` | Rapid writes produce single callback |
| `TestWatcher_InvalidJSON` | `pkg/config/watcher_test.go` | Invalid JSON does not trigger callback |
| `TestWatcher_ContextCancel` | `pkg/config/watcher_test.go` | Watcher stops on context cancellation |
| `TestWatcher_EditorRename` | `pkg/config/watcher_test.go` | Write-to-temp-then-rename pattern works |

### Integration Tests

| Test | Location | Description |
|------|----------|-------------|
| `TestRobot_ConfigReload_ParameterChange` | `pkg/robot/robot_test.go` | End-to-end: modify attribute, verify Reconfigure called |
| `TestRobot_ConfigReload_StructuralRejection` | `pkg/robot/robot_test.go` | End-to-end: add component, verify rejection |
| `TestRobot_ConfigReload_PartialFailure` | `pkg/robot/robot_test.go` | One component fails Reconfigure, others succeed |

### Satellite Repo Tests

Each satellite repo should test its own `Reconfigure()` implementation:

| Repo | Test |
|------|------|
| `gorai-tasmota` | `TestTasmota_Reconfigure_PollInterval` -- verify new poll interval takes effect |
| `gorai-tasmota` | `TestKauf_Reconfigure_DefaultBrightness` -- verify new defaults used on next on command |
| `gorai-light-controller` | `TestController_Reconfigure_OffsetChange` -- verify schedule recalculation |
| `gorai-light-controller` | `TestController_Reconfigure_WindowShift` -- verify immediate action if time falls in/out of window |

## 11. Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/fsnotify/fsnotify` | v1.8+ | Filesystem event notifications |

This is the only new external dependency. All other functionality uses the Go standard library and existing gorai packages.

## 12. Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Config file on NFS or network filesystem where fsnotify does not work | Document that hot-reload requires a local filesystem. Provide a SIGHUP fallback in a future iteration. |
| Editor writes partial file before rename | Debounce timer + `config.Load()` validation catches incomplete JSON |
| Race between watcher callback and robot shutdown | Context cancellation stops watcher; `handleConfigReload` checks `r.ctx.Done()` before applying |
| Component Reconfigure panics | Recover in the loop; log and continue with other components |
| Rapid config changes overwhelm the system | Debounce ensures at most one reload per 500ms |
