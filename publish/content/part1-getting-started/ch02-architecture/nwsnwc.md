## 2.5 Network Transparency (NWS/NWC)

One of GoRAI's most powerful features is network transparency: the ability to use resources the same way whether they're local (in the same process) or remote (on another machine).

### Local vs Remote Resources

Consider a motor. When it's local, you call methods directly:

```go
motor := createMotor()
motor.SetPower(ctx, 0.5)  // Direct method call
position, _ := motor.GetPosition(ctx)
```

When the motor runs on a different node (perhaps a microcontroller gateway), you still want the same interface. This is where NWS (Network Wrapper Server) and NWC (Network Wrapper Client) come in.

### NWS: Exposing Resources Over NATS

A Network Wrapper Server takes a local resource and exposes its methods over NATS:

```go
// On the node with the physical motor
motor := createMotor()

// Wrap it for network access
wrapper := nws.Wrap(node, motor, "gorai.motors.left_wheel")
```

Now method calls arrive as NATS messages. The wrapper:
1. Subscribes to request topics
2. Deserializes incoming requests
3. Calls the actual resource method
4. Serializes and returns the response

The topics follow a pattern:
- `gorai.motors.left_wheel.SetPower` - for SetPower calls
- `gorai.motors.left_wheel.GetPosition` - for GetPosition calls
- And so on for each method

### NWC: Consuming Remote Resources

A Network Wrapper Client creates a local proxy that forwards calls over NATS:

```go
// On a different node (or same node, doesn't matter)
motor := nwc.Motor(node, "gorai.motors.left_wheel")

// Use it like a local motor
motor.SetPower(ctx, 0.5)  // Becomes NATS request/reply
position, _ := motor.GetPosition(ctx)
```

The client proxy:
1. Serializes the method call and arguments
2. Sends a NATS request
3. Waits for the response
4. Deserializes and returns the result

### Transparent Location Abstraction

The magic is that consuming code doesn't know (or care) if a resource is local or remote:

```go
func RunBehavior(motor motor.Motor) {
    // This function works with local or remote motors
    for i := 0; i < 10; i++ {
        motor.SetPower(ctx, float64(i) / 10)
        time.Sleep(100 * time.Millisecond)
    }
    motor.Stop(ctx)
}

// Works with local motor
localMotor := createMotor()
RunBehavior(localMotor)

// Works with remote motor
remoteMotor := nwc.Motor(node, "gorai.motors.left_wheel")
RunBehavior(remoteMotor)
```

### Use Cases

**Distributed Robot Architecture**: Camera processing on a GPU node, motion control on the main board:

```go
// Vision node exposes camera and detector
camera := setupCamera()
nws.Wrap(visionNode, camera, "gorai.cameras.front")

detector := setupDetector()
nws.Wrap(visionNode, detector, "gorai.vision.detector")

// Brain node consumes them remotely
camera := nwc.Camera(brainNode, "gorai.cameras.front")
detector := nwc.Vision(brainNode, "gorai.vision.detector")
```

**Remote Monitoring/Control**: Operator station accessing robot resources:

```go
// On operator laptop
robot := nwc.Connect("nats://robot.local:4222")
motor := nwc.Motor(robot, "gorai.motors.left_wheel")

// Interactive control
motor.SetPower(ctx, *joystickInput)
```

**Testing with Resource Injection**: Use fake resources transparently:

```go
// In tests, expose a fake motor
fake := fake.NewMotor()
nws.Wrap(testNode, fake, "test.motor")

// Code under test connects to it
motor := nwc.Motor(sutNode, "test.motor")
RunBehavior(motor)

// Verify fake received expected calls
assert.True(t, fake.StopWasCalled())
```

### Performance Considerations

Network transparency has overhead:
- Serialization/deserialization for each call
- Network latency (microseconds locally, milliseconds across network)
- NATS message processing

For high-frequency operations (1kHz control loops), prefer local resources or the serial gateway pattern. Reserve NWS/NWC for:
- Infrequent operations (configuration, status checks)
- Operations where network latency is acceptable
- Cross-node coordination

### Implementation Details

Under the hood, NWS/NWC use NATS request/reply:

```
Client                      NATS                        Server
  │                           │                           │
  │ Request: SetPower(0.5)    │                           │
  │──────────────────────────>│                           │
  │                           │──────────────────────────>│
  │                           │                           │ Execute
  │                           │                           │
  │                           │<──────────────────────────│
  │<──────────────────────────│ Reply: OK                 │
  │                           │                           │
```

Errors propagate correctly—if the remote motor fails, the error returns through the client proxy. Timeouts are configurable. Context cancellation works as expected.

---

With the mental model established—nodes, resources, distributed architecture, configuration, and network transparency—you're ready to understand GoRAI's communication backbone. Chapter 3 dives deep into NATS.
