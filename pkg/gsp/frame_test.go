package gsp

import (
	"bytes"
	"testing"
)

func TestBuildFrame(t *testing.T) {
	payload := []byte("PING\r\n")
	frame, err := BuildFrame(payload)
	if err != nil {
		t.Fatalf("BuildFrame failed: %v", err)
	}

	// Check frame structure
	if frame[0] != STX {
		t.Errorf("frame[0] = 0x%02X, want STX (0x%02X)", frame[0], STX)
	}

	if frame[len(frame)-1] != ETX {
		t.Errorf("frame[last] = 0x%02X, want ETX (0x%02X)", frame[len(frame)-1], ETX)
	}

	// Check length encoding
	length := int(frame[1])<<8 | int(frame[2])
	if length != len(payload) {
		t.Errorf("length = %d, want %d", length, len(payload))
	}

	// Check payload
	extractedPayload := frame[3 : 3+length]
	if !bytes.Equal(extractedPayload, payload) {
		t.Errorf("payload = %q, want %q", extractedPayload, payload)
	}

	// Check CRC (over length + payload)
	crcData := frame[1 : 3+length]
	expectedCRC := CRC16(crcData)
	actualCRC := uint16(frame[3+length])<<8 | uint16(frame[3+length+1])
	if actualCRC != expectedCRC {
		t.Errorf("CRC = 0x%04X, want 0x%04X", actualCRC, expectedCRC)
	}
}

func TestBuildFramePayloadTooLong(t *testing.T) {
	payload := make([]byte, MaxPayloadSize+1)
	_, err := BuildFrame(payload)
	if err != ErrPayloadTooLong {
		t.Errorf("BuildFrame with oversized payload: err = %v, want ErrPayloadTooLong", err)
	}
}

func TestBuildFrameEmptyPayload(t *testing.T) {
	frame, err := BuildFrame([]byte{})
	if err != nil {
		t.Fatalf("BuildFrame with empty payload failed: %v", err)
	}

	// Should be: STX + len(0,0) + CRC(2) + ETX = 6 bytes
	if len(frame) != MinFrameSize {
		t.Errorf("empty frame length = %d, want %d", len(frame), MinFrameSize)
	}
}

func TestParserRoundTrip(t *testing.T) {
	testCases := [][]byte{
		[]byte("PING\r\n"),
		[]byte("PONG\r\n"),
		[]byte("PUB sensor.temp 4\r\n25.3"),
		[]byte("SUB motor.command 1\r\n"),
		[]byte("MSG motor.command 1 12\r\n{\"power\":0.5}"),
	}

	for _, original := range testCases {
		t.Run(string(original[:4]), func(t *testing.T) {
			frame, err := BuildFrame(original)
			if err != nil {
				t.Fatalf("BuildFrame failed: %v", err)
			}

			parser := NewParser(MaxPayloadSize + 64)
			parsed, err := parser.Feed(frame)
			if err != nil {
				t.Fatalf("Parser.Feed failed: %v", err)
			}
			if parsed == nil {
				t.Fatal("Parser.Feed returned nil frame")
			}

			if !bytes.Equal(parsed.Payload, original) {
				t.Errorf("parsed payload = %q, want %q", parsed.Payload, original)
			}
		})
	}
}

func TestParserIncrementalFeed(t *testing.T) {
	payload := []byte("PUB test 5\r\nhello")
	frame, _ := BuildFrame(payload)

	parser := NewParser(1024)

	// Feed one byte at a time
	var result *Frame
	for i := 0; i < len(frame); i++ {
		parsed, err := parser.Feed(frame[i : i+1])
		if err != nil {
			t.Fatalf("Parser.Feed failed at byte %d: %v", i, err)
		}
		if parsed != nil {
			result = parsed
		}
	}

	if result == nil {
		t.Fatal("Parser did not produce a frame")
	}
	if !bytes.Equal(result.Payload, payload) {
		t.Errorf("payload = %q, want %q", result.Payload, payload)
	}
}

func TestParserCRCError(t *testing.T) {
	payload := []byte("PING\r\n")
	frame, _ := BuildFrame(payload)

	// Corrupt the CRC
	frame[len(frame)-3] ^= 0xFF

	parser := NewParser(1024)
	_, err := parser.Feed(frame)
	if err != ErrInvalidCRC {
		t.Errorf("err = %v, want ErrInvalidCRC", err)
	}
}

func TestParserInvalidETX(t *testing.T) {
	payload := []byte("PING\r\n")
	frame, _ := BuildFrame(payload)

	// Corrupt the ETX
	frame[len(frame)-1] = 0xFF

	parser := NewParser(1024)
	_, err := parser.Feed(frame)
	if err != ErrInvalidETX {
		t.Errorf("err = %v, want ErrInvalidETX", err)
	}
}

func TestParserSkipsGarbage(t *testing.T) {
	payload := []byte("PING\r\n")
	frame, _ := BuildFrame(payload)

	// Prepend garbage
	garbage := []byte{0xFF, 0x00, 0xAA, 0x55}
	data := append(garbage, frame...)

	parser := NewParser(1024)
	parsed, err := parser.Feed(data)
	if err != nil {
		t.Fatalf("Parser.Feed failed: %v", err)
	}
	if parsed == nil {
		t.Fatal("Parser did not produce a frame")
	}
	if !bytes.Equal(parsed.Payload, payload) {
		t.Errorf("payload = %q, want %q", parsed.Payload, payload)
	}
}

func TestParserReset(t *testing.T) {
	parser := NewParser(1024)

	// Feed partial frame
	parser.Feed([]byte{STX, 0x00, 0x04, 'T', 'E'})

	// Reset
	parser.Reset()

	// Feed complete frame
	payload := []byte("OK")
	frame, _ := BuildFrame(payload)
	parsed, err := parser.Feed(frame)
	if err != nil {
		t.Fatalf("Parser.Feed after reset failed: %v", err)
	}
	if parsed == nil {
		t.Fatal("Parser did not produce a frame after reset")
	}
	if !bytes.Equal(parsed.Payload, payload) {
		t.Errorf("payload = %q, want %q", parsed.Payload, payload)
	}
}
