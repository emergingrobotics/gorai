# Test Plan: Configuration Hot-Reload

Tests for the configuration hot-reload feature described in CLAUDE.md under "Configuration Hot-Reload Standard." Three phases covering structural diff, config watcher, and robot reload integration.

## Phase 1: Structural Diff Tests

**File:** `pkg/config/diff_test.go`

These tests verify that `DiffConfigs(old, new *RDL)` correctly classifies changes as structural or attribute-only. The diff function returns a `ConfigDiff` containing:
- `Structural bool` -- true if any structural field changed
- `StructuralReasons []string` -- human-readable list of what changed
- `AttributeChanges []AttributeChange` -- list of (name, newAttributes) for attribute-only changes

### Test Functions

#### `TestDiffConfigs_IdenticalConfigs`

Two identical `*RDL` values with the same robot name, NATS config, components, and services. Expect `Structural == false`, `len(StructuralReasons) == 0`, `len(AttributeChanges) == 0`.

#### `TestDiffConfigs_ComponentAdded`

New config has one additional component. Expect `Structural == true`, `StructuralReasons` contains a string mentioning the added component name.

#### `TestDiffConfigs_ComponentRemoved`

New config is missing a component present in the old config. Expect `Structural == true`, `StructuralReasons` mentions the removed component name.

#### `TestDiffConfigs_ComponentTypeChanged`

Same component name, but `Type` field differs (e.g., `"sensor"` to `"motor"`). Expect `Structural == true`, reason mentions the component name and type change.

#### `TestDiffConfigs_ComponentModelChanged`

Same component name, but `Model` field differs (e.g., `"bme280"` to `"dht22"`). Expect `Structural == true`, reason mentions model change.

#### `TestDiffConfigs_ComponentDisabledChanged`

Component `Disabled` flag changes from `false` to `true` (or vice versa). Expect `Structural == true`, reason mentions disabled flag change. Per the spec, disabled flag changes are structural because they require lifecycle management.

#### `TestDiffConfigs_ComponentAttributesOnlyChanged`

Same component structure, but `Attributes` map differs (e.g., `"poll_interval"` changed from `30` to `60`). Expect `Structural == false`, `len(AttributeChanges) == 1` with the correct component name and new attributes map.

#### `TestDiffConfigs_ServiceAdded`

New config has an additional service. Expect `Structural == true`.

#### `TestDiffConfigs_ServiceRemoved`

New config is missing a service. Expect `Structural == true`.

#### `TestDiffConfigs_ServiceTypeChanged`

Same service name, `Type` differs. Expect `Structural == true`.

#### `TestDiffConfigs_ServiceModelChanged`

Same service name, `Model` differs. Expect `Structural == true`.

#### `TestDiffConfigs_ServiceAttributesOnlyChanged`

Same service structure, but `Attributes` map differs. Expect `Structural == false`, `len(AttributeChanges) == 1` with the correct service name and new attributes.

#### `TestDiffConfigs_RobotNameChanged`

`Robot.Name` differs between old and new. Expect `Structural == true`.

#### `TestDiffConfigs_NATSURLChanged`

`NATS.URL` differs between old and new. Expect `Structural == true`.

#### `TestDiffConfigs_MultipleAttributeChanges`

Three components and two services, each with different attribute changes. Expect `Structural == false`, `len(AttributeChanges) == 5` (or however many actually changed), each entry has the correct name and new attributes.

#### `TestDiffConfigs_ComponentOrderChanged`

Same set of components, but listed in a different order in the `Components` slice. Expect `Structural == false`, `len(AttributeChanges) == 0`. The diff must be order-independent -- it matches components by name, not by index.

#### `TestDiffConfigs_ServiceOrderChanged`

Same set of services in different order. Expect `Structural == false`, `len(AttributeChanges) == 0`.

---

## Phase 2: Config Watcher Tests

**File:** `pkg/config/watcher_test.go`

These tests verify that `NewWatcher(path string, callback func(*RDL, error))` correctly watches a config file for changes using fsnotify with 500ms debounce.

All tests use a temporary directory with a valid `robot.json` file. The callback captures results via a channel for assertion.

### Test Functions

#### `TestWatcher_FileModified`

1. Create a valid `robot.json` in a temp directory.
2. Start watcher with a callback that sends the loaded `*RDL` to a channel.
3. Write a modified (but still valid) config to the same path.
4. Assert: callback fires within 2 seconds, received `*RDL` is non-nil, error is nil, and the new config reflects the written changes.

#### `TestWatcher_DebounceMultipleRapidWrites`

1. Start watcher.
2. Write to the config file 5 times in rapid succession (< 100ms apart).
3. Wait 2 seconds.
4. Assert: callback fired exactly once (debounce collapsed all writes into one). The received config matches the last write.

#### `TestWatcher_InvalidJSONWrite`

1. Start watcher with valid config.
2. Overwrite the file with invalid JSON (e.g., `{{{`).
3. Assert: callback fires, `*RDL` argument is nil, error is non-nil and contains a parse/unmarshal error message.

#### `TestWatcher_FileDeleted`

1. Start watcher with valid config.
2. Delete the config file.
3. Wait 2 seconds.
4. Assert: no panic, no callback fires (or callback fires with an error -- either behavior is acceptable as long as no crash occurs). Watcher remains running.

#### `TestWatcher_ContextCancelled`

1. Start watcher with a cancellable context.
2. Cancel the context.
3. Assert: watcher goroutine exits cleanly within 2 seconds. No callback fires after cancellation. No goroutine leak (verify with `runtime.NumGoroutine` before and after, or use `goleak`).

#### `TestWatcher_NonexistentFile`

1. Call `NewWatcher` with a path to a file that does not exist.
2. Assert: returns a non-nil error immediately. No watcher goroutine is started.

---

## Phase 3: Robot Reload Integration Tests

**File:** `pkg/robot/reload_test.go`

These tests verify the full reload path in the `Robot` struct: watcher fires, diff is computed, `Reconfigure` is called on affected components/services, NATS events are published.

Tests use mock components, mock services, and a mock NATS client to avoid real hardware and network dependencies.

### Mock Types

```go
// mockReconfigurable implements resource.Resource with a recorded Reconfigure call.
type mockReconfigurable struct {
    name            resource.Name
    reconfigureCalls []resource.Config
    reconfigureError error
    mu              sync.Mutex
}

// mockNonReconfigurable is a plain struct that does NOT implement resource.Resource.
type mockNonReconfigurable struct{}

// mockNATSClient records published messages for assertion.
type mockNATSClient struct {
    published []publishedMessage
    mu        sync.Mutex
}

type publishedMessage struct {
    Subject string
    Data    []byte
}
```

### Test Functions

#### `TestReload_AttributeChangeOnComponent`

1. Create a `Robot` with one component (`"plug_a"`) backed by a `mockReconfigurable`.
2. Store the initial config in `r.cfg`.
3. Build a new config identical to the initial except `plug_a.Attributes["schedule_offset"]` changed from `0` to `-30`.
4. Trigger the reload path (call the internal reload method directly, or write the new config to a temp file and let the watcher fire).
5. Assert: `mockReconfigurable.reconfigureCalls` has length 1. The `Config.Attributes` in that call contains `"schedule_offset": -30`.

#### `TestReload_AttributeChangeOnService`

1. Create a `Robot` with one service (`"pool-lights"`) backed by a `mockReconfigurable`.
2. Change `pool-lights.Attributes["brightness"]` from `100` to `50` in the new config.
3. Trigger reload.
4. Assert: `mockReconfigurable.reconfigureCalls` has length 1 with the new brightness value.

#### `TestReload_StructuralChangeRejected`

1. Create a `Robot` with one component.
2. Build a new config that adds a second component (structural change).
3. Trigger reload.
4. Assert: no `Reconfigure` called on any mock. `r.cfg` still equals the original config. Error is logged (check logger output or return value).

#### `TestReload_NonResourceComponentSkipped`

1. Create a `Robot` with one component backed by `mockNonReconfigurable` (does not implement `resource.Resource`).
2. Change attributes for that component in the new config.
3. Trigger reload.
4. Assert: no panic. Reload completes without error. The non-resource component is skipped gracefully. `r.cfg` is updated to the new config (since the change was attribute-only and non-structural).

#### `TestReload_ReconfigureError`

1. Create a `Robot` with two components, both `mockReconfigurable`. Set `reconfigureError` on the first mock.
2. Change attributes on both components.
3. Trigger reload.
4. Assert: both mocks have `Reconfigure` called (the error on the first does not prevent the second from being called). The error from the first is logged but does not cause the reload to fail entirely. `r.cfg` is updated to the new config.

#### `TestReload_NATSEventOnSuccess`

1. Create a `Robot` with a `mockNATSClient` and two components with attribute changes.
2. Trigger reload.
3. Assert: `mockNATSClient.published` contains one message on subject `gorai.<robot>.system.config_reloaded`. Unmarshal the payload and verify:
   - `rejected` is `false`
   - `updated_components` contains the two component names
   - `timestamp` is a valid RFC3339 timestamp

#### `TestReload_NATSEventOnStructuralRejection`

1. Create a `Robot` with a `mockNATSClient`.
2. Trigger reload with a structural change (e.g., component added).
3. Assert: `mockNATSClient.published` contains one message on subject `gorai.<robot>.system.config_reloaded`. Unmarshal the payload and verify:
   - `rejected` is `true`
   - `reason` is a non-empty string describing what structural change was detected

#### `TestReload_ConfigUpdatedAfterSuccess`

1. Create a `Robot` with initial config.
2. Trigger reload with attribute-only changes.
3. Assert: `r.Config()` returns the new config, not the old one. Subsequent operations use the updated values.

#### `TestReload_ConfigNotUpdatedAfterRejection`

1. Create a `Robot` with initial config.
2. Trigger reload with a structural change.
3. Assert: `r.Config()` still returns the original config unchanged.

---

## Test Execution

Run all hot-reload tests:

```bash
make test-all
```

Run only the diff tests:

```bash
go test ./pkg/config/ -run TestDiffConfigs -v
```

Run only the watcher tests:

```bash
go test ./pkg/config/ -run TestWatcher -v
```

Run only the reload integration tests:

```bash
go test ./pkg/robot/ -run TestReload -v
```
