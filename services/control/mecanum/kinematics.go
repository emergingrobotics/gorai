package mecanum

import "math"

// WheelSpeeds holds the computed power for each mecanum wheel.
type WheelSpeeds struct {
	FL float64
	FR float64
	RL float64
	RR float64
}

// InverseKinematics computes individual wheel powers from body-frame velocity
// using standard robot body frame convention:
//
//   - vx: forward velocity (positive = forward, negative = backward)
//   - vy: lateral velocity (positive = right, negative = left)
//   - omega: rotational velocity (positive = counter-clockwise)
//   - lx: half wheelbase length along x-axis (front-to-rear center distance / 2) [m]
//   - ly: half track width along y-axis (left-to-right center distance / 2) [m]
//   - r: wheel radius [m]
//
// Standard mecanum wheel arrangement (X-configuration):
//
//	FL ---- FR
//	|   x+   |
//	| y- ← → y+
//	|   x-   |
//	RL ---- RR
//
// Returns normalized wheel speeds in [-1.0, 1.0].
func InverseKinematics(vx, vy, omega, lx, ly, r float64) WheelSpeeds {
	k := lx + ly
	inv_r := 1.0 / r

	fl := inv_r * (vx - vy - k*omega)
	fr := inv_r * (vx + vy + k*omega)
	rl := inv_r * (vx + vy - k*omega)
	rr := inv_r * (vx - vy + k*omega)

	return normalize(WheelSpeeds{FL: fl, FR: fr, RL: rl, RR: rr})
}

// normalize scales all wheel speeds so the maximum absolute value is 1.0.
// If all speeds are within [-1.0, 1.0], they are returned unchanged.
func normalize(ws WheelSpeeds) WheelSpeeds {
	max_abs := math.Max(
		math.Max(math.Abs(ws.FL), math.Abs(ws.FR)),
		math.Max(math.Abs(ws.RL), math.Abs(ws.RR)),
	)

	if max_abs <= 1.0 {
		return ws
	}

	return WheelSpeeds{
		FL: ws.FL / max_abs,
		FR: ws.FR / max_abs,
		RL: ws.RL / max_abs,
		RR: ws.RR / max_abs,
	}
}
