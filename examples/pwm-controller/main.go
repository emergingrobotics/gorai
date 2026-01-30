// Package main demonstrates using the gorai-gsp library to control
// PWM outputs on an RP2040-based controller running the rp2040-pwm firmware.
//
// This example shows:
// - Connecting to the PWM controller via serial
// - Setting PWM pulse widths for servo control
// - Enabling/disabling PWM channels
// - Sweeping servos through their range
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/thefloweringash/gorai-gsp/client"
	"github.com/thefloweringash/gorai-gsp/gsp"
	"github.com/thefloweringash/gorai-gsp/gsp/messages"
	"github.com/thefloweringash/gorai-gsp/transport"
	"go.bug.st/serial"
)

const (
	// Default PWM values for servo control
	servoMin    = 1000 // Minimum pulse width (µs)
	servoCenter = 1500 // Center/neutral position (µs)
	servoMax    = 2000 // Maximum pulse width (µs)
)

var verbose bool

func main() {
	// Parse command-line flags
	port := flag.String("port", "/dev/ttyACM0", "Serial port for PWM controller")
	baud := flag.Int("baud", 230400, "Baud rate")
	channel := flag.Int("channel", 0, "PWM channel to control (0-15)")
	demo := flag.String("demo", "sweep", "Demo mode: sweep, center, or manual")
	pulse := flag.Int("pulse", 1500, "Pulse width in µs (for manual mode)")
	flag.BoolVar(&verbose, "v", false, "Verbose output")
	flag.BoolVar(&verbose, "verbose", false, "Verbose output")
	flag.Parse()

	if verbose {
		log.Printf("PWM Controller Example")
		log.Printf("  Port:    %s", *port)
		log.Printf("  Baud:    %d", *baud)
		log.Printf("  Channel: %d", *channel)
		log.Printf("  Mode:    %s", *demo)
	}

	// Open serial port
	serialPort, err := openSerial(*port, *baud)
	if err != nil {
		log.Fatalf("Failed to open serial port: %v", err)
	}
	defer serialPort.Close()

	if verbose {
		log.Println("Serial port opened successfully")
	}

	// Create GSP client
	t := transport.NewReadWriter(serialPort)
	cfg := client.DefaultConfig()
	c := client.New(t, cfg)

	// Set up message handlers
	c.OnMessage(gsp.TypeHeartbeat, func(msg gsp.Message, hdr *gsp.Header) {
		if verbose {
			log.Println("Heartbeat received")
		}
	})

	c.OnMessage(gsp.TypePWMState, func(msg gsp.Message, hdr *gsp.Header) {
		if !verbose {
			return
		}
		if state, ok := msg.(*messages.PWMState); ok {
			for _, ch := range state.Channels {
				enabled := "disabled"
				if ch.Flags&messages.PWMStateFlagEnabled != 0 {
					enabled = "enabled"
				}
				failsafe := ""
				if ch.Flags&messages.PWMStateFlagFailsafe != 0 {
					failsafe = " (failsafe)"
				}
				log.Printf("  Channel %d: %d µs (%s%s)", ch.Channel, ch.PulseUS, enabled, failsafe)
			}
		}
	})

	c.OnError(func(err error) {
		if verbose {
			log.Printf("GSP error: %v", err)
		}
	})

	// Start the client's read loop
	c.Start()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Enable the channel
	if verbose {
		log.Printf("Enabling channel %d...", *channel)
	}
	if err := enableChannel(c, uint8(*channel), true); err != nil {
		if verbose {
			log.Printf("Warning: Failed to enable channel: %v", err)
		}
	}

	// Run the selected demo
	done := make(chan struct{})
	go func() {
		defer close(done)

		switch *demo {
		case "sweep":
			runSweepDemo(c, uint8(*channel), sigChan)
		case "center":
			runCenterDemo(c, uint8(*channel), sigChan)
		case "manual":
			runManualDemo(c, uint8(*channel), uint16(*pulse), sigChan)
		default:
			log.Printf("Unknown demo mode: %s", *demo)
		}
	}()

	<-done

	// Disable channel and cleanup
	if verbose {
		log.Printf("Disabling channel %d...", *channel)
	}
	if err := enableChannel(c, uint8(*channel), false); err != nil {
		if verbose {
			log.Printf("Warning: Failed to disable channel: %v", err)
		}
	}

	if verbose {
		log.Println("Shutting down...")
	}
	c.Close()
}

// openSerial opens the serial port with the specified configuration.
func openSerial(portName string, baudRate int) (serial.Port, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	return serial.Open(portName, mode)
}

// setPWM sets the pulse width for a PWM channel.
func setPWM(c *client.Client, channel uint8, pulseUS uint16) error {
	msg := &messages.PWMSet{
		Channels: []messages.PWMSetChannel{
			{Channel: channel, PulseUS: pulseUS},
		},
	}
	return c.Send(msg)
}

// enableChannel enables or disables a PWM channel.
func enableChannel(c *client.Client, channel uint8, enabled bool) error {
	msg := &messages.PWMEnable{
		Channels: []messages.PWMEnableChannel{
			{Channel: channel, Enabled: enabled},
		},
	}
	return c.Send(msg)
}

// runSweepDemo sweeps a servo back and forth through its range.
func runSweepDemo(c *client.Client, channel uint8, sigChan chan os.Signal) {
	if verbose {
		log.Println("Running sweep demo (Ctrl+C to stop)...")
	}

	ticker := time.NewTicker(20 * time.Millisecond) // 50Hz update rate
	defer ticker.Stop()

	pulse := servoCenter
	direction := 1
	step := 10

	for {
		select {
		case <-ticker.C:
			if err := setPWM(c, channel, uint16(pulse)); err != nil {
				if verbose {
					log.Printf("Error setting PWM: %v", err)
				}
			}

			pulse += direction * step

			// Reverse direction at limits
			if pulse >= servoMax {
				pulse = servoMax
				direction = -1
				if verbose {
					log.Printf("Sweep: max position (%d µs)", pulse)
				}
			} else if pulse <= servoMin {
				pulse = servoMin
				direction = 1
				if verbose {
					log.Printf("Sweep: min position (%d µs)", pulse)
				}
			}

		case <-sigChan:
			if verbose {
				log.Println("Stopping sweep demo...")
			}
			// Return to center
			setPWM(c, channel, servoCenter)
			return
		}
	}
}

// runCenterDemo sets the servo to center position and holds it.
func runCenterDemo(c *client.Client, channel uint8, sigChan chan os.Signal) {
	if verbose {
		log.Printf("Setting channel %d to center position (%d µs)...", channel, servoCenter)
	}

	if err := setPWM(c, channel, servoCenter); err != nil {
		if verbose {
			log.Printf("Error setting PWM: %v", err)
		}
		return
	}

	if verbose {
		log.Println("Holding center position (Ctrl+C to stop)...")
	}

	// Keep sending center position periodically to prevent failsafe
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := setPWM(c, channel, servoCenter); err != nil {
				if verbose {
					log.Printf("Error setting PWM: %v", err)
				}
			}
		case <-sigChan:
			if verbose {
				log.Println("Stopping center demo...")
			}
			return
		}
	}
}

// runManualDemo sets the servo to a specified position.
func runManualDemo(c *client.Client, channel uint8, pulseUS uint16, sigChan chan os.Signal) {
	if verbose {
		log.Printf("Setting channel %d to %d µs...", channel, pulseUS)
	}

	if err := setPWM(c, channel, pulseUS); err != nil {
		if verbose {
			log.Printf("Error setting PWM: %v", err)
		}
		return
	}

	if verbose {
		log.Println("Holding position (Ctrl+C to stop)...")
	}

	// Keep sending position periodically to prevent failsafe
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := setPWM(c, channel, pulseUS); err != nil {
				if verbose {
					log.Printf("Error setting PWM: %v", err)
				}
			}
		case <-sigChan:
			if verbose {
				log.Println("Stopping manual demo...")
			}
			return
		}
	}
}

// queryPWMState queries the current state of all PWM channels.
func queryPWMState(c *client.Client) error {
	msg := &messages.PWMQuery{
		Channel: 0xFF, // Query all channels
	}
	return c.Send(msg)
}

func usage() {
	fmt.Fprintf(os.Stderr, "PWM Controller Example - Control servos via GSP/2 protocol\n\n")
	fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "Options:\n")
	flag.PrintDefaults()
	fmt.Fprintf(os.Stderr, "\nExamples:\n")
	fmt.Fprintf(os.Stderr, "  %s -demo sweep              # Sweep servo on channel 0\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -demo center -channel 1  # Center servo on channel 1\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -demo manual -pulse 1200 # Set channel 0 to 1200 µs\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -port /dev/ttyUSB0       # Use different serial port\n", os.Args[0])
	fmt.Fprintf(os.Stderr, "  %s -v                       # Enable verbose output\n", os.Args[0])
}
