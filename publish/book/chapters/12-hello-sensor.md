# Hello Sensor Deep Dive

The best way to learn Gorai is to build something real. This chapter walks through the "hello-sensor" example in complete detail—a CPU temperature sensor that reads thermal data, publishes to NATS, and includes proper testing infrastructure.

This isn't a toy example. The patterns here—platform abstraction, fake implementations, statistics tracking, graceful shutdown—are exactly what you'll use in production code.

## What We're Building

The hello-sensor reads CPU temperature from the host system and publishes it to NATS. It demonstrates:

- Creating a Gorai node
- Platform-specific hardware access
- Implementing the Sensor interface
- Publishing structured messages
- Configuration and command-line flags
- Statistics collection
- Graceful shutdown
- Fake implementations for testing

The complete code is in `examples/hello-sensor/`.

## Architecture Overview

```
┌─────────────────────────────────────────┐
│              hello-sensor               │
├─────────────────────────────────────────┤
│  main.go                                │
│    ├── Create node                      │
│    ├── Create reader (platform-specific)│
│    ├── Create sensor component          │
│    └── Run publish loop                 │
├─────────────────────────────────────────┤
│  reader/                                │
│    ├── reader.go (interface)            │
│    ├── linux.go (thermal zones)         │
│    ├── darwin.go (osx-cpu-temp)         │
│    └── unsupported.go (stub)            │
├─────────────────────────────────────────┤
│  sensor/                                │
│    ├── temperature.go (component)       │
│    └── fake/fake.go (test double)       │
└─────────────────────────────────────────┘
```

**Separation of concerns**:

- `reader/`: Platform-specific temperature reading
- `sensor/`: Gorai component wrapping the reader
- `main.go`: Entry point orchestrating everything

## The Reader Package

The reader package abstracts platform-specific temperature reading behind a common interface.

### Interface Design

From `reader/reader.go`:

```go
// Reading represents a temperature reading from a thermal zone.
type Reading struct {
    Zone         string
    TemperatureC float64
    CriticalC    float64 // 0 if unknown
    WarningC     float64 // 0 if unknown
}

// Reader reads temperature from the host system.
type Reader interface {
    // Platform returns the platform name (e.g., "linux", "darwin").
    Platform() string

    // Zones returns available thermal zones.
    Zones(ctx context.Context) ([]string, error)

    // Read returns temperature for a specific zone.
    // Use "" or "default" for the primary zone.
    Read(ctx context.Context, zone string) (Reading, error)

    // Close releases resources.
    Close() error
}
```

**Design decisions**:

- **Platform()**: Identify which implementation is running
- **Zones()**: Discover available temperature sources
- **Read()**: Get temperature for a specific zone
- **Close()**: Clean shutdown pattern

The factory function selects the right implementation:

```go
func New() (Reader, error) {
    switch runtime.GOOS {
    case "linux":
        return newLinuxReader()
    case "darwin":
        return newDarwinReader()
    default:
        return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
    }
}
```

### Linux Implementation

Linux exposes thermal data via sysfs. From `reader/linux.go`:

```go
//go:build linux

package reader

const thermalBasePath = "/sys/class/thermal"

type linuxReader struct {
    zones []string
}

func newLinuxReader() (Reader, error) {
    r := &linuxReader{}

    // Discover thermal zones
    entries, err := os.ReadDir(thermalBasePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read thermal directory: %w", err)
    }

    for _, entry := range entries {
        if strings.HasPrefix(entry.Name(), "thermal_zone") {
            r.zones = append(r.zones, entry.Name())
        }
    }

    if len(r.zones) == 0 {
        return nil, fmt.Errorf("no thermal zones found")
    }

    return r, nil
}

func (r *linuxReader) Read(ctx context.Context, zone string) (Reading, error) {
    if zone == "" || zone == "default" {
        zone = r.zones[0]
    }

    reading := Reading{Zone: zone}

    // Read temperature (in millidegrees Celsius)
    tempPath := filepath.Join(thermalBasePath, zone, "temp")
    tempData, err := os.ReadFile(tempPath)
    if err != nil {
        return reading, fmt.Errorf("failed to read temperature: %w", err)
    }

    tempMilliC, err := strconv.ParseInt(strings.TrimSpace(string(tempData)), 10, 64)
    if err != nil {
        return reading, fmt.Errorf("failed to parse temperature: %w", err)
    }
    reading.TemperatureC = float64(tempMilliC) / 1000.0

    // Try to read trip points (optional)
    reading.CriticalC = r.readTripPoint(zone, "critical")
    reading.WarningC = r.readTripPoint(zone, "hot")

    return reading, nil
}
```

**Key points**:

- Temperature is in millidegrees (divide by 1000)
- Multiple thermal zones exist (CPU, GPU, WiFi, etc.)
- Trip points indicate thermal limits
- File reads can fail—handle errors gracefully

### Build Tags for Platform-Specific Code

Go build tags select which files compile:

```go
//go:build linux
```

This file only compiles on Linux. The darwin.go file has:

```go
//go:build darwin
```

This pattern provides:

- Compile-time platform selection
- Clean separation of platform code
- No runtime overhead
- IDE support (shows correct file for platform)

## The Sensor Component

The sensor package wraps the reader in a Gorai component.

### Implementing resource.Resource

From `sensor/temperature.go`:

```go
type TemperatureSensor struct {
    name   resource.Name
    config Config
    reader reader.Reader
    nc     *nats.Conn

    mu           sync.RWMutex
    running      bool
    cancel       context.CancelFunc
    readingCount uint64
    errorCount   uint64
    lastError    string
    lastReading  *TemperatureReading

    // Stats for diagnostics
    minTemp   float64
    maxTemp   float64
    sumTemp   float64
    statCount int
}
```

**State management**:

- `mu`: Protects concurrent access
- `running`: Tracks if publishing is active
- `cancel`: For stopping the publish loop
- Statistics for debugging and monitoring

### Implementing resource.Sensor

The key method for sensors:

```go
func (s *TemperatureSensor) Readings(ctx context.Context) (map[string]any, error) {
    reading, err := s.reader.Read(ctx, s.config.Zone)
    if err != nil {
        s.mu.Lock()
        s.errorCount++
        s.lastError = err.Error()
        s.mu.Unlock()
        return nil, err
    }

    s.mu.Lock()
    s.readingCount++
    s.updateStats(reading.TemperatureC)
    s.mu.Unlock()

    return map[string]any{
        "temperature_celsius":    reading.TemperatureC,
        "temperature_fahrenheit": reader.CelsiusToFahrenheit(reading.TemperatureC),
        "zone":                   reading.Zone,
        "critical_celsius":       reading.CriticalC,
        "warning_celsius":        reading.WarningC,
        "platform":               s.reader.Platform(),
    }, nil
}
```

**Pattern**: Read hardware, update stats, return map.

### Reconfigure() for Hot Reload

```go
func (s *TemperatureSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
    var cfg Config
    if err := conf.Unmarshal(&cfg); err != nil {
        return fmt.Errorf("failed to parse config: %w", err)
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    if cfg.Interval > 0 {
        s.config.Interval = cfg.Interval
    }
    if cfg.Topic != "" && cfg.Topic != s.config.Topic {
        s.config.Topic = cfg.Topic
    }

    return nil
}
```

Configuration changes take effect without restart.

### DoCommand() for Extensibility

Custom commands beyond the standard interface:

```go
func (s *TemperatureSensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
    if cmdName, ok := cmd["command"].(string); ok {
        switch cmdName {
        case "get_last_reading":
            s.mu.RLock()
            reading := s.lastReading
            s.mu.RUnlock()

            if reading == nil {
                return nil, fmt.Errorf("no reading available")
            }

            return map[string]any{
                "temperature_celsius":    reading.TemperatureCelsius,
                "temperature_fahrenheit": reading.TemperatureFahrenheit,
                "zone":                   reading.Zone,
            }, nil

        case "get_stats":
            s.mu.RLock()
            defer s.mu.RUnlock()

            avg := 0.0
            if s.statCount > 0 {
                avg = s.sumTemp / float64(s.statCount)
            }

            return map[string]any{
                "reading_count": s.readingCount,
                "error_count":   s.errorCount,
                "last_error":    s.lastError,
                "min_celsius":   s.minTemp,
                "max_celsius":   s.maxTemp,
                "avg_celsius":   avg,
            }, nil
        }
    }
    return nil, fmt.Errorf("unknown command: %v", cmd)
}
```

**Use cases**:

- Diagnostics and debugging
- Custom operations not in standard interface
- Integration with management tools

### Publishing Loop

The sensor publishes periodically:

```go
func (s *TemperatureSensor) Start(ctx context.Context) error {
    s.mu.Lock()
    if s.running {
        s.mu.Unlock()
        return fmt.Errorf("already running")
    }
    s.running = true

    ctx, cancel := context.WithCancel(ctx)
    s.cancel = cancel
    s.mu.Unlock()

    go s.run(ctx)
    return nil
}

func (s *TemperatureSensor) run(ctx context.Context) {
    ticker := time.NewTicker(s.config.Interval)
    defer ticker.Stop()

    for {
        select {
        case <-ctx.Done():
            s.mu.Lock()
            s.running = false
            s.mu.Unlock()
            return

        case <-ticker.C:
            s.publishReading(ctx)
        }
    }
}
```

**Pattern**: Ticker-based loop with context cancellation.

## The Fake Reader

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

## Main Entry Point

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

## Running the Example

### Start NATS

```bash
nats-server -js &
```

### Run with Fake Reader

```bash
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

## Observing the Output

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

```go
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

---

With hello-sensor thoroughly understood, Chapter 13 shows how to build your own custom components following these same patterns.
