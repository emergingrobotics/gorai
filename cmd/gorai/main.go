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

	// PWM components
	_ "github.com/gorai/gorai/components/pwm/gpiod"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
