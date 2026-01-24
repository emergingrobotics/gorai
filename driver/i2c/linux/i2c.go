//go:build linux

// Package linux provides Linux I2C implementation via /dev/i2c-*.
package linux

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/gorai/gorai/driver/i2c"
	"golang.org/x/sys/unix"
)

const (
	i2cSlave      = 0x0703
	i2cSlaveForce = 0x0706
)

// Bus implements i2c.Bus for Linux.
type Bus struct {
	file *os.File
	fd   int
	path string
	mu   sync.Mutex
}

// Open opens an I2C bus device.
func Open(devicePath string) (*Bus, error) {
	file, err := os.OpenFile(devicePath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open I2C bus %s: %w", devicePath, err)
	}

	return &Bus{
		file: file,
		fd:   int(file.Fd()),
		path: devicePath,
	}, nil
}

// Name implements driver.Driver.
func (b *Bus) Name() string {
	return fmt.Sprintf("i2c:%s", b.path)
}

// Open implements driver.Driver (no-op, already opened).
func (b *Bus) Open(ctx context.Context) error {
	return nil
}

// Close closes the I2C bus.
func (b *Bus) Close(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if b.file != nil {
		err := b.file.Close()
		b.file = nil
		return err
	}
	return nil
}

// Device returns a device handle for the given address.
func (b *Bus) Device(addr uint16) i2c.Device {
	return &Device{
		bus:  b,
		addr: addr,
	}
}

// setSlaveAddress sets the target I2C address.
func (b *Bus) setSlaveAddress(address uint16) error {
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(b.fd),
		i2cSlave,
		uintptr(address),
	)
	if errno != 0 {
		return fmt.Errorf("failed to set slave address 0x%02X: %v", address, errno)
	}
	return nil
}

// Device implements i2c.Device for Linux.
type Device struct {
	bus  *Bus
	addr uint16
}

// Address returns the device address.
func (d *Device) Address() uint16 {
	return d.addr
}

// Read reads n bytes from the device.
func (d *Device) Read(ctx context.Context, n int) ([]byte, error) {
	d.bus.mu.Lock()
	defer d.bus.mu.Unlock()

	if err := d.bus.setSlaveAddress(d.addr); err != nil {
		return nil, err
	}

	buf := make([]byte, n)
	read, err := d.bus.file.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}
	if read < n {
		return nil, fmt.Errorf("short read: got %d, expected %d", read, n)
	}

	return buf, nil
}

// Write writes data to the device.
func (d *Device) Write(ctx context.Context, data []byte) error {
	d.bus.mu.Lock()
	defer d.bus.mu.Unlock()

	if err := d.bus.setSlaveAddress(d.addr); err != nil {
		return err
	}

	written, err := d.bus.file.Write(data)
	if err != nil {
		return fmt.Errorf("write failed: %w", err)
	}
	if written < len(data) {
		return fmt.Errorf("short write: wrote %d, expected %d", written, len(data))
	}

	return nil
}

// WriteRead writes data then reads n bytes.
func (d *Device) WriteRead(ctx context.Context, write []byte, readLen int) ([]byte, error) {
	d.bus.mu.Lock()
	defer d.bus.mu.Unlock()

	if err := d.bus.setSlaveAddress(d.addr); err != nil {
		return nil, err
	}

	// Write
	if _, err := d.bus.file.Write(write); err != nil {
		return nil, fmt.Errorf("write failed: %w", err)
	}

	// Read
	buf := make([]byte, readLen)
	if _, err := d.bus.file.Read(buf); err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}

	return buf, nil
}

// ReadReg reads n bytes from a register.
func (d *Device) ReadReg(ctx context.Context, reg byte, n int) ([]byte, error) {
	return d.WriteRead(ctx, []byte{reg}, n)
}

// WriteReg writes data to a register.
func (d *Device) WriteReg(ctx context.Context, reg byte, data []byte) error {
	buf := make([]byte, 1+len(data))
	buf[0] = reg
	copy(buf[1:], data)
	return d.Write(ctx, buf)
}

// ReadByte reads a single byte.
func (d *Device) ReadByte(ctx context.Context) (byte, error) {
	data, err := d.Read(ctx, 1)
	if err != nil {
		return 0, err
	}
	return data[0], nil
}

// WriteByte writes a single byte.
func (d *Device) WriteByte(ctx context.Context, b byte) error {
	return d.Write(ctx, []byte{b})
}

// ReadByteReg reads a single byte from a register.
func (d *Device) ReadByteReg(ctx context.Context, reg byte) (byte, error) {
	data, err := d.ReadReg(ctx, reg, 1)
	if err != nil {
		return 0, err
	}
	return data[0], nil
}

// WriteByteReg writes a single byte to a register.
func (d *Device) WriteByteReg(ctx context.Context, reg, value byte) error {
	return d.WriteReg(ctx, reg, []byte{value})
}

// Verify interface compliance
var _ i2c.Bus = (*Bus)(nil)
var _ i2c.Device = (*Device)(nil)
