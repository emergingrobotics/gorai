# Gorai Serial Protocol v2 (GSP/2)

## Overview

GSP/2 is a transport-agnostic binary protocol designed for reliable communication between host systems and embedded devices. It supports multiple transport layers (UART, UDP, radio links) and provides versioned message handling, reliable delivery with ACK/ERR responses, and efficient binary encoding.

## Design Goals

1. **Transport Agnostic** - Works over UART, UDP, direct radio, or any byte-stream/datagram transport
2. **Version Negotiation** - Mix message versions at runtime, fallback to prior versions dynamically
3. **Reliable Delivery** - Message numbering with ACK/ERR correlation
4. **Efficiency** - Binary encoding, minimal overhead
5. **Forward Compatible** - Unknown message types/versions are safely rejected with ERR

---

## Frame Format

All messages are wrapped in a frame for transport integrity:

```
┌───────┬─────────┬───────────┬───────┬───────┐
│  STX  │ HEADER  │  PAYLOAD  │  CRC  │  ETX  │
│  1B   │   8B    │   0-1024B │  2B   │  1B   │
└───────┴─────────┴───────────┴───────┴───────┘
```

### Frame Fields

| Field | Size | Value | Description |
|-------|------|-------|-------------|
| STX | 1 | `0x02` | Start of frame marker |
| HEADER | 8 | See below | Message header |
| PAYLOAD | 0-1024 | Variable | Message-specific data |
| CRC | 2 | CRC-16 | Big-endian, over HEADER + PAYLOAD |
| ETX | 1 | `0x03` | End of frame marker |

**Frame Limits:**
- Minimum frame size: 12 bytes (empty payload)
- Maximum frame size: 1036 bytes (1024-byte payload)
- Maximum payload: 1024 bytes

### CRC-16 Specification

- **Algorithm:** CRC-16-CCITT
- **Polynomial:** 0x1021
- **Initial value:** 0xFFFF
- **Input/Output reflection:** None
- **Scope:** Calculated over HEADER + PAYLOAD bytes only

---

## Header Format

```
┌───────┬───────┬───────┬───────┬───────┬───────────┐
│  VER  │ FLAGS │ TYPE  │  NUM  │  LEN  │ RESERVED  │
│  1B   │  1B   │  1B   │  2B   │  2B   │    1B     │
└───────┴───────┴───────┴───────┴───────┴───────────┘
        Byte:  0     1     2    3-4   5-6      7
```

### Header Fields

| Field | Offset | Size | Description |
|-------|--------|------|-------------|
| VER | 0 | 1 | Protocol version (major.minor packed: `(major << 4) \| minor`) |
| FLAGS | 1 | 1 | Message flags (see below) |
| TYPE | 2 | 1 | Message type ID |
| NUM | 3 | 2 | Message sequence number (big-endian) |
| LEN | 5 | 2 | Payload length in bytes (big-endian) |
| RESERVED | 7 | 1 | Reserved for future use (must be 0x00) |

### VER Field Encoding

```
VER = (major << 4) | minor
```

- **Major version (4 bits):** 0-15, incompatible protocol changes
- **Minor version (4 bits):** 0-15, backward-compatible additions

Current version: `0x20` (v2.0)

### FLAGS Field

```
Bit 7 6 5 4 3 2 1 0
    │ │ │ │ │ │ │ └─ REQ_ACK: Request acknowledgment
    │ │ │ │ │ │ └─── PRIORITY: High priority message
    │ │ │ │ │ └───── FRAGMENT: Message is fragmented (future)
    │ │ │ │ └─────── LAST_FRAG: Last fragment (future)
    │ │ │ └───────── Reserved
    │ │ └─────────── Reserved
    │ └───────────── Reserved
    └─────────────── Reserved
```

| Flag | Bit | Description |
|------|-----|-------------|
| REQ_ACK | 0 | Sender requests ACK/ERR response |
| PRIORITY | 1 | High-priority message (skip normal queue) |
| FRAGMENT | 2 | This is a fragmented message |
| LAST_FRAG | 3 | This is the last fragment |

### NUM Field (Message Number)

- 16-bit unsigned integer, big-endian
- Wraps from 0xFFFF to 0x0000
- Used to correlate ACK/ERR responses to original requests
- Sender maintains incrementing counter per session
- Value 0x0000 reserved for unsolicited messages (no ACK expected)

---

## Message Types

### Type Ranges

| Range | Direction | Category |
|-------|-----------|----------|
| 0x00 | - | Reserved |
| 0x01-0x0F | Bidirectional | System/Control |
| 0x10-0x3F | Host → Device | Commands |
| 0x40-0x7F | Host → Device | Reserved (future commands) |
| 0x80-0x9F | Device → Host | Responses |
| 0xA0-0xBF | Device → Host | Events/Notifications |
| 0xC0-0xFE | - | Reserved |
| 0xFF | - | Reserved (invalid) |

### System Messages (0x01-0x0F)

| Type | Name | Direction | Description |
|------|------|-----------|-------------|
| 0x01 | PING | Bidirectional | Keep-alive request |
| 0x02 | PONG | Bidirectional | Keep-alive response |
| 0x03 | VER_REQ | Host → Device | Request version info |
| 0x04 | VER_RESP | Device → Host | Version info response |
| 0x05 | CAPS_REQ | Host → Device | Request capabilities |
| 0x06 | CAPS_RESP | Device → Host | Capabilities response |
| 0x07 | RESET | Host → Device | Reset device/subsystem |
| 0x08 | SYNC | Bidirectional | Synchronization marker |

### Command Messages (0x10-0x3F)

| Type | Name | Description |
|------|------|-------------|
| 0x10 | PWM_SET | Set PWM channel(s) |
| 0x11 | PWM_ENABLE | Enable/disable PWM channel(s) |
| 0x12 | PWM_QUERY | Query PWM state |
| 0x13 | PWM_CONFIG | Configure PWM parameters |
| 0x18 | GPIO_SET | Set GPIO state |
| 0x19 | GPIO_QUERY | Query GPIO state |
| 0x1A | GPIO_CONFIG | Configure GPIO |
| 0x20 | CONFIG_GET | Get configuration value |
| 0x21 | CONFIG_SET | Set configuration value |
| 0x22 | CONFIG_SAVE | Save configuration to flash |
| 0x23 | CONFIG_LOAD | Load configuration from flash |

### Response Messages (0x80-0x9F)

| Type | Name | Description |
|------|------|-------------|
| 0x80 | ACK | Positive acknowledgment |
| 0x81 | ERR | Error response |
| 0x82 | PWM_STATE | PWM state response |
| 0x83 | GPIO_STATE | GPIO state response |
| 0x84 | CONFIG_VALUE | Configuration value response |
| 0x85 | STATUS | General status response |

### Event Messages (0xA0-0xBF)

| Type | Name | Description |
|------|------|-------------|
| 0xA0 | HEARTBEAT | Periodic heartbeat |
| 0xA1 | FAILSAFE | Failsafe activated/deactivated |
| 0xA2 | FAULT | Hardware fault detected |
| 0xA3 | INPUT_EVENT | Input state change |

---

## ACK/ERR Response Format

### ACK (0x80)

Sent when a command completes successfully.

```
┌───────────┬───────────┐
│  REF_NUM  │  STATUS   │
│    2B     │    1B     │
└───────────┴───────────┘
```

| Field | Size | Description |
|-------|------|-------------|
| REF_NUM | 2 | Message NUM being acknowledged (big-endian) |
| STATUS | 1 | Status code (0x00 = success) |

### ERR (0x81)

Sent when a command fails.

```
┌───────────┬───────────┬───────────┬─────────────────┐
│  REF_NUM  │ ERR_CODE  │  DETAIL_LEN │    DETAIL     │
│    2B     │    2B     │     1B      │   0-128B      │
└───────────┴───────────┴─────────────┴─────────────────┘
```

| Field | Size | Description |
|-------|------|-------------|
| REF_NUM | 2 | Message NUM that caused error (big-endian) |
| ERR_CODE | 2 | Error code (big-endian, see Error Codes) |
| DETAIL_LEN | 1 | Length of detail string (0-128) |
| DETAIL | 0-128 | UTF-8 error detail (optional) |

---

## Error Codes (v1)

Error codes are versioned. The high byte indicates the error category, the low byte is the specific error.

### Error Code Format

```
ERR_CODE = (CATEGORY << 8) | SPECIFIC
```

### Categories

| Category | Range | Description |
|----------|-------|-------------|
| 0x00 | 0x0000-0x00FF | Success/No error |
| 0x01 | 0x0100-0x01FF | Protocol errors |
| 0x02 | 0x0200-0x02FF | Transport errors |
| 0x03 | 0x0300-0x03FF | Resource errors |
| 0x04 | 0x0400-0x04FF | PWM errors |
| 0x05 | 0x0500-0x05FF | GPIO errors |
| 0x06 | 0x0600-0x06FF | Config errors |
| 0x10 | 0x1000-0x10FF | Application-specific |

### Error Codes v1

```
// Success
0x0000  OK                      No error

// Protocol errors (0x01xx)
0x0100  UNKNOWN_VERSION         Unsupported protocol version
0x0101  UNKNOWN_TYPE            Unknown message type
0x0102  INVALID_LENGTH          Payload length mismatch
0x0103  INVALID_CRC             CRC check failed
0x0104  INVALID_FRAME           Malformed frame
0x0105  INVALID_HEADER          Invalid header field
0x0106  UNSUPPORTED_FLAG        Unsupported flag combination
0x0107  MESSAGE_TOO_LARGE       Message exceeds maximum size

// Transport errors (0x02xx)
0x0200  TX_FAILED               Transmission failed
0x0201  RX_TIMEOUT              Receive timeout
0x0202  BUFFER_OVERFLOW         Buffer overflow
0x0203  SYNC_LOST               Synchronization lost

// Resource errors (0x03xx)
0x0300  BUSY                    Resource busy
0x0301  NOT_READY               Resource not ready
0x0302  NO_MEMORY               Out of memory
0x0303  QUEUE_FULL              Message queue full

// PWM errors (0x04xx)
0x0400  PWM_INVALID_CHANNEL     Invalid PWM channel number
0x0401  PWM_CHANNEL_DISABLED    Channel is disabled
0x0402  PWM_VALUE_OUT_OF_RANGE  Value outside configured limits
0x0403  PWM_HARDWARE_FAULT      PWM hardware fault
0x0404  PWM_NOT_CONFIGURED      Channel not configured

// GPIO errors (0x05xx)
0x0500  GPIO_INVALID_PIN        Invalid GPIO pin
0x0501  GPIO_PIN_IN_USE         Pin already in use
0x0502  GPIO_WRONG_MODE         Pin in wrong mode

// Config errors (0x06xx)
0x0600  CONFIG_INVALID_KEY      Unknown configuration key
0x0601  CONFIG_INVALID_VALUE    Invalid configuration value
0x0602  CONFIG_READ_ONLY        Configuration is read-only
0x0603  CONFIG_FLASH_ERROR      Flash write/read error
```

---

## Message Payloads

### PWM_SET (0x10)

Set one or more PWM channels.

**Single Channel Format:**
```
┌─────────┬───────────┐
│ CHANNEL │ PULSE_US  │
│   1B    │    2B     │
└─────────┴───────────┘
```

**Batch Format:**
```
┌───────┬─────────┬───────────┬─────────┬───────────┬─────┐
│ COUNT │ CHAN[0] │ PULSE[0]  │ CHAN[1] │ PULSE[1]  │ ... │
│  1B   │   1B    │    2B     │   1B    │    2B     │     │
└───────┴─────────┴───────────┴─────────┴───────────┴─────┘
```

| Field | Size | Description |
|-------|------|-------------|
| COUNT | 1 | Number of channels (0 = single channel mode) |
| CHANNEL | 1 | Channel number (0-255) |
| PULSE_US | 2 | Pulse width in microseconds (big-endian) |

**Single channel:** COUNT=0, followed by one CHANNEL+PULSE_US pair (3 bytes)
**Batch:** COUNT=1-85, followed by COUNT × (CHANNEL+PULSE_US) pairs

Maximum batch size: 85 channels (1 + 85×3 = 256 bytes)

### PWM_ENABLE (0x11)

Enable or disable PWM channels.

**Single Channel:**
```
┌─────────┬─────────┐
│ CHANNEL │ ENABLED │
│   1B    │   1B    │
└─────────┴─────────┘
```

**Batch:**
```
┌───────┬─────────┬─────────┬─────────┬─────────┬─────┐
│ COUNT │ CHAN[0] │ EN[0]   │ CHAN[1] │ EN[1]   │ ... │
│  1B   │   1B    │   1B    │   1B    │   1B    │     │
└───────┴─────────┴─────────┴─────────┴─────────┴─────┘
```

| Field | Size | Description |
|-------|------|-------------|
| COUNT | 1 | Number of channels (0 = single channel mode) |
| CHANNEL | 1 | Channel number |
| ENABLED | 1 | 0x00 = disabled, 0x01 = enabled |

### PWM_STATE (0x82)

PWM state response (also used for PWM_QUERY response).

**Single Channel:**
```
┌─────────┬───────────┬─────────┐
│ CHANNEL │ PULSE_US  │ FLAGS   │
│   1B    │    2B     │   1B    │
└─────────┴───────────┴─────────┘
```

**Batch:**
```
┌───────┬─────────┬───────────┬─────────┬─────┐
│ COUNT │ STATE[0]│ STATE[1]  │ STATE[2]│ ... │
│  1B   │   4B    │    4B     │   4B    │     │
└───────┴─────────┴───────────┴─────────┴─────┘
```

| Field | Size | Description |
|-------|------|-------------|
| CHANNEL | 1 | Channel number |
| PULSE_US | 2 | Current pulse width (big-endian) |
| FLAGS | 1 | Bit 0: enabled, Bit 1: failsafe active |

### HEARTBEAT (0xA0)

Periodic heartbeat event (unsolicited).

```
┌───────────┬───────────┬─────────┬───────────┐
│ TIMESTAMP │ UPTIME_MS │  FLAGS  │  SEQ      │
│    4B     │    4B     │   1B    │   2B      │
└───────────┴───────────┴─────────┴───────────┘
```

| Field | Size | Description |
|-------|------|-------------|
| TIMESTAMP | 4 | Unix timestamp or device time (big-endian) |
| UPTIME_MS | 4 | Milliseconds since boot (big-endian) |
| FLAGS | 1 | Bit 0: failsafe, Bit 1: fault |
| SEQ | 2 | Heartbeat sequence number (big-endian) |

### VER_RESP (0x04)

Version information response.

```
┌─────────────┬─────────────┬───────────┬───────────────┐
│ PROTO_VER   │ FW_VERSION  │ HW_REV    │ DEVICE_ID     │
│    1B       │    3B       │   1B      │    8B         │
└─────────────┴─────────────┴───────────┴───────────────┘
```

| Field | Size | Description |
|-------|------|-------------|
| PROTO_VER | 1 | Supported protocol versions (bitmask) |
| FW_VERSION | 3 | Firmware version (major, minor, patch) |
| HW_REV | 1 | Hardware revision |
| DEVICE_ID | 8 | Unique device identifier |

### CAPS_RESP (0x06)

Capabilities response.

```
┌─────────────┬─────────────┬───────────┬───────────────┐
│ PWM_CHANNELS│ GPIO_PINS   │ FEATURES  │ MAX_BAUD      │
│    1B       │    1B       │   2B      │    4B         │
└─────────────┴─────────────┴───────────┴───────────────┘
```

| Field | Size | Description |
|-------|------|-------------|
| PWM_CHANNELS | 1 | Number of PWM channels |
| GPIO_PINS | 1 | Number of GPIO pins |
| FEATURES | 2 | Feature flags (bitmask) |
| MAX_BAUD | 4 | Maximum supported baud rate |

---

## Transport Considerations

### UART Transport

**Recommended Settings:**
- Baud rate: 230400 (default), up to 921600 supported
- Data bits: 8
- Stop bits: 1
- Parity: None
- Flow control: None (protocol handles flow via ACK)

**RP2040 UART Capabilities:**
- Hardware UART: Up to 7.8 Mbaud (limited by peripheral clock)
- Practical limit with standard crystals: 921600 baud reliable
- USB CDC: Up to 12 Mbps (USB Full Speed)

**Raspberry Pi / Orange Pi UART:**
- Hardware UART: 4+ Mbaud supported
- Practical limit: 921600-1500000 baud reliable
- Depends on cable length and EMI environment

**Baud Rate Selection Guide:**

| Baud Rate | Throughput | Latency (64B) | Cable Length | Reliability |
|-----------|------------|---------------|--------------|-------------|
| 115200 | 11.5 KB/s | 5.6 ms | 15m+ | Excellent |
| 230400 | 23.0 KB/s | 2.8 ms | 10m | Excellent |
| 460800 | 46.0 KB/s | 1.4 ms | 5m | Very Good |
| 921600 | 92.0 KB/s | 0.7 ms | 2m | Good |
| 1500000 | 150 KB/s | 0.4 ms | 1m | Fair |

**Recommendation:** Use 230400 as default (2× current speed), allow runtime negotiation up to 921600.

### UART Synchronization

On connection establishment:
1. Host sends 16 bytes of 0x00 (clears any partial frames)
2. Host sends SYNC message (TYPE=0x08)
3. Device responds with SYNC + VER_RESP
4. Host optionally negotiates baud rate change

### UDP Transport

When using UDP (e.g., WiFi, Ethernet):

- Frame format is identical (STX/ETX/CRC retained for consistency)
- One frame per UDP datagram
- Maximum datagram: 1036 bytes (fits in single Ethernet frame)
- NUM field used for duplicate detection and ordering
- Consider adding session ID for multi-client scenarios

### Radio Transport

For direct radio links (LoRa, nRF24, etc.):

- Frame format unchanged
- May need fragmentation for small MTU radios
- FRAGMENT/LAST_FRAG flags enable reassembly
- Consider FEC at radio layer, not protocol layer
- Increase ACK timeout for high-latency links

---

## Protocol Negotiation

### Version Handshake

```
Host                              Device
  │                                  │
  │──── VER_REQ ────────────────────>│
  │                                  │
  │<─── VER_RESP (v2.0, caps) ──────│
  │                                  │
  │     (use highest common version) │
```

### Baud Rate Negotiation (UART)

```
Host                              Device
  │                                  │
  │──── CAPS_REQ ──────────────────>│
  │                                  │
  │<─── CAPS_RESP (max_baud=921600)─│
  │                                  │
  │──── CONFIG_SET (baud=460800) ──>│
  │                                  │
  │<─── ACK ────────────────────────│
  │                                  │
  │     (both switch to new baud)    │
  │                                  │
  │──── PING ──────────────────────>│
  │<─── PONG ──────────────────────│
```

### Version Fallback

If device returns `ERR(UNKNOWN_VERSION)`:
1. Host decrements VER field and retries
2. Continue until compatible version found or v1.0 fails
3. Host maintains per-device version cache

Messages can specify different versions:
- System messages: Always use negotiated version
- PWM messages: May use older version if device firmware outdated
- Device reports supported versions per message type in CAPS_RESP

---

## Timing Parameters

| Parameter | Default | Range | Description |
|-----------|---------|-------|-------------|
| ACK_TIMEOUT | 100 ms | 10-5000 ms | Time to wait for ACK/ERR |
| RETRY_COUNT | 3 | 0-10 | Retries before failure |
| HEARTBEAT_INTERVAL | 1000 ms | 100-10000 ms | Heartbeat period |
| FAILSAFE_TIMEOUT | 500 ms | 100-5000 ms | PWM failsafe trigger |
| KEEPALIVE_INTERVAL | 200 ms | 50-1000 ms | PWM keepalive for safety |

---

## Example Message Sequences

### Set 16 PWM Channels

```
Host → Device:
  Frame: [STX][VER:0x20][FLAGS:0x01][TYPE:0x10][NUM:0x0042][LEN:0x0031][RSV:0x00]
         [COUNT:16][CH0:0][PULSE0:1500][CH1:1][PULSE1:1200]...[CRC][ETX]

Device → Host:
  Frame: [STX][VER:0x20][FLAGS:0x00][TYPE:0x80][NUM:0x0001][LEN:0x0003][RSV:0x00]
         [REF_NUM:0x0042][STATUS:0x00][CRC][ETX]
```

**Wire bytes (PWM_SET 16 channels):**
```
02                          # STX
20 01 10 00 42 00 31 00     # Header (VER=2.0, FLAGS=REQ_ACK, TYPE=PWM_SET, NUM=66, LEN=49, RSV=0)
10                          # COUNT=16
00 05 DC                    # CH0=0, PULSE=1500
01 04 B0                    # CH1=1, PULSE=1200
02 05 DC                    # CH2=2, PULSE=1500
...                         # (13 more channels)
XX XX                       # CRC-16
03                          # ETX
```

Total: 12 + 49 = 61 bytes for 16 channels (vs ~526 bytes in v1)

### Error Response

```
Device → Host (invalid channel):
  Frame: [STX][VER:0x20][FLAGS:0x00][TYPE:0x81][NUM:0x0002][LEN:0x0015][RSV:0x00]
         [REF_NUM:0x0042][ERR_CODE:0x0400][DETAIL_LEN:16]["invalid channel 99"][CRC][ETX]
```

---

## Implementation Notes

### RP2040 (TinyGo)

```go
// Header structure (8 bytes)
type Header struct {
    Ver      uint8   // Protocol version
    Flags    uint8   // Message flags
    Type     uint8   // Message type
    Num      uint16  // Message number (big-endian)
    Len      uint16  // Payload length (big-endian)
    Reserved uint8   // Must be 0
}

// Efficient parsing without allocation
func ParseHeader(buf []byte) Header {
    return Header{
        Ver:      buf[0],
        Flags:    buf[1],
        Type:     buf[2],
        Num:      uint16(buf[3])<<8 | uint16(buf[4]),
        Len:      uint16(buf[5])<<8 | uint16(buf[6]),
        Reserved: buf[7],
    }
}
```

### Memory Budget (RP2040)

- RX buffer: 1048 bytes (max frame)
- TX buffer: 1048 bytes (max frame)
- Parser state: ~32 bytes
- Message queue: 4 × 16 bytes = 64 bytes
- Total: ~2.2 KB (vs 264KB SRAM available)

### Go (Host)

```go
// Message interface for type-safe handling
type Message interface {
    Type() uint8
    Version() uint8
    Encode() []byte
}

// PWMSetMessage implements Message
type PWMSetMessage struct {
    Channels []PWMChannel
}

type PWMChannel struct {
    Channel uint8
    PulseUS uint16
}
```

---

## Migration from GSP v1

### Compatibility Mode

Devices can support both protocols:
1. On startup, device listens for both v1 (text) and v2 (binary) frames
2. First valid frame determines protocol version for session
3. Device can advertise v1 support in CAPS_RESP

### Message Mapping

| GSP v1 | GSP v2 |
|--------|--------|
| `PUB pwm.command` | `PWM_SET (0x10)` |
| `PUB pwm.enable` | `PWM_ENABLE (0x11)` |
| `PUB pwm.state.request` | `PWM_QUERY (0x12)` |
| `PING` | `PING (0x01)` |
| `PONG` | `PONG (0x02)` |
| `VER` | `VER_REQ (0x03)` |
| `VERSION` | `VER_RESP (0x04)` |

---

## Appendix A: Complete Type Registry

```
0x00  RESERVED
0x01  PING
0x02  PONG
0x03  VER_REQ
0x04  VER_RESP
0x05  CAPS_REQ
0x06  CAPS_RESP
0x07  RESET
0x08  SYNC
0x09-0x0F  Reserved (system)

0x10  PWM_SET
0x11  PWM_ENABLE
0x12  PWM_QUERY
0x13  PWM_CONFIG
0x14-0x17  Reserved (PWM)
0x18  GPIO_SET
0x19  GPIO_QUERY
0x1A  GPIO_CONFIG
0x1B-0x1F  Reserved (GPIO)
0x20  CONFIG_GET
0x21  CONFIG_SET
0x22  CONFIG_SAVE
0x23  CONFIG_LOAD
0x24-0x3F  Reserved (commands)

0x40-0x7F  Reserved (future commands)

0x80  ACK
0x81  ERR
0x82  PWM_STATE
0x83  GPIO_STATE
0x84  CONFIG_VALUE
0x85  STATUS
0x86-0x9F  Reserved (responses)

0xA0  HEARTBEAT
0xA1  FAILSAFE
0xA2  FAULT
0xA3  INPUT_EVENT
0xA4-0xBF  Reserved (events)

0xC0-0xFE  Reserved
0xFF  INVALID
```

---

## Appendix B: Quick Reference

### Frame Structure
```
[STX:1][VER:1][FLAGS:1][TYPE:1][NUM:2][LEN:2][RSV:1][PAYLOAD:0-1024][CRC:2][ETX:1]
```

### Common Operations

| Operation | Type | Payload |
|-----------|------|---------|
| Set 1 PWM | 0x10 | `[0x00][ch][pulse_hi][pulse_lo]` |
| Set N PWM | 0x10 | `[N][ch0][p0_hi][p0_lo]...[chN][pN_hi][pN_lo]` |
| Enable ch | 0x11 | `[0x00][ch][0x01]` |
| Ping | 0x01 | (empty) |
| Get version | 0x03 | (empty) |

### Size Comparison (16 channels)

| Protocol | Message Size | Overhead |
|----------|--------------|----------|
| GSP v1 (JSON) | ~526 bytes | 86% |
| GSP v2 (binary) | 61 bytes | 21% |

---

## Revision History

| Version | Date | Changes |
|---------|------|---------|
| 2.0 | 2025-01 | Initial GSP/2 specification |
