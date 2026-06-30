// Package gorai provides the main entrypoint for GoRAI robot projects.
package gorai

import (
	"fmt"
	"os"

	"github.com/emergingrobotics/gorai/cmd/gorai/commands"
)

// Run is the main entrypoint for GoRAI robot projects.
// Call this from your main.go after blank-importing component packages.
func Run() {
	if err := commands.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
