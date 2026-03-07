# Spec Compliance Review: Configuration Hot-Reload

**Reviewer:** AI Spec Compliance Agent
**Date:** 2026-02-22
**Spec Sources:** `CLAUDE.md` (Configuration Hot-Reload Standard), `docs/DESIGN-config-hot-reload.md`, `docs/TEST-PLAN-config-hot-reload.md`
**Implementation Files:** `pkg/config/diff.go`, `pkg/config/diff_test.go`, `pkg/config/watcher.go`, `pkg/config/watcher_test.go`, `pkg/robot/robot.go`, `pkg/robot/reload_test.go`, `pkg/topics/topics.go`

---

## Findings

### 1. Component/Service Matching: Name-Based vs Index-Based

**Severity:** MEDIUM

**Spec (DESIGN doc, Section 4):** "Components and services are matched by index position in the JSON arrays. This is deliberate: reordering components in the file is treated as a structural change because the framework has no way to distinguish 'component A was renamed' from 'component A was replaced' if we match by name alone. Index-based matching is simple and predictable."

**Spec (CLAUDE.md):** "Component count, names, types, models, disabled flags must match exactly" (step 3 of Robot Runtime Behavior). This is ambiguous but combined with the DESIGN doc, index-based matching was the intended approach.

**Implementation (`diff.go`):** Uses name-based matching via `map[string]ComponentConfig`. The comment on `StructuralDiff` explicitly says "Components and services are matched by name, not by index position."

**Impact:** The implementation diverges from the DESIGN doc's explicit design decision. With name-based matching, reordering components in the JSON file is NOT treated as a structural change (confirmed by `TestStructuralDiff_ComponentOrderChanged` and `TestStructuralDiff_ServiceOrderChanged` which assert reordering is non-structural). The DESIGN doc specifically calls out that reordering SHOULD be structural.

**Note:** The TEST-PLAN doc (`TestDiffConfigs_ComponentOrderChanged`) also expects order changes to be non-structural, which contradicts the DESIGN doc. The test plan and implementation agree with each other but disagree with the DESIGN doc.

---

### 2. NATS Structural Diff: Missing Field Comparisons

**Severity:** HIGH

**Spec (DESIGN doc, Section 4):** "URL, URLs, JetStream, CredentialsFile, TLS, ConnectTimeout, ReconnectWait, MaxReconnects must all match."

**Implementation (`diff.go`, `natsStructuralDiff`):** Only compares URL, JetStream, and CredentialsFile. The following fields are NOT compared:
- `URLs` (multi-server cluster URLs)
- `TLS` (TLS configuration including CA, cert, key files)
- `ConnectTimeout`
- `ReconnectWait`
- `MaxReconnects`

**Impact:** Changing the TLS config, adding cluster URLs, or modifying connection timeouts would be silently accepted as a non-structural change and would NOT take effect (since these are connection-level settings established at startup). This could lead to confusion where an operator edits NATS TLS settings, sees a successful reload, but the changes have no effect.

---

### 3. Dashboard Structural Diff: Missing Field Comparisons

**Severity:** MEDIUM

**Spec (DESIGN doc, Section 4):** "`dashboard.enabled`, `dashboard.listen`, `dashboard.websocket`, `dashboard.video` must match."

**Implementation (`diff.go`, `dashboardStructuralDiff`):** Only compares `Enabled` and `Listen`. Does NOT compare:
- `WebSocket` configuration
- `Video` configuration

**Impact:** Changes to WebSocket buffer sizes or video streaming settings would be silently accepted as parameter changes but would not take effect on the running dashboard. Lower severity than NATS because these are less likely to cause operational confusion.

---

### 4. Watcher Constructor Signature Mismatch

**Severity:** LOW

**Spec (DESIGN doc, Section 5):** Constructor signature is `func NewWatcher(path string, onChange func(*RDL), logger *slog.Logger) *Watcher` (returns only `*Watcher`).

**Implementation (`watcher.go`):** Returns `(*Watcher, error)`.

**Impact:** The implementation is actually better than the spec -- returning an error allows validation of the path at construction time. The DESIGN doc should be updated to reflect this. No functional issue.

---

### 5. Watcher.Start() is Non-Blocking vs Spec Says Blocking

**Severity:** LOW

**Spec (DESIGN doc, Section 5):** "Start begins watching the config file for changes. It blocks until the context is cancelled."

**Implementation (`watcher.go`):** `Start()` launches a goroutine via `go w.run(ctx, fsw)` and returns immediately (non-blocking).

**Impact:** The DESIGN doc's integration example in Section 7 shows `watcher.Start(r.ctx)` being called inline in `Robot.Start()`, which requires non-blocking behavior. The implementation is correct for actual usage; the DESIGN doc's Start() description is misleading but the integration example is consistent with the implementation.

---

### 6. Missing Panic Recovery in Reconfigure Loop

**Severity:** MEDIUM

**Spec (DESIGN doc, Section 12, Risks):** "Component Reconfigure panics" with mitigation "Recover in the loop; log and continue with other components."

**Implementation (`robot.go`, `handleConfigReload`):** No `recover()` call in the reconfigure loops. If any component's `Reconfigure()` panics, it will crash the entire watcher goroutine and potentially the robot process.

**Impact:** A buggy component that panics during reconfiguration would take down the hot-reload system entirely, and depending on goroutine structure, could crash the robot.

---

### 7. Missing `cfgMu` Write Lock in DESIGN Doc vs Correct Implementation

**Severity:** LOW (positive deviation)

**Spec (DESIGN doc, Section 7, Concurrency):** "A mutex (`cfgMu sync.RWMutex`) should be added to `Robot` to protect `r.cfg` access." (stated as a recommendation)

**Implementation (`robot.go`):** `cfgMu` is implemented and used correctly -- `RLock` for reads in `handleConfigReload` and `Config()`, `Lock` for the write after successful reload.

**Impact:** None. The implementation correctly follows the recommendation.

---

### 8. Missing Test: `TestWatcher_NonexistentFile`

**Severity:** LOW

**Spec (TEST-PLAN doc, Phase 2):** Specifies `TestWatcher_NonexistentFile` -- "Call NewWatcher with a path to a file that does not exist. Assert: returns a non-nil error immediately."

**Implementation (`watcher_test.go`):** This test does not exist.

**Impact:** The `NewWatcher` constructor does not actually validate that the file exists (it only calls `filepath.Abs`), so this test would likely fail if written. The constructor should validate file existence per the test plan, or the test plan should be updated.

---

### 9. Missing Test: `TestWatcher_FileDeleted`

**Severity:** LOW

**Spec (TEST-PLAN doc, Phase 2):** Specifies `TestWatcher_FileDeleted` -- verifying that deleting the config file does not cause a crash.

**Implementation (`watcher_test.go`):** This test does not exist.

**Impact:** Untested edge case. The implementation likely handles this gracefully (fsnotify would fire a Remove event which is filtered out), but it is not verified.

---

### 10. Missing Tests: NATS Event Tests

**Severity:** MEDIUM

**Spec (TEST-PLAN doc, Phase 3):** Specifies `TestReload_NATSEventOnSuccess` and `TestReload_NATSEventOnStructuralRejection` -- verifying that NATS events are published with correct payloads.

**Implementation (`reload_test.go`):** Neither test exists. The `newTestRobot()` helper creates a robot with `nats: nil`, so no NATS events are published during any test. There is no `mockNATSClient` as specified in the test plan.

**Impact:** The NATS event publishing path (`publishConfigReloadEvent`) is completely untested. The event payload structure, topic name, and failure handling during publish are not verified by any test.

---

### 11. Test Function Naming Convention Mismatch

**Severity:** LOW

**Spec (TEST-PLAN doc):** All diff tests use the `TestDiffConfigs_*` naming pattern (e.g., `TestDiffConfigs_IdenticalConfigs`, `TestDiffConfigs_ComponentAdded`).

**Implementation (`diff_test.go`):** Uses `TestStructuralDiff_*` and `TestAttributeDiff_*` naming patterns.

**Impact:** No functional impact. The test plan proposed a unified `DiffConfigs` function that returns a combined `ConfigDiff` struct, while the implementation uses two separate functions (`StructuralDiff` and `AttributeDiff`). The API is cleaner in the implementation.

---

### 12. Test Plan Proposed Unified `DiffConfigs` Function vs Two Separate Functions

**Severity:** LOW

**Spec (TEST-PLAN doc, Phase 1):** Describes a single `DiffConfigs(old, new *RDL)` function returning a `ConfigDiff` struct with `Structural bool`, `StructuralReasons []string`, and `AttributeChanges []AttributeChange`.

**Implementation (`diff.go`):** Two separate exported functions: `StructuralDiff(old, new *RDL) (string, bool)` and `AttributeDiff(old, new *RDL) ([]AttributeChange, []AttributeChange)`.

**Impact:** The DESIGN doc (Section 6) specifies two separate functions, which the implementation follows. The test plan diverges from the DESIGN doc. No functional issue -- the implementation matches the DESIGN doc.

---

### 13. Watcher Callback Signature: Error Not Passed to Callback

**Severity:** LOW

**Spec (TEST-PLAN doc, Phase 2):** Describes `NewWatcher(path string, callback func(*RDL, error))` -- callback receives both config and error.

**Implementation (`watcher.go`):** `onChange func(*RDL)` -- errors are handled internally (logged), and the callback is only invoked on success.

**Impact:** The implementation matches the DESIGN doc (Section 5), not the test plan. The implementation is arguably better: the robot handler only needs to deal with valid configs. The watcher handles error logging internally.

---

### 14. Watcher Performs Validation, Not Specified in DESIGN

**Severity:** LOW (positive deviation)

**Spec (DESIGN doc, Section 5):** Only mentions calling `config.Load(path)`. On success, invoke `onChange`.

**Implementation (`watcher.go`, `loadAndNotify`):** Calls both `Load(w.path)` AND `cfg.Validate()` before invoking the callback.

**Impact:** This is a positive deviation -- extra validation provides defense in depth. The spec's step 2 in CLAUDE.md says "Call config.Load(path) on the new file. If it fails, log error and return," which implies Load should handle validation, but explicit Validate is safer.

---

### 15. `handleConfigReload` Does Not Check `r.ctx.Done()` Before Applying

**Severity:** LOW

**Spec (DESIGN doc, Section 12, Risks):** "Race between watcher callback and robot shutdown" with mitigation: "Context cancellation stops watcher; `handleConfigReload` checks `r.ctx.Done()` before applying."

**Implementation (`robot.go`, `handleConfigReload`):** Does not check `r.ctx.Done()` at any point. The `loadAndNotify` function in the watcher checks context before loading, but the handler itself does not.

**Impact:** During shutdown, a reload could be in progress and attempt to reconfigure components that are being closed. The Reconfigure calls do pass `r.ctx` so individual components can check for cancellation, but the handler itself does not short-circuit.

---

### 16. `StructuralDiff` Returns Only First Reason

**Severity:** LOW

**Spec (DESIGN doc, Section 4):** "the first mismatch produces the rejection reason" -- this is by design.

**Spec (TEST-PLAN doc, Phase 1):** `StructuralReasons []string` -- implies multiple reasons can be collected.

**Implementation (`diff.go`):** Returns `(string, bool)` -- only the first reason.

**Impact:** Matches the DESIGN doc. The test plan's multi-reason approach would be more user-friendly for debugging but is not what was designed.

---

### 17. `NewWatcher` Does Not Validate File Existence

**Severity:** LOW

**Spec (DESIGN doc, Section 5, Error Handling):** "Returns an error if the config file path does not exist or is invalid."

**Implementation (`watcher.go`, `NewWatcher`):** Only calls `filepath.Abs(path)` which resolves the path but does not check if the file exists. A nonexistent file path would succeed at construction and only fail later when `Load()` is called.

**Impact:** Delayed error detection. An operator would not learn about a typo in the config path until the first file change event fires. The error would still be caught and logged, but feedback is delayed.

---

## Summary

| Severity | Count | Findings |
|----------|-------|----------|
| CRITICAL | 0 | -- |
| HIGH | 1 | #2: NATS structural diff missing 5 of 8 fields |
| MEDIUM | 4 | #1: Name-based vs index-based matching divergence from DESIGN doc; #3: Dashboard diff missing websocket/video; #6: No panic recovery in reconfigure loop; #10: NATS event publishing completely untested |
| LOW | 12 | #4, #5, #7, #8, #9, #11, #12, #13, #14, #15, #16, #17 |

### Recommended Priority Actions

1. **HIGH:** Add NATS structural diff comparisons for `URLs`, `TLS`, `ConnectTimeout`, `ReconnectWait`, and `MaxReconnects` fields in `pkg/config/diff.go`.
2. **MEDIUM:** Add dashboard structural diff comparisons for `WebSocket` and `Video` fields in `pkg/config/diff.go`.
3. **MEDIUM:** Add `recover()` in the reconfigure loops in `pkg/robot/robot.go` per the risk mitigation in the DESIGN doc.
4. **MEDIUM:** Add NATS event publishing tests with a mock NATS client in `pkg/robot/reload_test.go`.
5. **MEDIUM:** Resolve the name-based vs index-based matching discrepancy -- either update the DESIGN doc to endorse name-based matching (since test plan already agrees with implementation), or change the implementation to index-based.
