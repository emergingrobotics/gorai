package bno085

import (
	"math"
	"testing"

	"github.com/emergingrobotics/gorai/components/sensor"
)

const testEps = 1e-9

// quatClose reports whether two quaternions represent the same rotation (q and
// -q are equal), within eps.
func quatClose(a, b quat, eps float64) bool {
	dot := a.x*b.x + a.y*b.y + a.z*b.z + a.w*b.w
	return math.Abs(math.Abs(dot)-1) < eps
}

func TestMountingMatrixIdentity(t *testing.T) {
	m, err := mountingMatrix(sensor.Mounting{X: "+x", Y: "+y", Z: "+z"})
	if err != nil {
		t.Fatalf("identity mounting: %v", err)
	}
	want := [3][3]float64{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	if m != want {
		t.Fatalf("identity matrix = %v, want %v", m, want)
	}
	if d := det3(m); math.Abs(d-1) > testEps {
		t.Fatalf("identity det = %v, want 1", d)
	}
}

func TestMountingMatrixRemap(t *testing.T) {
	// body_x = +imu_y, body_y = -imu_x, body_z = +imu_z (90 deg about z).
	m, err := mountingMatrix(sensor.Mounting{X: "+y", Y: "-x", Z: "+z"})
	if err != nil {
		t.Fatalf("remap mounting: %v", err)
	}
	if d := det3(m); math.Abs(d-1) > testEps {
		t.Fatalf("remap det = %v, want +1 (right-handed)", d)
	}
	// An IMU vector along +x should read as body -y.
	x, y, z := applyMatrix(m, 1, 0, 0)
	if math.Abs(x) > testEps || math.Abs(y+1) > testEps || math.Abs(z) > testEps {
		t.Fatalf("apply(+x) = (%v,%v,%v), want (0,-1,0)", x, y, z)
	}
}

func TestMountingRejectsLeftHanded(t *testing.T) {
	if _, err := mountingMatrix(sensor.Mounting{X: "+x", Y: "+y", Z: "-z"}); err == nil {
		t.Fatalf("expected left-handed mounting to be rejected")
	}
}

func TestMountingRejectsReusedAxis(t *testing.T) {
	if _, err := mountingMatrix(sensor.Mounting{X: "+x", Y: "+x", Z: "+z"}); err == nil {
		t.Fatalf("expected reused-axis mounting to be rejected")
	}
}

func TestMountingRejectsBadSpec(t *testing.T) {
	if _, err := mountingMatrix(sensor.Mounting{X: "x", Y: "+y", Z: "+z"}); err == nil {
		t.Fatalf("expected bad axis spec to be rejected")
	}
}

func TestEulerQuatRoundTrip(t *testing.T) {
	roll, pitch, yaw := -0.25, 0.15, 0.9 // radians, away from gimbal lock
	q := eulerToQuat(roll, pitch, yaw)
	gr, gp, gy := quaternionToEuler(q.x, q.y, q.z, q.w)
	if math.Abs(gr-roll) > 1e-9 || math.Abs(gp-pitch) > 1e-9 || math.Abs(gy-yaw) > 1e-9 {
		t.Fatalf("round trip = (%v,%v,%v), want (%v,%v,%v)", gr, gp, gy, roll, pitch, yaw)
	}
}

func TestMatrixToQuatMatchesEuler(t *testing.T) {
	// The remap body_x=+imu_y, body_y=-imu_x rotates vectors by -90 deg about z.
	m, _ := mountingMatrix(sensor.Mounting{X: "+y", Y: "-x", Z: "+z"})
	got := matrixToQuat(m)
	want := eulerToQuat(0, 0, -math.Pi/2)
	if !quatClose(got, want, 1e-9) {
		t.Fatalf("matrixToQuat = %+v, want ~%+v", got, want)
	}
}

func TestAverageQuatsSignAligned(t *testing.T) {
	q := eulerToQuat(0.1, -0.2, 0.3)
	neg := quat{-q.x, -q.y, -q.z, -q.w}
	avg := averageQuats([]quat{q, neg, q})
	if !quatClose(avg, q, 1e-9) {
		t.Fatalf("averageQuats with opposite-sign duplicates = %+v, want ~%+v", avg, q)
	}
}

func TestOrientZerosReference(t *testing.T) {
	// With identity mount, orienting a raw quaternion against itself as the
	// reference must yield identity (this is what calibration does).
	raw := eulerToQuat(0.2, -0.1, 1.2)
	mount := identityQuat()
	out := orient(raw, mount, raw)
	if !quatClose(out, identityQuat(), 1e-9) {
		t.Fatalf("orient(raw, id, raw) = %+v, want identity", out)
	}
}
