# Gorai Dependencies

This directory contains installation guides for all external dependencies and development tools required for Gorai.

## Documentation Index

| Document | Description |
|----------|-------------|
| [development-tools.md](development-tools.md) | **Complete development environment setup** - Go, buf, linters, editors, utilities |
| [install-nats.md](install-nats.md) | NATS server and CLI installation guide |

---

## Quick Start

**One command to install all development tools:**

```bash
make dev-deps
```

This installs: buf, golangci-lint, nats-server, nats CLI, and air.

**Verify installation:**

```bash
make version
```

---

## Required Dependencies

| Dependency | Version | Purpose | Install Guide |
|------------|---------|---------|---------------|
| **Go** | 1.22+ | Build toolchain | [development-tools.md](development-tools.md#go-toolchain) |
| **buf** | 1.28+ | Protocol Buffer tooling | [development-tools.md](development-tools.md#protocol-buffer-tools) |
| **protoc-gen-go** | 1.31+ | Go code generation | [development-tools.md](development-tools.md#install-protoc-gen-go) |
| **NATS Server** | 2.10+ | Message broker | [install-nats.md](install-nats.md) |
| **NATS CLI** | 0.1.0+ | Administration & testing | [install-nats.md](install-nats.md) |

## Development Tools

| Tool | Purpose | Install Guide |
|------|---------|---------------|
| **golangci-lint** | Code linting (50+ linters) | [development-tools.md](development-tools.md#code-quality-tools) |
| **air** | Hot reload for development | [development-tools.md](development-tools.md#development-utilities) |
| **jq** | JSON query/manipulation | [development-tools.md](development-tools.md#json-processing) |
| **Podman** | Container runtime | [development-tools.md](development-tools.md#container-runtime) |

## Optional Dependencies

| Tool | Purpose | When Needed |
|------|---------|-------------|
| **osx-cpu-temp** | macOS CPU temperature | Running hello-sensor on macOS |
| **TinyGo** | Microcontroller support | Building for ESP32, Pico, etc. |
| **entr/watchexec** | File watching | Alternative to air |
| **stress** | Stress testing | Finding flaky tests |

---

## Version Requirements Summary

| Component | Minimum Version | Recommended |
|-----------|-----------------|-------------|
| Go | 1.22+ | Latest |
| NATS Server | 2.10+ | 2.10.22+ |
| NATS CLI | 0.1.0+ | 0.1.5+ |
| buf | 1.28+ | Latest |
| protoc-gen-go | 1.31+ | Latest |
| golangci-lint | 1.55+ | Latest |

---

## Go Module Dependencies

All Go library dependencies are managed via `go.mod`. To download:

```bash
go mod download
```

Current direct dependencies:
- `github.com/nats-io/nats.go` - NATS client library
- `google.golang.org/protobuf` - Protocol Buffers runtime

---

## Related Documentation

- [../nats/nats-setup.md](../nats/nats-setup.md) - Complete NATS configuration guide
- [../specs/howto-run-tests.md](../specs/howto-run-tests.md) - Testing guide
- [../Makefile](../Makefile) - All development commands
