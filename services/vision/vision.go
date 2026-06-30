// Package vision defines the vision service interface.
package vision

import (
	"context"
	"image"

	"github.com/emergingrobotics/gorai/services"
)

// Service provides computer vision capabilities.
type Service interface {
	service.Service

	// Detections returns object detections in the image.
	Detections(ctx context.Context, img image.Image, extra map[string]any) ([]Detection, error)

	// Classifications returns classification results for the image.
	Classifications(ctx context.Context, img image.Image, n int, extra map[string]any) ([]Classification, error)

	// GetObjectPointClouds returns 3D bounding boxes for detected objects.
	GetObjectPointClouds(ctx context.Context, cameraName string, extra map[string]any) ([]Object3D, error)

	// Properties returns the vision service properties.
	Properties(ctx context.Context) (Properties, error)
}

// Detection represents a detected object.
type Detection struct {
	// Bounding box in image coordinates.
	BoundingBox image.Rectangle
	// Class label.
	Label string
	// Confidence score from 0.0 to 1.0.
	Confidence float64
}

// Classification represents a classification result.
type Classification struct {
	// Class label.
	Label string
	// Confidence score from 0.0 to 1.0.
	Confidence float64
}

// Object3D represents a detected object with 3D information.
type Object3D struct {
	// Label for the object.
	Label string
	// Confidence score.
	Confidence float64
	// Center position in 3D space (meters).
	CenterX, CenterY, CenterZ float64
	// Dimensions in meters.
	Width, Height, Depth float64
}

// Properties describes vision service capabilities.
type Properties struct {
	// ClassificationSupported indicates if classification is available.
	ClassificationSupported bool
	// DetectionSupported indicates if detection is available.
	DetectionSupported bool
	// Object3DSupported indicates if 3D detection is available.
	Object3DSupported bool
	// Labels lists the available class labels.
	Labels []string
}
