# AI-Assisted Development

Gorai is designed with AI-assisted development in mind. Clear interfaces, consistent patterns, and comprehensive specifications make AI tools effective collaborators.

## The AI Development Philosophy

From Gorai's design principles: "use AI assisted coding wherever possible"

This isn't about replacing developers—it's about amplifying them:

- **Speed**: Generate boilerplate, tests, documentation faster
- **Consistency**: Follow established patterns automatically
- **Exploration**: Quickly prototype ideas
- **Learning**: Understand unfamiliar code through explanation

## Effective AI Prompting for Robotics

### Component Generation

```
Create a Gorai motor component for the L298N dual H-bridge that implements
the motor.Motor interface. Include:
- SetPower with power clamping to configured max
- Direction control via IN1/IN2 pins
- PWM via ENA pin
- Thread-safe state management
- A fake implementation for testing

Follow the patterns in components/motor/ and examples/hello-sensor/.
```

**Why this works**:

- Specifies the hardware target
- Lists required features
- References existing patterns
- Requests the fake (often forgotten)

### Test Generation

```
Write table-driven tests for the DRV8833 motor's SetPower method covering:
- Normal cases: 0, 0.5, 1.0, -0.5, -1.0
- Edge cases: values beyond range (clamp), exactly at limits
- Error conditions: motor in fault state, closed motor
Follow Gorai's testing patterns from specs/testing-approach.md
```

### Protocol Buffer Design

```
Design a protobuf message for an ultrasonic range sensor that includes:
- Header with timestamp and frame_id
- Range measurement in meters
- Field of view in radians
- Min and max range capabilities
- Radiation type (ultrasound)

Follow conventions in api/proto/gorai/sensor/sensor.proto
```

## Code Review with AI

Use AI to review before committing:

```
Review this motor driver implementation for:
- Interface compliance with motor.Motor
- Thread safety (all state access protected)
- Error handling (hardware failures, invalid inputs)
- Resource cleanup (Close properly releases resources)
- Testability (dependencies injectable)
```

Common issues AI catches:

- Missing mutex protection
- Unclosed resources
- Hardcoded values that should be config
- Missing context cancellation checks

## Documentation Generation

### GoDoc from Code

```
Generate GoDoc comments for all exported symbols in this file.
Follow the style in pkg/node/node.go - brief first sentence,
then details, then example if helpful.
```

### README Generation

```
Generate a README.md for this motor driver package including:
- Brief description
- Installation instructions
- Usage example
- Configuration reference table
- Link to related specs

Keep it concise - under 100 lines.
```

## Debugging Assistance

### Error Analysis

```
This motor command is failing with "context deadline exceeded".
The motor is configured with these pins: IN1=17, IN2=18, PWM=12.
The GPIO driver is gpio.RPiDriver.

Help me debug:
1. What could cause this timeout?
2. How can I add logging to narrow it down?
3. What should I check in hardware?
```

### Log Interpretation

```
These are the last 20 log lines before the crash.
The robot was navigating to waypoint 3.
Help me understand:
1. What was the sequence of events?
2. Where did things go wrong?
3. What additional logging would help?
```

### NATS Message Inspection

```
I'm seeing these messages on "gorai.motors.left.command" but
the motor isn't responding. The motor is subscribed to this topic.
Help me debug the message flow.
```

## Specification to Implementation

Gorai's specs are designed for AI consumption:

```
Implement the temperature sensor described in specs/hello-sensor-design.md.
Generate:
1. The reader interface and Linux implementation
2. The sensor component implementing resource.Sensor
3. Unit tests for both
4. A fake reader for testing

Use existing patterns from examples/hello-sensor/ as reference.
```

**Workflow**:

1. Write/review specification (human)
2. Generate initial implementation (AI)
3. Review and refine (human)
4. Generate tests (AI)
5. Run tests, fix issues (collaborative)

## Limitations and Pitfalls

### Hardware-Specific Knowledge Gaps

AI doesn't know:

- Your specific wiring
- Timing requirements of your hardware
- Environmental factors (EMI, temperature)

Always verify:

- Pin assignments match physical connections
- Timing meets hardware specs
- Edge cases tested on real hardware

### Testing on Real Hardware Still Required

AI-generated code may pass unit tests but fail on hardware:

- GPIO timing issues
- I2C address conflicts
- Power supply limitations

### Security Review Importance

AI may generate insecure patterns:

- Hardcoded credentials
- Unsafe command execution
- Unvalidated inputs

Review all AI-generated code for security implications.

### Safety-Critical Code

**Never use AI-generated code unreviewed for**:

- Emergency stop logic
- Motor power limiting
- Collision detection
- Human safety interlocks

These require human verification and testing.

## Workflow Integration

### Editor Integration

VS Code with Copilot/Cody:

- Inline completions as you type
- Chat for questions and generation
- Reference Gorai patterns in prompts

### CLI Tools

Use AI from command line:

```bash
# Generate component
ai "Create a Gorai component for BMP280 temperature/pressure sensor"

# Explain code
ai "Explain what this NATS subscription does" < code.go

# Debug
ai "Why might this test be flaky?" < test_output.txt
```

### CI/CD Assistance

AI can help with:

- Debugging failing CI
- Optimizing build times
- Generating release notes

```
These CI tests passed locally but fail on GitHub Actions.
The difference is: local is macOS, CI is ubuntu-latest.
Help me understand platform-specific issues.
```

## Best Practices for AI-Assisted Robotics Development

### Provide Context

Include relevant information:

```
I'm building a differential drive robot with:
- 2x DC motors with encoders
- Raspberry Pi 5
- DRV8833 motor driver

Help me implement velocity control using encoder feedback.
```

### Reference Existing Code

```
Following the pattern in components/motor/motor.go, create a
servo component that implements similar interfaces.
```

### Request Tests Alongside Code

```
Generate the motor driver implementation AND comprehensive
unit tests using the fake pattern.
```

### Iterate Incrementally

Rather than asking for a complete system:

1. Start with the interface
2. Add basic implementation
3. Add error handling
4. Add tests
5. Add documentation

### Validate Generated Code

Always:

1. Read the generated code
2. Run the tests
3. Test on actual hardware when applicable
4. Review for security issues

## Example: Complete AI-Assisted Workflow

**Step 1: Generate Interface**

```
Design an interface for a servo motor component that can:
- Move to absolute position (0-180 degrees)
- Move relative to current position
- Query current position
- Set movement speed

Follow Gorai's component interface patterns.
```

**Step 2: Generate Implementation**

```
Implement the servo interface using PWM control.
Assume standard servo timing: 500-2500 microseconds for 0-180 degrees.
Include thread-safe state management.
```

**Step 3: Generate Fake**

```
Create a fake implementation of the servo interface for testing.
Include helpers to:
- Query commanded position
- Simulate movement completion
- Inject errors
```

**Step 4: Generate Tests**

```
Write comprehensive tests covering:
- Normal operation (move to various positions)
- Edge cases (0, 180, out of range)
- Concurrent access
- Error conditions
```

**Step 5: Human Review and Integration**

Review generated code, test on hardware, integrate into project.

---

Chapter 18 concludes with the Gorai vision and your next steps as a developer.
