package gsp

import "errors"

// Frame parsing errors.
var (
	ErrFrameTooShort  = errors.New("gsp: frame too short")
	ErrInvalidSTX     = errors.New("gsp: invalid start byte")
	ErrInvalidETX     = errors.New("gsp: invalid end byte")
	ErrInvalidCRC     = errors.New("gsp: CRC mismatch")
	ErrPayloadTooLong = errors.New("gsp: payload exceeds maximum size")
	ErrBufferFull     = errors.New("gsp: buffer full")
	ErrIncomplete     = errors.New("gsp: incomplete frame")
)

// Protocol parsing errors.
var (
	ErrInvalidCommand = errors.New("gsp: invalid command")
	ErrMalformedMsg   = errors.New("gsp: malformed message")
)
