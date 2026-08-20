//go:build linux

package bmp180

import (
	"github.com/emergingrobotics/gorai/driver/i2c"
	linuxi2c "github.com/emergingrobotics/gorai/driver/i2c/linux"
)

// defaultOpener opens a real Linux I2C device.
func defaultOpener(bus string, addr uint16) (i2c.Device, error) {
	b, err := linuxi2c.Open(bus)
	if err != nil {
		return nil, err
	}
	return b.Device(addr), nil
}
