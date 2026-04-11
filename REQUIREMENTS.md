# Gorai Process Compose Runtime — Requirements

**Version:** 1.0
**Date:** 2026-04-11
**Status:** Draft

## 1. Overview

This document specifies requirements for replacing Gorai's current in-process orchestration and archived Quadlet/systemd generation with a **process-compose-based runtime**. The gorai controller becomes a compiler that transforms RDL configuration into a `process-compose.yaml` file, and an embedded NATS server provides zero-configuration messaging for single-robot deployments.

### 1.1 Goals

1. **Simplify the runtime model.** The minimum robot is one process (gorai with embedded NATS). All other processes are additive.
2. **Delegate process lifecycle to process-compose.** Gorai no longer manages child processes, restart policies, signal handling, or log capture directly.
3. **Support mixed workloads.** Native Go binaries and containerized services (any language, via podman) are managed uniformly by process-compose.
4. **Preserve the single-binary development story.** `gorai run robot.json` remains the simplest path — it compiles and launches in one step.
5. **Make observability optional.** VictoriaMetrics, VictoriaLogs, and the web dashboard are opt-in services, not default infrastructure.

### 1.2 Non-Goals

1. Fleet management or multi-node orchestration (future phase).
2. Kubernetes/K3s deployment target (future phase; this architecture is a stepping stone toward it).
3. Changes to RDL v2 schema beyond the additions specified in this document.
4. Changes to the component registry, Service RDL, or NATS topic resolution — those systems remain as-is.

### 1.3 Definitions

| Term | Meaning |
|---|---|
| **RDL** | Robot Definition Language — the `robot.json` configuration file |
| **Service binary** | A standalone executable that imports gorai component packages as Go libraries and communicates via NATS |
| **Container service** | A service packaged as an OCI container image, run via podman in foreground mode |
| **Embedded NATS** | NATS server running in-process within the gorai controller using `github.com/nats-io/nats-server/v2/server` |
| **External NATS** | A NATS server running as a separate process, managed independently or by process-compose |
| **Compile** | The act of transforming an RDL file into a `process-compose.yaml` file |

---

## 2. Architecture

### 2.1 Deployment Model

```
robot.json (RDL)
      |
      v
gorai compile --> process-compose.yaml
      |
      v
process-compose up
      |
      +-- gorai controller (with embedded NATS, or client-only)
      +-- [service-a]          (native binary, optional)
      +-- [service-b]          (podman container, optional)
      +-- [victoria-metrics]   (optional)
      +-- [victoria-logs]      (optional)
      +-- [dashboard]          (optional)
```

### 2.2 Minimum Robot

The simplest possible robot consists of a single process managed by process-compose:

```yaml
version: "0.5"
processes:
  gorai:
    command: "gorai run robot.json"
    availability:
      restart: on_failure
```

The gorai controller embeds the NATS server, initializes components (in-process via the registry), and runs services. No other processes are required.

### 2.3 Process Namespaces

Process-compose processes are organized into namespaces for clarity:

| Namespace | Contents |
|---|---|
| `infra` | NATS server (when external), VictoriaMetrics, VictoriaLogs |
| `core` | gorai controller |
| `services` | User-defined service binaries and container services |

When the gorai controller embeds NATS and no optional infrastructure is enabled, namespaces may be omitted (single process, no grouping needed).

---

## 3. Embedded NATS

### 3.1 Automatic Mode Selection

The gorai controller determines NATS mode from the RDL configuration:

| RDL Configuration | Behavior |
|---|---|
| No `nats` section | Embed NATS server, listen on `nats://localhost:4222` |
| `nats.url` is `localhost` or `127.0.0.1` and `nats.external` is absent or `false` | Embed NATS server, listen on configured URL |
| `nats.url` is `localhost` or `127.0.0.1` and `nats.external` is `true` | Do not embed. Connect as client only. Compiler emits a separate NATS process. |
| `nats.url` points to a non-local address | Do not embed. Connect as client only. Compiler does not emit a NATS process. |

### 3.2 RDL Schema Addition

Add one field to `NATSConfig`:

```go
type NATSConfig struct {
    // ... existing fields ...
    External bool `json:"external,omitempty"` // Force NATS to run as a separate process
}
```

**Default:** `false`

**Validation:** If `external` is `true` and `url` is non-local, emit a warning (external flag is redundant when URL is already remote).

### 3.3 Embedded Server Requirements

REQ-NATS-EMBED-1
: The gorai controller MUST embed the NATS server using `github.com/nats-io/nats-server/v2/server` as a Go library.

REQ-NATS-EMBED-2
: The embedded NATS server MUST listen on the port specified in `nats.url` (default 4222) and accept connections from external processes (services running alongside the controller).

REQ-NATS-EMBED-3
: The embedded NATS server MUST be fully started and accepting connections before the controller initializes any components or services.

REQ-NATS-EMBED-4
: The gorai controller MUST connect to the embedded NATS server as a regular client using the standard `gorainats.Client` — no in-process shortcuts. This ensures identical behavior between embedded and external modes.

REQ-NATS-EMBED-5
: JetStream MUST be enabled on the embedded server when `nats.jetstream` is `true` in the RDL. Storage directory defaults to `./data/jetstream/` relative to the working directory.

REQ-NATS-EMBED-6
: When `nats.tls` is configured, the embedded server MUST use the same TLS configuration for its listener.

REQ-NATS-EMBED-7
: On shutdown, the embedded NATS server MUST drain all connections and shut down after the controller has closed its own client connection.

### 3.4 External NATS Process

REQ-NATS-EXT-1
: When `nats.external` is `true` and the URL is local, the compiler MUST emit a `nats-server` process in the `infra` namespace.

REQ-NATS-EXT-2
: The emitted NATS process MUST include a readiness probe that checks TCP connectivity on the NATS port.

REQ-NATS-EXT-3
: All other processes MUST declare `depends_on` the NATS process with `condition: process_healthy`.

REQ-NATS-EXT-4
: The NATS process MUST have `availability.restart: always`.

REQ-NATS-EXT-5
: If `nats.jetstream` is `true`, the emitted NATS process command MUST include JetStream configuration flags (storage directory, max storage).

---

## 4. Compile Command

### 4.1 CLI Interface

REQ-COMPILE-1
: Add a new CLI command: `gorai compile <config> [flags]`

REQ-COMPILE-2
: The command MUST accept the following flags:

| Flag | Default | Description |
|---|---|---|
| `--output`, `-o` | `process-compose.yaml` | Output file path |
| `--stdout` | `false` | Write to stdout instead of file |

REQ-COMPILE-3
: The command MUST load and validate the RDL configuration identically to `gorai run` (including Service RDL merging, topic resolution, and all validation rules).

REQ-COMPILE-4
: The command MUST fail with a non-zero exit code and descriptive error if validation fails.

### 4.2 Compilation Rules

REQ-COMPILE-10
: The compiler MUST emit a valid process-compose YAML file (version `"0.5"`).

REQ-COMPILE-11
: The compiler MUST emit global environment variables shared by all processes:

| Variable | Value |
|---|---|
| `GORAI_ROBOT_NAME` | `robot.name` |
| `GORAI_NAMESPACE` | `robot.namespace` (or `robot.name` if absent) |
| `NATS_URL` | Effective NATS URL |

REQ-COMPILE-12
: The compiler MUST emit `ordered_shutdown: true` so processes stop in reverse dependency order.

### 4.3 Gorai Controller Process

REQ-COMPILE-20
: The compiler MUST always emit a `gorai` process in the `core` namespace.

REQ-COMPILE-21
: The gorai process command MUST be `gorai run <config-path>`.

REQ-COMPILE-22
: The gorai process MUST have `availability.restart: on_failure`.

REQ-COMPILE-23
: When NATS is external (separate process), the gorai process MUST declare `depends_on` the NATS process with `condition: process_healthy`.

### 4.4 Service Processes (Native Binaries)

REQ-COMPILE-30
: For each service where `external.enabled` is `true`, `external.command` is set, and `external.container` is absent, the compiler MUST emit a process entry.

REQ-COMPILE-31
: The process command MUST be the value of `external.command` with `external.args` appended.

REQ-COMPILE-32
: The process environment MUST include the resolved environment from `GetResolvedEnvironment()` — including `GORAI_SERVICE_NAME`, resolved topic variables (`INPUT_TOPIC_*`, `OUTPUT_TOPIC_*`, `GORAI_INPUT_TOPICS`, `GORAI_OUTPUT_TOPICS`), and any `external.env` overrides.

REQ-COMPILE-33
: The process `availability.restart` MUST map from `external.restart`:

| RDL Value | process-compose Value |
|---|---|
| `"always"` | `always` |
| `"on-failure"` | `on_failure` |
| `"never"` | `no` |
| (absent) | `on_failure` |

REQ-COMPILE-34
: If the service has `depends_on`, the compiler MUST emit corresponding process-compose `depends_on` entries. Dependencies on components (which run inside the gorai controller) MUST be translated to a dependency on the `gorai` process with `condition: process_healthy`.

REQ-COMPILE-35
: Service processes MUST be placed in the `services` namespace.

### 4.5 Service Processes (Containers)

REQ-COMPILE-40
: For each service where `external.enabled` is `true` and `external.container` is configured, the compiler MUST emit a process that runs the container via podman.

REQ-COMPILE-41
: The process command MUST use `podman run --rm --name <service-name>` (foreground mode, auto-remove on exit). This ensures process-compose's lifecycle management aligns with the container lifecycle.

REQ-COMPILE-42
: The emitted podman command MUST include:
- `--network host` (default) or the configured `container.network`
- `-e KEY=VALUE` for each resolved environment variable
- `-v SOURCE:TARGET` for each entry in `container.volumes`
- `--device PATH` for each entry in `container.devices`
- `--privileged` if `container.privileged` is `true`
- The container `image` as the final argument

REQ-COMPILE-43
: The process MUST include a `shutdown` block:

```yaml
shutdown:
  command: "podman stop -t 10 <service-name>"
  timeout_seconds: 15
```

REQ-COMPILE-44
: The process `availability.restart` MUST follow the same mapping as native binary services (REQ-COMPILE-33). Default for container services: `on_failure`.

REQ-COMPILE-45
: Container service processes MUST be placed in the `services` namespace.

REQ-COMPILE-46
: If `container.build` is configured, the compiler MUST NOT include build steps in the process-compose output. Building is the responsibility of `gorai build`, which runs before `gorai compile`.

### 4.6 Device Reset Processes

REQ-COMPILE-50
: For each device in `devices[]` with `reset_on_startup: true`, the compiler MUST emit a transient process that performs the device reset.

REQ-COMPILE-51
: The reset process command MUST publish the GSP/2 RESET message to NATS topic `<nats_prefix>.<device_id>.tx.system.reset` and wait 500ms for the device to enter listening state.

REQ-COMPILE-52
: The reset process MUST have `availability.restart: no` (run once).

REQ-COMPILE-53
: The reset process MUST depend on NATS being available (either the gorai controller with embedded NATS, or the external NATS process, with `condition: process_healthy`).

REQ-COMPILE-54
: Services and components that reference the device (via `attributes.device_id`) MUST depend on the reset process with `condition: process_completed`.

REQ-COMPILE-55
: The reset process command SHOULD be `gorai device reset --nats-prefix <prefix> --device-id <id>`. This requires a new `gorai device reset` subcommand (see Section 7).

### 4.7 Health Probes

REQ-COMPILE-60
: The gorai controller process MUST expose a health endpoint for process-compose readiness probes. The endpoint MUST be an HTTP GET on a configurable port (default: `127.0.0.1:4180`).

REQ-COMPILE-61
: The health endpoint MUST return HTTP 200 only when:
- The NATS connection is established (embedded or external)
- All non-disabled components have been initialized
- The robot has published `EventRobotReady`

REQ-COMPILE-62
: The compiler MUST emit a `readiness_probe` for the gorai controller process:

```yaml
readiness_probe:
  http_get:
    host: 127.0.0.1
    port: 4180
    path: /healthz
  initial_delay_seconds: 1
  period_seconds: 5
  failure_threshold: 3
```

REQ-COMPILE-63
: For external NATS, the compiler MUST emit a readiness probe that checks TCP connectivity on the NATS port:

```yaml
readiness_probe:
  exec:
    command: "nats-server --signal check=nats://localhost:4222"
  initial_delay_seconds: 1
  period_seconds: 2
  failure_threshold: 5
```

---

## 5. Optional Infrastructure Services

All infrastructure services are disabled by default. They are enabled via top-level RDL configuration sections.

### 5.1 RDL Schema Additions

Add two new optional top-level sections to the RDL:

```go
type RDL struct {
    // ... existing fields ...
    Metrics  *MetricsConfig  `json:"metrics,omitempty"`
    Logging  *LoggingConfig  `json:"logging,omitempty"`
}

type MetricsConfig struct {
    Enabled   bool   `json:"enabled"`
    Retention string `json:"retention,omitempty"` // Duration, e.g., "7d", "30d". Default: "7d"
    Listen    string `json:"listen,omitempty"`    // Address:port. Default: "127.0.0.1:8428"
}

type LoggingConfig struct {
    Enabled   bool   `json:"enabled"`
    Retention string `json:"retention,omitempty"` // Duration, e.g., "3d", "14d". Default: "3d"
    Listen    string `json:"listen,omitempty"`    // Address:port. Default: "127.0.0.1:9428"
}
```

### 5.2 VictoriaMetrics

REQ-INFRA-METRICS-1
: When `metrics.enabled` is `true`, the compiler MUST emit a VictoriaMetrics process in the `infra` namespace.

REQ-INFRA-METRICS-2
: The VictoriaMetrics process MUST use the single-node binary (`victoria-metrics`) with flags:
- `-retentionPeriod=<metrics.retention>` (default `7d`)
- `-httpListenAddr=<metrics.listen>` (default `127.0.0.1:8428`)
- `-storageDataPath=./data/victoria-metrics/`

REQ-INFRA-METRICS-3
: The VictoriaMetrics process MUST include a readiness probe on its HTTP listen port, path `/health`.

REQ-INFRA-METRICS-4
: The VictoriaMetrics process MUST have `availability.restart: always`.

REQ-INFRA-METRICS-5
: When metrics are enabled, the compiler MUST add `VICTORIA_METRICS_URL=http://<metrics.listen>` to the global environment so services can push metrics.

### 5.3 VictoriaLogs

REQ-INFRA-LOGS-1
: When `logging.enabled` is `true`, the compiler MUST emit a VictoriaLogs process in the `infra` namespace.

REQ-INFRA-LOGS-2
: The VictoriaLogs process MUST use the `victoria-logs` binary with flags:
- `-retentionPeriod=<logging.retention>` (default `3d`)
- `-httpListenAddr=<logging.listen>` (default `127.0.0.1:9428`)
- `-storageDataPath=./data/victoria-logs/`

REQ-INFRA-LOGS-3
: The VictoriaLogs process MUST include a readiness probe on its HTTP listen port, path `/health`.

REQ-INFRA-LOGS-4
: The VictoriaLogs process MUST have `availability.restart: always`.

REQ-INFRA-LOGS-5
: When logging is enabled, the compiler MUST add `VICTORIA_LOGS_URL=http://<logging.listen>` to the global environment so services can ship logs.

### 5.4 Dashboard

REQ-INFRA-DASH-1
: When `dashboard.enabled` is `true` (existing RDL field), the dashboard SHOULD continue to run inside the gorai controller process (current behavior). No separate process-compose process is needed.

REQ-INFRA-DASH-2
: If in a future iteration the dashboard becomes a standalone binary, the compiler MUST emit it as a process in the `infra` namespace with a readiness probe on its HTTP listen port.

---

## 6. Process Compose Log Management

Process-compose provides built-in log capture and rotation, eliminating the need for VictoriaLogs in simple deployments.

### 6.1 Log Configuration in Compiled Output

REQ-LOG-1
: The compiler MUST emit a `log_configuration` section in the process-compose output:

```yaml
log_configuration:
  fields_order:
    - time
    - level
    - message
  disable_json: false
  no_metadata: false
```

REQ-LOG-2
: The compiler MUST emit per-process `log_location` with rotation settings for all non-transient processes:

```yaml
processes:
  gorai:
    log_location: "./logs/gorai.log"
    log_rotation:
      max_size_mb: 10
      max_backups: 3
      max_age_days: 7
      compress: true
```

REQ-LOG-3
: The log directory MUST default to `./logs/` relative to the working directory.

REQ-LOG-4
: The RDL `log.level` field MUST be passed to the gorai controller process. Per-service log levels from `service.log_level` MUST be passed via the `LOG_LEVEL` environment variable.

REQ-LOG-5
: Transient processes (device reset) MUST NOT have `log_location` configured — their output appears only in the process-compose TUI.

---

## 7. New CLI Commands

### 7.1 `gorai device reset`

REQ-CLI-DEVICE-1
: Add a new CLI command: `gorai device reset --nats-prefix <prefix> --device-id <id> [--nats-url <url>]`

REQ-CLI-DEVICE-2
: The command MUST connect to NATS, publish the GSP/2 RESET message to `<nats_prefix>.<device_id>.tx.system.reset`, wait 500ms, then exit with code 0.

REQ-CLI-DEVICE-3
: If the NATS connection fails or the publish fails, the command MUST exit with a non-zero code and print the error to stderr.

REQ-CLI-DEVICE-4
: The `--nats-url` flag MUST default to the `NATS_URL` environment variable, falling back to `nats://localhost:4222`.

### 7.2 `gorai health`

REQ-CLI-HEALTH-1
: The gorai controller MUST start an HTTP health server on the address specified by a new `--health-listen` flag (default: `127.0.0.1:4180`).

REQ-CLI-HEALTH-2
: The health server MUST expose:
- `GET /healthz` — returns 200 when the controller is ready, 503 otherwise
- `GET /livez` — returns 200 when the process is running (always 200 after startup)

REQ-CLI-HEALTH-3
: The health server MUST start before component/service initialization so that process-compose can begin probing immediately.

---

## 8. Changes to `gorai run`

### 8.1 Behavior

REQ-RUN-1
: `gorai run <config>` MUST remain functional as a standalone command that runs the robot in the foreground without process-compose.

REQ-RUN-2
: When running standalone, `gorai run` MUST embed the NATS server according to the rules in Section 3.1.

REQ-RUN-3
: When running standalone, `gorai run` MUST start the health server (Section 7.2) to support process-compose readiness probes when the controller is launched by process-compose.

REQ-RUN-4
: `gorai run` MUST continue to manage in-process components via the existing registry pattern. Components are Go library code initialized inside the controller process — they are not separate processes.

REQ-RUN-5
: `gorai run` MUST continue to manage internal services (non-external, registered in the service registry) in-process.

REQ-RUN-6
: `gorai run` MUST NOT manage external services when running under process-compose. External services are process-compose's responsibility. Detection: if the `PC_PROC_NAME` environment variable is set (auto-injected by process-compose), the controller MUST skip spawning external services and log a message indicating that process-compose manages them.

### 8.2 Convenience Mode

REQ-RUN-10
: Add a new flag: `gorai run --compose` (or `gorai up` as an alias).

REQ-RUN-11
: When `--compose` is specified, the command MUST:
1. Compile the RDL to a temporary `process-compose.yaml`
2. Exec into `process-compose up -f <temp-file>`

This provides a single-command experience equivalent to `gorai compile && process-compose up`.

---

## 9. Changes to `gorai build`

REQ-BUILD-1
: `gorai build` MUST continue to build container images via podman for services with `external.container.build` configured.

REQ-BUILD-2
: `gorai build` MUST run before `gorai compile`. The compile step does not build images — it references images by name.

REQ-BUILD-3
: `gorai build` MUST validate that all referenced container images exist locally after building. If an image is referenced but has no build config and is not present locally, emit a warning.

---

## 10. Dependency Graph

The compiler generates a dependency graph within the process-compose output:

```
nats-server (if external)
  |
  +-- device-reset-* (transient, depends_on: nats healthy)
  |     |
  |     +-- [services that use the device]
  |
  +-- gorai (depends_on: nats healthy)
  |     |
  |     +-- [services that depend on components]
  |
  +-- victoria-metrics (depends_on: nats healthy, if enabled)
  +-- victoria-logs (if enabled)

When NATS is embedded:

gorai (no external dependencies, embeds NATS)
  |
  +-- device-reset-* (depends_on: gorai healthy)
  |     |
  |     +-- [services that use the device]
  |
  +-- [services that depend on components] (depends_on: gorai healthy)
  +-- victoria-metrics (if enabled)
  +-- victoria-logs (if enabled)
```

REQ-DEP-1
: The compiler MUST translate RDL `depends_on` references into process-compose `depends_on` entries.

REQ-DEP-2
: A service dependency on a component (which is in-process in the gorai controller) MUST be translated to a dependency on the `gorai` process with `condition: process_healthy`.

REQ-DEP-3
: A service dependency on another external service MUST be translated to a dependency on that service's process with `condition: process_started` (or `process_healthy` if the target has a readiness probe).

REQ-DEP-4
: The existing RDL circular dependency validation MUST run during compilation. The compiler MUST NOT emit a process-compose file with circular dependencies.

---

## 11. Shutdown Behavior

REQ-SHUTDOWN-1
: The compiled process-compose config MUST set `ordered_shutdown: true`.

REQ-SHUTDOWN-2
: The gorai controller process MUST have a shutdown configuration:

```yaml
shutdown:
  signal: 15
  timeout_seconds: 30
```

REQ-SHUTDOWN-3
: Container service processes MUST have a shutdown configuration using `podman stop` with a 10-second grace period (REQ-COMPILE-43).

REQ-SHUTDOWN-4
: Native binary service processes MUST receive SIGTERM with a 10-second timeout:

```yaml
shutdown:
  signal: 15
  timeout_seconds: 10
```

REQ-SHUTDOWN-5
: The gorai controller MUST handle SIGTERM by:
1. Publishing `EventRobotShutdown` to NATS
2. Closing internal services (reverse initialization order)
3. Closing components (reverse initialization order)
4. Closing the NATS client connection
5. Shutting down the embedded NATS server (if embedded)
6. Exiting with code 0

---

## 12. Code Removal

The following code becomes obsolete and MUST be removed or archived upon completion:

REQ-REMOVE-1
: `archive/pkg/quadlet/` — Quadlet systemd generator. Replaced by process-compose compiler output.

REQ-REMOVE-2
: `archive/pkg/runtime/` — External service manager (container_runner.go, process_runner.go, external_manager.go). Replaced by process-compose lifecycle management.

REQ-REMOVE-3
: `pkg/systemd/systemd.go` — Systemd unit file generator. Replaced by process-compose compiler output.

REQ-REMOVE-4
: External service spawning in `pkg/robot/robot.go` (`startExternalService`, `monitorExternalService`) — when running under process-compose, external services are managed by process-compose. The code path MUST be retained for standalone `gorai run` mode (without process-compose) but SHOULD be deprecated with a log message recommending `gorai run --compose`.

---

## 13. Example Compiled Output

Given the following RDL:

```json
{
  "version": "2",
  "robot": { "name": "scout", "description": "Outdoor scout robot" },
  "nats": { "url": "nats://localhost:4222" },
  "devices": [
    { "id": "pico-1", "nats_prefix": "gsp", "reset_on_startup": true }
  ],
  "components": [
    { "name": "left_motor", "type": "motor", "model": "remote",
      "attributes": { "device_id": "pico-1", "pin_a": 2, "pin_b": 3 } },
    { "name": "right_motor", "type": "motor", "model": "remote",
      "attributes": { "device_id": "pico-1", "pin_a": 4, "pin_b": 5 } }
  ],
  "services": [
    {
      "name": "navigator",
      "type": "navigation",
      "model": "waypoint",
      "external": {
        "enabled": true,
        "command": "/usr/local/bin/gorai-nav-waypoint",
        "restart": "on-failure"
      },
      "depends_on": ["left_motor", "right_motor"]
    },
    {
      "name": "detector",
      "type": "vision",
      "model": "yolo",
      "external": {
        "enabled": true,
        "restart": "on-failure",
        "container": {
          "image": "localhost/gorai-vision-yolo:latest",
          "devices": ["/dev/video0"],
          "volumes": ["./models:/models:ro"]
        }
      }
    }
  ],
  "metrics": { "enabled": true, "retention": "14d" }
}
```

The compiler MUST produce output equivalent to:

```yaml
version: "0.5"
ordered_shutdown: true

environment:
  - "GORAI_ROBOT_NAME=scout"
  - "GORAI_NAMESPACE=scout"
  - "NATS_URL=nats://localhost:4222"
  - "VICTORIA_METRICS_URL=http://127.0.0.1:8428"

log_configuration:
  fields_order: ["time", "level", "message"]
  disable_json: false

processes:
  gorai:
    command: "gorai run robot.json"
    namespace: core
    readiness_probe:
      http_get:
        host: 127.0.0.1
        port: 4180
        path: /healthz
      initial_delay_seconds: 1
      period_seconds: 5
      failure_threshold: 3
    availability:
      restart: on_failure
    shutdown:
      signal: 15
      timeout_seconds: 30
    log_location: "./logs/gorai.log"
    log_rotation:
      max_size_mb: 10
      max_backups: 3
      max_age_days: 7
      compress: true

  device-reset-pico-1:
    command: "gorai device reset --nats-prefix gsp --device-id pico-1"
    namespace: core
    depends_on:
      gorai:
        condition: process_healthy
    availability:
      restart: "no"

  navigator:
    command: "/usr/local/bin/gorai-nav-waypoint"
    namespace: services
    environment:
      - "GORAI_SERVICE_NAME=navigator"
      - "LOG_LEVEL=ERROR"
    depends_on:
      gorai:
        condition: process_healthy
      device-reset-pico-1:
        condition: process_completed
    availability:
      restart: on_failure
      backoff_seconds: 2
    shutdown:
      signal: 15
      timeout_seconds: 10
    log_location: "./logs/navigator.log"
    log_rotation:
      max_size_mb: 10
      max_backups: 3
      max_age_days: 7
      compress: true

  detector:
    command: >-
      podman run --rm --name detector
      --network host
      --device /dev/video0
      -v ./models:/models:ro
      -e GORAI_ROBOT_NAME=scout
      -e GORAI_NAMESPACE=scout
      -e GORAI_SERVICE_NAME=detector
      -e NATS_URL=nats://localhost:4222
      -e LOG_LEVEL=ERROR
      -e VICTORIA_METRICS_URL=http://127.0.0.1:8428
      localhost/gorai-vision-yolo:latest
    namespace: services
    availability:
      restart: on_failure
      backoff_seconds: 2
    shutdown:
      command: "podman stop -t 10 detector"
      timeout_seconds: 15
    log_location: "./logs/detector.log"
    log_rotation:
      max_size_mb: 10
      max_backups: 3
      max_age_days: 7
      compress: true

  victoria-metrics:
    command: >-
      victoria-metrics
      -retentionPeriod=14d
      -httpListenAddr=127.0.0.1:8428
      -storageDataPath=./data/victoria-metrics/
    namespace: infra
    readiness_probe:
      http_get:
        host: 127.0.0.1
        port: 8428
        path: /health
      initial_delay_seconds: 2
      period_seconds: 5
    availability:
      restart: always
    shutdown:
      signal: 15
      timeout_seconds: 10
    log_location: "./logs/victoria-metrics.log"
    log_rotation:
      max_size_mb: 10
      max_backups: 3
      max_age_days: 7
      compress: true
```

---

## 14. Testing Requirements

REQ-TEST-1
: Unit tests for the compiler MUST cover:
- Minimal RDL (robot name only) → single gorai process with embedded NATS
- RDL with external NATS → separate nats-server process emitted
- RDL with native binary services → correct process entries with env vars
- RDL with container services → correct podman run commands with shutdown blocks
- RDL with devices → transient reset processes with correct dependencies
- RDL with optional infrastructure (metrics, logging) → correct infra processes
- Dependency graph translation (component deps → gorai process deps)
- Circular dependency rejection
- Restart policy mapping
- Topic resolution into environment variables

REQ-TEST-2
: Unit tests for embedded NATS MUST cover:
- Server starts and accepts connections
- Server respects TLS configuration
- JetStream enabled/disabled based on config
- Server shuts down cleanly after client disconnect
- Controller connects as regular client (no in-process bypass)

REQ-TEST-3
: Unit tests for `gorai device reset` MUST cover:
- Publishes correct NATS message
- Exits 0 on success
- Exits non-zero on NATS connection failure
- Respects NATS_URL environment variable

REQ-TEST-4
: Unit tests for the health endpoint MUST cover:
- Returns 503 before initialization completes
- Returns 200 after EventRobotReady
- `/livez` returns 200 immediately after server starts

REQ-TEST-5
: Integration tests MUST verify the full pipeline:
- `gorai compile` produces valid process-compose YAML
- `process-compose up` successfully starts all processes
- Health probes pass within expected timeframes
- Ordered shutdown stops processes in correct order
- Device reset processes run and complete before dependent services start

---

## 15. Migration Path

REQ-MIGRATE-1
: Existing robots using `gorai run` with no external services MUST continue to work without any configuration changes. Embedded NATS is the default.

REQ-MIGRATE-2
: Existing robots with `external` services MUST work with `gorai run` (standalone mode, backward compatible). To use process-compose, users run `gorai compile && process-compose up` or `gorai run --compose`.

REQ-MIGRATE-3
: The `containers` section of RDL v1 (already deprecated) MUST NOT be supported by the compiler. Users must migrate to RDL v2 with `external.container` service configs.

REQ-MIGRATE-4
: Documentation MUST be updated to recommend `gorai run --compose` as the standard deployment method, with standalone `gorai run` positioned as the quick-start / development mode.

---

## 16. Dependencies

### 16.1 New Go Dependencies

| Package | Purpose |
|---|---|
| `github.com/nats-io/nats-server/v2/server` | Embedded NATS server |

### 16.2 External Tools

| Tool | Required By | Installation |
|---|---|---|
| `process-compose` | `gorai compile` output / `gorai run --compose` | User-installed, documented in setup guide |
| `podman` | Container service execution | User-installed when container services are used |
| `nats-server` | External NATS mode only | User-installed when `nats.external: true` |
| `victoria-metrics` | Optional metrics | User-installed when `metrics.enabled: true` |
| `victoria-logs` | Optional logging | User-installed when `logging.enabled: true` |

REQ-DEPS-1
: `gorai compile` MUST NOT require any external tools at compile time. It only generates YAML.

REQ-DEPS-2
: `gorai run --compose` MUST check that `process-compose` is installed and on PATH before exec. If missing, exit with a clear error message and installation instructions.

REQ-DEPS-3
: The compiler MUST NOT validate that referenced binaries (service commands, podman, victoria-metrics, etc.) exist. That validation happens at runtime when process-compose starts the processes.
