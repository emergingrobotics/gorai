package mecanum

import "math"

// WheelSpeeds holds the computed power for each mecanum wheel.
type WheelSpeeds struct {
	FL float64
	FR float64
	RL float64
	RR float64
}

// ComputeWheelMix computes wheel direction ratios from body-frame velocity.
// Used for keyboard/duty-cycle input where only direction matters.
//
// Standard robot body frame convention:
//   - vx: forward velocity (positive = forward, negative = backward)
//   - vy: lateral velocity (positive = right, negative = left)
//   - omega: rotational velocity (positive = counter-clockwise)
//   - lx: half wheelbase length along x-axis (front-to-rear center distance / 2) [m]
//   - ly: half track width along y-axis (left-to-right center distance / 2) [m]
//
// Standard mecanum wheel arrangement (X-configuration):
//
//	FL ---- FR
//	|   x+   |
//	| y- ← → y+
//	|   x-   |
//	RL ---- RR
//
// Returns raw wheel mix values (no normalization, no 1/r).
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
