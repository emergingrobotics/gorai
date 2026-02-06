# Gorai Serial Protocol v2 (GSP/2)

**MOVED TO SEPARATE REPOSITORY**

The Gorai Serial Protocol v2 (GSP/2) specification and implementation have been moved to a dedicated repository for better maintainability and reusability across projects.

## gorai-gsp Repository

**Repository:** [github.com/emergingrobotics/gorai-gsp](https://github.com/emergingrobotics/gorai-gsp)

**Local checkout:** `../gorai-gsp` (relative to this repository)

The gorai-gsp repository is the authoritative source for:

- **GSP/2 Protocol Specification** — Complete binary protocol documentation
- **Frame Format** — STX/ETX framing, CRC-16-CCITT error detection
- **Message Types** — System, PWM, motor, encoder, IMU, sensor, GPIO messages
- **Go Client Library** — Host-side implementation with async callbacks
- **TinyGo Device Library** — Microcontroller-compatible implementation
- **Transport Abstraction** — UART, USB CDC, UDP, radio support
- **Examples and Integration Guides**

## Why Separate Repository?

The gorai-gsp library is designed to be:

1. **Reusable** — Can be used in non-Gorai projects
2. **Independently versioned** — Separate release cycle from main Gorai framework
3. **Language-agnostic** — Protocol spec can be implemented in other languages
4. **Focused** — Clear scope and ownership

## Quick Links

- **Repository:** https://github.com/emergingrobotics/gorai-gsp
- **Documentation:** See README.md in the gorai-gsp repository
- **Local path:** `../gorai-gsp` (if checked out alongside this repository)

## Integration with Gorai

The main Gorai framework uses gorai-gsp as a Go module dependency:

```go
import "github.com/emergingrobotics/gorai-gsp/client"
import "github.com/emergingrobotics/gorai-gsp/gsp"
import "github.com/emergingrobotics/gorai-gsp/gsp/messages"
```

For TinyGo devices, use the device library:

```go
import "github.com/emergingrobotics/gorai-gsp/device"
```

---

**Last updated:** 2026-02-06
