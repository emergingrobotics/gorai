//go:build darwin

package reader

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

type darwinReader struct{}

func newDarwinReader() (Reader, error) {
	return &darwinReader{}, nil
}

func (r *darwinReader) Platform() string {
	return "darwin"
}

func (r *darwinReader) Zones(ctx context.Context) ([]string, error) {
	return []string{"cpu"}, nil
}

func (r *darwinReader) Read(ctx context.Context, zone string) (Reading, error) {
	reading := Reading{Zone: "cpu"}

	// Try osx-cpu-temp first (brew install osx-cpu-temp)
	if temp, err := r.readOSXCPUTemp(ctx); err == nil {
		reading.TemperatureC = temp
		return reading, nil
	}

	// Try powermetrics (requires sudo, may not work)
	if temp, err := r.readPowermetrics(ctx); err == nil {
		reading.TemperatureC = temp
		return reading, nil
	}

	return reading, fmt.Errorf("no temperature source available on macOS; install osx-cpu-temp: brew install osx-cpu-temp")
}

func (r *darwinReader) readOSXCPUTemp(ctx context.Context) (float64, error) {
	cmd := exec.CommandContext(ctx, "osx-cpu-temp")
	output, err := cmd.Output()
	if err != nil {
		return 0, err
	}

	// Output format: "45.2°C"
	tempStr := strings.TrimSpace(string(output))
	tempStr = strings.TrimSuffix(tempStr, "°C")
	return strconv.ParseFloat(tempStr, 64)
}

func (r *darwinReader) readPowermetrics(ctx context.Context) (float64, error) {
	// powermetrics requires root and is complex to parse
	return 0, fmt.Errorf("powermetrics not implemented")
}

func (r *darwinReader) Close() error {
	return nil
}

// newLinuxReader is a stub for Darwin builds.
func newLinuxReader() (Reader, error) {
	return nil, fmt.Errorf("linux reader not available on darwin")
}
