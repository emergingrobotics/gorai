# Dashboard Enhancement - Deferred Review Findings (2026-02-22)

## Fixed in This Iteration

- **D-1 (Critical)**: `data-state` attribute used display label instead of raw "on"/"off" value. Fixed in `writeStatusBadge()`.
- **D-2 (High)**: WebSocket reconnect used `location.reload()`. Replaced with exponential backoff reconnection.
- **D-3 (High)**: DoCommand didn't publish readings after state change (30s delay). Added `publishReadings()` calls after all successful state changes in tasmota, kauf, and shelly.
- **S-1 (Medium)**: `handleNATSCommand` used `context.Background()` with no timeout. Added `defaultCommandTimeout` context.
- **S-4 (Medium)**: `OnStatusChange` fired on every data message. Now compares old/new values before firing callback.
- **S-5 (Medium)**: CSS selector injection in `dashboard.js`. Added `escapeCSSSelector()` sanitization.

## Deferred (Pre-existing Issues)

### CRITICAL: XSS in camera/model sub-handlers (Security Review)
**Severity**: Critical (pre-existing)
**Rationale**: Camera and model handlers in `cameras/handler.go` and `models/handler.go` don't use `html.EscapeString()`. Not introduced by this change. Should be fixed in a dedicated security hardening pass.

### CRITICAL: HTML via fragmented writes (Design Review)
**Severity**: Medium (downgraded from critical for this change)
**Rationale**: The entire `handleIndex` function uses `w.Write([]byte(...))` instead of `html/template`. This is the pre-existing pattern. A template refactor is worthwhile but is a separate architectural effort, not part of this feature.

### HIGH: O(N*M) camera status lookup (Design Review)
**Severity**: Low
**Rationale**: Pre-existing code. Camera count is typically < 10. Not a real performance concern.

### HIGH: Service status doesn't check if actually running (Design Review)
**Severity**: Medium
**Rationale**: Valid concern. The dashboard currently only knows config-level enabled/disabled. Runtime health checking requires NATS heartbeat integration. Future enhancement.

### MEDIUM: CSP allows unsafe-inline (Security Review)
**Severity**: Low
**Rationale**: Pre-existing. The dashboard uses inline styles for margins. Extracting to CSS classes would allow stricter CSP.

### MEDIUM: Silent BroadcastJSON errors (Design Review)
**Severity**: Low
**Rationale**: Pre-existing. WebSocket broadcast is best-effort by design.

---

# Config Hot-Reload - Deferred Review Findings (2026-02-22)

## Fixed in This Iteration

- **HIGH: NATS structural diff missing fields** — Added comparison for `URLs`, `TLS`, `ConnectTimeout`, `ReconnectWait`, `MaxReconnects` in `natsStructuralDiff()`.
- **HIGH: Dashboard structural diff missing fields** — Added comparison for `WebSocket` and `Video` in `dashboardStructuralDiff()`.
- **HIGH: Unprotected `r.cfg` reads** — Added `cfgMu` locking to `getNATSURL()`, `Run()`, `Stop()`.
- **HIGH: No file size limit** — Added 1MB max file size check in `config.Load()`.
- **HIGH: Race condition in reload handler** — Added `reloadMu` to serialize concurrent `handleConfigReload` calls.
- **HIGH: No symlink protection** — Added `filepath.EvalSymlinks()` in `NewWatcher`.
- **MEDIUM: No panic recovery in reconfigure loop** — Added `safeReconfigure()` with `defer recover()`.
- **MEDIUM: handleConfigReload doesn't check ctx.Done()** — Added context cancellation check at top of handler.

## Deferred

### MEDIUM: Name-based vs index-based matching (Spec Review #1)
**Severity**: Medium
**Rationale**: Implementation uses name-based matching, which is more intuitive. Design doc specifies index-based. The design doc should be updated to reflect the implementation choice. No functional risk.

### MEDIUM: Config updated even when all Reconfigure calls fail (Design Review MEDIUM-4)
**Severity**: Medium
**Rationale**: By design, the stored config reflects the desired state. On next reload, unchanged attributes won't trigger re-reconfiguration since the diff is against the stored (desired) config. Acceptable for the current use case. A rollback mechanism would add significant complexity.

### MEDIUM: `depends_on` changes silently accepted (Design Review MEDIUM-6)
**Severity**: Low
**Rationale**: `depends_on` is only used at startup. Changes are stored and take effect on next restart. Logging a notice would be nice but not critical.

### MEDIUM: `log_level` per-component changes silently accepted (Design Review MEDIUM-7)
**Severity**: Low
**Rationale**: Per-component log level is not yet implemented in the runtime. When it is, it should be added to the hot-reload path.

### MEDIUM: Environment variable expansion scope (Security M1)
**Severity**: Medium
**Rationale**: Config file write access is a prerequisite for exploitation. On embedded SBC targets, the config is typically only writable by the robot user. An allowlist of env var prefixes is a good future hardening measure.

### MEDIUM: NATS config reload events disclose topology (Security M3)
**Severity**: Low
**Rationale**: NATS authorization should be configured at the infrastructure level. Component names are not considered secrets in the gorai architecture.

### MEDIUM: NATS event publishing untested (Spec Review #10)
**Severity**: Medium
**Rationale**: Requires a mock NATS client. The publish path is simple (JSON marshal + publish). Should be added in a testing infrastructure improvement pass.

### LOW: NewWatcher does not validate file existence (Spec Review #17)
**Severity**: Low
**Rationale**: File existence is validated when `Load()` is called on first change. Delayed error detection is acceptable.

### LOW: JSON marshal byte comparison efficiency (Design Review LOW-1)
**Severity**: Low
**Rationale**: Config sizes are small (< 50 components). JSON marshal is correct and deterministic.

### LOW: Shared config pointer without deep copy (Security L4)
**Severity**: Low
**Rationale**: All current callers treat config as read-only. A deep copy would add allocation overhead on every access.
