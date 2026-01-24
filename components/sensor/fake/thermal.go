package fake

import (
	"context"
	"fmt"
	"math"

	"github.com/gorai/gorai/components/sensor"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("thermal_array", "fake", NewThermalArray)
}

// ThermalArray is a fake thermal imaging sensor for testing.
type ThermalArray struct {
	name resource.Name

	// Grid data (rows x cols of temperatures in °C)
	grid [][]float64

	// Properties
	width, height      int
	ambientTemperature float64
}

// NewThermalArray creates a new fake thermal array.
func NewThermalArray(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "thermal_array", nameStr)

	width, height := 8, 8 // Default AMG8833 resolution
	if w, ok := conf["width"].(float64); ok {
		width = int(w)
	}
	if h, ok := conf["height"].(float64); ok {
		height = int(h)
	}

	t := &ThermalArray{
		name:               name,
		width:              width,
		height:             height,
		ambientTemperature: 25.0,
	}
	t.initGrid()

	return t, nil
}

// NewThermalArrayWithName creates a fake thermal array with a specific resource name.
func NewThermalArrayWithName(name resource.Name, width, height int) *ThermalArray {
	t := &ThermalArray{
		name:               name,
		width:              width,
		height:             height,
		ambientTemperature: 25.0,
	}
	t.initGrid()
	return t
}

// initGrid initializes the temperature grid with ambient temperature.
func (t *ThermalArray) initGrid() {
	t.grid = make([][]float64, t.height)
	for i := range t.grid {
		t.grid[i] = make([]float64, t.width)
		for j := range t.grid[i] {
			t.grid[i][j] = t.ambientTemperature
		}
	}
}

// Name returns the resource name.
func (t *ThermalArray) Name() resource.Name {
	return t.name
}

// Reconfigure updates the configuration.
func (t *ThermalArray) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (t *ThermalArray) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_state":
			min, max, _ := t.GetMinMaxTemperature(ctx)
			return map[string]any{
				"width":    t.width,
				"height":   t.height,
				"ambient":  t.ambientTemperature,
				"min_temp": min,
				"max_temp": max,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (t *ThermalArray) Close(ctx context.Context) error {
	return nil
}

// Readings returns all sensor readings as a map.
func (t *ThermalArray) Readings(ctx context.Context) (map[string]any, error) {
	min, max, _ := t.GetMinMaxTemperature(ctx)
	return map[string]any{
		"grid":     t.grid,
		"ambient":  t.ambientTemperature,
		"min_temp": min,
		"max_temp": max,
		"width":    t.width,
		"height":   t.height,
	}, nil
}

// GetTemperatureGrid returns the temperature grid in °C.
func (t *ThermalArray) GetTemperatureGrid(ctx context.Context) ([][]float64, error) {
	return t.grid, nil
}

// GetAmbientTemperature returns the ambient temperature in °C.
func (t *ThermalArray) GetAmbientTemperature(ctx context.Context) (float64, error) {
	return t.ambientTemperature, nil
}

// GetMinMaxTemperature returns the min and max temperatures.
func (t *ThermalArray) GetMinMaxTemperature(ctx context.Context) (min, max float64, err error) {
	min = math.MaxFloat64
	max = -math.MaxFloat64

	for _, row := range t.grid {
		for _, temp := range row {
			if temp < min {
				min = temp
			}
			if temp > max {
				max = temp
			}
		}
	}

	return min, max, nil
}

// GetResolution returns the grid dimensions.
func (t *ThermalArray) GetResolution(ctx context.Context) (width, height int, err error) {
	return t.width, t.height, nil
}

// SetGrid sets the temperature grid for testing.
func (t *ThermalArray) SetGrid(grid [][]float64) {
	t.grid = grid
	t.height = len(grid)
	if t.height > 0 {
		t.width = len(grid[0])
	}
}

// SetPixel sets a single pixel temperature for testing.
func (t *ThermalArray) SetPixel(row, col int, temp float64) {
	if row >= 0 && row < t.height && col >= 0 && col < t.width {
		t.grid[row][col] = temp
	}
}

// SetAmbientTemperature sets the ambient temperature for testing.
func (t *ThermalArray) SetAmbientTemperature(temp float64) {
	t.ambientTemperature = temp
}

// Verify interface compliance.
var _ sensor.ThermalArray = (*ThermalArray)(nil)
