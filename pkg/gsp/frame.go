package gsp

// Frame constants matching the RP2040 firmware.
const (
	STX = 0x02 // Start of frame
	ETX = 0x03 // End of frame

	// MaxPayloadSize is 50% larger than the longest expected command.
	// A 16-channel batch command is ~480 bytes, so 480 * 1.5 = 720.
	MaxPayloadSize = 720
	MinFrameSize   = 6 // STX + LEN(2) + CRC(2) + ETX
)

// Frame represents a parsed GSP frame.
type Frame struct {
	Payload []byte
}

// BuildFrame creates a GSP frame from payload.
// Format: [STX:1][LENGTH:2 big-endian][PAYLOAD:n][CRC:2 big-endian][ETX:1]
// CRC is calculated over LENGTH + PAYLOAD.
func BuildFrame(payload []byte) ([]byte, error) {
	if len(payload) > MaxPayloadSize {
		return nil, ErrPayloadTooLong
	}

	frameLen := 1 + 2 + len(payload) + 2 + 1
	frame := make([]byte, frameLen)

	// STX
	frame[0] = STX

	// Length (big-endian)
	frame[1] = byte(len(payload) >> 8)
	frame[2] = byte(len(payload))

	// Payload
	copy(frame[3:], payload)

	// CRC over length + payload
	crc := CRC16(frame[1 : 3+len(payload)])
	frame[3+len(payload)] = byte(crc >> 8)
	frame[3+len(payload)+1] = byte(crc)

	// ETX
	frame[frameLen-1] = ETX

	return frame, nil
}

// Parser handles incremental frame parsing from a serial stream.
type Parser struct {
	buf    []byte
	bufLen int
	state  parserState
	payLen uint16
}

type parserState int

const (
	stateWaitSTX parserState = iota
	stateReadLenHigh
	stateReadLenLow
	stateReadPayload
	stateReadCRCHigh
	stateReadCRCLow
	stateReadETX
)

// NewParser creates a new GSP frame parser.
func NewParser(bufSize int) *Parser {
	if bufSize < MaxPayloadSize+2 {
		bufSize = MaxPayloadSize + 2 // length bytes + payload
	}
	return &Parser{
		buf:   make([]byte, bufSize),
		state: stateWaitSTX,
	}
}

// Reset clears the parser state.
func (p *Parser) Reset() {
	p.bufLen = 0
	p.state = stateWaitSTX
	p.payLen = 0
}

// Feed processes incoming bytes and returns complete frames.
// Returns nil if no complete frame is available yet.
// Note: This only returns the first complete frame. Use FeedAll to process
// multiple frames from a single buffer.
func (p *Parser) Feed(data []byte) (*Frame, error) {
	for _, b := range data {
		frame, err := p.feedByte(b)
		if err != nil {
			p.Reset()
			return nil, err
		}
		if frame != nil {
			return frame, nil
		}
	}
	return nil, nil
}

// FeedAll processes incoming bytes and returns the first complete frame
// along with the number of bytes consumed. This allows the caller to
// process remaining bytes for additional frames.
func (p *Parser) FeedAll(data []byte) (*Frame, int, error) {
	for i, b := range data {
		frame, err := p.feedByte(b)
		if err != nil {
			p.Reset()
			return nil, i + 1, err
		}
		if frame != nil {
			return frame, i + 1, nil
		}
	}
	return nil, len(data), nil
}

func (p *Parser) feedByte(b byte) (*Frame, error) {
	switch p.state {
	case stateWaitSTX:
		if b == STX {
			p.state = stateReadLenHigh
			p.bufLen = 0
		}
		// Ignore bytes until STX

	case stateReadLenHigh:
		p.payLen = uint16(b) << 8
		// Store length high byte for CRC
		p.buf[0] = b
		p.bufLen = 1
		p.state = stateReadLenLow

	case stateReadLenLow:
		p.payLen |= uint16(b)
		// Store length low byte for CRC
		p.buf[1] = b
		p.bufLen = 2
		if p.payLen > MaxPayloadSize {
			return nil, ErrPayloadTooLong
		}
		if p.payLen == 0 {
			p.state = stateReadCRCHigh
		} else {
			p.state = stateReadPayload
		}

	case stateReadPayload:
		if p.bufLen >= len(p.buf) {
			return nil, ErrBufferFull
		}
		p.buf[p.bufLen] = b
		p.bufLen++
		// bufLen = 2 + payload bytes received
		if p.bufLen >= int(p.payLen)+2 {
			p.state = stateReadCRCHigh
		}

	case stateReadCRCHigh:
		// Store CRC high byte temporarily
		if p.bufLen >= len(p.buf)-1 {
			return nil, ErrBufferFull
		}
		p.buf[p.bufLen] = b
		p.bufLen++
		p.state = stateReadCRCLow

	case stateReadCRCLow:
		// CRC high is at buf[2+payLen]
		crcHigh := p.buf[2+p.payLen]
		crcExpected := (uint16(crcHigh) << 8) | uint16(b)

		// Validate CRC over length + payload (buf[0:2+payLen])
		crcActual := CRC16(p.buf[:2+p.payLen])
		if crcActual != crcExpected {
			return nil, ErrInvalidCRC
		}
		p.state = stateReadETX

	case stateReadETX:
		if b != ETX {
			return nil, ErrInvalidETX
		}
		// Frame complete - payload is at buf[2:2+payLen]
		payload := make([]byte, p.payLen)
		copy(payload, p.buf[2:2+p.payLen])
		p.Reset()
		return &Frame{Payload: payload}, nil
	}

	return nil, nil
}
