//go:build !linux && !darwin

package reader

import "fmt"

// newLinuxReader is a stub for unsupported platforms.
func newLinuxReader() (Reader, error) {
	return nil, fmt.Errorf("linux reader not available on this platform")
}

// newDarwinReader is a stub for unsupported platforms.
func newDarwinReader() (Reader, error) {
	return nil, fmt.Errorf("darwin reader not available on this platform")
}
