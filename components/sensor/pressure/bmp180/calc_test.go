package bmp180

import (
	"math"
	"testing"
)

// datasheetCal is the calibration example from the BMP180 datasheet (section
// 3.5), which pairs with UT=27898, UP=23843, oss=0 to give T=15.0°C, p=69964Pa.
func datasheetCal() calibration {
	return calibration{
		AC1: 408, AC2: -72, AC3: -14383,
		AC4: 32741, AC5: 32757, AC6: 23153,
		B1: 6190, B2: 4,
		MB: -32768, MC: -8711, MD: 2868,
	}
}

func TestCompensateDatasheetExample(t *testing.T) {
	cal := datasheetCal()
	tempC, pressurePa := cal.compensate(27898, 23843, 0)

	if math.Abs(tempC-15.0) > 0.05 {
		t.Errorf("temperature = %v °C, want 15.0", tempC)
	}
	if math.Abs(pressurePa-69964) > 1 {
		t.Errorf("pressure = %v Pa, want 69964", pressurePa)
	}
}

func TestParseCalibration(t *testing.T) {
	// Encode the datasheet coefficients big-endian and round-trip them.
	cal := datasheetCal()
	raw := make([]byte, 22)
	put := func(i int, v uint16) { raw[i] = byte(v >> 8); raw[i+1] = byte(v) }
	put(0, uint16(cal.AC1))
	put(2, uint16(cal.AC2))
	put(4, uint16(cal.AC3))
	put(6, cal.AC4)
	put(8, cal.AC5)
	put(10, cal.AC6)
	put(12, uint16(cal.B1))
	put(14, uint16(cal.B2))
	put(16, uint16(cal.MB))
	put(18, uint16(cal.MC))
	put(20, uint16(cal.MD))

	got, err := parseCalibration(raw)
	if err != nil {
		t.Fatalf("parseCalibration: %v", err)
	}
	if got != cal {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, cal)
	}
}

func TestParseCalibrationRejectsEmpty(t *testing.T) {
	if _, err := parseCalibration(make([]byte, 22)); err == nil {
		t.Errorf("expected error for all-zero calibration")
	}
}

func TestAltitudeAtSeaLevel(t *testing.T) {
	if a := altitude(101325.0); math.Abs(a) > 0.01 {
		t.Errorf("altitude at sea-level pressure = %v, want ~0", a)
	}
	if a := altitude(90000.0); a < 900 || a > 1100 {
		t.Errorf("altitude at 900 hPa = %v m, expected ~1000 m", a)
	}
}
