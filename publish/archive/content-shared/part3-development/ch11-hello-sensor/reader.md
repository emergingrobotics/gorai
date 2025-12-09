## 9.3 The Reader Package

The reader package abstracts platform-specific temperature reading behind a common interface.

### 9.3.1 Interface Design

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

### 9.3.2 Linux Implementation

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

### 9.3.3 Build Tags for Platform-Specific Code

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

**Stubs for unsupported platforms**:

When linux.go compiles, it includes a stub for darwin:

```go
// In linux.go
func newDarwinReader() (Reader, error) {
    return nil, fmt.Errorf("darwin reader not available on linux")
}
```

This ensures the `New()` function compiles on all platforms.

### Helper Functions

Temperature conversion:

```go
func CelsiusToFahrenheit(celsius float64) float64 {
    return celsius*9/5 + 32
}
```

Simple, but consistency matters—define it once, use everywhere.
