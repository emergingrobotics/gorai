package component_test

import (
	"testing"

	"github.com/gorai/gorai/components"
	"github.com/gorai/gorai/pkg/resource"
)

// TestComponent_IsResource verifies that Component embeds resource.Resource.
func TestComponent_IsResource(t *testing.T) {
	// This is a compile-time check that Component embeds resource.Resource.
	// If this compiles, the interface relationship is correct.
	var _ resource.Resource = (component.Component)(nil)
}

// TestActuator_IsResource verifies that Actuator embeds resource.Actuator.
func TestActuator_IsResource(t *testing.T) {
	// Actuator must implement both Component (which is resource.Resource)
	// and resource.Actuator (which also includes resource.Resource).
	var _ resource.Actuator = (component.Actuator)(nil)
	var _ component.Component = (component.Actuator)(nil)
}

// TestSensor_IsResource verifies that Sensor embeds resource.Sensor.
func TestSensor_IsResource(t *testing.T) {
	// Sensor must implement both Component and resource.Sensor.
	var _ resource.Sensor = (component.Sensor)(nil)
	var _ component.Component = (component.Sensor)(nil)
}
