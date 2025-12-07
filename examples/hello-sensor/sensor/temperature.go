// Package sensor provides the temperature sensor component.
package sensor

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorai/gorai/examples/hello-sensor/reader"
	"github.com/gorai/gorai/pkg/resource"
	"github.com/nats-io/nats.go"
)

// NATSGetter provides access to NATS connection.
type NATSGetter interface {
	NATS() *nats.Conn
}

// Config holds temperature sensor configuration.
type Config struct {
	// Name is the sensor name.
	Name string `json:"name"`

	// Zone is the thermal zone to read (empty for default).
	Zone string `json:"zone"`

	// Interval is the publishing interval.
	Interval time.Duration `json:"interval"`

	// Topic is the NATS topic for publishing readings.
	Topic string `json:"topic"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		Name:     "cpu_temp",
		Zone:     "",
		Interval: time.Second,
		Topic:    "gorai.hello.cpu_temp.data",
	}
}

// TemperatureReading represents a temperature measurement.
type TemperatureReading struct {
	Timestamp             time.Time `json:"timestamp"`
	TemperatureCelsius    float64   `json:"temperature_celsius"`
	TemperatureFahrenheit float64   `json:"temperature_fahrenheit"`
	Zone                  string    `json:"zone"`
	Source                string    `json:"source"`
	CriticalCelsius       float64   `json:"critical_celsius,omitempty"`
	WarningCelsius        float64   `json:"warning_celsius,omitempty"`
}

// TemperatureSensor reads and publishes CPU temperature.
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

// New creates a new temperature sensor.
func New(n NATSGetter, r reader.Reader, cfg Config) (*TemperatureSensor, error) {
	if r == nil {
		return nil, fmt.Errorf("reader is required")
	}

	if cfg.Name == "" {
		cfg.Name = "cpu_temp"
	}
	if cfg.Interval == 0 {
		cfg.Interval = time.Second
	}
	if cfg.Topic == "" {
		cfg.Topic = "gorai.hello.cpu_temp.data"
	}

	name := resource.NewComponentName("gorai", "sensor", cfg.Name)

	ts := &TemperatureSensor{
		name:   name,
		config: cfg,
		reader: r,
		nc:     n.NATS(),
	}

	return ts, nil
}

// Name returns the sensor's resource name.
func (s *TemperatureSensor) Name() resource.Name {
	return s.name
}

// Reconfigure updates the sensor configuration.
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
		// Would need to recreate publisher here
	}

	return nil
}

// DoCommand handles arbitrary commands.
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

// Readings returns the current sensor readings.
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

// updateStats updates min/max/sum statistics (must hold lock).
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

// Start begins periodic temperature publishing.
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

// run is the main publishing loop.
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

// publishReading reads and publishes a single reading.
func (s *TemperatureSensor) publishReading(ctx context.Context) {
	reading, err := s.reader.Read(ctx, s.config.Zone)
	if err != nil {
		s.mu.Lock()
		s.errorCount++
		s.lastError = err.Error()
		s.mu.Unlock()
		return
	}

	s.mu.Lock()
	s.readingCount++
	s.updateStats(reading.TemperatureC)
	s.mu.Unlock()

	tempReading := &TemperatureReading{
		Timestamp:             time.Now(),
		TemperatureCelsius:    reading.TemperatureC,
		TemperatureFahrenheit: reader.CelsiusToFahrenheit(reading.TemperatureC),
		Source:                "cpu",
		Zone:                  reading.Zone,
		CriticalCelsius:       reading.CriticalC,
		WarningCelsius:        reading.WarningC,
	}

	s.mu.Lock()
	s.lastReading = tempReading
	s.mu.Unlock()

	if s.nc != nil {
		data, err := json.Marshal(tempReading)
		if err == nil {
			s.nc.Publish(s.config.Topic, data)
		}
	}
}

// IsRunning returns true if the sensor is actively publishing.
func (s *TemperatureSensor) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Close stops the sensor and releases resources.
func (s *TemperatureSensor) Close(ctx context.Context) error {
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()

	return s.reader.Close()
}

// Platform returns the reader's platform.
func (s *TemperatureSensor) Platform() string {
	return s.reader.Platform()
}

// Zones returns the available thermal zones.
func (s *TemperatureSensor) Zones(ctx context.Context) ([]string, error) {
	return s.reader.Zones(ctx)
}
