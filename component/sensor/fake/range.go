package fake

import (
	"context"
	"fmt"

	"github.com/gorai/gorai/component/sensor"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("range_sensor", "fake", NewRangeSensor)
}

// RangeSensor is a fake range sensor for testing.
type RangeSensor struct {
	name resource.Name

	// Current reading
	ranges []float64

	// Properties
	minRange    float64
	maxRange    float64
	fieldOfView float64
}

// NewRangeSensor creates a new fake range sensor.
func NewRangeSensor(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "range_sensor", nameStr)

	minRange := 0.02
	if v, ok := conf["min_range"].(float64); ok {
		minRange = v
	}

	maxRange := 4.0
	if v, ok := conf["max_range"].(float64); ok {
		maxRange = v
	}

	return &RangeSensor{
		name:        name,
		ranges:      []float64{1.0}, // Default 1 meter
		minRange:    minRange,
		maxRange:    maxRange,
		fieldOfView: 0.44, // ~25 degrees
	}, nil
}

// NewRangeSensorWithName creates a fake range sensor with a specific resource name.
func NewRangeSensorWithName(name resource.Name) *RangeSensor {
	return &RangeSensor{
		name:        name,
		ranges:      []float64{1.0},
		minRange:    0.02,
		maxRange:    4.0,
		fieldOfView: 0.44,
	}
}

// Name returns the resource name.
func (r *RangeSensor) Name() resource.Name {
	return r.name
}

// Reconfigure updates the configuration.
func (r *RangeSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (r *RangeSensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "set_range":
			if rng, ok := cmd["range"].(float64); ok {
				r.ranges = []float64{rng}
				return map[string]any{"status": "ok"}, nil
			}
		case "get_state":
			return map[string]any{
				"ranges":    r.ranges,
				"min_range": r.minRange,
				"max_range": r.maxRange,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (r *RangeSensor) Close(ctx context.Context) error {
	return nil
}

// Readings returns all sensor readings as a map.
func (r *RangeSensor) Readings(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"range":     r.ranges[0],
		"ranges":    r.ranges,
		"min_range": r.minRange,
		"max_range": r.maxRange,
	}, nil
}

// GetRange returns the measured distance in meters.
func (r *RangeSensor) GetRange(ctx context.Context) (float64, error) {
	if len(r.ranges) == 0 {
		return 0, fmt.Errorf("no range data available")
	}
	return r.ranges[0], nil
}

// GetRanges returns multiple range measurements.
func (r *RangeSensor) GetRanges(ctx context.Context) ([]float64, error) {
	return r.ranges, nil
}

// GetMinRange returns the minimum detectable range.
func (r *RangeSensor) GetMinRange(ctx context.Context) (float64, error) {
	return r.minRange, nil
}

// GetMaxRange returns the maximum detectable range.
func (r *RangeSensor) GetMaxRange(ctx context.Context) (float64, error) {
	return r.maxRange, nil
}

// Properties returns range sensor properties.
func (r *RangeSensor) Properties(ctx context.Context) (sensor.RangeSensorProperties, error) {
	return sensor.RangeSensorProperties{
		MinRange:    r.minRange,
		MaxRange:    r.maxRange,
		FieldOfView: r.fieldOfView,
		NumPoints:   len(r.ranges),
	}, nil
}

// SetRange sets the range value for testing.
func (r *RangeSensor) SetRange(rng float64) {
	r.ranges = []float64{rng}
}

// SetRanges sets multiple range values for testing.
func (r *RangeSensor) SetRanges(ranges []float64) {
	r.ranges = ranges
}

// SetProperties sets the sensor properties for testing.
func (r *RangeSensor) SetProperties(minRange, maxRange, fov float64) {
	r.minRange = minRange
	r.maxRange = maxRange
	r.fieldOfView = fov
}

// Verify interface compliance.
var _ sensor.RangeSensor = (*RangeSensor)(nil)
