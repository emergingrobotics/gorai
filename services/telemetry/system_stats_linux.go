//go:build linux

package telemetry

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const thermalZonePath = "/sys/class/thermal/thermal_zone0/temp"

// LinuxSystemStatsCollector collects system stats on Linux.
type LinuxSystemStatsCollector struct {
	// Previous CPU stats for calculating usage
	prevIdle  uint64
	prevTotal uint64
	prevTime  time.Time
}

// NewLinuxSystemStatsCollector creates a new Linux system stats collector.
func NewLinuxSystemStatsCollector() *LinuxSystemStatsCollector {
	return &LinuxSystemStatsCollector{}
}

// Collect gathers current system statistics.
func (c *LinuxSystemStatsCollector) Collect() *SystemStatsResult {
	stats := &SystemStatsResult{
		Available: true,
	}

	// CPU usage
	if cpuPercent, err := c.getCPUPercent(); err == nil {
		stats.CpuPercent = cpuPercent
	}

	// Memory usage
	if memInfo, err := getMemoryInfo(); err == nil {
		stats.MemoryPercent = memInfo.UsedPercent
	}

	// Disk usage
	if diskInfo, err := getDiskUsage("/"); err == nil {
		stats.DiskPercent = diskInfo.UsedPercent
	}

	// Temperature
	if temp, err := readThermalZone(); err == nil {
		stats.TemperatureC = temp
	}

	// Uptime
	if uptime, err := getUptime(); err == nil {
		stats.UptimeSeconds = uptime
	}

	// Load average
	if load, err := getLoadAverage(); err == nil && len(load) > 0 {
		stats.LoadAverage_1M = load[0]
	}

	return stats
}

// getCPUPercent calculates CPU usage percentage.
func (c *LinuxSystemStatsCollector) getCPUPercent() (float64, error) {
	file, err := os.Open("/proc/stat")
	if err != nil {
		return 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if !scanner.Scan() {
		return 0, scanner.Err()
	}

	line := scanner.Text()
	if !strings.HasPrefix(line, "cpu ") {
		return 0, nil
	}

	fields := strings.Fields(line)
	if len(fields) < 5 {
		return 0, nil
	}

	var total, idle uint64
	for i := 1; i < len(fields); i++ {
		val, err := strconv.ParseUint(fields[i], 10, 64)
		if err != nil {
			continue
		}
		total += val
		if i == 4 { // idle is the 4th field (0-indexed: 3)
			idle = val
		}
	}

	now := time.Now()

	// Calculate delta
	if c.prevTotal > 0 {
		totalDelta := total - c.prevTotal
		idleDelta := idle - c.prevIdle

		if totalDelta > 0 {
			cpuPercent := 100.0 * float64(totalDelta-idleDelta) / float64(totalDelta)
			c.prevTotal = total
			c.prevIdle = idle
			c.prevTime = now
			return cpuPercent, nil
		}
	}

	c.prevTotal = total
	c.prevIdle = idle
	c.prevTime = now
	return 0, nil
}

// getMemoryInfo reads memory information from /proc/meminfo.
func getMemoryInfo() (*MemInfo, error) {
	file, err := os.Open("/proc/meminfo")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	memInfo := &MemInfo{}
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		key := strings.TrimSuffix(fields[0], ":")
		value, err := strconv.ParseUint(fields[1], 10, 64)
		if err != nil {
			continue
		}
		// Values in /proc/meminfo are in kB
		value *= 1024

		switch key {
		case "MemTotal":
			memInfo.Total = value
		case "MemAvailable":
			memInfo.Available = value
		}
	}

	if memInfo.Total > 0 {
		memInfo.Used = memInfo.Total - memInfo.Available
		memInfo.UsedPercent = 100.0 * float64(memInfo.Used) / float64(memInfo.Total)
	}

	return memInfo, nil
}

// getDiskUsage gets disk usage for a path.
func getDiskUsage(path string) (*DiskInfo, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, err
	}

	diskInfo := &DiskInfo{
		Total: stat.Blocks * uint64(stat.Bsize),
		Free:  stat.Bavail * uint64(stat.Bsize),
	}
	diskInfo.Used = diskInfo.Total - diskInfo.Free

	if diskInfo.Total > 0 {
		diskInfo.UsedPercent = 100.0 * float64(diskInfo.Used) / float64(diskInfo.Total)
	}

	return diskInfo, nil
}

// readThermalZone reads the CPU/SoC temperature.
func readThermalZone() (float64, error) {
	data, err := os.ReadFile(thermalZonePath)
	if err != nil {
		return 0, err
	}

	tempMilli, err := strconv.ParseInt(strings.TrimSpace(string(data)), 10, 64)
	if err != nil {
		return 0, err
	}

	return float64(tempMilli) / 1000.0, nil
}

// getUptime reads system uptime in seconds.
func getUptime() (int64, error) {
	data, err := os.ReadFile("/proc/uptime")
	if err != nil {
		return 0, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 1 {
		return 0, nil
	}

	uptime, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, err
	}

	return int64(uptime), nil
}

// getLoadAverage reads system load averages.
func getLoadAverage() ([]float64, error) {
	data, err := os.ReadFile("/proc/loadavg")
	if err != nil {
		return nil, err
	}

	fields := strings.Fields(string(data))
	if len(fields) < 3 {
		return nil, nil
	}

	loads := make([]float64, 3)
	for i := 0; i < 3; i++ {
		load, err := strconv.ParseFloat(fields[i], 64)
		if err != nil {
			continue
		}
		loads[i] = load
	}

	return loads, nil
}

// NewSystemStatsCollector creates a platform-specific stats collector.
func NewSystemStatsCollector() SystemStatsCollector {
	return NewLinuxSystemStatsCollector()
}
