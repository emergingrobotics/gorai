package hal

import (
	"fmt"
	"strconv"
	"strings"
)

// PinRef represents a board-agnostic pin reference.
type PinRef struct {
	// GPIO is the GPIO number (BCM on Pi, line number on others)
	GPIO int

	// Name is a named reference like "GPIO17", "SDA1", "LED"
	Name string

	// Physical is the physical header pin number
	Physical int
}

// ParsePinRef parses a pin reference from an RDL config value.
// Accepts: int, float64, string (e.g., "GPIO17", "PIN11", "17", "SDA1")
func ParsePinRef(v any) (PinRef, error) {
	switch val := v.(type) {
	case int:
		return PinRef{GPIO: val}, nil
	case float64:
		return PinRef{GPIO: int(val)}, nil
	case string:
		return parsePinString(val)
	default:
		return PinRef{}, fmt.Errorf("%w: unsupported type %T", ErrInvalidPinRef, v)
	}
}

func parsePinString(s string) (PinRef, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return PinRef{}, fmt.Errorf("%w: empty string", ErrInvalidPinRef)
	}

	upper := strings.ToUpper(s)

	// Check for physical pin format "PIN11" or "PHYSICAL11"
	if strings.HasPrefix(upper, "PIN") {
		n, err := strconv.Atoi(s[3:])
		if err != nil {
			return PinRef{}, fmt.Errorf("%w: invalid physical pin %q", ErrInvalidPinRef, s)
		}
		return PinRef{Physical: n}, nil
	}
	if strings.HasPrefix(upper, "PHYSICAL") {
		n, err := strconv.Atoi(s[8:])
		if err != nil {
			return PinRef{}, fmt.Errorf("%w: invalid physical pin %q", ErrInvalidPinRef, s)
		}
		return PinRef{Physical: n}, nil
	}

	// Check for pure number
	if n, err := strconv.Atoi(s); err == nil {
		return PinRef{GPIO: n}, nil
	}

	// Check for GPIO number "GPIO17"
	if strings.HasPrefix(upper, "GPIO") {
		n, err := strconv.Atoi(s[4:])
		if err != nil {
			return PinRef{}, fmt.Errorf("%w: invalid GPIO number %q", ErrInvalidPinRef, s)
		}
		return PinRef{GPIO: n}, nil
	}

	// Treat as named reference (e.g., "SDA1", "SCL1", "LED")
	return PinRef{Name: s}, nil
}

// String returns a string representation of the pin reference.
func (p PinRef) String() string {
	if p.Physical > 0 {
		return fmt.Sprintf("PIN%d", p.Physical)
	}
	if p.Name != "" {
		return p.Name
	}
	return fmt.Sprintf("GPIO%d", p.GPIO)
}

// IsZero returns true if the PinRef is unset.
func (p PinRef) IsZero() bool {
	return p.GPIO == 0 && p.Name == "" && p.Physical == 0
}

// PinMapper resolves pin references to GPIO numbers for a specific board.
type PinMapper interface {
	// Resolve converts a PinRef to a GPIO line number.
	Resolve(ref PinRef) (int, error)

	// PhysicalToGPIO converts a physical header pin to GPIO number.
	PhysicalToGPIO(physical int) (int, error)

	// NameToGPIO converts a named pin to GPIO number.
	NameToGPIO(name string) (int, error)
}

// GetPinMapper returns the appropriate pin mapper for a board.
func GetPinMapper(board Board) PinMapper {
	switch board {
	case BoardRaspberryPi5:
		return &rpi5PinMapper{}
	default:
		return &genericPinMapper{}
	}
}

// genericPinMapper is the fallback pin mapper (pass-through for GPIO numbers).
type genericPinMapper struct{}

func (m *genericPinMapper) Resolve(ref PinRef) (int, error) {
	if ref.Physical > 0 {
		return 0, fmt.Errorf("%w: physical pin references require board-specific mapping", ErrPinNotFound)
	}
	if ref.Name != "" {
		return m.NameToGPIO(ref.Name)
	}
	return ref.GPIO, nil
}

func (m *genericPinMapper) PhysicalToGPIO(physical int) (int, error) {
	return 0, fmt.Errorf("%w: physical pin %d not supported on generic Linux", ErrPinNotFound, physical)
}

func (m *genericPinMapper) NameToGPIO(name string) (int, error) {
	upper := strings.ToUpper(name)
	if strings.HasPrefix(upper, "GPIO") {
		n, err := strconv.Atoi(name[4:])
		if err != nil {
			return 0, fmt.Errorf("%w: invalid GPIO name %q", ErrPinNotFound, name)
		}
		return n, nil
	}
	return 0, fmt.Errorf("%w: named pin %q requires board-specific mapping", ErrPinNotFound, name)
}
