package bno085

import (
	"encoding/binary"
	"math"
)

// SHTP channels (Sensor Hub Transport Protocol).
const (
	chCommand    = 0
	chExecutable = 1
	chControl    = 2
	chInput      = 3
	chWake       = 4
	chGyro       = 5
)

// SH-2 report IDs.
const (
	reportSetFeature    = 0xFD
	reportBaseTimestamp = 0xFB
	reportAccelerometer = 0x01
	reportGyroscope     = 0x02
	reportMagnetometer  = 0x03
	reportLinearAccel   = 0x04
	reportRotationVec   = 0x05
	reportGravity       = 0x06
)

// Q-point scale factors (value * 2^-q) for the enabled reports.
const (
	qAccel    = 8  // m/s^2
	qGyro     = 9  // rad/s
	qMag      = 4  // uT
	qRotation = 14 // unit quaternion
)

// shtpHeader is the 4-byte SHTP packet header.
type shtpHeader struct {
	length  int  // cargo length excluding the 4-byte header
	channel byte // SHTP channel
	seq     byte // sequence number
	cont    bool // continuation of a previous transfer
}

// parseHeader decodes a 4-byte SHTP header. The 15-bit length field includes
// the header itself; the returned length excludes it.
func parseHeader(b []byte) (shtpHeader, bool) {
	if len(b) < 4 {
		return shtpHeader{}, false
	}
	raw := int(b[0]) | int(b[1])<<8
	total := raw & 0x7FFF
	h := shtpHeader{
		channel: b[2],
		seq:     b[3],
		cont:    raw&0x8000 != 0,
	}
	if total < 4 {
		h.length = 0
		return h, true
	}
	h.length = total - 4
	return h, true
}

// setFeatureCommand builds a 17-byte Set Feature control payload that requests
// a sensor report at the given interval (microseconds).
func setFeatureCommand(reportID byte, intervalUs uint32) []byte {
	cmd := make([]byte, 17)
	cmd[0] = reportSetFeature
	cmd[1] = reportID
	binary.LittleEndian.PutUint32(cmd[5:9], intervalUs)
	return cmd
}

// imuState holds the latest decoded IMU values.
type imuState struct {
	qi, qj, qk, qr float64
	ax, ay, az     float64
	gx, gy, gz     float64
	mx, my, mz     float64
	accuracyRad    float64

	calAccel uint8
	calGyro  uint8
	calMag   uint8

	haveQuat bool
}

// qToFloat converts a fixed-point Q value to float.
func qToFloat(raw int16, q uint) float64 {
	return float64(raw) * math.Pow(2, -float64(q))
}

// vec3 decodes three consecutive little-endian int16 values at off, scaled by q.
func vec3(data []byte, off int, q uint) (x, y, z float64) {
	x = qToFloat(int16(binary.LittleEndian.Uint16(data[off:off+2])), q)
	y = qToFloat(int16(binary.LittleEndian.Uint16(data[off+2:off+4])), q)
	z = qToFloat(int16(binary.LittleEndian.Uint16(data[off+4:off+6])), q)
	return
}

// reportLength returns the byte length of a sensor report by ID, or 0 if the
// report is unknown (which terminates parsing).
func reportLength(id byte) int {
	switch id {
	case reportBaseTimestamp:
		return 5
	case reportAccelerometer, reportGyroscope, reportMagnetometer,
		reportLinearAccel, reportGravity:
		return 10
	case reportRotationVec:
		return 14
	default:
		return 0
	}
}

// parseInputReports decodes an SHTP input-report cargo, updating st in place.
// It returns the number of recognized reports.
func parseInputReports(cargo []byte, st *imuState) int {
	count := 0
	i := 0
	for i < len(cargo) {
		id := cargo[i]
		n := reportLength(id)
		if n == 0 || i+n > len(cargo) {
			break
		}
		data := cargo[i : i+n]

		switch id {
		case reportBaseTimestamp:
			// timing only
		case reportAccelerometer:
			st.ax, st.ay, st.az = vec3(data, 4, qAccel)
			st.calAccel = data[2] & 0x03
			count++
		case reportGyroscope:
			st.gx, st.gy, st.gz = vec3(data, 4, qGyro)
			st.calGyro = data[2] & 0x03
			count++
		case reportMagnetometer:
			st.mx, st.my, st.mz = vec3(data, 4, qMag)
			st.calMag = data[2] & 0x03
			count++
		case reportRotationVec:
			st.qi = qToFloat(int16(binary.LittleEndian.Uint16(data[4:6])), qRotation)
			st.qj = qToFloat(int16(binary.LittleEndian.Uint16(data[6:8])), qRotation)
			st.qk = qToFloat(int16(binary.LittleEndian.Uint16(data[8:10])), qRotation)
			st.qr = qToFloat(int16(binary.LittleEndian.Uint16(data[10:12])), qRotation)
			st.accuracyRad = qToFloat(int16(binary.LittleEndian.Uint16(data[12:14])), 12)
			st.haveQuat = true
			count++
		}
		i += n
	}
	return count
}

// quaternionToEuler converts a quaternion (i, j, k, real) to roll/pitch/yaw in
// radians using the aerospace (Z-Y-X) convention.
func quaternionToEuler(i, j, k, r float64) (roll, pitch, yaw float64) {
	// roll (x-axis rotation)
	sinrCosp := 2 * (r*i + j*k)
	cosrCosp := 1 - 2*(i*i+j*j)
	roll = math.Atan2(sinrCosp, cosrCosp)

	// pitch (y-axis rotation)
	sinp := 2 * (r*j - k*i)
	if sinp >= 1 {
		pitch = math.Pi / 2
	} else if sinp <= -1 {
		pitch = -math.Pi / 2
	} else {
		pitch = math.Asin(sinp)
	}

	// yaw (z-axis rotation)
	sinyCosp := 2 * (r*k + i*j)
	cosyCosp := 1 - 2*(j*j+k*k)
	yaw = math.Atan2(sinyCosp, cosyCosp)
	return
}
