//go:build linux

package reader

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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

func (r *linuxReader) Platform() string {
	return "linux"
}

func (r *linuxReader) Zones(ctx context.Context) ([]string, error) {
	return r.zones, nil
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

func (r *linuxReader) readTripPoint(zone, tripType string) float64 {
	basePath := filepath.Join(thermalBasePath, zone)
	entries, err := os.ReadDir(basePath)
	if err != nil {
		return 0
	}

	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), "trip_point_") && strings.HasSuffix(entry.Name(), "_type") {
			typeData, err := os.ReadFile(filepath.Join(basePath, entry.Name()))
			if err != nil {
				continue
			}

			if strings.TrimSpace(string(typeData)) == tripType {
				// Found the trip type, read corresponding temp
				tempName := strings.Replace(entry.Name(), "_type", "_temp", 1)
				tempData, err := os.ReadFile(filepath.Join(basePath, tempName))
				if err != nil {
					continue
				}

				tempMilliC, err := strconv.ParseInt(strings.TrimSpace(string(tempData)), 10, 64)
				if err != nil {
					continue
				}
				return float64(tempMilliC) / 1000.0
			}
		}
	}

	return 0
}

func (r *linuxReader) Close() error {
	return nil
}

// newDarwinReader is a stub for Linux builds.
func newDarwinReader() (Reader, error) {
	return nil, fmt.Errorf("darwin reader not available on linux")
}
