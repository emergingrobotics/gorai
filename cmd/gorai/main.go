// Command gorai is the CLI tool for the Gorai robotics framework.
package main

import (
	"fmt"
	"os"

	"github.com/gorai/gorai/cmd/gorai/commands"

	// Import component packages to register them
	_ "github.com/gorai/gorai/components/serial"

	// Input components
	_ "github.com/gorai/gorai/components/input/keyboard"
	_ "github.com/gorai/gorai/components/input/remote"

	// Camera components
	_ "github.com/gorai/gorai/components/camera/remote"

	// PWM components
	_ "github.com/gorai/gorai/components/pwm/gpiod"
	_ "github.com/gorai/gorai/components/pwm/input/remote"
	_ "github.com/gorai/gorai/components/pwm/remote"

	// GPIO components
	_ "github.com/gorai/gorai/components/gpio/remote"

	// Motor components
	_ "github.com/gorai/gorai/components/motor/remote"

	// Encoder components
	_ "github.com/gorai/gorai/components/sensor/encoder/remote"

	// Bridge services
	_ "github.com/gorai/gorai/services/bridge/keyboard_publisher"

	// Control services
	_ "github.com/gorai/gorai/services/control/keypress_motor"
	_ "github.com/gorai/gorai/services/control/l298n"
	_ "github.com/gorai/gorai/services/control/mecanum"
	_ "github.com/gorai/gorai/services/control/velocity_input"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
