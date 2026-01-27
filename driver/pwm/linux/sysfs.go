//go:build linux

// Package linux provides Linux PWM implementations via sysfs.
package linux

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gorai/gorai/driver/pwm"
)

// Common errors for PWM operations.
var (
	ErrChipNotFound    = errors.New("PWM chip not found")
	ErrChannelNotFound = errors.New("PWM channel not found")
	ErrExportFailed    = errors.New("failed to export PWM channel")
	ErrChannelBusy     = errors.New("PWM channel is in use")
	ErrInvalidPeriod   = errors.New("invalid period value")
	ErrInvalidDuty     = errors.New("duty cycle exceeds period")
	ErrChipClosed      = errors.New("PWM chip is closed")
)

const (
	// sysfsBasePath is the root of the PWM sysfs interface.
	sysfsBasePath = "/sys/class/pwm"
)

// OpenChip opens a hardware PWM chip by number.
// Returns ErrChipNotFound if the chip doesn't exist.
//
// If the chip is not found, ensure PWM is enabled in the device tree:
//   - Raspberry Pi: Add dtoverlay=pwm-2chan to /boot/config.txt
//   - Orange Pi: Add overlays=pwm14-m0 to /boot/orangepiEnv.txt
func OpenChip(chipNum int) (pwm.Chip, error) {
	chipPath := filepath.Join(sysfsBasePath, fmt.Sprintf("pwmchip%d", chipNum))

	// Verify chip exists
	if _, err := os.Stat(chipPath); os.IsNotExist(err) {
		// Check if any PWM chips exist at all
		if entries, _ := os.ReadDir(sysfsBasePath); len(entries) == 0 {
			return nil, fmt.Errorf("%w: pwmchip%d - no PWM chips found. "+
				"PWM may need to be enabled in device tree (e.g., dtoverlay=pwm-2chan in /boot/config.txt)",
				ErrChipNotFound, chipNum)
		}
		return nil, fmt.Errorf("%w: pwmchip%d at %s (available chips: %v)",
			ErrChipNotFound, chipNum, chipPath, listAvailableChips())
	}

	// Read number of channels
	npwmPath := filepath.Join(chipPath, "npwm")
	data, err := os.ReadFile(npwmPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read npwm from %s: %w", npwmPath, err)
	}

	npwm, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return nil, fmt.Errorf("invalid npwm value %q: %w", string(data), err)
	}

	return &sysfsChip{
		num:      chipNum,
		path:     chipPath,
		npwm:     npwm,
		channels: make(map[int]*sysfsChannel),
	}, nil
}

// AvailableChips returns a list of available PWM chip numbers.
func AvailableChips() ([]int, error) {
	entries, err := os.ReadDir(sysfsBasePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No PWM support
		}
		return nil, fmt.Errorf("failed to read %s: %w", sysfsBasePath, err)
	}

	var chips []int
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, "pwmchip") {
			var num int
			if _, err := fmt.Sscanf(name, "pwmchip%d", &num); err == nil {
				chips = append(chips, num)
			}
		}
	}
	return chips, nil
}

// ChipExists returns true if the specified PWM chip exists.
func ChipExists(chipNum int) bool {
	chipPath := filepath.Join(sysfsBasePath, fmt.Sprintf("pwmchip%d", chipNum))
	_, err := os.Stat(chipPath)
	return err == nil
}

// listAvailableChips returns a string listing available chip numbers for error messages.
func listAvailableChips() string {
	chips, _ := AvailableChips()
	if len(chips) == 0 {
		return "none"
	}
	var parts []string
	for _, c := range chips {
		parts = append(parts, fmt.Sprintf("pwmchip%d", c))
	}
	return strings.Join(parts, ", ")
}
