# Test Plan: Gorai Process-Compose Runtime

**Version:** 1.0
**Date:** 2026-04-11
**Covers:** REQUIREMENTS.md Sections 3-14 (Embedded NATS, Compiler, CLI, Infrastructure, Integration)

---

## 1. Test Strategy

### 1.1 Approach

All implementation follows test-first development. Tests are written before implementation code. Each phase produces a package with its own `_test.go` files. Tests use the standard `testing` package with table-driven patterns matching the existing codebase style (see `pkg/config/config_test.go`).

### 1.2 Test Tiers

| Tier | Tag | Timeout | What it covers |
|---|---|---|---|
| Unit | (none) | 2m | Pure logic: config parsing, compiler output, restart mapping, dependency translation |
| Component | `component` | 2m | Single-package integration: embedded NATS start/stop, health server HTTP |
| Integration | `integration` | 5m | Cross-package: full compile pipeline, `gorai run` with embedded NATS and health |
| System | `system` | 10m | End-to-end: process-compose up/down, ordered shutdown, device reset sequencing |

### 1.3 Test Locations

| Package | New test files |
|---|---|
| `pkg/config/` | `config_processcompose_test.go` (schema additions) |
| `pkg/natsembed/` | `natsembed_test.go` |
| `pkg/health/` | `health_test.go` |
| `pkg/compiler/` | `compiler_test.go` |
| `cmd/gorai/commands/` | `compile_test.go`, `device_reset_test.go` |
| `tests/integration/` | `processcompose_test.go` |

### 1.4 Test Helpers

The following test infrastructure is needed before phase tests can run:

| Helper | Location | Purpose |
|---|---|---|
| `ConfigBuilder` | `internal/testutil/config_builder.go` | Fluent builder for `config.RDL` structs with sane defaults |
| `YAMLAssert` | `internal/testutil/yaml_assert.go` | Parse compiler YAML output and assert on paths (`AssertYAMLPath(t, yaml, "processes.gorai.command", "gorai run robot.json")`) |
| `TestNATSServer` | `internal/testutil/nats_helper.go` | Start/stop an embedded NATS server on a random port for tests, return client URL |
| `FreePort` | `internal/testutil/port.go` | Find an available TCP port for health server and NATS tests |
| `WaitForHTTP` | `internal/testutil/http_helper.go` | Poll an HTTP endpoint until it returns expected status or timeout |

---

## 2. Phase Alignment

### Phase 5a: Config Schema Tests

Package: `pkg/config/`
File: `config_processcompose_test.go`

These tests validate the new RDL fields (`NATSConfig.External`, `MetricsConfig`, `LoggingConfig`) parse correctly, apply defaults, and trigger validation rules.

### Phase 5b: Embedded NATS Tests

Package: `pkg/natsembed/`
File: `natsembed_test.go`

Build tag: `component` (starts real NATS server on random ports).

### Phase 5c: Health Server Tests

Package: `pkg/health/`
File: `health_test.go`

Build tag: `component` (starts real HTTP listener on random ports).

### Phase 5d: Compiler Tests

Package: `pkg/compiler/`
File: `compiler_test.go`

Pure unit tests (no build tag). The compiler is a pure function: `RDL -> YAML bytes`. No I/O, no servers.

### Phase 5e: CLI Tests

Package: `cmd/gorai/commands/`
Files: `compile_test.go`, `device_reset_test.go`

Build tag: `component` for tests that start NATS.

### Phase 5f: Integration Tests

Package: `tests/integration/`
File: `processcompose_test.go`

Build tag: `integration`. Requires NATS server and HTTP port access.

---

## 3. Test Cases by Phase

### 3.1 Phase 5a: Config Schema Tests

#### 3.1.1 NATSConfig.External Field

```go
func TestNATSConfig_ExternalFieldParsing(t *testing.T)
```
- **Input:** JSON with `"nats": {"url": "nats://localhost:4222", "external": true}`
- **Expected:** `cfg.NATS.External == true`
- **Assertions:** Field parses to `true`. Omitted field defaults to `false`.

```go
func TestNATSConfig_ExternalDefaultFalse(t *testing.T)
```
- **Input:** JSON with `"nats": {"url": "nats://localhost:4222"}` (no external field)
- **Expected:** `cfg.NATS.External == false`

```go
func TestNATSConfig_ExternalTrueRemoteURLWarning(t *testing.T)
```
- **Input:** JSON with `"nats": {"url": "nats://192.168.1.100:4222", "external": true}`
- **Expected:** `cfg.DeprecationWarnings()` or a new `cfg.Warnings()` includes a warning about redundant `external` flag.

#### 3.1.2 MetricsConfig

```go
func TestMetricsConfig_Parsing(t *testing.T)
```
Table-driven test:

| Subtest | Input | Expected |
|---|---|---|
| `enabled` | `"metrics": {"enabled": true}` | `cfg.Metrics.Enabled == true` |
| `with_retention` | `"metrics": {"enabled": true, "retention": "14d"}` | `cfg.Metrics.Retention == "14d"` |
| `with_listen` | `"metrics": {"enabled": true, "listen": "0.0.0.0:9090"}` | `cfg.Metrics.Listen == "0.0.0.0:9090"` |
| `absent` | no metrics section | `cfg.Metrics == nil` |
| `defaults` | `"metrics": {"enabled": true}` | Retention defaults to `"7d"`, Listen defaults to `"127.0.0.1:8428"` |

#### 3.1.3 LoggingConfig

```go
func TestLoggingConfig_Parsing(t *testing.T)
```
Table-driven, same structure as MetricsConfig:

| Subtest | Input | Expected |
|---|---|---|
| `enabled` | `"logging": {"enabled": true}` | `cfg.Logging.Enabled == true` |
| `defaults` | `"logging": {"enabled": true}` | Retention defaults to `"3d"`, Listen defaults to `"127.0.0.1:9428"` |
| `absent` | no logging section | `cfg.Logging == nil` |

#### 3.1.4 Validation

```go
func TestValidation_MetricsRetentionFormat(t *testing.T)
```
- **Input:** `"metrics": {"enabled": true, "retention": "invalid"}`
- **Expected:** Validation error mentioning retention format.

---

### 3.2 Phase 5b: Embedded NATS Tests

All tests use `FreePort()` to avoid port conflicts.

```go
func TestEmbeddedNATS_StartAndAcceptConnections(t *testing.T)
```
- **Setup:** Create `natsembed.Server` with random port.
- **Action:** `server.Start(ctx)`, then connect with `nats.Connect(server.ClientURL())`.
- **Assertions:** Connection succeeds. Publish/subscribe round-trip works.
- **Cleanup:** `server.Shutdown()`.
- **Covers:** REQ-NATS-EMBED-1, REQ-NATS-EMBED-2

```go
func TestEmbeddedNATS_ReadyBeforeReturn(t *testing.T)
```
- **Action:** After `server.Start()` returns, immediately attempt `nats.Connect()`.
- **Assertions:** Connection succeeds without retry loop. Server is accepting connections.
- **Covers:** REQ-NATS-EMBED-3

```go
func TestEmbeddedNATS_ClientConnectsAsRegularClient(t *testing.T)
```
- **Action:** Start server, connect with standard `nats.Connect()` using TCP URL.
- **Assertions:** Client `nats.Conn.ConnectedUrl()` matches the TCP listener URL (not in-process).
- **Covers:** REQ-NATS-EMBED-4

```go
func TestEmbeddedNATS_JetStreamEnabled(t *testing.T)
```
- **Setup:** Config with `JetStream: true`.
- **Action:** Start server, connect, call `js, err := nc.JetStream()`, create a stream.
- **Assertions:** No error. Stream creation succeeds.
- **Covers:** REQ-NATS-EMBED-5

```go
func TestEmbeddedNATS_JetStreamDisabledByDefault(t *testing.T)
```
- **Setup:** Config with `JetStream: false` (or omitted).
- **Action:** Start server, connect, attempt JetStream operations.
- **Assertions:** JetStream API returns error indicating JetStream is not enabled.

```go
func TestEmbeddedNATS_TLSConfiguration(t *testing.T)
```
- **Setup:** Generate self-signed certs in `t.TempDir()`. Config with TLS cert/key/CA paths.
- **Action:** Start server, connect with TLS client config.
- **Assertions:** TLS connection succeeds. Non-TLS connection fails.
- **Covers:** REQ-NATS-EMBED-6

```go
func TestEmbeddedNATS_ShutdownDrainsConnections(t *testing.T)
```
- **Setup:** Start server, connect, create subscription.
- **Action:** Call `server.Shutdown()`.
- **Assertions:** Client connection transitions to closed/disconnected state. No pending messages lost (drain completes).
- **Covers:** REQ-NATS-EMBED-7

```go
func TestEmbeddedNATS_ModeSelection(t *testing.T)
```
Table-driven test covering Section 3.1 mode selection:

| Subtest | NATS Config | Expected Mode |
|---|---|---|
| `no_nats_section` | `nil` | Embed, listen on `:4222` |
| `localhost_no_external` | `URL: "nats://localhost:4222"` | Embed |
| `localhost_external_true` | `URL: "nats://localhost:4222", External: true` | Client-only (do not embed) |
| `remote_url` | `URL: "nats://10.0.0.5:4222"` | Client-only |
| `127_0_0_1` | `URL: "nats://127.0.0.1:4222"` | Embed |

---

### 3.3 Phase 5c: Health Server Tests

```go
func TestHealthServer_503BeforeReady(t *testing.T)
```
- **Setup:** Start health server on random port. Do NOT call `SetReady()`.
- **Action:** `GET /healthz`.
- **Assertions:** HTTP 503.
- **Covers:** REQ-CLI-HEALTH-2, REQ-COMPILE-61

```go
func TestHealthServer_200AfterReady(t *testing.T)
```
- **Setup:** Start health server, call `SetReady()`.
- **Action:** `GET /healthz`.
- **Assertions:** HTTP 200.
- **Covers:** REQ-CLI-HEALTH-2, REQ-COMPILE-61

```go
func TestHealthServer_LivezAlways200(t *testing.T)
```
- **Setup:** Start health server. Do NOT call `SetReady()`.
- **Action:** `GET /livez`.
- **Assertions:** HTTP 200.
- **Covers:** REQ-CLI-HEALTH-2

```go
func TestHealthServer_LivezAfterReady(t *testing.T)
```
- **Setup:** Start health server, call `SetReady()`.
- **Action:** `GET /livez`.
- **Assertions:** HTTP 200.

```go
func TestHealthServer_StartsBeforeComponents(t *testing.T)
```
- **Setup:** Start health server.
- **Action:** Immediately `GET /livez` (before any component init).
- **Assertions:** HTTP 200 (server is listening).
- **Covers:** REQ-CLI-HEALTH-3

```go
func TestHealthServer_ConfigurableListen(t *testing.T)
```
- **Setup:** Start with `listen: "127.0.0.1:<random>"`.
- **Assertions:** Server binds to specified address.

```go
func TestHealthServer_DefaultPort(t *testing.T)
```
- **Assertions:** Default listen address is `127.0.0.1:4180`.

---

### 3.4 Phase 5d: Compiler Tests

The compiler is the largest test surface. All tests are pure unit tests that call `compiler.Compile(cfg *config.RDL) ([]byte, error)` and assert on the generated YAML.

#### 3.4.1 Minimal RDL

```go
func TestCompile_MinimalRDL(t *testing.T)
```
- **Input:** `ConfigBuilder().WithName("tiny").Build()`
- **Expected YAML assertions:**
  - `version` is `"0.5"`
  - `ordered_shutdown` is `true`
  - `environment` contains `GORAI_ROBOT_NAME=tiny`, `GORAI_NAMESPACE=tiny`, `NATS_URL=nats://localhost:4222`
  - Single process `gorai` exists
  - `gorai.command` is `gorai run <config-path>`
  - `gorai.namespace` is `core`
  - `gorai.availability.restart` is `on_failure`
  - `gorai.readiness_probe.http_get.path` is `/healthz`
  - `gorai.readiness_probe.http_get.port` is `4180`
  - `gorai.shutdown.signal` is `15`
  - `gorai.shutdown.timeout_seconds` is `30`
  - `gorai.log_location` is `./logs/gorai.log`
  - `log_configuration` section exists
- **Covers:** REQ-COMPILE-10, REQ-COMPILE-11, REQ-COMPILE-12, REQ-COMPILE-20, REQ-COMPILE-21, REQ-COMPILE-22, REQ-COMPILE-62, REQ-LOG-1, REQ-LOG-2

#### 3.4.2 External NATS

```go
func TestCompile_ExternalNATS(t *testing.T)
```
- **Input:** `ConfigBuilder().WithName("bot").WithNATS("nats://localhost:4222", true).Build()`
- **Expected YAML assertions:**
  - Process `nats-server` exists in `infra` namespace
  - `nats-server.availability.restart` is `always`
  - `nats-server` has readiness probe with exec command
  - `gorai.depends_on.nats-server.condition` is `process_healthy`
- **Covers:** REQ-NATS-EXT-1, REQ-NATS-EXT-2, REQ-NATS-EXT-3, REQ-NATS-EXT-4, REQ-COMPILE-23

```go
func TestCompile_ExternalNATSWithJetStream(t *testing.T)
```
- **Input:** External NATS with JetStream enabled.
- **Expected:** `nats-server` command includes JetStream flags (`-js`, storage dir).
- **Covers:** REQ-NATS-EXT-5

```go
func TestCompile_RemoteNATS_NoNATSProcess(t *testing.T)
```
- **Input:** `nats.url: "nats://10.0.0.5:4222"` (remote, non-local).
- **Expected:** No `nats-server` process emitted. `gorai` has no `depends_on` for NATS.

#### 3.4.3 Native Service Processes

```go
func TestCompile_NativeService(t *testing.T)
```
- **Input:** Service with `external.enabled: true`, `external.command: "/usr/bin/nav"`, `external.args: ["--mode", "waypoint"]`.
- **Expected YAML assertions:**
  - Process named after service exists
  - `command` is `/usr/bin/nav --mode waypoint`
  - `namespace` is `services`
  - `environment` includes `GORAI_SERVICE_NAME`
  - `shutdown.signal` is `15`
  - `shutdown.timeout_seconds` is `10`
  - `log_location` is `./logs/<service-name>.log`
  - `log_rotation` block present
- **Covers:** REQ-COMPILE-30, REQ-COMPILE-31, REQ-COMPILE-35, REQ-SHUTDOWN-4, REQ-LOG-2

```go
func TestCompile_NativeServiceEnvironment(t *testing.T)
```
- **Input:** Service with Service RDL defining topics, plus `external.env` overrides.
- **Expected:** Environment includes `GORAI_SERVICE_NAME`, `INPUT_TOPIC_*`, `OUTPUT_TOPIC_*`, `GORAI_INPUT_TOPICS`, `GORAI_OUTPUT_TOPICS`, and env overrides.
- **Covers:** REQ-COMPILE-32

```go
func TestCompile_NativeServiceLogLevel(t *testing.T)
```
- **Input:** Service with `log_level: "debug"`.
- **Expected:** `LOG_LEVEL=debug` in process environment.
- **Covers:** REQ-LOG-4

#### 3.4.4 Restart Policy Mapping

```go
func TestCompile_RestartPolicyMapping(t *testing.T)
```
Table-driven:

| Subtest | RDL `external.restart` | Expected `availability.restart` |
|---|---|---|
| `always` | `"always"` | `always` |
| `on_failure` | `"on-failure"` | `on_failure` |
| `never` | `"never"` | `no` |
| `absent` | `""` (omitted) | `on_failure` |

- **Covers:** REQ-COMPILE-33

#### 3.4.5 Container Service Processes

```go
func TestCompile_ContainerService(t *testing.T)
```
- **Input:** Service with `external.container.image: "localhost/vision:latest"`, devices, volumes.
- **Expected YAML assertions:**
  - Command starts with `podman run --rm --name <service-name>`
  - Command includes `--network host`
  - Command includes `--device /dev/video0`
  - Command includes `-v ./models:/models:ro`
  - Command includes `-e` for each resolved env var
  - Command ends with the image name
  - `shutdown.command` is `podman stop -t 10 <service-name>`
  - `shutdown.timeout_seconds` is `15`
  - `namespace` is `services`
- **Covers:** REQ-COMPILE-40, REQ-COMPILE-41, REQ-COMPILE-42, REQ-COMPILE-43, REQ-COMPILE-45

```go
func TestCompile_ContainerServiceCustomNetwork(t *testing.T)
```
- **Input:** Container with `network: "bridge"`.
- **Expected:** Command includes `--network bridge` instead of `--network host`.

```go
func TestCompile_ContainerServicePrivileged(t *testing.T)
```
- **Input:** Container with `privileged: true`.
- **Expected:** Command includes `--privileged`.

```go
func TestCompile_ContainerServiceBuildIgnored(t *testing.T)
```
- **Input:** Container with `build` config present.
- **Expected:** No build steps in output. Image referenced by name only.
- **Covers:** REQ-COMPILE-46

```go
func TestCompile_ContainerServiceRestartPolicy(t *testing.T)
```
- **Input:** Container service with various restart values.
- **Expected:** Same mapping as native services (REQ-COMPILE-33), default `on_failure`.
- **Covers:** REQ-COMPILE-44

#### 3.4.6 Device Reset Processes

```go
func TestCompile_DeviceReset(t *testing.T)
```
- **Input:** Device with `id: "pico-1"`, `nats_prefix: "gsp"`, `reset_on_startup: true`.
- **Expected YAML assertions:**
  - Process `device-reset-pico-1` exists
  - `command` is `gorai device reset --nats-prefix gsp --device-id pico-1`
  - `availability.restart` is `no`
  - `depends_on.gorai.condition` is `process_healthy` (embedded NATS case)
  - No `log_location` (transient process)
- **Covers:** REQ-COMPILE-50, REQ-COMPILE-51, REQ-COMPILE-52, REQ-COMPILE-53, REQ-COMPILE-55, REQ-LOG-5

```go
func TestCompile_DeviceResetExternalNATS(t *testing.T)
```
- **Input:** Device reset + external NATS.
- **Expected:** `device-reset-pico-1.depends_on` references `nats-server` with `process_healthy`.
- **Covers:** REQ-COMPILE-53

```go
func TestCompile_DeviceResetNoResetFlag(t *testing.T)
```
- **Input:** Device with `reset_on_startup: false`.
- **Expected:** No `device-reset-*` process emitted.

```go
func TestCompile_ServiceDependsOnDeviceReset(t *testing.T)
```
- **Input:** Service with `attributes.device_id: "pico-1"` referencing a device with `reset_on_startup: true`.
- **Expected:** Service process `depends_on` includes `device-reset-pico-1` with `condition: process_completed`.
- **Covers:** REQ-COMPILE-54

#### 3.4.7 Dependency Translation

```go
func TestCompile_ServiceDependsOnComponent(t *testing.T)
```
- **Input:** Service with `depends_on: ["left_motor"]` where `left_motor` is a component.
- **Expected:** Process `depends_on.gorai.condition` is `process_healthy`.
- **Covers:** REQ-DEP-2

```go
func TestCompile_ServiceDependsOnExternalService(t *testing.T)
```
- **Input:** Service A depends on external service B.
- **Expected:** Process A `depends_on` includes process B with `condition: process_started`.
- **Covers:** REQ-DEP-3

```go
func TestCompile_CircularDependencyRejected(t *testing.T)
```
- **Input:** Service A depends on B, service B depends on A.
- **Expected:** `Compile()` returns error containing "circular dependency".
- **Covers:** REQ-DEP-4

```go
func TestCompile_MultipleComponentDepsCollapse(t *testing.T)
```
- **Input:** Service depends on `["motor_a", "motor_b"]` (both components).
- **Expected:** Single `depends_on.gorai` entry (not duplicated).

#### 3.4.8 Infrastructure Services

```go
func TestCompile_MetricsEnabled(t *testing.T)
```
- **Input:** `metrics: {enabled: true, retention: "14d"}`.
- **Expected YAML assertions:**
  - Process `victoria-metrics` in `infra` namespace
  - Command includes `-retentionPeriod=14d`
  - Command includes `-httpListenAddr=127.0.0.1:8428`
  - Command includes `-storageDataPath=./data/victoria-metrics/`
  - Readiness probe on port 8428, path `/health`
  - `availability.restart` is `always`
  - Global env includes `VICTORIA_METRICS_URL=http://127.0.0.1:8428`
- **Covers:** REQ-INFRA-METRICS-1 through REQ-INFRA-METRICS-5

```go
func TestCompile_MetricsDisabledByDefault(t *testing.T)
```
- **Input:** No `metrics` section.
- **Expected:** No `victoria-metrics` process. No `VICTORIA_METRICS_URL` in global env.

```go
func TestCompile_LoggingEnabled(t *testing.T)
```
- **Input:** `logging: {enabled: true, retention: "7d"}`.
- **Expected YAML assertions:**
  - Process `victoria-logs` in `infra` namespace
  - Command includes `-retentionPeriod=7d`
  - Readiness probe on port 9428, path `/health`
  - Global env includes `VICTORIA_LOGS_URL=http://127.0.0.1:9428`
- **Covers:** REQ-INFRA-LOGS-1 through REQ-INFRA-LOGS-5

```go
func TestCompile_LoggingDisabledByDefault(t *testing.T)
```
- **Input:** No `logging` section.
- **Expected:** No `victoria-logs` process.

#### 3.4.9 Log Configuration

```go
func TestCompile_LogConfiguration(t *testing.T)
```
- **Expected:** Output contains `log_configuration` with `fields_order: [time, level, message]`.
- **Covers:** REQ-LOG-1

```go
func TestCompile_LogRotation(t *testing.T)
```
- **Expected:** All non-transient processes have `log_location` and `log_rotation` with `max_size_mb: 10`, `max_backups: 3`, `max_age_days: 7`, `compress: true`.
- **Covers:** REQ-LOG-2

```go
func TestCompile_TransientProcessNoLogLocation(t *testing.T)
```
- **Input:** Device with `reset_on_startup: true`.
- **Expected:** `device-reset-*` process has no `log_location` or `log_rotation`.
- **Covers:** REQ-LOG-5

#### 3.4.10 Shutdown Configuration

```go
func TestCompile_OrderedShutdown(t *testing.T)
```
- **Expected:** Top-level `ordered_shutdown: true`.
- **Covers:** REQ-SHUTDOWN-1

```go
func TestCompile_GoraiShutdownConfig(t *testing.T)
```
- **Expected:** `gorai.shutdown.signal` is `15`, `timeout_seconds` is `30`.
- **Covers:** REQ-SHUTDOWN-2

```go
func TestCompile_NativeServiceShutdownConfig(t *testing.T)
```
- **Expected:** Native service `shutdown.signal` is `15`, `timeout_seconds` is `10`.
- **Covers:** REQ-SHUTDOWN-4

#### 3.4.11 Topic Environment Variables

```go
func TestCompile_TopicEnvVars(t *testing.T)
```
- **Input:** Service with resolved topics: subscribe `cmd_vel` -> `gorai.bot.cmd_vel`, publish `odom` -> `gorai.bot.odom`.
- **Expected environment:**
  - `INPUT_TOPIC_CMD_VEL=gorai.bot.cmd_vel`
  - `OUTPUT_TOPIC_ODOM=gorai.bot.odom`
  - `GORAI_INPUT_TOPICS=cmd_vel=gorai.bot.cmd_vel`
  - `GORAI_OUTPUT_TOPICS=odom=gorai.bot.odom`
- **Covers:** REQ-COMPILE-32

#### 3.4.12 Full Example Validation

```go
func TestCompile_FullExampleFromRequirements(t *testing.T)
```
- **Input:** The exact RDL from REQUIREMENTS.md Section 13.
- **Expected:** Output structurally matches the example YAML from Section 13. Validate all process names, namespaces, dependencies, environment variables, readiness probes, shutdown blocks, and log locations.
- **Covers:** All compile requirements holistically.

---

### 3.5 Phase 5e: CLI Tests

#### 3.5.1 Compile Command

```go
func TestCompileCommand_WritesFile(t *testing.T)
```
- **Setup:** Create minimal RDL in `t.TempDir()`.
- **Action:** Run compile command with `--output <tempdir>/out.yaml`.
- **Assertions:** File exists, parses as valid YAML, contains `version: "0.5"`.
- **Covers:** REQ-COMPILE-1, REQ-COMPILE-2

```go
func TestCompileCommand_WritesStdout(t *testing.T)
```
- **Action:** Run compile command with `--stdout`.
- **Assertions:** YAML written to captured stdout buffer.
- **Covers:** REQ-COMPILE-2

```go
func TestCompileCommand_DefaultOutputFilename(t *testing.T)
```
- **Action:** Run compile command with no `--output` flag.
- **Assertions:** Writes to `process-compose.yaml` in working directory.
- **Covers:** REQ-COMPILE-2

```go
func TestCompileCommand_ValidationFailure(t *testing.T)
```
- **Input:** Invalid RDL (missing robot name).
- **Expected:** Non-zero exit code, error message on stderr.
- **Covers:** REQ-COMPILE-4

```go
func TestCompileCommand_ValidatesIdenticallyToRun(t *testing.T)
```
- **Input:** RDL with Service RDL references.
- **Expected:** Service RDL merging and topic resolution run during compile (same as `gorai run`).
- **Covers:** REQ-COMPILE-3

#### 3.5.2 Device Reset Command

```go
func TestDeviceResetCommand_PublishesResetMessage(t *testing.T)
```
- **Setup:** Start test NATS server. Subscribe to `gsp.pico-1.tx.system.reset`.
- **Action:** Run `gorai device reset --nats-prefix gsp --device-id pico-1 --nats-url <test-url>`.
- **Assertions:** Subscriber receives the GSP/2 RESET message. Command exits 0.
- **Covers:** REQ-CLI-DEVICE-1, REQ-CLI-DEVICE-2

```go
func TestDeviceResetCommand_ExitZeroOnSuccess(t *testing.T)
```
- **Setup:** Start test NATS server.
- **Action:** Run device reset command.
- **Assertions:** Exit code 0.
- **Covers:** REQ-CLI-DEVICE-2

```go
func TestDeviceResetCommand_ExitNonZeroOnConnectionFailure(t *testing.T)
```
- **Setup:** No NATS server running. Use a port that is not listening.
- **Action:** Run device reset command.
- **Assertions:** Exit code non-zero. Error message on stderr.
- **Covers:** REQ-CLI-DEVICE-3

```go
func TestDeviceResetCommand_RespectsNATSURLEnvVar(t *testing.T)
```
- **Setup:** Start test NATS server on random port. Set `NATS_URL` env var.
- **Action:** Run device reset command without `--nats-url` flag.
- **Assertions:** Connects to the URL from the env var.
- **Covers:** REQ-CLI-DEVICE-4

```go
func TestDeviceResetCommand_DefaultNATSURL(t *testing.T)
```
- **Setup:** No `NATS_URL` env var set.
- **Assertions:** Default URL is `nats://localhost:4222`.
- **Covers:** REQ-CLI-DEVICE-4

---

### 3.6 Phase 5f: Integration Tests

Build tag: `integration`. These tests exercise multiple packages working together.

```go
func TestIntegration_EmbeddedNATSInRobot(t *testing.T)
```
- **Setup:** Minimal RDL with no NATS section.
- **Action:** Start the robot controller (programmatically, not via exec).
- **Assertions:**
  - Embedded NATS starts and is reachable on the configured port.
  - Robot connects as a client and can publish/subscribe.
  - Health endpoint returns 503 during init, 200 after ready.
- **Covers:** REQ-RUN-2, REQ-NATS-EMBED-1 through REQ-NATS-EMBED-4

```go
func TestIntegration_PCProcNameSkipsExternalServices(t *testing.T)
```
- **Setup:** RDL with one external service. Set `PC_PROC_NAME=gorai` in environment.
- **Action:** Start the robot controller.
- **Assertions:** External service is NOT spawned. Log message indicates process-compose manages external services.
- **Covers:** REQ-RUN-6

```go
func TestIntegration_PCProcNameAbsentSpawnsExternalServices(t *testing.T)
```
- **Setup:** RDL with one external service. `PC_PROC_NAME` NOT set.
- **Action:** Start the robot controller (standalone mode).
- **Assertions:** External service spawn is attempted (or at least the code path enters the external service manager).
- **Covers:** REQ-RUN-1, REQ-RUN-5

```go
func TestIntegration_HealthEndpointLifecycle(t *testing.T)
```
- **Setup:** RDL with components.
- **Action:** Start controller. Poll health endpoint.
- **Assertions:**
  - `/livez` returns 200 immediately.
  - `/healthz` returns 503 initially.
  - `/healthz` transitions to 200 after `EventRobotReady`.
- **Covers:** REQ-CLI-HEALTH-1 through REQ-CLI-HEALTH-3, REQ-COMPILE-61

```go
func TestIntegration_CompileProducesValidYAML(t *testing.T)
```
- **Setup:** RDL from REQUIREMENTS.md Section 13 example.
- **Action:** Run the full compile pipeline.
- **Assertions:** Output is valid YAML. All processes present. Dependencies form a DAG.
- **Covers:** REQ-TEST-5 (partial)

```go
func TestIntegration_ShutdownOrder(t *testing.T)
```
- **Setup:** Start controller with embedded NATS and components.
- **Action:** Send SIGTERM.
- **Assertions (via log capture or event ordering):**
  - `EventRobotShutdown` published.
  - Services closed before components.
  - Components closed before NATS client.
  - NATS client closed before embedded server.
  - Exit code 0.
- **Covers:** REQ-SHUTDOWN-5

---

## 4. Edge Cases

### 4.1 Config Edge Cases

| Test | Input | Expected |
|---|---|---|
| `TestCompile_EmptyServicesArray` | `"services": []` | No service processes emitted |
| `TestCompile_DisabledService` | Service with `disabled: true` and `external.enabled: true` | Service process NOT emitted |
| `TestCompile_DisabledComponent` | Component with `disabled: true` | Component still in gorai controller but skipped in init |
| `TestCompile_RobotNameWithHyphens` | `robot.name: "my-robot"` | All env vars and process names use the hyphenated name |
| `TestCompile_NamespaceOverride` | `robot.namespace: "custom-ns"` | `GORAI_NAMESPACE=custom-ns` (not robot name) |
| `TestCompile_NoDevices` | No devices section | No `device-reset-*` processes |
| `TestCompile_MultipleDevices` | Two devices with `reset_on_startup: true` | Two separate reset processes, each with correct NATS prefix and device ID |

### 4.2 NATS Edge Cases

| Test | Input | Expected |
|---|---|---|
| `TestEmbeddedNATS_PortAlreadyInUse` | Start two servers on same port | Second start returns error, does not panic |
| `TestEmbeddedNATS_ShutdownIdempotent` | Call `Shutdown()` twice | No panic, no error on second call |
| `TestEmbeddedNATS_ShutdownWithNoClients` | Start server, no clients connected, shutdown | Clean shutdown, no hang |

### 4.3 Health Edge Cases

| Test | Input | Expected |
|---|---|---|
| `TestHealthServer_ConcurrentRequests` | 100 concurrent `GET /healthz` | All return consistent result, no race |
| `TestHealthServer_SetReadyIdempotent` | Call `SetReady()` twice | No panic, still returns 200 |
| `TestHealthServer_ShutdownWhileServing` | Shutdown server during active request | Graceful termination |

### 4.4 Compiler Edge Cases

| Test | Input | Expected |
|---|---|---|
| `TestCompile_ServiceNameConflictsWithGorai` | Service named `"gorai"` | Error: reserved process name |
| `TestCompile_ServiceNameConflictsWithNATSServer` | Service named `"nats-server"` (with external NATS) | Error: name conflict with infrastructure process |
| `TestCompile_DeviceResetNameCollision` | Device with `id: "gorai"` | Reset process name `device-reset-gorai` does not conflict |
| `TestCompile_ContainerEnvPassthrough` | Container with global + service + container-specific env | Env precedence: container-specific > service > global |
| `TestCompile_VolumePathsPreserved` | Volumes with `:ro`, `:rw`, `:z` suffixes | Suffixes preserved exactly in podman command |
| `TestCompile_EmptyExternalCommand` | `external.enabled: true` but no `command` and no `container` | Validation error |
| `TestCompile_ServiceRDLMergingDuringCompile` | Service referencing a `.rdl` file | Topics resolved, type/model merged before compilation |

### 4.5 Dependency Edge Cases

| Test | Input | Expected |
|---|---|---|
| `TestCompile_SelfDependency` | Service depends on itself | Circular dependency error |
| `TestCompile_DeepDependencyChain` | A -> B -> C -> D (4 levels) | All `depends_on` entries emitted correctly |
| `TestCompile_MixedComponentAndServiceDeps` | Service depends on [component, external service] | Two `depends_on` entries: gorai (process_healthy) + service (process_started) |
| `TestCompile_InternalServiceDependency` | External service depends on internal (non-external) service | Translated to dependency on `gorai` process |

---

## 5. Coverage Targets

| Package | Minimum Line Coverage | Rationale |
|---|---|---|
| `pkg/config/` (new fields only) | 90% | Schema parsing is critical; all code paths must be tested |
| `pkg/natsembed/` | 85% | Core infrastructure; error paths hard to test (port conflicts, TLS) |
| `pkg/health/` | 95% | Small surface area; all paths testable |
| `pkg/compiler/` | 90% | Core feature; every code path maps to a requirement |
| `cmd/gorai/commands/` (compile, device reset) | 80% | CLI wiring; some paths require process execution |
| `tests/integration/` | N/A | Integration tests measure behavior, not coverage |

Overall target for new code: **85% line coverage**.

Generate per-package coverage with:

```bash
make test-cover
```

---

## 6. Test Infrastructure

### 6.1 ConfigBuilder

Located at `internal/testutil/config_builder.go`. Fluent API for constructing `config.RDL` structs:

```go
cfg := testutil.NewConfigBuilder().
    WithName("scout").
    WithNamespace("ns").
    WithNATS("nats://localhost:4222", false).     // url, external
    WithJetStream(true).
    WithDevice("pico-1", "gsp", true).            // id, prefix, reset
    WithComponent("motor_a", "motor", "remote").
    WithExternalService("navigator", "/usr/bin/nav", []string{"--mode", "wp"}).
    WithContainerService("detector", "localhost/vision:latest").
    WithMetrics(true, "14d", "").
    WithLogging(true, "3d", "").
    Build()
```

### 6.2 YAMLAssert

Located at `internal/testutil/yaml_assert.go`. Helpers for asserting on YAML output without fragile string matching:

```go
// Parse YAML output into a map
doc := testutil.ParseYAML(t, yamlBytes)

// Assert path exists and has value
testutil.AssertYAMLString(t, doc, "version", "0.5")
testutil.AssertYAMLBool(t, doc, "ordered_shutdown", true)
testutil.AssertYAMLInt(t, doc, "processes.gorai.readiness_probe.http_get.port", 4180)
testutil.AssertYAMLContains(t, doc, "processes.gorai.command", "gorai run")
testutil.AssertYAMLExists(t, doc, "processes.victoria-metrics")
testutil.AssertYAMLNotExists(t, doc, "processes.victoria-logs")

// Assert on list values in environment
testutil.AssertYAMLListContains(t, doc, "environment", "GORAI_ROBOT_NAME=scout")
```

### 6.3 TestNATSServer

Located at `internal/testutil/nats_helper.go`:

```go
// Start returns a running NATS server and its client URL.
// Automatically cleaned up when the test ends.
ns := testutil.StartTestNATS(t)          // plain
ns := testutil.StartTestNATSWithJS(t)    // with JetStream
url := ns.ClientURL()                    // "nats://127.0.0.1:<random-port>"
```

### 6.4 FreePort and WaitForHTTP

```go
port := testutil.FreePort(t)
testutil.WaitForHTTP(t, "http://127.0.0.1:"+strconv.Itoa(port)+"/livez", 200, 5*time.Second)
```

---

## 7. Integration Test Plan (REQ-TEST-5)

REQ-TEST-5 specifies end-to-end pipeline validation. These are the most expensive tests and run only with the `integration` or `system` build tag.

### 7.1 Compile-and-Validate Pipeline

```go
// +build integration

func TestPipeline_CompileProducesValidProcessCompose(t *testing.T)
```
1. Load the full example RDL from Section 13 of REQUIREMENTS.md.
2. Call `compiler.Compile()`.
3. Write output to temp file.
4. Validate YAML structure: all required top-level keys, all processes, all dependencies.
5. Verify the YAML is parseable by a process-compose YAML schema validator (or at minimum, standard YAML parsing with structure checks).

### 7.2 Embedded NATS Full Lifecycle

```go
// +build integration

func TestPipeline_EmbeddedNATSFullLifecycle(t *testing.T)
```
1. Start the robot controller programmatically with embedded NATS.
2. Verify NATS is accepting connections.
3. Verify health endpoint transitions from 503 to 200.
4. Publish and subscribe to a test topic through the embedded NATS.
5. Send SIGTERM.
6. Verify shutdown completes within 30 seconds.
7. Verify NATS port is released.

### 7.3 Device Reset Sequencing

```go
// +build integration

func TestPipeline_DeviceResetSequencing(t *testing.T)
```
1. Compile an RDL with a device that has `reset_on_startup: true` and a service that depends on that device.
2. Verify the compiled YAML has:
   - `device-reset-*` depends on NATS being healthy.
   - Service depends on `device-reset-*` with `process_completed`.
3. Verify topological sort of the dependency graph matches expected order: NATS -> device-reset -> service.

### 7.4 Health Probe Integration

```go
// +build integration

func TestPipeline_HealthProbeMatchesCompilerOutput(t *testing.T)
```
1. Compile an RDL.
2. Extract the health probe config from the YAML (port, path).
3. Start the robot controller with the same config.
4. Verify the health endpoint is reachable at the port/path specified in the compiled output.

### 7.5 System Test: Process-Compose Up/Down (Manual or CI-gated)

```go
// +build system

func TestSystem_ProcessComposeUpDown(t *testing.T)
```
1. Check `process-compose` is on PATH; skip if not.
2. Compile the minimal RDL.
3. Run `process-compose up -f <file> &` in background.
4. Wait for health probe to return 200.
5. Send `process-compose down`.
6. Verify all processes exited cleanly.
7. Verify ordered shutdown (gorai exits last among core processes).

This test is gated on the `system` build tag and the presence of `process-compose` on PATH. It runs in CI only when explicitly enabled.

---

## 8. Requirement Traceability Matrix

| Requirement | Test(s) |
|---|---|
| REQ-NATS-EMBED-1 | `TestEmbeddedNATS_StartAndAcceptConnections` |
| REQ-NATS-EMBED-2 | `TestEmbeddedNATS_StartAndAcceptConnections` |
| REQ-NATS-EMBED-3 | `TestEmbeddedNATS_ReadyBeforeReturn` |
| REQ-NATS-EMBED-4 | `TestEmbeddedNATS_ClientConnectsAsRegularClient` |
| REQ-NATS-EMBED-5 | `TestEmbeddedNATS_JetStreamEnabled` |
| REQ-NATS-EMBED-6 | `TestEmbeddedNATS_TLSConfiguration` |
| REQ-NATS-EMBED-7 | `TestEmbeddedNATS_ShutdownDrainsConnections` |
| REQ-NATS-EXT-1 | `TestCompile_ExternalNATS` |
| REQ-NATS-EXT-2 | `TestCompile_ExternalNATS` |
| REQ-NATS-EXT-3 | `TestCompile_ExternalNATS` |
| REQ-NATS-EXT-4 | `TestCompile_ExternalNATS` |
| REQ-NATS-EXT-5 | `TestCompile_ExternalNATSWithJetStream` |
| REQ-COMPILE-1 | `TestCompileCommand_WritesFile` |
| REQ-COMPILE-2 | `TestCompileCommand_WritesFile`, `TestCompileCommand_WritesStdout`, `TestCompileCommand_DefaultOutputFilename` |
| REQ-COMPILE-3 | `TestCompileCommand_ValidatesIdenticallyToRun` |
| REQ-COMPILE-4 | `TestCompileCommand_ValidationFailure` |
| REQ-COMPILE-10 | `TestCompile_MinimalRDL` |
| REQ-COMPILE-11 | `TestCompile_MinimalRDL` |
| REQ-COMPILE-12 | `TestCompile_OrderedShutdown` |
| REQ-COMPILE-20 | `TestCompile_MinimalRDL` |
| REQ-COMPILE-21 | `TestCompile_MinimalRDL` |
| REQ-COMPILE-22 | `TestCompile_MinimalRDL` |
| REQ-COMPILE-23 | `TestCompile_ExternalNATS` |
| REQ-COMPILE-30 | `TestCompile_NativeService` |
| REQ-COMPILE-31 | `TestCompile_NativeService` |
| REQ-COMPILE-32 | `TestCompile_NativeServiceEnvironment`, `TestCompile_TopicEnvVars` |
| REQ-COMPILE-33 | `TestCompile_RestartPolicyMapping` |
| REQ-COMPILE-34 | `TestCompile_ServiceDependsOnComponent`, `TestCompile_ServiceDependsOnExternalService` |
| REQ-COMPILE-35 | `TestCompile_NativeService` |
| REQ-COMPILE-40 | `TestCompile_ContainerService` |
| REQ-COMPILE-41 | `TestCompile_ContainerService` |
| REQ-COMPILE-42 | `TestCompile_ContainerService`, `TestCompile_ContainerServiceCustomNetwork`, `TestCompile_ContainerServicePrivileged` |
| REQ-COMPILE-43 | `TestCompile_ContainerService` |
| REQ-COMPILE-44 | `TestCompile_ContainerServiceRestartPolicy` |
| REQ-COMPILE-45 | `TestCompile_ContainerService` |
| REQ-COMPILE-46 | `TestCompile_ContainerServiceBuildIgnored` |
| REQ-COMPILE-50 | `TestCompile_DeviceReset` |
| REQ-COMPILE-51 | `TestCompile_DeviceReset` |
| REQ-COMPILE-52 | `TestCompile_DeviceReset` |
| REQ-COMPILE-53 | `TestCompile_DeviceReset`, `TestCompile_DeviceResetExternalNATS` |
| REQ-COMPILE-54 | `TestCompile_ServiceDependsOnDeviceReset` |
| REQ-COMPILE-55 | `TestCompile_DeviceReset` |
| REQ-COMPILE-60 | `TestCompile_MinimalRDL` |
| REQ-COMPILE-61 | `TestHealthServer_503BeforeReady`, `TestHealthServer_200AfterReady` |
| REQ-COMPILE-62 | `TestCompile_MinimalRDL` |
| REQ-COMPILE-63 | `TestCompile_ExternalNATS` |
| REQ-CLI-DEVICE-1 | `TestDeviceResetCommand_PublishesResetMessage` |
| REQ-CLI-DEVICE-2 | `TestDeviceResetCommand_PublishesResetMessage`, `TestDeviceResetCommand_ExitZeroOnSuccess` |
| REQ-CLI-DEVICE-3 | `TestDeviceResetCommand_ExitNonZeroOnConnectionFailure` |
| REQ-CLI-DEVICE-4 | `TestDeviceResetCommand_RespectsNATSURLEnvVar`, `TestDeviceResetCommand_DefaultNATSURL` |
| REQ-CLI-HEALTH-1 | `TestHealthServer_ConfigurableListen` |
| REQ-CLI-HEALTH-2 | `TestHealthServer_503BeforeReady`, `TestHealthServer_200AfterReady`, `TestHealthServer_LivezAlways200` |
| REQ-CLI-HEALTH-3 | `TestHealthServer_StartsBeforeComponents` |
| REQ-INFRA-METRICS-1..5 | `TestCompile_MetricsEnabled` |
| REQ-INFRA-LOGS-1..5 | `TestCompile_LoggingEnabled` |
| REQ-LOG-1 | `TestCompile_LogConfiguration` |
| REQ-LOG-2 | `TestCompile_LogRotation` |
| REQ-LOG-4 | `TestCompile_NativeServiceLogLevel` |
| REQ-LOG-5 | `TestCompile_TransientProcessNoLogLocation` |
| REQ-SHUTDOWN-1 | `TestCompile_OrderedShutdown` |
| REQ-SHUTDOWN-2 | `TestCompile_GoraiShutdownConfig` |
| REQ-SHUTDOWN-3 | `TestCompile_ContainerService` |
| REQ-SHUTDOWN-4 | `TestCompile_NativeServiceShutdownConfig` |
| REQ-SHUTDOWN-5 | `TestIntegration_ShutdownOrder` |
| REQ-DEP-1 | `TestCompile_ServiceDependsOnComponent`, `TestCompile_ServiceDependsOnExternalService` |
| REQ-DEP-2 | `TestCompile_ServiceDependsOnComponent` |
| REQ-DEP-3 | `TestCompile_ServiceDependsOnExternalService` |
| REQ-DEP-4 | `TestCompile_CircularDependencyRejected` |
| REQ-RUN-1 | `TestIntegration_PCProcNameAbsentSpawnsExternalServices` |
| REQ-RUN-2 | `TestIntegration_EmbeddedNATSInRobot` |
| REQ-RUN-3 | `TestIntegration_HealthEndpointLifecycle` |
| REQ-RUN-6 | `TestIntegration_PCProcNameSkipsExternalServices` |
| REQ-TEST-1 | Phase 5d compiler tests (all) |
| REQ-TEST-2 | Phase 5b embedded NATS tests (all) |
| REQ-TEST-3 | Phase 5e device reset tests (all) |
| REQ-TEST-4 | Phase 5c health server tests (all) |
| REQ-TEST-5 | Phase 5f integration tests + Section 7 pipeline tests |
