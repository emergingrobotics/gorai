# Development Tools Guide

This document covers all development tools required for building, testing, and contributing to Gorai.

---

## Quick Start

Install all development dependencies with one command:

```bash
make dev-deps
```

This installs: buf, golangci-lint, nats-server, nats CLI, and air (hot reload).

---

## Required Tools

### Go Toolchain

| Tool | Minimum Version | Purpose |
|------|-----------------|---------|
| **go** | 1.22+ | Build, test, and run Go code |

#### Install Go

**Linux (Official Download)**:
```bash
# Download (check https://go.dev/dl/ for latest)
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz

# For ARM64 (Raspberry Pi, Jetson, Rock5B):
wget https://go.dev/dl/go1.22.0.linux-arm64.tar.gz

# Install
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# Add to PATH (add to ~/.bashrc or ~/.zshrc)
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$(go env GOPATH)/bin
```

**Linux (Package Manager)**:
```bash
# Fedora
sudo dnf install golang

# Ubuntu/Debian (may not be latest)
sudo apt-get install golang-go

# Arch
sudo pacman -S go
```

**macOS**:
```bash
brew install go
```

**Verify**:
```bash
go version
# go version go1.22.0 linux/amd64
```

---

### Protocol Buffer Tools

| Tool | Version | Purpose |
|------|---------|---------|
| **buf** | 1.28+ | Proto linting, generation, breaking change detection |
| **protoc-gen-go** | 1.31+ | Go code generation from protos |

#### Install buf

**Via Go (Recommended)**:
```bash
go install github.com/bufbuild/buf/cmd/buf@latest
```

**Binary Download**:
```bash
# Linux
curl -sSL https://github.com/bufbuild/buf/releases/download/v1.28.1/buf-Linux-x86_64 -o buf
chmod +x buf
sudo mv buf /usr/local/bin/

# macOS
curl -sSL https://github.com/bufbuild/buf/releases/download/v1.28.1/buf-Darwin-arm64 -o buf
chmod +x buf
sudo mv buf /usr/local/bin/
```

**macOS (Homebrew)**:
```bash
brew install bufbuild/buf/buf
```

**Verify**:
```bash
buf --version
# 1.28.1
```

#### Install protoc-gen-go

```bash
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
```

#### Usage

```bash
# Lint proto files
make proto-lint
# or: cd api && buf lint

# Generate Go code
make proto
# or: cd api && buf generate

# Check for breaking changes
make proto-breaking
# or: cd api && buf breaking --against '.git#branch=main'
```

---

### NATS Tools

| Tool | Version | Purpose |
|------|---------|---------|
| **nats-server** | 2.10+ | Message broker |
| **nats** (CLI) | 0.1.0+ | Administration, debugging |

See [install-nats.md](install-nats.md) for detailed installation and configuration.

**Quick Install**:
```bash
go install github.com/nats-io/nats-server/v2@latest
go install github.com/nats-io/natscli/nats@latest
```

**Verify**:
```bash
nats-server --version
# nats-server: v2.10.22

nats --version
# 0.1.5
```

---

### Code Quality Tools

| Tool | Version | Purpose |
|------|---------|---------|
| **golangci-lint** | 1.55+ | Comprehensive Go linter (runs 50+ linters) |
| **gofmt** | (bundled) | Code formatting |
| **go vet** | (bundled) | Static analysis |

#### Install golangci-lint

**Via Go**:
```bash
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

**Binary Download**:
```bash
# Linux
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(go env GOPATH)/bin v1.55.2

# macOS
brew install golangci-lint
```

**Verify**:
```bash
golangci-lint --version
# golangci-lint has version 1.55.2
```

**Usage**:
```bash
make lint        # Run all linters
make fmt         # Format code
make vet         # Run go vet
make check       # Run fmt-check, vet, lint, test
```

---

### Development Utilities

| Tool | Purpose | Install |
|------|---------|---------|
| **air** | Hot reload for development | `go install github.com/air-verse/air@latest` |
| **pkgsite** | Local Go documentation server | `go install golang.org/x/pkgsite/cmd/pkgsite@latest` |

#### Install air (Hot Reload)

```bash
go install github.com/air-verse/air@latest
```

**Usage**:
```bash
make watch       # Watch files and run tests on change
```

#### Install pkgsite (Local Docs)

```bash
go install golang.org/x/pkgsite/cmd/pkgsite@latest
```

**Usage**:
```bash
make docs-serve  # Serve docs at http://localhost:8080
```

---

## Optional Tools

### JSON Processing

| Tool | Purpose | Install |
|------|---------|---------|
| **jq** | JSON query and manipulation | See below |

**Install jq**:
```bash
# Linux (Debian/Ubuntu)
sudo apt-get install jq

# Linux (Fedora)
sudo dnf install jq

# Linux (Arch)
sudo pacman -S jq

# macOS
brew install jq
```

**Usage examples**:
```bash
# Pretty print JSON
echo '{"name":"gorai"}' | jq .

# Query NATS monitoring
curl -s http://localhost:8222/varz | jq .server_name

# Parse Go test JSON output
go test -json ./... | jq 'select(.Action=="fail")'
```

---

### File Watching (Alternative to air)

| Tool | Purpose | Install |
|------|---------|---------|
| **entr** | Run commands on file changes | See below |
| **watchexec** | Run commands on file changes | See below |

**Install entr**:
```bash
# Linux (Debian/Ubuntu)
sudo apt-get install entr

# Linux (Fedora)
sudo dnf install entr

# macOS
brew install entr
```

**Install watchexec**:
```bash
# Via Cargo (Rust)
cargo install watchexec-cli

# macOS
brew install watchexec
```

**Usage**:
```bash
# Run tests when Go files change
find . -name "*.go" | entr -c go test ./...

# With watchexec
watchexec -e go -- go test ./...
```

---

### Stress Testing

| Tool | Purpose | Install |
|------|---------|---------|
| **stress** | Run tests repeatedly to find flaky tests | `go install golang.org/x/tools/cmd/stress@latest` |

**Usage**:
```bash
# Run test 100 times to check for race conditions
stress -p 4 go test -race ./pkg/node/...
```

---

### Container Runtime

| Tool | Purpose | Install |
|------|---------|---------|
| **podman** | OCI container runtime (recommended) | See below |
| **podman-compose** | Multi-container orchestration | See below |

**Install Podman**:
```bash
# Linux (Fedora)
sudo dnf install podman podman-compose

# Linux (Debian/Ubuntu)
sudo apt-get install podman podman-compose

# macOS
brew install podman podman-compose
podman machine init
podman machine start
```

**Usage**:
```bash
# Run NATS in container
podman run -d --name gorai-nats -p 4222:4222 -p 8222:8222 nats:latest -js

# Use compose file
cd nats && podman-compose up -d
```

---

### TinyGo (Microcontrollers)

| Tool | Purpose | Install |
|------|---------|---------|
| **tinygo** | Go compiler for microcontrollers | See below |

**Install TinyGo**:
```bash
# Linux (download from https://tinygo.org/getting-started/install/)
wget https://github.com/tinygo-org/tinygo/releases/download/v0.30.0/tinygo_0.30.0_amd64.deb
sudo dpkg -i tinygo_0.30.0_amd64.deb

# macOS
brew tap tinygo-org/tools
brew install tinygo
```

**Verify**:
```bash
tinygo version
```

**Usage**:
```bash
make tinygo-check  # Check TinyGo compilation for supported targets
```

---

### macOS-Specific Tools

| Tool | Purpose | Install |
|------|---------|---------|
| **osx-cpu-temp** | Read CPU temperature (for hello-sensor) | `brew install osx-cpu-temp` |

---

## Editor/IDE Setup

### VS Code

Recommended extensions:
- **Go** (`golang.go`) - Official Go extension
- **vscode-proto3** - Protocol Buffer support
- **NATS Tools** - NATS support

**settings.json**:
```json
{
  "go.lintTool": "golangci-lint",
  "go.lintFlags": ["--fast"],
  "go.formatTool": "gofmt",
  "go.testFlags": ["-v"],
  "[go]": {
    "editor.formatOnSave": true,
    "editor.codeActionsOnSave": {
      "source.organizeImports": true
    }
  }
}
```

### GoLand / IntelliJ IDEA

- Enable Go Modules support
- Configure golangci-lint as external tool
- Set up Proto support via Protocol Buffers plugin

### Neovim

Recommended plugins:
- **nvim-lspconfig** with gopls
- **null-ls** with golangci-lint
- **nvim-treesitter** for syntax highlighting

---

## Environment Variables

Add to your shell profile (`~/.bashrc`, `~/.zshrc`):

```bash
# Go
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:$(go env GOPATH)/bin

# Optional: Go module proxy for faster downloads
export GOPROXY=https://proxy.golang.org,direct

# Optional: Disable telemetry
export DO_NOT_TRACK=1
```

---

## Makefile Commands Reference

| Command | Purpose |
|---------|---------|
| `make dev-deps` | Install all development dependencies |
| `make version` | Show versions of all installed tools |
| `make test` | Run unit tests |
| `make test-all` | Run all tests |
| `make lint` | Run linters |
| `make fmt` | Format code |
| `make proto` | Generate protobuf code |
| `make build` | Build binaries |
| `make nats-start` | Start local NATS server |
| `make nats-stop` | Stop local NATS server |
| `make watch` | Watch and test on file changes |
| `make hooks-install` | Install git pre-commit/pre-push hooks |

---

## Verification

After installing all tools, verify with:

```bash
make version
```

Expected output:
```
Gorai Development Environment

Go:       go version go1.22.0 linux/amd64
GOOS:     linux
GOARCH:   amd64

buf:      1.28.1
lint:     golangci-lint has version 1.55.2
nats:     nats-server: v2.10.22
```

---

## Summary: Minimum Development Setup

```bash
# 1. Install Go 1.22+
# (see instructions above for your OS)

# 2. Clone repository
git clone https://github.com/gorai/gorai.git
cd gorai

# 3. Install dev dependencies
make dev-deps

# 4. Verify
make version

# 5. Run tests
make test

# 6. Start developing!
```
