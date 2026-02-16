//go:build !linux

package telemetry

// stubStatsCollector is a no-op collector for non-Linux platforms.
type stubStatsCollector struct{}

func (s *stubStatsCollector) Collect() *SystemStatsResult {
	return &SystemStatsResult{Available: false}
}

// NewSystemStatsCollector creates a platform-specific stats collector.
// On non-Linux platforms this returns a stub that reports stats as unavailable.
func NewSystemStatsCollector() SystemStatsCollector {
	return &stubStatsCollector{}
}
