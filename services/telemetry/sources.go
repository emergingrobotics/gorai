package telemetry

import (
	"sync"
	"sync/atomic"
	"time"
)

// SourceData holds the latest data received from a source.
type SourceData struct {
	Data       any
	ReceivedAt time.Time
	Type       SourceType
}

// SourceStats tracks statistics for a data source.
type SourceStats struct {
	MessagesReceived atomic.Uint64
	MessagesDropped  atomic.Uint64
	LastReceived     time.Time
	ActualRate       float64

	// Rate tracking
	mu         sync.Mutex
	timestamps []time.Time
}

// UpdateRate updates the rate calculation using a sliding window.
func (s *SourceStats) UpdateRate() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.LastReceived = now

	// Add timestamp to sliding window
	s.timestamps = append(s.timestamps, now)

	// Remove timestamps older than 1 second
	cutoff := now.Add(-time.Second)
	for len(s.timestamps) > 0 && s.timestamps[0].Before(cutoff) {
		s.timestamps = s.timestamps[1:]
	}

	// Calculate rate
	s.ActualRate = float64(len(s.timestamps))
}

// GetRate returns the current rate.
func (s *SourceStats) GetRate() float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ActualRate
}

// ExtractGenericValue extracts a numeric value from generic source data.
func ExtractGenericValue(data *SourceData) float64 {
	if data == nil || data.Data == nil {
		return 0
	}

	switch v := data.Data.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	default:
		return 0
	}
}
