package i2c_bridge

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/sys/unix"
)

// I2C ioctl commands from linux/i2c-dev.h
const (
	// I2C_SLAVE sets the 7-bit slave address
	I2C_SLAVE = 0x0703

	// I2C_SLAVE_FORCE sets address even if in use by a driver
	I2C_SLAVE_FORCE = 0x0706
)

// I2CBus provides low-level I2C bus access via Linux i2c-dev.
type I2CBus struct {
	file *os.File
	fd   int
	mu   sync.Mutex
}

// OpenBus opens an I2C bus device.
func OpenBus(devicePath string) (*I2CBus, error) {
	file, err := os.OpenFile(devicePath, os.O_RDWR, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to open I2C bus %s: %w", devicePath, err)
	}

	return &I2CBus{
		file: file,
		fd:   int(file.Fd()),
	}, nil
}

// Close closes the I2C bus.
func (b *I2CBus) Close() error {
	if b.file != nil {
		return b.file.Close()
	}
	return nil
}

// setSlaveAddress sets the I2C slave address for subsequent operations.
// Must be called with mutex held.
func (b *I2CBus) setSlaveAddress(address uint8) error {
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(b.fd),
		I2C_SLAVE,
		uintptr(address),
	)
	if errno != 0 {
		return fmt.Errorf("failed to set slave address 0x%02X: %v", address, errno)
	}
	return nil
}

// ReadRegister reads data from a specific register address.
// This performs a write of the register address followed by a read.
func (b *I2CBus) ReadRegister(address uint8, register uint8, length int) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Set slave address
	if err := b.setSlaveAddress(address); err != nil {
		return nil, err
	}

	// Write register address
	_, err := b.file.Write([]byte{register})
	if err != nil {
		return nil, fmt.Errorf("failed to write register address: %w", err)
	}

	// Read data
	buf := make([]byte, length)
	n, err := b.file.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	if n < length {
		return nil, fmt.Errorf("short read: got %d bytes, expected %d", n, length)
	}

	return buf, nil
}

// ReadRaw reads data directly without writing a register address first.
func (b *I2CBus) ReadRaw(address uint8, length int) ([]byte, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Set slave address
	if err := b.setSlaveAddress(address); err != nil {
		return nil, err
	}

	// Read data directly
	buf := make([]byte, length)
	n, err := b.file.Read(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to read data: %w", err)
	}
	if n < length {
		return nil, fmt.Errorf("short read: got %d bytes, expected %d", n, length)
	}

	return buf, nil
}

// ProbeDevice checks if a device responds at the given address.
// Returns true if the device acknowledges, false otherwise.
func (b *I2CBus) ProbeDevice(address uint8) bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Set slave address
	if err := b.setSlaveAddress(address); err != nil {
		return false
	}

	// Try to read 1 byte
	buf := make([]byte, 1)
	_, err := b.file.Read(buf)

	return err == nil
}

// WriteRegister writes data to a specific register.
func (b *I2CBus) WriteRegister(address uint8, register uint8, data []byte) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Set slave address
	if err := b.setSlaveAddress(address); err != nil {
		return err
	}

	// Combine register address and data
	writeData := make([]byte, 1+len(data))
	writeData[0] = register
	copy(writeData[1:], data)

	// Write data
	n, err := b.file.Write(writeData)
	if err != nil {
		return fmt.Errorf("failed to write data: %w", err)
	}
	if n < len(writeData) {
		return fmt.Errorf("short write: wrote %d bytes, expected %d", n, len(writeData))
	}

	return nil
}

// ReadDevice reads data from a device according to the configuration.
// If readRegister >= 0, performs a register read; otherwise performs a raw read.
func (b *I2CBus) ReadDevice(config *DeviceConfig) ([]byte, error) {
	if config.ReadRegister >= 0 {
		return b.ReadRegister(config.Address, uint8(config.ReadRegister), config.ReadLength)
	}
	return b.ReadRaw(config.Address, config.ReadLength)
}

