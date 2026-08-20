package neo7m

import (
	"fmt"
	"strconv"
	"strings"
)

// gpsFix holds the latest parsed GNSS state.
type gpsFix struct {
	lat        float64
	lon        float64
	altM       float64
	hdop       float64
	quality    int
	satellites int
	headingDeg float64
	speedMps   float64
	hasFix     bool
}

// knotsToMps converts speed in knots to meters per second.
const knotsToMps = 0.514444

// parseSentence validates an NMEA sentence's checksum (when present) and
// returns its comma-separated fields (including the leading type token, e.g.
// "GPGGA"). ok is false for empty or corrupt sentences.
func parseSentence(line string) (fields []string, ok bool) {
	line = strings.TrimSpace(line)
	if len(line) < 6 || line[0] != '$' {
		return nil, false
	}
	body := line[1:]

	// Verify the checksum if one is present ("...*HH").
	if star := strings.LastIndexByte(body, '*'); star >= 0 {
		payload := body[:star]
		sum := body[star+1:]
		want, err := strconv.ParseUint(sum, 16, 8)
		if err != nil {
			return nil, false
		}
		if byte(want) != nmeaChecksum(payload) {
			return nil, false
		}
		body = payload
	}

	return strings.Split(body, ","), true
}

// nmeaChecksum computes the XOR checksum of the payload between '$' and '*'.
func nmeaChecksum(payload string) byte {
	var cs byte
	for i := 0; i < len(payload); i++ {
		cs ^= payload[i]
	}
	return cs
}

// sentenceType returns the 3-letter message type (e.g. "GGA") from a field
// token like "GPGGA"/"GNGGA", stripping the 2-char talker id.
func sentenceType(token string) string {
	if len(token) < 5 {
		return ""
	}
	return token[2:]
}

// applySentence parses a single NMEA sentence into the fix. It returns true if
// the sentence was a recognized position/velocity message.
func applySentence(fix *gpsFix, line string) bool {
	fields, ok := parseSentence(line)
	if !ok || len(fields) == 0 {
		return false
	}
	switch sentenceType(fields[0]) {
	case "GGA":
		return applyGGA(fix, fields)
	case "RMC":
		return applyRMC(fix, fields)
	default:
		return false
	}
}

// applyGGA parses a GGA (fix data) sentence.
// Fields: 0=type 1=time 2=lat 3=N/S 4=lon 5=E/W 6=quality 7=sats 8=hdop 9=alt 10=altUnit ...
func applyGGA(fix *gpsFix, f []string) bool {
	if len(f) < 10 {
		return false
	}
	quality, _ := strconv.Atoi(f[6])
	fix.quality = quality
	fix.hasFix = quality > 0

	if lat, err := parseCoordinate(f[2], f[3]); err == nil {
		fix.lat = lat
	}
	if lon, err := parseCoordinate(f[4], f[5]); err == nil {
		fix.lon = lon
	}
	if sats, err := strconv.Atoi(f[7]); err == nil {
		fix.satellites = sats
	}
	if hdop, err := strconv.ParseFloat(f[8], 64); err == nil {
		fix.hdop = hdop
	}
	if alt, err := strconv.ParseFloat(f[9], 64); err == nil {
		fix.altM = alt
	}
	return true
}

// applyRMC parses an RMC (recommended minimum) sentence.
// Fields: 0=type 1=time 2=status 3=lat 4=N/S 5=lon 6=E/W 7=speed(kn) 8=course ...
func applyRMC(fix *gpsFix, f []string) bool {
	if len(f) < 9 {
		return false
	}
	active := f[2] == "A"
	if active {
		fix.hasFix = true
	}
	if lat, err := parseCoordinate(f[3], f[4]); err == nil {
		fix.lat = lat
	}
	if lon, err := parseCoordinate(f[5], f[6]); err == nil {
		fix.lon = lon
	}
	if kn, err := strconv.ParseFloat(f[7], 64); err == nil {
		fix.speedMps = kn * knotsToMps
	}
	if course, err := strconv.ParseFloat(f[8], 64); err == nil {
		fix.headingDeg = course
	}
	return true
}

// parseCoordinate converts an NMEA ddmm.mmmm / dddmm.mmmm value plus its
// hemisphere indicator into signed decimal degrees.
func parseCoordinate(value, hemi string) (float64, error) {
	if value == "" || hemi == "" {
		return 0, fmt.Errorf("empty coordinate")
	}
	dot := strings.IndexByte(value, '.')
	if dot < 3 {
		return 0, fmt.Errorf("invalid coordinate %q", value)
	}
	// Degrees are all but the last two digits before the decimal point.
	degEnd := dot - 2
	deg, err := strconv.ParseFloat(value[:degEnd], 64)
	if err != nil {
		return 0, err
	}
	min, err := strconv.ParseFloat(value[degEnd:], 64)
	if err != nil {
		return 0, err
	}
	dd := deg + min/60.0
	switch hemi {
	case "S", "W":
		dd = -dd
	}
	return dd, nil
}
