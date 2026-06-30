// Package slam defines the SLAM (Simultaneous Localization and Mapping) service interface.
package slam

import (
	"context"

	"github.com/emergingrobotics/gorai/services"
)

// Service provides SLAM capabilities.
type Service interface {
	service.Service

	// Position returns the current estimated position.
	Position(ctx context.Context) (Pose, error)

	// PointCloudMap returns the current point cloud map.
	PointCloudMap(ctx context.Context) (PointCloud, error)

	// InternalState returns internal SLAM state for saving/debugging.
	InternalState(ctx context.Context) ([]byte, error)

	// Properties returns SLAM service properties.
	Properties(ctx context.Context) (Properties, error)
}

// Pose represents a 6DOF pose.
type Pose struct {
	// Position in meters.
	X, Y, Z float64
	// Orientation as quaternion.
	QX, QY, QZ, QW float64
}

// PointCloud represents a 3D point cloud.
type PointCloud interface {
	// Size returns the number of points.
	Size() int
	// At returns the point at index i.
	At(i int) (x, y, z float64)
}

// Properties describes SLAM service capabilities.
type Properties struct {
	// CloudSlam indicates if this is a cloud-based SLAM service.
	CloudSlam bool
	// MappingMode indicates if new map data is being created.
	MappingMode bool
	// InternalStateType describes the format of internal state data.
	InternalStateType string
	// SensorInfo describes sensors used by SLAM.
	SensorInfo SensorInfo
}

// SensorInfo describes sensors used by SLAM.
type SensorInfo struct {
	// CameraName is the name of the camera component used.
	CameraName string
	// MovementSensorName is the name of the movement sensor used.
	MovementSensorName string
}
