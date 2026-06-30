// Package gripper defines the gripper component interface.
package gripper

import (
	"context"

	"github.com/emergingrobotics/gorai/components"
)

// Gripper represents an end effector gripper.
type Gripper interface {
	component.Actuator

	// Open opens the gripper.
	Open(ctx context.Context) error

	// Grab closes the gripper to grab an object.
	// Returns true if an object was grabbed.
	Grab(ctx context.Context) (bool, error)

	// IsOpen returns true if the gripper is fully open.
	IsOpen(ctx context.Context) (bool, error)
}
