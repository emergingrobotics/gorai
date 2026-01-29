package gsp

import "testing"

func TestCRC16(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		expected uint16
	}{
		{
			name:     "empty",
			data:     []byte{},
			expected: 0xFFFF, // init value unchanged
		},
		{
			name:     "single byte",
			data:     []byte{0x00},
			expected: 0xE1F0,
		},
		{
			name:     "ping message",
			data:     []byte{0x00, 0x06, 'P', 'I', 'N', 'G', '\r', '\n'},
			expected: 0xDA76, // actual computed value
		},
		{
			name:     "short data",
			data:     []byte{0x01, 0x02, 0x03},
			expected: 0xADAD, // actual computed value
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CRC16(tt.data)
			if got != tt.expected {
				t.Errorf("CRC16(%v) = 0x%04X, want 0x%04X", tt.data, got, tt.expected)
			}
		})
	}
}

func TestValidateCRC(t *testing.T) {
	data := []byte{0x01, 0x02, 0x03}
	crc := CRC16(data)

	if !ValidateCRC(data, crc) {
		t.Error("ValidateCRC should return true for correct CRC")
	}

	if ValidateCRC(data, crc+1) {
		t.Error("ValidateCRC should return false for incorrect CRC")
	}
}
