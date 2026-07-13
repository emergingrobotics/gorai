// Command gorai is the CLI tool for the Gorai robotics framework.
package main

import (
	"github.com/emergingrobotics/gorai/pkg/gorai"

	// Remote proxy components (core infrastructure, stay in core)
	_ "github.com/emergingrobotics/gorai/components/camera/remote"
	_ "github.com/emergingrobotics/gorai/components/gpio/remote"
	_ "github.com/emergingrobotics/gorai/components/input/remote"
	_ "github.com/emergingrobotics/gorai/components/motor/remote"
	_ "github.com/emergingrobotics/gorai/components/pwm/input/remote"
	_ "github.com/emergingrobotics/gorai/components/pwm/linux"
	_ "github.com/emergingrobotics/gorai/components/pwm/ncp"
	_ "github.com/emergingrobotics/gorai/components/pwm/remote"
	_ "github.com/emergingrobotics/gorai/components/sensor/encoder/remote"
)

func main() {
	gorai.Run()
}
