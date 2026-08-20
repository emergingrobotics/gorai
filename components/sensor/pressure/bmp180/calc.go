package bmp180

import (
	"encoding/binary"
	"fmt"
	"math"
)

// calibration holds the BMP180 factory calibration coefficients read from
// EEPROM (registers 0xAA..0xBF).
type calibration struct {
	AC1, AC2, AC3      int16
	AC4, AC5, AC6      uint16
	B1, B2             int16
	MB, MC, MD         int16
}

// parseCalibration decodes the 22-byte EEPROM calibration block.
func parseCalibration(data []byte) (calibration, error) {
	if len(data) < 22 {
		return calibration{}, fmt.Errorf("calibration block too short: %d bytes", len(data))
	}
	u := func(i int) uint16 { return binary.BigEndian.Uint16(data[i : i+2]) }
	s := func(i int) int16 { return int16(u(i)) }

	cal := calibration{
		AC1: s(0), AC2: s(2), AC3: s(4),
		AC4: u(6), AC5: u(8), AC6: u(10),
		B1: s(12), B2: s(14),
		MB: s(16), MC: s(18), MD: s(20),
	}
	return cal, cal.validate()
}

// validate rejects the all-zero / all-ones EEPROM patterns that indicate a
// missing or faulty device.
func (c calibration) validate() error {
	vals := []int16{c.AC1, c.AC2, c.AC3, c.B1, c.B2, c.MB, c.MC, c.MD}
	for _, v := range vals {
		if v != 0 && v != -1 {
			return nil
		}
	}
	if c.AC4 == 0 || c.AC5 == 0 || c.AC6 == 0 {
		return fmt.Errorf("invalid calibration data (device not responding?)")
	}
	return nil
}

// compensate converts raw uncompensated temperature (ut) and pressure (up)
// readings into temperature (°C) and pressure (Pa) using the BMP180 datasheet
// fixed-point algorithm. oss is the oversampling setting (0-3).
func (c calibration) compensate(ut, up int32, oss uint) (tempC float64, pressurePa float64) {
	// Temperature.
	x1 := (ut - int32(c.AC6)) * int32(c.AC5) >> 15
	x2 := int32(c.MC) << 11 / (x1 + int32(c.MD))
	b5 := x1 + x2
	t := (b5 + 8) >> 4 // in 0.1 °C

	// Pressure.
	b6 := b5 - 4000
	x1 = (int32(c.B2) * (b6 * b6 >> 12)) >> 11
	x2 = int32(c.AC2) * b6 >> 11
	x3 := x1 + x2
	b3 := (((int32(c.AC1)*4 + x3) << oss) + 2) / 4
	x1 = int32(c.AC3) * b6 >> 13
	x2 = (int32(c.B1) * (b6 * b6 >> 12)) >> 16
	x3 = ((x1 + x2) + 2) >> 2
	b4 := uint32(c.AC4) * uint32(x3+32768) >> 15
	b7 := uint32(up-b3) * (50000 >> oss)

	var p int32
	if b7 < 0x80000000 {
		p = int32((b7 * 2) / b4)
	} else {
		p = int32((b7 / b4) * 2)
	}
	x1 = (p >> 8) * (p >> 8)
	x1 = (x1 * 3038) >> 16
	x2 = (-7357 * p) >> 16
	p = p + ((x1 + x2 + 3791) >> 4)

	return float64(t) / 10.0, float64(p)
}

// altitude estimates altitude (meters) from pressure using the international
// barometric formula referenced to standard sea-level pressure.
func altitude(pressurePa float64) float64 {
	const seaLevelPa = 101325.0
	return 44330.0 * (1.0 - math.Pow(pressurePa/seaLevelPa, 1.0/5.255))
}
