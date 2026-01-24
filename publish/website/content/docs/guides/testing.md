---
title: "Testing"
description: "Test Gorai components and services"
weight: 50
---

# Testing Guide

Robots are safety-critical systems. Testing is not optional—it's essential. Gorai provides patterns and tools for comprehensive testing.

## The Testing Pyramid

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

Catch bugs at the lowest level possible. Unit tests are fast and pinpoint problems.

## Test Categories and Build Tags

| Tag | Purpose | Speed | NATS | Hardware |
|-----|---------|-------|------|----------|
| (none) | Unit tests | <1s | No | No |
| `component` | Single component | 1-5s | Embedded | No |
| `integration` | Multi-component | 5-30s | Embedded | No |
| `hardware` | Real hardware | Variable | Real | Yes |

Run specific categories:

```bash
go test ./...                                    # Unit only
go test -tags=component ./...                   # Component
go test -tags=integration ./...                 # Integration
go test -tags="component integration" ./...     # Both
```

## Unit Testing Patterns

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
        {"negative", -40, -40},
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
    t.Helper()
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
    t.Parallel()
    motor := fake.NewMotor()
    // Test...
}
```

## Fake Implementations

Every component needs a fake for testing without hardware.

### Basic Fake

```go
type FakeReader struct {
    mu          sync.RWMutex
    temperature float64
    shouldError bool
    errorMsg    string
}

func (f *FakeReader) SetTemperature(celsius float64) {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.temperature = celsius
}

func (f *FakeReader) SetError(msg string) {
    f.mu.Lock()
    defer f.mu.Unlock()
    f.shouldError = true
    f.errorMsg = msg
}

func (f *FakeReader) Read(ctx context.Context, zone string) (Reading, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()

    if f.shouldError {
        return Reading{}, fmt.Errorf(f.errorMsg)
    }

    return Reading{
        Zone:         zone,
        TemperatureC: f.temperature,
    }, nil
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

## Testing with NATS

### Embedded NATS Server

For tests needing real NATS:

```go
//go:build component

func startTestNATS(t *testing.T) *nats.Conn {
    t.Helper()

    opts := &server.Options{
        Host:      "127.0.0.1",
        Port:      -1,  // Random port
        JetStream: true,
        StoreDir:  t.TempDir(),
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

    received := make(chan []byte, 1)
    nc.Subscribe("sensors.temp.data", func(m *nats.Msg) {
        received <- m.Data
    })

    // Create and start sensor
    sensor := createSensor(nc)
    sensor.Start(context.Background())

    select {
    case msg := <-received:
        // Verify message content
    case <-time.After(5 * time.Second):
        t.Fatal("timeout waiting for message")
    }
}
```

## Coverage Requirements

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

## CI/CD Integration

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

## Next Steps

- [Examples](/examples/)
- [Configuration Guide](../configuration/)
- [CLI Reference](/docs/reference/cli/)
