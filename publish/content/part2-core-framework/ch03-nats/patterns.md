## 3.3 GoRAI's NATS Patterns

GoRAI builds three communication patterns on NATS: Topics (pub/sub), Services (request/reply), and Actions (long-running with feedback).

### 3.3.1 Topics (Pub/Sub)

Topics are the primary pattern for streaming data. Sensors publish continuously; interested nodes subscribe.

**Publishing sensor data**:
```go
// Create a typed publisher
pub := pub.New[*sensor.Temperature](node, "gorai.sensors.temp.data")

// In your reading loop
for reading := range temperatureReadings() {
    msg := &sensor.Temperature{
        Header:      makeHeader(),
        Temperature: reading.Celsius,
        Variance:    reading.Variance,
    }
    pub.Publish(ctx, msg)
}
```

**Subscribing to sensor data**:
```go
sub.New[*sensor.Temperature](node, "gorai.sensors.temp.data",
    func(msg *sensor.Temperature) {
        log.Printf("Temperature: %.1f°C", msg.Temperature)
    })
```

**Telemetry publishing** follows the same pattern:
```go
// Battery monitor
battPub := pub.New[*sensor.BatteryState](node, "gorai.power.battery.state")

ticker := time.NewTicker(time.Second)
for range ticker.C {
    state := readBatteryState()
    battPub.Publish(ctx, state)
}
```

**Topic naming conventions**:
```
gorai.{node}.{component}.{datatype}

Examples:
gorai.hello.cpu_temp.data
gorai.sensors.imu.data
gorai.motors.left.feedback
gorai.cameras.front.image
```

*Cross-reference: See Chapter 4 for how sensor data flows over topics.*

### 3.3.2 Services (Request/Reply)

Services handle synchronous operations: "give me the current value" or "execute this command and tell me if it worked."

**Implementing a service**:
```go
// Register a handler for motor commands
nc.Subscribe("gorai.motors.left.set_power", func(msg *nats.Msg) {
    var req MotorPowerRequest
    proto.Unmarshal(msg.Data, &req)

    err := motor.SetPower(ctx, req.Power)

    resp := &MotorPowerResponse{Success: err == nil}
    if err != nil {
        resp.Error = err.Error()
    }

    data, _ := proto.Marshal(resp)
    msg.Respond(data)
})
```

**Calling a service**:
```go
req := &MotorPowerRequest{Power: 0.5}
data, _ := proto.Marshal(req)

respMsg, err := nc.Request("gorai.motors.left.set_power", data, time.Second)
if err != nil {
    return fmt.Errorf("request failed: %w", err)
}

var resp MotorPowerResponse
proto.Unmarshal(respMsg.Data, &resp)
if !resp.Success {
    return fmt.Errorf("motor error: %s", resp.Error)
}
```

**Timeout handling** is critical for robotics:
```go
// Short timeout for control commands
resp, err := nc.Request(subject, data, 100*time.Millisecond)
if err == nats.ErrTimeout {
    // Handle timeout—maybe stop motors for safety
    emergencyStop()
}
```

*Cross-reference: See Chapter 7 for higher-level service implementations.*

### 3.3.3 Actions (Long-Running)

Actions handle operations that take time and provide progress updates: navigation to a goal, arm movements, scanning routines.

The pattern involves three message types:
- **Goal**: What to do
- **Feedback**: Progress updates during execution
- **Result**: Final outcome

```
Client                           Server
  │                                │
  │ Goal: navigate to (10, 5)      │
  │───────────────────────────────>│
  │                                │ Start navigating
  │   Feedback: 20% complete       │
  │<───────────────────────────────│
  │   Feedback: 50% complete       │
  │<───────────────────────────────│
  │   Feedback: 80% complete       │
  │<───────────────────────────────│
  │                                │ Arrived
  │   Result: success              │
  │<───────────────────────────────│
  │                                │
```

**Server implementation**:
```go
server, _ := action.NewServer[*NavGoal, *NavFeedback, *NavResult](
    node, "navigation.go_to",
    func(ctx context.Context, handle *action.GoalHandle[*NavGoal, *NavFeedback, *NavResult]) {
        goal := handle.Goal()

        for !atGoal(goal.Position) {
            if handle.IsCanceling() {
                handle.SetCanceled(&NavResult{Success: false})
                return
            }

            // Move toward goal
            step := computeStep(goal.Position)
            executeStep(step)

            // Send feedback
            handle.SendFeedback(&NavFeedback{
                DistanceRemaining: distanceTo(goal.Position),
                Progress:          computeProgress(),
            })

            time.Sleep(100 * time.Millisecond)
        }

        handle.SetSucceeded(&NavResult{
            Success:       true,
            FinalPosition: currentPosition(),
        })
    },
)
```

**Client usage**:
```go
client, _ := action.NewClient[*NavGoal, *NavFeedback, *NavResult](
    node, "navigation.go_to")

goal := &NavGoal{Position: &geometry.Point{X: 10, Y: 5}}
handle, _ := client.SendGoal(ctx, goal)

// Monitor feedback
for fb := range handle.Feedback() {
    log.Printf("Progress: %.1f%%, Distance: %.2fm",
        fb.Progress*100, fb.DistanceRemaining)
}

// Get result
result, err := handle.Wait(ctx)
if result.Success {
    log.Printf("Arrived at %v", result.FinalPosition)
}
```

**Cancellation support**:
```go
// Client can cancel
handle.Cancel()

// Server checks for cancellation
if handle.IsCanceling() {
    cleanup()
    handle.SetCanceled(&NavResult{})
    return
}
```

Use actions for:
- Navigation to waypoints
- Arm trajectory execution
- Scanning/searching behaviors
- Any operation lasting more than a few seconds
