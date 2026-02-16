# Gorai Fix Plan

Detailed implementation plan for all issues found in the project review.
Items are ordered by priority: critical code bugs first, then security, then docs, then cleanup.

---

## Phase 1: Critical Code Bugs (Race Conditions and Leaks)

### Step 1: Fix race condition on Registration.desc (C1-CODE)

**File:** `pkg/mesh/registration.go`

1. Read current implementation of `sendHeartbeat()` and `Descriptor()`
2. Extend `r.mu` to protect `r.desc` fields, not just `r.status`
3. In `sendHeartbeat()`: acquire `r.mu` before writing `r.desc.LastSeen` and `r.desc.Status`
4. In `Descriptor()`: acquire `r.mu.RLock()`, return a copy of `r.desc`, unlock
5. Audit all other reads of `r.desc` fields for lock coverage

### Step 2: Fix race conditions in Camera (C2-CODE, C3-CODE)

**File:** `components/camera/v4l2/camera.go`

1. Add a `sync.Mutex` to the Camera struct (or use existing one if present)
2. Protect `lastPublishTime` reads (line 201) and writes (line 222) with the mutex
3. Protect `onFrame` reads in `handleFrame()` (line 217) and writes in `SetFrameCallback()` (line 261) with the mutex
4. Verify no deadlocks by checking lock ordering with the V4L2 driver callback

### Step 3: Fix goroutine leak in NATS Connect (C4-CODE)

**File:** `pkg/nats/client.go`

1. Read the current `Connect` function (lines 87-100)
2. Add cleanup goroutine: when context cancels, wait for the connect goroutine to finish, then close the connection if it succeeded
3. Implementation:
   ```go
   select {
   case <-ctx.Done():
       go func() {
           <-done
           if conn != nil {
               conn.Close()
           }
       }()
       return nil, ctx.Err()
   case <-done:
   ```

### Step 4: Fix WebSocket Hub goroutine leak (C5-CODE)

**File:** `pkg/dashboard/websocket.go`

1. Read the full `HandleWebSocket` and `Run` methods
2. Add a `done` channel or context to the Hub struct
3. Close `done` when `Run()` returns
4. In `HandleWebSocket`, use select on `done` when sending to register/unregister channels:
   ```go
   select {
   case h.unregister <- conn:
   case <-h.done:
   }
   ```
5. Same for register channel

### Step 5: Fix broken go.mod replace directive (C6-CODE)

**File:** `go.mod`

1. Evaluate whether `gorai-gps` is actually used — check imports in `components/serial/gps.go`
2. If the GPS component can compile without the replace (using a published module version), remove the replace directive
3. If no published version exists and GPS is not critical, add a build tag to make the GPS component optional, or comment the import with a clear TODO
4. Ensure `go vet ./...` passes after the change

---

## Phase 2: Critical Security Fixes

### Step 6: Validate external service commands (C1-SEC)

**File:** `pkg/robot/robot.go`

1. Read `startExternalService` (lines 692-713)
2. Add validation before `exec.CommandContext`:
   - Command must be an absolute path
   - Command path must exist and be executable
   - Reject shell metacharacters in args
   - Consider an allowlist configurable via robot config
3. Log a warning with the exact command being executed

### Step 7: Fix environment variable injection (C2-SEC)

**File:** `pkg/config/config.go`

1. Read `expandEnvVars` (lines 437-453)
2. JSON-escape the substituted values: after retrieving env var value, run it through `json.Marshal` (which escapes special characters), then strip the surrounding quotes
3. Add test cases for env vars containing quotes, braces, backslashes

### Step 8: Restrict NATS RPC method exposure (C3-SEC)

**File:** `nws/server.go`

1. Read `ResourceServer.Wrap()` (lines 36-65) and `invoke` (lines 85-153)
2. Add an allowlist parameter to `Wrap()`:
   ```go
   func (s *ResourceServer) Wrap(res interface{}, allowedMethods ...string) error
   ```
3. If allowedMethods is provided, only register those methods
4. If empty, register all (backwards compat) but log a warning
5. Exclude administrative methods (`Close`, `Reconfigure`) by default

### Step 9: Dashboard bind localhost + add auth (H1-SEC)

**Files:** `pkg/config/config.go`, `pkg/dashboard/dashboard.go`, `pkg/dashboard/server.go`

1. Change default bind address from `:8080` to `127.0.0.1:8080`
2. Add `Username` and `Password` fields to dashboard config
3. Add HTTP Basic Auth middleware in `server.go` that checks credentials if configured
4. When no credentials are configured and bind is not localhost, log a prominent warning

### Step 10: Fix WebSocket origin validation (H2-SEC)

**File:** `pkg/dashboard/websocket.go`

1. Change `InsecureSkipVerify: true` to `InsecureSkipVerify: false`
2. Add `OriginPatterns` that matches the dashboard's own host
3. Make origin patterns configurable via dashboard config

### Step 11: Wire up NATS TLS/auth config (H3-SEC)

**Files:** `pkg/nats/client.go`, `pkg/robot/robot.go`

1. Read the existing `NATSConfig` struct in `pkg/config/config.go` — it already has `CredentialsFile` and `TLS` fields
2. In `pkg/nats/client.go`, add NATS options for:
   - `nats.UserCredentials(cfg.CredentialsFile)` when set
   - `nats.Secure(tlsConfig)` when TLS is configured
3. In `pkg/robot/robot.go` `connectNATS`, pass through the TLS and credential config
4. Add a helper to construct `*tls.Config` from the TLS config fields

### Step 12: Fix XSS in dashboard (H4-SEC)

**File:** `pkg/dashboard/handlers.go`

1. Add `import "html"` at the top
2. Replace all raw `w.Write([]byte(value))` calls in HTML context with `w.Write([]byte(html.EscapeString(value)))`
3. Specific lines: 49, 104, 106, 108, 110, and any others writing config-derived values into HTML
4. Long-term: refactor to use `html/template` package

### Step 13: Add security headers (H5-SEC)

**File:** `pkg/dashboard/server.go`

1. Add a middleware function before route handlers:
   ```go
   func securityHeaders(next http.Handler) http.Handler {
       return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
           w.Header().Set("X-Frame-Options", "DENY")
           w.Header().Set("X-Content-Type-Options", "nosniff")
           w.Header().Set("Content-Security-Policy", "default-src 'self'")
           w.Header().Set("Referrer-Policy", "no-referrer")
           next.ServeHTTP(w, r)
       })
   }
   ```
2. Register with `r.Use(securityHeaders)`

### Step 14: Upgrade golang.org/x/crypto (M4-SEC)

**File:** `go.mod`

1. Run `go get golang.org/x/crypto@latest`
2. Run `go mod tidy`
3. Verify build passes

---

## Phase 3: Remaining Security Fixes

### Step 15: Authenticate mesh service registration (H6-SEC)

**File:** `pkg/mesh/registration.go`

1. Design: add an optional shared secret to mesh client config
2. When registering, include an HMAC of the service descriptor signed with the secret
3. In discovery/watchers, validate the HMAC before trusting a registration
4. When no secret is configured, log a warning but allow (backwards compat for development)

### Step 16: Fix GSP integer overflow (M1-SEC)

**File:** `pkg/gsp/protocol.go`

1. Read the payload length parsing loop (lines 106-114)
2. Add a check inside the loop: if `payloadLen > MaxPayloadSize`, return error immediately
3. Also limit the number of digits parsed (e.g., max 10 digits)

### Step 17: Validate serial port device paths (M5-SEC)

**File:** `services/gateway/config.go`

1. Add a validation function:
   ```go
   func validateDevicePath(path string) error {
       if !strings.HasPrefix(path, "/dev/tty") && !strings.HasPrefix(path, "/dev/serial/") {
           return fmt.Errorf("invalid device path: %s", path)
       }
       return nil
   }
   ```
2. Call it in config validation before opening the port
3. Apply same pattern to camera device paths in `pkg/robot/robot.go`

### Step 18: Add per-handler timeouts for dashboard (M2-SEC)

**File:** `pkg/dashboard/server.go`

1. Keep `WriteTimeout: 0` for the server (needed for streaming)
2. Wrap non-streaming handlers with `http.TimeoutHandler(handler, 30*time.Second, "timeout")`
3. Leave MJPEG stream and WebSocket handlers unwrapped

### Step 19: Add WebSocket connection limit (M3-SEC)

**File:** `pkg/dashboard/websocket.go`

1. Add a `maxClients` field to `WebSocketHub` (default 100)
2. In `HandleWebSocket`, check `h.ClientCount()` before accepting
3. Return 503 if limit exceeded

### Step 20: Remove middleware.RealIP or configure it (M6-SEC)

**File:** `pkg/dashboard/server.go`

1. Remove `r.Use(middleware.RealIP)` since the dashboard is not expected to run behind a reverse proxy
2. If reverse proxy support is needed later, make it configurable with trusted proxy addresses

---

## Phase 4: Code Quality Warnings

### Step 21: Handle json.Marshal errors (W2-CODE)

**Files:** `pkg/mesh/registration.go:207`, `pkg/mesh/micro.go:173,179`

1. Replace `hbData, _ := json.Marshal(hb)` with proper error handling
2. In `registration.go`: log error and return from heartbeat
3. In `micro.go` `respond()` and `respondError()`: return the error to caller

### Step 22: Replace custom itoa with strconv (W3-CODE)

**File:** `pkg/dashboard/handlers.go`

1. Remove the custom `itoa` function (lines 190-203)
2. Replace all calls to `itoa()` with `strconv.Itoa()`
3. Add `"strconv"` to imports

### Step 23: Compile regexes at package level (W8-CODE, W9-CODE)

**File:** `pkg/config/config.go`

1. Extract regex patterns from `validateName` and `expandEnvVars` to package-level vars:
   ```go
   var (
       nameStartRegex = regexp.MustCompile(`^[a-zA-Z]`)
       envVarRegex    = regexp.MustCompile(`\$\{([^}:]+)(?::-([^}]*))?\}`)
   )
   ```
2. Update the functions to use these compiled regexes

### Step 24: Fix QueryRegistry timeout (W7-CODE)

**File:** `pkg/mesh/micro.go`

1. Add a new constant: `const DefaultRPCTimeout = 5 * time.Second`
2. Replace `DefaultServiceTTL` with `DefaultRPCTimeout` on line 406

### Step 25: Fix mesh Client.Close() NATS connection leak (W6-CODE)

**File:** `pkg/mesh/client.go`

1. Read `Client` struct and `Close()` method
2. Track whether the client owns the NATS connection (created via `Connect()` vs passed in)
3. If owned, close the NATS connection in `Close()`

### Step 26: Fix ExternalService.Running synchronization (W10-CODE)

**File:** `pkg/robot/robot.go`

1. Read the external service management code
2. Ensure `Running` is accessed under a consistent lock
3. Consider using `atomic.Bool` instead of a plain bool

### Step 27: Remove unused Watcher mutex (I5-CODE)

**File:** `pkg/mesh/watcher.go`

1. Remove the unused `mu sync.RWMutex` field from the `Watcher` struct

---

## Phase 5: Documentation Fixes

### Step 28: Fix dead README links (C1-DOC)

**File:** `README.md`

1. Read all links in README.md
2. For each dead `docs/` or `specs/` link, either:
   - Update to point to gorai-docs repo (GitHub URL), or
   - Remove the link if the target no longer exists
3. Add a note at the top of any "References" section pointing to the gorai-docs repo

### Step 29: Update CLAUDE.md code structure (C2-DOC)

**File:** `CLAUDE.md`

1. Run `ls` on all top-level and `pkg/` directories
2. Rewrite the code structure tree to match reality
3. Remove `pkg/runtime/` (doesn't exist)
4. Add all missing packages with brief descriptions
5. Add missing top-level directories (`nws/`, `api/`, `internal/`, `scripts/`, `templates/`)

### Step 30: Fix broken Makefile targets (C3-DOC, C4-DOC, W5-DOC)

**File:** `Makefile`

1. Remove or update targets referencing `archive/examples/` (lines 458, 466-467, and dependent targets)
2. Remove or update targets referencing `tests/integration/`, `tests/module/`, `tests/system/`
3. Fix `test-component` to use `./components/...` (plural)
4. Ensure `make help` lists the corrected targets
5. Run `make help` to verify

### Step 31: Update README examples table (C5-DOC)

**File:** `README.md`

1. Add `hello-camera` and `pwm-controller` to the examples table
2. Verify each example directory exists and has a README

### Step 32: Document missing CLI commands (W1-DOC)

**Files:** `CLAUDE.md`, `README.md`

1. Read `cmd/gorai/commands/root.go` for the full command list
2. Add `gorai component`, `gorai list`, `gorai migrate`, `gorai mesh robots`, `gorai mesh reset` to both CLAUDE.md and README.md CLI sections

### Step 33: Update Built-in Components section (W2-DOC)

**File:** `README.md`

1. Scan all `init()` registrations in `components/` and `services/`
2. Update the "Built-in Components" table to reflect reality
3. Move `camera/v4l2` from "Coming Soon" to "Implemented"

### Step 34: Document undocumented directories (W8-DOC)

**Files:** `CLAUDE.md`, `README.md`

1. Add brief descriptions for `nws/`, `api/`, `internal/`, `scripts/`, `templates/`, `images/`

---

## Phase 6: Cleanup

### Step 35: Remove stale files from repo (I1-DOC, I2-DOC, I3-DOC, I4-DOC)

1. Delete `/gorai/repeat` (empty artifact)
2. Delete `/gorai/.DS_Store`
3. Delete `/gorai/gorai` (14MB compiled binary — build system uses `bin/`)
4. Add to `.gitignore`: `.DS_Store`, `.aider.*`, `gorai` (root binary)
5. Stage deletions and .gitignore update

### Step 36: Add missing help annotations to Makefile (I8-DOC)

**File:** `Makefile`

1. Add `## ` annotations to undocumented but useful targets
2. Run `make` (no args) to verify help output

---

## Phase 7: Robotics Safety

### Step 37: Implement emergency stop mechanism (L1-SEC)

1. Define a global emergency stop NATS topic (e.g., `gorai.estop`)
2. Add estop subscription to all actuator proxies (motor, servo, thruster)
3. When estop received: set all actuators to zero/safe state, reject further commands until cleared
4. Add `gorai estop` CLI command to publish the stop message
5. Add estop button to web dashboard

### Step 38: Add communication loss watchdog (L2-SEC)

1. Add a watchdog timer to motor/servo proxy: if no command received in N seconds, set power to zero
2. Configure timeout via component attributes (default 1 second)
3. Log warning when watchdog triggers

### Step 39: Add systemd hardening to generated units (L3-SEC)

**File:** `pkg/systemd/systemd.go`

1. Add security directives to the unit file template:
   ```
   NoNewPrivileges=yes
   ProtectSystem=strict
   ProtectHome=true
   PrivateTmp=true
   ```
2. Only add `PrivateDevices=true` when no hardware access is needed

### Step 40: Log warning for privileged containers (L4-SEC)

**File:** `pkg/robot/robot.go` (or wherever containers are started)

1. When `Privileged: true` is set, log a warning:
   ```
   "WARNING: container running in privileged mode — all security isolation disabled"
   ```

---

## Phase 8: Verification

### Step 41: Run build and tests

1. `make build` — verify clean build
2. `make test` — verify all unit tests pass
3. `go vet ./...` — verify no vet warnings
4. `make check` — run full check suite (fmt, vet, lint, test)

### Step 42: Re-run full project review

Repeat the original four-agent review to verify all issues are resolved:

1. Documentation review — check that all links work, structure matches, examples are complete
2. Code quality review — check that race conditions are fixed, errors handled, tests exist
3. Security review — check that auth, TLS, input validation, XSS fixes are in place
4. Packaging strategy — confirm single binary approach is clean

Compare the new report against this fix plan to identify any remaining gaps.

---

## Notes

- Each step should be implemented as a separate commit with a descriptive message
- Run `go vet ./...` after each phase to catch regressions early
- Security fixes (Phase 2-3) should be prioritized if the robot is network-accessible
- Documentation fixes (Phase 5) can be done in parallel with code fixes
- Phase 7 (robotics safety) is important but can be deferred if robots are not yet operating with real actuators
