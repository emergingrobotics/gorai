//go:build gorai_gps

// Package serial provides serial port based components.
package serial

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/gorai/gorai-gps/pkg/gps/publisher"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("serial", "gps", NewGPS)
}

// GPS is a component that publishes GPS NMEA data to NATS using gorai-gps.
type GPS struct {
	name      resource.Name
	config    *GPSConfig
	publisher *publisher.Publisher
	cancel    context.CancelFunc
	wg        sync.WaitGroup

	mu      sync.RWMutex
	running bool
}

// GPSConfig holds GPS component configuration.
type GPSConfig struct {
	Device    string
	BaudRate  int
	NATSURL   string
	Namespace string
	RobotName string
}

// NewGPS creates a new GPS component.
func NewGPS(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	// Extract name
	nameStr, _ := conf["name"].(string)
	if nameStr == "" {
		return nil, fmt.Errorf("GPS component requires a name")
	}
	name := resource.NewComponentName("gorai", "gps", nameStr)

	// Extract configuration from attributes
	device := "/dev/gps-sim"
	baudRate := 9600
	natsURL := "nats://localhost:4222"
	namespace := "gorai"
	robotName := "robot"

	if attrs, ok := conf["attributes"].(map[string]any); ok {
		if d, ok := attrs["device"].(string); ok {
			device = d
		}
		if br, ok := attrs["baud_rate"].(float64); ok {
			baudRate = int(br)
		}
	}

	// Get NATS URL from dependencies or config
	if url, ok := conf["nats_url"].(string); ok {
		natsURL = url
	}
	if ns, ok := conf["namespace"].(string); ok {
		namespace = ns
	}
	if rn, ok := conf["robot_name"].(string); ok {
		robotName = rn
	}

	gpsConfig := &GPSConfig{
		Device:    device,
		BaudRate:  baudRate,
		NATSURL:   natsURL,
		Namespace: namespace,
		RobotName: robotName,
	}

	gps := &GPS{
		name:   name,
		config: gpsConfig,
	}

	return gps, nil
}

// Name returns the resource name.
func (g *GPS) Name() resource.Name {
	return g.name
}

// Start starts the GPS publisher in a background goroutine.
func (g *GPS) Start(ctx context.Context) error {
	g.mu.Lock()
	if g.running {
		g.mu.Unlock()
		return nil
	}
	g.mu.Unlock()

	// Create publisher config
	pubConfig := &publisher.Config{
		NATSURL:   g.config.NATSURL,
		Device:    g.config.Device,
		BaudRate:  g.config.BaudRate,
		Namespace: g.config.Namespace,
		RobotName: g.config.RobotName,
		Logger:    log.Default(),
	}

	// Create publisher
	pub, err := publisher.New(pubConfig)
	if err != nil {
		return fmt.Errorf("failed to create GPS publisher: %w", err)
	}

	g.publisher = pub

	// Create cancellable context
	pubCtx, cancel := context.WithCancel(ctx)
	g.cancel = cancel

	// Start publisher in goroutine
	g.wg.Add(1)
	go func() {
		defer g.wg.Done()
		if err := pub.Start(pubCtx); err != nil && err != context.Canceled {
			log.Printf("GPS publisher error: %v", err)
		}
	}()

	g.mu.Lock()
	g.running = true
	g.mu.Unlock()

	return nil
}

// Reconfigure updates the configuration.
func (g *GPS) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	// For now, reconfiguration requires restart
	return nil
}

// DoCommand executes arbitrary commands.
func (g *GPS) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_statistics":
			if g.publisher != nil {
				lines, errors := g.publisher.GetStatistics()
				return map[string]any{
					"lines_published": lines,
					"errors":          errors,
				}, nil
			}
			return map[string]any{
				"lines_published": 0,
				"errors":          0,
			}, nil
		case "get_config":
			return map[string]any{
				"device":     g.config.Device,
				"baud_rate":  g.config.BaudRate,
				"nats_url":   g.config.NATSURL,
				"namespace":  g.config.Namespace,
				"robot_name": g.config.RobotName,
			}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (g *GPS) Close(ctx context.Context) error {
	g.mu.Lock()
	if !g.running {
		g.mu.Unlock()
		return nil
	}
	g.running = false
	g.mu.Unlock()

	// Cancel the publisher
	if g.cancel != nil {
		g.cancel()
	}

	// Wait for publisher to stop
	g.wg.Wait()

	return nil
}
