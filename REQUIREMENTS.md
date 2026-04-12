# Gorai Embedded NATS — Requirements

**Version:** 1.0
**Date:** 2026-04-12
**Status:** Active

## 1. Overview

This document specifies requirements for embedding the NATS server directly into the `gorai` binary, eliminating the need for users to install, configure, or manage an external NATS server for single-robot deployments.

### 1.1 Goals

1. **Zero-dependency single binary.** `gorai run robot.json` works out of the box with no external services. The user installs one binary and writes one JSON file.
2. **Preserve external NATS as an opt-in.** Advanced users and multi-robot deployments can still point at an external NATS server via RDL configuration.
3. **JetStream by default.** The embedded NATS server enables JetStream so mesh service discovery, event sourcing, and replay work without additional setup.
4. **No user-visible complexity.** Users should not need to understand NATS configuration, ports, or JetStream to get started.
5. **"npm for robotics" story.** GoRAI runs on hardware people already own (Raspberry Pi + cheap COTS kits like PiCar-X). One binary, one config file, done.

### 1.2 Non-Goals

1. Process-compose runtime integration (decided against for launch).
2. K3s or container orchestration (decided against for launch; future phase).
3. Multi-robot fleet management (future phase).
4. Health check HTTP server (was process-compose specific; not needed for single binary).
5. `gorai compile` command or process-compose YAML generation.
6. `gorai component search/add` registry (separate future work).

### 1.3 Definitions

| Term | Meaning |
|---|---|
| **RDL** | Robot Definition Language — the `robot.json` configuration file |
| **Embedded NATS** | NATS server running in-process within the gorai binary using `github.com/nats-io/nats-server/v2/server` |
| **External NATS** | A NATS server running as a separate process, managed independently by the user |

---

## 2. Requirements

### 2.1 Embedded NATS Server

**REQ-EMBED-1: Default to embedded NATS.**
When `gorai run` is invoked and the NATS URL is local (localhost or 127.0.0.1) and `nats.external` is not set to `true` in the RDL, the runtime MUST start an embedded NATS server in-process before connecting any clients.

**REQ-EMBED-2: Bind to localhost only.**
The embedded NATS server MUST bind to `127.0.0.1` by default. It MUST NOT bind to `0.0.0.0` or any external interface unless explicitly configured.

**REQ-EMBED-3: Startup ordering.**
The embedded NATS server MUST be fully ready to accept connections before any component or service initialization begins. The runtime MUST block on server readiness with a configurable timeout (default: 10 seconds).

**REQ-EMBED-4: JetStream enabled by default.**
The embedded NATS server MUST enable JetStream by default. JetStream storage directory MUST default to `./data/jetstream/` relative to the working directory.

**REQ-EMBED-5: Shutdown ordering.**
On robot shutdown, the runtime MUST: (1) stop all services, (2) stop all components, (3) drain and close the NATS client connection, (4) shut down the embedded NATS server. The NATS server shutdown MUST wait for in-flight messages to drain.

**REQ-EMBED-6: Port from RDL.**
If the RDL specifies `nats.url` with a port (e.g., `nats://localhost:4223`), the embedded NATS server MUST use that port. If no URL is specified, default to port 4222.

**REQ-EMBED-7: TLS passthrough.**
If the RDL specifies `nats.tls` configuration (CA, cert, key files), those MUST be passed to the embedded NATS server.

### 2.2 External NATS Opt-In

**REQ-EXTERNAL-1: Explicit external flag.**
The RDL `nats` section MUST support an `external` boolean field. When `true`, the runtime MUST NOT start an embedded NATS server and MUST connect to the URL as-is.

**REQ-EXTERNAL-2: Non-local URL implies external.**
If the NATS URL points to a non-local address (not localhost, not 127.0.0.1), the runtime MUST treat it as external and MUST NOT start an embedded NATS server, regardless of the `external` flag.

**REQ-EXTERNAL-3: Backward compatibility.**
Existing RDL files that do not include `nats.external` MUST work without modification. The default behavior (local URL, no external flag) MUST start embedded NATS.

### 2.3 RDL Configuration

**REQ-RDL-1: Minimal valid RDL.**
A valid RDL file MUST NOT require a `nats` section. When omitted, the runtime MUST use embedded NATS on `127.0.0.1:4222` with JetStream enabled.

**REQ-RDL-2: External NATS example.**
```json
{
  "nats": {
    "url": "nats://nats-server.local:4222",
    "external": true
  }
}
```

**REQ-RDL-3: Explicit local with JetStream disabled.**
```json
{
  "nats": {
    "url": "nats://localhost:4222",
    "jetstream": false
  }
}
```
When JetStream is explicitly set to `false`, the embedded NATS server MUST disable JetStream.

### 2.4 User Experience

**REQ-UX-1: No NATS prerequisite.**
The README and documentation MUST NOT list NATS server as a prerequisite for getting started. The quick start MUST be: install Go, build gorai, write JSON, run.

**REQ-UX-2: Startup logging.**
When the embedded NATS server starts, the runtime MUST log: the NATS URL, whether JetStream is enabled, and the JetStream storage directory.

**REQ-UX-3: External NATS guidance.**
The README MUST include a section explaining when and how to use an external NATS server (multi-robot, shared broker, production fleets).

### 2.5 Binary Size and Performance

**REQ-PERF-1: Binary size budget.**
The embedded NATS server dependency SHOULD NOT increase the binary size by more than 15MB over the current baseline.

**REQ-PERF-2: Memory overhead.**
The embedded NATS server with JetStream enabled SHOULD use less than 50MB of additional RAM at idle.

**REQ-PERF-3: Startup time.**
The embedded NATS server MUST be ready to accept connections within 2 seconds on a Raspberry Pi 5.

---

## 3. Implementation Notes

### 3.1 Source

The `pkg/embeddednats/` package from the `process-compose` branch contains a tested implementation of the embedded NATS server wrapper. It should be brought to `main` without the process-compose-specific code (health server, compile command, device commands).

### 3.2 Changes Required

1. **pkg/embeddednats/** — Server wrapper (from process-compose branch)
2. **pkg/config/config.go** — Add `External` field to `NATSConfig`, add `IsLocalURL()` method, add `ShouldEmbedNATS()` method to `RDL`
3. **pkg/robot/robot.go** — Start embedded NATS in `Start()`, shut down in `Stop()`
4. **go.mod** — Add `github.com/nats-io/nats-server/v2` dependency
5. **README.md** — Remove external NATS prerequisite, update quick start and architecture

### 3.3 What NOT to Bring from process-compose Branch

- `pkg/health/` — Process-compose readiness probes (not needed)
- `pkg/compose/` — Process-compose YAML compiler (not needed)
- `cmd/gorai/commands/compile.go` — Compile command (not needed)
- `cmd/gorai/commands/device.go` — Device management (not needed)
- `NOTES.md` — Process-compose notes (not needed)
- Process-compose detection via `PC_PROC_NAME` environment variable
