package bno085

import (
	"encoding/binary"
	"math"
	"testing"
)

func TestParseHeader(t *testing.T) {
	// length field = 4 (header) + 14 (cargo) = 18, channel 3, seq 7.
	h, ok := parseHeader([]byte{18, 0x00, 3, 7})
	if !ok {
		t.Fatalf("parseHeader failed")
	}
	if h.length != 14 {
		t.Errorf("length = %d, want 14", h.length)
	}
	if h.channel != 3 || h.seq != 7 {
		t.Errorf("channel/seq = %d/%d, want 3/7", h.channel, h.seq)
	}
	if h.cont {
		t.Errorf("unexpected continuation bit")
	}
}

func TestParseHeaderContinuation(t *testing.T) {
	// Top bit of the length MSB is the continuation flag.
	h, ok := parseHeader([]byte{0x08, 0x80, 2, 0})
	if !ok || !h.cont {
		t.Errorf("continuation bit not detected")
	}
	if h.length != 4 {
		t.Errorf("length = %d, want 4", h.length)
	}
}

func TestSetFeatureCommand(t *testing.T) {
	cmd := setFeatureCommand(reportRotationVec, 20000)
	if len(cmd) != 17 {
		t.Fatalf("len = %d, want 17", len(cmd))
	}
	if cmd[0] != reportSetFeature || cmd[1] != reportRotationVec {
		t.Errorf("header bytes = %x/%x", cmd[0], cmd[1])
	}
	if got := binary.LittleEndian.Uint32(cmd[5:9]); got != 20000 {
		t.Errorf("interval = %d, want 20000", got)
	}
}

// makeRotationVector builds a rotation-vector report for quaternion (i,j,k,r).
func makeRotationVector(i, j, k, r int16) []byte {
	b := make([]byte, 14)
	b[0] = reportRotationVec
	b[2] = 0x03 // accuracy bits
	binary.LittleEndian.PutUint16(b[4:6], uint16(i))
	binary.LittleEndian.PutUint16(b[6:8], uint16(j))
	binary.LittleEndian.PutUint16(b[8:10], uint16(k))
	binary.LittleEndian.PutUint16(b[10:12], uint16(r))
	return b
}

func TestParseInputReportsRotationVector(t *testing.T) {
	// Identity quaternion: real=1.0 => 2^14 in Q14.
	cargo := makeRotationVector(0, 0, 0, 1<<14)
	var st imuState
	if n := parseInputReports(cargo, &st); n != 1 {
		t.Fatalf("recognized reports = %d, want 1", n)
	}
	if !st.haveQuat {
		t.Fatalf("haveQuat = false")
	}
	if math.Abs(st.qr-1.0) > 1e-6 {
		t.Errorf("qr = %v, want 1.0", st.qr)
	}
	if st.qi != 0 || st.qj != 0 || st.qk != 0 {
		t.Errorf("expected zero vector part, got %v %v %v", st.qi, st.qj, st.qk)
	}
}

func TestParseInputReportsWithTimestampPrefix(t *testing.T) {
	// A base-timestamp report (5 bytes) followed by an accelerometer report.
	accel := make([]byte, 10)
	accel[0] = reportAccelerometer
	binary.LittleEndian.PutUint16(accel[4:6], uint16(int16(256)))   // x = 1.0 (Q8)
	binary.LittleEndian.PutUint16(accel[6:8], uint16(int16(512)))   // y = 2.0
	binary.LittleEndian.PutUint16(accel[8:10], 0xFF00) // z = -1.0 (int16 -256)

	cargo := append([]byte{reportBaseTimestamp, 0, 0, 0, 0}, accel...)
	var st imuState
	if n := parseInputReports(cargo, &st); n != 1 {
		t.Fatalf("recognized reports = %d, want 1", n)
	}
	if math.Abs(st.ax-1.0) > 1e-6 || math.Abs(st.ay-2.0) > 1e-6 || math.Abs(st.az+1.0) > 1e-6 {
		t.Errorf("accel = (%v,%v,%v), want (1,2,-1)", st.ax, st.ay, st.az)
	}
}

func TestQuaternionToEulerIdentity(t *testing.T) {
	roll, pitch, yaw := quaternionToEuler(0, 0, 0, 1)
	if math.Abs(roll) > 1e-9 || math.Abs(pitch) > 1e-9 || math.Abs(yaw) > 1e-9 {
		t.Errorf("identity euler = (%v,%v,%v), want zeros", roll, pitch, yaw)
	}
}

func TestQuaternionToEulerYaw90(t *testing.T) {
	// 90° yaw about Z: q = (0,0,sin45,cos45).
	s := math.Sqrt2 / 2
	_, _, yaw := quaternionToEuler(0, 0, s, s)
	if math.Abs(yaw-math.Pi/2) > 1e-6 {
		t.Errorf("yaw = %v, want pi/2", yaw)
	}
}
