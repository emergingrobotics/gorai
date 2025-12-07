package fake_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gorai/gorai/examples/hello-sensor/sensor/fake"
)

func TestFakeReader_Default(t *testing.T) {
	r := fake.New()

	if r.Platform() != "fake" {
		t.Errorf("Platform() = %q, want 'fake'", r.Platform())
	}

	reading, err := r.Read(context.Background(), "")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if reading.TemperatureC != 42.0 {
		t.Errorf("default temperature = %v, want 42.0", reading.TemperatureC)
	}
}

func TestFakeReader_SetTemperature(t *testing.T) {
	r := fake.New()
	r.SetTemperature(55.5)

	reading, err := r.Read(context.Background(), "")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if reading.TemperatureC != 55.5 {
		t.Errorf("temperature = %v, want 55.5", reading.TemperatureC)
	}
}

func TestFakeReader_SetThresholds(t *testing.T) {
	r := fake.New()
	r.SetThresholds(95.0, 80.0)

	reading, err := r.Read(context.Background(), "")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}

	if reading.CriticalC != 95.0 {
		t.Errorf("critical = %v, want 95.0", reading.CriticalC)
	}
	if reading.WarningC != 80.0 {
		t.Errorf("warning = %v, want 80.0", reading.WarningC)
	}
}

func TestFakeReader_SetError(t *testing.T) {
	r := fake.New()
	testErr := errors.New("sensor failure")
	r.SetError(testErr)

	_, err := r.Read(context.Background(), "")
	if err != testErr {
		t.Errorf("error = %v, want %v", err, testErr)
	}
}

func TestFakeReader_Zones(t *testing.T) {
	r := fake.New()
	r.SetZones([]string{"zone1", "zone2"})

	zones, err := r.Zones(context.Background())
	if err != nil {
		t.Fatalf("Zones failed: %v", err)
	}

	if len(zones) != 2 {
		t.Errorf("got %d zones, want 2", len(zones))
	}
	if zones[0] != "zone1" || zones[1] != "zone2" {
		t.Errorf("zones = %v, want [zone1, zone2]", zones)
	}
}

func TestFakeReader_Close(t *testing.T) {
	r := fake.New()
	if err := r.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
