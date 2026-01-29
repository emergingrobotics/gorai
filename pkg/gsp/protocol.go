package gsp

// Protocol version.
const Version = "v0.0.1"

// Protocol commands (NATS-like).
const (
	CmdPUB     = "PUB"
	CmdSUB     = "SUB"
	CmdUNSUB   = "UNSUB"
	CmdMSG     = "MSG"
	CmdPING    = "PING"
	CmdPONG    = "PONG"
	CmdVER     = "VER"
	CmdVERSION = "VERSION"
	CmdOK      = "+OK"
	CmdERR     = "-ERR"
)

// Message represents a parsed GSP protocol message.
type Message struct {
	Command string
	Subject string
	SubID   string // For SUB/MSG
	Payload []byte
}

// ParseMessage parses the payload of a GSP frame into a Message.
// Format:
//
//	PUB <subject> <len>\r\n<payload>
//	SUB <subject> <id>\r\n
//	UNSUB <id>\r\n
//	MSG <subject> <id> <len>\r\n<payload>
//	PING\r\n
//	PONG\r\n
//	VER\r\n
//	VERSION <version>\r\n
func ParseMessage(payload []byte) (*Message, error) {
	if len(payload) == 0 {
		return nil, ErrMalformedMsg
	}

	// Find command (first word)
	cmdEnd := 0
	for cmdEnd < len(payload) && payload[cmdEnd] != ' ' && payload[cmdEnd] != '\r' {
		cmdEnd++
	}
	if cmdEnd == 0 {
		return nil, ErrMalformedMsg
	}

	cmd := string(payload[:cmdEnd])
	rest := payload[cmdEnd:]

	switch cmd {
	case CmdPING:
		return &Message{Command: CmdPING}, nil

	case CmdPONG:
		return &Message{Command: CmdPONG}, nil

	case CmdVER:
		return &Message{Command: CmdVER}, nil

	case CmdVERSION:
		return parseVersion(rest)

	case CmdPUB:
		return parsePub(rest)

	case CmdSUB:
		return parseSub(rest)

	case CmdUNSUB:
		return parseUnsub(rest)

	case CmdMSG:
		return parseMsg(rest)

	default:
		return nil, ErrInvalidCommand
	}
}

// parsePub parses: " <subject> <len>\r\n<payload>"
func parsePub(data []byte) (*Message, error) {
	// Skip leading space
	if len(data) == 0 || data[0] != ' ' {
		return nil, ErrMalformedMsg
	}
	data = data[1:]

	// Find subject
	subjectEnd := 0
	for subjectEnd < len(data) && data[subjectEnd] != ' ' {
		subjectEnd++
	}
	if subjectEnd == 0 || subjectEnd >= len(data) {
		return nil, ErrMalformedMsg
	}
	subject := string(data[:subjectEnd])
	data = data[subjectEnd+1:]

	// Parse length
	payloadLen := 0
	lenEnd := 0
	for lenEnd < len(data) && data[lenEnd] >= '0' && data[lenEnd] <= '9' {
		payloadLen = payloadLen*10 + int(data[lenEnd]-'0')
		lenEnd++
	}
	if lenEnd == 0 {
		return nil, ErrMalformedMsg
	}
	data = data[lenEnd:]

	// Expect \r\n
	if len(data) < 2 || data[0] != '\r' || data[1] != '\n' {
		return nil, ErrMalformedMsg
	}
	data = data[2:]

	// Extract payload
	if len(data) < payloadLen {
		return nil, ErrMalformedMsg
	}

	return &Message{
		Command: CmdPUB,
		Subject: subject,
		Payload: data[:payloadLen],
	}, nil
}

// parseSub parses: " <subject> <id>\r\n"
func parseSub(data []byte) (*Message, error) {
	if len(data) == 0 || data[0] != ' ' {
		return nil, ErrMalformedMsg
	}
	data = data[1:]

	// Find subject
	subjectEnd := 0
	for subjectEnd < len(data) && data[subjectEnd] != ' ' {
		subjectEnd++
	}
	if subjectEnd == 0 || subjectEnd >= len(data) {
		return nil, ErrMalformedMsg
	}
	subject := string(data[:subjectEnd])
	data = data[subjectEnd+1:]

	// Find ID (until \r\n)
	idEnd := 0
	for idEnd < len(data) && data[idEnd] != '\r' {
		idEnd++
	}
	if idEnd == 0 {
		return nil, ErrMalformedMsg
	}
	id := string(data[:idEnd])

	return &Message{
		Command: CmdSUB,
		Subject: subject,
		SubID:   id,
	}, nil
}

// parseUnsub parses: " <id>\r\n"
func parseUnsub(data []byte) (*Message, error) {
	if len(data) == 0 || data[0] != ' ' {
		return nil, ErrMalformedMsg
	}
	data = data[1:]

	idEnd := 0
	for idEnd < len(data) && data[idEnd] != '\r' {
		idEnd++
	}
	if idEnd == 0 {
		return nil, ErrMalformedMsg
	}

	return &Message{
		Command: CmdUNSUB,
		SubID:   string(data[:idEnd]),
	}, nil
}

// parseMsg parses: " <subject> <id> <len>\r\n<payload>"
func parseMsg(data []byte) (*Message, error) {
	if len(data) == 0 || data[0] != ' ' {
		return nil, ErrMalformedMsg
	}
	data = data[1:]

	// Find subject
	subjectEnd := 0
	for subjectEnd < len(data) && data[subjectEnd] != ' ' {
		subjectEnd++
	}
	if subjectEnd == 0 || subjectEnd >= len(data) {
		return nil, ErrMalformedMsg
	}
	subject := string(data[:subjectEnd])
	data = data[subjectEnd+1:]

	// Find ID
	idEnd := 0
	for idEnd < len(data) && data[idEnd] != ' ' {
		idEnd++
	}
	if idEnd == 0 || idEnd >= len(data) {
		return nil, ErrMalformedMsg
	}
	id := string(data[:idEnd])
	data = data[idEnd+1:]

	// Parse length
	payloadLen := 0
	lenEnd := 0
	for lenEnd < len(data) && data[lenEnd] >= '0' && data[lenEnd] <= '9' {
		payloadLen = payloadLen*10 + int(data[lenEnd]-'0')
		lenEnd++
	}
	if lenEnd == 0 {
		return nil, ErrMalformedMsg
	}
	data = data[lenEnd:]

	// Expect \r\n
	if len(data) < 2 || data[0] != '\r' || data[1] != '\n' {
		return nil, ErrMalformedMsg
	}
	data = data[2:]

	if len(data) < payloadLen {
		return nil, ErrMalformedMsg
	}

	return &Message{
		Command: CmdMSG,
		Subject: subject,
		SubID:   id,
		Payload: data[:payloadLen],
	}, nil
}

// FormatPub creates a PUB message payload.
func FormatPub(subject string, payload []byte) []byte {
	// "PUB <subject> <len>\r\n<payload>"
	header := []byte("PUB " + subject + " ")
	lenStr := formatInt(len(payload))

	result := make([]byte, 0, len(header)+len(lenStr)+2+len(payload))
	result = append(result, header...)
	result = append(result, lenStr...)
	result = append(result, '\r', '\n')
	result = append(result, payload...)
	return result
}

// FormatSub creates a SUB message payload.
func FormatSub(subject string, id string) []byte {
	return []byte("SUB " + subject + " " + id + "\r\n")
}

// FormatUnsub creates an UNSUB message payload.
func FormatUnsub(id string) []byte {
	return []byte("UNSUB " + id + "\r\n")
}

// FormatMsg creates a MSG message payload.
func FormatMsg(subject, id string, payload []byte) []byte {
	// "MSG <subject> <id> <len>\r\n<payload>"
	header := []byte("MSG " + subject + " " + id + " ")
	lenStr := formatInt(len(payload))

	result := make([]byte, 0, len(header)+len(lenStr)+2+len(payload))
	result = append(result, header...)
	result = append(result, lenStr...)
	result = append(result, '\r', '\n')
	result = append(result, payload...)
	return result
}

// FormatPing creates a PING message payload.
func FormatPing() []byte {
	return []byte("PING\r\n")
}

// FormatPong creates a PONG response payload.
func FormatPong() []byte {
	return []byte("PONG\r\n")
}

// FormatVer creates a VER request payload.
func FormatVer() []byte {
	return []byte("VER\r\n")
}

// FormatVersion creates a VERSION response payload.
func FormatVersion(version string) []byte {
	return []byte("VERSION " + version + "\r\n")
}

// parseVersion parses: " <version>\r\n"
func parseVersion(data []byte) (*Message, error) {
	// Skip leading space
	if len(data) == 0 || data[0] != ' ' {
		return nil, ErrMalformedMsg
	}
	data = data[1:]

	// Find version string (until \r\n or end)
	verEnd := 0
	for verEnd < len(data) && data[verEnd] != '\r' && data[verEnd] != '\n' {
		verEnd++
	}
	if verEnd == 0 {
		return nil, ErrMalformedMsg
	}

	return &Message{
		Command: CmdVERSION,
		Payload: data[:verEnd],
	}, nil
}

// FormatOK creates an +OK response payload.
func FormatOK() []byte {
	return []byte("+OK\r\n")
}

// FormatErr creates a -ERR response payload.
func FormatErr(code, message string) []byte {
	return []byte("-ERR " + code + " " + message + "\r\n")
}

// formatInt converts an integer to string bytes without allocations.
func formatInt(n int) []byte {
	if n == 0 {
		return []byte{'0'}
	}

	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return buf[i:]
}
