// Command hello-sensor demonstrates a simple CPU temperature sensor.
//
// This example shows how to:
//   - Create a Gorai node
//   - Implement a sensor component
//   - Publish readings to NATS
//   - Use platform-specific code for reading hardware data
//
// Usage:
//
//	go run ./examples/hello-sensor
//	go run ./examples/hello-sensor -interval 500ms -topic my.temp
//
// Subscribe to readings:
//
//	nats sub "gorai.hello.cpu_temp.data"
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorai/gorai/examples/hello-sensor/reader"
	"github.com/gorai/gorai/examples/hello-sensor/sensor"
	"github.com/gorai/gorai/examples/hello-sensor/sensor/fake"
	"github.com/gorai/gorai/pkg/node"
)

func main() {
	// Parse flags
	natsURL := flag.String("nats", "nats://localhost:4222", "NATS server URL")
	interval := flag.Duration("interval", time.Second, "Publishing interval")
	topic := flag.String("topic", "gorai.hello.cpu_temp.data", "NATS topic for readings")
	zone := flag.String("zone", "", "Thermal zone to read (empty for default)")
	useFake := flag.Bool("fake", false, "Use fake reader instead of real hardware")
	fakeTemp := flag.Float64("fake-temp", 42.0, "Temperature for fake reader")
	flag.Parse()

	// Create context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	// Create node
	n, err := node.New("hello_sensor", node.WithNATS(*natsURL))
	if err != nil {
		log.Fatalf("Failed to create node: %v", err)
	}
	defer n.Close()

	// Create reader
	var r reader.Reader
	if *useFake {
		fr := fake.New()
		fr.SetTemperature(*fakeTemp)
		r = fr
		log.Printf("Using fake reader with temperature %.1f°C", *fakeTemp)
	} else {
		r, err = reader.New()
		if err != nil {
			log.Fatalf("Failed to create reader: %v", err)
		}
	}
	defer r.Close()

	// Show available zones
	zones, err := r.Zones(ctx)
	if err != nil {
		log.Printf("Warning: failed to get zones: %v", err)
	} else {
		log.Printf("Platform: %s, Available zones: %v", r.Platform(), zones)
	}

	// Create sensor
	cfg := sensor.Config{
		Name:     "cpu_temp",
		Zone:     *zone,
		Interval: *interval,
		Topic:    *topic,
	}

	tempSensor, err := sensor.New(n, r, cfg)
	if err != nil {
		log.Fatalf("Failed to create sensor: %v", err)
	}
	defer tempSensor.Close(ctx)

	// Print initial reading
	readings, err := tempSensor.Readings(ctx)
	if err != nil {
		log.Printf("Warning: initial reading failed: %v", err)
	} else {
		log.Printf("Initial reading: %.1f°C (%.1f°F)",
			readings["temperature_celsius"],
			readings["temperature_fahrenheit"])
	}

	// Start publishing
	log.Printf("Starting sensor, publishing to %s every %v", cfg.Topic, cfg.Interval)
	if err := tempSensor.Start(ctx); err != nil {
		log.Fatalf("Failed to start sensor: %v", err)
	}

	// Print status periodically
	statusTicker := time.NewTicker(10 * time.Second)
	defer statusTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Get final stats
			stats, _ := tempSensor.DoCommand(ctx, map[string]any{"command": "get_stats"})
			if stats != nil {
				fmt.Printf("\nFinal stats:\n")
				fmt.Printf("  Readings: %v\n", stats["reading_count"])
				fmt.Printf("  Errors: %v\n", stats["error_count"])
				fmt.Printf("  Min: %.1f°C, Max: %.1f°C, Avg: %.1f°C\n",
					stats["min_celsius"], stats["max_celsius"], stats["avg_celsius"])
			}
			return

		case <-statusTicker.C:
			stats, err := tempSensor.DoCommand(ctx, map[string]any{"command": "get_stats"})
			if err != nil {
				log.Printf("Failed to get stats: %v", err)
				continue
			}

			lastReading, _ := tempSensor.DoCommand(ctx, map[string]any{"command": "get_last_reading"})
			if lastReading != nil {
				log.Printf("Status: %.1f°C | Readings: %v | Errors: %v",
					lastReading["temperature_celsius"],
					stats["reading_count"],
					stats["error_count"])
			}
		}
	}
}
