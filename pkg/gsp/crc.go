// Package gsp implements the Gorai Serial Protocol for bridging
// serial devices to NATS messaging.
package gsp

// CRC-16-CCITT parameters matching the RP2040 firmware.
const (
	crcPolynomial = 0x1021
	crcInit       = 0xFFFF
)

// CRC16 computes CRC-16-CCITT checksum.
// This implementation matches the RP2040 firmware exactly:
// polynomial 0x1021, init 0xFFFF, no reflection.
func CRC16(data []byte) uint16 {
	crc := uint16(crcInit)
	for _, b := range data {
		crc ^= uint16(b) << 8
		for i := 0; i < 8; i++ {
			if crc&0x8000 != 0 {
				crc = (crc << 1) ^ crcPolynomial
			} else {
				crc <<= 1
			}
		}
	}
	return crc
}

// ValidateCRC checks if the CRC matches the expected value.
func ValidateCRC(data []byte, expected uint16) bool {
	return CRC16(data) == expected
}
