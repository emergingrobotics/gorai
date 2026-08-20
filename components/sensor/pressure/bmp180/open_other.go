//go:build !linux

package bmp180

import (
	"fmt"

	"github.com/emergingrobotics/gorai/driver/i2c"
)

// defaultOpener returns an error on non-Linux platforms where the sysfs I2C
// driver is unavailable.
func defaultOpener(bus string, addr uint16) (i2c.Device, error) {
	return nil, fmt.Errorf("bmp180 I2C driver is only supported on linux")
}
