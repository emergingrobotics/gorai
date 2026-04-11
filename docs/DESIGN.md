# Gorai Process-Compose Runtime -- Design Document

**Version:** 1.0
**Date:** 2026-04-11
**Status:** Draft
**Requirements:** `/gorai/REQUIREMENTS.md`

---

## 1. Overview

### 1.1 What We Are Building

A process-compose-based runtime that replaces Gorai's in-process external service management and archived Quadlet/systemd generation. The system has three parts:

1. **Embedded NATS server** -- the gorai controller optionally runs NATS in-process, eliminating the external NATS dependency for single-robot deployments.
2. **RDL-to-process-compose compiler** -- transforms `robot.json` into a `process-compose.yaml` file that declares all processes, dependencies, health probes, and shutdown behavior.
3. **Health/readiness HTTP server** -- exposes `/healthz` and `/livez` endpoints so process-compose can gate dependent service startup on controller readiness.

### 1.2 Why

The current architecture (`/gorai/pkg/robot/robot.go`) manages external service processes directly via `os/exec`, implementing restart logic, signal forwarding, and log capture in application code. This duplicates functionality that process-compose provides natively. The embedded NATS server eliminates the most common setup friction: installing and starting a separate NATS server before running a robot.

### 1.3 Constraints

- `gorai run robot.json` (standalone, no process-compose) remains fully functional.
- The component registry, Service RDL, and NATS topic resolution (`/gorai/pkg/config/topic_resolver.go`, `/gorai/pkg/config/service_merger.go`) are unchanged.
- RDL v2 schema changes are limited to three additions: `NATSConfig.External`, `MetricsConfig`, `LoggingConfig`.

---

## 2. Architecture

### 2.1 Deployment Model

```
robot.json (RDL)
      |
      v
gorai compile ---------> process-compose.yaml
      |                         |
      v                         v
gorai run robot.json     process-compose up
(standalone mode)        (managed mode)
      |                         |
      +-- embedded NATS         +-- gorai controller (embedded NATS or client-only)
      +-- components            +-- [service binaries]
      +-- internal services     +-- [podman containers]
                                +-- [victoria-metrics]   (optional)
                                +-- [victoria-logs]      (optional)
```

**Standalone mode** (`gorai run`): The controller embeds NATS, initializes components and internal services in-process, and optionally spawns external services (backward compatibility). No process-compose required.

**Managed mode** (`gorai compile && process-compose up`, or `gorai run --compose`): The controller runs as one process among many. External services are process-compose's responsibility. The controller detects managed mode via the `PC_PROC_NAME` environment variable (injected by process-compose) and skips external service spawning.

### 2.2 Process Namespaces

| Namespace | Contents |
|-----------|----------|
| `infra` | NATS server (when external), VictoriaMetrics, VictoriaLogs |
| `core` | gorai controller, device reset processes |
| `services` | User-defined service binaries and container services |

When the gorai controller embeds NATS and no optional infrastructure is enabled, namespaces are omitted (single process).

### 2.3 Component Diagram

```
+---------------------------------------------------------------+
|                     process-compose                            |
|                                                                |
|  +-----------+   +-----------+   +-----------+   +-----------+ |
|  | infra     |   | core      |   | services  |   | services  | |
|  |           |   |           |   |           |   |           | |
|  | victoria- |   | gorai     |   | navigator |   | detector  | |
|  | metrics   |   | controller|   | (binary)  |   | (podman)  | |
|  |           |   |           |   |           |   |           | |
|  |           |   | +-------+ |   |           |   |           | |
|  |           |   | |embed  | |   |           |   |           | |
|  |           |   | |NATS   | |   |           |   |           | |
|  |           |   | +-------+ |   |           |   |           | |
|  |           |   | +-------+ |   |           |   |           | |
|  |           |   | |health | |   |           |   |           | |
|  |           |   | |:4180  | |   |           |   |           | |
|  |           |   | +-------+ |   |           |   |           | |
|  +-----------+   +-----------+   +-----------+   +-----------+ |
+---------------------------------------------------------------+
         All processes communicate via NATS (port 4222)
```

---

## 3. Package Design

### 3.1 `pkg/embeddednats` -- Embedded NATS Server Lifecycle

**Responsibility:** Start and stop an in-process NATS server. The caller receives a running server and connects to it as a regular NATS client via `pkg/nats.Client`.

**File:** `/gorai/pkg/embeddednats/server.go`

```go
package embeddednats

import (
    "context"
    "log/slog"

    "github.com/nats-io/nats-server/v2/server"
)

// Config holds configuration for the embedded NATS server.
type Config struct {
    Host           string // Listen host. Default: "127.0.0.1"
    Port           int    // Listen port. Default: 4222
    JetStream      bool   // Enable JetStream
    JetStreamDir   string // JetStream storage directory. Default: "./data/jetstream/"
    TLSCertFile    string // TLS certificate file
    TLSKeyFile     string // TLS key file
    TLSCAFile      string // TLS CA file
}

// Server wraps a nats-server instance for in-process use.
type Server struct {
    server *server.Server
    config Config
    logger *slog.Logger
}

// Option configures a Server.
type Option func(*Server)

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option

// New creates a new embedded NATS server without starting it.
func New(cfg Config, opts ...Option) (*Server, error)

// Start starts the server and blocks until it is ready to accept connections.
// Returns an error if the server fails to become ready within 10 seconds.
func (s *Server) Start(ctx context.Context) error

// ClientURL returns the URL clients should use to connect (e.g., "nats://127.0.0.1:4222").
func (s *Server) ClientURL() string

// Shutdown drains all connections and stops the server.
// Must be called after all clients have closed their connections.
func (s *Server) Shutdown(ctx context.Context) error

// Ready returns true if the server is accepting connections.
func (s *Server) Ready() bool
```

**Integration:** Called from `pkg/robot/robot.go` in `Start()`, before `connectNATS()`. The embedded server must be fully ready before any NATS client connects.

### 3.2 `pkg/health` -- Health/Readiness HTTP Server

**Responsibility:** Expose `/healthz` (readiness) and `/livez` (liveness) HTTP endpoints. The readiness state is set by the robot lifecycle.

**File:** `/gorai/pkg/health/server.go`

```go
package health

import (
    "context"
    "log/slog"
    "net/http"
    "sync/atomic"
)

// Server provides health and readiness HTTP endpoints.
type Server struct {
    ready  atomic.Bool
    server *http.Server
    logger *slog.Logger
}

// Option configures a Server.
type Option func(*Server)

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option

// New creates a health server listening on the given address.
// The server starts immediately in a goroutine.
// /livez returns 200 as soon as the server is running.
// /healthz returns 503 until SetReady(true) is called.
func New(listenAddress string, opts ...Option) (*Server, error)

// SetReady updates the readiness state.
// Call SetReady(true) after EventRobotReady is published.
// Call SetReady(false) when the robot begins shutdown.
func (s *Server) SetReady(ready bool)

// Shutdown stops the HTTP server gracefully.
func (s *Server) Shutdown(ctx context.Context) error
```

**Endpoints:**

| Path | Method | Ready=false | Ready=true |
|------|--------|-------------|------------|
| `/livez` | GET | 200 `{"status":"alive"}` | 200 `{"status":"alive"}` |
| `/healthz` | GET | 503 `{"status":"not_ready"}` | 200 `{"status":"ready"}` |

**Integration:** Created in `cmdRun()` (`/gorai/cmd/gorai/commands/run.go`) before `robot.Start()`. The health server starts before component initialization so process-compose can begin probing immediately. After `robot.Start()` publishes `EventRobotReady`, the caller sets `health.SetReady(true)`.

### 3.3 `pkg/compose` -- RDL-to-Process-Compose Compiler

**Responsibility:** Transform a validated `config.RDL` into a process-compose YAML document. Pure data transformation with no side effects.

**Files:**

| File | Contents |
|------|----------|
| `/gorai/pkg/compose/compiler.go` | `Compiler` type, `Compile()` entry point |
| `/gorai/pkg/compose/model.go` | Process-compose YAML struct types |
| `/gorai/pkg/compose/nats.go` | NATS mode selection and process emission |
| `/gorai/pkg/compose/services.go` | Service-to-process translation (native + container) |
| `/gorai/pkg/compose/devices.go` | Device reset process emission |
| `/gorai/pkg/compose/infra.go` | VictoriaMetrics, VictoriaLogs process emission |
| `/gorai/pkg/compose/dependencies.go` | Dependency graph translation |

```go
package compose

import (
    "github.com/gorai/gorai/pkg/config"
)

// Compiler transforms RDL configuration into process-compose YAML.
type Compiler struct {
    config     *config.RDL
    configPath string
}

// New creates a compiler for the given RDL configuration.
// configPath is the path to the RDL file, used in the gorai run command.
func New(cfg *config.RDL, configPath string) *Compiler

// Compile produces a ProcessCompose model from the RDL.
// Returns an error if the configuration cannot be compiled (e.g., circular dependencies).
func (c *Compiler) Compile() (*ProcessCompose, error)

// Render serializes a ProcessCompose model to YAML bytes.
func Render(pc *ProcessCompose) ([]byte, error)
```

### 3.4 `pkg/compose/model.go` -- Process-Compose YAML Types

```go
package compose

// ProcessCompose represents a complete process-compose.yaml file.
type ProcessCompose struct {
    Version          string            `yaml:"version"`
    OrderedShutdown  bool              `yaml:"ordered_shutdown,omitempty"`
    Environment      []string          `yaml:"environment,omitempty"`
    LogConfiguration *LogConfiguration `yaml:"log_configuration,omitempty"`
    Processes        map[string]Process `yaml:"processes"`
}

// Process represents a single process-compose process entry.
type Process struct {
    Command        string              `yaml:"command"`
    Namespace      string              `yaml:"namespace,omitempty"`
    Environment    []string            `yaml:"environment,omitempty"`
    DependsOn      map[string]DependsOn `yaml:"depends_on,omitempty"`
    Availability   *Availability       `yaml:"availability,omitempty"`
    ReadinessProbe *ReadinessProbe     `yaml:"readiness_probe,omitempty"`
    Shutdown       *Shutdown           `yaml:"shutdown,omitempty"`
    LogLocation    string              `yaml:"log_location,omitempty"`
    LogRotation    *LogRotation        `yaml:"log_rotation,omitempty"`
}

// DependsOn defines a process dependency.
type DependsOn struct {
    Condition string `yaml:"condition"`
}

// Availability defines restart behavior.
type Availability struct {
    Restart        string `yaml:"restart"`
    BackoffSeconds int    `yaml:"backoff_seconds,omitempty"`
}

// ReadinessProbe defines a health check for process readiness.
type ReadinessProbe struct {
    HTTPGet             *HTTPGet `yaml:"http_get,omitempty"`
    Exec                *Exec   `yaml:"exec,omitempty"`
    InitialDelaySeconds int     `yaml:"initial_delay_seconds,omitempty"`
    PeriodSeconds       int     `yaml:"period_seconds,omitempty"`
    FailureThreshold    int     `yaml:"failure_threshold,omitempty"`
}

// HTTPGet defines an HTTP GET readiness probe.
type HTTPGet struct {
    Host string `yaml:"host"`
    Port int    `yaml:"port"`
    Path string `yaml:"path"`
}

// Exec defines a command-based readiness probe.
type Exec struct {
    Command string `yaml:"command"`
}

// Shutdown defines process shutdown behavior.
type Shutdown struct {
    Command        string `yaml:"command,omitempty"`
    Signal         int    `yaml:"signal,omitempty"`
    TimeoutSeconds int    `yaml:"timeout_seconds"`
}

// LogConfiguration defines global log settings.
type LogConfiguration struct {
    FieldsOrder []string `yaml:"fields_order,omitempty"`
    DisableJSON bool     `yaml:"disable_json"`
    NoMetadata  bool     `yaml:"no_metadata"`
}

// LogRotation defines per-process log rotation.
type LogRotation struct {
    MaxSizeMB  int  `yaml:"max_size_mb"`
    MaxBackups int  `yaml:"max_backups"`
    MaxAgeDays int  `yaml:"max_age_days"`
    Compress   bool `yaml:"compress"`
}
```

### 3.5 CLI Additions in `cmd/gorai/commands/`

**New files:**

| File | Command | Description |
|------|---------|-------------|
| `/gorai/cmd/gorai/commands/compile.go` | `gorai compile <config>` | Compile RDL to process-compose YAML |
| `/gorai/cmd/gorai/commands/device.go` | `gorai device reset` | Send GSP/2 RESET via NATS |

**Modified files:**

| File | Changes |
|------|---------|
| `/gorai/cmd/gorai/commands/root.go` | Add `compile`, `device`, and `up` command dispatch |
| `/gorai/cmd/gorai/commands/run.go` | Add `--compose`, `--health-listen` flags; integrate embedded NATS and health server |

---

## 4. Module Interfaces

### 4.1 Embedded NATS -- Full Interface

```go
// ConfigFromRDL extracts embedded NATS configuration from the RDL.
// Returns nil if NATS should not be embedded (external mode or remote URL).
func ConfigFromRDL(natsCfg *config.NATSConfig) *Config
```

This function encapsulates the NATS mode selection logic (Section 7) and is the single place where the decision tree is evaluated.

### 4.2 Health Server -- Full Interface

```go
// IsReady returns the current readiness state.
func (s *Server) IsReady() bool
```

### 4.3 Compiler -- Full Interface

```go
// NATSMode describes how NATS is deployed.
type NATSMode int

const (
    NATSModeEmbedded NATSMode = iota // Embedded in gorai controller
    NATSModeExternal                  // Separate process managed by process-compose
    NATSModeRemote                    // Remote server, not managed
)

// DetectNATSMode determines the NATS deployment mode from RDL configuration.
func DetectNATSMode(natsCfg *config.NATSConfig) NATSMode

// MapRestartPolicy converts an RDL restart value to a process-compose restart value.
func MapRestartPolicy(rdlRestart string) string
```

Restart policy mapping:

| RDL Value | process-compose Value |
|-----------|----------------------|
| `"always"` | `"always"` |
| `"on-failure"` | `"on_failure"` |
| `"never"` | `"no"` |
| `""` (absent) | `"on_failure"` |

### 4.4 Device Reset Command

```go
// cmdDeviceReset implements "gorai device reset".
func cmdDeviceReset() error
```

Flags: `--nats-prefix`, `--device-id`, `--nats-url` (default from `NATS_URL` env, fallback `nats://localhost:4222`).

Behavior: Connect to NATS, publish `{"subsystem":0}` to `<prefix>.<device-id>.tx.system.reset`, flush, sleep 500ms, exit 0. On failure, exit non-zero with error to stderr.

---

## 5. Config Changes

### 5.1 NATSConfig.External

**File:** `/gorai/pkg/config/config.go`

Add to existing `NATSConfig` struct:

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
    External        bool       `json:"external,omitempty"`

    Container string `json:"container,omitempty"` // Deprecated
}
```

**Validation rule:** If `External` is `true` and the URL resolves to a non-local address, emit a warning (redundant flag).

### 5.2 MetricsConfig

**File:** `/gorai/pkg/config/config.go`

Add to `RDL` struct:

```go
type RDL struct {
    // ... existing fields ...
    Metrics  *MetricsConfig  `json:"metrics,omitempty"`
    Logging  *LoggingConfig  `json:"logging,omitempty"`
}

type MetricsConfig struct {
    Enabled   bool   `json:"enabled"`
    Retention string `json:"retention,omitempty"` // Default: "7d"
    Listen    string `json:"listen,omitempty"`    // Default: "127.0.0.1:8428"
}

type LoggingConfig struct {
    Enabled   bool   `json:"enabled"`
    Retention string `json:"retention,omitempty"` // Default: "3d"
    Listen    string `json:"listen,omitempty"`    // Default: "127.0.0.1:9428"
}
```

### 5.3 Defaults

Applied in `/gorai/pkg/config/config.go` `applyDefaults()`:

| Field | Default |
|-------|---------|
| `MetricsConfig.Retention` | `"7d"` |
| `MetricsConfig.Listen` | `"127.0.0.1:8428"` |
| `LoggingConfig.Retention` | `"3d"` |
| `LoggingConfig.Listen` | `"127.0.0.1:9428"` |

Defaults are applied only when the parent struct is non-nil (the feature is explicitly enabled).

---

## 6. Integration Points

### 6.1 `pkg/robot/robot.go` Changes

The `Robot` struct gains two optional fields:

```go
type Robot struct {
    // ... existing fields ...
    embeddedNATS *embeddednats.Server // nil when NATS is external/remote
    health       *health.Server      // nil when health server is disabled
}
```

**`Start()` method changes:**

```
Current flow:                        New flow:
  connectNATS()                        maybeStartEmbeddedNATS()
  resetDevices()                       connectNATS()
  startDashboard()                     resetDevices()
  publishStartupEvent(Started)         startDashboard()
  detectHardware()                     publishStartupEvent(Started)
  start components                     detectHardware()
  start services                       start components
  publishStartupEvent(Ready)           start services (skip external if PC_PROC_NAME set)
                                       publishStartupEvent(Ready)
                                       health.SetReady(true)
```

The `maybeStartEmbeddedNATS()` method calls `embeddednats.ConfigFromRDL()` to decide whether to embed. If embedding, it starts the server and uses its `ClientURL()` to connect the standard NATS client.

**`Stop()` method changes:**

```
Current flow:                        New flow:
  publishStartupEvent(Shutdown)        health.SetReady(false)
  stopExternalServices()               publishStartupEvent(Shutdown)
  stop dashboard                       stopExternalServices()  (standalone only)
  stop services                        stop dashboard
  stop components                      stop services
  cancel context                       stop components
  close NATS                           cancel context
                                       close NATS client
                                       shutdown embedded NATS  (if embedded)
                                       shutdown health server
```

**External service management:**

The `startExternalService()` and `monitorExternalService()` methods in `robot.go` remain for standalone mode but are skipped when `PC_PROC_NAME` is set:

```go
func (r *Robot) isUnderProcessCompose() bool {
    return os.Getenv("PC_PROC_NAME") != ""
}
```

When `isUnderProcessCompose()` returns true, the service loop in `Start()` logs a message and skips external services.

### 6.2 `cmd/gorai/commands/run.go` Changes

The `cmdRun()` function gains:

1. **`--compose` flag:** When set, compile the RDL to a temp file and `exec` into `process-compose up`.
2. **`--health-listen` flag:** Default `127.0.0.1:4180`. Creates a `health.Server` before `robot.Start()`.
3. **Health server lifecycle:** Started before `robot.Start()`, readiness set after `robot.Start()` succeeds, shutdown in the cleanup path.

```go
// Pseudocode for the --compose path
func cmdRunCompose(configPath string, cfg *config.RDL) error {
    compiler := compose.New(cfg, configPath)
    pc, err := compiler.Compile()
    // write to temp file
    yaml, _ := compose.Render(pc)
    tmpFile := writeTempFile(yaml)
    // exec replaces the current process
    return syscall.Exec(processComposeBinary, []string{"process-compose", "up", "-f", tmpFile}, os.Environ())
}
```

### 6.3 `cmd/gorai/commands/root.go` Changes

Add command dispatch entries:

```go
case "compile":
    return cmdCompile()
case "device":
    return cmdDevice()
case "up":
    // Alias for "run --compose"
    os.Args = append(os.Args[:1], append([]string{"run", "--compose"}, os.Args[2:]...)...)
    return cmdRun()
```

---

## 7. NATS Mode Selection

The decision tree is evaluated in `embeddednats.ConfigFromRDL()` and `compose.DetectNATSMode()`:

```
                    Is nats.url configured?
                    /                     \
                  No                      Yes
                  |                        |
          Embed NATS on              Is URL local?
          localhost:4222          (localhost/127.0.0.1)
                                  /                 \
                                Yes                  No
                                 |                    |
                          Is nats.external        Remote mode:
                              true?               client only,
                            /       \             compiler does NOT
                          No        Yes           emit NATS process
                          |          |
                    Embed NATS    External mode:
                    on configured  client only,
                    URL            compiler emits
                                   nats-server process
```

**Truth table:**

| `nats` section | `nats.url` | `nats.external` | Mode | Compiler emits NATS process? |
|----------------|------------|-----------------|------|------------------------------|
| absent | (default localhost:4222) | (default false) | Embedded | No |
| present | localhost/127.0.0.1 | false or absent | Embedded | No |
| present | localhost/127.0.0.1 | true | External | Yes |
| present | remote address | any | Remote | No |

**URL locality check:**

```go
func isLocalURL(natsURL string) bool {
    u, err := url.Parse(natsURL)
    if err != nil {
        return false
    }
    host := u.Hostname()
    return host == "localhost" || host == "127.0.0.1" || host == "::1" || host == ""
}
```

---

## 8. Compiler Pipeline

The `Compile()` method executes these steps in order:

### Step 1: Detect NATS mode

Call `DetectNATSMode(cfg.NATS)` to determine embedded, external, or remote.

### Step 2: Build global environment

```go
env := []string{
    fmt.Sprintf("GORAI_ROBOT_NAME=%s", cfg.Robot.Name),
    fmt.Sprintf("GORAI_NAMESPACE=%s", cfg.GetEffectiveNamespace()),
    fmt.Sprintf("NATS_URL=%s", effectiveNATSURL),
}
```

If `metrics.enabled`, append `VICTORIA_METRICS_URL`. If `logging.enabled`, append `VICTORIA_LOGS_URL`.

### Step 3: Emit infrastructure processes

- **External NATS** (if `NATSModeExternal`): Emit `nats-server` process in `infra` namespace with TCP readiness probe, `restart: always`.
- **VictoriaMetrics** (if `metrics.enabled`): Emit process in `infra` namespace with HTTP readiness probe on `/health`.
- **VictoriaLogs** (if `logging.enabled`): Emit process in `infra` namespace with HTTP readiness probe on `/health`.

### Step 4: Emit gorai controller process

Emit `gorai` process in `core` namespace:
- Command: `gorai run <config-path>`
- Readiness probe: HTTP GET on `/healthz` at the health listen address
- Availability: `restart: on_failure`
- Shutdown: `signal: 15, timeout_seconds: 30`
- Log location and rotation
- If external NATS: `depends_on: {nats-server: {condition: process_healthy}}`

### Step 5: Emit device reset processes

For each device with `reset_on_startup: true`:
- Process name: `device-reset-<device-id>`
- Command: `gorai device reset --nats-prefix <prefix> --device-id <id>`
- Namespace: `core`
- Availability: `restart: "no"` (transient)
- Depends on the NATS source (gorai if embedded, nats-server if external) with `condition: process_healthy`
- No log location (transient)

### Step 6: Emit native binary service processes

For each service where `external.enabled` is true, `external.command` is set, and `external.container` is nil:
- Process name: service name
- Command: `external.command` with `external.args` appended
- Namespace: `services`
- Environment: from `config.GetResolvedEnvironment()` (includes `GORAI_SERVICE_NAME`, resolved topics, log level)
- Availability: restart mapped via `MapRestartPolicy(external.restart)`, `backoff_seconds: 2`
- Shutdown: `signal: 15, timeout_seconds: 10`
- Dependencies: translated per Section 9
- Log location and rotation

### Step 7: Emit container service processes

For each service where `external.enabled` is true and `external.container` is configured:
- Process name: service name
- Command: `podman run --rm --name <service-name>` with all flags (network, devices, volumes, env vars, privileged, image)
- Namespace: `services`
- Availability: restart mapped via `MapRestartPolicy(external.restart)`, `backoff_seconds: 2`
- Shutdown: `command: "podman stop -t 10 <service-name>", timeout_seconds: 15`
- Dependencies: translated per Section 9
- Log location and rotation

Container environment variables are passed inline via `-e KEY=VALUE` in the podman command (not as process-compose environment) because the container has its own environment.

### Step 8: Emit log configuration

```yaml
log_configuration:
  fields_order: ["time", "level", "message"]
  disable_json: false
  no_metadata: false
```

### Step 9: Set global flags

- `version: "0.5"`
- `ordered_shutdown: true`

---

## 9. Dependency Graph Generation

### 9.1 Dependency Sources

Dependencies come from two places in the RDL:
1. `service.depends_on[]` -- explicit string references to component or service names
2. `device.reset_on_startup` + `component.attributes.device_id` -- implicit device dependencies

### 9.2 Translation Rules

The compiler builds a lookup table of all names and their hosting location:

```go
type processLocation struct {
    processName string    // process-compose process name
    isComponent bool      // true = runs inside gorai controller
    isDevice    bool      // true = device reset process
    hasProbe    bool      // true = has a readiness probe
}
```

All components are mapped to `processName: "gorai"`, `isComponent: true`.
All external services are mapped to their own process name.
All devices with `reset_on_startup` are mapped to `processName: "device-reset-<id>"`.

**Translation logic per dependency:**

```
For each dep in service.depends_on:
    target = lookup[dep]
    if target.isComponent:
        emit depends_on[gorai] = {condition: process_healthy}
    else if target.hasProbe:
        emit depends_on[target.processName] = {condition: process_healthy}
    else:
        emit depends_on[target.processName] = {condition: process_started}
```

**Device dependency injection:**

For each service that references a device (via `attributes.device_id`) where that device has a reset process:

```
emit depends_on[device-reset-<device-id>] = {condition: process_completed}
```

This is computed by scanning `cfg.Components` for `attributes.device_id`, finding which services depend on those components, and propagating the device dependency to those services.

### 9.3 Transitive Device Dependencies

When a service depends on a component that uses a device:

```
navigator depends_on [left_motor, right_motor]
left_motor.attributes.device_id = "pico-1"
right_motor.attributes.device_id = "pico-1"
pico-1.reset_on_startup = true

Result for navigator process:
  depends_on:
    gorai: {condition: process_healthy}           # from component deps
    device-reset-pico-1: {condition: process_completed}  # from device dep
```

### 9.4 Circular Dependency Validation

The existing `checkCircularDependencies()` in `/gorai/pkg/config/config.go` (lines 656-714) runs during `cfg.Validate()`, which is called before compilation. The compiler does not need separate cycle detection.

---

## 10. Shutdown Sequence

### 10.1 Process-Compose Ordered Shutdown

With `ordered_shutdown: true`, process-compose stops processes in reverse dependency order:

```
1. services (navigator, detector) -- SIGTERM, 10s timeout
2. device-reset-* -- already completed, no action
3. gorai controller -- SIGTERM, 30s timeout
4. infra (victoria-metrics, victoria-logs) -- SIGTERM, 10s timeout
5. nats-server (if external) -- last to stop
```

### 10.2 Gorai Controller Internal Shutdown (on SIGTERM)

When the gorai controller receives SIGTERM (signal 15), the `Stop()` method in `/gorai/pkg/robot/robot.go` executes:

```
1. health.SetReady(false)              -- signal to process-compose: not ready
2. Publish EventRobotShutdown to NATS  -- notify subscribers
3. Stop external services              -- standalone mode only (no-op under process-compose)
4. Stop dashboard
5. Close internal services             -- reverse initialization order
6. Close components                    -- reverse initialization order
7. Cancel internal context
8. Close NATS client connection        -- drain + close
9. Shutdown embedded NATS server       -- drain all connections, stop listener
10. Shutdown health server
```

### 10.3 Container Service Shutdown

Container services receive a separate shutdown command instead of a signal:

```yaml
shutdown:
  command: "podman stop -t 10 <service-name>"
  timeout_seconds: 15
```

`podman stop -t 10` sends SIGTERM to the container's PID 1, waits 10 seconds, then sends SIGKILL. The process-compose timeout of 15 seconds gives podman time to complete.

---

## 11. Data Flow Diagrams

### 11.1 Startup Sequence (Managed Mode)

```
process-compose up
    |
    +--> [infra] start nats-server (if external)
    |         |
    |         +--> readiness probe passes (TCP :4222)
    |
    +--> [infra] start victoria-metrics (if enabled)
    |
    +--> [core] start gorai controller
    |         |
    |         +--> (internal) start health server on :4180
    |         |        /livez -> 200 immediately
    |         |        /healthz -> 503
    |         |
    |         +--> (internal) start embedded NATS (if not external)
    |         +--> (internal) connect NATS client
    |         +--> (internal) initialize components
    |         +--> (internal) initialize internal services
    |         +--> (internal) publish EventRobotReady
    |         +--> (internal) health.SetReady(true)
    |                  /healthz -> 200
    |
    +--> [core] start device-reset-pico-1
    |         |   (depends_on gorai: process_healthy)
    |         |
    |         +--> publish RESET to NATS, wait 500ms, exit 0
    |
    +--> [services] start navigator
    |         (depends_on gorai: process_healthy)
    |         (depends_on device-reset-pico-1: process_completed)
    |
    +--> [services] start detector
              (no explicit dependencies beyond global env)
```

### 11.2 Compile Flow

```
Input: robot.json (validated RDL with merged Service RDLs)
    |
    v
(1) DetectNATSMode()
    |
    v
(2) Build global environment variables
    |
    v
(3) Emit infra processes
    |   +-- nats-server (if external)
    |   +-- victoria-metrics (if metrics.enabled)
    |   +-- victoria-logs (if logging.enabled)
    |
    v
(4) Emit gorai controller process
    |   +-- command, readiness probe, shutdown config
    |   +-- depends_on nats-server (if external)
    |
    v
(5) Emit device reset processes
    |   +-- one per device with reset_on_startup
    |   +-- depends_on NATS source (gorai or nats-server)
    |
    v
(6) Build dependency lookup table
    |   +-- map component names -> gorai process
    |   +-- map service names -> their process names
    |   +-- map device IDs -> device-reset-* process names
    |
    v
(7) Emit service processes (native binaries)
    |   +-- resolve environment via GetResolvedEnvironment()
    |   +-- translate depends_on via lookup table
    |
    v
(8) Emit service processes (containers)
    |   +-- build podman run command with all flags
    |   +-- translate depends_on via lookup table
    |
    v
(9) Emit log configuration + global settings
    |
    v
Output: ProcessCompose struct
    |
    v
Render() -> process-compose.yaml
```

### 11.3 Shutdown Sequence

```
process-compose receives SIGINT/SIGTERM
    |
    v
ordered_shutdown: true
    |
    v
(1) Stop services namespace (reverse order)
    |   +-- navigator: SIGTERM, wait 10s
    |   +-- detector: "podman stop -t 10 detector", wait 15s
    |
    v
(2) Stop core namespace
    |   +-- gorai controller: SIGTERM, wait 30s
    |         |
    |         +--> (internal) health.SetReady(false)
    |         +--> (internal) publish EventRobotShutdown
    |         +--> (internal) close services (reverse order)
    |         +--> (internal) close components (reverse order)
    |         +--> (internal) close NATS client (drain)
    |         +--> (internal) shutdown embedded NATS
    |         +--> (internal) shutdown health server
    |         +--> exit 0
    |
    v
(3) Stop infra namespace
    |   +-- victoria-metrics: SIGTERM, wait 10s
    |   +-- victoria-logs: SIGTERM, wait 10s
    |   +-- nats-server (if external): SIGTERM
    |
    v
All processes stopped. process-compose exits.
```

---

## 12. File Summary

### New files

| Path | Package | Description |
|------|---------|-------------|
| `pkg/embeddednats/server.go` | `embeddednats` | Embedded NATS server lifecycle |
| `pkg/health/server.go` | `health` | Health/readiness HTTP server |
| `pkg/compose/compiler.go` | `compose` | Compiler entry point |
| `pkg/compose/model.go` | `compose` | Process-compose YAML struct types |
| `pkg/compose/nats.go` | `compose` | NATS mode detection and process emission |
| `pkg/compose/services.go` | `compose` | Service-to-process translation |
| `pkg/compose/devices.go` | `compose` | Device reset process emission |
| `pkg/compose/infra.go` | `compose` | Infrastructure process emission |
| `pkg/compose/dependencies.go` | `compose` | Dependency graph translation |
| `cmd/gorai/commands/compile.go` | `commands` | `gorai compile` command |
| `cmd/gorai/commands/device.go` | `commands` | `gorai device reset` command |

### Modified files

| Path | Changes |
|------|---------|
| `pkg/config/config.go` | Add `External` to `NATSConfig`; add `MetricsConfig`, `LoggingConfig` types; add `Metrics`, `Logging` fields to `RDL` |
| `pkg/robot/robot.go` | Add embedded NATS start/stop; add health server integration; skip external services under process-compose |
| `cmd/gorai/commands/run.go` | Add `--compose`, `--health-listen` flags; health server lifecycle; compose convenience mode |
| `cmd/gorai/commands/root.go` | Add `compile`, `device`, `up` command dispatch |

### Removed/archived files (post-completion)

| Path | Reason |
|------|--------|
| `archive/pkg/quadlet/` | Replaced by process-compose compiler |
| `archive/pkg/runtime/` | Replaced by process-compose lifecycle |
| `pkg/systemd/systemd.go` | Replaced by process-compose compiler |
