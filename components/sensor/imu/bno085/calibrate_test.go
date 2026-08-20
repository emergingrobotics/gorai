package bno085

import (
	"context"
	"io"
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/emergingrobotics/gorai/components/sensor"
)

// newConfiguredAHRS builds an AHRS with an identity frame and a preset raw
// orientation, without any I2C device or read loop, for transform/calibration
// tests.
func newConfiguredAHRS(raw quat) *AHRS {
	a := &AHRS{
		logger:      slog.New(slog.NewTextHandler(io.Discard, nil)),
		mounting:    identityMounting(),
		mountMatrix: [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}},
		mountQuat:   identityQuat(),
		offsetQuat:  identityQuat(),
	}
	a.config.ReportIntervalMs = 2
	a.state = imuState{qi: raw.x, qj: raw.y, qk: raw.z, qr: raw.w, haveQuat: true}
	return a
}

func TestCalibrateZeroesOrientation(t *testing.T) {
	raw := eulerToQuat(0.1, -0.2, 1.0) // arbitrary steady orientation
	a := newConfiguredAHRS(raw)

	// Before calibration the reading reflects the raw orientation.
	roll, pitch, yaw, _ := a.GetEulerAngles(context.Background())
	if math.Abs(roll-0.1) > 1e-6 || math.Abs(pitch+0.2) > 1e-6 || math.Abs(yaw-1.0) > 1e-6 {
		t.Fatalf("pre-calibration euler = (%v,%v,%v)", roll, pitch, yaw)
	}

	if err := a.Calibrate(context.Background(), 40*time.Millisecond); err != nil {
		t.Fatalf("Calibrate: %v", err)
	}

	oc, _ := a.OrientationConfig(context.Background())
	if !oc.Zeroed {
		t.Fatalf("expected zeroed=true after calibration")
	}
	// After calibration the same steady orientation reads as zero.
	roll, pitch, yaw, _ = a.GetEulerAngles(context.Background())
	if math.Abs(roll) > 1e-6 || math.Abs(pitch) > 1e-6 || math.Abs(yaw) > 1e-6 {
		t.Fatalf("post-calibration euler = (%v,%v,%v), want ~0", roll, pitch, yaw)
	}

	// Clearing the zero reverts to the (identity) offset.
	if err := a.ClearZero(context.Background()); err != nil {
		t.Fatalf("ClearZero: %v", err)
	}
	roll, _, yaw, _ = a.GetEulerAngles(context.Background())
	if math.Abs(roll-0.1) > 1e-6 || math.Abs(yaw-1.0) > 1e-6 {
		t.Fatalf("after clear euler = (%v,_,%v), want raw", roll, yaw)
	}
}

func TestCalibrateWithoutDataFails(t *testing.T) {
	a := newConfiguredAHRS(identityQuat())
	a.state.haveQuat = false
	if err := a.Calibrate(context.Background(), 20*time.Millisecond); err == nil {
		t.Fatalf("expected error when no orientation samples available")
	}
}

func TestSetMountingRemapsVectors(t *testing.T) {
	a := newConfiguredAHRS(identityQuat())
	a.state.ax, a.state.ay, a.state.az = 1, 0, 0
	if err := a.SetMounting(context.Background(), sensor.Mounting{X: "+y", Y: "-x", Z: "+z"}); err != nil {
		t.Fatalf("SetMounting: %v", err)
	}
	x, y, z, _ := a.LinearAcceleration(context.Background())
	if math.Abs(x) > 1e-9 || math.Abs(y+1) > 1e-9 || math.Abs(z) > 1e-9 {
		t.Fatalf("remapped accel = (%v,%v,%v), want (0,-1,0)", x, y, z)
	}
}

func TestSetOffsetAppliesWhenNotZeroed(t *testing.T) {
	a := newConfiguredAHRS(identityQuat())
	if err := a.SetOffset(context.Background(), 0, 0, 90); err != nil {
		t.Fatalf("SetOffset: %v", err)
	}
	// Raw identity orientation, offset yaw 90 -> reading yaw should be -90
	// (reference-relative), i.e. magnitude 90 deg.
	_, _, yaw, _ := a.GetEulerAngles(context.Background())
	if math.Abs(math.Abs(yaw)-math.Pi/2) > 1e-6 {
		t.Fatalf("offset yaw = %v rad, want +-pi/2", yaw)
	}
}
