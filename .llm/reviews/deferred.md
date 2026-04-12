# Deferred Review Findings

## Remediated (Critical/High)

| ID | Severity | Finding | Resolution |
|---|---|---|---|
| S-01 | Critical | Command injection in compiler.go podman command construction | Added `shellQuote()` to all user-controlled values in command strings |
| S-02 | High | Same injection in native binary and device reset commands | Applied shellQuote to all interpolated values |
| S-03 | High | Temp file leak in run --compose (syscall.Exec prevents cleanup) | Write to deterministic `process-compose.yaml` path instead of temp file |
| S-04 | High | Env var values unquoted in podman -e flags | Applied shellQuote to all -e key=value pairs |
| H3 | High | json.Encode errors ignored in health server | Added error handling with logger.Error |
| H1-partial | High | Health server missing HTTP timeouts | Added ReadTimeout, WriteTimeout, IdleTimeout |
| Spec | High | --health-listen flag not wired to WithHealthListen | Fixed: now passed to robot.New() |
| Spec | High | Device reset payload mismatch (nil vs {"subsystem":0}) | Fixed: device.go now sends matching payload |
| M3 | Medium | reset_count snake_case in robot.go | Renamed to resetCount |
| M2 | Medium | extractNATSPort dead code in compiler.go | Removed |

## Deferred (Medium/Low)

| ID | Severity | Finding | Rationale for Deferral |
|---|---|---|---|
| S-05 | Medium | No TLS min version set on embedded NATS | NATS server defaults to TLS 1.2+. The embedded server is localhost-only by default; explicit TLS min version is a future hardening item. |
| S-06 | Medium | No NATS URL scheme validation | Process-compose commands are generated at compile time from validated configs, not user HTTP input. SSRF risk is negligible. |
| S-07 | Medium | No path traversal protection on --output flag | The compile command writes to a user-specified path. This is a CLI tool, not a web server. Standard filesystem permissions apply. |
| S-08 | Medium | process-compose.yaml is world-readable (0644) | Fixed to 0600 in run --compose. For gorai compile output, users control file permissions. |
| S-09 | Low | NATS credentials file path not validated | Credentials file is read by the NATS client library, which handles errors. Validating existence at config time would add unnecessary coupling. |
| H1 | High | embeddednats.Start()/Shutdown() lack context.Context parameters | Design specifies this but current implementation works correctly. Adding context support is a backward-compatible enhancement for a follow-up PR. |
| H2 | High | Functional options vs config structs inconsistency | The existing codebase uses both patterns. Standardizing is a refactoring task, not a correctness issue. |
| H4 | Medium | ConfigFromRDL/DetectNATSMode functions not centralized | NATS mode logic is correctly implemented in config.ShouldEmbedNATS() and compose.shouldEmitExternalNATS(). A design doc alignment PR can follow. |
| REQ-TEST-5 | N/A | Integration tests requiring process-compose runtime | Requires process-compose installed in CI. Deferred to CI pipeline setup. |
| REQ-MIGRATE-4 | N/A | Documentation updates | README and docs updates deferred to documentation PR. |
| REQ-REMOVE-1/2/3 | N/A | Archive/remove obsolete code | Code removal is a separate cleanup PR to avoid mixing feature and deletion changes. |
