package registry_test

import (
	"context"
	"testing"

	"github.com/emergingrobotics/gorai/pkg/registry"

	// Import fake implementations to trigger registrations
	_ "github.com/emergingrobotics/gorai/components/motor/fake"
	_ "github.com/emergingrobotics/gorai/components/sensor/fake"
	_ "github.com/emergingrobotics/gorai/components/servo/fake"
	_ "github.com/emergingrobotics/gorai/components/stepper/fake"
	_ "github.com/emergingrobotics/gorai/components/thruster/fake"
	_ "github.com/emergingrobotics/gorai/components/valve/fake"
)

func TestRegisteredSensorComponents(t *testing.T) {
	sensorTypes := []string{
		"imu",
		"ahrs",
		"gps",
		"encoder",
		"range_sensor",
		"lidar",
		"presence_sensor",
		"thermal_array",
		"force_sensor",
		"force_6dof",
		"current_sensor",
		"reflectance_sensor",
	}

	for _, sensorType := range sensorTypes {
		ctor, err := registry.LookupComponent(sensorType, "fake")
		if err != nil {
			t.Errorf("sensor %q not registered: %v", sensorType, err)
			continue
		}

		// Try to create an instance
		ctx := context.Background()
		conf := registry.Config{"name": "test"}
		_, err = ctor(ctx, nil, conf)
		if err != nil {
			t.Errorf("failed to create %q: %v", sensorType, err)
		}
	}
}

func TestRegisteredActuatorComponents(t *testing.T) {
	actuatorTypes := []string{
		"motor",
		"servo",
		"stepper",
		"thruster",
		"valve",
	}

	for _, actuatorType := range actuatorTypes {
		ctor, err := registry.LookupComponent(actuatorType, "fake")
		if err != nil {
			t.Errorf("actuator %q not registered: %v", actuatorType, err)
			continue
		}

		// Try to create an instance
		ctx := context.Background()
		conf := registry.Config{"name": "test"}
		_, err = ctor(ctx, nil, conf)
		if err != nil {
			t.Errorf("failed to create %q: %v", actuatorType, err)
		}
	}
}

func TestListComponents(t *testing.T) {
	components := registry.ListComponents()

	// Check that we have sensors
	expectedSensors := []string{"imu", "ahrs", "gps", "encoder"}
	for _, s := range expectedSensors {
		if _, ok := components[s]; !ok {
			t.Errorf("expected sensor %q in component list", s)
		}
	}

	// Check that we have actuators
	expectedActuators := []string{"motor", "servo", "stepper", "thruster", "valve"}
	for _, a := range expectedActuators {
		if _, ok := components[a]; !ok {
			t.Errorf("expected actuator %q in component list", a)
		}
	}
}

func TestLookupComponent_NotFound(t *testing.T) {
	_, err := registry.LookupComponent("nonexistent", "fake")
	if err == nil {
		t.Error("expected error for nonexistent component")
	}

	_, err = registry.LookupComponent("motor", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent model")
	}
}
