package behavior

import (
	"context"
	"time"

	"github.com/gorai/gorai/pkg/resource"
)

// DerivedSensor is a virtual sensor exposed by a behavior.
// It provides computed, fused, or inferred data as a byproduct of
// the behavior's operation.
//
// Examples:
//   - Localization behavior exposes estimated_pose sensor
//   - Object tracking behavior exposes tracked_objects sensor
//   - Person following behavior exposes target_person sensor
type DerivedSensor interface {
	resource.Resource
	resource.Sensor

	// SourceBehavior returns the behavior that produces this sensor.
	SourceBehavior() resource.Name

	// DataType returns the type of data this sensor provides.
	DataType() string

	// UpdateRate returns how often this sensor updates.
	// Returns 0 for event-driven sensors.
	UpdateRate() time.Duration

	// LastUpdate returns when the sensor was last updated.
	LastUpdate(ctx context.Context) (time.Time, error)
}

// DerivedSensorConfig configures a derived sensor.
type DerivedSensorConfig struct {
	// Name is the sensor name.
	Name string

	// Type is the data type (e.g., "pose", "detections", "velocity").
	Type string

	// Subject is the NATS subject to publish on.
	Subject string

	// UpdateRateHz is how often to publish (0 = event-driven).
	UpdateRateHz float64

	// Description describes what data this sensor provides.
	Description string
}

// Common derived sensor data types.
const (
	// DerivedTypeEstimatedPose is fused position data.
	DerivedTypeEstimatedPose = "estimated_pose"

	// DerivedTypeTrackedObjects is object tracking data.
	DerivedTypeTrackedObjects = "tracked_objects"

	// DerivedTypeTargetPerson is person tracking data.
	DerivedTypeTargetPerson = "target_person"

	// DerivedTypeAnomalies is anomaly detection data.
	DerivedTypeAnomalies = "anomalies"

	// DerivedTypeSceneContext is scene understanding data.
	DerivedTypeSceneContext = "scene_context"

	// DerivedTypeIntent is recognized intent data.
	DerivedTypeIntent = "intent"

	// DerivedTypePlan is current plan data.
	DerivedTypePlan = "plan"

	// DerivedTypeReasoningTrace is LLM reasoning data.
	DerivedTypeReasoningTrace = "reasoning_trace"

	// DerivedTypeMapUpdate is incremental map data.
	DerivedTypeMapUpdate = "map_update"

	// DerivedTypeNavigationStatus is navigation progress data.
	DerivedTypeNavigationStatus = "navigation_status"
)

// TrackedObject represents a tracked object from an object tracking behavior.
type TrackedObject struct {
	// ID is the unique tracking ID.
	ID string

	// Class is the object class label.
	Class string

	// Confidence is the detection confidence (0-1).
	Confidence float64

	// BoundingBox is the 2D bounding box [x, y, width, height].
	BoundingBox [4]float64

	// Position3D is the 3D position in meters (if available).
	Position3D *[3]float64

	// Velocity is the velocity vector (if available).
	Velocity *[3]float64

	// Age is how long this object has been tracked.
	Age time.Duration

	// LastSeen is when the object was last detected.
	LastSeen time.Time
}

// EstimatedPose represents a fused pose estimate.
type EstimatedPose struct {
	// Position in meters (x, y, z).
	Position [3]float64

	// Orientation as quaternion (w, x, y, z).
	Orientation [4]float64

	// Covariance is the 6x6 pose covariance (if available).
	Covariance []float64

	// FrameID is the reference frame.
	FrameID string

	// Timestamp is when this estimate was made.
	Timestamp time.Time

	// Sources lists the sensor sources used.
	Sources []string
}

// Anomaly represents a detected anomaly.
type Anomaly struct {
	// ID is the unique anomaly ID.
	ID string

	// Type is the anomaly type.
	Type string

	// Description describes the anomaly.
	Description string

	// Confidence is the detection confidence (0-1).
	Confidence float64

	// Location is the spatial location (if applicable).
	Location *[3]float64

	// DetectedAt is when the anomaly was detected.
	DetectedAt time.Time

	// Severity indicates importance (0 = low, 1 = high).
	Severity float64
}

// SceneContext represents understanding of the current scene.
type SceneContext struct {
	// Description is a natural language description.
	Description string

	// Labels lists semantic labels present.
	Labels []string

	// Objects lists detected objects with positions.
	Objects []TrackedObject

	// Relationships describes object relationships.
	Relationships []string

	// Timestamp is when this context was generated.
	Timestamp time.Time
}
