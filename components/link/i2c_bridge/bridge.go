// Package i2c_bridge provides an I2C bridge component that reads raw data
// from I2C devices and publishes it to the NATS message bus.
package i2c_bridge

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorai/gorai/driver/hal"
	"github.com/gorai/gorai/driver/i2c"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("link", "i2c_bridge", New)
}

// State represents the operational state of the bridge.
type State int

const (
	StateClosed State = iota
	StateOpening
	StateRunning
	StateError
)

// String returns the string representation of the state.
func (s State) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpening:
		return "opening"
	case StateRunning:
		return "running"
	case StateError:
		return "error"
	default:
		return "unknown"
	}
}

// DeviceState holds the runtime state of a single I2C device.
type DeviceState struct {
	Name            string
	Address         uint8
	Enabled         bool
	Connected       bool
	ReadsSuccessful uint64
	ReadsFailed     uint64
	LastReadTime    time.Time
	ActualRateHz    float64
}

// Bridge implements the I2C bridge component.
type Bridge struct {
	name   resource.Name
	config *Config
	logger *slog.Logger

	hal hal.HAL
	bus i2c.Bus // HAL-provided I2C bus interface

	mu           sync.RWMutex
	state        State
	errorMsg     string
	deviceStates map[string]*DeviceState

	stopCh chan struct{}
	wg     sync.WaitGroup

	// Metrics
	totalReads        atomic.Uint64
	totalErrors       atomic.Uint64
	messagesPublished atomic.Uint64

	// Data callback for publishing (set by caller or NATS integration)
	onData func(deviceName string, data []byte, address uint8, register int)
}

// New creates a new I2C bridge component.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	// Convert registry.Config to resource.Config
	resConf := resource.NewConfig(conf)

	cfg, err := NewConfigFromResource(resConf)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	// Get name from config
	name := "i2c_bridge"
	if n, ok := conf["name"].(string); ok {
		name = n
	}

	// Get HAL from dependencies
	halAny, err := deps.Get("hal")
	if err != nil {
		return nil, fmt.Errorf("HAL not available: %w (ensure platform section is configured)", err)
	}
	h, ok := halAny.(hal.HAL)
	if !ok {
		return nil, fmt.Errorf("invalid HAL type: %T", halAny)
	}

	// Get I2C bus from HAL
	bus, err := h.I2C(cfg.Bus)
	if err != nil {
		return nil, fmt.Errorf("failed to open I2C bus %d: %w", cfg.Bus, err)
	}

	b := &Bridge{
		name:         resource.NewComponentName("gorai", "link", name),
		config:       cfg,
		logger:       slog.Default().With("component", "i2c_bridge", "name", name),
		hal:          h,
		bus:          bus,
		state:        StateClosed,
		deviceStates: make(map[string]*DeviceState),
		stopCh:       make(chan struct{}),
	}

	// Initialize device states
	for _, dev := range cfg.Devices {
		b.deviceStates[dev.Name] = &DeviceState{
			Name:    dev.Name,
			Address: dev.Address,
			Enabled: dev.Enabled,
		}
	}

	// Open the I2C bus
	if err := b.open(ctx); err != nil {
		return nil, fmt.Errorf("failed to open I2C bridge: %w", err)
	}

	return b, nil
}

// open initializes the I2C bus and starts polling.
func (b *Bridge) open(ctx context.Context) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	b.state = StateOpening

	// Probe devices using HAL's I2C bus
	for _, dev := range b.config.Devices {
		state := b.deviceStates[dev.Name]

		// Try to read one byte to probe device
		device := b.bus.Device(uint16(dev.Address))
		_, err := device.Read(ctx, 1)
		if err == nil {
			state.Connected = true
			b.logger.Info("device found", "name", dev.Name, "address", fmt.Sprintf("0x%02X", dev.Address))
		} else {
			state.Connected = false
			b.logger.Warn("device not found", "name", dev.Name, "address", fmt.Sprintf("0x%02X", dev.Address))
		}
	}

	b.state = StateRunning
	b.errorMsg = ""

	// Start polling goroutines for each enabled device
	for i := range b.config.Devices {
		dev := &b.config.Devices[i]
		if dev.Enabled {
			b.wg.Add(1)
			go b.pollDevice(ctx, dev)
		}
	}

	b.logger.Info("I2C bridge started", "bus", b.config.Bus, "board", b.hal.Board())
	return nil
}

// pollDevice polls a single I2C device at the configured rate.
func (b *Bridge) pollDevice(ctx context.Context, dev *DeviceConfig) {
	defer b.wg.Done()

	interval := time.Duration(float64(time.Second) / dev.PollRateHz)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Get device handle from bus
	device := b.bus.Device(uint16(dev.Address))

	var lastRead time.Time

	for {
		select {
		case <-b.stopCh:
			return

		case <-ticker.C:
			// Check if device is still enabled
			b.mu.RLock()
			state := b.deviceStates[dev.Name]
			enabled := state.Enabled
			b.mu.RUnlock()

			if !enabled {
				continue
			}

			// Read from device using HAL's i2c.Device interface
			var data []byte
			var err error

			if dev.ReadRegister >= 0 {
				// Register read
				data, err = device.ReadReg(ctx, byte(dev.ReadRegister), dev.ReadLength)
			} else {
				// Raw read
				data, err = device.Read(ctx, dev.ReadLength)
			}

			b.mu.Lock()
			if err != nil {
				state.ReadsFailed++
				state.Connected = false
				b.totalErrors.Add(1)
				b.mu.Unlock()

				b.logger.Debug("read failed", "device", dev.Name, "error", err)
				continue
			}

			state.ReadsSuccessful++
			state.Connected = true
			state.LastReadTime = time.Now()

			// Calculate actual rate
			if !lastRead.IsZero() {
				elapsed := time.Since(lastRead)
				state.ActualRateHz = float64(time.Second) / float64(elapsed)
			}
			lastRead = time.Now()
			b.mu.Unlock()

			b.totalReads.Add(1)

			// Call data callback if set
			if b.onData != nil {
				b.onData(dev.Name, data, dev.Address, dev.ReadRegister)
				b.messagesPublished.Add(1)
			}
		}
	}
}

// SetDataCallback sets the callback function for received data.
// This is called for each successful read with the device name, raw data,
// address, and register.
func (b *Bridge) SetDataCallback(callback func(deviceName string, data []byte, address uint8, register int)) {
	b.onData = callback
}

// Name returns the resource name.
func (b *Bridge) Name() resource.Name {
	return b.name
}

// Reconfigure updates the bridge configuration.
func (b *Bridge) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	cfg, err := NewConfigFromResource(conf)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// For now, require restart for config changes
	// A more sophisticated implementation would handle dynamic reconfiguration
	return fmt.Errorf("reconfiguration not supported, restart required")
}

// DoCommand handles arbitrary commands.
func (b *Bridge) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	cmdName, _ := cmd["command"].(string)

	switch cmdName {
	case "get_state":
		b.mu.RLock()
		state := b.state.String()
		b.mu.RUnlock()
		return map[string]any{"state": state}, nil

	case "get_device_state":
		deviceName, _ := cmd["device"].(string)

		b.mu.RLock()
		defer b.mu.RUnlock()

		state := b.deviceStates[deviceName]
		if state == nil {
			return nil, fmt.Errorf("device %q not found", deviceName)
		}

		return map[string]any{
			"name":             state.Name,
			"address":          fmt.Sprintf("0x%02X", state.Address),
			"enabled":          state.Enabled,
			"connected":        state.Connected,
			"reads_successful": state.ReadsSuccessful,
			"reads_failed":     state.ReadsFailed,
			"actual_rate_hz":   state.ActualRateHz,
		}, nil

	case "enable_device":
		deviceName, _ := cmd["device"].(string)

		b.mu.Lock()
		defer b.mu.Unlock()

		state := b.deviceStates[deviceName]
		if state == nil {
			return nil, fmt.Errorf("device %q not found", deviceName)
		}

		state.Enabled = true
		return map[string]any{"success": true}, nil

	case "disable_device":
		deviceName, _ := cmd["device"].(string)

		b.mu.Lock()
		defer b.mu.Unlock()

		state := b.deviceStates[deviceName]
		if state == nil {
			return nil, fmt.Errorf("device %q not found", deviceName)
		}

		state.Enabled = false
		return map[string]any{"success": true}, nil

	case "probe_bus":
		// Scan the bus for devices (addresses 0x08 - 0x77)
		found := []string{}
		for addr := uint8(0x08); addr <= 0x77; addr++ {
			device := b.bus.Device(uint16(addr))
			_, err := device.Read(ctx, 1)
			if err == nil {
				found = append(found, fmt.Sprintf("0x%02X", addr))
			}
		}
		return map[string]any{"devices": found}, nil

	case "read_once":
		deviceName, _ := cmd["device"].(string)

		b.mu.RLock()
		var dev *DeviceConfig
		for i := range b.config.Devices {
			if b.config.Devices[i].Name == deviceName {
				dev = &b.config.Devices[i]
				break
			}
		}
		b.mu.RUnlock()

		if dev == nil {
			return nil, fmt.Errorf("device %q not found", deviceName)
		}

		device := b.bus.Device(uint16(dev.Address))
		var data []byte
		var err error

		if dev.ReadRegister >= 0 {
			data, err = device.ReadReg(ctx, byte(dev.ReadRegister), dev.ReadLength)
		} else {
			data, err = device.Read(ctx, dev.ReadLength)
		}

		if err != nil {
			return nil, fmt.Errorf("read failed: %w", err)
		}

		return map[string]any{
			"device": deviceName,
			"data":   data,
			"length": len(data),
		}, nil

	case "get_stats":
		b.mu.RLock()
		connectedDevices := []string{}
		failedDevices := []string{}
		for name, state := range b.deviceStates {
			if state.Connected {
				connectedDevices = append(connectedDevices, name)
			} else {
				failedDevices = append(failedDevices, name)
			}
		}
		b.mu.RUnlock()

		return map[string]any{
			"total_reads":        b.totalReads.Load(),
			"total_errors":       b.totalErrors.Load(),
			"messages_published": b.messagesPublished.Load(),
			"connected_devices":  connectedDevices,
			"failed_devices":     failedDevices,
		}, nil

	default:
		return nil, fmt.Errorf("unknown command: %s", cmdName)
	}
}

// Close releases all resources.
func (b *Bridge) Close(ctx context.Context) error {
	b.mu.Lock()
	if b.state == StateClosed {
		b.mu.Unlock()
		return nil
	}
	b.state = StateClosed
	b.mu.Unlock()

	// Signal all goroutines to stop
	close(b.stopCh)

	// Wait for all goroutines to finish
	b.wg.Wait()

	// Note: I2C bus cleanup is handled by HAL.Close()
	// Don't close the bus here as other components may share it

	b.logger.Info("I2C bridge closed")
	return nil
}

// Type returns the link type.
func (b *Bridge) Type() resource.LinkType {
	return resource.LinkTypeI2C
}

// Direction returns the link direction.
func (b *Bridge) Direction() resource.LinkDirection {
	return resource.LinkBidirectional
}

// IsConnected returns true if at least one device is connected.
func (b *Bridge) IsConnected(ctx context.Context) (bool, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	for _, state := range b.deviceStates {
		if state.Connected {
			return true, nil
		}
	}
	return false, nil
}

// GetStats returns link statistics.
func (b *Bridge) GetStats(ctx context.Context) (*resource.LinkStats, error) {
	return &resource.LinkStats{
		BytesSent:     0, // I2C bridge doesn't track bytes sent
		BytesReceived: b.totalReads.Load() * 14, // Approximate
		MessagesSent:  0,
		MessagesRecv:  b.totalReads.Load(),
		ErrorCount:    b.totalErrors.Load(),
	}, nil
}

// GetState returns the current operational state.
func (b *Bridge) GetState() State {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.state
}

// GetDeviceStates returns the current state of all devices.
func (b *Bridge) GetDeviceStates() map[string]*DeviceState {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Return a copy
	result := make(map[string]*DeviceState)
	for name, state := range b.deviceStates {
		copy := *state
		result[name] = &copy
	}
	return result
}

// GetBusID returns the I2C bus ID.
func (b *Bridge) GetBusID() int {
	return b.config.Bus
}

