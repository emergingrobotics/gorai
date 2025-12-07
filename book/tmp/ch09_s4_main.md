## 9.5 The Fake Reader

Test doubles enable testing without hardware.

From `sensor/fake/fake.go`:

```go
package fake

type FakeReader struct {
    mu          sync.RWMutex
    temperature float64
    zones       []string
    shouldError bool
    errorMsg    string
}

func New() *FakeReader {
    return &FakeReader{
        temperature: 42.0,
        zones:       []string{"fake_zone0"},
    }
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

func (f *FakeReader) Read(ctx context.Context, zone string) (reader.Reading, error) {
    f.mu.RLock()
    defer f.mu.RUnlock()

    if f.shouldError {
        return reader.Reading{}, fmt.Errorf(f.errorMsg)
    }

    return reader.Reading{
        Zone:         zone,
        TemperatureC: f.temperature,
        CriticalC:    100.0,
        WarningC:     80.0,
    }, nil
}
```

**Key features**:
- `SetTemperature()`: Control returned value
- `SetError()`: Simulate hardware failures
- Thread-safe with mutex
- Implements full Reader interface


## 9.6 Main Entry Point

From `main.go`:

```go
func main() {
    // Parse flags
    natsURL := flag.String("nats", "nats://localhost:4222", "NATS server URL")
    interval := flag.Duration("interval", time.Second, "Publishing interval")
    topic := flag.String("topic", "gorai.hello.cpu_temp.data", "NATS topic")
    zone := flag.String("zone", "", "Thermal zone to read")
    useFake := flag.Bool("fake", false, "Use fake reader")
    fakeTemp := flag.Float64("fake-temp", 42.0, "Temperature for fake reader")
    flag.Parse()

    // Create context with signal handling
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigCh
        log.Println("Shutting down...")
        cancel()
    }()

    // Create node
    n, err := node.New("hello_sensor", node.WithNATS(*natsURL))
    if err != nil {
        log.Fatalf("Failed to create node: %v", err)
    }
    defer n.Close()

    // Create reader
    var r reader.Reader
    if *useFake {
        fr := fake.New()
        fr.SetTemperature(*fakeTemp)
        r = fr
        log.Printf("Using fake reader with temperature %.1f°C", *fakeTemp)
    } else {
        r, err = reader.New()
        if err != nil {
            log.Fatalf("Failed to create reader: %v", err)
        }
    }
    defer r.Close()

    // Create sensor
    cfg := sensor.Config{
        Name:     "cpu_temp",
        Zone:     *zone,
        Interval: *interval,
        Topic:    *topic,
    }

    tempSensor, err := sensor.New(n, r, cfg)
    if err != nil {
        log.Fatalf("Failed to create sensor: %v", err)
    }
    defer tempSensor.Close(ctx)

    // Start publishing
    log.Printf("Starting sensor, publishing to %s every %v", cfg.Topic, cfg.Interval)
    if err := tempSensor.Start(ctx); err != nil {
        log.Fatalf("Failed to start sensor: %v", err)
    }

    // Wait for shutdown
    <-ctx.Done()
}
```

**Patterns demonstrated**:
1. **Flag parsing**: Configurable without recompiling
2. **Signal handling**: Graceful shutdown on Ctrl+C
3. **Dependency injection**: Reader selected at runtime
4. **Resource cleanup**: defer statements ensure cleanup


## 9.7 Running the Example

### Start NATS

```bash
./scripts/start.sh
# or
nats-server -js &
```

### Run with Fake Reader

```bash
./scripts/hello.sh -fake -fake-temp 65.0

# or directly
go run ./examples/hello-sensor -fake -fake-temp 65.0
```

Output:
```
Using fake reader with temperature 65.0°C
Platform: fake, Available zones: [fake_zone0]
Initial reading: 65.0°C (149.0°F)
Starting sensor, publishing to gorai.hello.cpu_temp.data every 1s
```

### Subscribe to Readings

In another terminal:

```bash
nats sub "gorai.hello.cpu_temp.data"
```

Output:
```
[#1] Received on "gorai.hello.cpu_temp.data"
{"timestamp":"2024-01-15T10:30:00Z","temperature_celsius":65,"temperature_fahrenheit":149,"zone":"fake_zone0","source":"cpu"}
```

### Run with Real Hardware

```bash
go run ./examples/hello-sensor
```

On Linux, this reads from `/sys/class/thermal/thermal_zone*/temp`.


## 9.8 Observing the Output

### JSON Message Format

Published messages are JSON:

```json
{
  "timestamp": "2024-01-15T10:30:00.123Z",
  "temperature_celsius": 42.5,
  "temperature_fahrenheit": 108.5,
  "zone": "thermal_zone0",
  "source": "cpu",
  "critical_celsius": 105.0,
  "warning_celsius": 85.0
}
```

### Statistics via DoCommand

Query sensor statistics:

```bash
# In your code or via test
stats, _ := tempSensor.DoCommand(ctx, map[string]any{"command": "get_stats"})
fmt.Printf("Readings: %v, Errors: %v\n", stats["reading_count"], stats["error_count"])
fmt.Printf("Min: %.1f°C, Max: %.1f°C, Avg: %.1f°C\n",
    stats["min_celsius"], stats["max_celsius"], stats["avg_celsius"])
```

### Monitoring Message Flow

```bash
# Watch all messages
nats sub ">" --raw

# Count messages per second
watch -n1 "nats sub gorai.hello.cpu_temp.data --count 1 2>/dev/null"
```

*Cross-reference: See Chapter 3 for NATS CLI usage.*

---

With hello-sensor thoroughly understood, Chapter 10 shows how to build your own custom components following these same patterns.
