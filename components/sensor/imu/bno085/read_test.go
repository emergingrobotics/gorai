package bno085

import (
	"context"
	"io"
	"log/slog"
	"math"
	"testing"
)

// fakeI2CDevice models the BNO08x I2C behavior: every read is a fresh
// transaction that returns bytes from the start of the current SHTP packet. A
// packet is only consumed (advancing to the next) once a read requests at least
// the full packet length. Reading only the 4-byte header therefore does NOT
// advance the stream, which is the trap the driver must avoid.
type fakeI2CDevice struct {
	packets [][]byte
}

func (f *fakeI2CDevice) Read(ctx context.Context, n int) ([]byte, error) {
	out := make([]byte, n)
	if len(f.packets) == 0 {
		return out, nil // zero header => empty packet
	}
	front := f.packets[0]
	copy(out, front) // copies min(n, len(front)), zero-padding the rest
	if n >= len(front) {
		f.packets = f.packets[1:]
	}
	return out, nil
}

func (f *fakeI2CDevice) Address() uint16                                          { return 0x4A }
func (f *fakeI2CDevice) Write(ctx context.Context, data []byte) error             { return nil }
func (f *fakeI2CDevice) WriteRead(ctx context.Context, w []byte, n int) ([]byte, error) {
	return make([]byte, n), nil
}
func (f *fakeI2CDevice) ReadReg(ctx context.Context, reg byte, n int) ([]byte, error) {
	return make([]byte, n), nil
}
func (f *fakeI2CDevice) WriteReg(ctx context.Context, reg byte, data []byte) error { return nil }
func (f *fakeI2CDevice) ReadByteCtx(ctx context.Context) (byte, error)             { return 0, nil }
func (f *fakeI2CDevice) WriteByteCtx(ctx context.Context, b byte) error            { return nil }
func (f *fakeI2CDevice) ReadByteReg(ctx context.Context, reg byte) (byte, error)   { return 0, nil }
func (f *fakeI2CDevice) WriteByteReg(ctx context.Context, reg, value byte) error   { return nil }

// frame wraps a cargo in a 4-byte SHTP header for the given channel.
func frame(channel byte, cargo []byte) []byte {
	total := len(cargo) + 4
	pkt := make([]byte, total)
	pkt[0] = byte(total & 0xFF)
	pkt[1] = byte((total >> 8) & 0x7F)
	pkt[2] = channel
	pkt[3] = 0
	copy(pkt[4:], cargo)
	return pkt
}

func newTestAHRS(dev *fakeI2CDevice) *AHRS {
	return &AHRS{
		logger: slog.New(slog.NewTextHandler(io.Discard, nil)),
		dev:    dev,
	}
}

// TestReadPacketFramesFullPacket verifies the driver reads header + cargo in one
// transaction and decodes the input reports. This is the regression guard for
// the two-transaction framing bug where the cargo read re-read the header.
func TestReadPacketFramesFullPacket(t *testing.T) {
	// Accelerometer report (id 0x01, 10 bytes): x raw=256 -> 1.0, y=0, z=512 -> 2.0
	// at Q8, with calibration status 3.
	accel := []byte{
		reportAccelerometer, 0x00, 0x03, 0x00,
		0x00, 0x01, // x = 256
		0x00, 0x00, // y = 0
		0x00, 0x02, // z = 512
	}
	dev := &fakeI2CDevice{packets: [][]byte{frame(chInput, accel)}}
	a := newTestAHRS(dev)

	res, err := a.readPacket(context.Background())
	if err != nil {
		t.Fatalf("readPacket: %v", err)
	}
	if res.channel != chInput {
		t.Fatalf("channel = %d, want %d", res.channel, chInput)
	}
	if res.reports != 1 {
		t.Fatalf("reports = %d, want 1", res.reports)
	}

	const eps = 1e-6
	if math.Abs(a.state.ax-1.0) > eps || math.Abs(a.state.az-2.0) > eps {
		t.Fatalf("accel = (%v, %v, %v), want x=1.0 z=2.0", a.state.ax, a.state.ay, a.state.az)
	}
	if a.state.calAccel != 3 {
		t.Fatalf("calAccel = %d, want 3", a.state.calAccel)
	}

	// The packet must have been consumed; a follow-up read yields an empty packet.
	res2, err := a.readPacket(context.Background())
	if err != nil {
		t.Fatalf("second readPacket: %v", err)
	}
	if res2.reports != 0 || len(res2.cargo) != 0 {
		t.Fatalf("expected empty follow-up packet, got reports=%d cargo=%d", res2.reports, len(res2.cargo))
	}
}

// TestReadPacketEmpty verifies a zero-length header yields no reports and a
// non-negative channel (i.e. it is not treated as a bad-header failure).
func TestReadPacketEmpty(t *testing.T) {
	dev := &fakeI2CDevice{}
	a := newTestAHRS(dev)

	res, err := a.readPacket(context.Background())
	if err != nil {
		t.Fatalf("readPacket: %v", err)
	}
	if res.channel < 0 {
		t.Fatalf("empty read should report channel 0, got %d", res.channel)
	}
	if res.reports != 0 {
		t.Fatalf("reports = %d, want 0", res.reports)
	}
}
