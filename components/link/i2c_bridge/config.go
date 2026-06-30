package i2c_bridge

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emergingrobotics/gorai/pkg/resource"
)

// Config holds the configuration for the I2C bridge component.
type Config struct {
	// Bus is the I2C bus number (e.g., 1 for /dev/i2c-1).
	// Replaces the old "device" field which was board-specific.
	Bus int `json:"bus"`

	// Devices is the list of I2C devices to poll.
	Devices []DeviceConfig `json:"devices"`
}

// DeviceConfig holds the configuration for a single I2C device.
type DeviceConfig struct {
	// Name is a unique identifier for this device.
	Name string `json:"name"`

	// Address is the 7-bit I2C address (0-127).
	Address uint8 `json:"address"`

	// ReadRegister is the starting register address to read.
	// Use -1 for raw read (no register address written first).
	ReadRegister int `json:"read_register"`

	// ReadLength is the number of bytes to read per poll.
	ReadLength int `json:"read_length"`

	// PollRateHz is the polling frequency in Hz.
	PollRateHz float64 `json:"poll_rate_hz"`

	// Enabled indicates whether to poll this device.
	Enabled bool `json:"enabled"`
}

// NewConfigFromResource parses a resource.Config into a Config.
func NewConfigFromResource(conf resource.Config) (*Config, error) {
	cfg := &Config{
		Bus:     1, // Default to bus 1
		Devices: []DeviceConfig{},
	}

	// Parse bus number (new field)
	if bus, ok := conf.Attributes["bus"].(float64); ok {
		cfg.Bus = int(bus)
	} else if bus, ok := conf.Attributes["bus"].(int); ok {
		cfg.Bus = bus
	}

	// Legacy support: parse device path and extract bus number
	if device, ok := conf.Attributes["device"].(string); ok && device != "" {
		busID, err := ParseBusID(device)
		if err == nil {
			cfg.Bus = busID
		}
	}

	// Parse devices array (required)
	devicesRaw, ok := conf.Attributes["devices"].([]any)
	if !ok {
		return nil, fmt.Errorf("devices configuration is required")
	}

	for i, devRaw := range devicesRaw {
		devMap, ok := devRaw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("device[%d]: invalid format", i)
		}

		dev, err := parseDeviceConfig(devMap, i)
		if err != nil {
			return nil, err
		}
		cfg.Devices = append(cfg.Devices, *dev)
	}

	return cfg, nil
}

func parseDeviceConfig(m map[string]any, index int) (*DeviceConfig, error) {
	cfg := &DeviceConfig{
		ReadRegister: -1,
		PollRateHz:   100.0,
		Enabled:      true,
	}

	// Required fields
	if name, ok := m["name"].(string); ok {
		cfg.Name = name
	} else {
		return nil, fmt.Errorf("device[%d]: name is required", index)
	}

	// Address can be float64 (JSON) or int
	switch addr := m["address"].(type) {
	case float64:
		cfg.Address = uint8(addr)
	case int:
		cfg.Address = uint8(addr)
	default:
		return nil, fmt.Errorf("device[%d]: address is required", index)
	}

	// Read length (required)
	switch length := m["read_length"].(type) {
	case float64:
		cfg.ReadLength = int(length)
	case int:
		cfg.ReadLength = length
	default:
		return nil, fmt.Errorf("device[%d]: read_length is required", index)
	}

	// Optional fields
	switch reg := m["read_register"].(type) {
	case float64:
		cfg.ReadRegister = int(reg)
	case int:
		cfg.ReadRegister = reg
	}

	if rate, ok := m["poll_rate_hz"].(float64); ok {
		cfg.PollRateHz = rate
	}

	if enabled, ok := m["enabled"].(bool); ok {
		cfg.Enabled = enabled
	}

	return cfg, nil
}

// Validate checks if the configuration is valid.
func (c *Config) Validate() error {
	if c.Bus < 0 {
		return fmt.Errorf("bus number must be non-negative")
	}

	if len(c.Devices) == 0 {
		return fmt.Errorf("at least one device is required")
	}

	deviceNames := make(map[string]bool)
	deviceAddrs := make(map[uint8]string)

	for i, dev := range c.Devices {
		// Validate required fields
		if dev.Name == "" {
			return fmt.Errorf("device[%d]: name is required", i)
		}

		// Validate address range (7-bit: 0-127)
		if dev.Address > 127 {
			return fmt.Errorf("device[%d]: address %d out of range (0-127)", i, dev.Address)
		}

		// Validate read length
		if dev.ReadLength <= 0 {
			return fmt.Errorf("device[%d]: read_length must be positive", i)
		}
		if dev.ReadLength > 256 {
			return fmt.Errorf("device[%d]: read_length %d exceeds maximum (256)", i, dev.ReadLength)
		}

		// Validate register address
		if dev.ReadRegister < -1 || dev.ReadRegister > 255 {
			return fmt.Errorf("device[%d]: read_register %d out of range (-1 to 255)", i, dev.ReadRegister)
		}

		// Validate poll rate
		if dev.PollRateHz <= 0 {
			return fmt.Errorf("device[%d]: poll_rate_hz must be positive", i)
		}
		if dev.PollRateHz > 1000 {
			return fmt.Errorf("device[%d]: poll_rate_hz %f exceeds maximum (1000)", i, dev.PollRateHz)
		}

		// Check for duplicate names
		if deviceNames[dev.Name] {
			return fmt.Errorf("device[%d]: duplicate device name %q", i, dev.Name)
		}
		deviceNames[dev.Name] = true

		// Check for duplicate addresses
		if existing, ok := deviceAddrs[dev.Address]; ok {
			return fmt.Errorf("device[%d]: address 0x%02X already used by device %q", i, dev.Address, existing)
		}
		deviceAddrs[dev.Address] = dev.Name
	}

	return nil
}

// ParseBusID extracts the bus number from a device path like "/dev/i2c-1".
func ParseBusID(devicePath string) (int, error) {
	// Expected format: /dev/i2c-N
	parts := strings.Split(devicePath, "-")
	if len(parts) < 2 {
		return 0, fmt.Errorf("invalid device path format: %s", devicePath)
	}

	busID, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 0, fmt.Errorf("failed to parse bus ID from %s: %w", devicePath, err)
	}

	return busID, nil
}

