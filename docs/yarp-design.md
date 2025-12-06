# YARP Design Summary

**YARP: Yet Another Robot Platform**

This document summarizes the design and architecture of YARP (Yet Another Robot Platform), a robotics middleware developed by the Italian Institute of Technology (IIT) and the RobotCub Consortium. YARP aims to increase the longevity of robot software projects through loose coupling and transport-neutral communication.

## Core Philosophy

YARP's design philosophy centers on several key principles:

1. **Loose Coupling**: Components communicate through well-defined interfaces, minimizing dependencies
2. **Transport Neutrality**: Communication abstractions allow swapping network protocols without code changes
3. **Longevity**: The architecture is designed to outlive specific hardware and software dependencies
4. **Non-intrusive**: YARP is "reluctant middleware" - it doesn't try to be an operating system or control your system

## Architecture Overview

YARP consists of three main libraries:

- **YARP_os**: Core OS abstraction and communication infrastructure
- **YARP_sig**: Signal processing (images, audio, etc.)
- **YARP_dev**: Device driver interfaces and implementations

## Communication Model

### Ports

The fundamental communication primitive in YARP is the **Port**. Ports follow the Observer design pattern and support:

- **Many-to-many connections**: A port can connect to multiple other ports
- **Named addressing**: Ports have human-readable names (e.g., `/camera`, `/motor/position`)
- **Multiple protocols**: Each connection can use a different transport protocol

```
/camera ----[mcast]----> /viewer1
        ----[mcast]----> /tracker/image
/tracker/position --[tcp]--> /motor/position
```

### Port Types

1. **Port**: Basic streaming port with blocking read/write
2. **BufferedPort\<T\>**: Buffered port with decoupled sender/receiver timing
3. **RpcClient / RpcServer**: Specialized ports for request-reply patterns

### Buffering Policies

BufferedPort supports two buffering modes:

- **ODP (Oldest Packet Drop)**: Default - drops old messages to reduce latency
- **FIFO (Strict)**: Preserves all messages via `setStrict()` / `writeStrict()`

## Transport Carriers

YARP abstracts network transport through "carriers". Each carrier implements a specific protocol:

| Carrier | Use Case |
|---------|----------|
| `tcp` | Reliable point-to-point communication |
| `udp` | Fast, unreliable communication |
| `mcast` | Efficient multicast to many receivers |
| `shmem` | Fast local inter-process communication |
| `local` | Thread-to-thread within a process |
| `text` | Human-readable for debugging/telnet |

### Protocol Handshake

Connections follow a multi-phase protocol:

1. **Initiation**: TCP socket connection to receiver
2. **Header**: 8-byte protocol specifier + sender name
3. **Header Reply**: Acknowledgement, possible protocol switch
4. **Index**: Metadata about payload
5. **Payload**: User data
6. **Acknowledgement**: Receipt confirmation (carrier-dependent)

Protocol specifiers use magic bytes, e.g.:
- TCP: `Y A 0xE4 0x1E 0x00 0x00 R P`
- UDP: `Y A 0x61 0x1E 0x00 0x00 R P`
- Text: `C O N N E C T  ` (space at end)

## Name Server

The **YARP Name Server** maintains a registry of all ports:

- Port names → network addresses (IP + socket port)
- Port properties (accepted carriers, offered carriers)
- Network routing information

### Name Server Commands

| Command | Description |
|---------|-------------|
| `query /port` | Get port registration info |
| `register /port` | Register a new port |
| `unregister /port` | Remove port registration |
| `list` | List all registered ports |
| `route /src /dst` | Find optimal connection carrier |

### Discovery

Ports discover the name server through:
1. Configuration file (`~/.yarp/conf/yarp.conf`)
2. Multicast discovery (224.2.1.1:10001)

### Namespaces

Multiple name servers can coexist using different namespaces (e.g., `/root` vs `/my/root`), enabling isolated YARP networks.

## Device Abstraction

### Device Drivers

YARP device drivers inherit from `DeviceDriver` and implement family-specific interfaces:

```cpp
class IFrameGrabberImage {
    virtual bool getImage(ImageOf<PixelRgb>& image) = 0;
    virtual int height() const = 0;
    virtual int width() const = 0;
};
```

### PolyDriver

The `PolyDriver` class enables runtime device instantiation:

```cpp
PolyDriver dd("dragonfly");  // Load by name
IFrameGrabberImage *grabber;
dd.view(grabber);            // Get interface
```

### Network Wrapper Architecture (NWS/NWC)

YARP separates local device access from remote access:

- **NWS (Network Wrapper Server)**: Thin layer exposing device interfaces over the network
- **NWC (Network Wrapper Client)**: Client-side proxy implementing the same interfaces

This allows applications to work identically with local or remote devices:

```
[Device] <--attach--> [NWS] <--network--> [NWC] <--interface--> [Application]
```

Naming convention: `deviceName_nws_yarp`, `deviceName_nwc_yarp`

## Data Types and Serialization

### Bottle

The `Bottle` class is YARP's flexible, self-describing data container:
- Nested lists of primitive types (int, double, string, etc.)
- Automatic text/binary conversion
- Interoperable with command-line tools

### Portable Interface

Custom types implement `Portable` for serialization:

```cpp
class Target : public Portable {
    int x, y;
    bool write(ConnectionWriter& connection) override {
        connection.appendInt32(x);
        connection.appendInt32(y);
        return true;
    }
    bool read(ConnectionReader& connection) override {
        x = connection.expectInt32();
        y = connection.expectInt32();
        return !connection.isError();
    }
};
```

### Thrift IDL

YARP integrates Apache Thrift for interface definition:

```thrift
service Demo {
    i32 get_answer();
    bool set_answer(1:i32 rightAnswer);
    bool start();
    bool stop();
}
```

The `yarp_idl_to_dir` CMake macro generates C++ code for RPC communication.

## Port Monitors

Port monitors enable dynamic data processing on connections:

- Written in Lua scripts or compiled DLLs
- Loaded at runtime without recompiling
- Callbacks: `create`, `accept`, `update`, `destroy`

Use cases:
- Data filtering and transformation
- Logging and monitoring
- Quality of Service enforcement

### Port Arbitration

Multiple inputs to a single port can be arbitrated using:
- Event-based selection constraints
- First-order logic rules for data selection
- Time-based events with configurable lifetimes

## ROS Interoperability

YARP supports interoperability with ROS through:
- `tcpros` carrier for ROS-style communication
- Separate NWS/NWC for YARP and ROS middleware
- Both can attach to the same underlying device

## Key Command-Line Tools

| Tool | Purpose |
|------|---------|
| `yarpserver` | Name server |
| `yarpdev` | Device instantiation |
| `yarprobotinterface` | Robot configuration and launch |
| `yarp connect` | Create port connections |
| `yarp read/write` | Debug ports |
| `yarp rpc` | Send RPC commands |

## Design Patterns Used

1. **Observer Pattern**: Port connections for streaming data
2. **Factory Pattern**: Device driver registration and creation
3. **Proxy Pattern**: NWC as remote interface proxy
4. **Strategy Pattern**: Carrier selection for connections

## Lessons for Gorai

Key takeaways from YARP's design that may inform Gorai:

1. **Name-based addressing** with centralized registry enables flexible topology
2. **Transport abstraction** allows protocol evolution without API changes
3. **Device interfaces** separate hardware access from algorithms
4. **Network wrappers** cleanly separate local and remote access
5. **Self-describing data** (Bottle) simplifies debugging and interop
6. **IDL code generation** reduces boilerplate for RPC
7. **Port monitors** enable non-invasive data processing
8. **Multicast discovery** simplifies deployment
