# Development Environment

A well-configured development environment accelerates learning and productivity. This chapter covers everything you need to start building with Gorai.

**Our Promise**: You should go from zero to running code in under 15 minutes. If it takes longer, that's a bug in our documentation—please file an issue.

Developer experience starts here. We've invested heavily in making setup painless because friction at the beginning turns away contributors. Every minute you spend fighting tools is a minute you're not building robots.

## Prerequisites

### Go 1.21+

Gorai requires Go 1.21 or later for generics support.

**Linux (apt)**:

```bash
sudo add-apt-repository ppa:longsleep/golang-backports
sudo apt update
sudo apt install golang-go
```

**Linux (manual)**:

```bash
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**macOS**:

```bash
brew install go
```

Verify:

```bash
go version
# go version go1.22.0 linux/amd64
```

### NATS Server

**Linux/macOS**:

```bash
# Via package manager
brew install nats-server  # macOS
# or
go install github.com/nats-io/nats-server/v2@latest

# Or download binary
curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.7/nats-server-v2.10.7-linux-amd64.zip -o nats-server.zip
unzip nats-server.zip
sudo mv nats-server-v2.10.7-linux-amd64/nats-server /usr/local/bin/
```

Verify:

```bash
nats-server --version
```

### Protocol Buffers Toolchain

For working with proto files:

```bash
# Install protoc compiler
# Linux
sudo apt install -y protobuf-compiler

# macOS
brew install protobuf

# Install Go plugins
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Install buf (recommended)
go install github.com/bufbuild/buf/cmd/buf@latest
```

### Optional: TinyGo

For microcontroller development:

```bash
# Linux
wget https://github.com/tinygo-org/tinygo/releases/download/v0.30.0/tinygo_0.30.0_amd64.deb
sudo dpkg -i tinygo_0.30.0_amd64.deb

# macOS
brew tap tinygo-org/tools
brew install tinygo

# Verify
tinygo version
```

## Project Setup

### Clone the Repository

```bash
git clone https://github.com/gorai/gorai
cd gorai
```

### Download Dependencies

```bash
go mod download
```

### Verify Build

```bash
go build ./...
```

### Run Tests

```bash
go test ./...
```

If tests pass, your environment is ready.

## Essential Tools

### nats-server

The message broker:

```bash
# Start with JetStream enabled
nats-server -js

# With verbose logging
nats-server -js -V

# With config file
nats-server -c nats.conf
```

### nats CLI

The command-line client:

```bash
go install github.com/nats-io/natscli/nats@latest

# Basic commands
nats sub ">"        # Subscribe to all
nats pub foo bar    # Publish message
nats server info    # Server status
```

### buf

Protocol buffer management:

```bash
# Generate Go code from protos
buf generate

# Lint proto files
buf lint

# Check for breaking changes
buf breaking --against .git#branch=main
```

### air (Hot Reload)

Automatic rebuilds during development:

```bash
go install github.com/air-verse/air@latest

# Run with hot reload
air
```

Configure with `.air.toml`:

```toml
[build]
cmd = "go build -o ./tmp/main ./cmd/myapp"
bin = "./tmp/main"
include_ext = ["go", "proto"]
```

### dlv (Debugger)

Go debugger:

```bash
go install github.com/go-delve/delve/cmd/dlv@latest

# Debug a program
dlv debug ./examples/hello-sensor

# Attach to running process
dlv attach <pid>
```

## IDE Configuration

### VS Code

Recommended extensions:

- **Go** (golang.go): Essential Go support
- **NATS** (nats-io.vscode-nats): NATS syntax and tools
- **Proto 3** (zxh404.vscode-proto3): Protocol buffer support

Settings (`.vscode/settings.json`):

```json
{
    "go.useLanguageServer": true,
    "go.lintTool": "golangci-lint",
    "go.testFlags": ["-v"],
    "editor.formatOnSave": true,
    "[go]": {
        "editor.defaultFormatter": "golang.go"
    }
}
```

Launch config (`.vscode/launch.json`):

```json
{
    "version": "0.2.0",
    "configurations": [
        {
            "name": "Hello Sensor",
            "type": "go",
            "request": "launch",
            "mode": "debug",
            "program": "${workspaceFolder}/examples/hello-sensor",
            "args": ["-fake", "-fake-temp", "50"]
        }
    ]
}
```

### GoLand

GoLand works out of the box with Go projects. Recommended settings:

- Enable "Optimize imports on the fly"
- Configure "Go | Go Modules" for the project
- Set up run configurations for examples

## Scripts and Automation

Gorai includes helper scripts:

### Makefile Targets

```makefile
.PHONY: build test run clean

build:
	go build ./...

test:
	go test ./...

test-quick:
	go test -tags=component ./...

test-all:
	go test -tags="component integration" ./...

run-hello:
	go run ./examples/hello-sensor -fake

proto:
	buf generate

lint:
	golangci-lint run

nats-start:
	nats-server -js &

nats-stop:
	pkill nats-server

clean:
	rm -rf tmp/ bin/
```

## Hardware Setup

### Reference Platform: Raspberry Pi 5

The recommended starting platform:

**OS Installation**:

1. Download Raspberry Pi OS (64-bit)
2. Flash with Raspberry Pi Imager
3. Enable SSH in settings
4. Boot and connect

**Go Installation**:

```bash
# On the Pi
wget https://go.dev/dl/go1.22.0.linux-arm64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-arm64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

**Clone and Build**:

```bash
git clone https://github.com/gorai/gorai
cd gorai
go build ./...
```

**GPIO Access**:

```bash
# Add user to gpio group
sudo usermod -aG gpio $USER
# Logout and login for group change
```

### Other Supported Boards

| Board | CPU | RAM | NPU | Notes |
|-------|-----|-----|-----|-------|
| Raspberry Pi 5 | Cortex-A76 | 8GB | No | Best starter board |
| Orange Pi 5 | RK3588S | 8-16GB | 6 TOPS | Best for AI |
| Jetson Orin Nano | Cortex-A78 | 8GB | GPU | CUDA support |
| BeagleBone AI-64 | TDA4VM | 4GB | 8 TOPS | Real-time PRUs |

## Microcontroller Development

### TinyGo Setup

Install TinyGo for your platform, then:

```bash
# Flash to RP2040 (Raspberry Pi Pico)
tinygo flash -target=pico ./examples/tinygo/blink

# Flash to ESP32
tinygo flash -target=esp32 ./examples/tinygo/blink
```

### Serial Gateway

Connect microcontroller to Linux board:

```
┌────────────────────┐     USB/UART     ┌────────────────────┐
│   Linux Board      │◄────────────────►│   Microcontroller  │
│   (Go + NATS)      │                  │   (TinyGo)         │
│                    │                  │                    │
│   Serial Gateway   │                  │   Motor Driver     │
│   Node             │                  │   PWM/GPIO         │
└────────────────────┘                  └────────────────────┘
```

The serial gateway translates NATS messages to a compact serial protocol.

## Documentation as First-Class Code

In Gorai, documentation isn't an afterthought—it's as important as the code itself.

**Why?** Because a feature nobody understands is a feature nobody uses. We want contributions, and contributors need to understand the codebase.

**What this means for you:**

- Every public function has godoc comments
- Every package has a doc.go explaining its purpose
- Every example runs and is tested
- Confusing code is a bug

**What we ask from contributors:**

- Add comments to your code
- Update docs when you change behavior
- Write examples that teach
- File issues for confusing documentation

The best documentation is written by people who just learned something—fresh eyes catch assumptions that experts miss.

---

With your development environment configured, Chapter 12 provides a deep dive into the hello-sensor example, showing how all these pieces come together.
