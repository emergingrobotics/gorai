package sensor_test

import (
	"context"
	"testing"

	"github.com/gorai/gorai/components/sensor"
	"github.com/gorai/gorai/components/sensor/fake"
	"github.com/gorai/gorai/pkg/resource"
)

// Interface compliance tests

func TestIMU_IsComponent(t *testing.T) {
	var _ sensor.IMU = (*fake.IMU)(nil)
}

func TestAHRS_IsIMU(t *testing.T) {
	var _ sensor.AHRS = (*fake.AHRS)(nil)
	var _ sensor.IMU = (*fake.AHRS)(nil)
}

func TestGPS_IsComponent(t *testing.T) {
	var _ sensor.GPS = (*fake.GPS)(nil)
}

func TestEncoder_IsComponent(t *testing.T) {
	var _ sensor.Encoder = (*fake.Encoder)(nil)
}

func TestRangeSensor_IsComponent(t *testing.T) {
	var _ sensor.RangeSensor = (*fake.RangeSensor)(nil)
}

func TestLiDAR_IsComponent(t *testing.T) {
	var _ sensor.LiDAR = (*fake.LiDAR)(nil)
}

func TestPresenceSensor_IsComponent(t *testing.T) {
	var _ sensor.PresenceSensor = (*fake.PresenceSensor)(nil)
}

func TestThermalArray_IsComponent(t *testing.T) {
	var _ sensor.ThermalArray = (*fake.ThermalArray)(nil)
}

func TestForceSensor_IsComponent(t *testing.T) {
	var _ sensor.ForceSensor = (*fake.ForceSensor)(nil)
}

func TestForce6DOF_IsForceSensor(t *testing.T) {
	var _ sensor.Force6DOF = (*fake.Force6DOF)(nil)
	var _ sensor.ForceSensor = (*fake.Force6DOF)(nil)
}

func TestCurrentSensor_IsComponent(t *testing.T) {
	var _ sensor.CurrentSensor = (*fake.CurrentSensor)(nil)
}

func TestReflectanceSensor_IsComponent(t *testing.T) {
	var _ sensor.ReflectanceSensor = (*fake.ReflectanceSensor)(nil)
}

// IMU tests

func TestFakeIMU_LinearAcceleration(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "imu", "test")
	imu := fake.NewIMUWithName(name)

	imu.SetAcceleration(1.0, 2.0, 9.81)

	x, y, z, err := imu.LinearAcceleration(ctx)
	if err != nil {
		t.Fatalf("LinearAcceleration failed: %v", err)
	}
	if x != 1.0 || y != 2.0 || z != 9.81 {
		t.Errorf("acceleration = (%v, %v, %v), want (1.0, 2.0, 9.81)", x, y, z)
	}
}

func TestFakeIMU_AngularVelocity(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "imu", "test")
	imu := fake.NewIMUWithName(name)

	imu.SetAngularVelocity(0.1, 0.2, 0.3)

	x, y, z, err := imu.AngularVelocity(ctx)
	if err != nil {
		t.Fatalf("AngularVelocity failed: %v", err)
	}
	if x != 0.1 || y != 0.2 || z != 0.3 {
		t.Errorf("angular velocity = (%v, %v, %v), want (0.1, 0.2, 0.3)", x, y, z)
	}
}

func TestFakeIMU_Orientation(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "imu", "test")
	imu := fake.NewIMUWithName(name)

	// Default is identity quaternion
	x, y, z, w, err := imu.Orientation(ctx)
	if err != nil {
		t.Fatalf("Orientation failed: %v", err)
	}
	if w != 1.0 || x != 0 || y != 0 || z != 0 {
		t.Errorf("orientation = (%v, %v, %v, %v), want identity quaternion", x, y, z, w)
	}
}

func TestFakeIMU_GetMagneticField(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "imu", "test")
	imu := fake.NewIMUWithName(name)

	imu.SetMagneticField(25.0, 10.0, 45.0)

	x, y, z, err := imu.GetMagneticField(ctx)
	if err != nil {
		t.Fatalf("GetMagneticField failed: %v", err)
	}
	if x != 25.0 || y != 10.0 || z != 45.0 {
		t.Errorf("magnetic field = (%v, %v, %v), want (25.0, 10.0, 45.0)", x, y, z)
	}
}

// GPS tests

func TestFakeGPS_Position(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "gps", "test")
	gps := fake.NewGPSWithName(name)

	gps.SetPosition(37.7749, -122.4194, 10.0)

	lat, lng, alt, err := gps.Position(ctx)
	if err != nil {
		t.Fatalf("Position failed: %v", err)
	}
	if lat != 37.7749 || lng != -122.4194 || alt != 10.0 {
		t.Errorf("position = (%v, %v, %v), want (37.7749, -122.4194, 10.0)", lat, lng, alt)
	}
}

func TestFakeGPS_Fix(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "gps", "test")
	gps := fake.NewGPSWithName(name)

	// Default is no fix
	fix, err := gps.Fix(ctx)
	if err != nil {
		t.Fatalf("Fix failed: %v", err)
	}
	if fix != sensor.FixNone {
		t.Errorf("fix = %v, want FixNone", fix)
	}

	// Set GPS fix with satellites
	gps.SetFix(sensor.FixGPS, 8)

	fix, err = gps.Fix(ctx)
	if err != nil {
		t.Fatalf("Fix failed: %v", err)
	}
	if fix != sensor.FixGPS {
		t.Errorf("fix = %v, want FixGPS", fix)
	}

	sats, err := gps.GetSatellitesUsed(ctx)
	if err != nil {
		t.Fatalf("GetSatellitesUsed failed: %v", err)
	}
	if sats != 8 {
		t.Errorf("satellites = %v, want 8", sats)
	}
}

func TestFakeGPS_Heading(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "gps", "test")
	gps := fake.NewGPSWithName(name)

	gps.SetHeading(45.5)

	heading, err := gps.GetHeading(ctx)
	if err != nil {
		t.Fatalf("GetHeading failed: %v", err)
	}
	if heading != 45.5 {
		t.Errorf("heading = %v, want 45.5", heading)
	}
}

// Encoder tests

func TestFakeEncoder_Position(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "encoder", "test")
	enc := fake.NewEncoderWithName(name)

	enc.SetPosition(1000)

	pos, err := enc.Position(ctx)
	if err != nil {
		t.Fatalf("Position failed: %v", err)
	}
	if pos != 1000 {
		t.Errorf("position = %v, want 1000", pos)
	}
}

func TestFakeEncoder_ResetPosition(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "encoder", "test")
	enc := fake.NewEncoderWithName(name)

	enc.SetPosition(1000)

	if err := enc.ResetPosition(ctx); err != nil {
		t.Fatalf("ResetPosition failed: %v", err)
	}

	pos, err := enc.Position(ctx)
	if err != nil {
		t.Fatalf("Position failed: %v", err)
	}
	if pos != 0 {
		t.Errorf("position after reset = %v, want 0", pos)
	}
}

func TestFakeEncoder_GetVelocity(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "encoder", "test")
	enc := fake.NewEncoderWithName(name)

	enc.SetVelocity(100.5)

	vel, err := enc.GetVelocity(ctx)
	if err != nil {
		t.Fatalf("GetVelocity failed: %v", err)
	}
	if vel != 100.5 {
		t.Errorf("velocity = %v, want 100.5", vel)
	}
}

// RangeSensor tests

func TestFakeRangeSensor_GetRange(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "range_sensor", "test")
	rs := fake.NewRangeSensorWithName(name)

	rs.SetRange(2.5)

	rng, err := rs.GetRange(ctx)
	if err != nil {
		t.Fatalf("GetRange failed: %v", err)
	}
	if rng != 2.5 {
		t.Errorf("range = %v, want 2.5", rng)
	}
}

func TestFakeRangeSensor_Properties(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "range_sensor", "test")
	rs := fake.NewRangeSensorWithName(name)

	props, err := rs.Properties(ctx)
	if err != nil {
		t.Fatalf("Properties failed: %v", err)
	}
	if props.MinRange != 0.02 {
		t.Errorf("MinRange = %v, want 0.02", props.MinRange)
	}
	if props.MaxRange != 4.0 {
		t.Errorf("MaxRange = %v, want 4.0", props.MaxRange)
	}
}

// LiDAR tests

func TestFakeLiDAR_GetScan(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "lidar", "test")
	lidar := fake.NewLiDARWithName(name)

	scan, err := lidar.GetScan(ctx)
	if err != nil {
		t.Fatalf("GetScan failed: %v", err)
	}
	if scan == nil {
		t.Fatal("scan is nil")
	}
	if len(scan.Ranges) == 0 {
		t.Error("expected non-empty ranges")
	}
}

func TestFakeLiDAR_GetPointCloud_2D(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "lidar", "test")
	lidar := fake.NewLiDARWithName(name)

	// 2D LiDAR should not support point cloud
	_, err := lidar.GetPointCloud(ctx)
	if err == nil {
		t.Error("expected error for 2D LiDAR GetPointCloud")
	}
}

func TestFakeLiDAR_SetScanMode(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "lidar", "test")
	lidar := fake.NewLiDARWithName(name)

	if err := lidar.SetScanMode(ctx, "boost"); err != nil {
		t.Fatalf("SetScanMode failed: %v", err)
	}
}

// PresenceSensor tests

func TestFakePresenceSensor_IsPresenceDetected(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "presence_sensor", "test")
	ps := fake.NewPresenceSensorWithName(name)

	// Default is no presence
	detected, err := ps.IsPresenceDetected(ctx)
	if err != nil {
		t.Fatalf("IsPresenceDetected failed: %v", err)
	}
	if detected {
		t.Error("expected no presence initially")
	}

	ps.SetPresence(true)

	detected, err = ps.IsPresenceDetected(ctx)
	if err != nil {
		t.Fatalf("IsPresenceDetected failed: %v", err)
	}
	if !detected {
		t.Error("expected presence after SetPresence(true)")
	}
}

func TestFakePresenceSensor_MotionState(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "presence_sensor", "test")
	ps := fake.NewPresenceSensorWithName(name)

	ps.SetMotionState(sensor.MotionMoving)

	state, err := ps.GetMotionState(ctx)
	if err != nil {
		t.Fatalf("GetMotionState failed: %v", err)
	}
	if state != sensor.MotionMoving {
		t.Errorf("motion state = %v, want MotionMoving", state)
	}
}

// ThermalArray tests

func TestFakeThermalArray_GetTemperatureGrid(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thermal_array", "test")
	thermal := fake.NewThermalArrayWithName(name, 8, 8)

	grid, err := thermal.GetTemperatureGrid(ctx)
	if err != nil {
		t.Fatalf("GetTemperatureGrid failed: %v", err)
	}
	if len(grid) != 8 {
		t.Errorf("grid rows = %v, want 8", len(grid))
	}
	if len(grid[0]) != 8 {
		t.Errorf("grid cols = %v, want 8", len(grid[0]))
	}
}

func TestFakeThermalArray_GetMinMaxTemperature(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "thermal_array", "test")
	thermal := fake.NewThermalArrayWithName(name, 8, 8)

	thermal.SetPixel(0, 0, 20.0)
	thermal.SetPixel(7, 7, 35.0)

	min, max, err := thermal.GetMinMaxTemperature(ctx)
	if err != nil {
		t.Fatalf("GetMinMaxTemperature failed: %v", err)
	}
	if min != 20.0 {
		t.Errorf("min temp = %v, want 20.0", min)
	}
	if max != 35.0 {
		t.Errorf("max temp = %v, want 35.0", max)
	}
}

// ForceSensor tests

func TestFakeForceSensor_GetForce(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "force_sensor", "test")
	fs := fake.NewForceSensorWithName(name)

	fs.SetForce(10.5)

	force, err := fs.GetForce(ctx)
	if err != nil {
		t.Fatalf("GetForce failed: %v", err)
	}
	if force != 10.5 {
		t.Errorf("force = %v, want 10.5", force)
	}
}

func TestFakeForceSensor_Tare(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "force_sensor", "test")
	fs := fake.NewForceSensorWithName(name)

	fs.SetForce(5.0)

	if err := fs.Tare(ctx); err != nil {
		t.Fatalf("Tare failed: %v", err)
	}

	force, err := fs.GetForce(ctx)
	if err != nil {
		t.Fatalf("GetForce failed: %v", err)
	}
	if force != 0 {
		t.Errorf("force after tare = %v, want 0", force)
	}

	// Add more force
	fs.SetForce(15.0)

	force, err = fs.GetForce(ctx)
	if err != nil {
		t.Fatalf("GetForce failed: %v", err)
	}
	if force != 10.0 {
		t.Errorf("force = %v, want 10.0 (15.0 - 5.0 tare)", force)
	}
}

// Force6DOF tests

func TestFakeForce6DOF_GetWrench(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "force_6dof", "test")
	f6 := fake.NewForce6DOFWithName(name)

	f6.SetWrench(1.0, 2.0, 3.0, 0.1, 0.2, 0.3)

	wrench, err := f6.GetWrench(ctx)
	if err != nil {
		t.Fatalf("GetWrench failed: %v", err)
	}
	if wrench.ForceX != 1.0 || wrench.ForceY != 2.0 || wrench.ForceZ != 3.0 {
		t.Errorf("forces = (%v, %v, %v), want (1.0, 2.0, 3.0)",
			wrench.ForceX, wrench.ForceY, wrench.ForceZ)
	}
	if wrench.TorqueX != 0.1 || wrench.TorqueY != 0.2 || wrench.TorqueZ != 0.3 {
		t.Errorf("torques = (%v, %v, %v), want (0.1, 0.2, 0.3)",
			wrench.TorqueX, wrench.TorqueY, wrench.TorqueZ)
	}
}

// CurrentSensor tests

func TestFakeCurrentSensor_GetCurrent(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "current_sensor", "test")
	cs := fake.NewCurrentSensorWithName(name)

	cs.SetMeasurements(5.0, 12.0)

	current, err := cs.GetCurrent(ctx)
	if err != nil {
		t.Fatalf("GetCurrent failed: %v", err)
	}
	if current != 5.0 {
		t.Errorf("current = %v, want 5.0", current)
	}

	voltage, err := cs.GetVoltage(ctx)
	if err != nil {
		t.Fatalf("GetVoltage failed: %v", err)
	}
	if voltage != 12.0 {
		t.Errorf("voltage = %v, want 12.0", voltage)
	}

	power, err := cs.GetPower(ctx)
	if err != nil {
		t.Fatalf("GetPower failed: %v", err)
	}
	if power != 60.0 {
		t.Errorf("power = %v, want 60.0 (5.0 * 12.0)", power)
	}
}

// ReflectanceSensor tests

func TestFakeReflectanceSensor_GetReflectances(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "reflectance_sensor", "test")
	rs := fake.NewReflectanceSensorWithName(name, 8)

	refs, err := rs.GetReflectances(ctx)
	if err != nil {
		t.Fatalf("GetReflectances failed: %v", err)
	}
	if len(refs) != 8 {
		t.Errorf("reflectances length = %v, want 8", len(refs))
	}
}

func TestFakeReflectanceSensor_SimulateLine(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "reflectance_sensor", "test")
	rs := fake.NewReflectanceSensorWithName(name, 8)

	// Simulate line in center
	rs.SimulateLine(0.5, 0.2)

	pos, err := rs.GetLinePosition(ctx)
	if err != nil {
		t.Fatalf("GetLinePosition failed: %v", err)
	}
	// Should be close to center (3.5 on 0-7 scale)
	if pos < 3.0 || pos > 4.0 {
		t.Errorf("line position = %v, expected ~3.5 (center)", pos)
	}
}

func TestFakeReflectanceSensor_Calibrate(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "reflectance_sensor", "test")
	rs := fake.NewReflectanceSensorWithName(name, 8)

	if err := rs.Calibrate(ctx); err != nil {
		t.Fatalf("Calibrate failed: %v", err)
	}
}

// AHRS tests

func TestFakeAHRS_GetEulerAngles(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "ahrs", "test")
	ahrs := fake.NewAHRSWithName(name)

	ahrs.SetEulerAngles(0.1, 0.2, 0.3)

	roll, pitch, yaw, err := ahrs.GetEulerAngles(ctx)
	if err != nil {
		t.Fatalf("GetEulerAngles failed: %v", err)
	}
	if roll != 0.1 || pitch != 0.2 || yaw != 0.3 {
		t.Errorf("euler = (%v, %v, %v), want (0.1, 0.2, 0.3)", roll, pitch, yaw)
	}
}

func TestFakeAHRS_GetCalibrationStatus(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "ahrs", "test")
	ahrs := fake.NewAHRSWithName(name)

	// Default is fully calibrated
	sys, gyro, accel, mag, err := ahrs.GetCalibrationStatus(ctx)
	if err != nil {
		t.Fatalf("GetCalibrationStatus failed: %v", err)
	}
	if sys != 3 || gyro != 3 || accel != 3 || mag != 3 {
		t.Errorf("calibration = (%v, %v, %v, %v), want (3, 3, 3, 3)", sys, gyro, accel, mag)
	}

	ahrs.SetCalibrationStatus(1, 2, 3, 0)

	sys, gyro, accel, mag, err = ahrs.GetCalibrationStatus(ctx)
	if err != nil {
		t.Fatalf("GetCalibrationStatus failed: %v", err)
	}
	if sys != 1 || gyro != 2 || accel != 3 || mag != 0 {
		t.Errorf("calibration = (%v, %v, %v, %v), want (1, 2, 3, 0)", sys, gyro, accel, mag)
	}
}
