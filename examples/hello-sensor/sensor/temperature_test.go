package sensor_test

import (
	"context"
	"testing"
	"time"

	"github.com/gorai/gorai/examples/hello-sensor/sensor"
	"github.com/gorai/gorai/examples/hello-sensor/sensor/fake"
	"github.com/nats-io/nats.go"
)

// mockNATSGetter provides a mock NATS getter for testing.
type mockNATSGetter struct {
	nc *nats.Conn
}

func (m *mockNATSGetter) NATS() *nats.Conn {
	return m.nc
}

func TestTemperatureSensor_New(t *testing.T) {
	r := fake.New()
	cfg := sensor.DefaultConfig()

	mock := &mockNATSGetter{nc: nil}
	s, err := sensor.New(mock, r, cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	if s.Platform() != "fake" {
		t.Errorf("Platform() = %q, want 'fake'", s.Platform())
	}
}

func TestTemperatureSensor_NilReader(t *testing.T) {
	mock := &mockNATSGetter{nc: nil}
	_, err := sensor.New(mock, nil, sensor.DefaultConfig())
	if err == nil {
		t.Error("expected error for nil reader")
	}
}

func TestTemperatureSensor_Readings(t *testing.T) {
	r := fake.New()
	r.SetTemperature(55.0)

	mock := &mockNATSGetter{nc: nil}
	s, err := sensor.New(mock, r, sensor.DefaultConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()
	readings, err := s.Readings(ctx)
	if err != nil {
		t.Fatalf("Readings failed: %v", err)
	}

	if temp := readings["temperature_celsius"].(float64); temp != 55.0 {
		t.Errorf("temperature_celsius = %v, want 55.0", temp)
	}

	if fahrenheit := readings["temperature_fahrenheit"].(float64); fahrenheit != 131.0 {
		t.Errorf("temperature_fahrenheit = %v, want 131.0", fahrenheit)
	}

	if platform := readings["platform"].(string); platform != "fake" {
		t.Errorf("platform = %q, want 'fake'", platform)
	}
}

func TestTemperatureSensor_Name(t *testing.T) {
	r := fake.New()
	cfg := sensor.Config{Name: "test_sensor"}

	mock := &mockNATSGetter{nc: nil}
	s, err := sensor.New(mock, r, cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	expected := "gorai:component:sensor/test_sensor"
	if s.Name().String() != expected {
		t.Errorf("Name() = %q, want %q", s.Name().String(), expected)
	}
}

func TestTemperatureSensor_Zones(t *testing.T) {
	r := fake.New()
	r.SetZones([]string{"zone0", "zone1"})

	mock := &mockNATSGetter{nc: nil}
	s, err := sensor.New(mock, r, sensor.DefaultConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()
	zones, err := s.Zones(ctx)
	if err != nil {
		t.Fatalf("Zones failed: %v", err)
	}

	if len(zones) != 2 {
		t.Errorf("got %d zones, want 2", len(zones))
	}
}

func TestTemperatureSensor_DoCommand_GetStats(t *testing.T) {
	r := fake.New()
	r.SetTemperature(50.0)

	mock := &mockNATSGetter{nc: nil}
	s, err := sensor.New(mock, r, sensor.DefaultConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()

	// Take a few readings to generate stats
	for i := 0; i < 3; i++ {
		r.SetTemperature(float64(40 + i*10)) // 40, 50, 60
		s.Readings(ctx)
	}

	stats, err := s.DoCommand(ctx, map[string]any{"command": "get_stats"})
	if err != nil {
		t.Fatalf("DoCommand failed: %v", err)
	}

	if count := stats["reading_count"].(uint64); count != 3 {
		t.Errorf("reading_count = %v, want 3", count)
	}

	if min := stats["min_celsius"].(float64); min != 40.0 {
		t.Errorf("min_celsius = %v, want 40.0", min)
	}

	if max := stats["max_celsius"].(float64); max != 60.0 {
		t.Errorf("max_celsius = %v, want 60.0", max)
	}
}

func TestTemperatureSensor_StartStop(t *testing.T) {
	r := fake.New()

	mock := &mockNATSGetter{nc: nil}
	cfg := sensor.Config{
		Name:     "test",
		Interval: 50 * time.Millisecond,
	}

	s, err := sensor.New(mock, r, cfg)
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start should succeed
	if err := s.Start(ctx); err != nil {
		t.Fatalf("Start failed: %v", err)
	}

	if !s.IsRunning() {
		t.Error("expected sensor to be running after Start")
	}

	// Starting again should fail
	if err := s.Start(ctx); err == nil {
		t.Error("expected error when starting already running sensor")
	}

	// Let it run a bit
	time.Sleep(100 * time.Millisecond)

	// Cancel and wait for cleanup
	cancel()
	time.Sleep(50 * time.Millisecond)

	// Close should work
	if err := s.Close(context.Background()); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestTemperatureSensor_Close(t *testing.T) {
	r := fake.New()

	mock := &mockNATSGetter{nc: nil}
	s, err := sensor.New(mock, r, sensor.DefaultConfig())
	if err != nil {
		t.Fatalf("New failed: %v", err)
	}

	ctx := context.Background()
	if err := s.Close(ctx); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := sensor.DefaultConfig()

	if cfg.Name != "cpu_temp" {
		t.Errorf("Name = %q, want 'cpu_temp'", cfg.Name)
	}

	if cfg.Interval != time.Second {
		t.Errorf("Interval = %v, want 1s", cfg.Interval)
	}

	if cfg.Topic != "gorai.hello.cpu_temp.data" {
		t.Errorf("Topic = %q, want 'gorai.hello.cpu_temp.data'", cfg.Topic)
	}
}
