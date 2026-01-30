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
// Usage:
//
//	go run . /dev/ttyACM0
//	go run . -v /dev/ttyACM0       # verbose - show TX/RX messages
//	go run . -d /dev/ttyACM0       # debug - show raw bytes and frame details
package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorai/rp2040-pwm/firmware/gsp"
	"go.bug.st/serial"
)

const (
	robotID       = "gorai"
	numChannels   = 16
	stepSize      = 50
	stepInterval  = 250 * time.Millisecond
	holdDuration  = 5 * time.Second
	keepAliveRate = 200 * time.Millisecond

	// Buffer sizes: 50% larger than longest expected line.
	// Longest batch command (16 channels) is ~480 bytes, so 480 * 1.5 = 720.
	maxCommandSize = 480
	bufferSize     = maxCommandSize + maxCommandSize/2 // 720
)

var (
	verbose bool
	debug   bool
)

type Client struct {
	port   serial.Port
	parser *gsp.Parser
}

func main() {
	flag.BoolVar(&verbose, "v", false, "verbose output - show TX/RX messages")
	flag.BoolVar(&verbose, "verbose", false, "verbose output - show TX/RX messages")
	flag.BoolVar(&debug, "d", false, "debug output - show raw bytes and frame details")
	flag.BoolVar(&debug, "debug", false, "debug output - show raw bytes and frame details")
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
	client, err := NewClient(portName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening port %s: %v\n", portName, err)
		os.Exit(1)
	}
	defer client.Close()

	// Handle Ctrl+C gracefully
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start reader goroutine for responses
	go client.readLoop()

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
	client.sendPing()
	time.Sleep(200 * time.Millisecond)

	// Run the test loop
	cycle := 1
	for {
		fmt.Printf("=== Cycle %d ===\n", cycle)

		// Step 1: Enable all channels and set to 1500us
		fmt.Println("Enabling all channels at 1500us...")
		client.enableAll(true)
		client.setAllChannels(1500)
		time.Sleep(500 * time.Millisecond)

		// Step 2: Ramp down from 1500 to 1000
		fmt.Println("Ramping down: 1500 -> 1000...")
		if interrupted := client.ramp(1500, 1000, sigCh); interrupted {
			break
		}

		// Step 3: Ramp up from 1000 to 2000
		fmt.Println("Ramping up: 1000 -> 2000...")
		if interrupted := client.ramp(1000, 2000, sigCh); interrupted {
			break
		}

		// Step 4: Ramp down from 2000 to 1500
		fmt.Println("Ramping down: 2000 -> 1500...")
		if interrupted := client.ramp(2000, 1500, sigCh); interrupted {
			break
		}

		// Step 5: Hold at 1500 for 5 seconds
		fmt.Printf("Holding at 1500us for %v...\n", holdDuration)
		if interrupted := client.hold(1500, holdDuration, sigCh); interrupted {
			break
		}

		fmt.Println()
		cycle++
	}

	// Clean shutdown - center all channels
	fmt.Println("\nShutting down - centering all channels...")
	client.setAllChannels(1500)
	time.Sleep(200 * time.Millisecond)
	fmt.Println("Done.")
}

func NewClient(portName string) (*Client, error) {
	mode := &serial.Mode{
		BaudRate: 115200,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}

	port, err := serial.Open(portName, mode)
	if err != nil {
		return nil, err
	}

	// Clear any stale data in serial buffers by sending sync bytes.
	// The firmware parser ignores non-STX bytes when waiting for frame start.
	syncBytes := make([]byte, 16)
	port.Write(syncBytes)
	time.Sleep(50 * time.Millisecond)

	// Drain any pending input
	port.SetReadTimeout(100)
	discardBuf := make([]byte, 1024)
	for {
		n, _ := port.Read(discardBuf)
		if n == 0 {
			break
		}
	}

	return &Client{
		port:   port,
		parser: gsp.NewParser(bufferSize),
	}, nil
}

func (c *Client) Close() {
	c.port.Close()
}

func (c *Client) readLoop() {
	buf := make([]byte, bufferSize)
	for {
		c.port.SetReadTimeout(100)
		n, err := c.port.Read(buf)
		if err != nil || n == 0 {
			continue
		}

		if debug {
			fmt.Printf("  RX raw [%d bytes]: %s\n", n, hex.EncodeToString(buf[:n]))
		}

		// Parse responses
		for i := 0; i < n; i++ {
			frame, err := c.parser.Feed(buf[i : i+1])
			if err != nil {
				if debug {
					fmt.Printf("  RX parse error: %v\n", err)
				}
				continue
			}
			if frame != nil {
				c.handleFrame(frame)
			}
		}
	}
}

func (c *Client) handleFrame(frame *gsp.Frame) {
	if debug {
		fmt.Printf("  RX frame [%d bytes]: %s\n", len(frame.Payload), string(frame.Payload))
	}

	msg, err := gsp.ParseMessage(frame.Payload)
	if err != nil {
		if verbose {
			fmt.Printf("  RX [raw]: %s\n", string(frame.Payload))
		}
		return
	}

	switch msg.Command {
	case gsp.CmdPONG:
		if verbose {
			fmt.Println("  RX: PONG")
		}
	case gsp.CmdPUB:
		// Skip heartbeat unless debug
		if strings.HasSuffix(msg.Subject, ".heartbeat") && !debug {
			return
		}
		if verbose {
			subj := formatSubject(msg.Subject)
			fmt.Printf("  RX [%s]: %s\n", subj, string(msg.Payload))
		}
	default:
		if verbose {
			fmt.Printf("  RX: %s %s\n", msg.Command, string(msg.Payload))
		}
	}
}

func formatSubject(subject string) string {
	parts := strings.Split(subject, ".")
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], ".")
	}
	return subject
}

func (c *Client) sendPing() {
	if verbose {
		fmt.Println("  TX: PING")
	}
	payload := []byte("PING\r\n")
	frame, _ := gsp.BuildFrame(payload)
	if debug {
		fmt.Printf("  TX frame [%d bytes]: %s\n", len(frame), hex.EncodeToString(frame))
	}
	c.port.Write(frame)
	time.Sleep(50 * time.Millisecond) // Allow firmware to process before next command
}

func (c *Client) sendPub(subject string, payload string) {
	fullSubject := robotID + ".mcu." + subject
	msg := gsp.FormatPub(fullSubject, []byte(payload))
	frame, err := gsp.BuildFrame(msg)
	if err != nil {
		fmt.Printf("  TX BuildFrame error: %v (msg len=%d)\n", err, len(msg))
		return
	}

	if verbose {
		// Truncate payload for display if too long
		displayPayload := payload
		if len(displayPayload) > 80 {
			displayPayload = displayPayload[:77] + "..."
		}
		fmt.Printf("  TX [%s]: %s\n", subject, displayPayload)
	}
	if debug {
		fmt.Printf("  TX msg [%d bytes]: %s\n", len(msg), string(msg))
		fmt.Printf("  TX frame [%d bytes]: %s\n", len(frame), hex.EncodeToString(frame))
	}

	n, err := c.port.Write(frame)
	if debug {
		if err != nil {
			fmt.Printf("  TX error: %v\n", err)
		} else {
			fmt.Printf("  TX wrote %d bytes\n", n)
		}
	}
}

func (c *Client) enableAll(enabled bool) {
	payload := fmt.Sprintf(`{"enabled":%t}`, enabled)
	c.sendPub("pwm.enable", payload)
}

func (c *Client) setAllChannels(pulseUs int) {
	// Send individual commands for all channels in rapid succession.
	// Batch commands have reliability issues, so we use individual commands
	// sent quickly to achieve the same effect.
	if verbose && !debug {
		fmt.Printf("  TX [pwm.command]: all channels -> %dus\n", pulseUs)
	}
	for i := 0; i < numChannels; i++ {
		payload := fmt.Sprintf(`{"channel":%d,"pulse_us":%d}`, i, pulseUs)
		c.sendPubQuiet("pwm.command", payload)
	}
	time.Sleep(20 * time.Millisecond) // Allow firmware to process all commands
}

func (c *Client) setAllChannelsQuiet(pulseUs int) {
	// Send individual commands for all channels in rapid succession (for keep-alive)
	for i := 0; i < numChannels; i++ {
		payload := fmt.Sprintf(`{"channel":%d,"pulse_us":%d}`, i, pulseUs)
		c.sendPubQuiet("pwm.command", payload)
	}
	time.Sleep(20 * time.Millisecond) // Allow firmware to process all commands
}

func (c *Client) sendPubQuiet(subject string, payload string) {
	fullSubject := robotID + ".mcu." + subject
	msg := gsp.FormatPub(fullSubject, []byte(payload))
	frame, _ := gsp.BuildFrame(msg)

	if debug {
		fmt.Printf("  TX [%s]: %s\n", subject, payload)
		fmt.Printf("  TX frame [%d bytes]: %s\n", len(frame), hex.EncodeToString(frame))
	}

	c.port.Write(frame)
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
			c.setAllChannelsQuiet(current)
		case <-stepTicker.C:
			current += step
			if (step > 0 && current >= end) || (step < 0 && current <= end) {
				current = end
				c.setAllChannels(current)
				fmt.Printf("  %dus\n", current)
				return false
			}
			c.setAllChannels(current)
			fmt.Printf("  %dus\n", current)
		}
	}
}

// hold maintains a PWM value for the specified duration, sending keep-alive commands.
// Returns true if interrupted by signal.
func (c *Client) hold(pulseUs int, duration time.Duration, sigCh chan os.Signal) bool {
	c.setAllChannels(pulseUs)

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
			c.setAllChannels(pulseUs)
		}
	}
}
