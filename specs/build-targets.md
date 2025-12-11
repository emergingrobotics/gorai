# Build Targets Specification

**Version:** 1.0
**Status:** Draft
**Last Updated:** 2024

## 1. Overview

This specification defines the supported build targets, cross-compilation requirements, and build system conventions for Gorai robot binaries.

### 1.1 Design Goals

1. **Cross-compile from anywhere**: Build ARM binaries on x86, and vice versa
2. **Single command**: `make build-pi` produces deployable binary
3. **Reproducible**: Same source produces identical binaries
4. **No CGO by default**: Pure Go for easy cross-compilation
5. **CGO opt-in**: Support CGO when needed (OpenCV, hardware libs)

---

## 2. Supported Target Platforms

### 2.1 Primary Targets

| Target ID | GOOS | GOARCH | Description | Priority |
|-----------|------|--------|-------------|----------|
| `linux-arm64` | linux | arm64 | Raspberry Pi 4/5, Orange Pi 5, Jetson | Primary |
| `linux-amd64` | linux | amd64 | x86-64 Linux servers, Intel NUCs | Primary |

### 2.2 Secondary Targets

| Target ID | GOOS | GOARCH | Description | Priority |
|-----------|------|--------|-------------|----------|
| `linux-armv7` | linux | arm | Raspberry Pi 3, Pi Zero W, BeagleBone | Secondary |
| `linux-armv6` | linux | arm (GOARM=6) | Raspberry Pi Zero (original) | Secondary |
| `darwin-arm64` | darwin | arm64 | macOS Apple Silicon (dev) | Secondary |
| `darwin-amd64` | darwin | amd64 | macOS Intel (dev) | Secondary |

### 2.3 Future Targets

| Target ID | GOOS | GOARCH | Description | Status |
|-----------|------|--------|-------------|--------|
| `windows-amd64` | windows | amd64 | Windows development | Planned |
| `freebsd-amd64` | freebsd | amd64 | FreeBSD servers | Planned |

---

## 3. Hardware Compatibility Matrix

### 3.1 Single Board Computers

| Board | Architecture | Target ID | Tested | Notes |
|-------|--------------|-----------|--------|-------|
| Raspberry Pi 5 | ARM64 | `linux-arm64` | Yes | Recommended |
| Raspberry Pi 4 | ARM64 | `linux-arm64` | Yes | Recommended |
| Raspberry Pi 3 B+ | ARM64/ARMv7 | `linux-arm64` or `linux-armv7` | Yes | ARM64 preferred |
| Raspberry Pi Zero 2 W | ARM64 | `linux-arm64` | Yes | Limited RAM |
| Raspberry Pi Zero W | ARMv6 | `linux-armv6` | No | Very limited |
| Orange Pi 5 | ARM64 | `linux-arm64` | Yes | High performance |
| Orange Pi 5 Plus | ARM64 | `linux-arm64` | Yes | High performance |
| Jetson Nano | ARM64 | `linux-arm64` | Yes | GPU support |
| Jetson Orin Nano | ARM64 | `linux-arm64` | Yes | GPU support |
| BeagleBone Black | ARMv7 | `linux-armv7` | Planned | |
| Intel NUC | AMD64 | `linux-amd64` | Yes | |

### 3.2 Operating System Requirements

| OS | Version | Status | Notes |
|----|---------|--------|-------|
| Raspberry Pi OS (64-bit) | Bookworm | Supported | Recommended for Pi |
| Raspberry Pi OS (32-bit) | Bookworm | Supported | Use `linux-armv7` |
| Ubuntu Server | 22.04+ | Supported | |
| Ubuntu Desktop | 22.04+ | Supported | |
| Debian | 12+ | Supported | |
| Armbian | Latest | Supported | For Orange Pi |
| Jetson Linux (L4T) | R35+ | Supported | For Jetson |

---

## 4. Build Environment

### 4.1 Go Version Requirements

| Requirement | Version | Notes |
|-------------|---------|-------|
| Minimum | Go 1.21 | Required for generics, slog |
| Recommended | Go 1.22+ | Latest stable |
| Maximum | None | Track latest |

### 4.2 Build Dependencies

#### Required (No CGO)

```bash
# Go toolchain only - no external dependencies for pure Go builds
go version  # Must be 1.21+
```

#### Optional (With CGO)

For builds requiring CGO (OpenCV, hardware-specific libs):

```bash
# For linux-arm64 cross-compilation with CGO
sudo apt install gcc-aarch64-linux-gnu g++-aarch64-linux-gnu

# For linux-armv7 cross-compilation with CGO
sudo apt install gcc-arm-linux-gnueabihf g++-arm-linux-gnueabihf

# For OpenCV support
# (requires cross-compiled OpenCV libraries - complex)
```

### 4.3 Development Tools

| Tool | Purpose | Installation |
|------|---------|--------------|
| `make` | Build orchestration | System package |
| `buf` | Protocol Buffers | `go install github.com/bufbuild/buf/cmd/buf@latest` |
| `golangci-lint` | Linting | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| `nats-server` | Local testing | `go install github.com/nats-io/nats-server/v2@latest` |

---

## 5. Build Commands

### 5.1 Makefile Targets

```makefile
# Default: build for current platform
build:
	go build -o build/bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Build for specific targets
build-linux-arm64:
	GOOS=linux GOARCH=arm64 go build -o build/bin/linux-arm64/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

build-linux-armv7:
	GOOS=linux GOARCH=arm GOARM=7 go build -o build/bin/linux-armv7/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -o build/bin/linux-amd64/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 go build -o build/bin/darwin-arm64/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Convenience aliases
build-pi: build-linux-arm64      # Raspberry Pi 4/5
build-pi3: build-linux-armv7     # Raspberry Pi 3 (32-bit)
build-jetson: build-linux-arm64  # Jetson boards
build-linux: build-linux-amd64   # Generic Linux x86

# Build all targets
build-all: build-linux-arm64 build-linux-armv7 build-linux-amd64 build-darwin-arm64

# Build with version info
build-release:
	go build -ldflags "-X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)" \
		-o build/bin/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
```

### 5.2 Build Variables

| Variable | Description | Default | Example |
|----------|-------------|---------|---------|
| `BINARY_NAME` | Output binary name | Project directory name | `myrobot` |
| `VERSION` | Version string | `dev` | `1.0.0` |
| `COMMIT` | Git commit hash | `git rev-parse --short HEAD` | `abc1234` |
| `DATE` | Build date | `date -u +%Y-%m-%dT%H:%M:%SZ` | `2024-01-15T10:30:00Z` |
| `CGO_ENABLED` | Enable CGO | `0` | `1` |

### 5.3 Build Flags

#### Standard Flags

```bash
# Production build (default)
go build -o binary ./cmd/myrobot

# With debug symbols stripped (smaller binary)
go build -ldflags="-s -w" -o binary ./cmd/myrobot

# With race detector (development only, not for ARM)
go build -race -o binary ./cmd/myrobot
```

#### Version Embedding

```bash
VERSION=1.0.0
COMMIT=$(git rev-parse --short HEAD)
DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)

go build -ldflags "-X main.version=$VERSION -X main.commit=$COMMIT -X main.date=$DATE" \
    -o binary ./cmd/myrobot
```

Access in code:
```go
var (
    version = "dev"
    commit  = "unknown"
    date    = "unknown"
)

func main() {
    fmt.Printf("Version: %s, Commit: %s, Date: %s\n", version, commit, date)
}
```

---

## 6. Binary Output

### 6.1 Output Directory Structure

```
build/
├── bin/
│   ├── myrobot                    # Current platform
│   ├── linux-arm64/
│   │   └── myrobot                # Pi 4/5, Jetson
│   ├── linux-armv7/
│   │   └── myrobot                # Pi 3
│   ├── linux-amd64/
│   │   └── myrobot                # x86 Linux
│   └── darwin-arm64/
│       └── myrobot                # Mac M1/M2
└── release/
    ├── myrobot-1.0.0-linux-arm64.tar.gz
    ├── myrobot-1.0.0-linux-armv7.tar.gz
    ├── myrobot-1.0.0-linux-amd64.tar.gz
    └── checksums.txt
```

### 6.2 Binary Naming Convention

Development builds:
```
{binary_name}
```

Release builds:
```
{binary_name}-{version}-{goos}-{goarch}.tar.gz
```

Examples:
- `myrobot-1.0.0-linux-arm64.tar.gz`
- `myrobot-1.0.0-linux-armv7.tar.gz`

### 6.3 Release Archive Contents

```
myrobot-1.0.0-linux-arm64/
├── myrobot              # Binary
├── robot.json.example   # Example configuration
├── myrobot.service      # systemd unit file
├── install.sh           # Installation script
├── README.md            # Quick start guide
└── LICENSE              # License file
```

---

## 7. CGO Considerations

### 7.1 CGO-Free Builds (Default)

Most Gorai functionality works without CGO:

| Feature | CGO Required | Notes |
|---------|--------------|-------|
| Core runtime | No | Pure Go |
| NATS messaging | No | Pure Go |
| GPIO (sysfs) | No | Pure Go via /sys |
| I2C (sysfs) | No | Pure Go via /dev/i2c-* |
| SPI (sysfs) | No | Pure Go via /dev/spidev* |
| Serial | No | Pure Go |
| Protocol Buffers | No | Pure Go |
| JSON config | No | Pure Go |
| Most sensors | No | Pure Go drivers |

### 7.2 CGO-Required Features

| Feature | Library | Notes |
|---------|---------|-------|
| OpenCV | gocv | Image processing, some vision |
| TensorFlow Lite | tflite-go | ML inference (alternative: pure Go ONNX) |
| V4L2 (advanced) | Some wrappers | Basic V4L2 can be pure Go |
| GPIO (high-speed) | pigpio | Only for very high-speed GPIO |

### 7.3 Cross-Compiling with CGO

```makefile
# Cross-compile for ARM64 with CGO
build-linux-arm64-cgo:
	CGO_ENABLED=1 \
	CC=aarch64-linux-gnu-gcc \
	CXX=aarch64-linux-gnu-g++ \
	GOOS=linux GOARCH=arm64 \
	go build -o build/bin/linux-arm64/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Cross-compile for ARMv7 with CGO
build-linux-armv7-cgo:
	CGO_ENABLED=1 \
	CC=arm-linux-gnueabihf-gcc \
	CXX=arm-linux-gnueabihf-g++ \
	GOOS=linux GOARCH=arm GOARM=7 \
	go build -o build/bin/linux-armv7/$(BINARY_NAME) ./cmd/$(BINARY_NAME)
```

### 7.4 Static vs Dynamic Linking

**Dynamic (default with CGO):**
- Smaller binary
- Requires libraries on target
- May have glibc version issues

**Static:**
```bash
CGO_ENABLED=1 go build -ldflags "-linkmode external -extldflags -static" -o binary ./cmd/myrobot
```
- Larger binary
- No runtime dependencies
- More portable

---

## 8. Build Optimization

### 8.1 Binary Size Reduction

```bash
# Strip debug symbols
go build -ldflags="-s -w" -o binary ./cmd/myrobot

# Use UPX compression (optional, may affect startup time)
upx --best binary
```

Typical sizes:
| Build Type | Size (approx) |
|------------|---------------|
| Default | 15-25 MB |
| Stripped (-s -w) | 10-18 MB |
| UPX compressed | 4-8 MB |

### 8.2 Build Caching

Go build caching is automatic. For CI:

```bash
# Set cache directory
export GOCACHE=/path/to/cache

# Pre-download modules
go mod download
```

### 8.3 Parallel Builds

```makefile
# Build all platforms in parallel
build-all-parallel:
	$(MAKE) -j4 build-linux-arm64 build-linux-armv7 build-linux-amd64 build-darwin-arm64
```

---

## 9. Testing Builds

### 9.1 Verification

```bash
# Check binary architecture
file build/bin/linux-arm64/myrobot
# Output: ELF 64-bit LSB executable, ARM aarch64, version 1 (SYSV), statically linked, ...

file build/bin/linux-armv7/myrobot
# Output: ELF 32-bit LSB executable, ARM, EABI5 version 1 (SYSV), ...
```

### 9.2 QEMU Testing (Optional)

Test ARM binaries on x86:

```bash
# Install QEMU
sudo apt install qemu-user-static

# Run ARM64 binary
qemu-aarch64-static build/bin/linux-arm64/myrobot --version

# Run ARMv7 binary
qemu-arm-static build/bin/linux-armv7/myrobot --version
```

### 9.3 Integration Testing Matrix

| Test Type | linux-arm64 | linux-armv7 | linux-amd64 | darwin-arm64 |
|-----------|-------------|-------------|-------------|--------------|
| Unit tests | CI (QEMU) | CI (QEMU) | CI (native) | CI (native) |
| Integration | Hardware | Hardware | CI (native) | Local |
| Hardware | Pi 4/5 | Pi 3 | NUC | N/A |

---

## 10. CI/CD Integration

### 10.1 GitHub Actions Example

```yaml
name: Build

on: [push, pull_request]

jobs:
  build:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        target:
          - { goos: linux, goarch: arm64, name: linux-arm64 }
          - { goos: linux, goarch: arm, goarm: 7, name: linux-armv7 }
          - { goos: linux, goarch: amd64, name: linux-amd64 }
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build
        env:
          GOOS: ${{ matrix.target.goos }}
          GOARCH: ${{ matrix.target.goarch }}
          GOARM: ${{ matrix.target.goarm }}
        run: |
          go build -o build/bin/${{ matrix.target.name }}/myrobot ./cmd/myrobot

      - uses: actions/upload-artifact@v4
        with:
          name: binary-${{ matrix.target.name }}
          path: build/bin/${{ matrix.target.name }}/
```

### 10.2 Release Workflow

```yaml
name: Release

on:
  push:
    tags: ['v*']

jobs:
  release:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Build all platforms
        run: make build-all-release

      - name: Create checksums
        run: |
          cd build/release
          sha256sum *.tar.gz > checksums.txt

      - name: Create Release
        uses: softprops/action-gh-release@v1
        with:
          files: |
            build/release/*.tar.gz
            build/release/checksums.txt
```

---

## 11. Troubleshooting

### 11.1 Common Build Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `cannot find package` | Missing dependency | Run `go mod tidy` |
| `exec format error` | Wrong architecture | Check GOOS/GOARCH |
| `undefined: ...` | CGO required | Enable CGO or use pure Go alternative |
| `glibc version` | Library mismatch | Build static or match glibc versions |

### 11.2 Cross-Compilation Issues

**Problem:** `gcc not found` when CGO_ENABLED=1

**Solution:**
```bash
# Install cross-compilers
sudo apt install gcc-aarch64-linux-gnu gcc-arm-linux-gnueabihf
```

**Problem:** Binary doesn't run on target

**Solution:**
```bash
# Verify architecture
file binary
# Check dependencies
ldd binary  # Should show "not a dynamic executable" for static builds
```

---

## Appendix A: Quick Reference

### Build Commands Cheat Sheet

```bash
# Development (current platform)
make build

# Raspberry Pi 4/5, Orange Pi 5, Jetson
make build-pi

# Raspberry Pi 3 (32-bit)
make build-pi3

# Linux x86-64
make build-linux

# All platforms
make build-all

# Release with version
VERSION=1.0.0 make build-release
```

### Environment Variables

```bash
export GOOS=linux        # Target OS
export GOARCH=arm64      # Target architecture
export GOARM=7           # ARM version (for GOARCH=arm)
export CGO_ENABLED=0     # Disable CGO (default for cross-compile)
export CC=...            # C compiler for CGO
export CXX=...           # C++ compiler for CGO
```
