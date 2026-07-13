//go:build linux

package linux

import (
	driverpwm "github.com/emergingrobotics/gorai/driver/pwm"
	linuxpwm "github.com/emergingrobotics/gorai/driver/pwm/linux"
)

// defaultOpener opens a real sysfs PWM channel.
func defaultOpener(chip, channel int) (driverpwm.Channel, error) {
	c, err := linuxpwm.OpenChip(chip)
	if err != nil {
		return nil, err
	}
	return c.Channel(channel)
}
