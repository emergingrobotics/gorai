// Package resource provides the base resource interface and types for Gorai.
package resource

import (
	"fmt"
	"strings"
)

// Name identifies a resource with hierarchical naming.
type Name struct {
	// Namespace identifies the organization or scope (e.g., "gorai", "mycompany").
	Namespace string

	// Type is either "component" or "service".
	Type string

	// Subtype identifies the specific kind (e.g., "motor", "camera", "vision").
	Subtype string

	// Name is the instance name (e.g., "left_motor", "front_camera").
	Name string
}

// NewName creates a new resource name.
func NewName(namespace, typ, subtype, name string) Name {
	return Name{
		Namespace: namespace,
		Type:      typ,
		Subtype:   subtype,
		Name:      name,
	}
}

// NewComponentName creates a new component resource name.
func NewComponentName(namespace, subtype, name string) Name {
	return Name{
		Namespace: namespace,
		Type:      "component",
		Subtype:   subtype,
		Name:      name,
	}
}

// NewServiceName creates a new service resource name.
func NewServiceName(namespace, subtype, name string) Name {
	return Name{
		Namespace: namespace,
		Type:      "service",
		Subtype:   subtype,
		Name:      name,
	}
}

// String returns the full resource name in the format "namespace:type:subtype/name".
func (n Name) String() string {
	return fmt.Sprintf("%s:%s:%s/%s", n.Namespace, n.Type, n.Subtype, n.Name)
}

// Short returns just the subtype/name portion.
func (n Name) Short() string {
	return fmt.Sprintf("%s/%s", n.Subtype, n.Name)
}

// Topic returns a NATS-compatible topic string.
func (n Name) Topic() string {
	return fmt.Sprintf("%s.%s.%s.%s", n.Namespace, n.Type, n.Subtype, n.Name)
}

// Validate checks if the name has all required fields.
func (n Name) Validate() error {
	var missing []string
	if n.Namespace == "" {
		missing = append(missing, "namespace")
	}
	if n.Type == "" {
		missing = append(missing, "type")
	}
	if n.Subtype == "" {
		missing = append(missing, "subtype")
	}
	if n.Name == "" {
		missing = append(missing, "name")
	}
	if len(missing) > 0 {
		return fmt.Errorf("incomplete resource name, missing: %s", strings.Join(missing, ", "))
	}

	// Validate type
	if n.Type != "component" && n.Type != "service" {
		return fmt.Errorf("invalid resource type %q: must be 'component' or 'service'", n.Type)
	}

	return nil
}

// IsComponent returns true if this is a component resource.
func (n Name) IsComponent() bool {
	return n.Type == "component"
}

// IsService returns true if this is a service resource.
func (n Name) IsService() bool {
	return n.Type == "service"
}

// Equal returns true if two names are equal.
func (n Name) Equal(other Name) bool {
	return n.Namespace == other.Namespace &&
		n.Type == other.Type &&
		n.Subtype == other.Subtype &&
		n.Name == other.Name
}

// ParseName parses a resource name string in the format "namespace:type:subtype/name".
func ParseName(s string) (Name, error) {
	// Format: namespace:type:subtype/name
	parts := strings.SplitN(s, ":", 3)
	if len(parts) != 3 {
		return Name{}, fmt.Errorf("invalid resource name format %q: expected namespace:type:subtype/name", s)
	}

	namespace := parts[0]
	typ := parts[1]
	subtypeAndName := parts[2]

	subparts := strings.SplitN(subtypeAndName, "/", 2)
	if len(subparts) != 2 {
		return Name{}, fmt.Errorf("invalid resource name format %q: expected subtype/name after type", s)
	}

	name := Name{
		Namespace: namespace,
		Type:      typ,
		Subtype:   subparts[0],
		Name:      subparts[1],
	}

	if err := name.Validate(); err != nil {
		return Name{}, err
	}

	return name, nil
}
