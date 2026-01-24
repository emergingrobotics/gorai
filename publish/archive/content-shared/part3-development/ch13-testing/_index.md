# Chapter 11: Testing Strategies

Robots are safety-critical systems. Bugs can cause physical damage. Testing is not optional—it's essential.

## 11.1 The Testing Pyramid

GoRAI follows a testing pyramid with more unit tests at the base:

```
        /\
       /  \  Hardware Tests (1%)
      /    \  System Tests (4%)
     /      \  Module Tests (5%)
    /        \  Integration Tests (10%)
   /          \  Component Tests (20%)
  /            \  Unit Tests (60%)
 /______________\
```

**Philosophy**: Catch bugs at the lowest level possible. Unit tests are fast, reliable, and pinpoint problems. Higher-level tests catch integration issues but are slower and harder to debug.

## 11.2 Test Categories and Build Tags

GoRAI uses build tags to organize tests:

| Tag | Purpose | Speed | NATS | Hardware |
|-----|---------|-------|------|----------|
| (none) | Unit tests | <1s | No | No |
| `component` | Single component | 1-5s | Embedded | No |
| `integration` | Multi-component | 5-30s | Embedded | No |
| `module` | Full module lifecycle | 30s+ | Embedded | No |
| `system` | Complete robot config | Minutes | Real | Optional |
| `hardware` | Real hardware | Variable | Real | Yes |

Run specific categories:
```bash
go test ./...                                    # Unit only
go test -tags=component ./...                   # Component
go test -tags=integration ./...                 # Integration
go test -tags="component integration" ./...     # Both
```

## 11.3 Unit Testing Patterns

### Table-Driven Tests

The Go idiom for comprehensive testing:

```go
func TestCelsiusToFahrenheit(t *testing.T) {
    tests := []struct {
        name       string
        celsius    float64
        fahrenheit float64
    }{
        {"freezing", 0, 32},
        {"boiling", 100, 212},
        {"body temp", 37, 98.6},
        {"negative", -40, -40},  // Same in both!
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := CelsiusToFahrenheit(tt.celsius)
            if math.Abs(got-tt.fahrenheit) > 0.1 {
                t.Errorf("CelsiusToFahrenheit(%v) = %v, want %v",
                    tt.celsius, got, tt.fahrenheit)
            }
        })
    }
}
```

### Test Helpers

Mark helper functions with `t.Helper()`:

```go
func assertNoError(t *testing.T, err error) {
    t.Helper()  // Points to caller in failure output
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func createTestSensor(t *testing.T) *TemperatureSensor {
    t.Helper()
    fake := fake.New()
    fake.SetTemperature(25.0)
    sensor, err := sensor.New(nil, fake, sensor.DefaultConfig())
    assertNoError(t, err)
    t.Cleanup(func() {
        sensor.Close(context.Background())
    })
    return sensor
}
```

### Parallel Tests

Speed up test suites:

```go
func TestMotor_SetPower(t *testing.T) {
    t.Parallel()  // Run concurrently with other parallel tests

    motor := fake.NewMotor()
    // Test...
}

func TestMotor_Stop(t *testing.T) {
    t.Parallel()  // Also runs in parallel

    motor := fake.NewMotor()
    // Test...
}
```

**Avoid shared state** in parallel tests.

## 11.4 Fake Implementations

Every component needs a fake for testing.

### Hooks for Custom Behavior

```go
type FakeMotor struct {
    mu sync.RWMutex

    // State
    power   float64
    moving  bool

    // Hooks for custom test behavior
    OnSetPower func(power float64) error
    OnStop     func() error
}

func (m *FakeMotor) SetPower(ctx context.Context, power float64) error {
    if m.OnSetPower != nil {
        return m.OnSetPower(power)
    }

    m.mu.Lock()
    m.power = power
    m.moving = power != 0
    m.mu.Unlock()
    return nil
}
```

Usage in tests:
```go
func TestBehavior_StopsOnOverheat(t *testing.T) {
    motor := fake.NewMotor()

    // Track if stop was called
    stopCalled := false
    motor.OnStop = func() error {
        stopCalled = true
        return nil
    }

    // Run behavior with high temperature
    behavior := NewThermalSafety(motor, tempSensor)
    tempSensor.SetTemperature(95.0)  // Overheat!
    behavior.Check()

    assert.True(t, stopCalled, "should stop motor on overheat")
}
```

### Error Injection

```go
func TestSensor_HandlesReadError(t *testing.T) {
    fake := fake.NewReader()
    fake.SetError("I2C timeout")

    sensor, _ := sensor.New(nil, fake, cfg)
    _, err := sensor.Readings(context.Background())

    assert.Error(t, err)
    assert.Contains(t, err.Error(), "I2C timeout")
}
```

## 11.5 Testing with NATS

### Embedded NATS Server

For tests needing real NATS:

```go
//go:build component

package sensor_test

import (
    "testing"
    "time"

    "github.com/nats-io/nats-server/v2/server"
    "github.com/nats-io/nats.go"
)

func startTestNATS(t *testing.T) *nats.Conn {
    t.Helper()

    opts := &server.Options{
        Host:       "127.0.0.1",
        Port:       -1,  // Random port
        JetStream:  true,
        StoreDir:   t.TempDir(),
    }

    ns, err := server.NewServer(opts)
    if err != nil {
        t.Fatal(err)
    }

    go ns.Start()
    if !ns.ReadyForConnections(10 * time.Second) {
        t.Fatal("NATS server not ready")
    }

    t.Cleanup(func() {
        ns.Shutdown()
    })

    nc, err := nats.Connect(ns.ClientURL())
    if err != nil {
        t.Fatal(err)
    }

    t.Cleanup(func() {
        nc.Close()
    })

    return nc
}
```

### Testing Pub/Sub

```go
func TestSensor_PublishesReadings(t *testing.T) {
    nc := startTestNATS(t)

    // Subscribe before publishing
    received := make(chan *sensor.Temperature, 1)
    sub, _ := nc.Subscribe("gorai.sensors.temp.data", func(m *nats.Msg) {
        var temp sensor.Temperature
        proto.Unmarshal(m.Data, &temp)
        received <- &temp
    })
    defer sub.Unsubscribe()

    // Create and start sensor
    n, _ := node.New("test", node.WithConnection(nc))
    sensor := createSensor(n)
    sensor.Start(context.Background())

    // Wait for message
    select {
    case msg := <-received:
        assert.InDelta(t, 25.0, msg.Temperature, 0.1)
    case <-time.After(5 * time.Second):
        t.Fatal("timeout waiting for message")
    }
}
```

### Testing Request/Reply

```go
func TestService_RespondsToRequest(t *testing.T) {
    nc := startTestNATS(t)

    // Set up service
    nc.Subscribe("test.service", func(m *nats.Msg) {
        m.Respond([]byte("pong"))
    })

    // Make request
    resp, err := nc.Request("test.service", []byte("ping"), time.Second)
    assert.NoError(t, err)
    assert.Equal(t, []byte("pong"), resp.Data)
}
```

## 11.6 Component Tests

Test a single component with real dependencies:

```go
//go:build component

func TestTemperatureSensor_Component(t *testing.T) {
    nc := startTestNATS(t)
    n, _ := node.New("test", node.WithConnection(nc))
    defer n.Close()

    fake := fake.NewReader()
    fake.SetTemperature(42.0)

    cfg := sensor.Config{
        Name:     "test_temp",
        Interval: 100 * time.Millisecond,
        Topic:    "test.temp.data",
    }

    s, _ := sensor.New(n, fake, cfg)
    defer s.Close(context.Background())

    // Test readings
    readings, err := s.Readings(context.Background())
    assert.NoError(t, err)
    assert.Equal(t, 42.0, readings["temperature_celsius"])

    // Test publishing
    s.Start(context.Background())
    time.Sleep(200 * time.Millisecond)

    stats, _ := s.DoCommand(context.Background(), map[string]any{"command": "get_stats"})
    assert.Greater(t, stats["reading_count"].(uint64), uint64(0))
}
```

## 11.7 Integration Tests

Test multiple components together:

```go
//go:build integration

func TestVisionPipeline_Integration(t *testing.T) {
    nc := startTestNATS(t)
    n, _ := node.New("test", node.WithConnection(nc))
    defer n.Close()

    // Create fake camera
    camera := fake.NewCamera()
    camera.SetImage(testImage)

    // Create real vision service with fake camera
    vision, _ := vision.New(n, camera, visionConfig)
    defer vision.Close(context.Background())

    // Test detection pipeline
    img, _ := camera.Image(context.Background())
    detections, err := vision.DetectObjects(context.Background(), img)

    assert.NoError(t, err)
    assert.Greater(t, len(detections.Detections), 0)
}
```

## 11.8 Hardware Tests

Tests requiring real hardware:

```go
//go:build hardware && linux

func TestLinuxThermalReader_Hardware(t *testing.T) {
    reader, err := reader.New()
    if err != nil {
        t.Skip("thermal zones not available:", err)
    }
    defer reader.Close()

    zones, _ := reader.Zones(context.Background())
    assert.Greater(t, len(zones), 0)

    reading, err := reader.Read(context.Background(), "")
    assert.NoError(t, err)
    assert.Greater(t, reading.TemperatureC, 0.0)
    assert.Less(t, reading.TemperatureC, 120.0)  // Sanity check
}
```

Run with:
```bash
go test -tags=hardware ./...
```

## 11.9 Coverage Requirements

| Package | Target |
|---------|--------|
| pkg/* | 80% |
| components/* | 75% |
| examples/* | 80% |

Check coverage:
```bash
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## 11.10 CI/CD Integration

GitHub Actions workflow:

```yaml
name: Test

on: [push, pull_request]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'

      - name: Unit Tests
        run: go test -race ./...

      - name: Component Tests
        run: go test -race -tags=component ./...

      - name: Coverage
        run: |
          go test -coverprofile=coverage.out ./...
          go tool cover -func=coverage.out
```

---

Chapter 12 explores integrating AI/ML into your tested components.
