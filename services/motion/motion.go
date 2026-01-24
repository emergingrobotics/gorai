// Package motion defines the motion planning service interface.
package motion

import (
	"context"

	"github.com/gorai/gorai/services"
)

// Service provides motion planning capabilities.
type Service interface {
	service.Service

	// Move moves a component to the specified pose.
	Move(ctx context.Context, componentName string, destination Pose, constraints *Constraints) (bool, error)

	// GetPose gets the current pose of a component in a reference frame.
	GetPose(ctx context.Context, componentName, referenceFrame string) (Pose, error)

	// StopPlan stops the currently executing motion plan.
	StopPlan(ctx context.Context) error

	// ListPlanStatuses lists the statuses of all plans.
	ListPlanStatuses(ctx context.Context) ([]PlanStatus, error)

	// GetPlan gets details of a specific plan.
	GetPlan(ctx context.Context, componentName string, lastPlanOnly bool) (Plan, error)
}

// Pose represents a 6DOF pose in a reference frame.
type Pose struct {
	// Reference frame for this pose.
	ReferenceFrame string
	// Position in meters.
	X, Y, Z float64
	// Orientation as quaternion.
	QX, QY, QZ, QW float64
}

// Constraints defines motion planning constraints.
type Constraints struct {
	// LinearTolerance is acceptable position error in meters.
	LinearTolerance float64
	// AngularTolerance is acceptable orientation error in radians.
	AngularTolerance float64
	// Obstacles lists obstacles to avoid.
	Obstacles []Obstacle
}

// Obstacle represents an obstacle to avoid during motion.
type Obstacle struct {
	// Name of the obstacle.
	Name string
	// Geometries defining the obstacle shape.
	Geometries []Geometry
}

// Geometry represents a geometric shape.
type Geometry struct {
	// Type of geometry (box, sphere, capsule).
	Type string
	// Pose of the geometry center.
	Pose Pose
	// Dimensions (interpretation depends on Type).
	Dimensions []float64
}

// PlanStatus represents the status of a motion plan.
type PlanStatus struct {
	// ComponentName is the component being moved.
	ComponentName string
	// State of the plan.
	State PlanState
	// Reason for the current state.
	Reason string
}

// PlanState represents motion plan states.
type PlanState int

const (
	PlanStateUnspecified PlanState = iota
	PlanStateInProgress
	PlanStateStopped
	PlanStateSucceeded
	PlanStateFailed
)

// Plan represents a motion plan.
type Plan struct {
	// ID uniquely identifies this plan.
	ID string
	// ComponentName is the component being moved.
	ComponentName string
	// Steps in the plan.
	Steps []PlanStep
	// Status of the plan.
	Status PlanStatus
}

// PlanStep represents a single step in a motion plan.
type PlanStep struct {
	// Pose at this step.
	Pose Pose
	// Configuration at this step (joint positions, etc.).
	Configuration map[string]float64
}
