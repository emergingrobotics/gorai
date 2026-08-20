// Package drive defines the differential (tank) drive actuator interface.
//
// A Drive is commanded by a normalized motion intent - surge (forward/back)
// and yaw (turn) - and owns the mixing to individual actuators (e.g. two
// thrusters). This keeps callers (teleop UI, autonomy behaviors) decoupled
// from the actuator layout and lets the implementation regulate timing,
// slew-limiting, and failsafe behavior.
package drive

import (
	"context"

	component "github.com/emergingrobotics/gorai/components"
)

// Drive is a normalized surge/yaw motion actuator.
type Drive interface {
	component.Component

	// SetIntent sets the normalized motion intent. surge and yaw are clamped
	// to [-1, 1]. surge > 0 is forward; yaw > 0 turns to starboard (right).
	SetIntent(ctx context.Context, surge, yaw float64) error

	// Stop commands zero motion (neutral) immediately.
	Stop(ctx context.Context) error

	// Arm runs the arming sequence for the underlying actuators.
	Arm(ctx context.Context) error

	// State returns the latest drive state.
	State(ctx context.Context) (State, error)
}

// State is a snapshot of drive intent and the mixed actuator outputs.
type State struct {
	// Surge and Yaw are the last commanded normalized intent (-1..1).
	Surge float64 `json:"surge"`
	Yaw   float64 `json:"yaw"`

	// Left and Right are the mixed, slew-limited actuator outputs (-1..1).
	Left  float64 `json:"left"`
	Right float64 `json:"right"`

	// Armed reports whether the drive has been armed.
	Armed bool `json:"armed"`

	// Active is false when the drive is holding neutral due to the
	// command-timeout failsafe.
	Active bool `json:"active"`
}
