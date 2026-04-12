// Package discovery provides dynamic device discovery and auto-adoption.
//
// The discovery package enables robots to discover hardware at runtime
// that isn't explicitly defined in the RDL configuration. It works with
// gateways and the mesh to find devices, apply adoption rules, and create
// proxy components.
//
// # Key Concepts
//
//   - Sources: Where to find devices (gateways, mesh queries)
//   - Rules: How to map discovered capabilities to component types
//   - Adoption: Creating proxy components for discovered devices
//   - Dynamic Dependencies: Services depending on @discovered: resources
//
// # Usage
//
//	mgr := discovery.NewManager(meshClient, natsConn, config)
//	mgr.Start(ctx)
//	// Discovered devices are automatically adopted
//	adopted := mgr.GetAdopted()
package discovery

import (
	"time"

	"github.com/gorai/gorai/pkg/mesh"
)

// Config holds discovery configuration from RDL.
type Config struct {
	// Enabled enables discovery.
	Enabled bool `json:"enabled"`

	// AutoAdopt automatically adopts discovered devices.
	AutoAdopt bool `json:"auto_adopt"`

	// Sources defines where to discover devices.
	Sources []Source `json:"sources"`

	// Rules defines how to adopt discovered devices.
	Rules []Rule `json:"rules"`

	// ScanInterval is how often to scan for new devices (default: 5s).
	ScanInterval time.Duration `json:"scan_interval,omitempty"`
}

// DefaultConfig returns the default discovery configuration.
func DefaultConfig() *Config {
	return &Config{
		Enabled:      false,
		AutoAdopt:    true,
		ScanInterval: 5 * time.Second,
	}
}

// Source defines where to discover devices.
type Source struct {
	// Type is the source type: "gateway", "mesh".
	Type string `json:"type"`

	// Gateway is the gateway name (for type "gateway").
	Gateway string `json:"gateway,omitempty"`

	// Query is the mesh query (for type "mesh").
	Query *mesh.Query `json:"query,omitempty"`

	// AdoptAs is the default adoption type if no rule matches.
	AdoptAs string `json:"adopt_as,omitempty"`
}

// Rule defines how to adopt a discovered device.
type Rule struct {
	// Match specifies conditions for this rule.
	Match Match `json:"match"`

	// AdoptAs specifies how to adopt matching devices.
	AdoptAs AdoptAs `json:"adopt_as"`

	// Config provides additional configuration for adopted component.
	Config map[string]any `json:"config,omitempty"`

	// Enabled allows disabling rules (default: true).
	Enabled *bool `json:"enabled,omitempty"`
}

// IsEnabled returns whether the rule is enabled.
func (r Rule) IsEnabled() bool {
	return r.Enabled == nil || *r.Enabled
}

// Match defines conditions for a rule to apply.
type Match struct {
	// Capability matches devices with this capability.
	Capability string `json:"capability,omitempty"`

	// Subtype matches devices with this subtype.
	Subtype string `json:"subtype,omitempty"`

	// Model matches devices with this model.
	Model string `json:"model,omitempty"`

	// Metadata matches devices with these metadata key-value pairs.
	Metadata map[string]string `json:"metadata,omitempty"`

	// NamePattern matches device names (supports * wildcard).
	NamePattern string `json:"name_pattern,omitempty"`
}

// Matches returns true if the service descriptor matches this condition.
func (m Match) Matches(desc mesh.ServiceDescriptor) bool {
	if m.Capability != "" {
		caps, ok := desc.Metadata["capabilities"]
		if !ok || !containsCapability(caps, m.Capability) {
			return false
		}
	}

	if m.Subtype != "" && desc.Subtype != m.Subtype {
		return false
	}

	if m.Model != "" && desc.Model != m.Model {
		return false
	}

	for k, v := range m.Metadata {
		if desc.Metadata[k] != v {
			return false
		}
	}

	if m.NamePattern != "" && !matchPattern(desc.Name, m.NamePattern) {
		return false
	}

	return true
}

// containsCapability checks if caps (comma-separated) contains cap.
func containsCapability(caps, cap string) bool {
	for _, c := range splitCapabilities(caps) {
		if c == cap {
			return true
		}
	}
	return false
}

// splitCapabilities splits a comma-separated capability string.
func splitCapabilities(caps string) []string {
	if caps == "" {
		return nil
	}
	var result []string
	start := 0
	for i := 0; i <= len(caps); i++ {
		if i == len(caps) || caps[i] == ',' {
			s := caps[start:i]
			// Trim spaces
			for len(s) > 0 && s[0] == ' ' {
				s = s[1:]
			}
			for len(s) > 0 && s[len(s)-1] == ' ' {
				s = s[:len(s)-1]
			}
			if s != "" {
				result = append(result, s)
			}
			start = i + 1
		}
	}
	return result
}

// matchPattern matches a name against a pattern with * wildcard.
func matchPattern(name, pattern string) bool {
	if pattern == "*" {
		return true
	}
	if pattern == "" {
		return name == ""
	}

	// Simple wildcard matching
	pi := 0
	ni := 0
	starIdx := -1
	matchIdx := 0

	for ni < len(name) {
		if pi < len(pattern) && (pattern[pi] == name[ni] || pattern[pi] == '?') {
			pi++
			ni++
		} else if pi < len(pattern) && pattern[pi] == '*' {
			starIdx = pi
			matchIdx = ni
			pi++
		} else if starIdx != -1 {
			pi = starIdx + 1
			matchIdx++
			ni = matchIdx
		} else {
			return false
		}
	}

	for pi < len(pattern) && pattern[pi] == '*' {
		pi++
	}

	return pi == len(pattern)
}

// AdoptAs specifies how to adopt a discovered device.
type AdoptAs struct {
	// Type is the component type (e.g., "motor", "sensor").
	Type string `json:"type"`

	// Subtype is the component subtype (e.g., "imu", "gps").
	Subtype string `json:"subtype,omitempty"`

	// Model is the proxy model to use (e.g., "remote-pwm").
	Model string `json:"model,omitempty"`

	// Name overrides the component name (default: device name).
	Name string `json:"name,omitempty"`
}

// AdoptedResource represents a discovered and adopted resource.
type AdoptedResource struct {
	// ID is a unique identifier for this adopted resource.
	ID string `json:"id"`

	// Source is where this resource was discovered.
	Source string `json:"source"`

	// SourceType is the source type ("gateway" or "mesh").
	SourceType string `json:"source_type"`

	// Descriptor is the mesh service descriptor.
	Descriptor mesh.ServiceDescriptor `json:"descriptor"`

	// AdoptedAs is how this resource was adopted.
	AdoptedAs AdoptAs `json:"adopted_as"`

	// Config is the merged configuration.
	Config map[string]any `json:"config,omitempty"`

	// Proxy is the created proxy component (not serialized).
	Proxy any `json:"-"`

	// AdoptedAt is when this resource was adopted.
	AdoptedAt time.Time `json:"adopted_at"`
}

// ComponentType returns the full component type string.
func (a AdoptedResource) ComponentType() string {
	if a.AdoptedAs.Subtype != "" {
		return a.AdoptedAs.Type + "/" + a.AdoptedAs.Subtype
	}
	return a.AdoptedAs.Type
}

// ComponentName returns the component name.
func (a AdoptedResource) ComponentName() string {
	if a.AdoptedAs.Name != "" {
		return a.AdoptedAs.Name
	}
	return a.Descriptor.Name
}

// Event represents a discovery event.
type Event struct {
	// Type is the event type.
	Type EventType `json:"type"`

	// Resource is the affected resource (for adopt/remove events).
	Resource *AdoptedResource `json:"resource,omitempty"`

	// Service is the discovered service (for discover events).
	Service *mesh.ServiceDescriptor `json:"service,omitempty"`

	// Error is set if the event represents an error.
	Error error `json:"error,omitempty"`

	// Timestamp is when the event occurred.
	Timestamp time.Time `json:"timestamp"`
}

// EventType identifies the type of discovery event.
type EventType string

const (
	// EventDiscovered indicates a new device was discovered.
	EventDiscovered EventType = "discovered"

	// EventAdopted indicates a device was adopted.
	EventAdopted EventType = "adopted"

	// EventRemoved indicates an adopted device was removed.
	EventRemoved EventType = "removed"

	// EventError indicates an error occurred.
	EventError EventType = "error"
)

// DependencyPattern represents a parsed @discovered: dependency.
type DependencyPattern struct {
	// Type is the component type (e.g., "motor", "sensor").
	Type string

	// Subtype is the optional subtype (e.g., "imu").
	Subtype string

	// Name is the optional name pattern (e.g., "*", "left").
	Name string

	// Raw is the original pattern string.
	Raw string
}

// Matches returns true if the adopted resource matches this pattern.
func (p DependencyPattern) Matches(r AdoptedResource) bool {
	if p.Type != "*" && r.AdoptedAs.Type != p.Type {
		return false
	}

	if p.Subtype != "" && p.Subtype != "*" && r.AdoptedAs.Subtype != p.Subtype {
		return false
	}

	if p.Name != "" && p.Name != "*" && r.ComponentName() != p.Name {
		return false
	}

	return true
}

// ParseDependencyPattern parses a @discovered: dependency pattern.
// Format: @discovered:type/subtype/name or @discovered:type/*
func ParseDependencyPattern(pattern string) (*DependencyPattern, error) {
	const prefix = "@discovered:"
	if len(pattern) <= len(prefix) {
		return nil, nil // Not a discovered pattern
	}

	if pattern[:len(prefix)] != prefix {
		return nil, nil // Not a discovered pattern
	}

	rest := pattern[len(prefix):]
	dp := &DependencyPattern{Raw: pattern}

	// Parse type/subtype/name
	parts := splitPath(rest)

	if len(parts) >= 1 {
		dp.Type = parts[0]
	}
	if len(parts) >= 2 {
		// Could be subtype or name
		if len(parts) == 2 {
			// type/name format
			dp.Name = parts[1]
		} else {
			dp.Subtype = parts[1]
		}
	}
	if len(parts) >= 3 {
		dp.Name = parts[2]
	}

	if dp.Type == "" {
		dp.Type = "*"
	}

	return dp, nil
}

// splitPath splits a path by /
func splitPath(s string) []string {
	var parts []string
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == '/' {
			if i > start {
				parts = append(parts, s[start:i])
			}
			start = i + 1
		}
	}
	return parts
}
