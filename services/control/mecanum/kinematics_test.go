package mecanum

import (
	"math"
	"testing"
)

const epsilon = 1e-9

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

func TestForward(t *testing.T) {
	ws := InverseKinematics(0, 1, 0, 0.1, 0.1)
	if !approxEqual(ws.FL, 1.0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, 1.0) {
		t.Errorf("forward: FL=%f FR=%f RL=%f RR=%f, expected all 1.0", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestBackward(t *testing.T) {
	ws := InverseKinematics(0, -1, 0, 0.1, 0.1)
	if !approxEqual(ws.FL, -1.0) || !approxEqual(ws.FR, -1.0) ||
		!approxEqual(ws.RL, -1.0) || !approxEqual(ws.RR, -1.0) {
		t.Errorf("backward: FL=%f FR=%f RL=%f RR=%f, expected all -1.0", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestStrafeRight(t *testing.T) {
	ws := InverseKinematics(1, 0, 0, 0.1, 0.1)
	// Strafe right: FL=-1, FR=+1, RL=+1, RR=-1
	if !approxEqual(ws.FL, -1.0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, -1.0) {
		t.Errorf("strafe right: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestStrafeLeft(t *testing.T) {
	ws := InverseKinematics(-1, 0, 0, 0.1, 0.1)
	// Strafe left: FL=+1, FR=-1, RL=-1, RR=+1
	if !approxEqual(ws.FL, 1.0) || !approxEqual(ws.FR, -1.0) ||
		!approxEqual(ws.RL, -1.0) || !approxEqual(ws.RR, 1.0) {
		t.Errorf("strafe left: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestRotateCCW(t *testing.T) {
	ws := InverseKinematics(0, 0, 1, 0.1, 0.1)
	// CCW rotation: FL<0, FR>0, RL<0, RR>0
	k := 0.1 + 0.1
	if !approxEqual(ws.FL, -k) || !approxEqual(ws.FR, k) ||
		!approxEqual(ws.RL, -k) || !approxEqual(ws.RR, k) {
		t.Errorf("rotate CCW: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestRotateCW(t *testing.T) {
	ws := InverseKinematics(0, 0, -1, 0.1, 0.1)
	k := 0.1 + 0.1
	if !approxEqual(ws.FL, k) || !approxEqual(ws.FR, -k) ||
		!approxEqual(ws.RL, k) || !approxEqual(ws.RR, -k) {
		t.Errorf("rotate CW: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestZeroInput(t *testing.T) {
	ws := InverseKinematics(0, 0, 0, 0.1, 0.1)
	if !approxEqual(ws.FL, 0) || !approxEqual(ws.FR, 0) ||
		!approxEqual(ws.RL, 0) || !approxEqual(ws.RR, 0) {
		t.Errorf("zero: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestDiagonalForwardRight(t *testing.T) {
	ws := InverseKinematics(1, 1, 0, 0.1, 0.1)
	// Forward+right: FL=0, FR=2, RL=2, RR=0 -> normalized: FL=0, FR=1, RL=1, RR=0
	if !approxEqual(ws.FL, 0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, 0) {
		t.Errorf("diag fwd-right: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestNormalization(t *testing.T) {
	ws := InverseKinematics(2, 2, 0, 0.1, 0.1)
	// All values should be in [-1.0, 1.0]
	if math.Abs(ws.FL) > 1.0+epsilon || math.Abs(ws.FR) > 1.0+epsilon ||
		math.Abs(ws.RL) > 1.0+epsilon || math.Abs(ws.RR) > 1.0+epsilon {
		t.Errorf("normalization failed: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
	// Maximum should be exactly 1.0
	max_abs := math.Max(
		math.Max(math.Abs(ws.FL), math.Abs(ws.FR)),
		math.Max(math.Abs(ws.RL), math.Abs(ws.RR)),
	)
	if !approxEqual(max_abs, 1.0) {
		t.Errorf("max_abs=%f, expected 1.0", max_abs)
	}
}

func TestForwardWithRotation(t *testing.T) {
	ws := InverseKinematics(0, 1, 0.5, 0.1, 0.1)
	// Forward + CCW rotation: FL should be less than FR
	if ws.FL >= ws.FR {
		t.Errorf("forward+CCW: FL=%f should be < FR=%f", ws.FL, ws.FR)
	}
	// All positive (forward dominates small rotation)
	if ws.FL < 0 || ws.FR < 0 || ws.RL < 0 || ws.RR < 0 {
		t.Errorf("forward+small-CCW should have all positive: FL=%f FR=%f RL=%f RR=%f",
			ws.FL, ws.FR, ws.RL, ws.RR)
	}
}
