package gsp

import (
	"bytes"
	"testing"
)

func TestParseMessagePing(t *testing.T) {
	msg, err := ParseMessage([]byte("PING\r\n"))
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}
	if msg.Command != CmdPING {
		t.Errorf("Command = %q, want %q", msg.Command, CmdPING)
	}
}

func TestParseMessagePong(t *testing.T) {
	msg, err := ParseMessage([]byte("PONG\r\n"))
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}
	if msg.Command != CmdPONG {
		t.Errorf("Command = %q, want %q", msg.Command, CmdPONG)
	}
}

func TestParseMessagePub(t *testing.T) {
	msg, err := ParseMessage([]byte("PUB sensor.temp 4\r\n25.3"))
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}
	if msg.Command != CmdPUB {
		t.Errorf("Command = %q, want %q", msg.Command, CmdPUB)
	}
	if msg.Subject != "sensor.temp" {
		t.Errorf("Subject = %q, want %q", msg.Subject, "sensor.temp")
	}
	if !bytes.Equal(msg.Payload, []byte("25.3")) {
		t.Errorf("Payload = %q, want %q", msg.Payload, "25.3")
	}
}

func TestParseMessageSub(t *testing.T) {
	msg, err := ParseMessage([]byte("SUB motor.command 1\r\n"))
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}
	if msg.Command != CmdSUB {
		t.Errorf("Command = %q, want %q", msg.Command, CmdSUB)
	}
	if msg.Subject != "motor.command" {
		t.Errorf("Subject = %q, want %q", msg.Subject, "motor.command")
	}
	if msg.SubID != "1" {
		t.Errorf("SubID = %q, want %q", msg.SubID, "1")
	}
}

func TestParseMessageUnsub(t *testing.T) {
	msg, err := ParseMessage([]byte("UNSUB 42\r\n"))
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}
	if msg.Command != CmdUNSUB {
		t.Errorf("Command = %q, want %q", msg.Command, CmdUNSUB)
	}
	if msg.SubID != "42" {
		t.Errorf("SubID = %q, want %q", msg.SubID, "42")
	}
}

func TestParseMessageMsg(t *testing.T) {
	msg, err := ParseMessage([]byte("MSG motor.left.command 5 13\r\n{\"power\":0.5}"))
	if err != nil {
		t.Fatalf("ParseMessage failed: %v", err)
	}
	if msg.Command != CmdMSG {
		t.Errorf("Command = %q, want %q", msg.Command, CmdMSG)
	}
	if msg.Subject != "motor.left.command" {
		t.Errorf("Subject = %q, want %q", msg.Subject, "motor.left.command")
	}
	if msg.SubID != "5" {
		t.Errorf("SubID = %q, want %q", msg.SubID, "5")
	}
	if !bytes.Equal(msg.Payload, []byte("{\"power\":0.5}")) {
		t.Errorf("Payload = %q, want %q", msg.Payload, "{\"power\":0.5}")
	}
}

func TestParseMessageInvalidCommand(t *testing.T) {
	_, err := ParseMessage([]byte("INVALID\r\n"))
	if err != ErrInvalidCommand {
		t.Errorf("err = %v, want ErrInvalidCommand", err)
	}
}

func TestParseMessageMalformed(t *testing.T) {
	testCases := []struct {
		name  string
		input []byte
	}{
		{"empty", []byte{}},
		{"pub_no_subject", []byte("PUB \r\n")},
		{"pub_no_length", []byte("PUB test \r\n")},
		{"sub_no_id", []byte("SUB test \r\n")},
		{"msg_no_length", []byte("MSG test 1 \r\n")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := ParseMessage(tc.input)
			if err != ErrMalformedMsg {
				t.Errorf("err = %v, want ErrMalformedMsg", err)
			}
		})
	}
}

func TestFormatPub(t *testing.T) {
	result := FormatPub("sensor.temp", []byte("25.3"))
	expected := []byte("PUB sensor.temp 4\r\n25.3")
	if !bytes.Equal(result, expected) {
		t.Errorf("FormatPub = %q, want %q", result, expected)
	}
}

func TestFormatSub(t *testing.T) {
	result := FormatSub("motor.command", "1")
	expected := []byte("SUB motor.command 1\r\n")
	if !bytes.Equal(result, expected) {
		t.Errorf("FormatSub = %q, want %q", result, expected)
	}
}

func TestFormatUnsub(t *testing.T) {
	result := FormatUnsub("42")
	expected := []byte("UNSUB 42\r\n")
	if !bytes.Equal(result, expected) {
		t.Errorf("FormatUnsub = %q, want %q", result, expected)
	}
}

func TestFormatMsg(t *testing.T) {
	result := FormatMsg("motor.command", "1", []byte("hello"))
	expected := []byte("MSG motor.command 1 5\r\nhello")
	if !bytes.Equal(result, expected) {
		t.Errorf("FormatMsg = %q, want %q", result, expected)
	}
}

func TestFormatPing(t *testing.T) {
	result := FormatPing()
	expected := []byte("PING\r\n")
	if !bytes.Equal(result, expected) {
		t.Errorf("FormatPing = %q, want %q", result, expected)
	}
}

func TestFormatPong(t *testing.T) {
	result := FormatPong()
	expected := []byte("PONG\r\n")
	if !bytes.Equal(result, expected) {
		t.Errorf("FormatPong = %q, want %q", result, expected)
	}
}

func TestFormatOK(t *testing.T) {
	result := FormatOK()
	expected := []byte("+OK\r\n")
	if !bytes.Equal(result, expected) {
		t.Errorf("FormatOK = %q, want %q", result, expected)
	}
}

func TestFormatErr(t *testing.T) {
	result := FormatErr("E001", "test error")
	expected := []byte("-ERR E001 test error\r\n")
	if !bytes.Equal(result, expected) {
		t.Errorf("FormatErr = %q, want %q", result, expected)
	}
}

func TestFormatParseRoundTrip(t *testing.T) {
	// Test that Format -> Parse returns the same data
	t.Run("PUB", func(t *testing.T) {
		payload := []byte("test payload data")
		formatted := FormatPub("test.subject", payload)
		parsed, err := ParseMessage(formatted)
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}
		if parsed.Subject != "test.subject" {
			t.Errorf("Subject = %q, want %q", parsed.Subject, "test.subject")
		}
		if !bytes.Equal(parsed.Payload, payload) {
			t.Errorf("Payload = %q, want %q", parsed.Payload, payload)
		}
	})

	t.Run("SUB", func(t *testing.T) {
		formatted := FormatSub("test.subject", "123")
		parsed, err := ParseMessage(formatted)
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}
		if parsed.Subject != "test.subject" {
			t.Errorf("Subject = %q, want %q", parsed.Subject, "test.subject")
		}
		if parsed.SubID != "123" {
			t.Errorf("SubID = %q, want %q", parsed.SubID, "123")
		}
	})

	t.Run("MSG", func(t *testing.T) {
		payload := []byte("{\"key\":\"value\"}")
		formatted := FormatMsg("test.subject", "5", payload)
		parsed, err := ParseMessage(formatted)
		if err != nil {
			t.Fatalf("ParseMessage failed: %v", err)
		}
		if parsed.Subject != "test.subject" {
			t.Errorf("Subject = %q, want %q", parsed.Subject, "test.subject")
		}
		if parsed.SubID != "5" {
			t.Errorf("SubID = %q, want %q", parsed.SubID, "5")
		}
		if !bytes.Equal(parsed.Payload, payload) {
			t.Errorf("Payload = %q, want %q", parsed.Payload, payload)
		}
	})
}
