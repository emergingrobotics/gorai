## 5.7 Base Interface (Mobile Robots)

The Base interface abstracts mobile robot locomotion:

```go
type Base interface {
    component.Actuator

    // SetVelocity sets linear and angular velocity.
    SetVelocity(ctx context.Context, linear, angular float64) error

    // MoveStraight moves the robot forward by the specified distance.
    MoveStraight(ctx context.Context, distanceMm int, velocity float64) error

    // Spin rotates the robot in place by the specified angle.
    Spin(ctx context.Context, angleDeg, velocity float64) error

    // GetVelocities returns current linear and angular velocities.
    GetVelocities(ctx context.Context) (linear, angular float64, err error)
}
```

### Drive Types

**Differential Drive**: Two independently controlled wheels
```go
// Convert linear/angular to wheel velocities
func (b *DiffDrive) setWheelVelocities(linear, angular float64) {
    // v_left = linear - angular * (wheelbase / 2)
    // v_right = linear + angular * (wheelbase / 2)
    vLeft := linear - angular*b.wheelbase/2
    vRight := linear + angular*b.wheelbase/2

    b.leftMotor.SetVelocity(ctx, vLeft/b.wheelRadius)
    b.rightMotor.SetVelocity(ctx, vRight/b.wheelRadius)
}
```

**Mecanum Wheels**: Omnidirectional movement
```go
// Four-wheel mecanum kinematics
func (b *Mecanum) setWheelVelocities(vx, vy, angular float64) {
    // Each wheel contributes differently to motion
    fl := vx - vy - angular*(b.lx+b.ly)  // Front left
    fr := vx + vy + angular*(b.lx+b.ly)  // Front right
    rl := vx + vy - angular*(b.lx+b.ly)  // Rear left
    rr := vx - vy + angular*(b.lx+b.ly)  // Rear right

    b.motors["fl"].SetVelocity(ctx, fl)
    b.motors["fr"].SetVelocity(ctx, fr)
    b.motors["rl"].SetVelocity(ctx, rl)
    b.motors["rr"].SetVelocity(ctx, rr)
}
```

**Ackermann Steering**: Car-like steering
```go
func (b *Ackermann) SetVelocity(ctx context.Context, linear, angular float64) error {
    // Convert angular velocity to steering angle
    // Using bicycle model: angular = linear * tan(steering) / wheelbase
    if linear != 0 {
        steering := math.Atan(angular * b.wheelbase / linear)
        b.steeringServo.Move(ctx, steeringToDegrees(steering))
    }

    b.driveMotor.SetVelocity(ctx, linear)
    return nil
}
```

### Velocity Commands

Typically expressed as Twist (linear + angular):
```protobuf
message Twist {
    Vector3 linear = 1;   // Linear velocity (x=forward, y=left, z=up)
    Vector3 angular = 2;  // Angular velocity (roll, pitch, yaw)
}
```

For ground robots, typically only use:
- linear.x: Forward/backward
- angular.z: Turn left/right


## 5.8 Arm Interface (Manipulators)

Robotic arms require sophisticated interfaces:

```go
type Arm interface {
    component.Actuator

    // EndPosition returns the current end effector pose.
    EndPosition(ctx context.Context) (*spatialmath.Pose, error)

    // MoveToPosition moves the end effector to the target pose.
    MoveToPosition(ctx context.Context, pose *spatialmath.Pose) error

    // JointPositions returns current joint angles.
    JointPositions(ctx context.Context) ([]float64, error)

    // MoveToJointPositions sets joint angles directly.
    MoveToJointPositions(ctx context.Context, positions []float64) error
}
```

### Joint Space vs Task Space

**Joint space**: Direct control of each joint angle
```go
// Move each joint to specific angle
positions := []float64{0, -45, 90, 0, 45, 0}  // degrees
arm.MoveToJointPositions(ctx, positions)
```
- Direct, predictable
- Requires knowing valid configurations
- Good for predefined poses

**Task space**: Control end effector position/orientation
```go
// Move end effector to position
pose := spatialmath.NewPoseFromPoint(r3.Vector{X: 0.3, Y: 0.1, Z: 0.4})
arm.MoveToPosition(ctx, pose)
```
- More intuitive for applications
- Requires inverse kinematics
- May have multiple solutions or none

### Forward/Inverse Kinematics Concepts

**Forward kinematics**: Joints → End effector position
```
Given: Joint angles [θ1, θ2, θ3, ...]
Find: End effector pose (x, y, z, rotation)
```
Always has a unique solution.

**Inverse kinematics**: End effector position → Joints
```
Given: Desired end effector pose
Find: Joint angles to achieve it
```
May have:
- Multiple solutions (elbow up vs elbow down)
- No solution (target unreachable)
- Singularities (infinite solutions along an axis)

### Trajectory Planning

Moving from A to B requires planning:

```go
type Trajectory struct {
    Points []TrajectoryPoint
}

type TrajectoryPoint struct {
    Time     time.Duration
    Joints   []float64
    Velocity []float64
}

func (a *Arm) ExecuteTrajectory(ctx context.Context, traj *Trajectory) error {
    start := time.Now()

    for _, point := range traj.Points {
        // Wait for point time
        elapsed := time.Since(start)
        if point.Time > elapsed {
            time.Sleep(point.Time - elapsed)
        }

        // Move to point
        if err := a.MoveToJointPositions(ctx, point.Joints); err != nil {
            return err
        }
    }
    return nil
}
```

**Trajectory types**:
- Point-to-point: Direct joint interpolation
- Cartesian: Straight line in task space
- Spline: Smooth curves through waypoints

---

With sensors and actuators covered, Chapter 6 explores vision—the intersection of sensors and AI that enables robots to perceive and understand their environment.
