//go:build !linux

package linux

import (
	"fmt"
	"runtime"

	driverpwm "github.com/emergingrobotics/gorai/driver/pwm"
)

// defaultOpener reports that native sysfs PWM is only available on Linux.
func defaultOpener(chip, channel int) (driverpwm.Channel, error) {
	return nil, fmt.Errorf("native sysfs PWM is only supported on linux (running on %s)", runtime.GOOS)
}
