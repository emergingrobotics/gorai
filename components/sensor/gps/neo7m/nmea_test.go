package neo7m

import (
	"math"
	"testing"
)

func approx(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func TestParseSentenceChecksum(t *testing.T) {
	good := "$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*47"
	if _, ok := parseSentence(good); !ok {
		t.Errorf("valid checksum rejected")
	}
	bad := "$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*00"
	if _, ok := parseSentence(bad); ok {
		t.Errorf("invalid checksum accepted")
	}
	if _, ok := parseSentence(""); ok {
		t.Errorf("empty sentence accepted")
	}
}

func TestApplyGGA(t *testing.T) {
	var fix gpsFix
	if !applySentence(&fix, "$GPGGA,123519,4807.038,N,01131.000,E,1,08,0.9,545.4,M,46.9,M,,*47") {
		t.Fatalf("GGA not applied")
	}
	if !approx(fix.lat, 48.1173, 1e-3) {
		t.Errorf("lat = %v, want ~48.1173", fix.lat)
	}
	if !approx(fix.lon, 11.5167, 1e-3) {
		t.Errorf("lon = %v, want ~11.5167", fix.lon)
	}
	if fix.satellites != 8 {
		t.Errorf("sats = %d, want 8", fix.satellites)
	}
	if !approx(fix.hdop, 0.9, 1e-6) {
		t.Errorf("hdop = %v, want 0.9", fix.hdop)
	}
	if !approx(fix.altM, 545.4, 1e-6) {
		t.Errorf("alt = %v, want 545.4", fix.altM)
	}
	if fix.quality != 1 || !fix.hasFix {
		t.Errorf("quality = %d hasFix = %v, want 1/true", fix.quality, fix.hasFix)
	}
}

func TestApplyRMC(t *testing.T) {
	var fix gpsFix
	if !applySentence(&fix, "$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W*6A") {
		t.Fatalf("RMC not applied")
	}
	if !approx(fix.lat, 48.1173, 1e-3) {
		t.Errorf("lat = %v, want ~48.1173", fix.lat)
	}
	if !approx(fix.headingDeg, 84.4, 1e-6) {
		t.Errorf("heading = %v, want 84.4", fix.headingDeg)
	}
	// 22.4 knots -> ~11.52 m/s
	if !approx(fix.speedMps, 22.4*knotsToMps, 1e-6) {
		t.Errorf("speed = %v", fix.speedMps)
	}
}

func TestParseCoordinateSouthWest(t *testing.T) {
	lat, err := parseCoordinate("4807.038", "S")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !approx(lat, -48.1173, 1e-3) {
		t.Errorf("south lat = %v, want ~-48.1173", lat)
	}
}

func TestUnknownSentenceIgnored(t *testing.T) {
	var fix gpsFix
	if applySentence(&fix, "$GPGSV,3,1,11,01,40,083,46*7B") {
		t.Errorf("GSV should be ignored")
	}
}
