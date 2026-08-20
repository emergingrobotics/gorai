//go:build !linux

package bno085

import (
	"fmt"

	"github.com/emergingrobotics/gorai/driver/i2c"
)

// defaultOpener returns an error on non-Linux platforms where the sysfs I2C
// driver is unavailable.
func defaultOpener(bus string, addr uint16) (i2c.Device, error) {
	return nil, fmt.Errorf("bno085 I2C driver is only supported on linux")
}
