package keyboard

import (
	"encoding/binary"
	"fmt"
	"unsafe"

	"golang.org/x/sys/unix"
)

// Linux evdev event types
const (
	evSyn uint16 = 0x00 // Synchronization events
	evKey uint16 = 0x01 // Key/button events
)

// Key event values
const (
	keyRelease int32 = 0 // Key released
	keyPress   int32 = 1 // Key pressed
	keyRepeat  int32 = 2 // Key held (repeat)
)

// inputEventSize is the size of input_event struct on 64-bit Linux.
// struct input_event {
//     struct timeval time;  // 16 bytes (8 sec + 8 usec)
//     __u16 type;           // 2 bytes
//     __u16 code;           // 2 bytes
//     __s32 value;          // 4 bytes
// }
const inputEventSize = 24

// inputEvent represents a Linux input_event structure.
type inputEvent struct {
	TimeSec  int64  // Seconds
	TimeUsec int64  // Microseconds
	Type     uint16 // Event type (EV_KEY = 1)
	Code     uint16 // Key code
	Value    int32  // 0=release, 1=press, 2=repeat
}

// parseInputEvent parses a raw 24-byte buffer into an inputEvent.
func parseInputEvent(buf []byte) (inputEvent, error) {
	if len(buf) < inputEventSize {
		return inputEvent{}, fmt.Errorf("buffer too small: need %d bytes, got %d", inputEventSize, len(buf))
	}

	return inputEvent{
		TimeSec:  int64(binary.LittleEndian.Uint64(buf[0:8])),
		TimeUsec: int64(binary.LittleEndian.Uint64(buf[8:16])),
		Type:     binary.LittleEndian.Uint16(buf[16:18]),
		Code:     binary.LittleEndian.Uint16(buf[18:20]),
		Value:    int32(binary.LittleEndian.Uint32(buf[20:24])),
	}, nil
}

// isKeyEvent returns true if this is a key press/release/repeat event.
func (e inputEvent) isKeyEvent() bool {
	return e.Type == evKey
}

// isPress returns true if this is a key press event.
func (e inputEvent) isPress() bool {
	return e.Value == keyPress
}

// isRelease returns true if this is a key release event.
func (e inputEvent) isRelease() bool {
	return e.Value == keyRelease
}

// isRepeat returns true if this is a key repeat event.
func (e inputEvent) isRepeat() bool {
	return e.Value == keyRepeat
}

// IOCTL commands for evdev
// From linux/input.h
const (
	// EVIOCGRAB grabs/releases the device for exclusive access.
	// arg: 1 to grab, 0 to release
	ioctlEviocgrab = 0x40044590

	// EVIOCGNAME gets the device name.
	// arg: buffer to receive name
	ioctlEviocgname = 0x81004506
)

// grabDevice grabs the device for exclusive access.
func grabDevice(fd int) error {
	return unix.IoctlSetInt(fd, ioctlEviocgrab, 1)
}

// releaseDevice releases the device from exclusive access.
func releaseDevice(fd int) error {
	return unix.IoctlSetInt(fd, ioctlEviocgrab, 0)
}

// getDeviceName retrieves the name of the input device.
func getDeviceName(fd int) (string, error) {
	buf := make([]byte, 256)
	// EVIOCGNAME(len) = _IOC(_IOC_READ, 'E', 0x06, len)
	// For 256 bytes: 0x81004506
	_, _, errno := unix.Syscall(
		unix.SYS_IOCTL,
		uintptr(fd),
		uintptr(0x81004506), // EVIOCGNAME(256)
		uintptr(unsafe.Pointer(&buf[0])),
	)
	if errno != 0 {
		return "", fmt.Errorf("ioctl EVIOCGNAME failed: %v", errno)
	}

	// Find null terminator
	for i, b := range buf {
		if b == 0 {
			return string(buf[:i]), nil
		}
	}
	return string(buf), nil
}

