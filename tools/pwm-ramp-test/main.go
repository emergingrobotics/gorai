// PWM Ramp Test - Tests all 16 PWM channels on the RP2040 PWM controller.
//
// This program:
//   1. Enables all 16 channels and sets them to 1500us
//   2. Ramps down to 1000us in steps of 50 every 250ms (all 16 channels updated together)
//   3. Ramps up to 2000us in steps of 50 every 250ms (all 16 channels updated together)
//   4. Ramps down to 1500us in steps of 50 every 250ms (all 16 channels updated together)
//   5. Holds for 5 seconds
//   6. Repeats
//
// It speaks GSP/2 (the typed binary Gorai Serial Protocol) via
// github.com/emergingrobotics/gorai-gsp, matching the rp2040-pwm firmware.
//
// Usage:
//
//	go run . /dev/ttyACM0
//	go run . -v /dev/ttyACM0       # verbose - show TX/RX messages
//	go run . -d /dev/ttyACM0       # debug - show payload bytes
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/emergingrobotics/gorai-gsp/client"
	"github.com/emergingrobotics/gorai-gsp/gsp"
	"github.com/emergingrobotics/gorai-gsp/gsp/messages"
	"github.com/emergingrobotics/gorai-gsp/transport"
	"go.bug.st/serial"
)

const (
	numChannels  = 16
	stepSize     = 50
	stepInterval = 250 * time.Millisecond
	holdDuration = 5 * time.Second
	// keepAliveRate must be shorter than the firmware failsafe timeout (default
	// 500ms) so held/ramping channels are not zeroed out mid-test.
	keepAliveRate = 200 * time.Millisecond

	baudRate    = 115200
	readTimeout = 100 * time.Millisecond
)

var (
	verbose bool
	debug   bool
)

type Client struct {
	port serial.Port
	gsp  *client.Client
}

func main() {
	flag.BoolVar(&verbose, "v", false, "verbose output - show TX/RX messages")
	flag.BoolVar(&verbose, "verbose", false, "verbose output - show TX/RX messages")
	flag.BoolVar(&debug, "d", false, "debug output - show payload bytes")
	flag.BoolVar(&debug, "debug", false, "debug output - show payload bytes")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options] <serial-port>\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s /dev/ttyACM0\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -v /dev/ttyACM0\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -d /dev/ttyACM0\n", os.Args[0])
	}
	flag.Parse()

	// Debug implies verbose
	if debug {
		verbose = true
	}

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}

	portName := flag.Arg(0)
	c, err := NewClient(portName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening port %s: %v\n", portName, err)
		os.Exit(1)
	}
	defer c.Close()

	// Handle Ctrl+C gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	fmt.Println("PWM Ramp Test")
	fmt.Println("=============")
	fmt.Printf("Port: %s\n", portName)
	fmt.Printf("Channels: 0-%d\n", numChannels-1)
	fmt.Printf("Step size: %dus every %v\n", stepSize, stepInterval)
	if verbose {
		fmt.Println("Verbose: ON")
	}
	if debug {
		fmt.Println("Debug: ON")
	}
	fmt.Println()
	fmt.Println("Pattern: 1500 -> 1000 -> 2000 -> 1500 -> hold 5s -> repeat")
	fmt.Println()
	fmt.Println("Press Ctrl+C to exit")
	fmt.Println()

	// Send ping to verify connectivity
	if verbose {
		fmt.Println("Sending PING to verify connectivity...")
	}
	c.sendPing()
	time.Sleep(200 * time.Millisecond)

	// Run the test loop
	cycle := 1
	for {
		fmt.Printf("=== Cycle %d ===\n", cycle)

		// Step 1: Enable all channels and set to 1500us
		fmt.Println("Enabling all channels at 1500us...")
		c.enableAll(true)
		c.setAllChannels(1500, true)
		time.Sleep(500 * time.Millisecond)

		// Step 2: Ramp down from 1500 to 1000
		fmt.Println("Ramping down: 1500 -> 1000...")
		if interrupted := c.ramp(1500, 1000, sigCh); interrupted {
			break
		}

		// Step 3: Ramp up from 1000 to 2000
		fmt.Println("Ramping up: 1000 -> 2000...")
		if interrupted := c.ramp(1000, 2000, sigCh); interrupted {
			break
		}

		// Step 4: Ramp down from 2000 to 1500
		fmt.Println("Ramping down: 2000 -> 1500...")
		if interrupted := c.ramp(2000, 1500, sigCh); interrupted {
			break
		}

		// Step 5: Hold at 1500 for 5 seconds
		fmt.Printf("Holding at 1500us for %v...\n", holdDuration)
		if interrupted := c.hold(1500, holdDuration, sigCh); interrupted {
			break
		}

		fmt.Println()
		cycle++
	}

	// Clean shutdown - center all channels
	fmt.Println("\nShutting down - centering all channels...")
	c.setAllChannels(1500, true)
	time.Sleep(200 * time.Millisecond)
	fmt.Println("Done.")
}

func NewClient(portName string) (*Client, error) {
	mode := &serial.Mode{
		BaudRate: baudRate,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, err
	}

	// A read timeout lets the GSP client's read loop poll without blocking
	// forever, so Close() shuts it down promptly.
	if err := port.SetReadTimeout(readTimeout); err != nil {
		port.Close()
		return nil, err
	}

	cfg := client.DefaultConfig()
	gc := client.New(transport.NewReadWriter(port), cfg)

	c := &Client{port: port, gsp: gc}
	c.registerHandlers()
	gc.Start()
	return c, nil
}

func (c *Client) Close() {
	// Closing the GSP client also closes the underlying transport (the port).
	_ = c.gsp.Close()
}

// registerHandlers wires up the responses we care about for verbose output.
func (c *Client) registerHandlers() {
	c.gsp.OnMessage(gsp.TypePong, func(msg gsp.Message, hdr *gsp.Header) {
		if verbose {
			fmt.Println("  RX: PONG")
		}
	})
	c.gsp.OnMessage(gsp.TypeStatus, func(msg gsp.Message, hdr *gsp.Header) {
		if verbose {
			fmt.Println("  RX: STATUS")
		}
	})
	c.gsp.OnMessage(gsp.TypeHeartbeat, func(msg gsp.Message, hdr *gsp.Header) {
		// Heartbeats are frequent; only show them in debug mode.
		if debug {
			fmt.Println("  RX: HEARTBEAT")
		}
	})
	c.gsp.OnMessage(gsp.TypePWMState, func(msg gsp.Message, hdr *gsp.Header) {
		if debug {
			fmt.Println("  RX: PWM_STATE")
		}
	})
	c.gsp.OnError(func(err error) {
		if debug {
			fmt.Printf("  RX error: %v\n", err)
		}
	})
}

// send transmits a GSP/2 message, logging it according to the verbosity flags.
func (c *Client) send(label string, msg gsp.Message, announce bool) {
	if announce && verbose {
		fmt.Printf("  TX: %s\n", label)
	}
	if debug {
		if payload, err := msg.Encode(); err == nil {
			fmt.Printf("  TX %s payload [%d bytes]: %s\n", label, len(payload), hex.EncodeToString(payload))
		}
	}
	if err := c.gsp.Send(msg); err != nil && debug {
		fmt.Printf("  TX error: %v\n", err)
	}
}

func (c *Client) sendPing() {
	c.send("PING", &messages.Ping{}, true)
	time.Sleep(50 * time.Millisecond) // Allow firmware to process before next command
}

func (c *Client) enableAll(enabled bool) {
	channels := make([]messages.PWMEnableChannel, numChannels)
	for i := range channels {
		channels[i] = messages.PWMEnableChannel{Channel: uint8(i), Enabled: enabled}
	}
	label := "PWM_ENABLE (all)"
	if !enabled {
		label = "PWM_DISABLE (all)"
	}
	c.send(label, &messages.PWMEnable{Channels: channels}, true)
}

// setAllChannels sets every channel to pulseUs in a single batch PWM_SET.
func (c *Client) setAllChannels(pulseUs int, announce bool) {
	channels := make([]messages.PWMSetChannel, numChannels)
	for i := range channels {
		channels[i] = messages.PWMSetChannel{Channel: uint8(i), PulseUS: uint16(pulseUs)}
	}
	if announce && verbose && !debug {
		fmt.Printf("  TX [PWM_SET]: all channels -> %dus\n", pulseUs)
	}
	c.send(fmt.Sprintf("PWM_SET (all -> %dus)", pulseUs), &messages.PWMSet{Channels: channels}, false)
}

// ramp smoothly changes PWM from start to end in steps.
// Returns true if interrupted by signal.
func (c *Client) ramp(start, end int, sigCh chan os.Signal) bool {
	step := stepSize
	if end < start {
		step = -stepSize
	}

	current := start
	stepTicker := time.NewTicker(stepInterval)
	keepAliveTicker := time.NewTicker(keepAliveRate)
	defer stepTicker.Stop()
	defer keepAliveTicker.Stop()

	for {
		select {
		case <-sigCh:
			return true
		case <-keepAliveTicker.C:
			// Send keep-alive to prevent failsafe (every 200ms)
			c.setAllChannels(current, false)
		case <-stepTicker.C:
			current += step
			if (step > 0 && current >= end) || (step < 0 && current <= end) {
				current = end
				c.setAllChannels(current, true)
				fmt.Printf("  %dus\n", current)
				return false
			}
			c.setAllChannels(current, true)
			fmt.Printf("  %dus\n", current)
		}
	}
}

// hold maintains a PWM value for the specified duration, sending keep-alive commands.
// Returns true if interrupted by signal.
func (c *Client) hold(pulseUs int, duration time.Duration, sigCh chan os.Signal) bool {
	c.setAllChannels(pulseUs, true)

	ticker := time.NewTicker(keepAliveRate)
	defer ticker.Stop()

	timeout := time.After(duration)

	for {
		select {
		case <-sigCh:
			return true
		case <-timeout:
			return false
		case <-ticker.C:
			// Send keep-alive to prevent failsafe
			c.setAllChannels(pulseUs, false)
		}
	}
}
