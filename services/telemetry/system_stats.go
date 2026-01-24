package telemetry

// SystemStatsCollector provides system statistics.
type SystemStatsCollector interface {
	// Collect gathers current system statistics.
	Collect() *SystemStatsResult
}

// SystemStatsResult holds the collected system stats.
type SystemStatsResult struct {
	Available      bool
	CpuPercent     float64
	MemoryPercent  float64
	DiskPercent    float64
	TemperatureC   float64
	UptimeSeconds  int64
	LoadAverage_1M float64
}

// MemInfo holds memory information.
type MemInfo struct {
	Total       uint64
	Available   uint64
	Used        uint64
	UsedPercent float64
}

// DiskInfo holds disk usage information.
type DiskInfo struct {
	Total       uint64
	Used        uint64
	Free        uint64
	UsedPercent float64
}
