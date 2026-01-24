# Gorai Testing Approach Specification

**Version 0.1.0**

This specification defines how to write, structure, and configure tests for the Gorai robotics framework.

For instructions on running tests, see [How to Run Tests](howto-run-tests.md).

---

## Table of Contents

1. [Testing Philosophy](#testing-philosophy)
2. [Testing Pyramid](#testing-pyramid)
3. [Test Categories](#test-categories)
4. [Writing Unit Tests](#writing-unit-tests)
5. [Writing Component Tests](#writing-component-tests)
6. [Writing Integration Tests](#writing-integration-tests)
7. [Writing Module Tests](#writing-module-tests)
8. [Writing System Tests](#writing-system-tests)
9. [Writing Hardware Tests](#writing-hardware-tests)
10. [Fake Implementations](#fake-implementations)
11. [Test Infrastructure](#test-infrastructure)
12. [TinyGo Testing](#tinygo-testing)
13. [CI Configuration](#ci-configuration)
14. [Coverage Requirements](#coverage-requirements)

---

## Testing Philosophy

### Guiding Principles

1. **Fast feedback**: Unit tests run in milliseconds; developers get immediate feedback
2. **Isolation**: Each test is independent; no shared state between tests
3. **Reproducibility**: Tests produce the same results regardless of execution order
4. **NATS-native**: Integration tests use real (embedded) NATS, not mocks
5. **Hardware abstraction**: Use fakes for hardware; real hardware tests are opt-in
6. **Table-driven**: Prefer table-driven tests for comprehensive coverage

### What to Test

| Always Test | Sometimes Test | Rarely Test |
|-------------|----------------|-------------|
| Public API | Internal helpers | Generated code |
| Error paths | Edge cases | Trivial getters/setters |
| Concurrency | Performance | Third-party libraries |
| State transitions | Resource cleanup | |

### Test Naming

Tests **SHALL** follow this naming pattern:

```
TestSubject_Scenario_ExpectedBehavior
```

Examples:
```go
func TestNew_WithValidConfig_ReturnsNode(t *testing.T) {}
func TestPublish_WhenDisconnected_ReturnsError(t *testing.T) {}
func TestMotor_SetPower_ClampsToRange(t *testing.T) {}
```

---

## Testing Pyramid

```mermaid
flowchart TB
    subgraph pyramid["Testing Pyramid"]
        direction TB
        HW["Hardware Tests<br/>1% - Real hardware, opt-in"]
        SYS["System Tests<br/>4% - Full robot config"]
        MOD["Module Tests<br/>5% - Module lifecycle"]
        INT["Integration Tests<br/>10% - Multi-component"]
        COMP["Component Tests<br/>20% - Single component"]
        UNIT["Unit Tests<br/>60% - Functions/types"]
    end

    UNIT --> COMP --> INT --> MOD --> SYS --> HW
```

### Layer Characteristics

| Layer | Build Tag | Speed | NATS | Hardware | Typical Count |
|-------|-----------|-------|------|----------|---------------|
| Unit | (none) | <10ms | No | No | Hundreds |
| Component | `component` | <100ms | Embedded | Fake | Dozens |
| Integration | `integration` | <1s | Embedded | Fake | Tens |
| Module | `module` | <5s | Embedded | Fake | Few |
| System | `system` | <30s | Embedded | Simulated | Few |
| Hardware | `hardware` | Variable | Real | Real | Few |

---

## Test Categories

### Build Tags

Tests **SHALL** use build tags to categorize:

```go
//go:build component

package motor_test
```

| Tag | Purpose | Location |
|-----|---------|----------|
| (none) | Unit tests | Same package as code |
| `component` | Component tests | `components/*/` |
| `integration` | Integration tests | `tests/integration/` |
| `module` | Module lifecycle tests | `tests/module/` |
| `system` | Full robot tests | `tests/system/` |
| `hardware` | Real hardware tests | `driver/*/` or `tests/hardware/` |
| `benchmark` | Performance tests | `tests/benchmark/` |

### Directory Structure

```
github.com/gorai/gorai/
├── pkg/
│   └── node/
│       ├── node.go
│       └── node_test.go           # Unit tests (no tag)
│
├── components/
│   └── motor/
│       ├── motor.go
│       ├── motor_test.go          # Unit tests
│       ├── motor_component_test.go # Component tests (//go:build component)
│       └── fake/
│           └── fake.go            # Fake implementation
│
├── tests/
│   ├── integration/
│   │   └── pubsub_test.go         # //go:build integration
│   ├── module/
│   │   └── motor_test.go          # //go:build module
│   ├── system/
│   │   └── robot_test.go          # //go:build system
│   └── benchmark/
│       └── nats_bench_test.go     # //go:build benchmark
│
├── internal/
│   └── testutil/                  # Shared test utilities
│       ├── testutil.go
│       ├── nats.go
│       └── assertions.go
│
└── testdata/                      # Test fixtures
    ├── configs/
    └── golden/
```

---

## Writing Unit Tests

Unit tests verify individual functions and types in isolation.

### Table-Driven Pattern

All unit tests **SHOULD** use table-driven patterns:

```go
func TestParseConfig(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    *Config
        wantErr bool
    }{
        {
            name:  "valid JSON",
            input: `{"name": "robot1"}`,
            want:  &Config{Name: "robot1"},
        },
        {
            name:    "invalid JSON",
            input:   `{invalid}`,
            wantErr: true,
        },
        {
            name:    "empty input",
            input:   "",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseConfig([]byte(tt.input))

            if (err != nil) != tt.wantErr {
                t.Errorf("ParseConfig() error = %v, wantErr %v", err, tt.wantErr)
                return
            }

            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("ParseConfig() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Parallel Tests

Tests that don't share state **SHOULD** run in parallel:

```go
func TestNode_Connect(t *testing.T) {
    t.Parallel() // Enable parallel execution

    tests := []struct {
        name string
        url  string
    }{
        {"valid URL", "nats://localhost:4222"},
        {"with credentials", "nats://user:pass@localhost:4222"},
    }

    for _, tt := range tests {
        tt := tt // Capture range variable
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel() // Each subtest also parallel
            // ... test logic
        })
    }
}
```

### Test Helpers

Use `t.Helper()` for helper functions:

```go
func assertNoError(t *testing.T, err error) {
    t.Helper() // Marks this as helper; errors report caller's line
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func assertEqual[T comparable](t *testing.T, got, want T) {
    t.Helper()
    if got != want {
        t.Errorf("got %v, want %v", got, want)
    }
}
```

### Cleanup

Use `t.Cleanup()` for resource cleanup:

```go
func TestWithTempFile(t *testing.T) {
    f, err := os.CreateTemp("", "test-*")
    assertNoError(t, err)

    t.Cleanup(func() {
        f.Close()
        os.Remove(f.Name())
    })

    // Test logic using f
}
```

---

## Writing Component Tests

Component tests verify a single Gorai component with dependencies faked.

### Structure

```go
//go:build component

package motor_test

import (
    "context"
    "testing"

    "github.com/gorai/gorai/components/motor"
    "github.com/gorai/gorai/components/motor/fake"
)

func TestGPIOMotor_SetPower(t *testing.T) {
    ctx := context.Background()

    // Create fake dependencies
    fakeGPIO := fake.NewGPIO()

    // Create component under test
    m, err := motor.NewGPIO(motor.GPIOConfig{
        Pin:       18,
        Frequency: 1000,
        GPIO:      fakeGPIO,
    })
    if err != nil {
        t.Fatalf("failed to create motor: %v", err)
    }
    defer m.Close(ctx)

    // Test positive power
    t.Run("positive power", func(t *testing.T) {
        err := m.SetPower(ctx, 0.75)
        if err != nil {
            t.Errorf("SetPower(0.75) error: %v", err)
        }

        if fakeGPIO.DutyCycle() != 0.75 {
            t.Errorf("duty cycle = %v, want 0.75", fakeGPIO.DutyCycle())
        }
    })

    // Test clamping
    t.Run("clamps to range", func(t *testing.T) {
        err := m.SetPower(ctx, 1.5) // Over max
        if err != nil {
            t.Errorf("SetPower(1.5) error: %v", err)
        }

        if fakeGPIO.DutyCycle() != 1.0 {
            t.Errorf("duty cycle = %v, want 1.0 (clamped)", fakeGPIO.DutyCycle())
        }
    })
}
```

### Testing State Transitions

```go
func TestMotor_StateTransitions(t *testing.T) {
    ctx := context.Background()
    m := createTestMotor(t)

    // Initial state
    if m.IsRunning() {
        t.Error("motor should not be running initially")
    }

    // Start
    m.SetPower(ctx, 0.5)
    if !m.IsRunning() {
        t.Error("motor should be running after SetPower")
    }

    // Stop
    m.Stop(ctx)
    if m.IsRunning() {
        t.Error("motor should not be running after Stop")
    }
}
```

---

## Writing Integration Tests

Integration tests verify multiple components through NATS.

### Embedded NATS Server

Integration tests **SHALL** use the embedded NATS server from testutil:

```go
//go:build integration

package integration_test

import (
    "context"
    "testing"
    "time"

    "github.com/gorai/gorai/internal/testutil"
    "github.com/gorai/gorai/pkg/node"
    "github.com/gorai/gorai/pkg/pub"
    "github.com/gorai/gorai/pkg/sub"
)

func TestPubSub_RoundTrip(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // Start embedded NATS (automatically cleaned up)
    tn := testutil.StartNATS(t)

    // Create publisher
    pubNode, err := node.New("publisher", node.WithNATS(tn.URL))
    testutil.RequireNoError(t, err)
    defer pubNode.Close()

    // Create subscriber
    subNode, err := node.New("subscriber", node.WithNATS(tn.URL))
    testutil.RequireNoError(t, err)
    defer subNode.Close()

    // Set up subscription
    received := make(chan string, 1)
    _, err = sub.New[TestMessage](subNode, "test.topic", func(msg *TestMessage) {
        received <- msg.Data
    })
    testutil.RequireNoError(t, err)

    // Wait for subscription to be ready
    time.Sleep(50 * time.Millisecond)

    // Publish
    publisher := pub.New[TestMessage](pubNode, "test.topic")
    err = publisher.Publish(ctx, &TestMessage{Data: "hello"})
    testutil.RequireNoError(t, err)

    // Verify
    select {
    case data := <-received:
        if data != "hello" {
            t.Errorf("received %q, want %q", data, "hello")
        }
    case <-ctx.Done():
        t.Fatal("timeout waiting for message")
    }
}
```

### Testing Request/Reply

```go
func TestService_RequestReply(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    tn := testutil.StartNATS(t)

    // Service node
    svcNode, err := node.New("service", node.WithNATS(tn.URL))
    testutil.RequireNoError(t, err)
    defer svcNode.Close()

    // Register handler
    svcNode.HandleService("math.add", func(req []byte) ([]byte, error) {
        var input struct{ A, B int }
        json.Unmarshal(req, &input)
        result := struct{ Sum int }{Sum: input.A + input.B}
        return json.Marshal(result)
    })

    // Client node
    clientNode, err := node.New("client", node.WithNATS(tn.URL))
    testutil.RequireNoError(t, err)
    defer clientNode.Close()

    // Make request
    resp, err := clientNode.Request(ctx, "math.add", []byte(`{"A": 2, "B": 3}`))
    testutil.RequireNoError(t, err)

    var result struct{ Sum int }
    json.Unmarshal(resp, &result)

    if result.Sum != 5 {
        t.Errorf("sum = %d, want 5", result.Sum)
    }
}
```

---

## Writing Module Tests

Module tests verify a complete module's lifecycle.

### Module Test Structure

```go
//go:build module

package module_test

import (
    "context"
    "testing"
    "time"

    "github.com/gorai/gorai/internal/testutil"
    "github.com/gorai/gorai/components/motor"
    fakemotor "github.com/gorai/gorai/components/motor/fake"
)

func TestMotorModule_Lifecycle(t *testing.T) {
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    // Infrastructure
    tn := testutil.StartNATS(t)
    fakeMotor := fakemotor.New()

    // Create and start module
    module, err := motor.NewModule(motor.ModuleConfig{
        NATSURL: tn.URL,
        Name:    "left_motor",
        Motor:   fakeMotor,
    })
    testutil.RequireNoError(t, err)

    // Run module in background
    moduleCtx, moduleCancel := context.WithCancel(ctx)
    t.Cleanup(moduleCancel)

    go module.Run(moduleCtx)

    // Wait for ready
    testutil.AssertEventually(t, func() bool {
        return module.Ready()
    }, 5*time.Second, "module not ready")

    // Create test client
    client, err := testutil.NewNATSClient(t, tn.URL)
    testutil.RequireNoError(t, err)

    // Test: SetPower via NATS
    t.Run("SetPower", func(t *testing.T) {
        _, err := client.Request(ctx, "motor.left_motor.set_power",
            []byte(`{"power": 0.75}`))
        testutil.RequireNoError(t, err)

        testutil.AssertEventually(t, func() bool {
            return fakeMotor.Power() == 0.75
        }, time.Second, "power not set")
    })

    // Test: GetPosition via NATS
    t.Run("GetPosition", func(t *testing.T) {
        fakeMotor.SetPosition(42.0)

        resp, err := client.Request(ctx, "motor.left_motor.get_position", nil)
        testutil.RequireNoError(t, err)

        var pos struct{ Position float64 }
        json.Unmarshal(resp, &pos)

        if pos.Position != 42.0 {
            t.Errorf("position = %v, want 42.0", pos.Position)
        }
    })

    // Test: Telemetry published
    t.Run("Telemetry", func(t *testing.T) {
        telemetry := make(chan []byte, 10)
        sub, _ := client.Subscribe("motor.left_motor.telemetry.>", func(msg []byte) {
            telemetry <- msg
        })
        defer sub.Unsubscribe()

        select {
        case <-telemetry:
            // Received telemetry
        case <-time.After(2 * time.Second):
            t.Error("no telemetry received")
        }
    })
}
```

### Testing Reconfiguration

```go
func TestModule_HotReconfigure(t *testing.T) {
    // ... setup ...

    // Initial config
    module.Configure(initialConfig)
    testutil.AssertEventually(t, func() bool {
        return module.Ready()
    }, 5*time.Second, "not ready after initial config")

    // Reconfigure
    module.Configure(updatedConfig)
    testutil.AssertEventually(t, func() bool {
        // Verify new configuration applied
        return module.CurrentConfig().Equals(updatedConfig)
    }, 5*time.Second, "not ready after reconfig")
}
```

---

## Writing System Tests

System tests verify complete robot configurations.

### Robot Configuration Test

```go
//go:build system

package system_test

import (
    "context"
    "testing"
    "time"

    "github.com/gorai/gorai/internal/testutil"
    "github.com/gorai/gorai/pkg/robot"
)

func TestDifferentialRobot_Navigation(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping system test in short mode")
    }

    ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
    defer cancel()

    tn := testutil.StartNATS(t)

    // Load robot with fake hardware
    r, err := robot.Load("testdata/differential.json", robot.Options{
        NATSURL:  tn.URL,
        UseFakes: true,
    })
    testutil.RequireNoError(t, err)

    // Start robot
    robotCtx, robotCancel := context.WithCancel(ctx)
    t.Cleanup(robotCancel)
    go r.Run(robotCtx)

    testutil.AssertEventually(t, func() bool {
        return r.Ready()
    }, 10*time.Second, "robot not ready")

    client := r.Client()

    // Test navigation sequence
    t.Run("MoveToWaypoint", func(t *testing.T) {
        err := client.Navigation().MoveTo(ctx, 1.0, 0.0, 0.0)
        testutil.RequireNoError(t, err)

        // Wait for completion or timeout
        testutil.AssertEventually(t, func() bool {
            pose, _ := client.Base().GetPose(ctx)
            return pose.X > 0.9 // Close to target
        }, 10*time.Second, "did not reach waypoint")
    })
}
```

---

## Writing Hardware Tests

Hardware tests require physical hardware and run only when explicitly enabled.

### Hardware Test Structure

```go
//go:build hardware && raspberry_pi

package hardware_test

import (
    "testing"
    "time"

    "github.com/gorai/gorai/driver/gpio"
)

func TestGPIO_BlinkLED(t *testing.T) {
    pin := gpio.Pin(18) // Configurable via env or flag

    p, err := gpio.Open(pin)
    if err != nil {
        t.Fatalf("failed to open GPIO %d: %v", pin, err)
    }
    defer p.Close()

    err = p.SetOutput()
    if err != nil {
        t.Fatalf("failed to set output: %v", err)
    }

    // Blink test
    for i := 0; i < 5; i++ {
        p.High()
        time.Sleep(100 * time.Millisecond)
        p.Low()
        time.Sleep(100 * time.Millisecond)
    }

    t.Log("LED blink test complete - verify visually")
}
```

### Platform-Specific Tags

```go
//go:build hardware && jetson

package hardware_test

// Tests for NVIDIA Jetson platform
```

```go
//go:build hardware && rk3588

package hardware_test

// Tests for RK3588 boards (Orange Pi 5, Rock 5B)
```

### Skipping When Hardware Unavailable

```go
func TestI2C_ReadSensor(t *testing.T) {
    bus, err := i2c.Open(1)
    if err != nil {
        t.Skipf("I2C bus not available: %v", err)
    }
    defer bus.Close()

    // ... test with hardware
}
```

---

## Fake Implementations

Every component interface **SHALL** have a fake implementation.

### Fake Structure

```go
// components/motor/fake/fake.go
package fake

import (
    "context"
    "sync"

    "github.com/gorai/gorai/components/motor"
)

// Motor is a fake motor for testing
type Motor struct {
    mu       sync.Mutex
    power    float64
    position float64
    stopped  bool

    // Hooks for custom behavior
    SetPowerFunc    func(float64) error
    GetPositionFunc func() (float64, error)
    StopFunc        func() error
}

var _ motor.Motor = (*Motor)(nil)

func New() *Motor {
    return &Motor{}
}

func (m *Motor) Name() string {
    return "fake"
}

func (m *Motor) SetPower(ctx context.Context, power float64) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.SetPowerFunc != nil {
        return m.SetPowerFunc(power)
    }

    m.power = power
    m.stopped = false
    return nil
}

func (m *Motor) GetPosition(ctx context.Context) (float64, error) {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.GetPositionFunc != nil {
        return m.GetPositionFunc()
    }

    return m.position, nil
}

func (m *Motor) Stop(ctx context.Context) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    if m.StopFunc != nil {
        return m.StopFunc()
    }

    m.power = 0
    m.stopped = true
    return nil
}

func (m *Motor) Close(ctx context.Context) error {
    return m.Stop(ctx)
}

func (m *Motor) Reconfigure(ctx context.Context, cfg motor.Config) error {
    return nil
}

// Test inspection methods

func (m *Motor) Power() float64 {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.power
}

func (m *Motor) SetPosition(pos float64) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.position = pos
}

func (m *Motor) IsStopped() bool {
    m.mu.Lock()
    defer m.mu.Unlock()
    return m.stopped
}
```

### Using Hooks for Error Injection

```go
func TestMotor_SetPowerError(t *testing.T) {
    m := fake.New()
    m.SetPowerFunc = func(power float64) error {
        return errors.New("hardware fault")
    }

    err := m.SetPower(context.Background(), 0.5)
    if err == nil {
        t.Error("expected error, got nil")
    }
}
```

---

## Test Infrastructure

### Shared Utilities

```go
// internal/testutil/testutil.go
package testutil

import (
    "testing"
    "time"
)

// RequireNoError fails immediately if err is not nil
func RequireNoError(t *testing.T, err error) {
    t.Helper()
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

// AssertEventually retries condition until timeout
func AssertEventually(t *testing.T, condition func() bool, timeout time.Duration, msg string) {
    t.Helper()
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if condition() {
            return
        }
        time.Sleep(10 * time.Millisecond)
    }
    t.Fatalf("condition not met within %v: %s", timeout, msg)
}

// AssertNever verifies condition never becomes true
func AssertNever(t *testing.T, condition func() bool, duration time.Duration, msg string) {
    t.Helper()
    deadline := time.Now().Add(duration)
    for time.Now().Before(deadline) {
        if condition() {
            t.Fatalf("condition became true: %s", msg)
        }
        time.Sleep(10 * time.Millisecond)
    }
}
```

### NATS Test Server

```go
// internal/testutil/nats.go
package testutil

import (
    "fmt"
    "net"
    "testing"
    "time"

    "github.com/nats-io/nats-server/v2/server"
    "github.com/nats-io/nats.go"
)

type TestNATS struct {
    Server *server.Server
    URL    string
}

func StartNATS(t *testing.T) *TestNATS {
    t.Helper()

    opts := &server.Options{
        Host:   "127.0.0.1",
        Port:   -1, // Random port
        NoLog:  true,
        NoSigs: true,
    }

    ns, err := server.NewServer(opts)
    if err != nil {
        t.Fatalf("failed to create NATS server: %v", err)
    }

    go ns.Start()

    if !ns.ReadyForConnections(5 * time.Second) {
        t.Fatal("NATS server not ready")
    }

    addr := ns.Addr().(*net.TCPAddr)
    tn := &TestNATS{
        Server: ns,
        URL:    fmt.Sprintf("nats://127.0.0.1:%d", addr.Port),
    }

    t.Cleanup(func() {
        ns.Shutdown()
        ns.WaitForShutdown()
    })

    return tn
}

func (tn *TestNATS) Connect(t *testing.T) *nats.Conn {
    t.Helper()

    nc, err := nats.Connect(tn.URL)
    if err != nil {
        t.Fatalf("failed to connect to NATS: %v", err)
    }

    t.Cleanup(func() {
        nc.Close()
    })

    return nc
}
```

### Test Fixtures

```go
// internal/testutil/fixtures.go
package testutil

import (
    "os"
    "path/filepath"
    "runtime"
    "testing"
)

func FixturePath(t *testing.T, name string) string {
    t.Helper()

    _, filename, _, ok := runtime.Caller(0)
    if !ok {
        t.Fatal("failed to get caller info")
    }

    root := filepath.Dir(filepath.Dir(filepath.Dir(filename)))
    return filepath.Join(root, "testdata", name)
}

func LoadFixture(t *testing.T, name string) []byte {
    t.Helper()

    data, err := os.ReadFile(FixturePath(t, name))
    if err != nil {
        t.Fatalf("failed to load fixture %s: %v", name, err)
    }
    return data
}
```

### Golden Files

```go
// internal/testutil/golden.go
package testutil

import (
    "bytes"
    "flag"
    "os"
    "path/filepath"
    "testing"
)

var update = flag.Bool("update", false, "update golden files")

func AssertGolden(t *testing.T, name string, actual []byte) {
    t.Helper()

    golden := filepath.Join("testdata", "golden", name+".golden")

    if *update {
        os.MkdirAll(filepath.Dir(golden), 0755)
        os.WriteFile(golden, actual, 0644)
        return
    }

    expected, err := os.ReadFile(golden)
    if err != nil {
        t.Fatalf("failed to read golden file: %v", err)
    }

    if !bytes.Equal(expected, actual) {
        t.Errorf("output mismatch\nwant:\n%s\ngot:\n%s", expected, actual)
    }
}
```

---

## TinyGo Testing

TinyGo code has limited testing capabilities due to runtime constraints.

### What to Test Where

| Test Type | Where to Run | How |
|-----------|--------------|-----|
| Logic/algorithms | Standard Go | `go test` |
| Protocol encoding | Standard Go | `go test` |
| Build verification | CI | `tinygo build` |
| Hardware integration | Target device | Flash and verify |

### TinyGo Test Example

```go
// serial/gsp/gsp_test.go - runs with standard go test
package gsp

import (
    "bytes"
    "testing"
)

func TestFrame_Encode(t *testing.T) {
    frame := Frame{
        Type:    TypePublish,
        Topic:   "sensor.imu",
        Payload: []byte(`{"x":1.0}`),
    }

    encoded := frame.Encode()
    decoded, err := DecodeFrame(encoded)
    if err != nil {
        t.Fatalf("decode failed: %v", err)
    }

    if decoded.Topic != frame.Topic {
        t.Errorf("topic = %s, want %s", decoded.Topic, frame.Topic)
    }

    if !bytes.Equal(decoded.Payload, frame.Payload) {
        t.Errorf("payload mismatch")
    }
}

func TestCRC16(t *testing.T) {
    tests := []struct {
        data []byte
        want uint16
    }{
        {[]byte{0x01, 0x02, 0x03}, 0x6131},
        {[]byte("hello"), 0x34D2},
    }

    for _, tt := range tests {
        got := CRC16(tt.data)
        if got != tt.want {
            t.Errorf("CRC16(%v) = %04X, want %04X", tt.data, got, tt.want)
        }
    }
}
```

### Build Verification in CI

```yaml
# .github/workflows/tinygo.yml
- name: TinyGo build check
  run: |
    tinygo build -target=esp32 -o /dev/null ./...
    tinygo build -target=pico -o /dev/null ./...
    tinygo build -target=arduino-mega2560 -o /dev/null ./...
```

---

## CI Configuration

### GitHub Actions Workflow

```yaml
# .github/workflows/test.yml
name: Tests

on: [push, pull_request]

jobs:
  unit:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go test -race -coverprofile=coverage.out ./...
      - uses: codecov/codecov-action@v3

  component:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go test -tags=component -race ./components/...

  integration:
    runs-on: ubuntu-latest
    needs: [unit, component]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go test -tags=integration -race ./tests/integration/...

  module:
    runs-on: ubuntu-latest
    needs: [integration]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go test -tags=module -timeout=5m ./tests/module/...

  system:
    runs-on: ubuntu-latest
    needs: [module]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.21'
      - run: go test -tags=system -timeout=10m ./tests/system/...
```

### Pre-commit Hook

```bash
#!/bin/bash
# .git/hooks/pre-commit
set -e
gofmt -l . | grep -q . && echo "Run 'go fmt ./...'" && exit 1
go vet ./...
go test -short ./...
```

---

## Coverage Requirements

### Minimum Coverage

| Package | Line Coverage |
|---------|---------------|
| `pkg/*` | 80% |
| `components/*` | 75% |
| `services/*` | 75% |
| `driver/*` | 60% |
| `accel/*` | 70% |

### Enforcing Coverage

```go
// internal/testutil/coverage.go
package testutil

import (
    "os/exec"
    "strconv"
    "strings"
    "testing"
)

func RequireCoverage(t *testing.T, pkg string, threshold float64) {
    t.Helper()

    out, err := exec.Command("go", "test", "-cover", pkg).Output()
    if err != nil {
        t.Fatalf("coverage check failed: %v", err)
    }

    // Parse "coverage: 85.2% of statements"
    output := string(out)
    idx := strings.Index(output, "coverage: ")
    if idx == -1 {
        return // No coverage info
    }

    pctStr := output[idx+10:]
    pctStr = pctStr[:strings.Index(pctStr, "%")]
    pct, _ := strconv.ParseFloat(pctStr, 64)

    if pct < threshold {
        t.Errorf("coverage %.1f%% below threshold %.1f%%", pct, threshold)
    }
}
```

---

## Summary

| Aspect | Specification |
|--------|---------------|
| Unit test pattern | Table-driven with `t.Run()` |
| Test naming | `TestSubject_Scenario_ExpectedBehavior` |
| Build tags | `component`, `integration`, `module`, `system`, `hardware` |
| NATS testing | Use `testutil.StartNATS(t)` |
| Hardware abstraction | Fake implementations in `components/*/fake/` |
| Cleanup | Use `t.Cleanup()` |
| Parallelism | Use `t.Parallel()` where safe |
| Coverage target | 75% overall |
| TinyGo | Standard Go tests for logic, build verification in CI |
