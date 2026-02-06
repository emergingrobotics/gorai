# Serial Interface Specification

**MOVED TO SEPARATE REPOSITORY**

The Gorai Serial Protocol (GSP) has been moved to its own repository for better maintainability and reusability across projects.

## gorai-gsp Repository

**Repository:** [github.com/emergingrobotics/gorai-gsp](https://github.com/emergingrobotics/gorai-gsp)

**Local checkout:** `../gorai-gsp` (relative to this repository)

The gorai-gsp repository contains:

- **GSP/2 Protocol Specification** — Compact binary protocol for reliable bidirectional communication
- **Go Client Library** — Host-side implementation with async callbacks
- **TinyGo Device Library** — Microcontroller-compatible implementation
- **Transport Abstraction** — Works over UART, USB CDC, UDP, or radio links
- **Complete Documentation** — Protocol details, examples, and integration guides

## Key Features

- **Compact binary format** — 9-byte frame overhead (89% size reduction vs JSON)
- **CRC-16-CCITT** error detection
- **Bidirectional** — both host and device can initiate messages
- **40+ message types** — PWM, motors, encoders, IMU, sensors, GPIO, configuration
- **TinyGo compatible** — tested on RP2040, ESP32, and other microcontrollers

## Quick Links

- **Repository:** https://github.com/emergingrobotics/gorai-gsp
- **Documentation:** See README.md in the gorai-gsp repository
- **Local path:** `../gorai-gsp` (if checked out alongside this repository)

## Migration Note

This file previously contained the GSP v1 specification (text-based protocol). That has been superseded by GSP/2 (binary protocol) in the gorai-gsp repository. The v1 specification is preserved in the gorai-gsp repository for historical reference.

---

**Last updated:** 2026-02-06
