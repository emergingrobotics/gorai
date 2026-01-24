package keypress_motor

// AngleToPulse converts an angle to a servo pulse width in microseconds.
// The angle is mapped from the min/max range to the 1000-2000µs pulse range.
//
// Standard servo mapping:
//   min_angle → 1000 µs
//   max_angle → 2000 µs
func AngleToPulse(angle, minAngle, maxAngle float64) float64 {
	// Normalize angle to 0.0-1.0 range
	normalized := (angle - minAngle) / (maxAngle - minAngle)

	// Map to pulse width (1000-2000 µs)
	return 1000.0 + normalized*1000.0
}

// PulseToAngle converts a servo pulse width in microseconds to an angle.
// This is the inverse of AngleToPulse.
func PulseToAngle(pulseUs, minAngle, maxAngle float64) float64 {
	// Normalize pulse to 0.0-1.0 range
	normalized := (pulseUs - 1000.0) / 1000.0

	// Map to angle range
	return minAngle + normalized*(maxAngle-minAngle)
}

// SpeedToPulse converts a speed value (-1.0 to +1.0) to a servo pulse width.
// This is used for continuous rotation servos.
//
// Mapping:
//   -1.0 (full reverse) → 1000 µs
//    0.0 (stop)         → 1500 µs
//   +1.0 (full forward) → 2000 µs
func SpeedToPulse(speed float64) float64 {
	return 1500.0 + speed*500.0
}

// PulseToSpeed converts a servo pulse width to a speed value (-1.0 to +1.0).
// This is the inverse of SpeedToPulse.
func PulseToSpeed(pulseUs float64) float64 {
	return (pulseUs - 1500.0) / 500.0
}

// Clamp restricts a value to the specified range.
func Clamp(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

