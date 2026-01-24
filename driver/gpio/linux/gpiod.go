//go:build linux

// Package linux provides Linux GPIO implementation via libgpiod/gpiocdev.
package linux

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gorai/gorai/driver/gpio"
	"github.com/warthog618/go-gpiocdev"
)

// Driver implements gpio.Driver for Linux using gpiocdev.
type Driver struct {
	chipPath string
	chip     *gpiocdev.Chip
	mu       sync.Mutex
	pins     map[int]*Pin
	closed   bool
}

// NewDriver creates a new Linux GPIO driver for the given chip path.
// chipPath is typically "/dev/gpiochip0" or "/dev/gpiochip4".
func NewDriver(chipPath string) (*Driver, error) {
	chip, err := gpiocdev.NewChip(chipPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open GPIO chip %s: %w", chipPath, err)
	}

	return &Driver{
		chipPath: chipPath,
		chip:     chip,
		pins:     make(map[int]*Pin),
	}, nil
}

// Name implements driver.Driver.
func (d *Driver) Name() string {
	return fmt.Sprintf("gpio:%s", d.chipPath)
}

// Open implements driver.Driver (no-op, already opened).
func (d *Driver) Open(ctx context.Context) error {
	return nil
}

// Close implements driver.Driver.
func (d *Driver) Close(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil
	}
	d.closed = true

	// Close all pins first
	for _, pin := range d.pins {
		if pin.line != nil {
			pin.line.Close()
		}
	}
	d.pins = nil

	// Close chip
	if d.chip != nil {
		err := d.chip.Close()
		d.chip = nil
		return err
	}
	return nil
}

// Pin returns a GPIO pin by number.
func (d *Driver) Pin(number int) (gpio.Pin, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return nil, fmt.Errorf("GPIO driver is closed")
	}

	// Return cached pin if already requested
	if pin, ok := d.pins[number]; ok {
		return pin, nil
	}

	// Create new pin (line not yet requested)
	pin := &Pin{
		driver: d,
		number: number,
		name:   fmt.Sprintf("GPIO%d", number),
		dir:    gpio.Input, // Default to input
	}
	d.pins[number] = pin

	return pin, nil
}

// PinByName returns a GPIO pin by name.
// For Linux, this just parses "GPIO<number>" format.
func (d *Driver) PinByName(name string) (gpio.Pin, error) {
	var number int
	if _, err := fmt.Sscanf(name, "GPIO%d", &number); err != nil {
		return nil, fmt.Errorf("invalid GPIO name %q, expected GPIO<number>", name)
	}
	return d.Pin(number)
}

// Pin implements gpio.Pin for Linux.
type Pin struct {
	driver *Driver
	number int
	name   string
	line   *gpiocdev.Line
	dir    gpio.Direction
	mu     sync.Mutex
}

// Number implements driver.Pin.
func (p *Pin) Number() int {
	return p.number
}

// Name implements driver.Pin.
func (p *Pin) Name() string {
	return p.name
}

// SetDirection implements gpio.Pin.
func (p *Pin) SetDirection(ctx context.Context, dir gpio.Direction) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// If direction hasn't changed and line is already requested, no-op
	if p.line != nil && p.dir == dir {
		return nil
	}

	// Close existing line if any
	if p.line != nil {
		p.line.Close()
		p.line = nil
	}

	// Request line with new direction
	var line *gpiocdev.Line
	var err error

	if dir == gpio.Output {
		line, err = p.driver.chip.RequestLine(p.number, gpiocdev.AsOutput(0))
	} else {
		line, err = p.driver.chip.RequestLine(p.number, gpiocdev.AsInput)
	}

	if err != nil {
		return fmt.Errorf("failed to set direction for GPIO%d: %w", p.number, err)
	}

	p.line = line
	p.dir = dir
	return nil
}

// Direction implements gpio.Pin.
func (p *Pin) Direction() gpio.Direction {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.dir
}

// Read implements gpio.Pin.
func (p *Pin) Read(ctx context.Context) (bool, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Ensure line is requested for input
	if p.line == nil {
		line, err := p.driver.chip.RequestLine(p.number, gpiocdev.AsInput)
		if err != nil {
			return false, fmt.Errorf("failed to request GPIO%d for input: %w", p.number, err)
		}
		p.line = line
		p.dir = gpio.Input
	}

	val, err := p.line.Value()
	if err != nil {
		return false, fmt.Errorf("failed to read GPIO%d: %w", p.number, err)
	}

	return val == 1, nil
}

// Write implements gpio.Pin.
func (p *Pin) Write(ctx context.Context, high bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Ensure line is requested for output
	if p.line == nil || p.dir != gpio.Output {
		// Close existing line if any
		if p.line != nil {
			p.line.Close()
		}
		line, err := p.driver.chip.RequestLine(p.number, gpiocdev.AsOutput(0))
		if err != nil {
			return fmt.Errorf("failed to request GPIO%d for output: %w", p.number, err)
		}
		p.line = line
		p.dir = gpio.Output
	}

	val := 0
	if high {
		val = 1
	}

	if err := p.line.SetValue(val); err != nil {
		return fmt.Errorf("failed to write GPIO%d: %w", p.number, err)
	}

	return nil
}

// PWM returns the PWM interface for this pin.
// GPIO pins don't support hardware PWM directly.
func (p *Pin) PWM() (gpio.PWM, error) {
	return nil, fmt.Errorf("hardware PWM not supported on GPIO%d, use SoftwarePWM via HAL", p.number)
}

// WaitForEdge implements gpio.EdgeWaiter.
func (p *Pin) WaitForEdge(ctx context.Context, edge gpio.Edge, timeout time.Duration) (bool, error) {
	p.mu.Lock()

	// Close existing line since we need to reconfigure with edge detection
	if p.line != nil {
		p.line.Close()
		p.line = nil
	}

	// Create channel for edge events
	eventCh := make(chan gpiocdev.LineEvent, 1)
	handler := func(evt gpiocdev.LineEvent) {
		select {
		case eventCh <- evt:
		default:
		}
	}

	var edgeOpt gpiocdev.LineReqOption
	switch edge {
	case gpio.EdgeRising:
		edgeOpt = gpiocdev.WithRisingEdge
	case gpio.EdgeFalling:
		edgeOpt = gpiocdev.WithFallingEdge
	case gpio.EdgeBoth:
		edgeOpt = gpiocdev.WithBothEdges
	default:
		p.mu.Unlock()
		return false, fmt.Errorf("invalid edge type")
	}

	line, err := p.driver.chip.RequestLine(p.number,
		gpiocdev.AsInput,
		edgeOpt,
		gpiocdev.WithEventHandler(handler),
	)
	if err != nil {
		p.mu.Unlock()
		return false, fmt.Errorf("failed to request GPIO%d with edge detection: %w", p.number, err)
	}
	p.line = line
	p.dir = gpio.Input
	p.mu.Unlock()

	// Wait for edge event, context cancellation, or timeout
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-eventCh:
		return true, nil
	case <-time.After(timeout):
		return false, nil
	}
}

// Verify interface compliance
var _ gpio.Driver = (*Driver)(nil)
var _ gpio.Pin = (*Pin)(nil)
var _ gpio.EdgeWaiter = (*Pin)(nil)
