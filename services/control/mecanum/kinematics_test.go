package mecanum

import (
	"math"
	"testing"
)

const epsilon = 1e-9

const (
	test_lx = 0.1
	test_ly = 0.1
)

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < epsilon
}

// fullPipeline runs ComputeWheelMix -> NormalizeSpeeds -> ScaleWheelSpeeds.
func fullPipeline(vx, vy, omega, lx, ly float64) WheelSpeeds {
	mix := ComputeWheelMix(vx, vy, omega, lx, ly)
	duty := NormalizeSpeeds(mix)
	speed := math.Max(math.Max(math.Abs(vx), math.Abs(vy)), math.Abs(omega))
	return ScaleWheelSpeeds(duty, speed)
}

// Body frame: x+ forward, y+ left, z+ up, omega positive = CCW.

func TestInverseKinematicsForward(t *testing.T) {
	r := 0.04
	lx, ly := 0.105, 0.045
	ws := InverseKinematics(1, 0, 0, lx, ly, r)
	// All wheels same direction for pure forward
	expected := 1.0 / r
	if !approxEqual(ws.FL, expected) || !approxEqual(ws.FR, expected) ||
		!approxEqual(ws.RL, expected) || !approxEqual(ws.RR, expected) {
		t.Errorf("forward: FL=%f FR=%f RL=%f RR=%f, expected all %f", ws.FL, ws.FR, ws.RL, ws.RR, expected)
	}
}

func TestInverseKinematicsStrafeLeft(t *testing.T) {
	r := 0.04
	lx, ly := test_lx, test_ly
	ws := InverseKinematics(0, 1, 0, lx, ly, r)
	expected := 1.0 / r
	if !approxEqual(ws.FL, expected) || !approxEqual(ws.FR, -expected) ||
		!approxEqual(ws.RL, -expected) || !approxEqual(ws.RR, expected) {
		t.Errorf("strafe left (y+): FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestInverseKinematicsStrafeRight(t *testing.T) {
	r := 0.04
	lx, ly := test_lx, test_ly
	ws := InverseKinematics(0, -1, 0, lx, ly, r)
	expected := 1.0 / r
	if !approxEqual(ws.FL, -expected) || !approxEqual(ws.FR, expected) ||
		!approxEqual(ws.RL, expected) || !approxEqual(ws.RR, -expected) {
		t.Errorf("strafe right (y-): FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestInverseKinematicsRotateCCW(t *testing.T) {
	r := 0.04
	k := test_lx + test_ly
	ws := InverseKinematics(0, 0, 1, test_lx, test_ly, r)
	expected := k / r
	if !approxEqual(ws.FL, -expected) || !approxEqual(ws.FR, expected) ||
		!approxEqual(ws.RL, -expected) || !approxEqual(ws.RR, expected) {
		t.Errorf("rotate CCW: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestInverseKinematicsZeroRadius(t *testing.T) {
	ws := InverseKinematics(1, 0, 0, test_lx, test_ly, 0)
	if ws.FL != 0 || ws.FR != 0 || ws.RL != 0 || ws.RR != 0 {
		t.Errorf("zero radius should return zero: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestComputeWheelMixForward(t *testing.T) {
	ws := ComputeWheelMix(1, 0, 0, test_lx, test_ly)
	if !approxEqual(ws.FL, 1.0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, 1.0) {
		t.Errorf("forward: FL=%f FR=%f RL=%f RR=%f, expected all 1.0", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestComputeWheelMixStrafeLeft(t *testing.T) {
	ws := ComputeWheelMix(0, 1, 0, test_lx, test_ly)
	if !approxEqual(ws.FL, 1.0) || !approxEqual(ws.FR, -1.0) ||
		!approxEqual(ws.RL, -1.0) || !approxEqual(ws.RR, 1.0) {
		t.Errorf("strafe left (y+): FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestComputeWheelMixStrafeRight(t *testing.T) {
	ws := ComputeWheelMix(0, -1, 0, test_lx, test_ly)
	if !approxEqual(ws.FL, -1.0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, -1.0) {
		t.Errorf("strafe right (y-): FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestComputeWheelMixRotateCCW(t *testing.T) {
	k := test_lx + test_ly
	ws := ComputeWheelMix(0, 0, 1, test_lx, test_ly)
	if !approxEqual(ws.FL, -k) || !approxEqual(ws.FR, k) ||
		!approxEqual(ws.RL, -k) || !approxEqual(ws.RR, k) {
		t.Errorf("rotate CCW: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestNormalizeSpeedsAlwaysNormalizes(t *testing.T) {
	ws := NormalizeSpeeds(WheelSpeeds{FL: 0.1, FR: 0.2, RL: 0.15, RR: 0.05})
	max_abs := math.Max(
		math.Max(math.Abs(ws.FL), math.Abs(ws.FR)),
		math.Max(math.Abs(ws.RL), math.Abs(ws.RR)),
	)
	if !approxEqual(max_abs, 1.0) {
		t.Errorf("expected max_abs=1.0 after normalize, got %f", max_abs)
	}
	if !approxEqual(ws.FR, 1.0) {
		t.Errorf("FR should be 1.0 (was max), got %f", ws.FR)
	}
}

func TestNormalizeSpeedsZeroInput(t *testing.T) {
	ws := NormalizeSpeeds(WheelSpeeds{})
	if ws.FL != 0 || ws.FR != 0 || ws.RL != 0 || ws.RR != 0 {
		t.Errorf("zero input should give zero output")
	}
}

func TestScaleWheelSpeeds(t *testing.T) {
	ws := ScaleWheelSpeeds(WheelSpeeds{FL: 1, FR: -1, RL: 0, RR: 0.5}, 0.25)
	if !approxEqual(ws.FL, 0.25) || !approxEqual(ws.FR, -0.25) ||
		!approxEqual(ws.RL, 0) || !approxEqual(ws.RR, 0.125) {
		t.Errorf("scale: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestFullPipelineForward(t *testing.T) {
	ws := fullPipeline(1, 0, 0, test_lx, test_ly)
	if !approxEqual(ws.FL, 1.0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, 1.0) {
		t.Errorf("forward: FL=%f FR=%f RL=%f RR=%f, expected all 1.0", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestFullPipelineBackward(t *testing.T) {
	ws := fullPipeline(-1, 0, 0, test_lx, test_ly)
	if !approxEqual(ws.FL, -1.0) || !approxEqual(ws.FR, -1.0) ||
		!approxEqual(ws.RL, -1.0) || !approxEqual(ws.RR, -1.0) {
		t.Errorf("backward: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestFullPipelineStrafeLeft(t *testing.T) {
	ws := fullPipeline(0, 1, 0, test_lx, test_ly)
	if !approxEqual(ws.FL, 1.0) || !approxEqual(ws.FR, -1.0) ||
		!approxEqual(ws.RL, -1.0) || !approxEqual(ws.RR, 1.0) {
		t.Errorf("strafe left (y+): FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestFullPipelineStrafeRight(t *testing.T) {
	ws := fullPipeline(0, -1, 0, test_lx, test_ly)
	if !approxEqual(ws.FL, -1.0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, -1.0) {
		t.Errorf("strafe right (y-): FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestFullPipelineRotateCCW(t *testing.T) {
	ws := fullPipeline(0, 0, 1, test_lx, test_ly)
	if !approxEqual(ws.FL, -1.0) || !approxEqual(ws.FR, 1.0) ||
		!approxEqual(ws.RL, -1.0) || !approxEqual(ws.RR, 1.0) {
		t.Errorf("rotate CCW: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestFullPipelineRotateCW(t *testing.T) {
	ws := fullPipeline(0, 0, -1, test_lx, test_ly)
	if !approxEqual(ws.FL, 1.0) || !approxEqual(ws.FR, -1.0) ||
		!approxEqual(ws.RL, 1.0) || !approxEqual(ws.RR, -1.0) {
		t.Errorf("rotate CW: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestFullPipelineZeroInput(t *testing.T) {
	ws := fullPipeline(0, 0, 0, test_lx, test_ly)
	if !approxEqual(ws.FL, 0) || !approxEqual(ws.FR, 0) ||
		!approxEqual(ws.RL, 0) || !approxEqual(ws.RR, 0) {
		t.Errorf("zero: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestFullPipelineDiagonalForwardRight(t *testing.T) {
	ws := fullPipeline(1, 1, 0, test_lx, test_ly)
	// vx=1, vy=1 -> mix fl=2, fr=0, rl=0, rr=2 -> duty fl=1, fr=0, rl=0, rr=1
	if !approxEqual(ws.FL, 1.0) || !approxEqual(ws.FR, 0) ||
		!approxEqual(ws.RL, 0) || !approxEqual(ws.RR, 1.0) {
		t.Errorf("diag fwd-right: FL=%f FR=%f RL=%f RR=%f", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestFullPipelineSpeedScaling(t *testing.T) {
	ws := fullPipeline(0.25, 0, 0, test_lx, test_ly)
	if !approxEqual(ws.FL, 0.25) || !approxEqual(ws.FR, 0.25) ||
		!approxEqual(ws.RL, 0.25) || !approxEqual(ws.RR, 0.25) {
		t.Errorf("speed scaling: FL=%f FR=%f RL=%f RR=%f, expected all 0.25", ws.FL, ws.FR, ws.RL, ws.RR)
	}
}

func TestFullPipelineRotationSpeedScaling(t *testing.T) {
	ws := fullPipeline(0, 0, 0.25, test_lx, test_ly)
	max_abs := math.Max(
		math.Max(math.Abs(ws.FL), math.Abs(ws.FR)),
		math.Max(math.Abs(ws.RL), math.Abs(ws.RR)),
	)
	if !approxEqual(max_abs, 0.25) {
		t.Errorf("rotation at 0.25 should scale to max 0.25, got max_abs=%f", max_abs)
	}
}

func TestFullPipelineForwardWithRotation(t *testing.T) {
	ws := fullPipeline(1, 0, 0.5, test_lx, test_ly)
	if ws.FL >= ws.FR {
		t.Errorf("forward+CCW: FL=%f should be < FR=%f", ws.FL, ws.FR)
	}
	if ws.FL < 0 || ws.FR < 0 || ws.RL < 0 || ws.RR < 0 {
		t.Errorf("forward+small-CCW should have all positive: FL=%f FR=%f RL=%f RR=%f",
			ws.FL, ws.FR, ws.RL, ws.RR)
	}
}
