package ncp

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/emergingrobotics/gorai/components/sensor"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

// fakeAHRS is a minimal orientationSensor for adapter testing.
type fakeAHRS struct {
	name             resource.Name
	roll, pitch, yaw float64
	mounting         sensor.Mounting
	offsetDeg        [3]float64
	zeroed           bool
	calibrations     int
	cleared          int
}

func (f *fakeAHRS) Name() resource.Name { return f.name }
func (f *fakeAHRS) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	return nil
}
func (f *fakeAHRS) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	return nil, nil
}
func (f *fakeAHRS) Close(ctx context.Context) error { return nil }

func (f *fakeAHRS) LinearAcceleration(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}
func (f *fakeAHRS) AngularVelocity(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}
func (f *fakeAHRS) Orientation(ctx context.Context) (x, y, z, w float64, err error) {
	return 0, 0, 0, 1, nil
}
func (f *fakeAHRS) GetMagneticField(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}
func (f *fakeAHRS) GetEulerAngles(ctx context.Context) (roll, pitch, yaw float64, err error) {
	return f.roll, f.pitch, f.yaw, nil
}
func (f *fakeAHRS) GetQuaternion(ctx context.Context) (x, y, z, w float64, err error) {
	return 0, 0, 0, 1, nil
}
func (f *fakeAHRS) GetLinearAccelerationWithoutGravity(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}
func (f *fakeAHRS) GetGravityVector(ctx context.Context) (x, y, z float64, err error) {
	return 0, 0, 0, nil
}
func (f *fakeAHRS) GetCalibrationStatus(ctx context.Context) (sys, gyro, accel, mag uint8, err error) {
	return 3, 3, 2, 1, nil
}

func (f *fakeAHRS) Calibrate(ctx context.Context, dur time.Duration) error {
	f.calibrations++
	f.zeroed = true
	return nil
}
func (f *fakeAHRS) ClearZero(ctx context.Context) error {
	f.cleared++
	f.zeroed = false
	return nil
}
func (f *fakeAHRS) SetMounting(ctx context.Context, m sensor.Mounting) error {
	f.mounting = m
	return nil
}
func (f *fakeAHRS) SetOffset(ctx context.Context, rollDeg, pitchDeg, yawDeg float64) error {
	f.offsetDeg = [3]float64{rollDeg, pitchDeg, yawDeg}
	return nil
}
func (f *fakeAHRS) OrientationConfig(ctx context.Context) (sensor.OrientationState, error) {
	return sensor.OrientationState{Mounting: f.mounting, OffsetDeg: f.offsetDeg, Zeroed: f.zeroed}, nil
}

var (
	_ sensor.AHRS                    = (*fakeAHRS)(nil)
	_ sensor.OrientationConfigurable = (*fakeAHRS)(nil)
)

func TestAdapterForAHRS(t *testing.T) {
	f := &fakeAHRS{name: resource.NewComponentName("gorai", "ahrs", "imu")}
	if _, ok := adapterFor(f); !ok {
		t.Fatalf("expected AHRS to be adapted as a capability")
	}
}

func TestAHRSCapabilityStateAndDispatch(t *testing.T) {
	f := &fakeAHRS{
		name:     resource.NewComponentName("gorai", "ahrs", "imu"),
		yaw:      math.Pi / 2, // 90 deg
		mounting: sensor.Mounting{X: "+x", Y: "+y", Z: "+z"},
	}
	cap := ahrsCapability(f)
	ctx := context.Background()

	st, err := cap.state(ctx)
	if err != nil {
		t.Fatalf("state: %v", err)
	}
	if got := st["yaw"].(float64); math.Abs(got-90) > 1e-6 {
		t.Fatalf("state yaw = %v, want 90", got)
	}
	if _, ok := st["heading"]; ok {
		t.Fatalf("state must not include a heading field (yaw is not a compass heading)")
	}

	// set_mounting
	if _, err := cap.dispatch(ctx, Request{Method: "set_mounting", Args: map[string]any{"x": "+y", "y": "-x", "z": "+z"}}); err != nil {
		t.Fatalf("set_mounting: %v", err)
	}
	if f.mounting.X != "+y" || f.mounting.Y != "-x" {
		t.Fatalf("mounting not applied: %+v", f.mounting)
	}

	// calibrate
	if _, err := cap.dispatch(ctx, Request{Method: "calibrate", Args: map[string]any{"duration_ms": float64(50)}}); err != nil {
		t.Fatalf("calibrate: %v", err)
	}
	if f.calibrations != 1 || !f.zeroed {
		t.Fatalf("calibrate not applied: calls=%d zeroed=%v", f.calibrations, f.zeroed)
	}

	// set_offset
	if _, err := cap.dispatch(ctx, Request{Method: "set_offset", Args: map[string]any{"roll_deg": float64(1), "pitch_deg": float64(2), "yaw_deg": float64(3)}}); err != nil {
		t.Fatalf("set_offset: %v", err)
	}
	if f.offsetDeg != [3]float64{1, 2, 3} {
		t.Fatalf("offset not applied: %v", f.offsetDeg)
	}

	// clear_zero
	if _, err := cap.dispatch(ctx, Request{Method: "clear_zero"}); err != nil {
		t.Fatalf("clear_zero: %v", err)
	}
	if f.cleared != 1 || f.zeroed {
		t.Fatalf("clear_zero not applied: cleared=%d zeroed=%v", f.cleared, f.zeroed)
	}

	// unknown method
	if _, err := cap.dispatch(ctx, Request{Method: "bogus"}); err == nil {
		t.Fatalf("expected error for unknown method")
	}
}
