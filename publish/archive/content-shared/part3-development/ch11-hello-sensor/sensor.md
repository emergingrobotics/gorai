## 9.4 The Sensor Component

The sensor package wraps the reader in a GoRAI component.

### 9.4.1 Implementing resource.Resource

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

**Name() implementation**:

```go
func (s *TemperatureSensor) Name() resource.Name {
    return s.name
}
```

The name is created during construction:

```go
name := resource.NewComponentName("gorai", "sensor", cfg.Name)
```

### 9.4.2 Implementing resource.Sensor

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

### 9.4.3 Reconfigure() for Hot Reload

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

### 9.4.4 DoCommand() for Extensibility

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

### 9.4.5 Publishing Loop

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

### 9.4.6 Statistics Tracking

```go
func (s *TemperatureSensor) updateStats(temp float64) {
    if s.statCount == 0 {
        s.minTemp = temp
        s.maxTemp = temp
    } else {
        if temp < s.minTemp {
            s.minTemp = temp
        }
        if temp > s.maxTemp {
            s.maxTemp = temp
        }
    }
    s.sumTemp += temp
    s.statCount++
}
```

Simple but useful for monitoring temperature over time.
