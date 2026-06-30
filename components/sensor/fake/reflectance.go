package fake

import (
	"context"
	"fmt"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("reflectance_sensor", "fake", NewReflectanceSensor)
}

// ReflectanceSensor is a fake reflectance sensor for testing.
type ReflectanceSensor struct {
	name resource.Name

	// Reflectance values (0.0-1.0) per channel
	reflectances []float64

	// Calibration values
	calibrated bool
	minValues  []float64
	maxValues  []float64
}

// NewReflectanceSensor creates a new fake reflectance sensor.
func NewReflectanceSensor(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "reflectance_sensor", nameStr)

	numChannels := 8 // Default QTR-8RC
	if v, ok := conf["num_channels"].(float64); ok {
		numChannels = int(v)
	}

	reflectances := make([]float64, numChannels)
	minValues := make([]float64, numChannels)
	maxValues := make([]float64, numChannels)

	for i := range reflectances {
		reflectances[i] = 0.5 // Default mid-value
		maxValues[i] = 1.0
	}

	return &ReflectanceSensor{
		name:         name,
		reflectances: reflectances,
		minValues:    minValues,
		maxValues:    maxValues,
	}, nil
}

// NewReflectanceSensorWithName creates a fake reflectance sensor with a specific resource name.
func NewReflectanceSensorWithName(name resource.Name, numChannels int) *ReflectanceSensor {
	reflectances := make([]float64, numChannels)
	minValues := make([]float64, numChannels)
	maxValues := make([]float64, numChannels)

	for i := range reflectances {
		reflectances[i] = 0.5
		maxValues[i] = 1.0
	}

	return &ReflectanceSensor{
		name:         name,
		reflectances: reflectances,
		minValues:    minValues,
		maxValues:    maxValues,
	}
}

// Name returns the resource name.
func (r *ReflectanceSensor) Name() resource.Name {
	return r.name
}

// Reconfigure updates the configuration.
func (r *ReflectanceSensor) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (r *ReflectanceSensor) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_state":
			pos, _ := r.GetLinePosition(ctx)
			return map[string]any{
				"reflectances":  r.reflectances,
				"line_position": pos,
				"calibrated":    r.calibrated,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (r *ReflectanceSensor) Close(ctx context.Context) error {
	return nil
}

// Readings returns all sensor readings as a map.
func (r *ReflectanceSensor) Readings(ctx context.Context) (map[string]any, error) {
	pos, _ := r.GetLinePosition(ctx)
	return map[string]any{
		"reflectances":  r.reflectances,
		"line_position": pos,
		"calibrated":    r.calibrated,
	}, nil
}

// GetReflectances returns reflectance values (0.0-1.0) for each channel.
func (r *ReflectanceSensor) GetReflectances(ctx context.Context) ([]float64, error) {
	return r.reflectances, nil
}

// GetLinePosition returns a weighted average position of the line.
func (r *ReflectanceSensor) GetLinePosition(ctx context.Context) (float64, error) {
	if len(r.reflectances) == 0 {
		return 0, fmt.Errorf("no reflectance data")
	}

	// Calculate weighted average position
	// Lower reflectance = darker = line
	// Invert so darker areas have higher weight
	var weightedSum, totalWeight float64
	for i, ref := range r.reflectances {
		// Invert: 1.0 - ref makes dark areas (line) have higher weight
		weight := 1.0 - ref
		weightedSum += float64(i) * weight
		totalWeight += weight
	}

	if totalWeight < 0.001 {
		// No line detected, return center
		return float64(len(r.reflectances)-1) / 2.0, nil
	}

	return weightedSum / totalWeight, nil
}

// Calibrate performs calibration.
func (r *ReflectanceSensor) Calibrate(ctx context.Context) error {
	// In a fake sensor, just mark as calibrated
	r.calibrated = true
	// Store current values as calibration reference
	copy(r.minValues, r.reflectances)
	for i := range r.maxValues {
		r.maxValues[i] = 1.0
	}
	return nil
}

// SetReflectances sets the reflectance values for testing.
func (r *ReflectanceSensor) SetReflectances(values []float64) {
	r.reflectances = values
}

// SetChannel sets a single channel's reflectance for testing.
func (r *ReflectanceSensor) SetChannel(index int, value float64) {
	if index >= 0 && index < len(r.reflectances) {
		r.reflectances[index] = value
	}
}

// SimulateLine simulates a line at a given position for testing.
// position is 0.0 to 1.0 across the sensor array.
// width is the line width as a fraction of the array.
func (r *ReflectanceSensor) SimulateLine(position, width float64) {
	n := len(r.reflectances)
	lineCenter := position * float64(n-1)
	lineWidth := width * float64(n)

	for i := range r.reflectances {
		dist := abs(float64(i) - lineCenter)
		if dist < lineWidth/2 {
			// On the line (dark)
			r.reflectances[i] = 0.1
		} else {
			// Off the line (light)
			r.reflectances[i] = 0.9
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Verify interface compliance.
var _ sensor.ReflectanceSensor = (*ReflectanceSensor)(nil)
