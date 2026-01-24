// Package camera defines the camera component interface.
package camera

import (
	"context"
	"image"

	"github.com/gorai/gorai/components"
)

// Camera represents a camera that can capture images.
type Camera interface {
	component.Component

	// Image returns the current camera image.
	Image(ctx context.Context) (image.Image, error)

	// Stream returns a channel of images for continuous capture.
	Stream(ctx context.Context) (<-chan image.Image, error)

	// Properties returns the camera's properties.
	Properties(ctx context.Context) (Properties, error)
}

// Properties describes camera capabilities.
type Properties struct {
	// Width is the image width in pixels.
	Width int
	// Height is the image height in pixels.
	Height int
	// FrameRate is the capture rate in frames per second.
	FrameRate float64
	// SupportsPTZ indicates pan-tilt-zoom support.
	SupportsPTZ bool
	// SupportsDepth indicates depth sensing support.
	SupportsDepth bool
	// IntrinsicParameters contains camera calibration data.
	IntrinsicParameters *IntrinsicParameters
}

// IntrinsicParameters contains camera calibration parameters.
type IntrinsicParameters struct {
	Width            int
	Height           int
	FocalXPx         float64
	FocalYPx         float64
	CenterXPx        float64
	CenterYPx        float64
	DistortionParams []float64
}

// PointCloud represents a 3D point cloud from a depth camera.
type PointCloud interface {
	// Size returns the number of points.
	Size() int
	// At returns the point at index i as (x, y, z).
	At(i int) (float64, float64, float64)
}
