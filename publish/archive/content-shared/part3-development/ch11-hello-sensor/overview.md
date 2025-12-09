# Chapter 9: Hello Sensor Deep Dive

This chapter walks through a complete, working GoRAI component: the `hello-sensor` example. By understanding every line, you'll be ready to build your own components.

## 9.1 What We're Building

The hello-sensor reads CPU temperature from the host system and publishes it to NATS. It demonstrates:

- Creating a GoRAI node
- Platform-specific hardware access
- Implementing the Sensor interface
- Publishing structured messages
- Configuration and command-line flags
- Statistics collection
- Graceful shutdown
- Fake implementations for testing

The complete code is in `examples/hello-sensor/`.

## 9.2 Architecture Overview

```
┌─────────────────────────────────────────┐
│              hello-sensor               │
├─────────────────────────────────────────┤
│  main.go                                │
│    ├── Create node                      │
│    ├── Create reader (platform-specific)│
│    ├── Create sensor component          │
│    └── Run publish loop                 │
├─────────────────────────────────────────┤
│  reader/                                │
│    ├── reader.go (interface)            │
│    ├── linux.go (thermal zones)         │
│    ├── darwin.go (osx-cpu-temp)         │
│    └── unsupported.go (stub)            │
├─────────────────────────────────────────┤
│  sensor/                                │
│    ├── temperature.go (component)       │
│    └── fake/fake.go (test double)       │
└─────────────────────────────────────────┘
```

**Separation of concerns**:
- `reader/`: Platform-specific temperature reading
- `sensor/`: GoRAI component wrapping the reader
- `main.go`: Entry point orchestrating everything

*Cross-reference: See Chapter 4 for the sensor interface this implements.*
