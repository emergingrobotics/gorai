package keypress_motor

// AngleToPulse converts an angle to a servo pulse width in microseconds.
// The angle is mapped from the min/max angle range to the min/max pulse range.
//
// Standard servo mapping:
//
//	min_angle → minPulseUs
//	max_angle → maxPulseUs
func AngleToPulse(angle, minAngle, maxAngle, minPulseUs, maxPulseUs float64) float64 {
	// Normalize angle to 0.0-1.0 range
	normalized := (angle - minAngle) / (maxAngle - minAngle)

	// Map to pulse width using actual PWM component limits
	return minPulseUs + normalized*(maxPulseUs-minPulseUs)
}

// PulseToAngle converts a servo pulse width in microseconds to an angle.
// This is the inverse of AngleToPulse.
func PulseToAngle(pulseUs, minAngle, maxAngle, minPulseUs, maxPulseUs float64) float64 {
	// Normalize pulse to 0.0-1.0 range
	normalized := (pulseUs - minPulseUs) / (maxPulseUs - minPulseUs)

	// Map to angle range
	return minAngle + normalized*(maxAngle-minAngle)
}

// SpeedToPulse converts a speed value (-1.0 to +1.0) to a servo pulse width.
// This is used for continuous rotation servos.
//
// Mapping:
//
//	-1.0 (full reverse) → minPulseUs
//	 0.0 (stop)         → center pulse
//	+1.0 (full forward) → maxPulseUs
func SpeedToPulse(speed, minPulseUs, maxPulseUs float64) float64 {
	centerPulse := (minPulseUs + maxPulseUs) / 2.0
	halfRange := (maxPulseUs - minPulseUs) / 2.0
	return centerPulse + speed*halfRange
}

// PulseToSpeed converts a servo pulse width to a speed value (-1.0 to +1.0).
// This is the inverse of SpeedToPulse.
func PulseToSpeed(pulseUs, minPulseUs, maxPulseUs float64) float64 {
	centerPulse := (minPulseUs + maxPulseUs) / 2.0
	halfRange := (maxPulseUs - minPulseUs) / 2.0
	return (pulseUs - centerPulse) / halfRange
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
