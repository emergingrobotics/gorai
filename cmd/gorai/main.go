// Command gorai is the CLI tool for the Gorai robotics framework.
package main

import (
	"github.com/gorai/gorai/pkg/gorai"

	// Remote proxy components (core infrastructure, stay in core)
	_ "github.com/gorai/gorai/components/camera/remote"
	_ "github.com/gorai/gorai/components/gpio/remote"
	_ "github.com/gorai/gorai/components/input/remote"
	_ "github.com/gorai/gorai/components/motor/remote"
	_ "github.com/gorai/gorai/components/pwm/input/remote"
	_ "github.com/gorai/gorai/components/pwm/remote"
	_ "github.com/gorai/gorai/components/sensor/encoder/remote"
)

func main() {
	gorai.Run()
}
