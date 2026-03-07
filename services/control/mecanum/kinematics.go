package mecanum

import "math"

// WheelSpeeds holds the computed angular velocity (rad/s) or duty ratio for each mecanum wheel.
type WheelSpeeds struct {
	FL float64
	FR float64
	RL float64
	RR float64
}

// InverseKinematics computes wheel angular velocities (rad/s) from body-frame velocity.
//
// Robot body frame: x+ forward, y+ left, z+ up, omega positive = CCW.
// Input: vx, vy (m/s), omega (rad/s), lx, ly (half wheelbase/track in m), r (wheel radius in m).
// Output: wheel angular velocities (rad/s).
//
// Standard mecanum wheel arrangement (X-configuration):
//
//	FL ---- FR
//	|   x+   |
//	| y+ ← → y-
//	|   x-   |
//	RL ---- RR
func InverseKinematics(vx, vy, omega, lx, ly, r float64) WheelSpeeds {
	if r <= 0 {
		return WheelSpeeds{}
	}
	inv_r := 1.0 / r
	k := lx + ly
	return WheelSpeeds{
		FL: inv_r * (vx + vy - k*omega),
		FR: inv_r * (vx - vy + k*omega),
		RL: inv_r * (vx - vy - k*omega),
		RR: inv_r * (vx + vy + k*omega),
	}
}

// ComputeWheelMix computes wheel direction ratios from body-frame velocity.
// Uses same sign pattern as InverseKinematics but omits 1/r (legacy; prefer InverseKinematics).
//
// Robot body frame: x+ forward, y+ left, z+ up.
func ComputeWheelMix(vx, vy, omega, lx, ly float64) WheelSpeeds {
	k := lx + ly
	return WheelSpeeds{
		FL: vx + vy - k*omega,
		FR: vx - vy + k*omega,
		RL: vx - vy - k*omega,
		RR: vx + vy + k*omega,
	}
}

// NormalizeSpeeds scales all wheel speeds so the maximum absolute value is 1.0.
// Always normalizes (even when all values are below 1.0).
// Zero input returns zero output.
func NormalizeSpeeds(ws WheelSpeeds) WheelSpeeds {
	max_abs := math.Max(
		math.Max(math.Abs(ws.FL), math.Abs(ws.FR)),
		math.Max(math.Abs(ws.RL), math.Abs(ws.RR)),
	)

	if max_abs <= 0 {
		return WheelSpeeds{}
	}

	return WheelSpeeds{
		FL: ws.FL / max_abs,
		FR: ws.FR / max_abs,
		RL: ws.RL / max_abs,
		RR: ws.RR / max_abs,
	}
}

// ScaleWheelSpeeds multiplies all wheel speeds by factor.
func ScaleWheelSpeeds(ws WheelSpeeds, factor float64) WheelSpeeds {
	return WheelSpeeds{
		FL: ws.FL * factor,
		FR: ws.FR * factor,
		RL: ws.RL * factor,
		RR: ws.RR * factor,
	}
}
