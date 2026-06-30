//go:build linux

package linux

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/emergingrobotics/gorai/driver/pwm"
)

const (
	// Timing constants for sysfs operations.
	exportRetryDelay = 10 * time.Millisecond
	exportMaxRetries = 10
)

// sysfsChip implements pwm.Chip for Linux sysfs PWM.
type sysfsChip struct {
	num      int
	path     string
	npwm     int
	mu       sync.Mutex
	channels map[int]*sysfsChannel
	closed   bool
}

// Name implements driver.Driver.
func (c *sysfsChip) Name() string {
	return fmt.Sprintf("pwm:sysfs:chip%d", c.num)
}

// Open implements driver.Driver.
// The chip is already opened by OpenChip(), so this is a no-op.
func (c *sysfsChip) Open(ctx context.Context) error {
	return nil
}

// Channels returns the number of PWM channels on this chip.
func (c *sysfsChip) Channels() int {
	return c.npwm
}

// Channel returns a PWM channel, exporting it if necessary.
func (c *sysfsChip) Channel(n int) (pwm.Channel, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil, ErrChipClosed
	}

	if n < 0 || n >= c.npwm {
		return nil, fmt.Errorf("%w: channel %d (chip has %d channels)",
			ErrChannelNotFound, n, c.npwm)
	}

	// Return cached channel if already exported
	if ch, ok := c.channels[n]; ok {
		return ch, nil
	}

	// Export the channel
	ch, err := c.exportChannel(n)
	if err != nil {
		return nil, err
	}

	c.channels[n] = ch
	return ch, nil
}

// exportChannel exports a PWM channel via sysfs.
func (c *sysfsChip) exportChannel(n int) (*sysfsChannel, error) {
	channelPath := filepath.Join(c.path, fmt.Sprintf("pwm%d", n))

	// Check if already exported
	if _, err := os.Stat(channelPath); err == nil {
		// Already exported, try to use it
		return newSysfsChannel(c, n, channelPath)
	}

	// Export the channel
	exportPath := filepath.Join(c.path, "export")
	if err := os.WriteFile(exportPath, []byte(fmt.Sprintf("%d", n)), 0644); err != nil {
		// Check if it's because it's already exported (race condition)
		if _, statErr := os.Stat(channelPath); statErr == nil {
			return newSysfsChannel(c, n, channelPath)
		}
		return nil, fmt.Errorf("%w: channel %d: %v", ErrExportFailed, n, err)
	}

	// Wait for sysfs to create the channel directory
	for i := 0; i < exportMaxRetries; i++ {
		if _, err := os.Stat(channelPath); err == nil {
			break
		}
		time.Sleep(exportRetryDelay)
	}

	// Verify export succeeded
	if _, err := os.Stat(channelPath); err != nil {
		return nil, fmt.Errorf("%w: channel %d directory not created at %s",
			ErrExportFailed, n, channelPath)
	}

	return newSysfsChannel(c, n, channelPath)
}

// unexportChannel unexports a PWM channel.
func (c *sysfsChip) unexportChannel(n int) error {
	unexportPath := filepath.Join(c.path, "unexport")
	if err := os.WriteFile(unexportPath, []byte(fmt.Sprintf("%d", n)), 0644); err != nil {
		// Ignore errors if channel wasn't exported
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to unexport channel %d: %w", n, err)
	}
	return nil
}

// Close releases all PWM channels and closes the chip.
func (c *sysfsChip) Close(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return nil
	}
	c.closed = true

	var errs []error

	// Disable and unexport all channels
	for n, ch := range c.channels {
		// Disable first to stop PWM output
		if ch.enabled {
			if err := ch.disableUnsafe(); err != nil {
				errs = append(errs, fmt.Errorf("disable channel %d: %w", n, err))
			}
		}

		// Unexport the channel
		if err := c.unexportChannel(n); err != nil {
			errs = append(errs, fmt.Errorf("unexport channel %d: %w", n, err))
		}
	}

	c.channels = nil

	if len(errs) > 0 {
		return fmt.Errorf("errors closing PWM chip %d: %v", c.num, errs)
	}
	return nil
}

// Verify interface compliance
var _ pwm.Chip = (*sysfsChip)(nil)
