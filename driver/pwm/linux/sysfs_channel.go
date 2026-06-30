//go:build linux

package linux

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/driver/pwm"
)

const (
	// sysfsWriteDelay is a small delay between sysfs writes to ensure
	// the kernel has time to process previous writes.
	sysfsWriteDelay = 1 * time.Millisecond
)

// sysfsChannel implements pwm.Channel for Linux sysfs PWM.
type sysfsChannel struct {
	chip *sysfsChip
	num  int
	path string
	mu   sync.RWMutex

	// Cached state
	period   uint64 // nanoseconds
	duty     uint64 // nanoseconds
	enabled  bool
	inverted bool
}

// newSysfsChannel creates a channel wrapper for an exported PWM channel.
func newSysfsChannel(chip *sysfsChip, num int, path string) (*sysfsChannel, error) {
	ch := &sysfsChannel{
		chip: chip,
		num:  num,
		path: path,
	}

	// Read current state from sysfs
	if err := ch.syncState(); err != nil {
		return nil, fmt.Errorf("failed to read channel %d state: %w", num, err)
	}

	return ch, nil
}

// syncState reads current values from sysfs files.
func (ch *sysfsChannel) syncState() error {
	// Read period
	if data, err := os.ReadFile(filepath.Join(ch.path, "period")); err == nil {
		if v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			ch.period = v
		}
	}

	// Read duty_cycle
	if data, err := os.ReadFile(filepath.Join(ch.path, "duty_cycle")); err == nil {
		if v, err := strconv.ParseUint(strings.TrimSpace(string(data)), 10, 64); err == nil {
			ch.duty = v
		}
	}

	// Read enable
	if data, err := os.ReadFile(filepath.Join(ch.path, "enable")); err == nil {
		ch.enabled = strings.TrimSpace(string(data)) == "1"
	}

	// Read polarity (may not be supported on all systems)
	if data, err := os.ReadFile(filepath.Join(ch.path, "polarity")); err == nil {
		ch.inverted = strings.TrimSpace(string(data)) == "inversed"
	}

	return nil
}

// writeFile writes a value to a sysfs file with proper error handling.
func (ch *sysfsChannel) writeFile(name string, value string) error {
	path := filepath.Join(ch.path, name)

	// Small delay to ensure previous writes are complete
	time.Sleep(sysfsWriteDelay)

	if err := os.WriteFile(path, []byte(value), 0644); err != nil {
		return fmt.Errorf("write %s=%s to %s: %w", name, value, path, err)
	}
	return nil
}

// Enable starts PWM signal generation.
func (ch *sysfsChannel) Enable(ctx context.Context) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ch.enabled {
		return nil
	}

	if err := ch.writeFile("enable", "1"); err != nil {
		return err
	}

	ch.enabled = true
	return nil
}

// Disable stops PWM signal generation.
func (ch *sysfsChannel) Disable(ctx context.Context) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	return ch.disableUnsafe()
}

// disableUnsafe disables PWM without locking (for use when lock is already held).
func (ch *sysfsChannel) disableUnsafe() error {
	if !ch.enabled {
		return nil
	}

	if err := ch.writeFile("enable", "0"); err != nil {
		return err
	}

	ch.enabled = false
	return nil
}

// Enabled returns whether PWM output is currently enabled.
func (ch *sysfsChannel) Enabled() bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.enabled
}

// SetPeriod sets the PWM period in nanoseconds.
// Note: Period must be set before duty_cycle, and duty_cycle must be <= period.
func (ch *sysfsChannel) SetPeriod(ctx context.Context, ns uint64) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ns == 0 {
		return ErrInvalidPeriod
	}

	// If new period is smaller than current duty, we must reduce duty first
	// to satisfy the kernel constraint that duty_cycle <= period
	if ns < ch.duty {
		if err := ch.writeFile("duty_cycle", "0"); err != nil {
			return fmt.Errorf("failed to reset duty before period change: %w", err)
		}
		ch.duty = 0
	}

	if err := ch.writeFile("period", strconv.FormatUint(ns, 10)); err != nil {
		return err
	}

	ch.period = ns
	return nil
}

// Period returns the current period in nanoseconds.
func (ch *sysfsChannel) Period() uint64 {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.period
}

// SetDuty sets the duty cycle (high time) in nanoseconds.
func (ch *sysfsChannel) SetDuty(ctx context.Context, ns uint64) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	if ns > ch.period {
		return fmt.Errorf("%w: duty %d ns > period %d ns", ErrInvalidDuty, ns, ch.period)
	}

	if err := ch.writeFile("duty_cycle", strconv.FormatUint(ns, 10)); err != nil {
		return err
	}

	ch.duty = ns
	return nil
}

// Duty returns the current duty in nanoseconds.
func (ch *sysfsChannel) Duty() uint64 {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.duty
}

// SetDutyCycle sets the duty cycle as a fraction (0.0 to 1.0).
func (ch *sysfsChannel) SetDutyCycle(ctx context.Context, duty float64) error {
	if duty < 0.0 || duty > 1.0 {
		return fmt.Errorf("duty cycle must be 0.0-1.0, got %f", duty)
	}

	ch.mu.RLock()
	period := ch.period
	ch.mu.RUnlock()

	if period == 0 {
		return fmt.Errorf("period must be set before duty cycle")
	}

	ns := uint64(float64(period) * duty)
	return ch.SetDuty(ctx, ns)
}

// DutyCycle returns the duty cycle as a fraction (0.0 to 1.0).
func (ch *sysfsChannel) DutyCycle() float64 {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if ch.period == 0 {
		return 0.0
	}
	return float64(ch.duty) / float64(ch.period)
}

// SetFrequency sets the PWM frequency in Hz.
func (ch *sysfsChannel) SetFrequency(ctx context.Context, hz float64) error {
	if hz <= 0 {
		return fmt.Errorf("frequency must be positive, got %f Hz", hz)
	}

	// Convert frequency to period in nanoseconds
	// period_ns = 1e9 / frequency_hz
	periodNs := uint64(1e9 / hz)

	// Preserve duty cycle ratio when changing frequency
	ch.mu.RLock()
	oldPeriod := ch.period
	oldDuty := ch.duty
	ch.mu.RUnlock()

	var newDuty uint64
	if oldPeriod > 0 {
		ratio := float64(oldDuty) / float64(oldPeriod)
		newDuty = uint64(float64(periodNs) * ratio)
	}

	// Set new period
	if err := ch.SetPeriod(ctx, periodNs); err != nil {
		return fmt.Errorf("failed to set period for %f Hz: %w", hz, err)
	}

	// Restore duty cycle ratio
	if newDuty > 0 {
		if err := ch.SetDuty(ctx, newDuty); err != nil {
			return fmt.Errorf("failed to restore duty cycle: %w", err)
		}
	}

	return nil
}

// Frequency returns the current frequency in Hz.
func (ch *sysfsChannel) Frequency() float64 {
	ch.mu.RLock()
	defer ch.mu.RUnlock()

	if ch.period == 0 {
		return 0.0
	}
	return 1e9 / float64(ch.period)
}

// SetPolarity sets the output polarity.
// Note: Polarity can only be changed when the channel is disabled.
func (ch *sysfsChannel) SetPolarity(ctx context.Context, inverted bool) error {
	ch.mu.Lock()
	defer ch.mu.Unlock()

	// Polarity can only be changed when disabled on most systems
	wasEnabled := ch.enabled
	if wasEnabled {
		if err := ch.writeFile("enable", "0"); err != nil {
			return fmt.Errorf("failed to disable for polarity change: %w", err)
		}
		ch.enabled = false
	}

	polarity := "normal"
	if inverted {
		polarity = "inversed"
	}

	if err := ch.writeFile("polarity", polarity); err != nil {
		// Polarity may not be supported on all PWM controllers
		// Re-enable and return without error (polarity is optional)
		if wasEnabled {
			ch.writeFile("enable", "1")
			ch.enabled = true
		}
		// Don't fail if polarity not supported, just ignore
		return nil
	}

	ch.inverted = inverted

	// Re-enable if it was enabled before
	if wasEnabled {
		if err := ch.writeFile("enable", "1"); err != nil {
			return fmt.Errorf("failed to re-enable after polarity change: %w", err)
		}
		ch.enabled = true
	}

	return nil
}

// Polarity returns true if output is inverted.
func (ch *sysfsChannel) Polarity() bool {
	ch.mu.RLock()
	defer ch.mu.RUnlock()
	return ch.inverted
}

// Verify interface compliance
var _ pwm.Channel = (*sysfsChannel)(nil)
