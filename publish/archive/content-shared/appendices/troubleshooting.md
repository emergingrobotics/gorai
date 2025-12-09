# Troubleshooting

Common issues and solutions when working with GoRAI.

## NATS Connectivity

### Can't connect to NATS server

```
failed to connect to NATS: nats: no servers available for connection
```

**Solutions**:

1. Check NATS server is running: `nats server info`
2. Check URL: default is `nats://localhost:4222`
3. Check firewall allows port 4222
4. Verify network connectivity: `ping localhost`

### JetStream not available

```
JetStream not available
```

**Solutions**:

1. Start NATS with `-js` flag: `nats-server -js`
2. Check JetStream is enabled in config file
3. Verify storage directory is writable

### Slow message delivery

**Possible causes**:

1. Network congestion
2. Large message payloads
3. Consumer processing too slow

**Solutions**:

1. Use QoS settings appropriate for your use case
2. Compress large payloads
3. Add buffering or parallel consumers

## Hardware Access Issues

### Permission denied for GPIO

```
open /sys/class/gpio/export: permission denied
```

**Solutions**:

1. Add user to gpio group: `sudo usermod -aG gpio $USER`
2. Logout and login for group changes to take effect
3. Or run with sudo (not recommended for production)

### I2C device not found

```
no I2C device at address 0x68
```

**Solutions**:

1. Check wiring - verify connections
2. Verify I2C enabled: `sudo raspi-config` → Interfaces → I2C
3. Scan bus: `i2cdetect -y 1`
4. Check device address in datasheet (some have configurable addresses)
5. Check for address conflicts with other devices

### Camera not accessible

```
cannot open video device /dev/video0
```

**Solutions**:

1. Check camera is connected: `ls /dev/video*`
2. Add user to video group: `sudo usermod -aG video $USER`
3. Check another application isn't using the camera
4. Verify driver is loaded: `lsmod | grep uvc`

## Build Problems

### Module not found

```
cannot find module providing package github.com/gorai/gorai/...
```

**Solutions**:

1. Run `go mod download`
2. Check Go version ≥ 1.21: `go version`
3. Verify network access to github.com
4. Clear module cache: `go clean -modcache`

### CGo errors

```
cgo: C compiler "gcc" not found
```

**Solutions**:

1. Install build tools: `sudo apt install build-essential`
2. Or use pure Go alternatives (check package documentation)
3. Set `CGO_ENABLED=0` for pure Go builds (if supported)

### TinyGo compilation errors

```
error: could not find wasm-opt
```

**Solutions**:

1. Install binaryen: `sudo apt install binaryen`
2. Update TinyGo to latest version
3. Check target is supported: `tinygo targets`

## Runtime Issues

### High CPU usage

**Possible causes**:

1. Tight polling loops
2. Unthrottled message processing
3. Memory allocation in hot paths

**Solutions**:

1. Add appropriate sleep/delays in loops
2. Use channel buffering
3. Profile with `go tool pprof`

### Memory leaks

**Possible causes**:

1. Goroutines not terminating
2. Subscriptions not cleaned up
3. Accumulated data structures

**Solutions**:

1. Ensure proper context cancellation
2. Call `Close()` on all resources
3. Use `runtime/pprof` to profile memory

### Message ordering issues

**NATS guarantees**:

- Per-publisher ordering preserved
- No global ordering across publishers

**Solutions**:

1. Use sequence numbers in messages
2. Use JetStream for ordering guarantees
3. Design for eventual consistency

## Getting Help

If you can't resolve an issue:

1. Check [GitHub Issues](https://github.com/gorai/gorai/issues)
2. Search [Discussions](https://github.com/gorai/gorai/discussions)
3. Include in bug reports:
   - Go version
   - NATS server version
   - Hardware platform
   - Minimal reproduction code
