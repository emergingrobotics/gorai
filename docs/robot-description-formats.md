# Robot Description Languages: A Comprehensive Analysis

## Overview

Robot description languages are XML-based (or similar) formats that describe the physical, kinematic, dynamic, and behavioral properties of robots. The field has not converged on a single standard, and each format has different strengths and limitations depending on use case.

## Primary Formats

### 1. URDF (Unified Robot Description Format)

**Origin:** ROS ecosystem (~2009)
**Specification:** [ROS 2 URDF Documentation](https://docs.ros.org/en/humble/Tutorials/Intermediate/URDF/URDF-Main.html)

**What it describes:**

| Category | Support | Notes |
|----------|---------|-------|
| **Links** (rigid bodies) | Yes | Visual, collision, inertial properties |
| **Joints** | Yes | Revolute, prismatic, fixed, continuous, floating, planar |
| **Kinematics** | Yes | Joint limits, axis, origin |
| **Dynamics** | Partial | Inertia, mass, friction (limited) |
| **Visual geometry** | Yes | Meshes, primitives, materials |
| **Collision geometry** | Yes | Simplified shapes for collision detection |
| **Sensors** | No | No native support |
| **Actuators** | Partial | Only via `<transmission>` element |
| **Controllers** | No | Handled separately (ros2_control) |
| **World/Environment** | No | Single robot only |
| **Closed kinematic loops** | No | Tree structure only |

**Key structure:**

```xml
<robot name="my_robot">
  <link name="base_link">
    <visual>...</visual>
    <collision>...</collision>
    <inertial>...</inertial>
  </link>
  <joint name="joint1" type="revolute">
    <parent link="base_link"/>
    <child link="link1"/>
    <axis xyz="0 0 1"/>
    <limit lower="-3.14" upper="3.14" effort="100" velocity="1.0"/>
  </joint>
  <transmission name="trans1">
    <type>transmission_interface/SimpleTransmission</type>
    <joint name="joint1"/>
    <actuator name="motor1"/>
  </transmission>
</robot>
```

**Limitations:**

- Cannot represent parallel robots or closed-loop mechanisms
- No sensor definitions
- Cannot position robot in a world
- Large files become unwieldy (addressed by Xacro)

---

### 2. Xacro (XML Macros)

**Origin:** ROS ecosystem
**Specification:** [Xacro GitHub](https://github.com/ros/xacro), [ROS 2 Tutorial](https://docs.ros.org/en/humble/Tutorials/Intermediate/URDF/Using-Xacro-to-Clean-Up-a-URDF-File.html)

**What it adds to URDF:**

- **Properties/Constants**: Define values once, reuse everywhere
- **Math expressions**: `${pi/2}`, `${wheel_radius * 2}`
- **Macros**: Parameterized reusable blocks
- **Includes**: Modular file organization
- **Conditionals**: `<xacro:if>`, `<xacro:unless>`

**Example:**

```xml
<xacro:property name="wheel_radius" value="0.05"/>
<xacro:macro name="wheel" params="prefix reflect">
  <link name="${prefix}_wheel">
    <visual>
      <geometry><cylinder radius="${wheel_radius}" length="0.02"/></geometry>
    </visual>
  </link>
</xacro:macro>
<xacro:wheel prefix="left" reflect="1"/>
<xacro:wheel prefix="right" reflect="-1"/>
```

Xacro is a preprocessor—it compiles down to standard URDF.

---

### 3. SDF (Simulation Description Format)

**Origin:** Gazebo simulator
**Specification:** [sdformat.org](http://sdformat.org/), [GitHub](https://github.com/gazebosim/sdformat)

**What it describes:**

| Category | Support | Notes |
|----------|---------|-------|
| **Links/Joints** | Yes | Same as URDF, plus more joint types |
| **Kinematics** | Yes | Supports closed kinematic loops |
| **Dynamics** | Yes | Full physics properties |
| **Sensors** | Yes | Cameras, lidars, IMUs, contact, GPS, etc. |
| **Actuators** | Partial | Via plugins |
| **World/Environment** | Yes | Lighting, physics settings, static objects |
| **Multiple robots** | Yes | Full scene description |
| **State** | Yes | Can store robot/world state |
| **Plugins** | Yes | Extensible via Gazebo plugins |

**Key structure:**

```xml
<sdf version="1.9">
  <world name="default">
    <physics type="ode">...</physics>
    <light type="directional" name="sun">...</light>
    <model name="my_robot">
      <link name="base_link">...</link>
      <joint name="joint1" type="revolute">...</joint>
      <sensor name="camera" type="camera">
        <camera>
          <horizontal_fov>1.047</horizontal_fov>
          <image><width>640</width><height>480</height></image>
        </camera>
      </sensor>
      <plugin name="diff_drive" filename="libgazebo_ros_diff_drive.so">
        <wheel_separation>0.5</wheel_separation>
      </plugin>
    </model>
  </world>
</sdf>
```

**Advantages over URDF:**

- Native sensor support (cameras, lidars, IMUs, contact sensors)
- Full world description (environments, lighting, physics)
- Closed-loop kinematic chains
- Multiple robots in one file

**Limitations:**

- No `<transmission>` tag (harder to integrate with ros2_control)
- Less portable outside Gazebo ecosystem

---

### 4. SRDF (Semantic Robot Description Format)

**Origin:** MoveIt (motion planning)
**Specification:** [MoveIt URDF/SRDF Tutorial](https://moveit.picknik.ai/main/doc/examples/urdf_srdf/urdf_srdf_tutorial.html), [GitHub](https://github.com/moveit/srdfdom/tree/ros2)

**What it describes (complements URDF):**

| Category | Description |
|----------|-------------|
| **Planning groups** | Named collections of joints/links (e.g., "arm", "gripper") |
| **End effectors** | Which group is the end effector, parent link |
| **Virtual joints** | Robot-to-world attachment (fixed, planar, floating) |
| **Passive joints** | Unactuated joints (e.g., casters) |
| **Disabled collisions** | Pairs of links that never collide (optimization) |
| **Named configurations** | Predefined poses ("home", "ready", "tucked") |

**Example:**

```xml
<robot name="my_robot">
  <group name="arm">
    <chain base_link="base_link" tip_link="gripper_link"/>
  </group>
  <group name="gripper">
    <joint name="finger_left_joint"/>
    <joint name="finger_right_joint"/>
  </group>
  <end_effector name="gripper_ee" parent_link="wrist" group="gripper"/>
  <virtual_joint name="world_joint" type="fixed"
                 parent_frame="world" child_link="base_link"/>
  <group_state name="home" group="arm">
    <joint name="joint1" value="0"/>
    <joint name="joint2" value="-1.57"/>
  </group_state>
  <disable_collisions link1="base_link" link2="link1" reason="Adjacent"/>
</robot>
```

SRDF is **semantic**—it describes intent and usage patterns, not physical properties.

---

### 5. MJCF (MuJoCo Format)

**Origin:** MuJoCo physics engine (now open source via DeepMind)
**Specification:** [MuJoCo XML Reference](https://mujoco.readthedocs.io/en/latest/XMLreference.html)

**What it describes:**

| Category | Support | Notes |
|----------|---------|-------|
| **Bodies/Joints** | Yes | Hierarchical body tree |
| **Kinematics** | Yes | Including tendon routing, closed loops |
| **Dynamics** | Yes | Contact, friction, soft constraints |
| **Actuators** | Yes | Motors, muscles, position/velocity servos |
| **Sensors** | Yes | Force, torque, accelerometer, gyro, touch |
| **Tendons** | Yes | Spatial/fixed tendons, pulley systems |
| **Equality constraints** | Yes | Joint coupling, closed loops |
| **Contact tuning** | Yes | Per-geom contact parameters |

**Key structure:**

```xml
<mujoco model="robot">
  <default>
    <joint damping="0.1"/>
    <geom friction="0.8 0.02 0.01"/>
  </default>
  <worldbody>
    <body name="torso" pos="0 0 1">
      <joint name="root" type="free"/>
      <geom type="capsule" size="0.1 0.3"/>
      <body name="upper_arm" pos="0.15 0 0.25">
        <joint name="shoulder" type="hinge" axis="0 1 0"/>
        <geom type="capsule" size="0.05 0.2"/>
      </body>
    </body>
  </worldbody>
  <actuator>
    <motor joint="shoulder" gear="50"/>
  </actuator>
  <sensor>
    <jointpos joint="shoulder"/>
    <touch site="fingertip"/>
  </sensor>
</mujoco>
```

**Unique strengths:**

- Native actuator models (motors, muscles, servos)
- Tendon/pulley systems for cable-driven robots
- Excellent for RL/ML applications
- CSS-like defaults cascade

---

### 6. COLLADA (.dae)

**Origin:** Khronos Group (3D interchange)
**Specification:** [Wikipedia](https://en.wikipedia.org/wiki/COLLADA)

**What it describes:**

- 3D geometry and materials
- Kinematics (since v1.5)
- Physics/dynamics (since v1.4)
- Animations

**Robotics usage:**

- Intermediate format for mesh exchange
- IKFast (OpenRAVE) uses COLLADA for analytical IK generation
- Can represent closed kinematic loops
- Convert URDF to COLLADA via `urdf_to_collada`

**Limitation:** More complex than needed for pure robotics; primarily a 3D content format.

---

### 7. USD (Universal Scene Description)

**Origin:** Pixar, now NVIDIA Omniverse/Isaac Sim
**Specification:** [NVIDIA USD](https://www.nvidia.com/en-us/omniverse/usd/)

**What it describes:**

- Complete 3D scenes with composition/layering
- Physics properties (via PhysX integration)
- Materials and rendering
- Semantic annotations

**Robotics usage in Isaac Sim:**

- Import URDF/MJCF with USD conversion
- Photorealistic rendering for synthetic data
- Multi-robot simulation at scale
- "SimReady" assets with physics/semantic properties

**Unique strengths:**

- Layer composition (override properties non-destructively)
- Massive scale (factory-level digital twins)
- GPU-accelerated rendering and physics

---

### 8. YARP Robot Configuration

**Origin:** iCub/IIT
**Specification:** [iCub Robot Configuration](https://icub-tech-iit.github.io/documentation/icub_kinematics/icub-robot-configuration/icub-robot-configuration/)

YARP itself doesn't define a robot description format—it **uses URDF and SDF** for models. The [icub-models](https://github.com/robotology/icub-models) repository provides URDF/SDF files for iCub robots.

**What YARP adds:**

- **Device configuration**: XML files describing hardware parameters
- **yarprobotinterface**: Loads device configurations to start robot
- **Resource finding**: `YARP_ROBOT_NAME` environment variable for model lookup

```xml
<!-- YARP device configuration (not robot geometry) -->
<device name="left_arm_mc" type="canBusMotionControl">
  <param name="CanDeviceNum">0</param>
  <param name="CanAddress">1 2 3 4</param>
  <group name="VELOCITY">
    <param name="Shifts">8 8 8 8</param>
  </group>
</device>
```

YARP separates **model description** (URDF/SDF) from **device/hardware configuration** (custom XML).

---

## Comparison Matrix

| Feature | URDF | SDF | SRDF | MJCF | USD |
|---------|------|-----|------|------|-----|
| **Geometry** | Yes | Yes | No | Yes | Yes |
| **Kinematics** | Yes | Yes | No | Yes | Yes |
| **Closed loops** | No | Yes | No | Yes | Yes |
| **Dynamics/Inertia** | Yes | Yes | No | Yes | Yes |
| **Sensors** | No | Yes | No | Yes | Yes |
| **Actuators** | Partial | Plugin | No | Yes | Plugin |
| **Controllers** | No | Plugin | No | No | Plugin |
| **Planning groups** | No | No | Yes | No | No |
| **World/Environment** | No | Yes | No | Yes | Yes |
| **Multiple robots** | No | Yes | No | Yes | Yes |
| **ROS integration** | Native | Via Gazebo | MoveIt | External | Isaac |

---

## Simulation vs. Runtime Configuration Analysis

A critical distinction exists between data needed for **simulation** (physics engines, visualization, training environments) versus data needed to **configure and run actual robot software**. Most robot description formats were designed primarily for simulation, with runtime configuration being an afterthought or handled separately.

### Purpose Classification by Format

| Format | Primary Purpose | Simulation | Runtime Config | Notes |
|--------|----------------|------------|----------------|-------|
| **URDF** | Both | 70% | 30% | Geometry for sim; kinematics for control |
| **Xacro** | Authoring | 100% | 0% | Preprocessor only, compiles to URDF |
| **SDF** | Simulation | 95% | 5% | Designed for Gazebo physics simulation |
| **SRDF** | Runtime | 5% | 95% | Motion planning configuration |
| **MJCF** | Simulation | 90% | 10% | Physics/RL training focus |
| **USD** | Simulation | 99% | 1% | Rendering and digital twins |
| **COLLADA** | Interchange | 100% | 0% | Mesh/animation exchange |
| **YARP Config** | Runtime | 0% | 100% | Hardware device configuration |

### What Simulation Needs (But Runtime Doesn't)

These properties are essential for physics simulation but largely irrelevant to robot software:

```
Simulation-Only Data:
├── Inertia tensors (mass distribution)
├── Collision meshes (simplified geometry)
├── Friction coefficients
├── Contact parameters (stiffness, damping)
├── Visual meshes and materials
├── Lighting and rendering properties
├── Physics solver settings (step size, iterations)
├── World/environment geometry
└── Sensor noise models
```

**Why runtime doesn't need this:**
- Real robot has actual physics—no need to simulate it
- Collision detection happens in the real world
- Visual appearance is irrelevant to control software
- Sensor noise is measured, not modeled

### What Runtime Needs (But Simulation Doesn't)

These properties are essential for running robot software but often missing from description formats:

```
Runtime-Only Data:
├── Hardware interface parameters
│   ├── Serial port paths (/dev/ttyUSB0)
│   ├── CAN bus IDs and baud rates
│   ├── I2C addresses
│   └── GPIO pin assignments
├── Communication configuration
│   ├── Topic names and namespaces
│   ├── Message types and frequencies
│   └── QoS settings
├── Controller parameters
│   ├── PID gains (tuned for real hardware)
│   ├── Motion limits (calibrated)
│   └── Safety thresholds
├── Calibration data
│   ├── Encoder offsets
│   ├── Camera intrinsics
│   └── Sensor biases
├── Operational parameters
│   ├── Update rates
│   ├── Timeout values
│   └── Retry policies
└── Component dependencies
    ├── Which services need which hardware
    └── Startup ordering
```

### What Both Need (The Overlap)

A small subset of data serves both simulation and runtime:

```
Shared Data:
├── Kinematic structure (links, joints, transforms)
├── Joint types and axes
├── Joint limits (position, velocity, effort)
├── Link names and hierarchy
└── Basic geometric dimensions
```

This overlap is why URDF became popular—it captures the essential kinematic model that both simulators and control software need.

### Analysis by Format

#### URDF: The Uncomfortable Middle Ground

URDF tries to serve both purposes but does neither perfectly:

| Element | Sim Use | Runtime Use |
|---------|---------|-------------|
| `<link><inertial>` | Physics calculation | Rarely used |
| `<link><visual>` | Rendering | Unused |
| `<link><collision>` | Collision detection | Unused |
| `<joint><limit>` | Physics constraints | **Control limits** |
| `<joint><axis>` | Physics | **Kinematics** |
| `<transmission>` | Sim actuation | **ros2_control mapping** |

The `<transmission>` element is the only URDF feature designed specifically for runtime—it maps joints to actuators for ros2_control.

#### SDF: Simulation First

SDF was designed for Gazebo and it shows:

- **Sensor definitions** describe simulated sensors, not real hardware
- **Plugin system** is Gazebo-specific
- **World elements** have no runtime equivalent
- **Physics settings** are simulation parameters

SDF sensors describe *what to simulate* (camera resolution, lidar range) not *how to connect* (USB device, driver parameters).

#### SRDF: Pure Runtime Configuration

SRDF is the exception—it's entirely about runtime behavior:

- **Planning groups**: Which joints to plan together
- **Named poses**: Predefined configurations for easy recall
- **Disabled collisions**: Optimization for planning (not physics)
- **End effectors**: Semantic information for manipulation

SRDF has zero simulation purpose; it configures MoveIt's motion planning.

#### MJCF: Simulation with Better Actuator Models

MJCF's actuator section is more sophisticated than URDF but still simulation-focused:

```xml
<actuator>
  <motor joint="shoulder" gear="50"/>           <!-- Simulated motor -->
  <position joint="elbow" kp="100"/>            <!-- Simulated servo -->
  <velocity joint="wrist" kv="10"/>             <!-- Simulated velocity control -->
</actuator>
```

These define *how to simulate* control, not *how to interface* with real motors.

### The Gap: What's Missing for Runtime

No standard format adequately describes:

1. **Hardware Binding**
   ```
   How does "motor1" connect to physical hardware?
   - ODrive on CAN bus 0, axis 1?
   - GPIO PWM on pin 18?
   - Dynamixel ID 3 on /dev/ttyUSB0?
   ```

2. **Protocol Configuration**
   ```
   How do we communicate with the device?
   - Baud rate, data bits, parity?
   - CAN message IDs and formats?
   - Register addresses for I2C?
   ```

3. **Calibration Storage**
   ```
   What are the tuned values for THIS specific robot?
   - Encoder zero positions
   - PID gains that work on real hardware
   - Sensor offsets measured during calibration
   ```

4. **Operational Semantics**
   ```
   How should the software behave?
   - Update rates and priorities
   - Error handling policies
   - Graceful degradation modes
   ```

### How ROS 2 Handles This

ROS 2 acknowledges the gap by using multiple configuration layers:

```
ROS 2 Configuration Stack:
├── URDF/Xacro          → Kinematics + simulation
├── SRDF                → Motion planning semantics
├── ros2_control YAML   → Hardware interface config
├── Parameter files     → Runtime tuning
└── Launch files        → Composition and startup
```

The ros2_control YAML fills the hardware binding gap:

```yaml
controller_manager:
  ros__parameters:
    joint_state_broadcaster:
      type: joint_state_broadcaster/JointStateBroadcaster
    arm_controller:
      type: joint_trajectory_controller/JointTrajectoryController

arm_controller:
  ros__parameters:
    joints:
      - shoulder_joint
      - elbow_joint
    command_interfaces:
      - position
    state_interfaces:
      - position
      - velocity
```

But hardware specifics (serial ports, CAN IDs) go in yet another layer.

### Implications for Gorai

Given this analysis, Gorai should:

1. **Don't reinvent URDF** for kinematics—import it when needed
2. **Focus on runtime configuration** since that's the underserved area
3. **Separate concerns clearly**:

```
Gorai Configuration Model:
├── Robot Model (import URDF)
│   └── Kinematics, joint limits, link structure
├── Component Manifest (Gorai-native)
│   ├── Component types and subtypes
│   ├── Hardware bindings (Link references)
│   ├── NATS topic mappings
│   └── Dependencies
├── Hardware Configuration (Gorai-native)
│   ├── Link definitions (serial, CAN, I2C, radio)
│   ├── Protocol parameters
│   └── Device addresses
└── Runtime Parameters (Gorai-native)
    ├── Control gains
    ├── Update rates
    ├── Calibration values
    └── Safety limits
```

4. **Make simulation optional**—not all Gorai robots need simulation
5. **Provide URDF export** for users who want Gazebo/MuJoCo integration

---

## What's NOT Typically Described

Most robot description formats focus on **physical/geometric** properties. They typically do **not** describe:

1. **Software architecture** - Node graphs, communication patterns
2. **Behavior/Logic** - State machines, decision trees
3. **Mission/Task definitions** - What the robot should accomplish
4. **Network topology** - How components communicate
5. **Safety constraints** - Operational limits, zones, interlocks
6. **Calibration data** - Camera intrinsics, sensor offsets
7. **Maintenance info** - Part numbers, service intervals

These are handled by separate configuration systems (ROS 2 launch files, behavior trees, parameter servers, etc.).

---

## Implications for Gorai

Given this landscape, Gorai could consider:

1. **Support URDF/SDF import** for compatibility with existing robot models
2. **Define a separate "component manifest"** that describes what URDF doesn't:
   - Component types (sensor/actuator/service)
   - NATS topic mappings
   - Hardware interface parameters
   - Behavioral capabilities
3. **Use configuration (not description) for runtime** - Similar to YARP's separation of model vs. device config
4. **Consider a lightweight schema** that references URDF for geometry but adds Gorai-specific metadata

---

## Sources

- [ROS 2 URDF Documentation](https://docs.ros.org/en/humble/Tutorials/Intermediate/URDF/URDF-Main.html)
- [SDFormat.org](http://sdformat.org/)
- [MoveIt SRDF Tutorial](https://moveit.picknik.ai/main/doc/examples/urdf_srdf/urdf_srdf_tutorial.html)
- [MuJoCo XML Reference](https://mujoco.readthedocs.io/en/latest/XMLreference.html)
- [NVIDIA Isaac Sim Documentation](https://docs.isaacsim.omniverse.nvidia.com/latest/index.html)
- [iCub Models Repository](https://github.com/robotology/icub-models)
- [Xacro GitHub](https://github.com/ros/xacro)
- [COLLADA Wikipedia](https://en.wikipedia.org/wiki/COLLADA)
