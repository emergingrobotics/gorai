// Command gorai is the CLI tool for the Gorai robotics framework.
package main

import (
	"fmt"
	"os"

	"github.com/gorai/gorai/cmd/gorai/commands"
)

func main() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
