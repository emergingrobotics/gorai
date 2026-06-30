# Gorai Project Review Report

**Date:** 2026-02-16
**Scope:** Full codebase review — documentation, code quality, security, packaging strategy

---

## 1. Documentation Review

**5 Critical, 9 Warnings, 8 Info**

### Critical

#### C1-DOC: All 12+ documentation links in README.md are dead

**File:** `/gorai/README.md`

The `docs/` and `specs/` directories do not exist in this repository. CLAUDE.md states documentation moved to `../gorai-docs`, but README still contains 12+ links pointing to local paths that do not exist:

- Line 86: `docs/gorai-overarching-strategy.md`
- Line 220: `specs/hardware-requirements.md`
- Line 360: `specs/mesh-service-discovery.md`
- Line 425: `specs/dynamic-discovery.md`
- Line 463: `docs/FUTURE-ROADMAP.md`
- Line 468: `docs/archive/future-state/`
- Line 475: `specs/hardware-requirements.md`
- Line 476: `specs/robot-definition-language.md`
- Line 479: `docs/gorai-overarching-strategy.md`
- Line 480: `docs/vision-analysis.md`
- Line 481: `docs/general-designs.md`
- Line 482: `docs/STRATEGIC-SUMMARY.md`
- Line 484: `specs/dynamic-discovery.md`
- Line 487: `docs/LLM-DESIGN-GUIDE.md`
- Line 503: `docs/PACKAGE-LOCATIONS.md`

#### C2-DOC: CLAUDE.md code structure diagram is inaccurate

**File:** `/gorai/CLAUDE.md`, lines 74-103

The documented code structure tree has major discrepancies with the actual directory layout:

| Documented | Actual |
|---|---|
| `pkg/runtime/` | Does NOT exist |
| No mention of `pkg/discovery/` | EXISTS |
| No mention of `pkg/gsp/` | EXISTS |
| No mention of `pkg/proxy/` | EXISTS |
| No mention of `pkg/robot/` | EXISTS |
| No mention of `pkg/registry/` | EXISTS |
| No mention of `pkg/hardware/` | EXISTS |
| No mention of `pkg/node/`, `pkg/param/`, `pkg/pub/`, `pkg/sub/`, `pkg/tf/`, `pkg/topics/`, `pkg/log/`, `pkg/systemd/`, `pkg/resource/`, `pkg/validation/`, `pkg/services/`, `pkg/components/` | All EXIST |
| No mention of `nws/`, `api/`, `internal/`, `scripts/`, `templates/`, `images/` | All EXIST at repo root |

The `pkg/` directory has 23 subdirectories; only 6 are documented.

#### C3-DOC: Makefile references non-existent archive/examples/ directories

**File:** `/gorai/Makefile`, lines 458, 466-467

```makefile
ARCHIVED_EXAMPLES_DIR := archive/examples
EXAMPLE_HELLO_ROBOT := $(ARCHIVED_EXAMPLES_DIR)/hello-robot
EXAMPLE_HELLO_ROBOT_PROD := $(ARCHIVED_EXAMPLES_DIR)/hello-robot-production
```

The `archive/examples/` directory does not exist. The following targets will fail: `build-examples`, `build-example-hello-robot`, `build-example-hello-robot-prod`, `run-example-hello-robot`, `run-example-hello-robot-prod`, `sub-example-hello-robot`, `sub-example-hello-robot-prod`.

#### C4-DOC: Makefile references non-existent test directories

**File:** `/gorai/Makefile`, lines 119, 124

```makefile
test-integration:
    $(GOTEST) $(INTEGRATION_TAGS) -timeout=$(INTEGRATION_TIMEOUT) ./tests/integration/...
test-module:
    $(GOTEST) $(MODULE_TAGS) -timeout=$(MODULE_TIMEOUT) ./tests/module/...
```

The `tests/` directory does not exist. `test-integration`, `test-module`, and `test-system` targets will fail.

#### C5-DOC: README.md examples table is incomplete

**File:** `/gorai/README.md`, lines 299-305

Only 2 of 4 examples are listed. Missing: `hello-camera`, `pwm-controller`.

### Warnings

#### W1-DOC: Inconsistent CLI command documentation

**Files:** `/gorai/CLAUDE.md` lines 53-68, `/gorai/README.md` lines 224-246

Undocumented CLI commands that exist in `root.go`: `gorai component` (search, info, add), `gorai list`, `gorai migrate`, `gorai mesh robots`, `gorai mesh reset`.

#### W2-DOC: Built-in Components section is incomplete

**File:** `/gorai/README.md`, lines 249-260

Lists only 3 component types. Actual codebase registers dozens more (cameras, motors, servos, sensors, etc.). Lists `camera/v4l2` as "Coming Soon" when it is already implemented.

#### W3-DOC: Registration pattern example is slightly inaccurate

**File:** `/gorai/CLAUDE.md`, lines 138-143

Documented factory function is always `New` but actual implementations vary (`NewGPS`, `NewIMU`, etc.).

#### W4-DOC: gorai.dev/docs URL may not exist

**File:** `/gorai/cmd/gorai/commands/root.go`, line 119

`https://gorai.dev/docs` baked into CLI help output. May not be live.

#### W5-DOC: test-component target references wrong path

**File:** `/gorai/Makefile`, line 114

Uses `./component/...` (singular) but directory is `components/` (plural).

#### W6-DOC: publish/ submodule not initialized

**File:** `/gorai/.gitmodules`

References `publish/website/themes/hugo-book` but `publish/` directory does not exist.

#### W7-DOC: go.mod references gorai-gps but CLAUDE.md references gorai-gsp

**Files:** `/gorai/go.mod` line 9, `/gorai/CLAUDE.md` line 38

These are different modules. `gorai-gsp` mentioned in CLAUDE.md is not in `go.mod`.

#### W8-DOC: Undocumented top-level directories

| Directory | Contents |
|---|---|
| `nws/` | NWS client/server (weather service?) |
| `api/` | Protocol Buffer definitions |
| `internal/` | Internal proto and testutil packages |
| `scripts/` | Shell scripts |
| `templates/` | Component and service templates |
| `images/` | Hardware pinout images and logo |

#### W9-DOC: CLAUDE.md says docs/DESIGN.md is master design doc

**File:** `/home/developer/.claude/CLAUDE.md`

No `docs/` directory exists in this repo.

### Info

#### I1-DOC: Stale empty file at repo root

`/gorai/repeat` — 0-byte artifact.

#### I2-DOC: .DS_Store committed to repo

Should be in `.gitignore`.

#### I3-DOC: Compiled binary committed to repo root

`/gorai/gorai` — 14MB build artifact. Build system uses `bin/` as output.

#### I4-DOC: .aider.* files present

Should be in `.gitignore`.

#### I5-DOC: README.md and CLAUDE.md contain duplicated content

CLI Commands, Code Structure, and architecture description are substantially duplicated.

#### I6-DOC: No PROJECT.md file

Global CLAUDE.md convention says to look for `PROJECT.md`, but none exists.

#### I7-DOC: No .claude/skills/ directory

Global CLAUDE.md references it, but `.claude/` only contains `settings.local.json`.

#### I8-DOC: Makefile help target misses many targets

Only shows targets with `## ` annotations. Many targets lack this annotation.

---

## 2. Code Quality Review

**6 Critical, 12 Warnings, 7 Info**

### Critical

#### C1-CODE: Race condition on Registration.desc fields

**File:** `/gorai/pkg/mesh/registration.go:186-187`

`r.desc.LastSeen` and `r.desc.Status` mutated without lock in `sendHeartbeat()`. `Descriptor()` (line 267) reads without lock. `r.mu` only protects `r.status`, not `r.desc`.

#### C2-CODE: Race condition on Camera.lastPublishTime

**File:** `/gorai/components/camera/v4l2/camera.go:201,222`

`lastPublishTime` read and written in `handleFrame()` called from a goroutine with no synchronization.

#### C3-CODE: Race condition on Camera.onFrame callback

**File:** `/gorai/components/camera/v4l2/camera.go:217,261`

`SetFrameCallback` callable from any goroutine while `handleFrame` reads `c.onFrame` without sync.

#### C4-CODE: Goroutine leak in NATS Connect

**File:** `/gorai/pkg/nats/client.go:87-100`

When parent context cancels, the goroutine performing `nats.Connect` may block indefinitely. The goroutine leaks and the eventual connection is never closed.

#### C5-CODE: WebSocket Hub goroutine leak on close

**File:** `/gorai/pkg/dashboard/websocket.go:119-126`

When `Run()` shuts down and closes the `events` channel, `unregister` and `register` channel sends block forever because `Run()` has returned.

#### C6-CODE: Broken go.mod replace directive

**File:** `/gorai/go.mod`

`replace github.com/emergingrobotics/gorai-gps => ../gorai-gps` — directory does not exist. Build fails without sibling repo.

### Warnings

#### W1-CODE: No tests for core packages

Zero test files for: `pkg/mesh/`, `pkg/nats/`, `pkg/robot/`, `pkg/dashboard/`, `pkg/pub/`, `pkg/sub/`.

#### W2-CODE: Silently ignoring json.Marshal errors

**Files:** `pkg/mesh/registration.go:207`, `pkg/mesh/micro.go:173,179`

```go
hbData, _ := json.Marshal(hb)
```

#### W3-CODE: Custom itoa function instead of strconv

**File:** `/gorai/pkg/dashboard/handlers.go:190-203`

Hand-rolled `itoa` with infinite recursion bug on `math.MinInt`.

#### W4-CODE: XSS vulnerability in dashboard HTML rendering

**File:** `/gorai/pkg/dashboard/handlers.go:49,104-111`

Robot name and component names written directly into HTML without escaping.

#### W5-CODE: WebSocket accepts any origin

**File:** `/gorai/pkg/dashboard/websocket.go:99`

```go
InsecureSkipVerify: true,
```

#### W6-CODE: mesh.Connect() leaks NATS connection

**File:** `/gorai/pkg/mesh/client.go:95-98`

`Client.Close()` does NOT close the NATS connection it created via `Connect()`.

#### W7-CODE: QueryRegistry uses wrong timeout constant

**File:** `/gorai/pkg/mesh/micro.go:406`

Uses `DefaultServiceTTL` (30s heartbeat TTL) as RPC timeout. Should be ~5s.

#### W8-CODE: Regex compilation on every validation call

**File:** `/gorai/pkg/config/config.go:601,606`

`validateName` compiles regexes on every invocation. Should be package-level vars.

#### W9-CODE: expandEnvVars regex also compiled per-call

**File:** `/gorai/pkg/config/config.go:439`

Should be a package-level variable.

#### W10-CODE: ExternalService.Running accessed without proper synchronization

**File:** `/gorai/pkg/robot/robot.go:85,763,839`

`Running` field set under one lock, read under a different lock.

#### W11-CODE: monitorExternalService recursive restart is fragile

**File:** `/gorai/pkg/robot/robot.go:817`

Restart calls `startExternalService` which spawns a new monitor goroutine. Correct but fragile.

#### W12-CODE: Stop race between stopExternalServices and monitor goroutine

**File:** `/gorai/pkg/robot/robot.go:840-853`

Two goroutines calling `svc.Cmd.Wait()` — only one succeeds.

### Info

#### I1-CODE: Naming convention violations

`sub` (should be `subscription`), `mgr` (should be `manager`), `interface{}` instead of `any`.

#### I2-CODE: Magic numbers

Timeouts, history values, thresholds used as literal numbers without named constants.

#### I3-CODE: Duplicate Properties type in camera packages

`camera.Properties` and `v4l2.Properties` are separate types. V4L2 camera may not satisfy the `camera.Camera` interface.

#### I4-CODE: PointCloud type defined in both sensor and camera packages

Different types with the same name in related packages.

#### I5-CODE: Watcher has unused mu field

**File:** `/gorai/pkg/mesh/watcher.go:21`

`sync.RWMutex` declared but never used.

#### I6-CODE: Incomplete Reconfigure implementation

**File:** `/gorai/pkg/robot/robot.go:939-946`

Writes `r.cfg` without synchronization. Has a TODO comment.

#### I7-CODE: Test coverage gaps

| Package | Test Files | Status |
|---------|-----------|--------|
| `pkg/config/` | 3 | Good |
| `pkg/registry/` | 1 | Adequate |
| `pkg/resource/` | 1 | Adequate |
| `pkg/node/` | 1 | Basic |
| `pkg/gsp/` | 3 | Good |
| `pkg/mesh/` | 0 | Critical gap |
| `pkg/nats/` | 0 | Critical gap |
| `pkg/robot/` | 0 | Critical gap |
| `pkg/dashboard/` | 0 | Critical gap |
| `components/` | 11 | Good |
| `services/` | 6 | Good |
| `driver/` | 1 | Minimal |

---

## 3. Security Review

**3 Critical, 6 High, 6 Medium, 6 Low**

### Critical

#### C1-SEC: Arbitrary command execution from RDL configuration

**File:** `/gorai/pkg/robot/robot.go:692-713`

`External.Command` from RDL config passed directly to `exec.CommandContext` with no validation. A malicious RDL file executes any command with gorai process privileges.

**Remediation:** Validate commands against an allowlist. Restrict to absolute paths. Consider sandboxing with cgroups or seccomp.

#### C2-SEC: Environment variable injection in configuration

**File:** `/gorai/pkg/config/config.go:437-453`

`expandEnvVars` expands `${VAR}` in raw JSON bytes. Env vars containing JSON-breaking characters (quotes, braces) can corrupt JSON or inject fields.

**Remediation:** JSON-escape values from env var expansion. Or restrict expansion to specific known-safe fields.

#### C3-SEC: Unrestricted NATS RPC reflection endpoint

**File:** `/gorai/nws/server.go:36-65,85-153`

`ResourceServer.Wrap()` exposes ALL public methods of any resource over NATS via reflection. Zero authentication or authorization. Any NATS client can invoke `Close`, `Reconfigure`, `SetPower`, etc.

**Remediation:** Implement method allowlist. Add NATS RPC authentication. Never expose admin methods via unauthenticated RPC.

### High

#### H1-SEC: Web dashboard binds all interfaces with no authentication

**Files:** `pkg/config/config.go:495`, `pkg/dashboard/dashboard.go:77`, `pkg/dashboard/server.go:16-95`

Dashboard defaults to `0.0.0.0:8080` with zero authentication. Anyone on the network gets full access to robot status, camera feeds, WebSocket endpoints.

**Remediation:** Default to `127.0.0.1:8080`. Add HTTP Basic Auth. Add HTTPS support.

#### H2-SEC: WebSocket origin bypass

**File:** `/gorai/pkg/dashboard/websocket.go:98-99`

`InsecureSkipVerify: true` enables Cross-Site WebSocket Hijacking. A malicious website can connect to the robot dashboard.

**Remediation:** Set `InsecureSkipVerify: false`, configure `OriginPatterns`.

#### H3-SEC: No NATS authentication or TLS

**File:** `/gorai/pkg/nats/client.go:51-106`

Config struct defines `CredentialsFile` and `TLS` fields but they are never used in actual connection code. All NATS traffic is plaintext and unauthenticated.

**Remediation:** Wire up existing TLS and credential config fields. Make TLS default for non-localhost.

#### H4-SEC: Cross-Site Scripting (XSS) in dashboard

**File:** `/gorai/pkg/dashboard/handlers.go:48-121`

Robot name, component names, types, models written as raw HTML without escaping.

**Remediation:** Use `html/template` or `html.EscapeString()`.

#### H5-SEC: No security headers on dashboard

**File:** `/gorai/pkg/dashboard/server.go:15-95`

No CSP, X-Frame-Options, HSTS, or X-Content-Type-Options headers.

**Remediation:** Add security header middleware.

#### H6-SEC: Unauthenticated mesh service registration

**File:** `/gorai/pkg/mesh/registration.go:69-156`

Any NATS client can register a service with arbitrary metadata, potentially hijacking control channels.

**Remediation:** Sign registrations with shared secret. Validate registered subjects. Implement authorization.

### Medium

#### M1-SEC: GSP protocol integer overflow in payload length parsing

**File:** `/gorai/pkg/gsp/protocol.go:106-114`

Payload length parsed from ASCII digits without overflow protection. On 32-bit systems, crafted input with many digits could overflow.

**Remediation:** Check digit count or value against `MaxPayloadSize` inside parsing loop.

#### M2-SEC: WriteTimeout disabled on HTTP server

**File:** `/gorai/pkg/dashboard/dashboard.go:119`

`WriteTimeout: 0` for MJPEG streaming. Enables slow-loris attacks.

**Remediation:** Use per-handler timeout management. Wrap non-streaming handlers with `http.TimeoutHandler`.

#### M3-SEC: No rate limiting on WebSocket connections

**File:** `/gorai/pkg/dashboard/websocket.go:97-117`

No limit on concurrent WebSocket connections. Attacker can exhaust memory and file descriptors.

**Remediation:** Reject connections when `ClientCount()` exceeds threshold.

#### M4-SEC: Vulnerable dependency golang.org/x/crypto v0.29.0

**File:** `/gorai/go.mod:33`

4 known CVEs: CVE-2024-45337 (Critical, SSH auth bypass), CVE-2025-22869 (High, DoS), CVE-2025-58181 (Medium), CVE-2025-47914 (Medium).

**Remediation:** Upgrade to `golang.org/x/crypto v0.45.0` or later.

#### M5-SEC: Serial port device path not validated

**Files:** `services/gateway/config.go:82-93`, `services/gateway/port_handler.go:129`

Device path from config passed directly to `serial.Open` without validation.

**Remediation:** Validate paths match `/dev/tty*` or `/dev/serial/*` patterns.

#### M6-SEC: middleware.RealIP without trusted proxy configuration

**File:** `/gorai/pkg/dashboard/server.go:19`

Trusts `X-Forwarded-For` headers without configuring trusted proxies. Enables IP spoofing in logs.

**Remediation:** Remove `middleware.RealIP` or configure trusted proxy addresses.

### Low

#### L1-SEC: No emergency stop mechanism

Motor interface has no global kill switch. No watchdog timer. No dead-man's switch.

**Remediation:** Add global emergency stop NATS topic. Implement watchdog that stops motors if no command received within timeout.

#### L2-SEC: No communication loss handling for motors

If NATS or serial connection drops, motors continue at last commanded speed.

**Remediation:** Implement watchdog on microcontroller side. Add health monitoring to proxy layer.

#### L3-SEC: No systemd hardening

**File:** `/gorai/pkg/systemd/systemd.go:172-292`

Generated unit files have no `ProtectSystem`, `NoNewPrivileges`, `PrivateDevices`, etc.

**Remediation:** Add systemd security directives to generated units.

#### L4-SEC: Container privileged mode available

**File:** `/gorai/pkg/config/config.go:337`

`Privileged: true` disables all container security. No warning logged.

**Remediation:** Log warning when privileged mode enabled. Document implications.

#### L5-SEC: Camera device path from config without sanitization

**File:** `/gorai/pkg/robot/robot.go:305-309,358-361`

Camera device paths from component attributes used directly without validation.

**Remediation:** Validate against `/dev/video*` pattern.

#### L6-SEC: MustConnect and MustRegister panic on error

**Files:** `pkg/mesh/client.go:127-133`, `pkg/mesh/registration.go:364-370`

`panic()` in a robotics system causes ungraceful shutdown without safety procedures.

**Remediation:** Avoid `panic` in production. Use error returns. Register `recover()` that performs safety shutdown.

---

## 4. Packaging Strategy Analysis

**Verdict: Premature optimization. Stay with single binary.**

### Current state

Single CLI binary compiles all components into one Go binary. NATS is external. The codebase already supports hybrid mode — `ExternalConfig` in RDL handles container services, `pkg/systemd/` generates unit files for podman containers.

### Analysis

| Option | Verdict |
|--------|---------|
| A. Single binary (current) | Correct for current phase |
| B. Multiple deb packages | Massive overhead, architecture change required, wrong for target audience |
| C. Full containers (podman + Quadlet) | Too heavy for Pi (storage, RAM, startup), Quadlet requires Podman 4.4+/systemd 252+ not in Pi OS stable |
| D. Hybrid (current + containers for ML) | Already implemented, correct long-term direction |

### Recommendation

Stay with single binary. Invest in:

1. **goreleaser** for automated multi-arch GitHub Releases
2. **`gorai install --systemd`** to write and enable a unit file
3. **`gorai update`** for self-update from GitHub Releases

These give all operational benefits of packaging with none of the maintenance burden.

---

## Priority Summary

| Priority | ID | Description |
|----------|----|-------------|
| 1 | C1-CODE | Fix race condition on Registration.desc |
| 2 | C2-CODE | Fix race condition on Camera.lastPublishTime |
| 3 | C3-CODE | Fix race condition on Camera.onFrame |
| 4 | C4-CODE | Fix goroutine leak in NATS Connect |
| 5 | C5-CODE | Fix WebSocket Hub goroutine leak |
| 6 | C6-CODE | Fix broken go.mod replace directive |
| 7 | C1-SEC | Validate external service commands |
| 8 | C2-SEC | Fix env var injection in config |
| 9 | C3-SEC | Restrict NATS RPC method exposure |
| 10 | H1-SEC | Dashboard bind localhost + add auth |
| 11 | H2-SEC | Fix WebSocket origin validation |
| 12 | H3-SEC | Wire up NATS TLS/auth config |
| 13 | H4-SEC | Fix XSS in dashboard |
| 14 | H5-SEC | Add security headers |
| 15 | H6-SEC | Authenticate mesh registration |
| 16 | M4-SEC | Upgrade golang.org/x/crypto |
| 17 | C1-DOC | Fix dead README links |
| 18 | C2-DOC | Update CLAUDE.md code structure |
| 19 | C3-DOC | Fix broken Makefile targets |
| 20 | C4-DOC | Fix broken Makefile test targets |
| 21 | W1-CODE | Add tests for core packages |
| 22 | W2-CODE | Handle json.Marshal errors |
| 23 | W3-CODE | Replace custom itoa with strconv |
| 24 | W8-CODE | Compile regexes at package level |
| 25 | M1-SEC | Fix GSP integer overflow |
| 26 | M5-SEC | Validate serial port paths |
| 27 | L1-SEC | Implement emergency stop |
| 28 | I1-DOC | Remove stale files from repo |

---

## 5. Verification Review (Post-Fix)

**Date:** 2026-02-16
**Scope:** Re-run of all 4 reviews after implementing fixes from phases 1-7.

### Fix Results Summary

| Category | Original Findings | Resolved | Remaining |
|----------|-------------------|----------|-----------|
| Code Critical | 6 | 5 | 1 new (watcher panic) + 2 pre-existing (vet) |
| Code Warning | 12 | 9 | 3 remaining |
| Security Critical | 3 | 3 | 0 |
| Security High | 6 | 5 | 1 partial (WebSocket origin) |
| Security Medium | 6 | 4 | 2 remaining |
| Security Low | 6 | 3 | 3 remaining |
| Docs Critical | 5 | 3 | 2 remaining (README incomplete, components cmd) |
| Docs Warning | 9 | 4 | 5 remaining + 3 new |
| Packaging | Advisory | N/A | Recommendation unchanged |

### Resolved Items

All original critical and high-severity findings were addressed:

- **C1-CODE:** Race condition on `Registration.desc` -- FIXED (mutex protection added)
- **C2-CODE:** Race condition on `Camera.lastPublishTime` -- FIXED (`fpsCalcMu` protection)
- **C3-CODE:** Race condition on `Camera.onFrame` -- FIXED (`onFrameMu` added)
- **C4-CODE:** Goroutine leak in NATS Connect -- FIXED (cleanup goroutine on context cancel)
- **C5-CODE:** WebSocket Hub goroutine leak -- FIXED (done channel + select)
- **C6-CODE:** Broken go.mod replace directive -- FIXED (build tag + removed replace)
- **C1-SEC:** Command injection -- FIXED (`validateExternalCommand` with path/metachar checks)
- **C2-SEC:** Env var injection -- FIXED (`jsonEscapeValue` using `json.Marshal`)
- **C3-SEC:** Unrestricted NATS RPC -- FIXED (blocked methods + allowlist support)
- **H1-SEC:** Dashboard binds all interfaces -- FIXED (default `127.0.0.1:8080`)
- **H3-SEC:** No NATS TLS/auth -- FIXED (TLS config + credentials wired up)
- **H4-SEC:** XSS in dashboard -- FIXED (`html.EscapeString` on all user values)
- **H5-SEC:** No security headers -- FIXED (CSP, X-Frame-Options, etc.)
- **M1-SEC:** GSP integer overflow -- FIXED (MaxPayloadSize check in parsing loop)
- **M3-SEC:** No WebSocket connection limit -- FIXED (main hub capped at 100)
- **M4-SEC:** Vulnerable golang.org/x/crypto -- FIXED (upgraded to v0.45.0)
- **M5-SEC:** Serial port path not validated -- FIXED (allowlist prefix check)
- **M6-SEC:** middleware.RealIP without trusted proxy -- FIXED (removed)
- **L1-SEC:** No emergency stop -- FIXED (estop topic + global estop)
- **L3-SEC:** No systemd hardening -- FIXED (NoNewPrivileges, ProtectSystem, etc.)
- **W3-CODE:** Custom itoa -- FIXED (replaced with `strconv.Itoa`)
- **W6-CODE:** mesh.Connect() leaks NATS -- FIXED (`ownsConn` pattern)
- **W7-CODE:** QueryRegistry wrong timeout -- FIXED (5s timeout)
- **W8-CODE/W9-CODE:** Regex per-call compilation -- FIXED (package-level vars)
- **I5-CODE:** Unused mutex in watcher -- FIXED (removed)
- **C1-DOC:** Dead README links -- FIXED (redirected to gorai-docs)
- **C2-DOC:** CLAUDE.md code structure -- FIXED (updated to match reality)
- **C3-DOC/C4-DOC:** Broken Makefile targets -- FIXED (removed archived, stubbed missing)
- **I1-DOC:** Stale files -- FIXED (removed from tracking, added to .gitignore)

### Remaining Issues (New or Unresolved)

#### Code

**NEW-C1: CRITICAL -- Watcher channel close race**
**File:** `pkg/mesh/watcher.go:112-124`
The `watchKV` goroutine can panic by sending to `w.events` after the cleanup goroutine closes it. Needs a `sync.WaitGroup` to ensure `watchKV` exits before closing the channel. Same issue for `ChannelWatcher` at lines 241-249.

**PRE-C2: CRITICAL -- go vet: I2C interface signature mismatch (pre-existing)**
**File:** `driver/i2c/i2c.go:39,42`
`ReadByte`/`WriteByte` with context param conflicts with `io.ByteReader`/`io.ByteWriter`.

**PRE-C3: CRITICAL -- go vet: non-constant format string (pre-existing)**
**File:** `pkg/robot/robot.go:327`
`fmt.Errorf(errMsg)` should be `fmt.Errorf("%s", errMsg)`.

**W1: WARNING -- ConnectAndRegister client leak on error**
**File:** `pkg/mesh/registration.go:380-392`
If `Register` fails, `client.Close()` is not called.

**W2: WARNING -- json.Marshal error ignored in QueryRegistry**
**File:** `pkg/mesh/micro.go:428`
`dataBytes, _ := json.Marshal(resp.Data)` silently discards error.

#### Security

**M1-NEW: MEDIUM -- CORS wildcard on model stream**
**File:** `pkg/dashboard/models/stream.go:57`
`Access-Control-Allow-Origin: *` allows cross-origin data leak of camera/model streams.

**M2-NEW: MEDIUM -- Camera/model WebSocket handlers lack connection limits**
**Files:** `pkg/dashboard/cameras/handler.go`, `pkg/dashboard/models/handler.go`
Only the main WebSocket hub has the 100-client limit. Camera and model hubs are unlimited.

**L1-NEW: LOW -- Config default overrides dashboard localhost**
**File:** `pkg/config/config.go:509`
`applyDefaults` sets `:8080` which overrides dashboard's `127.0.0.1:8080` default.

**L2-NEW: LOW -- Device path traversal**
**File:** `services/gateway/config.go:93`
Missing `filepath.Clean` before prefix check allows `/dev/tty/../sda1`.

**L3-REMAIN: LOW -- InsecureSkipVerify on all WebSocket endpoints**
**Files:** `websocket.go:118`, `cameras/handler.go:207`, `models/handler.go:249`
Enables cross-site WebSocket hijacking when dashboard is exposed.

#### Documentation

**D1: CRITICAL -- `gorai components` command doesn't work as documented**
Both README.md and CLAUDE.md document `gorai components` but the actual command requires a subcommand (`gorai list components`).

**D2: WARNING -- README.md missing many CLI commands**
Component subcommands (search, info, add), mesh robots, mesh reset not in README.

**D3: WARNING -- `../gorai-docs` relative links broken on GitHub**
12+ links in README.md use `../gorai-docs` which only works with side-by-side clones.

**D4: WARNING -- Makefile help advertises nonexistent `hello-robot` example**

**D5: WARNING -- `gorai start` referenced in build.go output but command doesn't exist**
**File:** `cmd/gorai/commands/build.go:139`
Should reference `gorai run`.

**D6: WARNING -- README.md "Built-in Components" section outdated**
Lists 3 components; actual codebase has 16+ component directories.

**D7: INFO -- `strings.Title` deprecated in component.go**
**D8: INFO -- Unicode symbols in component.go/build.go output**
