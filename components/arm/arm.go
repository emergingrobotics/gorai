// Package arm defines the robotic arm component interface.
package arm

import (
	"context"

	"github.com/gorai/gorai/components"
)

// Arm represents a robotic arm.
type Arm interface {
	component.Actuator

	// EndPosition returns the current end effector pose.
	EndPosition(ctx context.Context) (Pose, error)

	// MoveToPosition moves the end effector to the given pose.
	MoveToPosition(ctx context.Context, pose Pose, speed float64) error

	// MoveToJointPositions moves to the specified joint positions.
	MoveToJointPositions(ctx context.Context, positions []float64, speed float64) error

	// JointPositions returns the current joint positions in radians.
	JointPositions(ctx context.Context) ([]float64, error)

	// ModelFrame returns the kinematic model of the arm.
	ModelFrame(ctx context.Context) (ModelFrame, error)
}

// Pose represents a 6DOF pose (position + orientation).
type Pose struct {
	X, Y, Z    float64 // Position in meters
	OX, OY, OZ float64 // Orientation as axis-angle (axis * angle)
	Theta      float64 // Angle in radians (if using axis-angle)
}

// ModelFrame describes the kinematic structure of the arm.
type ModelFrame struct {
	// Name of the model.
	Name string
	// DOF is the degrees of freedom.
	DOF int
	// JointNames are the names of each joint.
	JointNames []string
	// JointLimits are the min/max for each joint.
	JointLimits []JointLimit
}

// JointLimit defines the range of motion for a joint.
type JointLimit struct {
	Min, Max float64 // radians for revolute, meters for prismatic
}
