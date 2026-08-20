package neo7m

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/emergingrobotics/gorai/driver/serial"
	"github.com/emergingrobotics/gorai/pkg/registry"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("gps", "neo7m", New)
}

// portOpener opens a serial port for the given config. Overridable in tests.
type portOpener func(cfg serial.Config) (serial.Port, error)

// GPS is a NEO-7M GPS driver reading NMEA over a serial port.
type GPS struct {
	name   resource.Name
	config Config
	logger *slog.Logger
	opener portOpener

	port serial.Port

	mu  sync.RWMutex
	fix gpsFix

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// New creates a NEO-7M GPS component.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)

	cfg, err := ParseConfig(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	logger := slog.Default()
	if loggerRes, err := deps.Get("logger"); err == nil {
		if l, ok := loggerRes.(*slog.Logger); ok {
			logger = l
		}
	}

	g := &GPS{
		name:   resource.NewComponentName("gorai", "gps", nameStr),
		config: cfg,
		logger: logger.With("component", "gps/neo7m", "name", nameStr),
		opener: defaultOpener,
	}
	return g, nil
}

// Start opens the serial port and begins the NMEA read loop.
func (g *GPS) Start(ctx context.Context) error {
	port, err := g.opener(g.config.serialConfig())
	if err != nil {
		return fmt.Errorf("failed to open GPS serial port: %w", err)
	}
	g.port = port

	loopCtx, cancel := context.WithCancel(ctx)
	g.cancel = cancel

	g.wg.Add(1)
	go g.readLoop(loopCtx)

	g.logger.Info("NEO-7M GPS started", "path", g.config.Path, "baud", g.config.BaudRate)
	return nil
}

// readLoop reads NMEA lines and updates the fix until the context is cancelled.
func (g *GPS) readLoop(ctx context.Context) {
	defer g.wg.Done()

	scanner := bufio.NewScanner(g.port)
	scanner.Buffer(make([]byte, 0, 1024), 4096)

	for {
		if ctx.Err() != nil {
			return
		}
		if !scanner.Scan() {
			// Read error or timeout; retry unless cancelled.
			if ctx.Err() != nil {
				return
			}
			if err := scanner.Err(); err != nil {
				g.logger.Debug("GPS read error", "error", err)
			}
			// Reset the scanner to keep reading after a transient error.
			scanner = bufio.NewScanner(g.port)
			scanner.Buffer(make([]byte, 0, 1024), 4096)
			continue
		}
		line := scanner.Text()
		g.mu.Lock()
		applySentence(&g.fix, line)
		g.mu.Unlock()
	}
}

// snapshot returns a copy of the current fix.
func (g *GPS) snapshot() gpsFix {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.fix
}

// Name returns the resource name.
func (g *GPS) Name() resource.Name {
	return g.name
}

// Reconfigure is not supported; restart the component instead.
func (g *GPS) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return fmt.Errorf("reconfiguration not supported, restart component instead")
}

// DoCommand exposes a state query.
func (g *GPS) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if name, _ := cmd["command"].(string); name == "get_state" {
		return g.Readings(ctx)
	}
	return nil, fmt.Errorf("unknown command: %v", cmd["command"])
}

// Close stops the read loop and closes the serial port.
func (g *GPS) Close(ctx context.Context) error {
	if g.cancel != nil {
		g.cancel()
	}
	if g.port != nil {
		_ = g.port.Close(ctx)
	}
	done := make(chan struct{})
	go func() { g.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-ctx.Done():
	}
	return nil
}

// Readings returns the latest GPS readings.
func (g *GPS) Readings(ctx context.Context) (map[string]any, error) {
	f := g.snapshot()
	return map[string]any{
		"latitude":        f.lat,
		"longitude":       f.lon,
		"altitude":        f.altM,
		"altitude_m":      f.altM,
		"hdop":            f.hdop,
		"fix":             int(fixType(f.quality)),
		"fix_quality":     f.quality,
		"satellites":      f.satellites,
		"satellites_used": f.satellites,
		"heading":         f.headingDeg,
		"speed_mps":       f.speedMps,
		"has_fix":         f.hasFix,
	}, nil
}

// Position returns latitude, longitude (degrees) and altitude (meters).
func (g *GPS) Position(ctx context.Context) (lat, lng, alt float64, err error) {
	f := g.snapshot()
	return f.lat, f.lon, f.altM, nil
}

// LinearVelocity returns velocity in m/s (ENU) derived from course + speed.
func (g *GPS) LinearVelocity(ctx context.Context) (x, y, z float64, err error) {
	f := g.snapshot()
	// Course is clockwise from north; convert to east/north components.
	rad := f.headingDeg * (math.Pi / 180.0)
	east := f.speedMps * math.Sin(rad)
	north := f.speedMps * math.Cos(rad)
	return east, north, 0, nil
}

// Accuracy estimates horizontal/vertical accuracy from HDOP using a nominal
// user-equivalent range error.
func (g *GPS) Accuracy(ctx context.Context) (horizontal, vertical float64, err error) {
	f := g.snapshot()
	const uere = 5.0 // meters, nominal
	if f.hdop <= 0 {
		return 0, 0, nil
	}
	return f.hdop * uere, f.hdop * uere * 1.5, nil
}

// Fix returns the current fix type.
func (g *GPS) Fix(ctx context.Context) (sensor.FixType, error) {
	return fixType(g.snapshot().quality), nil
}

// GetHeading returns the course over ground in degrees (0-360).
func (g *GPS) GetHeading(ctx context.Context) (float64, error) {
	return g.snapshot().headingDeg, nil
}

// GetSatellitesUsed returns the number of satellites used in the fix.
func (g *GPS) GetSatellitesUsed(ctx context.Context) (int, error) {
	return g.snapshot().satellites, nil
}

// fixType maps an NMEA GGA fix quality to a sensor.FixType.
func fixType(quality int) sensor.FixType {
	switch quality {
	case 1:
		return sensor.FixGPS
	case 2:
		return sensor.FixDGPS
	case 4, 5:
		return sensor.FixRTK
	default:
		return sensor.FixNone
	}
}

// Verify interface compliance.
var _ sensor.GPS = (*GPS)(nil)
