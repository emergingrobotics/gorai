# Prosumer Robotics Sensors: A Comprehensive Analysis

## Overview

This document analyzes the most common sensor types used in prosumer robotics, covering vision systems, range finders, motion detection, orientation sensing, environmental monitoring, and more. For each sensor type, we examine the control parameters, data outputs, electrical interfaces, and communication protocols relevant to Gorai integration.

## Sensor Categories

### 1. Vision Systems (Cameras)

**Use Cases:** Object detection, navigation, SLAM, teleoperation, visual servoing

Vision sensors range from simple webcams to sophisticated depth cameras with onboard AI processing.

#### Categories

| Type | Technology | Output | Best For |
|------|------------|--------|----------|
| **RGB Camera** | CMOS/CCD | Video frames | Object recognition, streaming |
| **Stereo Camera** | Dual CMOS + processing | RGB + depth | SLAM, 3D mapping |
| **Structured Light** | IR projector + camera | Depth map | Close-range depth |
| **ToF Camera** | Time-of-Flight array | Depth map | Real-time depth |
| **Event Camera** | DVS pixels | Event stream | High-speed motion |

#### Stereo/Depth Camera Specifications

**Stereolabs ZED 2i**
```
Sensor Specifications:
├── Resolution: Up to 2K (2208x1242) stereo
├── Frame Rate: Up to 100 FPS at 720p
├── Depth Range: 0.2 - 20 meters
├── Baseline: 120mm
├── Field of View: 110° (H) x 70° (V)
├── IMU: 6-axis, factory calibrated
├── Sensors: Barometer, magnetometer, temperature
└── Rating: IP66

Interface:
├── Connection: USB 3.0
├── SDK: ZED SDK (CUDA required)
├── Output: RGB, Depth, Point Cloud, IMU
└── ROS/ROS2: Native support
```

**Intel RealSense D455/D457**
```
Sensor Specifications:
├── Depth Technology: Active IR Stereo
├── Depth Resolution: Up to 1280x720
├── Depth Frame Rate: Up to 90 FPS
├── Depth Range: 0.4 - 6 meters (indoor)
├── RGB Resolution: 1920x1080
├── Field of View: 87° x 58°
├── Baseline: ~95mm
└── IMU: Built-in (D455)

Interface:
├── D455: USB 3.1
├── D457: GMSL2/FAKRA (automotive grade)
├── SDK: librealsense
├── Output: Depth, RGB, IR, Point Cloud
└── ROS/ROS2: Native packages
```

**Luxonis OAK-D**
```
Sensor Specifications:
├── Stereo Baseline: 75mm
├── Depth Resolution: 640x400 to 1280x800
├── RGB Resolution: Up to 4K
├── Onboard AI: 4 TOPS (1.4 TOPS for AI)
├── Neural Inference: MobileNet, YOLO, etc.
├── Encoding: H.264, H.265, MJPEG
└── Frame Rate: 4K@30, 1080p@60

Interface:
├── Connection: USB 3.1 Type-C
├── Power: USB or external 5V
├── SDK: DepthAI (Python/C++)
├── Output: RGB, Depth, Neural output
└── ROS/ROS2: depthai-ros package
```

#### Key Parameters for Vision Systems

```
Camera Parameters:
├── Resolution (width x height pixels)
├── Frame Rate (FPS)
├── Field of View (degrees, H x V)
├── Pixel Size (µm)
├── Sensor Format (1/2", 1/3", etc.)
├── Lens Type (fixed, interchangeable)
├── Shutter Type (rolling, global)
└── Low Light Performance (lux rating)

Depth Parameters:
├── Minimum Range (meters)
├── Maximum Range (meters)
├── Depth Accuracy (% or mm)
├── Depth Resolution (bits)
├── Baseline (mm, for stereo)
├── Fill Rate (%)
└── Latency (ms)

Control Parameters:
├── Exposure (auto/manual, µs)
├── Gain (dB or ISO equivalent)
├── White Balance (K)
├── Focus (if adjustable)
├── HDR Mode (on/off)
└── Depth Mode (high accuracy/density)
```

#### Electrical Interfaces

| Camera Type | Power | Data | Typical Voltage |
|-------------|-------|------|-----------------|
| USB Camera | USB | USB 2.0/3.0 | 5V via USB |
| MIPI CSI | External | CSI-2 lanes | 3.3V/1.8V |
| GMSL2 | External | Coaxial | 12V nominal |
| GigE Vision | PoE | Ethernet | 48V PoE |

---

### 2. LiDAR Sensors

**Use Cases:** SLAM, obstacle avoidance, mapping, autonomous navigation

LiDAR provides precise distance measurements using laser light, creating 2D or 3D point clouds.

#### Categories

| Type | Dimensions | Range | Use Case |
|------|------------|-------|----------|
| **2D Scanning** | Single plane | 6-40m | Indoor navigation |
| **3D Scanning** | Multi-plane | 10-200m | Outdoor mapping |
| **Solid State** | Fixed array | 5-50m | Automotive, drones |

#### Common 2D LiDAR Specifications

**SLAMTEC RPLIDAR A1**
```
Specifications:
├── Range: 0.15 - 12 meters
├── Sample Rate: 8000 samples/sec
├── Scan Rate: 5.5 Hz (360°)
├── Angular Resolution: ~1°
├── Range Resolution: <0.5% of distance
├── Laser Class: Class 1 (eye-safe)
└── Dimensions: 70mm diameter

Interface:
├── Communication: UART (3.3V TTL)
├── Baud Rate: 115200 default
├── Connector: 5-pin (VCC, GND, TX, RX, MOTOCTL)
├── Power: 5V, ~500mA
├── Data Format: Binary packets
└── SDK: rplidar_sdk (C++, ROS)
```

**SLAMTEC RPLIDAR A3**
```
Specifications:
├── Range: 0.2 - 25 meters
├── Sample Rate: 16000 samples/sec
├── Scan Rate: 5-20 Hz (configurable)
├── Angular Resolution: ~0.225°
├── Modes: Indoor, Outdoor (daylight resistant)
├── Laser Class: Class 1
└── Dimensions: 76mm diameter

Interface:
├── Communication: UART (3.3V TTL)
├── USB: Via adapter
├── Baud Rate: 256000
└── ROS/ROS2: rplidar_ros package
```

**SLAMTEC RPLIDAR C1 (DTOF)**
```
Specifications:
├── Technology: Direct Time of Flight
├── Range: 0.05 - 12 meters (white)
├── Range (black): 0.05 - 6 meters
├── Sample Rate: 5000 samples/sec
├── Scan Rate: 5-10 Hz
├── Small Object Detection: Enhanced
└── Price: Lower cost than A-series

Interface:
├── Communication: UART
├── ROS/ROS2: Supported
└── Power: 5V
```

#### Key LiDAR Parameters

```
LiDAR Parameters:
├── Maximum Range (meters)
├── Minimum Range (meters)
├── Range Accuracy (mm or %)
├── Angular Resolution (degrees)
├── Sample Rate (points/second)
├── Scan Frequency (Hz)
├── Field of View (degrees)
├── Laser Wavelength (nm)
├── Laser Class (1, 1M, 2, etc.)
└── Ambient Light Immunity (klux)

Control Parameters:
├── Scan Mode (standard, boost, sensitivity)
├── Motor Speed (RPM or Hz)
├── Point Quality Threshold
├── Intensity Output (on/off)
└── Express Scan Mode (on/off)

Output Data:
├── Distance (mm)
├── Angle (degrees, 0.01° resolution typical)
├── Signal Quality/Intensity (0-255)
├── Scan Index
└── Timestamp (for synchronization)
```

#### LiDAR Communication Protocol Example (RPLIDAR)

```
Request Packet:
├── Start Flag: 0xA5
├── Command: 0x20 (scan), 0x25 (stop), etc.
└── Payload: Command specific

Response Packet:
├── Start Flag 1: 0xA5
├── Start Flag 2: 0x5A
├── Data Length: 5 bytes (typical)
├── Send Mode: Single/Continuous
├── Data Type: Measurement/Device info
└── Data: [Distance, Angle, Quality]

Scan Data Format (per point):
├── Quality (6 bits)
├── Start Flag (1 bit)
├── Reserved (1 bit)
├── Angle (15 bits, Q6 fixed point)
└── Distance (16 bits, mm)
```

---

### 3. Ultrasonic Sensors

**Use Cases:** Close-range obstacle detection, liquid level sensing, parking assist

Ultrasonic sensors use sound waves to measure distance, effective for detecting any solid surface.

#### HC-SR04 Specifications

```
Specifications:
├── Operating Voltage: 5V DC
├── Operating Current: <15mA
├── Frequency: 40 kHz
├── Range: 2cm - 400cm
├── Practical Range: 10cm - 250cm (best results)
├── Accuracy: ±3mm
├── Sensing Angle: 30° cone
├── Effective Angle: 15° cone
├── Dimensions: 45mm x 20mm x 15mm
└── Weight: 10 grams

Pinout:
├── VCC: 5V power
├── Trig: Trigger input (5V or 3.3V)
├── Echo: Echo output (5V logic!)
└── GND: Ground
```

#### Timing Protocol

```
Measurement Sequence:
1. Set Trig pin HIGH for 10µs
2. Sensor emits 8 cycles of 40kHz ultrasound
3. Echo pin goes HIGH
4. Echo pin goes LOW when echo received
5. Measure Echo pulse duration

Distance Calculation:
├── Distance (cm) = (Echo Time µs) / 58
├── Distance (inch) = (Echo Time µs) / 148
└── Based on speed of sound: 343 m/s at 20°C

Timing Constraints:
├── Minimum delay between measurements: 50ms
├── Maximum echo time: ~23ms (400cm)
├── Timeout recommended: 30ms
└── Temperature affects accuracy (speed of sound varies)
```

#### Interface Considerations

```
3.3V Microcontroller Compatibility:
├── Trig: Can accept 3.3V input
├── Echo: Outputs 5V! Requires level shifting
│   ├── Voltage divider (simple)
│   ├── Level shifter IC (reliable)
│   └── Series resistor + clamp diode
└── Some newer modules: 3.3V-5V tolerant

Multiple Sensor Handling:
├── Sequential triggering (avoid crosstalk)
├── Minimum spacing: 15° angular separation
├── Stagger timing by ~50ms between sensors
└── Some modules support PWM output mode
```

#### Key Parameters

```
Ultrasonic Parameters:
├── Operating Frequency (kHz)
├── Detection Range Min/Max (cm)
├── Beam Angle (degrees)
├── Resolution (mm)
├── Update Rate (Hz)
├── Dead Zone (minimum range)
└── Temperature Sensitivity

Control Parameters:
├── Trigger Pulse Width (µs)
├── Timeout Duration (ms)
├── Measurement Interval (ms)
└── Gain/Sensitivity (on some modules)
```

---

### 4. Time-of-Flight (ToF) Point Sensors

**Use Cases:** Precise distance measurement, gesture detection, proximity sensing

ToF sensors measure distance by timing laser light round trips, providing millimeter accuracy.

#### VL53L0X Specifications

```
Specifications:
├── Technology: SPAD (Single Photon Avalanche Diode)
├── Range: 30mm - 1200mm (default mode)
├── Long Range: Up to 2000mm (reflective target)
├── Accuracy: ±3% typical
├── Resolution: 1mm
├── Laser: 940nm VCSEL, Class 1 (eye-safe)
├── Field of View: ~25° cone
├── Measurement Time: <30ms
├── Package: 4.4 x 2.4 x 1.0 mm

Interface:
├── Protocol: I2C
├── Address: 0x29 (default, changeable)
├── Voltage: 2.6V - 3.5V (core)
├── Breakout boards: 2.6V - 5.5V
├── Current: 10-40mA (during ranging)
└── GPIO: Interrupt output available
```

#### VL53L1X Specifications

```
Specifications:
├── Technology: SPAD with extended range
├── Range: 30mm - 4000mm
├── Accuracy: ±3% (±10% in difficult conditions)
├── Resolution: 1mm
├── Update Rate: Up to 50 Hz
├── Laser: 940nm VCSEL, Class 1
├── Field of View: 27° (programmable ROI)
├── Distance Modes: Short (<1.3m), Medium, Long (4m)
├── Cover Glass: Compensation available

Interface:
├── Protocol: I2C
├── Address: 0x29 (default)
├── I2C Speed: Up to 400kHz
└── Interrupt: Configurable threshold
```

#### Key Parameters

```
ToF Sensor Parameters:
├── Maximum Range (mm)
├── Minimum Range (mm)
├── Accuracy (% or mm)
├── Field of View (degrees)
├── Update Rate (Hz)
├── Laser Wavelength (nm)
├── Laser Power/Class
└── Ambient Light Immunity (klux)

Control Parameters:
├── Distance Mode (short/medium/long)
├── Timing Budget (measurement time, ms)
├── Inter-measurement Period (ms)
├── Region of Interest (ROI, VL53L1X)
├── Threshold Interrupt (mm)
├── Signal Rate Limit
└── Sigma Limit (repeatability)

Output Data:
├── Distance (mm)
├── Signal Rate (MCPS - Mega Counts Per Second)
├── Ambient Rate (MCPS)
├── Range Status (valid, sigma fail, signal fail, etc.)
└── Effective SPAD Count
```

#### Multiple Sensor Configuration

```
Multi-Sensor I2C Setup:
├── Default address: 0x29 for all sensors
├── Strategy: Sequential initialization
│   1. Hold all XSHUT pins LOW
│   2. Release one sensor's XSHUT
│   3. Assign new I2C address
│   4. Repeat for each sensor
├── Address range: 0x29 - 0x7F
└── Alternative: I2C multiplexer (TCA9548A)
```

---

### 5. IMU (Inertial Measurement Units)

**Use Cases:** Orientation sensing, motion tracking, stabilization, dead reckoning

IMUs combine accelerometers, gyroscopes, and sometimes magnetometers to track motion and orientation.

#### Categories

| Type | Sensors | Fusion | Best For |
|------|---------|--------|----------|
| **6-DOF** | Accel + Gyro | External | Basic motion |
| **9-DOF** | Accel + Gyro + Mag | External | Full orientation |
| **AHRS** | 9-DOF + processor | Internal | Ready-to-use heading |

#### MPU6050 Specifications

```
Specifications:
├── Accelerometer: 3-axis, ±2/4/8/16g
├── Gyroscope: 3-axis, ±250/500/1000/2000°/s
├── ADC Resolution: 16-bit
├── Sample Rate: Up to 1kHz
├── On-chip DMP: Digital Motion Processor
├── Temperature Sensor: Built-in
├── Package: QFN 4x4x0.9mm
└── Price: Very low cost (~$2)

Interface:
├── Protocol: I2C (primary), SPI
├── I2C Address: 0x68 (AD0=LOW), 0x69 (AD0=HIGH)
├── I2C Speed: Up to 400kHz
├── Voltage: 2.375V - 3.46V
├── Current: 3.9mA (typical)
└── Interrupt: Data ready, FIFO, motion detect
```

#### BNO055 Specifications

```
Specifications:
├── Accelerometer: 3-axis, ±2/4/8/16g
├── Gyroscope: 3-axis, ±125/250/500/1000/2000°/s
├── Magnetometer: 3-axis, ±1300/2500µT
├── Processor: 32-bit ARM Cortex-M0
├── On-chip Fusion: Quaternions, Euler, vectors
├── Calibration: Automatic
├── Output Rate: Up to 100Hz (orientation)
├── Package: 5.2 x 3.8 x 1.1mm
└── Price: Higher (~$15-30)

Output Modes:
├── Raw sensor data
├── Fused absolute orientation
├── Linear acceleration (gravity removed)
├── Gravity vector
└── Quaternion (4 values)

Interface:
├── Protocol: I2C (primary), UART
├── I2C Address: 0x28 (default), 0x29
├── I2C Speed: Up to 400kHz
├── Voltage: 3.3V (2.4V-3.6V)
└── Current: 12.3mA (typical)
```

#### Key Parameters

```
Accelerometer Parameters:
├── Range (±g)
├── Resolution (bits)
├── Sensitivity (LSB/g)
├── Noise Density (µg/√Hz)
├── Zero-g Offset (mg)
├── Bandwidth (Hz)
└── Sample Rate (Hz)

Gyroscope Parameters:
├── Range (±°/s)
├── Resolution (bits)
├── Sensitivity (LSB/°/s)
├── Noise Density (°/s/√Hz)
├── Zero-rate Offset (°/s)
├── Bandwidth (Hz)
└── Sample Rate (Hz)

Magnetometer Parameters:
├── Range (±µT or ±Gauss)
├── Resolution (bits)
├── Sensitivity (LSB/µT)
├── Noise (nT RMS)
└── Update Rate (Hz)

Fusion Output:
├── Quaternion (w, x, y, z)
├── Euler Angles (roll, pitch, yaw)
├── Linear Acceleration (m/s²)
├── Angular Velocity (°/s or rad/s)
├── Gravity Vector (m/s²)
└── Heading/Bearing (degrees)
```

#### Comparison

| Feature | MPU6050 | BNO055 |
|---------|---------|--------|
| Axes | 6 (accel + gyro) | 9 (+ magnetometer) |
| On-chip Fusion | DMP (limited) | Full AHRS |
| Quaternion Output | Requires DMP setup | Native |
| Calibration | Manual | Automatic |
| Accuracy | Good | Excellent |
| Noise | Moderate | Low |
| Price | Very Low | Moderate |

---

### 6. Line Following / Reflectance Sensors

**Use Cases:** Line following robots, edge detection, surface contrast detection

Reflectance sensors use IR LEDs and photodetectors to measure surface reflectivity.

#### QTR-8RC Specifications

```
Specifications:
├── Channels: 8 IR LED/phototransistor pairs
├── Spacing: 9.525mm (0.375") between sensors
├── Operating Voltage: 3.3V - 5V
├── Current: 100mA (LEDs on)
├── Output: RC timing (digital I/O compatible)
├── Sensing Distance: 3mm optimal, 9.5mm max
├── Dimensions: 75mm x 13mm x 3mm
└── LED Control: MOSFET for power saving

Output Type (RC):
├── Fast decay = high reflectance (white)
├── Slow decay = low reflectance (black)
├── Typical bright reading: <500µs
├── Typical dark reading: >1000µs
└── Maximum reading: ~2500µs (timeout)
```

#### QTR-8A Specifications

```
Specifications:
├── Channels: 8 IR LED/phototransistor pairs
├── Output: Analog voltage (0V to VCC)
├── High reflectance: Low voltage
├── Low reflectance: High voltage
├── Other specs: Same as QTR-8RC

ADC Requirements:
├── 10-bit ADC: 0-1023 values
├── Calibration recommended
└── Faster reads than RC variant
```

#### Reading Sequence (RC Type)

```
RC Timing Method:
1. Drive I/O pin HIGH (output mode)
2. Wait ≥10µs for capacitor charge
3. Set pin to INPUT (high-impedance)
4. Start timer
5. Wait for pin to go LOW
6. Stop timer, record duration

Timing vs Reflectance:
├── White surface: ~100-500µs
├── Gray surface: ~500-1500µs
├── Black surface: ~1500-2500µs
└── No reflection: Timeout (~2500µs)

Power Saving:
├── LED enable pin available
├── Turn off when not reading
├── Can reduce current from 100mA to <10mA
└── Cycle at 100Hz = 90% power savings
```

#### Key Parameters

```
Reflectance Sensor Parameters:
├── Number of Channels
├── Channel Spacing (mm)
├── Operating Voltage (V)
├── Sensing Distance (mm)
├── Output Type (analog/RC)
├── LED Wavelength (typically 940nm IR)
└── Ambient Light Sensitivity

Control Parameters:
├── LED Enable (on/off)
├── Calibration Values (min/max per channel)
├── Threshold (for binary detection)
├── Integration Time (RC timeout)
└── Sample Rate (Hz)

Output Data:
├── Raw Values (per channel)
├── Calibrated Values (0-1000 normalized)
├── Line Position (weighted average)
└── Binary Detection (on/off line)
```

---

### 7. PIR Motion / Presence Sensors

**Use Cases:** Human presence detection, security, power management, wake-on-motion

PIR sensors detect infrared radiation changes from warm bodies moving across their field of view.

#### HC-SR501 Specifications

```
Specifications:
├── Detection Range: 3-7 meters (adjustable)
├── Detection Angle: ~120° cone
├── Operating Voltage: 5V - 20V DC
├── Quiescent Current: <50µA
├── Output: Digital HIGH (3.3V) when triggered
├── Output Duration: 3 sec - 5 min (adjustable)
├── Trigger Mode: Single (L), Repeating (H)
├── Block Time: ~2.5 seconds (re-trigger delay)
├── Warm-up Time: ~60 seconds
└── Dimensions: 32mm x 24mm

Pinout:
├── VCC: 5-20V power
├── OUT: 3.3V digital output
└── GND: Ground

Adjustments:
├── Sensitivity potentiometer (range)
├── Time delay potentiometer (output duration)
└── Mode jumper (single/repeat trigger)
```

#### Key Parameters

```
PIR Sensor Parameters:
├── Detection Range (meters)
├── Detection Angle (degrees)
├── Sensitivity (adjustable)
├── Operating Temperature Range
├── Output Voltage (typically 3.3V)
├── Block Time (re-trigger delay)
└── Warm-up Time

Limitations:
├── Cannot detect stationary subjects
├── Affected by ambient temperature
├── No distance information
├── No occupant count
├── Can false-trigger on pets/heat sources
└── Glass blocks IR detection
```

---

### 8. mmWave Radar Sensors

**Use Cases:** Presence detection, gesture recognition, vital signs monitoring, through-wall detection

mmWave radar provides superior presence detection including stationary subjects, unaffected by lighting or temperature.

#### DFRobot C4001 Specifications

```
Specifications:
├── Frequency: 24GHz
├── Modulation: FMCW
├── Presence Detection Range: Up to 8m
├── Motion Detection Range: Up to 12m
├── Horizontal FOV: 100°
├── Distance Measurement: 1.2m - 12m
├── Speed Detection: 0.1 - 10 m/s
├── Accuracy: >95% detection rate
├── Operating Temperature: -40°C to +85°C
└── Dimensions: 24 x 28mm

Interface:
├── Protocol: I2C or UART
├── I2C Address: Configurable
├── Voltage: 3.3V - 5V
├── Current: ~100mA active
└── Connector: Gravity (PH2.0)
```

#### mmWave vs PIR Comparison

| Feature | PIR | mmWave |
|---------|-----|--------|
| Stationary Detection | No | Yes |
| Through Materials | No | Yes (some) |
| Distance Info | No | Yes |
| Speed Info | No | Yes |
| Lighting Dependency | None | None |
| Temperature Affected | Yes | No |
| Cost | Very Low | Moderate |
| Power | Very Low | Moderate |

#### Key Parameters

```
mmWave Parameters:
├── Operating Frequency (24GHz, 60GHz, 77GHz)
├── Modulation Type (FMCW, CW)
├── Presence Detection Range (m)
├── Motion Detection Range (m)
├── Distance Resolution (cm)
├── Speed Range (m/s)
├── Field of View (degrees)
├── Update Rate (Hz)
└── False Positive Rate

Output Data:
├── Presence (boolean)
├── Motion State (static/moving)
├── Distance (cm/m)
├── Speed (m/s)
├── Direction (approaching/receding)
├── Signal Strength
└── Target Count (some sensors)
```

---

### 9. Thermal Imaging Sensors

**Use Cases:** Heat detection, thermal mapping, fever screening, hotspot identification

Low-resolution thermal arrays provide temperature distribution imaging at accessible prices.

#### AMG8833 Specifications

```
Specifications:
├── Resolution: 8x8 pixels (64 total)
├── Temperature Range: 0°C - 80°C
├── Accuracy: ±2.5°C
├── Human Detection Range: Up to 7m
├── Field of View: 60°
├── Frame Rate: Up to 10 Hz
├── Package: SMD, compact
└── Price: ~$40

Interface:
├── Protocol: I2C
├── Address: 0x68 or 0x69
├── Voltage: 3.3V
├── Current: 4.5mA (normal), 0.2mA (sleep)
└── Interrupt: Temperature threshold available
```

#### MLX90640 Specifications

```
Specifications:
├── Resolution: 32x24 pixels (768 total)
├── Temperature Range: -40°C - 300°C
├── Accuracy: ±1°C (0-100°C range)
├── Frame Rate: Up to 64 Hz
├── Field of View: 55° x 35° (BAA) or 110° x 75° (BAB)
├── Noise (NETD): 0.1K RMS @ 1Hz
└── Price: ~$60-80

Interface:
├── Protocol: I2C
├── Address: 0x33 (default)
├── Voltage: 3.3V
├── Current: 18mA (typical)
└── I2C Speed: Up to 1MHz
```

#### Key Parameters

```
Thermal Array Parameters:
├── Resolution (pixels)
├── Temperature Range (°C)
├── Accuracy (°C)
├── Field of View (degrees)
├── Frame Rate (Hz)
├── NETD (noise equivalent temp difference)
├── Pixel Pitch (mm)
└── Response Time (ms)

Output Data:
├── Temperature Array (°C per pixel)
├── Ambient Temperature (°C)
├── Maximum Temperature (°C)
├── Minimum Temperature (°C)
└── Thermal Image (interpolated)
```

---

### 10. GPS/GNSS Modules

**Use Cases:** Outdoor navigation, geofencing, tracking, waypoint following

GPS modules provide absolute position using satellite signals.

#### NEO-6M Specifications

```
Specifications:
├── Satellites: GPS (22 tracking, 50 channels)
├── Position Accuracy: 2.5m CEP
├── Update Rate: 5 Hz maximum
├── Sensitivity: -161 dBm (tracking)
├── Cold Start: ~27 seconds
├── Warm Start: ~1 second
├── Hot Start: <1 second
├── Antenna: Integrated ceramic patch
└── Note: End-of-Life, use NEO-M9N for new designs

Interface:
├── Protocol: UART (NMEA/UBX)
├── Default Baud: 9600
├── Voltage: 2.7V - 3.6V
├── Current: 45mA (acquisition), 11mA (power save)
├── EEPROM: Configuration storage
└── PPS: 1 pulse-per-second output
```

#### NEO-M9N (Recommended Replacement)

```
Specifications:
├── Satellites: GPS, GLONASS, Galileo, BeiDou
├── Concurrent Reception: 4 GNSS
├── Position Accuracy: 2.0m CEP
├── Update Rate: Up to 25 Hz
├── Sensitivity: -167 dBm (tracking)
├── Cold Start: ~24 seconds
└── Current: 25mA (continuous)
```

#### NMEA Protocol

```
Common NMEA Sentences:
├── $GPGGA: Position, fix quality, altitude
├── $GPRMC: Position, velocity, time
├── $GPVTG: Course and speed
├── $GPGSA: DOP and active satellites
└── $GPGSV: Satellites in view

Example GPRMC:
$GPRMC,123519,A,4807.038,N,01131.000,E,022.4,084.4,230394,003.1,W*6A

Fields:
├── Time: 12:35:19 UTC
├── Status: A=Active (valid)
├── Latitude: 48°07.038'N
├── Longitude: 11°31.000'E
├── Speed: 22.4 knots
├── Course: 84.4°
├── Date: 23/03/94
├── Magnetic Variation: 3.1°W
└── Checksum: *6A
```

#### Key Parameters

```
GPS Parameters:
├── Position Accuracy (m CEP)
├── Update Rate (Hz)
├── Time to First Fix (TTFF, seconds)
├── Channels (tracking/acquisition)
├── Supported Constellations
├── Sensitivity (dBm)
├── Power Consumption (mW)
└── Antenna Type (patch, helical, external)

Output Data:
├── Latitude (degrees)
├── Longitude (degrees)
├── Altitude (meters)
├── Speed (m/s or knots)
├── Course/Heading (degrees)
├── Fix Quality (none, GPS, DGPS, RTK)
├── Satellites Used
├── HDOP/PDOP (accuracy indicators)
└── UTC Time
```

---

### 11. Magnetometers / Compasses

**Use Cases:** Heading/bearing, north reference, magnetic field detection

Magnetometers measure magnetic field strength, typically used for compass heading.

#### HMC5883L / QMC5883L Specifications

```
Specifications (QMC5883L):
├── Axes: 3-axis magnetometer
├── Field Range: ±2, ±8 Gauss
├── Resolution: 16-bit ADC
├── Output Rate: Up to 200 Hz
├── Sensitivity: Variable with range
├── Heading Accuracy: 1-2°
├── Package: 3x3x0.9mm
└── Note: QMC5883L replaces discontinued HMC5883L

Interface:
├── Protocol: I2C
├── Address: 0x0D (QMC5883L), 0x1E (HMC5883L)
├── Voltage: 2.16V - 3.6V
├── Current: ~75µA
└── DRDY: Data ready pin available
```

#### Key Parameters

```
Magnetometer Parameters:
├── Field Range (Gauss or µT)
├── Resolution (bits)
├── Sensitivity (LSB/Gauss)
├── Noise (mGauss RMS)
├── Output Data Rate (Hz)
├── Heading Accuracy (degrees)
└── Temperature Coefficient

Calibration Requirements:
├── Hard Iron: Fixed magnetic offsets
├── Soft Iron: Field distortion
├── Procedure: Figure-8 rotation
└── Store: Min/max per axis, scale factors

Output Data:
├── Magnetic Field X, Y, Z (raw or µT)
├── Heading/Azimuth (degrees)
└── Temperature (some sensors)
```

---

### 12. Rotary Encoders

**Use Cases:** Motor position feedback, wheel odometry, user input, joint angles

Encoders convert rotational motion into digital signals for position and velocity measurement.

#### Categories

| Type | Reference | Output | Use Case |
|------|-----------|--------|----------|
| **Incremental** | Relative | Pulses (A, B, Z) | Speed, relative position |
| **Absolute** | Fixed | Digital code | Power-on position |
| **Optical** | Light/slots | High resolution | Precision motion |
| **Magnetic** | Hall effect | Moderate resolution | Harsh environments |

#### Incremental Encoder Specifications

```
Typical Optical Incremental Encoder:
├── Resolution: 100-10000 PPR (pulses per revolution)
├── Quadrature Output: A and B channels (90° phase)
├── Index Pulse: Z channel (once per revolution)
├── Count Modes:
│   ├── 1x: Count A rising edges
│   ├── 2x: Count A rising and falling
│   └── 4x: Count A and B all edges (4x PPR)
├── Maximum Frequency: 100kHz - 1MHz
└── Shaft Diameter: 6mm, 8mm typical

Output Types:
├── TTL (5V push-pull)
├── Open Collector (requires pull-up)
├── Line Driver (differential, RS-422)
└── Voltage: 5V, 3.3V variants available
```

#### AS5600 Magnetic Encoder

```
Specifications:
├── Technology: Contactless magnetic
├── Resolution: 12-bit (4096 positions/rev)
├── Interface: I2C, Analog, PWM output
├── Update Rate: ~150µs/sample
├── Accuracy: 0.0879° (12-bit)
├── Operating Voltage: 3.3V or 5V
├── Magnet: Diametrically magnetized (included)
└── Package: SOIC-8

Features:
├── No mechanical wear
├── Configurable via I2C
├── Programmable zero position
├── Burn-in configuration to OTP
└── Low power: 6.5mA typical
```

#### Key Parameters

```
Encoder Parameters:
├── Resolution (PPR or bits)
├── Maximum Speed (RPM)
├── Maximum Frequency (Hz)
├── Output Type (TTL, open collector, etc.)
├── Index Pulse (present/absent)
├── Shaft Diameter (mm)
├── Operating Voltage (V)
└── Operating Temperature Range

Quadrature Decoding:
├── Channel A: Position pulses
├── Channel B: 90° offset (direction)
├── Direction: A leads B = CW, B leads A = CCW
├── Z Index: Reference point each revolution
└── Count: 4x mode = 4 × PPR counts/rev
```

---

### 13. Force/Torque Sensors

**Use Cases:** Gripper force control, contact detection, weight measurement, collision sensing

Force sensors measure applied forces for precise manipulation and safety.

#### FSR (Force Sensing Resistor) Specifications

```
Typical FSR402:
├── Force Range: 0.2N - 20N
├── Sensitivity: Resistance decreases with force
├── Response Time: <10µs
├── Resistance (no force): >1MΩ
├── Resistance (10N): ~3kΩ
├── Repeatability: ±2%
├── Operating Temperature: -40°C to +85°C
├── Lifetime: >10 million actuations
└── Thickness: ~0.3mm

Interface:
├── Output: Analog resistance
├── Measurement: Voltage divider circuit
├── ADC: 10-bit minimum recommended
└── Conditioning: Op-amp for better linearity
```

#### Load Cell Specifications

```
Typical Strain Gauge Load Cell:
├── Capacity: 1kg, 5kg, 10kg, etc.
├── Output: 1-2 mV/V (requires amplifier)
├── Accuracy: 0.05% - 0.1%
├── Excitation: 5-10V DC
├── Bridge Resistance: 350Ω - 1kΩ
└── Overload: 150% safe, 200% max

Amplifier (HX711):
├── Gain: 128 or 64 (selectable)
├── ADC: 24-bit
├── Sample Rate: 10 or 80 SPS
├── Interface: Clock + Data (proprietary)
├── Voltage: 2.6V - 5.5V
└── Channels: 2 differential inputs
```

#### 6-Axis Force/Torque Sensors

```
Industrial F/T Sensor (e.g., ATI Mini45):
├── Axes: 6 (Fx, Fy, Fz, Tx, Ty, Tz)
├── Force Range: ±145N to ±580N
├── Torque Range: ±5Nm to ±20Nm
├── Resolution: <1/5000 of range
├── Overload: 2-8x rated load
├── Output: Analog (6 channels) or digital

Prosumer Options:
├── Robotiq FT 300
├── OnRobot HEX
├── FUTEK miniature series
└── DIY: Multiple load cells
```

#### Key Parameters

```
Force Sensor Parameters:
├── Range (N or kg)
├── Resolution (N or bits)
├── Accuracy/Linearity (%)
├── Repeatability (%)
├── Hysteresis (%)
├── Response Time (ms)
├── Overload Rating (%)
└── Temperature Sensitivity

Output Data:
├── Force (N or raw)
├── Tare/Zero offset
├── Calibrated weight (kg/g)
└── Multiple axes if applicable
```

---

### 14. Current Sensors

**Use Cases:** Motor current monitoring, battery management, overcurrent protection, stall detection

Current sensors measure electrical current for power monitoring and motor control.

#### ACS712 Specifications

```
Specifications:
├── Variants: 5A, 20A, 30A
├── Sensitivity: 185mV/A (5A), 100mV/A (20A), 66mV/A (30A)
├── Technology: Hall Effect
├── Isolation: 2.4kVRMS
├── Bandwidth: 80kHz
├── Output: Analog voltage (VCC/2 at 0A)
├── Supply: 5V
├── Accuracy: ±1.5%
└── Package: SOIC-8

Interface:
├── Output: Analog 0-5V
├── Zero Current: VCC/2 (2.5V)
├── Positive Current: 2.5V + (I × sensitivity)
├── Negative Current: 2.5V - (I × sensitivity)
├── ADC: 10-bit minimum
└── Bidirectional: Yes (AC or DC)
```

#### INA219 Specifications

```
Specifications:
├── Voltage Range: 0 - 26V (bus)
├── Shunt Voltage: ±320mV max
├── Current Range: ±3.2A (with 0.1Ω shunt)
├── Resolution: 12-bit ADC
├── Accuracy: ±1% (INA219A)
├── Measures: Voltage, Current, Power
└── Package: SOT-23 (8-pin)

Interface:
├── Protocol: I2C
├── Address: 0x40-0x4F (configurable)
├── Voltage: 3.0V - 5.5V
├── Current: 1mA typical
└── Alert: Programmable threshold
```

#### Key Parameters

```
Current Sensor Parameters:
├── Range (A)
├── Sensitivity (mV/A or LSB/A)
├── Accuracy (%)
├── Bandwidth (Hz)
├── Response Time (µs)
├── Isolation (V)
├── Operating Voltage (V)
└── Bidirectional (yes/no)

Output Data:
├── Current (A)
├── Voltage (V) - if supported
├── Power (W) - if calculated
└── Overcurrent Alert
```

---

## Communication Protocol Summary

### I2C (Inter-Integrated Circuit)

```
Characteristics:
├── Wires: 2 (SDA + SCL)
├── Speed: 100kHz (standard), 400kHz (fast), 1MHz+ (fast+)
├── Addressing: 7-bit (128 devices)
├── Topology: Multi-master, multi-slave bus
├── Distance: <1 meter typical
└── Pull-ups: Required (4.7kΩ typical)

Common Sensors:
├── IMUs (MPU6050, BNO055)
├── ToF (VL53L0X, VL53L1X)
├── Thermal (AMG8833, MLX90640)
├── Current (INA219)
├── Magnetometer (HMC5883L)
└── Encoders (AS5600)

Pros:
├── Simple wiring
├── Multiple devices on bus
├── Built into most MCUs
└── Bidirectional

Cons:
├── Short distance
├── Address conflicts possible
├── Clock stretching issues
└── Slower than SPI
```

### SPI (Serial Peripheral Interface)

```
Characteristics:
├── Wires: 4 minimum (MOSI, MISO, SCK, CS)
├── Speed: Up to 50MHz+
├── Addressing: Chip Select per device
├── Topology: Master-slave
├── Distance: <1 meter typical
└── Full Duplex: Yes

Common Sensors:
├── High-speed ADCs
├── Display controllers
├── SD cards
└── Some IMUs (optional)

Pros:
├── Very fast
├── Full duplex
├── Simple protocol
└── No addressing overhead

Cons:
├── Many wires
├── CS pin per device
├── Short distance
└── No error detection
```

### UART (Universal Asynchronous Receiver-Transmitter)

```
Characteristics:
├── Wires: 2 (TX, RX) or 1 (half-duplex)
├── Speed: 9600 - 1Mbps typical
├── Addressing: None (point-to-point) or packet-based
├── Distance: <15m at low baud rates
└── Flow Control: Optional (RTS/CTS)

Common Sensors:
├── GPS (NMEA)
├── LiDAR (RPLIDAR)
├── Smart servos (Dynamixel)
└── mmWave radar

Pros:
├── Simple point-to-point
├── Long distance possible
├── Built into most MCUs
└── Flexible protocols

Cons:
├── Point-to-point (usually)
├── Baud rate matching required
├── No built-in addressing
└── Half-duplex timing sensitive
```

### Analog

```
Characteristics:
├── Signal: 0-3.3V or 0-5V typical
├── Resolution: Depends on ADC (10-16 bit)
├── Speed: Limited by ADC sample rate
├── Distance: Short (noise sensitive)
└── Conditioning: Often required

Common Sensors:
├── Ultrasonic (Echo pulse timing)
├── Potentiometers
├── FSR force sensors
├── Current sensors (ACS712)
├── Reflectance sensors (QTR-A)
└── Temperature sensors (analog)

Pros:
├── Simple interface
├── Low latency
├── Direct reading
└── No protocol overhead

Cons:
├── Noise sensitive
├── Single channel per wire
├── Limited resolution
└── May need conditioning
```

### GPIO (General Purpose I/O)

```
Digital Input:
├── Ultrasonic trigger/echo
├── PIR motion detection
├── Encoder pulses
├── Interrupt-driven events
└── RC timing (QTR-RC)

PWM Input:
├── RC receivers
├── Some distance sensors
└── Encoder frequency output

Timing-Critical:
├── Ultrasonic echo measurement
├── RC pulse width
├── Encoder counting
└── Requires hardware timers/interrupts
```

---

## Electrical Interface Summary

| Sensor Type | Typical Voltage | Interface | Current |
|-------------|-----------------|-----------|---------|
| Depth Camera | 5V USB | USB 3.0 | 500mA-2A |
| LiDAR | 5V | UART | 300-500mA |
| Ultrasonic | 5V | GPIO | 15mA |
| ToF | 3.3V | I2C | 20-40mA |
| IMU | 3.3V | I2C/SPI | 5-15mA |
| PIR | 5V | GPIO | 50µA |
| mmWave | 3.3-5V | UART/I2C | 100mA |
| Thermal Array | 3.3V | I2C | 5-20mA |
| GPS | 3.3V | UART | 30-50mA |
| Magnetometer | 3.3V | I2C | 100µA |
| Encoder | 3.3-5V | GPIO/I2C | 5-20mA |
| Force Sensor | 5V | Analog | <1mA |
| Current Sensor | 5V | Analog/I2C | 10mA |

---

## Gorai Integration Considerations

### Sensor Interface Hierarchy

```go
// Base Sensor interface
type Sensor interface {
    Name() string
    Type() SensorType
    Initialize() error
    Close() error
    IsReady() bool
}

// Distance sensors (LiDAR, Ultrasonic, ToF)
type DistanceSensor interface {
    Sensor
    GetDistance() (float64, error)  // meters
    GetRange() (min, max float64)
}

// Scanning distance sensors
type ScanningDistanceSensor interface {
    DistanceSensor
    GetScan() ([]ScanPoint, error)  // angle, distance, intensity
    GetScanRate() float64           // Hz
}

// Vision sensors
type VisionSensor interface {
    Sensor
    GetFrame() (Frame, error)
    GetResolution() (width, height int)
    GetFrameRate() float64
}

// Depth sensors
type DepthSensor interface {
    VisionSensor
    GetDepthFrame() (DepthFrame, error)
    GetPointCloud() (PointCloud, error)
}

// IMU sensors
type IMUSensor interface {
    Sensor
    GetAcceleration() (x, y, z float64, error)  // m/s²
    GetAngularVelocity() (x, y, z float64, error)  // rad/s
}

// AHRS (with fusion)
type AHRSSensor interface {
    IMUSensor
    GetOrientation() (Quaternion, error)
    GetEulerAngles() (roll, pitch, yaw float64, error)
}

// Position sensors
type PositionSensor interface {
    Sensor
    GetPosition() (lat, lon, alt float64, error)
    GetVelocity() (float64, error)  // m/s
    GetHeading() (float64, error)   // degrees
}

// Presence sensors
type PresenceSensor interface {
    Sensor
    IsPresenceDetected() bool
    GetDistance() (float64, error)  // if available
}

// Force sensors
type ForceSensor interface {
    Sensor
    GetForce() (float64, error)  // N
    Tare() error
}

// Encoder sensors
type EncoderSensor interface {
    Sensor
    GetPosition() (int64, error)     // counts
    GetVelocity() (float64, error)   // counts/sec
    Reset() error
}
```

### Link Types for Sensors

| Sensor Category | Primary Link | Alternative |
|-----------------|-------------|-------------|
| Depth Camera | USB | MIPI CSI, GMSL |
| LiDAR | Serial | USB |
| Ultrasonic | GPIO | - |
| ToF | I2C | - |
| IMU | I2C | SPI |
| PIR | GPIO | - |
| mmWave | UART | I2C |
| Thermal Array | I2C | - |
| GPS | Serial | I2C (some) |
| Magnetometer | I2C | SPI |
| Encoder | GPIO | I2C, SPI |
| Force/Load Cell | Analog (ADC) | I2C (HX711) |
| Current Sensor | Analog | I2C (INA219) |

### Configuration Example

```json
{
  "sensors": [
    {
      "name": "front_lidar",
      "type": "lidar_2d",
      "driver": "rplidar",
      "link": "serial_0",
      "config": {
        "port": "/dev/ttyUSB0",
        "baud_rate": 115200,
        "scan_mode": "standard",
        "frame_id": "lidar_link"
      }
    },
    {
      "name": "depth_camera",
      "type": "depth_camera",
      "driver": "realsense",
      "link": "usb",
      "config": {
        "serial_number": "123456789",
        "depth_width": 640,
        "depth_height": 480,
        "depth_fps": 30,
        "enable_rgb": true
      }
    },
    {
      "name": "imu",
      "type": "imu_9dof",
      "driver": "bno055",
      "link": "i2c_1",
      "config": {
        "address": "0x28",
        "mode": "ndof",
        "frame_id": "imu_link"
      }
    },
    {
      "name": "front_ultrasonic",
      "type": "ultrasonic",
      "driver": "hcsr04",
      "link": "gpio",
      "config": {
        "trigger_pin": 23,
        "echo_pin": 24,
        "max_range": 4.0
      }
    },
    {
      "name": "gps",
      "type": "gnss",
      "driver": "nmea",
      "link": "serial_1",
      "config": {
        "port": "/dev/ttyACM0",
        "baud_rate": 9600
      }
    },
    {
      "name": "wheel_encoder_left",
      "type": "encoder",
      "driver": "quadrature",
      "link": "gpio",
      "config": {
        "pin_a": 17,
        "pin_b": 27,
        "ppr": 1000,
        "invert": false
      }
    }
  ]
}
```

---

## Summary Tables

### Distance Sensing Comparison

| Sensor | Range | Accuracy | FOV | Update | Interface | Cost |
|--------|-------|----------|-----|--------|-----------|------|
| Ultrasonic (HC-SR04) | 2cm-4m | ±3mm | 30° | 50Hz | GPIO | $2 |
| ToF (VL53L0X) | 3cm-1.2m | ±3% | 25° | 50Hz | I2C | $8 |
| ToF (VL53L1X) | 3cm-4m | ±3% | 27° | 50Hz | I2C | $12 |
| 2D LiDAR (RPLIDAR A1) | 15cm-12m | ±<1% | 360° | 5.5Hz | UART | $100 |
| 2D LiDAR (RPLIDAR A3) | 20cm-25m | ±<1% | 360° | 15Hz | UART | $300 |

### IMU Comparison

| Sensor | DOF | Fusion | Accuracy | Interface | Cost |
|--------|-----|--------|----------|-----------|------|
| MPU6050 | 6 | DMP | Moderate | I2C | $2 |
| MPU9250 | 9 | DMP | Moderate | I2C/SPI | $5 |
| BNO055 | 9 | Internal | High | I2C | $20 |
| ICM-20948 | 9 | DMP | High | I2C/SPI | $10 |

### Presence Detection Comparison

| Sensor | Static | Range | Through Wall | Interface | Cost |
|--------|--------|-------|--------------|-----------|------|
| PIR (HC-SR501) | No | 7m | No | GPIO | $2 |
| mmWave (24GHz) | Yes | 8-12m | Partial | UART | $20 |
| Thermal (AMG8833) | Yes | 7m | No | I2C | $40 |

---

## Sources

- [Stereolabs ZED Documentation](https://www.stereolabs.com/docs/)
- [Intel RealSense Documentation](https://dev.intelrealsense.com/docs)
- [Luxonis DepthAI Documentation](https://docs.luxonis.com/)
- [SLAMTEC RPLIDAR Documentation](https://www.slamtec.com/en/Support)
- [STMicroelectronics VL53L0X/VL53L1X Datasheets](https://www.st.com/en/imaging-and-photonics-solutions.html)
- [InvenSense MPU6050 Datasheet](https://invensense.tdk.com/products/motion-tracking/6-axis/mpu-6050/)
- [Bosch BNO055 Datasheet](https://www.bosch-sensortec.com/products/smart-sensors/bno055/)
- [Pololu QTR Sensor Documentation](https://www.pololu.com/category/123/pololu-qtr-reflectance-sensors)
- [u-blox NEO-6M Documentation](https://www.u-blox.com/en/product/neo-6-series)
- [Adafruit Learning System](https://learn.adafruit.com/)
- [DFRobot Wiki](https://wiki.dfrobot.com/)
- [Texas Instruments mmWave Sensors](https://www.ti.com/sensors/mmwave-radar/overview.html)
