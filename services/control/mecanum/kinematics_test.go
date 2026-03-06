package mecanum

import (
	"math"
	"testing"
)

const epsilon = 1e-9

// Standard test parameters (meters).
const (
	test_lx = 0.1
	test_ly = 0.1
	test_r  = 0.03
)

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// Body frame: vx = forward (+), vy = right (+), omega = CCW (+).

func TestForward(t *testing.T) {
	ws := InverseKinematics(1, 0, 0, test_lx, test_ly, test_r)
	if !approxEqual(ws.FL, 1.0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, 1.0) {
		t.Errorf("forward: FL=%f FR=%f RL=%f RR=%f, expected all 1.0", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestBackward(t *testing.T) {
	ws := InverseKinematics(-1, 0, 0, test_lx, test_ly, test_r)
	if !approxEqual(ws.FL, -1.0) || !approxEqual(ws.FR, -1.0) ||
		!approxEqual(ws.RL, -1.0) || !approxEqual(ws.RR, -1.0) {
		t.Errorf("backward: FL=%f FR=%f RL=%f RR=%f, expected all -1.0", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestStrafeRight(t *testing.T) {
	ws := InverseKinematics(0, 1, 0, test_lx, test_ly, test_r)
	if !approxEqual(ws.FL, -1.0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, -1.0) {
		t.Errorf("strafe right: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestStrafeLeft(t *testing.T) {
	ws := InverseKinematics(0, -1, 0, test_lx, test_ly, test_r)
	if !approxEqual(ws.FL, 1.0) || !approxEqual(ws.FR, -1.0) ||
		!approxEqual(ws.RL, -1.0) || !approxEqual(ws.RR, 1.0) {
		t.Errorf("strafe left: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestRotateCCW(t *testing.T) {
	ws := InverseKinematics(0, 0, 1, test_lx, test_ly, test_r)
	// Pure rotation: raw values are ±k/r which exceed 1.0, so normalization kicks in.
	if !approxEqual(ws.FL, -1.0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, -1.0) || !approxEqual(ws.RR, 1.0) {
		t.Errorf("rotate CCW: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestRotateCW(t *testing.T) {
	ws := InverseKinematics(0, 0, -1, test_lx, test_ly, test_r)
	if !approxEqual(ws.FL, 1.0) || !approxEqual(ws.FR, -1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, -1.0) {
		t.Errorf("rotate CW: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestZeroInput(t *testing.T) {
	ws := InverseKinematics(0, 0, 0, test_lx, test_ly, test_r)
	if !approxEqual(ws.FL, 0) || !approxEqual(ws.FR, 0) ||
		!approxEqual(ws.RL, 0) || !approxEqual(ws.RR, 0) {
		t.Errorf("zero: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestDiagonalForwardRight(t *testing.T) {
	ws := InverseKinematics(1, 1, 0, test_lx, test_ly, test_r)
	// vx=1, vy=1 → fl=0, fr=2/r, rl=2/r, rr=0 → normalized: 0, 1, 1, 0
	if !approxEqual(ws.FL, 0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, 0) {
		t.Errorf("diag fwd-right: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestNormalization(t *testing.T) {
	ws := InverseKinematics(2, 2, 0, test_lx, test_ly, test_r)
	if math.Abs(ws.FL) > 1.0+epsilon || math.Abs(ws.FR) > 1.0+epsilon ||
		math.Abs(ws.RL) > 1.0+epsilon || math.Abs(ws.RR) > 1.0+epsilon {
		t.Errorf("normalization failed: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
	max_abs := math.Max(
		math.Max(math.Abs(ws.FL), math.Abs(ws.FR)),
		math.Max(math.Abs(ws.RL), math.Abs(ws.RR)),
	)
	if !approxEqual(max_abs, 1.0) {
		t.Errorf("max_abs=%f, expected 1.0", max_abs)
	}
}

func TestForwardWithRotation(t *testing.T) {
	ws := InverseKinematics(1, 0, 0.5, test_lx, test_ly, test_r)
	// Forward + CCW rotation: FL should be less than FR
	if ws.FL >= ws.FR {
		t.Errorf("forward+CCW: FL=%f should be < FR=%f", ws.FL, ws.FR)
	}
	if ws.FL < 0 || ws.FR < 0 || ws.RL < 0 || ws.RR < 0 {
		t.Errorf("forward+small-CCW should have all positive: FL=%f FR=%f RL=%f RR=%f",
			ws.FL, ws.FR, ws.RL, ws.RR)
	}
}
