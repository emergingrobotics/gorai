//go:build linux

package neo7m

import (
	"github.com/emergingrobotics/gorai/driver/serial"
	linuxserial "github.com/emergingrobotics/gorai/driver/serial/linux"
)

// defaultOpener opens a real Linux serial port.
func defaultOpener(cfg serial.Config) (serial.Port, error) {
	return linuxserial.Open(cfg)
}
