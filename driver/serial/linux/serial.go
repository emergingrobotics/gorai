//go:build linux

// Package linux implements the serial.Port interface on Linux by wrapping the
// cross-platform go.bug.st/serial library.
package linux

import (
	"context"
	"fmt"
	"sync"

	"github.com/emergingrobotics/gorai/driver/serial"
	bugst "go.bug.st/serial"
)

// Port implements serial.Port over a go.bug.st/serial port.
type Port struct {
	mu   sync.Mutex
	cfg  serial.Config
	port bugst.Port
}

// Open opens the serial device described by cfg and returns a ready Port.
func Open(cfg serial.Config) (*Port, error) {
	p := &Port{cfg: cfg}
	if err := p.Open(context.Background()); err != nil {
		return nil, err
	}
	return p, nil
}

// mode translates a serial.Config into a go.bug.st mode.
func mode(cfg serial.Config) *bugst.Mode {
	m := &bugst.Mode{
		BaudRate: cfg.BaudRate,
		DataBits: cfg.DataBits,
	}
	switch cfg.Parity {
	case serial.ParityOdd:
		m.Parity = bugst.OddParity
	case serial.ParityEven:
		m.Parity = bugst.EvenParity
	case serial.ParityMark:
		m.Parity = bugst.MarkParity
	case serial.ParitySpace:
		m.Parity = bugst.SpaceParity
	default:
		m.Parity = bugst.NoParity
	}
	switch cfg.StopBits {
	case serial.StopBits15:
		m.StopBits = bugst.OnePointFiveStopBits
	case serial.StopBits2:
		m.StopBits = bugst.TwoStopBits
	default:
		m.StopBits = bugst.OneStopBit
	}
	return m
}

// Name implements driver.Driver.
func (p *Port) Name() string {
	return fmt.Sprintf("serial:%s", p.cfg.Path)
}

// Open opens the underlying device.
func (p *Port) Open(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.port != nil {
		return nil
	}
	port, err := bugst.Open(p.cfg.Path, mode(p.cfg))
	if err != nil {
		return fmt.Errorf("failed to open serial port %s: %w", p.cfg.Path, err)
	}
	p.port = port
	return nil
}

// Close closes the underlying device.
func (p *Port) Close(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.port == nil {
		return nil
	}
	err := p.port.Close()
	p.port = nil
	return err
}

// Read reads bytes from the port.
func (p *Port) Read(b []byte) (int, error) {
	p.mu.Lock()
	port := p.port
	p.mu.Unlock()
	if port == nil {
		return 0, fmt.Errorf("serial port not open")
	}
	return port.Read(b)
}

// Write writes bytes to the port.
func (p *Port) Write(b []byte) (int, error) {
	p.mu.Lock()
	port := p.port
	p.mu.Unlock()
	if port == nil {
		return 0, fmt.Errorf("serial port not open")
	}
	return port.Write(b)
}

// SetBaudRate updates the baud rate.
func (p *Port) SetBaudRate(baud int) error {
	p.cfg.BaudRate = baud
	return p.applyMode()
}

// SetDataBits updates the data bits.
func (p *Port) SetDataBits(bits int) error {
	p.cfg.DataBits = bits
	return p.applyMode()
}

// SetStopBits updates the stop bits.
func (p *Port) SetStopBits(bits serial.StopBits) error {
	p.cfg.StopBits = bits
	return p.applyMode()
}

// SetParity updates the parity.
func (p *Port) SetParity(parity serial.Parity) error {
	p.cfg.Parity = parity
	return p.applyMode()
}

// applyMode pushes the current config to the open port.
func (p *Port) applyMode() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.port == nil {
		return nil
	}
	return p.port.SetMode(mode(p.cfg))
}

// Flush discards buffered input and output.
func (p *Port) Flush(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.port == nil {
		return nil
	}
	if err := p.port.ResetInputBuffer(); err != nil {
		return err
	}
	return p.port.ResetOutputBuffer()
}

// Available is not supported by the underlying library; it returns 0.
func (p *Port) Available() (int, error) {
	return 0, nil
}

// Verify interface compliance.
var _ serial.Port = (*Port)(nil)
