# How to Run Gorai Tests

**Version 0.1.0**

This guide provides practical examples for running tests during Gorai development.

---

## Table of Contents

1. [Quick Reference](#quick-reference)
2. [Running Unit Tests](#running-unit-tests)
3. [Running Component Tests](#running-component-tests)
4. [Running Integration Tests](#running-integration-tests)
5. [Running Module Tests](#running-module-tests)
6. [Running System Tests](#running-system-tests)
7. [Running Hardware Tests](#running-hardware-tests)
8. [Filtering Tests](#filtering-tests)
9. [Watch Mode Development](#watch-mode-development)
10. [Interactive Module Testing](#interactive-module-testing)
11. [Coverage Reports](#coverage-reports)
12. [Benchmarks](#benchmarks)
13. [CI/CD Commands](#cicd-commands)
14. [Troubleshooting](#troubleshooting)

---

## Quick Reference

```bash
# Most common commands
go test ./...                                    # All unit tests
go test -v ./pkg/node                            # Verbose, single package
go test -run TestPublish ./...                   # Tests matching pattern
go test -tags=component ./components/...          # Component tests
go test -tags=integration ./tests/integration    # Integration tests
go test -tags=module ./tests/module              # Module tests
go test -cover ./...                             # With coverage
go test -race ./...                              # With race detection
```

### Makefile Shortcuts

```bash
make test              # Unit tests (default)
make test-quick        # Unit + component (fast feedback)
make test-prepush      # Unit + component + integration (before pushing)
make test-all          # All tests (full regression)
make test-coverage     # Generate coverage report
```

---

## Running Unit Tests

Unit tests are the fastest and most frequently run tests.

### All Unit Tests

```bash
# Run all unit tests
go test ./...

# With verbose output
go test -v ./...

# Fail fast (stop on first failure)
go test -failfast ./...
```

### Single Package

```bash
# Test specific package
go test ./pkg/node
go test ./pkg/pub
go test ./components/motor

# Multiple packages
go test ./pkg/node ./pkg/sub ./pkg/pub

# Package and subpackages
go test ./pkg/...
go test ./components/...
```

### Single Test

```bash
# Run specific test by name
go test -v ./pkg/node -run TestNew

# Run test with exact match
go test -v ./pkg/node -run "^TestNew$"

# Run specific subtest
go test -v ./pkg/node -run "TestNew/with_valid_config"
```

### Examples

```bash
# Test the node package
$ go test ./pkg/node
ok      github.com/gorai/gorai/pkg/node    0.023s

# Verbose output showing each test
$ go test -v ./pkg/node
=== RUN   TestNew
=== RUN   TestNew/with_valid_config
=== RUN   TestNew/with_empty_name
--- PASS: TestNew (0.00s)
    --- PASS: TestNew/with_valid_config (0.00s)
    --- PASS: TestNew/with_empty_name (0.00s)
PASS
ok      github.com/gorai/gorai/pkg/node    0.024s

# Test with race detection
$ go test -race ./pkg/node
ok      github.com/gorai/gorai/pkg/node    0.156s
```

---

## Running Component Tests

Component tests verify individual components with fake dependencies.

### All Component Tests

```bash
# All component tests
go test -tags=component ./components/...

# With verbose output
go test -v -tags=component ./components/...

# With race detection
go test -race -tags=component ./components/...
```

### Specific Component

```bash
# Motor component tests
go test -tags=component ./components/motor

# Camera component tests
go test -tags=component ./components/camera

# All sensor components
go test -tags=component ./components/sensor/...
```

### Examples

```bash
# Test motor component
$ go test -tags=component ./components/motor
ok      github.com/gorai/gorai/components/motor    0.089s

# Verbose motor tests
$ go test -v -tags=component ./components/motor
=== RUN   TestGPIOMotor_SetPower
=== RUN   TestGPIOMotor_SetPower/positive_power
=== RUN   TestGPIOMotor_SetPower/negative_power
=== RUN   TestGPIOMotor_SetPower/zero_power
--- PASS: TestGPIOMotor_SetPower (0.02s)
=== RUN   TestGPIOMotor_Stop
--- PASS: TestGPIOMotor_Stop (0.01s)
PASS
ok      github.com/gorai/gorai/components/motor    0.091s
```

---

## Running Integration Tests

Integration tests verify multiple components working together through NATS.

### All Integration Tests

```bash
# All integration tests
go test -tags=integration ./tests/integration/...

# With timeout (integration tests may take longer)
go test -tags=integration -timeout=2m ./tests/integration/...

# With race detection
go test -race -tags=integration ./tests/integration/...
```

### Specific Integration Test

```bash
# Pub/sub integration tests
go test -tags=integration ./tests/integration -run TestPubSub

# Service integration tests
go test -tags=integration ./tests/integration -run TestService

# Action integration tests
go test -tags=integration ./tests/integration -run TestAction
```

### Examples

```bash
# Run all integration tests
$ go test -tags=integration ./tests/integration/...
ok      github.com/gorai/gorai/tests/integration    1.234s

# Verbose pub/sub test
$ go test -v -tags=integration ./tests/integration -run TestPubSub_IMU
=== RUN   TestPubSub_IMU
    integration_test.go:45: starting embedded NATS server
    integration_test.go:52: publisher connected
    integration_test.go:58: subscriber connected
    integration_test.go:71: message received in 12.3ms
--- PASS: TestPubSub_IMU (0.15s)
PASS
ok      github.com/gorai/gorai/tests/integration    0.162s
```

---

## Running Module Tests

Module tests verify complete module lifecycle through NATS.

### All Module Tests

```bash
# All module tests
go test -tags=module ./tests/module/...

# With extended timeout
go test -tags=module -timeout=5m ./tests/module/...

# Verbose
go test -v -tags=module ./tests/module/...
```

### Specific Module

```bash
# Motor module tests
go test -tags=module ./tests/module -run TestMotorModule

# Camera module tests
go test -tags=module ./tests/module -run TestCameraModule

# Vision module tests
go test -tags=module ./tests/module -run TestVisionModule
```

### Examples

```bash
# Test motor module lifecycle
$ go test -v -tags=module ./tests/module -run TestMotorModule_Lifecycle
=== RUN   TestMotorModule_Lifecycle
    motor_test.go:34: starting embedded NATS
    motor_test.go:41: creating fake motor
    motor_test.go:48: starting module
    motor_test.go:55: module ready
=== RUN   TestMotorModule_Lifecycle/SetPower
    motor_test.go:62: sending power command via NATS
    motor_test.go:68: verifying motor state
--- PASS: TestMotorModule_Lifecycle/SetPower (0.12s)
=== RUN   TestMotorModule_Lifecycle/GetPosition
--- PASS: TestMotorModule_Lifecycle/GetPosition (0.08s)
=== RUN   TestMotorModule_Lifecycle/Stop
--- PASS: TestMotorModule_Lifecycle/Stop (0.05s)
--- PASS: TestMotorModule_Lifecycle (2.34s)
PASS
ok      github.com/gorai/gorai/tests/module    2.456s
```

---

## Running System Tests

System tests verify complete robot configurations.

### All System Tests

```bash
# All system tests
go test -tags=system ./tests/system/...

# With extended timeout
go test -tags=system -timeout=10m ./tests/system/...

# Skip slow tests
go test -tags=system -short ./tests/system/...
```

### Specific Robot Configuration

```bash
# Minimal robot test
go test -tags=system ./tests/system -run TestMinimalRobot

# Differential drive robot
go test -tags=system ./tests/system -run TestDifferentialDrive

# Pan-tilt platform
go test -tags=system ./tests/system -run TestPanTilt
```

### Examples

```bash
# Test differential drive robot
$ go test -v -tags=system ./tests/system -run TestDifferentialDrive
=== RUN   TestDifferentialDriveRobot
    robot_test.go:28: loading robot configuration
    robot_test.go:35: starting robot with fake hardware
    robot_test.go:42: robot ready (12 components initialized)
=== RUN   TestDifferentialDriveRobot/MoveForward
--- PASS: TestDifferentialDriveRobot/MoveForward (0.23s)
=== RUN   TestDifferentialDriveRobot/Turn
--- PASS: TestDifferentialDriveRobot/Turn (0.21s)
=== RUN   TestDifferentialDriveRobot/EmergencyStop
--- PASS: TestDifferentialDriveRobot/EmergencyStop (0.15s)
--- PASS: TestDifferentialDriveRobot (5.67s)
PASS
ok      github.com/gorai/gorai/tests/system    5.891s
```

---

## Running Hardware Tests

Hardware tests require physical hardware and are opt-in.

### Prerequisites

1. Hardware must be connected
2. Appropriate permissions (GPIO access, serial ports)
3. Hardware test configuration file

### Running Hardware Tests

```bash
# Raspberry Pi GPIO tests
go test -tags="hardware,raspberry_pi" ./driver/gpio/...

# Raspberry Pi I2C tests
go test -tags="hardware,raspberry_pi" ./driver/i2c/...

# Serial port tests
go test -tags="hardware" ./driver/serial/... -port=/dev/ttyUSB0

# All hardware tests (requires all hardware)
go test -tags=hardware ./...
```

### Platform-Specific Tests

```bash
# Raspberry Pi 5
go test -tags="hardware,raspberry_pi,pi5" ./...

# NVIDIA Jetson
go test -tags="hardware,jetson" ./...

# RK3588 boards (Orange Pi 5, Rock 5B)
go test -tags="hardware,rk3588" ./...
```

### Examples

```bash
# Test GPIO on Raspberry Pi
$ sudo go test -v -tags="hardware,raspberry_pi" ./driver/gpio -run TestBlink
=== RUN   TestBlink
    gpio_test.go:23: opening GPIO pin 18
    gpio_test.go:28: blinking LED 10 times
    gpio_test.go:35: blink complete
--- PASS: TestBlink (2.01s)
PASS
ok      github.com/gorai/gorai/driver/gpio    2.034s
```

---

## Filtering Tests

### By Test Name

```bash
# Exact match
go test ./... -run "^TestPublish$"

# Prefix match
go test ./... -run "TestPublish"

# Suffix match
go test ./... -run ".*Error$"

# Pattern match
go test ./... -run "TestMotor.*Power"

# Multiple patterns (OR)
go test ./... -run "TestPublish|TestSubscribe"
```

### By Subtest Name

```bash
# Specific subtest
go test ./pkg/node -run "TestNew/with_valid_config"

# Pattern in subtest
go test ./pkg/node -run "TestNew/.*error"
```

### By Package Path

```bash
# Single package
go test ./pkg/node

# Package tree
go test ./pkg/...

# Multiple specific packages
go test ./pkg/node ./pkg/pub ./pkg/sub

# Exclude pattern (using grep)
go test $(go list ./... | grep -v /vendor/)
```

### Combining Filters

```bash
# Component tests for motor, matching SetPower
go test -tags=component ./components/motor -run TestSetPower

# Integration tests for pub/sub with race detection
go test -tags=integration -race ./tests/integration -run TestPubSub

# All tests except benchmarks
go test ./... -run "^Test"
```

### Examples

```bash
# Find all tests matching "Motor"
$ go test -v ./... -run Motor -list ".*"
TestGPIOMotor_SetPower
TestGPIOMotor_GetPosition
TestGPIOMotor_Stop
TestMotorModule_Lifecycle

# Run only error case tests
$ go test -v ./pkg/... -run ".*Error.*"
=== RUN   TestNew_EmptyName_ReturnsError
--- PASS: TestNew_EmptyName_ReturnsError (0.00s)
=== RUN   TestPublish_Disconnected_ReturnsError
--- PASS: TestPublish_Disconnected_ReturnsError (0.00s)
```

---

## Watch Mode Development

Automatically re-run tests when files change.

### Using entr

```bash
# Install entr
sudo apt install entr  # Debian/Ubuntu
brew install entr      # macOS

# Watch and run tests for a package
find ./pkg/node -name "*.go" | entr -c go test ./pkg/node

# Watch entire project, run unit tests
find . -name "*.go" | entr -c go test ./...

# Watch and run specific test
find ./pkg/node -name "*.go" | entr -c go test -v ./pkg/node -run TestNew
```

### Using air

```bash
# Install air
go install github.com/air-verse/air@latest

# Create configuration
cat > .air.toml << 'EOF'
[build]
cmd = "go test -tags=component ./..."
bin = ""
delay = 500
exclude_dir = ["vendor", "testdata", ".git"]
include_ext = ["go"]
[log]
time = true
EOF

# Run
air
```

### Using watchexec

```bash
# Install watchexec
cargo install watchexec-cli  # via Rust
brew install watchexec       # macOS

# Watch and run tests
watchexec -e go "go test ./pkg/node"

# With clear screen
watchexec -c -e go "go test -v ./pkg/node"
```

### Examples

```bash
# Development workflow with entr
$ find ./pkg/node -name "*.go" | entr -c go test -v ./pkg/node

# Output on each save:
=== RUN   TestNew
--- PASS: TestNew (0.00s)
=== RUN   TestClose
--- PASS: TestClose (0.00s)
PASS
ok      github.com/gorai/gorai/pkg/node    0.023s

# [file saved, tests re-run automatically]

=== RUN   TestNew
--- PASS: TestNew (0.00s)
=== RUN   TestClose
--- PASS: TestClose (0.00s)
=== RUN   TestNewFeature
--- PASS: TestNewFeature (0.00s)
PASS
ok      github.com/gorai/gorai/pkg/node    0.025s
```

---

## Interactive Module Testing

Test modules interactively using NATS CLI during development.

### Setup

```bash
# Install NATS CLI
go install github.com/nats-io/natscli/nats@latest

# Terminal 1: Start NATS server
nats-server

# Terminal 2: Start your module
go run ./cmd/gorai module start motor --config examples/motor_config.json
```

### Publishing Messages

```bash
# Publish to a topic
nats pub sensor.imu '{"x": 1.0, "y": 0.0, "z": 9.81}'

# Publish motor command
nats pub motor.left.command '{"power": 0.5}'

# Publish with headers
nats pub sensor.camera.frame --header "timestamp:1234567890" < frame.bin
```

### Subscribing to Topics

```bash
# Subscribe to single topic
nats sub sensor.imu

# Subscribe with wildcard
nats sub "sensor.>"

# Subscribe to all motor telemetry
nats sub "motor.*.telemetry"

# Subscribe with message limit
nats sub sensor.imu --count=10
```

### Request/Reply

```bash
# Make service request
nats req motor.left.get_position ''

# Request with payload
nats req motor.left.set_power '{"power": 0.75}'

# Request with timeout
nats req --timeout=5s navigation.get_pose ''
```

### Monitoring

```bash
# Monitor all messages
nats sub ">"

# Monitor with timestamps
nats sub ">" --timestamp

# Monitor specific pattern with details
nats sub "motor.>" --dump
```

### Example Session

```bash
# Terminal 1: Start NATS
$ nats-server
[1234] Starting nats-server
[1234] Listening for client connections on 0.0.0.0:4222

# Terminal 2: Start motor module
$ go run ./cmd/gorai module start motor --config test_motor.json
INFO starting motor module
INFO connected to NATS at nats://localhost:4222
INFO motor module ready

# Terminal 3: Interact with module
$ nats sub "motor.left.telemetry.>" &
Subscribing on motor.left.telemetry.>

$ nats req motor.left.get_position ''
{"position": 0.0, "timestamp": "2024-01-15T10:30:00Z"}

$ nats pub motor.left.command '{"power": 0.5}'
Published 18 bytes to motor.left.command

# Telemetry appears:
[motor.left.telemetry.position] {"position": 0.5, "velocity": 1.2}
[motor.left.telemetry.position] {"position": 1.0, "velocity": 1.2}

$ nats req motor.left.stop ''
{"success": true}
```

---

## Coverage Reports

### Quick Coverage

```bash
# Coverage for single package
go test -cover ./pkg/node
# ok    github.com/gorai/gorai/pkg/node    0.023s    coverage: 85.2% of statements

# Coverage for all packages
go test -cover ./...
```

### Coverage Profile

```bash
# Generate coverage profile
go test -coverprofile=coverage.out ./...

# Generate with component tests
go test -tags=component -coverprofile=coverage.out ./...

# View coverage in terminal
go tool cover -func=coverage.out

# View coverage by function
go tool cover -func=coverage.out | grep -E "^total:|100.0%"
```

### HTML Report

```bash
# Generate HTML report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Open in browser
open coverage.html        # macOS
xdg-open coverage.html    # Linux
```

### Coverage by Package

```bash
# Detailed coverage output
$ go test -cover ./pkg/...
ok      github.com/gorai/gorai/pkg/node      0.023s    coverage: 85.2%
ok      github.com/gorai/gorai/pkg/pub       0.018s    coverage: 92.1%
ok      github.com/gorai/gorai/pkg/sub       0.019s    coverage: 88.7%
ok      github.com/gorai/gorai/pkg/services   0.021s    coverage: 79.3%

# Function-level coverage
$ go tool cover -func=coverage.out | head -20
github.com/gorai/gorai/pkg/node/node.go:25:     New             100.0%
github.com/gorai/gorai/pkg/node/node.go:45:     Close           85.7%
github.com/gorai/gorai/pkg/node/node.go:62:     Connect         100.0%
github.com/gorai/gorai/pkg/node/node.go:89:     Publish         90.0%
...
total:                                          (statements)    86.4%
```

### Coverage Thresholds

```bash
# Check if coverage meets threshold (in CI)
COVERAGE=$(go test -cover ./... | grep -oP '\d+\.\d+(?=%)' | awk '{s+=$1; c++} END {print s/c}')
if (( $(echo "$COVERAGE < 75" | bc -l) )); then
    echo "Coverage $COVERAGE% is below 75% threshold"
    exit 1
fi
```

---

## Benchmarks

### Running Benchmarks

```bash
# Run all benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkPublish ./pkg/pub

# Run with memory allocation stats
go test -bench=. -benchmem ./...

# Run for specific duration
go test -bench=. -benchtime=5s ./...

# Run specific number of iterations
go test -bench=. -benchtime=1000x ./...
```

### Benchmark Examples

```bash
# Run pub/sub benchmarks
$ go test -bench=. -benchmem ./pkg/pub
goos: linux
goarch: amd64
pkg: github.com/gorai/gorai/pkg/pub
BenchmarkPublish-8           1000000      1052 ns/op     256 B/op      4 allocs/op
BenchmarkPublishParallel-8   5000000       234 ns/op     256 B/op      4 allocs/op
PASS
ok      github.com/gorai/gorai/pkg/pub    3.456s

# Compare benchmarks
$ go test -bench=. -benchmem ./pkg/pub > old.txt
# [make changes]
$ go test -bench=. -benchmem ./pkg/pub > new.txt
$ benchstat old.txt new.txt
name              old time/op    new time/op    delta
Publish-8           1.05µs ± 2%    0.89µs ± 1%   -15.24%
PublishParallel-8    234ns ± 3%     198ns ± 2%   -15.38%
```

### CPU and Memory Profiling

```bash
# Generate CPU profile
go test -bench=BenchmarkPublish -cpuprofile=cpu.prof ./pkg/pub

# Generate memory profile
go test -bench=BenchmarkPublish -memprofile=mem.prof ./pkg/pub

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof

# Web interface
go tool pprof -http=:8080 cpu.prof
```

---

## CI/CD Commands

### Pre-commit (Fast)

```bash
# Format check
gofmt -l . | grep -q . && echo "Run 'go fmt ./...'" && exit 1

# Vet
go vet ./...

# Unit tests
go test -short ./...
```

### Pre-push (Thorough)

```bash
# Unit + component tests with race detection
go test -tags=component -race ./...

# Integration tests
go test -tags=integration -race ./tests/integration/...
```

### CI Pipeline (Complete)

```bash
# Full test suite
go test -tags="component,integration,module,system" -race -coverprofile=coverage.out ./...

# Generate reports
go tool cover -html=coverage.out -o coverage.html
go tool cover -func=coverage.out > coverage.txt

# Check coverage threshold
TOTAL=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')
if (( $(echo "$TOTAL < 75" | bc -l) )); then
    exit 1
fi
```

### Nightly (Extended)

```bash
# All tests including slow ones
go test -tags="component,integration,module,system" -timeout=30m ./...

# Benchmarks with comparison
go test -tags=benchmark -bench=. -benchmem ./tests/benchmark/... > benchmark.txt

# Race detection on everything
go test -tags="component,integration,module,system" -race ./...
```

---

## Troubleshooting

### Common Issues

#### Tests Timeout

```bash
# Increase timeout
go test -timeout=5m ./...

# Check for hanging tests
go test -v -timeout=30s ./pkg/node 2>&1 | tee test.log
```

#### NATS Connection Failures

```bash
# Check if NATS is running (for integration tests)
nats-server --version

# Use embedded NATS (tests should do this automatically)
go test -tags=integration -v ./tests/integration -run TestPubSub 2>&1 | grep -i nats
```

#### Permission Denied (Hardware Tests)

```bash
# GPIO access on Raspberry Pi
sudo usermod -a -G gpio $USER
# Log out and back in

# Serial port access
sudo usermod -a -G dialout $USER
# Log out and back in

# Run with sudo if needed
sudo go test -tags=hardware ./driver/gpio/...
```

#### Race Conditions

```bash
# Run with race detector
go test -race ./...

# If race detected, run specific test with verbose
go test -race -v ./pkg/node -run TestConcurrent

# Increase GOMAXPROCS to increase race detection likelihood
GOMAXPROCS=8 go test -race ./...
```

#### Flaky Tests

```bash
# Run test multiple times
for i in {1..10}; do go test ./pkg/node -run TestFlaky || break; done

# Run with stress tool
go install golang.org/x/tools/cmd/stress@latest
stress -p 4 go test ./pkg/node -run TestFlaky
```

### Debugging Tests

```bash
# Verbose output
go test -v ./pkg/node

# Print to stdout during test (normally suppressed)
go test -v ./pkg/node 2>&1

# Use delve debugger
dlv test ./pkg/node -- -test.run TestSpecific

# Generate test binary for debugging
go test -c ./pkg/node
dlv exec ./node.test -- -test.run TestSpecific
```

### Environment Variables

```bash
# Skip slow tests
GORAI_SKIP_SLOW=1 go test ./...

# Use specific NATS server
GORAI_NATS_URL=nats://192.168.1.100:4222 go test -tags=integration ./...

# Enable debug logging in tests
GORAI_DEBUG=1 go test -v ./...

# Hardware test configuration
GORAI_GPIO_PIN=18 go test -tags=hardware ./driver/gpio/...
```

---

## Summary

| Task | Command |
|------|---------|
| Quick unit tests | `go test ./...` |
| Verbose single package | `go test -v ./pkg/node` |
| Run specific test | `go test -run TestName ./...` |
| Component tests | `go test -tags=component ./components/...` |
| Integration tests | `go test -tags=integration ./tests/integration` |
| Module tests | `go test -tags=module ./tests/module` |
| System tests | `go test -tags=system ./tests/system` |
| Hardware tests | `go test -tags=hardware ./driver/...` |
| With coverage | `go test -cover ./...` |
| Coverage report | `go test -coverprofile=c.out ./... && go tool cover -html=c.out` |
| With race detection | `go test -race ./...` |
| Benchmarks | `go test -bench=. -benchmem ./...` |
| Watch mode | `find . -name "*.go" \| entr -c go test ./pkg/node` |
| Full regression | `go test -tags="component,integration,module,system" ./...` |
