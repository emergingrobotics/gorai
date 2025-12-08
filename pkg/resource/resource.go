package resource

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// Resource is the base interface for all Gorai components and services.
type Resource interface {
	// Name returns the unique resource identifier.
	Name() Name

	// Reconfigure updates the resource with new configuration.
	// This should be safe to call while the resource is running.
	Reconfigure(ctx context.Context, deps Dependencies, conf Config) error

	// DoCommand executes arbitrary commands for extensibility.
	// This allows custom functionality beyond the standard interface.
	DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error)

	// Close releases all resources and stops background operations.
	Close(ctx context.Context) error
}

// Sensor represents resources that provide readings.
type Sensor interface {
	Resource

	// Readings returns the current sensor readings as key-value pairs.
	// The keys and value types depend on the specific sensor implementation.
	Readings(ctx context.Context) (map[string]any, error)
}

// Actuator represents resources that can move or perform physical actions.
type Actuator interface {
	Resource

	// IsMoving returns true if the actuator is currently in motion.
	IsMoving(ctx context.Context) (bool, error)

	// Stop halts all motion immediately.
	Stop(ctx context.Context) error
}

// Power represents resources that manage energy storage and distribution.
// Examples: batteries, power supplies, power distribution units.
type Power interface {
	Resource

	// GetCapacity returns total capacity in watt-hours (Wh).
	GetCapacity(ctx context.Context) (float64, error)

	// GetLevel returns current charge level as a ratio (0.0 - 1.0).
	GetLevel(ctx context.Context) (float64, error)

	// GetVoltage returns current voltage in volts.
	GetVoltage(ctx context.Context) (float64, error)

	// GetCurrent returns current draw in amps (positive = discharging).
	GetCurrent(ctx context.Context) (float64, error)

	// IsCharging returns true if currently charging.
	IsCharging(ctx context.Context) (bool, error)
}

// Bounds represents a 3D bounding box.
type Bounds struct {
	MinX, MinY, MinZ float64
	MaxX, MaxY, MaxZ float64
}

// Space represents physical volumes that can contain things.
// Examples: containers, cargo bays, workspaces, zones.
type Space interface {
	Resource

	// GetVolume returns volume in cubic meters.
	GetVolume(ctx context.Context) (float64, error)

	// GetBounds returns the bounding box geometry.
	GetBounds(ctx context.Context) (*Bounds, error)

	// GetContents returns identifiers of what's currently in this space.
	GetContents(ctx context.Context) ([]string, error)

	// IsEmpty returns true if space contains nothing.
	IsEmpty(ctx context.Context) (bool, error)
}

// LinkType identifies the transport mechanism of a link.
type LinkType int

const (
	LinkTypeSerial LinkType = iota
	LinkTypeIP
	LinkTypeNATS
	LinkTypeCAN
	LinkTypeI2C
	LinkTypeSPI
)

// String returns the string representation of a LinkType.
func (lt LinkType) String() string {
	switch lt {
	case LinkTypeSerial:
		return "serial"
	case LinkTypeIP:
		return "ip"
	case LinkTypeNATS:
		return "nats"
	case LinkTypeCAN:
		return "can"
	case LinkTypeI2C:
		return "i2c"
	case LinkTypeSPI:
		return "spi"
	default:
		return "unknown"
	}
}

// LinkDirection identifies whether a link is point-to-point or broadcast.
type LinkDirection int

const (
	// LinkBidirectional is point-to-point, request/response communication.
	LinkBidirectional LinkDirection = iota
	// LinkBroadcast is one-to-many, pub/sub communication.
	LinkBroadcast
)

// String returns the string representation of a LinkDirection.
func (ld LinkDirection) String() string {
	switch ld {
	case LinkBidirectional:
		return "bidirectional"
	case LinkBroadcast:
		return "broadcast"
	default:
		return "unknown"
	}
}

// LinkStats contains statistics about a communication link.
type LinkStats struct {
	BytesSent     uint64
	BytesReceived uint64
	MessagesSent  uint64
	MessagesRecv  uint64
	ErrorCount    uint64
	Latency       time.Duration
}

// Link represents a communication link between nodes.
// Links can be bidirectional (serial, TCP) or broadcast (NATS, CAN).
type Link interface {
	Resource

	// Type returns the link transport type.
	Type() LinkType

	// Direction returns whether the link is bidirectional or broadcast.
	Direction() LinkDirection

	// IsConnected returns true if the link is active.
	IsConnected(ctx context.Context) (bool, error)

	// GetStats returns link statistics.
	GetStats(ctx context.Context) (*LinkStats, error)
}

// Typed is an optional interface for resources that can report their type info.
type Typed interface {
	// Type returns the resource subtype (e.g., "motor", "camera").
	Type() string
}

// Dependencies provides access to dependent resources.
type Dependencies interface {
	// Get returns a resource by name.
	Get(name Name) (Resource, error)

	// GetByType returns all resources of a given subtype.
	GetByType(subtype string) ([]Resource, error)

	// All returns all available resources.
	All() []Resource
}

// Config holds resource configuration.
type Config struct {
	// Attributes holds typed configuration key-value pairs.
	Attributes map[string]any

	// Raw holds the original JSON configuration bytes for custom parsing.
	Raw []byte
}

// NewConfig creates a new Config from a map of attributes.
func NewConfig(attrs map[string]any) Config {
	raw, _ := json.Marshal(attrs)
	return Config{
		Attributes: attrs,
		Raw:        raw,
	}
}

// NewConfigFromJSON creates a new Config from JSON bytes.
func NewConfigFromJSON(data []byte) (Config, error) {
	var attrs map[string]any
	if err := json.Unmarshal(data, &attrs); err != nil {
		return Config{}, fmt.Errorf("failed to parse config JSON: %w", err)
	}
	return Config{
		Attributes: attrs,
		Raw:        data,
	}, nil
}

// Unmarshal decodes the config into a struct.
func (c Config) Unmarshal(v any) error {
	if len(c.Raw) == 0 {
		// If no raw data, marshal attributes first
		data, err := json.Marshal(c.Attributes)
		if err != nil {
			return fmt.Errorf("failed to marshal attributes: %w", err)
		}
		c.Raw = data
	}
	if err := json.Unmarshal(c.Raw, v); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	return nil
}

// Get returns a configuration value by key.
func (c Config) Get(key string) (any, bool) {
	v, ok := c.Attributes[key]
	return v, ok
}

// GetString returns a string configuration value.
func (c Config) GetString(key string) (string, bool) {
	v, ok := c.Attributes[key]
	if !ok {
		return "", false
	}
	s, ok := v.(string)
	return s, ok
}

// GetInt returns an integer configuration value.
func (c Config) GetInt(key string) (int, bool) {
	v, ok := c.Attributes[key]
	if !ok {
		return 0, false
	}
	// JSON numbers are float64
	switch n := v.(type) {
	case float64:
		return int(n), true
	case int:
		return n, true
	case int64:
		return int(n), true
	default:
		return 0, false
	}
}

// GetFloat returns a float configuration value.
func (c Config) GetFloat(key string) (float64, bool) {
	v, ok := c.Attributes[key]
	if !ok {
		return 0, false
	}
	f, ok := v.(float64)
	return f, ok
}

// GetBool returns a boolean configuration value.
func (c Config) GetBool(key string) (bool, bool) {
	v, ok := c.Attributes[key]
	if !ok {
		return false, false
	}
	b, ok := v.(bool)
	return b, ok
}

// simpleDeps is a simple implementation of Dependencies.
type simpleDeps struct {
	resources map[string]Resource
}

// NewSimpleDependencies creates a Dependencies from a map of resources.
func NewSimpleDependencies(resources map[string]Resource) Dependencies {
	return &simpleDeps{resources: resources}
}

// EmptyDependencies returns an empty Dependencies.
func EmptyDependencies() Dependencies {
	return &simpleDeps{resources: make(map[string]Resource)}
}

func (d *simpleDeps) Get(name Name) (Resource, error) {
	r, ok := d.resources[name.String()]
	if !ok {
		return nil, fmt.Errorf("resource not found: %s", name.String())
	}
	return r, nil
}

func (d *simpleDeps) GetByType(subtype string) ([]Resource, error) {
	var result []Resource
	for _, r := range d.resources {
		if r.Name().Subtype == subtype {
			result = append(result, r)
		}
	}
	return result, nil
}

func (d *simpleDeps) All() []Resource {
	result := make([]Resource, 0, len(d.resources))
	for _, r := range d.resources {
		result = append(result, r)
	}
	return result
}
