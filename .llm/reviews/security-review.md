# Security Review: Configuration Hot-Reload Feature

**Date:** 2026-02-22
**Scope:** Config file watcher, hot-reload handler, config loading/parsing, environment variable expansion, NATS event publishing.

**Files reviewed:**
- `/er/gorai/pkg/config/diff.go`
- `/er/gorai/pkg/config/watcher.go`
- `/er/gorai/pkg/config/config.go` (Load, LoadFromBytes, expandEnvVars, Validate)
- `/er/gorai/pkg/robot/robot.go` (handleConfigReload, startConfigWatcher, publishConfigReloadEvent)
- `/er/gorai/pkg/topics/topics.go`

---

## Critical Findings

None.

---

## High Findings

### H1: No File Size Limit on Config Reload

**Severity:** HIGH
**Location:** `/er/gorai/pkg/config/config.go:411-417`

`Load()` calls `os.ReadFile(path)` with no file size limit. An attacker who can write to the config file (or replace it via symlink) can cause the robot process to allocate unbounded memory by placing a multi-gigabyte file at the config path. This is invoked on every detected file change via the watcher.

The `expandEnvVars` function then runs a regex replacement across the entire byte slice, compounding memory usage (the regex engine creates intermediate allocations). Combined with the debounce window of only 500ms, an attacker could trigger repeated multi-gigabyte allocations.

**Impact:** Denial of service via OOM kill of the robot process. On resource-constrained SBC targets (Raspberry Pi with 1-4GB RAM), this is particularly dangerous.

**Recommendation:** Add a maximum file size check before reading. A reasonable limit (e.g., 1MB) should be enforced:

```go
info, err := os.Stat(path)
if err != nil { return nil, err }
if info.Size() > maxConfigFileSize {
    return nil, fmt.Errorf("config file too large: %d bytes (max %d)", info.Size(), maxConfigFileSize)
}
```

### H2: Race Condition Between Config Read and Config Apply

**Severity:** HIGH
**Location:** `/er/gorai/pkg/robot/robot.go:1018-1097`

In `handleConfigReload`, the old config is read under `cfgMu.RLock()` (line 1019-1021), then the lock is released. The structural diff and attribute diff are computed without holding any lock. Then individual components are reconfigured (lines 1039-1089), and finally the new config is stored under `cfgMu.Lock()` (line 1092-1094).

Between releasing the read lock and acquiring the write lock, another goroutine could trigger a second reload (the debounce timer fires a goroutine via `time.AfterFunc`, which is not serialized). Two concurrent `handleConfigReload` calls could:

1. Both read the same `oldCfg`.
2. Both compute diffs against the same baseline.
3. Both call `Reconfigure()` on the same components concurrently.
4. Both overwrite `r.cfg`, with the last write winning regardless of ordering.

While `Reconfigure()` implementations are required to be thread-safe per the contract, the double-reconfigure with potentially different configs and the non-atomic read-diff-apply-store sequence can lead to inconsistent state where the stored config does not match what components were actually configured with.

**Impact:** Components could end up configured with attributes from one config version while `r.cfg` reflects a different version. This is a correctness issue that could cause operational errors in a running robot.

**Recommendation:** Serialize config reloads. Either:
- Hold `cfgMu.Lock()` (write lock) for the entire `handleConfigReload` duration, or
- Use a dedicated reload mutex (separate from the config read mutex) to ensure only one reload runs at a time, or
- Use a single-item channel as a serialization mechanism.

### H3: No Symlink or Path Traversal Protection on Watched File

**Severity:** HIGH
**Location:** `/er/gorai/pkg/config/watcher.go:28-39`, `/er/gorai/pkg/config/config.go:411-417`

`NewWatcher` resolves the config path to an absolute path via `filepath.Abs()`, but does not resolve symlinks (`filepath.EvalSymlinks`). The watcher monitors the parent directory. An attacker who can create symlinks in the watched directory can:

1. Replace the config file with a symlink to an arbitrary file (e.g., `/etc/shadow`, `/proc/self/environ`).
2. The watcher detects the change and calls `Load()`, which reads the symlink target.
3. While the content likely fails JSON parsing (and `Load` returns an error), the file contents are read into memory first.
4. If the symlink points to a valid JSON file elsewhere on the filesystem, it could be loaded as the robot's config, potentially with attacker-controlled attributes.

Additionally, the file name matching at line 82 (`filepath.Base(event.Name) != w.fileName`) only checks the base name, not the full resolved path.

**Impact:** On a multi-user system or where the config directory is writable by other processes, an attacker could redirect config loading to an arbitrary file. On embedded targets with single-user setups, this is lower risk.

**Recommendation:**
- Resolve symlinks at watcher creation time: `absPath, err := filepath.EvalSymlinks(absPath)`
- Before loading, verify the file is not a symlink: `info, _ := os.Lstat(path); if info.Mode()&os.ModeSymlink != 0 { reject }`
- Verify file ownership and permissions match expected values.

---

## Medium Findings

### M1: Environment Variable Expansion Leaks Process Environment into Config

**Severity:** MEDIUM
**Location:** `/er/gorai/pkg/config/config.go:450-467`

`expandEnvVars` expands `${VAR}` references by reading from the process environment via `os.Getenv()`. Any environment variable accessible to the robot process can be injected into the config, including potentially sensitive variables like `AWS_SECRET_ACCESS_KEY`, `DATABASE_URL`, `API_KEY`, etc.

The expansion happens before JSON parsing, operating on raw bytes. While the `jsonEscapeValue` function (line 441-448) prevents JSON structure injection (a good mitigation), the expanded values still flow into component and service attributes, which are then:
1. Published over NATS in config reload events (attribute values appear in `ConfigReloadEvent` indirectly via component names).
2. Passed to component constructors and `Reconfigure()` calls.
3. Potentially logged by components.

An attacker who can modify the config file could add `${AWS_SECRET_ACCESS_KEY}` as an attribute value, and the expanded secret would flow through the system.

**Impact:** Information disclosure of process environment variables via config file manipulation. The attacker needs write access to the config file, which somewhat limits the attack surface, but an unintentional leak (e.g., a user adding `${HOME}` to a description field) could expose paths.

**Recommendation:**
- Consider an allowlist of environment variable prefixes that are eligible for expansion (e.g., `GORAI_`, `NATS_`).
- Document that environment variable expansion has access to the full process environment.

### M2: Debounce Timer Does Not Cancel on Context Done

**Severity:** MEDIUM
**Location:** `/er/gorai/pkg/config/watcher.go:93-95`

The debounce timer is created with `time.AfterFunc`, which fires its callback in a new goroutine. When the context is cancelled (line 72-74), the `run` loop stops the debounce timer if it exists. However, there is a race: if the timer fires between the last event and context cancellation, the `loadAndNotify` goroutine runs concurrently with shutdown.

`loadAndNotify` does check `ctx.Done()` at the top (line 108-112), but between that check and the `onChange` callback (line 126), the context could be cancelled. The `onChange` callback (`handleConfigReload`) then calls `Reconfigure()` on components that may already be shutting down, and updates `r.cfg` after the robot has begun its `Stop()` sequence.

**Impact:** Config reload during shutdown could interfere with graceful shutdown, potentially causing panics or resource leaks if components are reconfigured after being closed.

**Recommendation:** Use a done channel or mutex in the watcher to ensure `loadAndNotify` is not invoked after the watcher is stopped. Alternatively, pass the context into `onChange` and check it before applying changes.

### M3: NATS Config Reload Events Disclose Internal Topology

**Severity:** MEDIUM
**Location:** `/er/gorai/pkg/robot/robot.go:1099-1121`, `/er/gorai/pkg/topics/topics.go:167-175`

The `ConfigReloadEvent` published to NATS includes:
- Lists of all updated component names
- Lists of all updated service names
- Lists of all failed component and service names
- On rejection: the reason string, which includes specific field values (e.g., `"robot name changed from \"prod-bot\" to \"test-bot\""`)

The NATS topic `gorai.<robot>.system.config_reloaded` has no access control. Any NATS subscriber can observe:
- The full inventory of component and service names
- Which components/services were reconfigured (revealing which parameters changed)
- Structural details of the configuration (names, types revealed in rejection reasons)

**Impact:** Information disclosure of robot topology and configuration changes to any NATS subscriber. On a shared NATS server, this reveals internal architecture to other tenants.

**Recommendation:**
- Consider using NATS authorization to restrict who can subscribe to `system.*` topics.
- Sanitize rejection reasons to avoid leaking specific field values (e.g., `"structural change in robot identity"` instead of `"robot name changed from X to Y"`).

### M4: Component Name Not Validated Against NATS Wildcards

**Severity:** MEDIUM
**Location:** `/er/gorai/pkg/config/config.go:612-629`, `/er/gorai/pkg/topics/topics.go:63-67`

The `validateName` function allows names matching `^[a-zA-Z][a-zA-Z0-9_-]*$`, which correctly excludes NATS wildcard characters (`*`, `>`, `.`). This is good. However, the topic builder at `topics.go:66` uses string interpolation:

```go
return fmt.Sprintf("gorai.%s.%s.%s", b.robotID, component, messageType)
```

The `robotID` (which comes from `cfg.Robot.Name`) is validated by the same pattern, so NATS wildcard injection via robot name or component name is prevented by the current validation rules.

However, the `messageType` parameter in `Component()` is not validated and is passed as a raw string. If a caller passes a user-controlled value as `messageType`, it could contain NATS wildcards. Currently all callers use constants, so the risk is theoretical.

**Impact:** Low currently, but a future code change could introduce NATS topic injection.

**Recommendation:** Add a comment or assertion to `Component()` documenting that `messageType` must be a constant, or validate it.

---

## Low Findings

### L1: No File Permission Check on Config File

**Severity:** LOW
**Location:** `/er/gorai/pkg/config/config.go:411-417`

`Load()` reads the config file without checking its permissions. On Unix systems, a config file that is world-writable (e.g., mode 0666) means any local user can modify the robot's configuration, triggering a hot-reload with attacker-controlled content.

**Impact:** On multi-user systems, an overly permissive config file allows any user to reconfigure the robot. On single-user embedded systems, this is a non-issue.

**Recommendation:** Log a warning if the config file or its parent directory is writable by group or others. Consider refusing to load files with mode more permissive than 0644.

### L2: JSON Parsing Errors Logged with Raw File Content Context

**Severity:** LOW
**Location:** `/er/gorai/pkg/config/config.go:425`, `/er/gorai/pkg/config/watcher.go:115-116`

When `json.Unmarshal` fails, the error message from the Go JSON parser includes a snippet of the problematic content (character position and nearby bytes). The watcher logs this error. If the config file contained expanded environment variables with secrets, the error message could include fragments of those secrets in the log output.

**Impact:** Potential leakage of secret fragments in log files when a malformed config triggers a parse error after environment variable expansion.

**Recommendation:** Log the error type and position but not the raw error string from `json.Unmarshal`, or redact the error message.

### L3: Watcher Monitors Entire Directory, Not Just Config File

**Severity:** LOW
**Location:** `/er/gorai/pkg/config/watcher.go:51-52`

The watcher adds the parent directory to fsnotify to handle editor rename patterns (a correct design choice). However, it receives events for all files in that directory. While it filters by filename (line 82), a high volume of file operations in the config directory (e.g., writing temporary files, log rotation in the same directory) generates events that must be processed and discarded.

**Impact:** Minor CPU overhead. Not exploitable unless the attacker can create a very high volume of file operations in the config directory, which would require local file system access.

**Recommendation:** No action required. The filename filtering is correct. This is a known limitation of directory-based file watching.

### L4: Config Stored Config Pointer Is Shared Without Deep Copy

**Severity:** LOW
**Location:** `/er/gorai/pkg/robot/robot.go:1092-1094`, `/er/gorai/pkg/robot/robot.go:1124-1128`

`handleConfigReload` stores `newCfg` directly: `r.cfg = newCfg`. The `Config()` accessor returns this pointer. Any caller that modifies the returned `*config.RDL` (or its nested maps like `Attributes`) mutates the shared config. Additionally, the `newCfg` passed to `Reconfigure()` shares attribute maps with `r.cfg`.

**Impact:** Accidental mutation of shared config state. This is primarily a correctness issue, but in a security context, a misbehaving component could alter config values seen by other components.

**Recommendation:** Deep copy the config before storing, or document that the returned config must not be mutated.

---

## Positive Findings (What Is Done Well)

1. **Validation before apply**: The watcher calls `cfg.Validate()` (watcher.go:120) before invoking the `onChange` callback. Invalid configs are rejected with a log message and never reach the robot runtime.

2. **Structural change rejection**: The `StructuralDiff` function comprehensively checks all structural fields (robot name, namespace, NATS config, dashboard config, component/service identity). Structural changes are cleanly rejected without partial application.

3. **Debounce mechanism**: The 500ms debounce (watcher.go:13) correctly handles editor write patterns that produce multiple filesystem events per save. The timer reset logic (lines 90-95) ensures only the final write triggers a reload.

4. **JSON escape in env var expansion**: The `jsonEscapeValue` function (config.go:441-448) uses `json.Marshal` to properly escape expanded environment variable values, preventing JSON structure injection. An attacker who sets `MY_VAR=", "evil": true, "x":"` cannot break out of a JSON string value.

5. **Name validation regex**: The `nameFullPattern` (`^[a-zA-Z][a-zA-Z0-9_-]*$`) effectively prevents NATS wildcard injection, path traversal characters, and shell metacharacters in component/service names.

6. **Context-aware watcher**: The watcher checks `ctx.Done()` in multiple places (run loop, loadAndNotify), providing clean shutdown behavior.

7. **Error resilience**: Config load failures and validation failures are logged and the current config is preserved. The system never enters a partially configured state on load failure.

8. **Component-level error isolation**: If one component's `Reconfigure()` fails, other components are still reconfigured. Failures are tracked and reported in the NATS event.

---

## Summary

| Severity | Count | Key Issues |
|----------|-------|------------|
| Critical | 0 | -- |
| High | 3 | No file size limit; race condition in reload handler; no symlink protection |
| Medium | 4 | Env var expansion scope; debounce/shutdown race; NATS event info disclosure; topic builder safety |
| Low | 4 | No permission check; secret leakage in parse errors; directory event volume; shared config pointer |

The hot-reload implementation follows a sound architectural pattern: parse, validate, diff, then apply. The structural change rejection logic is thorough and correct. The primary security concerns center on the file I/O path (no size limits, no symlink checks, no permission validation) and a concurrency issue in the reload handler. On single-user embedded targets, the file I/O risks are mitigated by the deployment model. On multi-user or network-accessible systems, the file size limit (H1) and race condition (H2) should be addressed before production use.
