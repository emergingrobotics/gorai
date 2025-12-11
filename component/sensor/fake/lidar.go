package fake

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/gorai/gorai/component/sensor"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("lidar", "fake", NewLiDAR)
}

// LiDAR is a fake LiDAR sensor for testing.
type LiDAR struct {
	name resource.Name

	// Current scan data
	scan       *sensor.LaserScan
	pointCloud *sensor.PointCloud

	// Properties
	minRange          float64
	maxRange          float64
	angularResolution float64
	sampleRate        int
	is3D              bool
	scanMode          string
	scanRate          float64
}

// NewLiDAR creates a new fake LiDAR.
func NewLiDAR(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "lidar", nameStr)

	l := &LiDAR{
		name:              name,
		minRange:          0.15,
		maxRange:          12.0,
		angularResolution: 1.0,
		sampleRate:        8000,
		is3D:              false,
		scanMode:          "standard",
		scanRate:          10.0,
	}

	// Generate default scan data
	l.generateDefaultScan()

	return l, nil
}

// NewLiDARWithName creates a fake LiDAR with a specific resource name.
func NewLiDARWithName(name resource.Name) *LiDAR {
	l := &LiDAR{
		name:              name,
		minRange:          0.15,
		maxRange:          12.0,
		angularResolution: 1.0,
		sampleRate:        8000,
		is3D:              false,
		scanMode:          "standard",
		scanRate:          10.0,
	}
	l.generateDefaultScan()
	return l
}

// generateDefaultScan creates a default 360-degree scan.
func (l *LiDAR) generateDefaultScan() {
	numPoints := int(360.0 / l.angularResolution)
	ranges := make([]float64, numPoints)
	intensities := make([]float64, numPoints)

	for i := range ranges {
		ranges[i] = 5.0 // Default 5 meters
		intensities[i] = 100.0
	}

	l.scan = &sensor.LaserScan{
		AngleMin:       0,
		AngleMax:       2 * math.Pi,
		AngleIncrement: l.angularResolution * math.Pi / 180.0,
		Ranges:         ranges,
		Intensities:    intensities,
		Timestamp:      time.Now().UnixNano(),
	}
}

// Name returns the resource name.
func (l *LiDAR) Name() resource.Name {
	return l.name
}

// Reconfigure updates the configuration.
func (l *LiDAR) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}

// DoCommand executes arbitrary commands.
func (l *LiDAR) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_state":
			return map[string]any{
				"scan_mode":  l.scanMode,
				"scan_rate":  l.scanRate,
				"num_points": len(l.scan.Ranges),
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (l *LiDAR) Close(ctx context.Context) error {
	return nil
}

// Readings returns all sensor readings as a map.
func (l *LiDAR) Readings(ctx context.Context) (map[string]any, error) {
	return map[string]any{
		"scan_mode":  l.scanMode,
		"scan_rate":  l.scanRate,
		"num_points": len(l.scan.Ranges),
		"min_range":  l.minRange,
		"max_range":  l.maxRange,
	}, nil
}

// GetScan returns a 2D laser scan.
func (l *LiDAR) GetScan(ctx context.Context) (*sensor.LaserScan, error) {
	// Update timestamp
	l.scan.Timestamp = time.Now().UnixNano()
	return l.scan, nil
}

// GetPointCloud returns a 3D point cloud.
func (l *LiDAR) GetPointCloud(ctx context.Context) (*sensor.PointCloud, error) {
	if !l.is3D {
		return nil, fmt.Errorf("3D point cloud not supported on 2D LiDAR")
	}
	return l.pointCloud, nil
}

// GetScanRate returns the current scan rate in Hz.
func (l *LiDAR) GetScanRate(ctx context.Context) (float64, error) {
	return l.scanRate, nil
}

// SetScanMode sets the scanning mode.
func (l *LiDAR) SetScanMode(ctx context.Context, mode string) error {
	l.scanMode = mode
	return nil
}

// GetProperties returns LiDAR properties.
func (l *LiDAR) GetProperties(ctx context.Context) (sensor.LiDARProperties, error) {
	return sensor.LiDARProperties{
		MinRange:          l.minRange,
		MaxRange:          l.maxRange,
		AngularResolution: l.angularResolution,
		SampleRate:        l.sampleRate,
		Is3D:              l.is3D,
	}, nil
}

// SetScan sets the scan data for testing.
func (l *LiDAR) SetScan(scan *sensor.LaserScan) {
	l.scan = scan
}

// SetPointCloud sets the point cloud for testing.
func (l *LiDAR) SetPointCloud(cloud *sensor.PointCloud) {
	l.pointCloud = cloud
	l.is3D = true
}

// SetProperties sets the LiDAR properties for testing.
func (l *LiDAR) SetProperties(minRange, maxRange, angRes float64, sampleRate int, is3D bool) {
	l.minRange = minRange
	l.maxRange = maxRange
	l.angularResolution = angRes
	l.sampleRate = sampleRate
	l.is3D = is3D
}

// Verify interface compliance.
var _ sensor.LiDAR = (*LiDAR)(nil)
