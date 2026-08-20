package bno085

import (
	"fmt"
	"math"

	"github.com/emergingrobotics/gorai/components/sensor"
)

// quat is a unit quaternion in (x, y, z, w) order, matching the driver state.
type quat struct {
	x, y, z, w float64
}

// identityQuat returns the no-rotation quaternion.
func identityQuat() quat { return quat{0, 0, 0, 1} }

// conjugate returns the quaternion inverse for a unit quaternion.
func (q quat) conjugate() quat { return quat{-q.x, -q.y, -q.z, q.w} }

// normalize scales the quaternion to unit length.
func (q quat) normalize() quat {
	n := math.Sqrt(q.x*q.x + q.y*q.y + q.z*q.z + q.w*q.w)
	if n == 0 {
		return identityQuat()
	}
	return quat{q.x / n, q.y / n, q.z / n, q.w / n}
}

// mul returns the Hamilton product a*b (apply b, then a in the same frame).
func (a quat) mul(b quat) quat {
	return quat{
		x: a.w*b.x + a.x*b.w + a.y*b.z - a.z*b.y,
		y: a.w*b.y - a.x*b.z + a.y*b.w + a.z*b.x,
		z: a.w*b.z + a.x*b.y - a.y*b.x + a.z*b.w,
		w: a.w*b.w - a.x*b.x - a.y*b.y - a.z*b.z,
	}
}

// eulerToQuat builds a quaternion from aerospace ZYX Euler angles (radians),
// matching the convention used by quaternionToEuler.
func eulerToQuat(roll, pitch, yaw float64) quat {
	cr, sr := math.Cos(roll*0.5), math.Sin(roll*0.5)
	cp, sp := math.Cos(pitch*0.5), math.Sin(pitch*0.5)
	cy, sy := math.Cos(yaw*0.5), math.Sin(yaw*0.5)
	return quat{
		x: sr*cp*cy - cr*sp*sy,
		y: cr*sp*cy + sr*cp*sy,
		z: cr*cp*sy - sr*sp*cy,
		w: cr*cp*cy + sr*sp*sy,
	}
}

// averageQuats returns the sign-aligned normalized average of a set of unit
// quaternions. For near-static samples this is an accurate mean orientation.
func averageQuats(qs []quat) quat {
	if len(qs) == 0 {
		return identityQuat()
	}
	ref := qs[0]
	var sx, sy, sz, sw float64
	for _, q := range qs {
		// Flip samples into the same hemisphere as the reference so opposite-
		// sign representations of the same rotation do not cancel.
		if q.x*ref.x+q.y*ref.y+q.z*ref.z+q.w*ref.w < 0 {
			q = quat{-q.x, -q.y, -q.z, -q.w}
		}
		sx, sy, sz, sw = sx+q.x, sy+q.y, sz+q.z, sw+q.w
	}
	return quat{sx, sy, sz, sw}.normalize()
}

// axisVector maps an axis spec ("+x","-y",...) to a signed unit vector index.
// It returns the column index (0..2) and the sign.
func axisVector(spec string) (idx int, sign float64, err error) {
	if len(spec) != 2 {
		return 0, 0, fmt.Errorf("axis %q must be like \"+x\" or \"-z\"", spec)
	}
	switch spec[0] {
	case '+':
		sign = 1
	case '-':
		sign = -1
	default:
		return 0, 0, fmt.Errorf("axis %q must start with + or -", spec)
	}
	switch spec[1] {
	case 'x', 'X':
		idx = 0
	case 'y', 'Y':
		idx = 1
	case 'z', 'Z':
		idx = 2
	default:
		return 0, 0, fmt.Errorf("axis %q must reference x, y or z", spec)
	}
	return idx, sign, nil
}

// mountingMatrix builds the 3x3 rotation matrix M such that v_body = M * v_imu.
// Row i is the signed unit vector of the IMU axis assigned to body axis i. It
// validates that the mapping is a right-handed orthonormal permutation.
func mountingMatrix(m sensor.Mounting) ([3][3]float64, error) {
	var mat [3][3]float64
	used := [3]bool{}
	for row, spec := range []string{m.X, m.Y, m.Z} {
		idx, sign, err := axisVector(spec)
		if err != nil {
			return mat, err
		}
		if used[idx] {
			return mat, fmt.Errorf("mounting reuses IMU axis %q", spec[1:])
		}
		used[idx] = true
		mat[row][idx] = sign
	}
	if det3(mat) < 0 {
		return mat, fmt.Errorf("mounting is left-handed (mirrored); flip one axis sign")
	}
	return mat, nil
}

// det3 returns the determinant of a 3x3 matrix.
func det3(m [3][3]float64) float64 {
	return m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
}

// applyMatrix returns M * (x, y, z).
func applyMatrix(m [3][3]float64, x, y, z float64) (float64, float64, float64) {
	return m[0][0]*x + m[0][1]*y + m[0][2]*z,
		m[1][0]*x + m[1][1]*y + m[1][2]*z,
		m[2][0]*x + m[2][1]*y + m[2][2]*z
}

// matrixToQuat converts an orthonormal rotation matrix to a unit quaternion.
func matrixToQuat(m [3][3]float64) quat {
	trace := m[0][0] + m[1][1] + m[2][2]
	var q quat
	switch {
	case trace > 0:
		s := math.Sqrt(trace+1.0) * 2
		q.w = 0.25 * s
		q.x = (m[2][1] - m[1][2]) / s
		q.y = (m[0][2] - m[2][0]) / s
		q.z = (m[1][0] - m[0][1]) / s
	case m[0][0] > m[1][1] && m[0][0] > m[2][2]:
		s := math.Sqrt(1.0+m[0][0]-m[1][1]-m[2][2]) * 2
		q.w = (m[2][1] - m[1][2]) / s
		q.x = 0.25 * s
		q.y = (m[0][1] + m[1][0]) / s
		q.z = (m[0][2] + m[2][0]) / s
	case m[1][1] > m[2][2]:
		s := math.Sqrt(1.0+m[1][1]-m[0][0]-m[2][2]) * 2
		q.w = (m[0][2] - m[2][0]) / s
		q.x = (m[0][1] + m[1][0]) / s
		q.y = 0.25 * s
		q.z = (m[1][2] + m[2][1]) / s
	default:
		s := math.Sqrt(1.0+m[2][2]-m[0][0]-m[1][1]) * 2
		q.w = (m[1][0] - m[0][1]) / s
		q.x = (m[0][2] + m[2][0]) / s
		q.y = (m[1][2] + m[2][1]) / s
		q.z = 0.25 * s
	}
	return q.normalize()
}
