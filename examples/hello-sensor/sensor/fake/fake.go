// Package fake provides a fake temperature reader for testing.
package fake

import (
	"context"
	"sync"

	"github.com/gorai/gorai/examples/hello-sensor/reader"
)

// Reader is a fake temperature reader for testing.
type Reader struct {
	mu          sync.RWMutex
	temperature float64
	criticalC   float64
	warningC    float64
	err         error
	zones       []string
}

// New creates a new fake reader with default values.
func New() *Reader {
	return &Reader{
		temperature: 42.0,
		criticalC:   100.0,
		warningC:    85.0,
		zones:       []string{"fake_zone0"},
	}
}

// Platform returns "fake".
func (r *Reader) Platform() string {
	return "fake"
}

// Zones returns the fake zones.
func (r *Reader) Zones(ctx context.Context) ([]string, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	if r.err != nil {
		return nil, r.err
	}
	return r.zones, nil
}

// Read returns the configured fake reading.
func (r *Reader) Read(ctx context.Context, zone string) (reader.Reading, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if r.err != nil {
		return reader.Reading{}, r.err
	}

	z := zone
	if z == "" || z == "default" {
		z = r.zones[0]
	}

	return reader.Reading{
		Zone:         z,
		TemperatureC: r.temperature,
		CriticalC:    r.criticalC,
		WarningC:     r.warningC,
	}, nil
}

// Close does nothing for the fake reader.
func (r *Reader) Close() error {
	return nil
}

// SetTemperature sets the temperature to return.
func (r *Reader) SetTemperature(temp float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.temperature = temp
}

// SetThresholds sets the critical and warning thresholds.
func (r *Reader) SetThresholds(critical, warning float64) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.criticalC = critical
	r.warningC = warning
}

// SetError sets an error to return on Read.
func (r *Reader) SetError(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.err = err
}

// SetZones sets the available zones.
func (r *Reader) SetZones(zones []string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.zones = zones
}

// Verify interface compliance.
var _ reader.Reader = (*Reader)(nil)
