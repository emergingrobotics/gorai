//go:build !linux

package neo7m

import (
	"fmt"

	"github.com/emergingrobotics/gorai/driver/serial"
)

// defaultOpener returns an error on non-Linux platforms where the sysfs/UART
// serial driver is unavailable.
func defaultOpener(cfg serial.Config) (serial.Port, error) {
	return nil, fmt.Errorf("neo7m serial driver is only supported on linux")
}
