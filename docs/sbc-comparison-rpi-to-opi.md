# SBC Comparison: Raspberry Pi 5 vs Orange Pi 5 for Robotics

This document evaluates the Orange Pi 5 series against the Raspberry Pi 5 with AI accelerators for use with Gorai robotics projects.

## Executive Summary

**Recommendation: Raspberry Pi 5 + Hailo AI Kit**

While the Orange Pi 5 series offers attractive pricing with an integrated 6 TOPS NPU, the Raspberry Pi 5 ecosystem provides superior compatibility with robotics kits, better AI accelerator options, and a significantly larger support community. The Orange Pi's GPIO and mounting hole incompatibilities make it unsuitable as a drop-in replacement for Raspberry Pi-based robotics platforms.

## Price Comparison

| Configuration | Price (USD) | NPU Performance | RAM Options |
|---------------|-------------|-----------------|-------------|
| Orange Pi 5 (standard) | $60-138 | 6 TOPS (built-in) | 4/8/16GB |
| Orange Pi 5B | $90-150 | 6 TOPS (built-in) | 4/8/16GB + eMMC |
| Orange Pi 5 Pro | $60-109 | 6 TOPS (built-in) | 4/8/16GB |
| Raspberry Pi 5 | $60-80 | None | 4/8GB |
| RPi 5 + Hailo AI Kit | ~$150 | 13 TOPS | 4/8GB |
| RPi 5 + AI HAT+ (26 TOPS) | ~$190 | 26 TOPS | 4/8GB |

The Orange Pi appears more cost-effective for integrated NPU functionality, but the Raspberry Pi 5 with Hailo offers 2x or more AI performance at a modest premium.

## Hardware Compatibility

### GPIO Header

This is a critical differentiator:

| Board | GPIO Pins | RPi HAT Compatible |
|-------|-----------|-------------------|
| Orange Pi 5 | 26-pin | No |
| Orange Pi 5B | 26-pin | No |
| Orange Pi 5 Pro | 40-pin | Partial |
| Orange Pi 5 Plus | 40-pin | Partial |
| Raspberry Pi 5 | 40-pin | Yes |

The Orange Pi 5 and 5B use a **26-pin GPIO header**, not the standard 40-pin layout. This is incompatible with:

- SunFounder PiCar-X and similar robot kits
- Most Raspberry Pi HATs
- Standard robotics expansion boards

Only the Orange Pi 5 Pro and 5 Plus have the full 40-pin header, but even these have functional differences.

### GPIO Functional Differences

Testing has revealed that Orange Pi 5 boards have issues with:

- **Internal pull-up/pull-down resistors**: Do not function correctly via libgpiod
- **Pin mapping**: Different from Raspberry Pi despite similar physical layout
- **I2S audio interfaces**: Not clearly exposed on GPIO header

These differences require software modifications and may cause compatibility issues with sensors and peripherals designed for Raspberry Pi.

### Mounting Holes

Orange Pi boards **do not use identical mounting hole placement** to Raspberry Pi. This affects:

- Robot chassis compatibility
- Case/enclosure fit
- HAT mechanical alignment
- Off-the-shelf robotics kits

Projects using existing Raspberry Pi-compatible chassis will require modifications or custom mounting solutions.

### Robotics Kit Compatibility

| Kit | Raspberry Pi 5 | Orange Pi 5/5B | Orange Pi 5 Pro |
|-----|----------------|----------------|-----------------|
| SunFounder PiCar-X | Yes | No | Requires modification |
| SunFounder PiCar-S | Yes | No | Requires modification |
| Generic RPi robot HATs | Yes | No | Partial |
| Custom builds | Yes | Yes | Yes |

## AI/NPU Capabilities

### Rockchip RK3588S NPU (Orange Pi 5 Series)

**Specifications:**
- 6 TOPS computing power
- Supports INT4/INT8/INT16/FP16 hybrid computing
- Integrated into SoC (no additional hardware needed)

**Software Stack:**
- RKNN-Toolkit2 for model conversion
- Workflow: PyTorch/TensorFlow → ONNX → RKNN → NPU
- Active development with YOLO support via Ultralytics
- RKNN-LLM for small language models (TinyLlama, Qwen-1.8B)

**Considerations:**
- Proprietary toolchain with smaller community
- Models must be converted (cannot run TensorFlow/PyTorch directly)
- Documentation primarily in Chinese with English translations
- Fewer pre-converted models available

### Google Coral TPU (Not Recommended)

**Status: Effectively Abandoned**

Google has ceased active development of the Coral platform:

- Last official update: May 2022
- GASKET driver incompatible with Linux kernel ≥6.4
- PyCoral library only supports Python 3.9
- No response to community issues or pull requests
- Driver removed from Linux kernel staging

Community workarounds exist but long-term viability is uncertain. **We do not recommend Coral TPU for new projects.**

### Hailo AI Accelerators (Raspberry Pi)

**Raspberry Pi AI Kit (Hailo-8L):**
- 13 TOPS performance
- $70 add-on price
- Official Raspberry Pi support
- Excellent power efficiency (3-4 TOPS/W, ~1-2W typical)

**Raspberry Pi AI HAT+ (Hailo-8):**
- 26 TOPS performance
- $110 add-on price
- Maximum edge AI capability

**Software Stack:**
- Native support for TensorFlow, TensorFlow Lite, ONNX, PyTorch
- Well-documented APIs
- Active development with regular updates
- Large model zoo with pre-optimized networks

## Software Ecosystem

| Factor | Orange Pi 5 | Raspberry Pi 5 |
|--------|-------------|----------------|
| OS Options | Orange Pi OS, Ubuntu, Debian, Android | Raspberry Pi OS, Ubuntu, many others |
| Community Size | Small | Very large |
| Documentation | Limited, often Chinese-first | Extensive, English-primary |
| Troubleshooting Resources | Sparse | Abundant |
| Third-party Libraries | Limited | Extensive |
| Robot Framework Support | Minimal | Strong (ROS2, etc.) |

### Key Software Differences

1. **GPIO Libraries**: Orange Pi requires OPi.GPIO or WiringOP instead of RPi.GPIO
2. **Camera Support**: Different camera interfaces and drivers
3. **PWM/I2C/SPI**: May require different configuration
4. **Package Availability**: Some ARM packages not tested on Orange Pi

## Performance Comparison

### CPU Performance

The Orange Pi 5 series uses the Rockchip RK3588S with:
- 4x Cortex-A76 @ 2.4GHz
- 4x Cortex-A55 @ 1.8GHz
- Mali-G610 GPU

The Raspberry Pi 5 uses the Broadcom BCM2712 with:
- 4x Cortex-A76 @ 2.4GHz
- VideoCore VII GPU

The Orange Pi has more CPU cores and generally better multi-threaded performance, while single-threaded performance is comparable.

### AI Inference Performance

| Platform | Model | Performance |
|----------|-------|-------------|
| Orange Pi 5 (RK3588S NPU) | ResNet18 | ~244 FPS |
| Orange Pi 5 (RK3588S NPU) | YOLOv5s | ~30-50 FPS |
| RPi 5 + Hailo-8L (13 TOPS) | YOLOv5s | ~60-80 FPS |
| RPi 5 + Hailo-8 (26 TOPS) | YOLOv5s | ~100+ FPS |

The Hailo accelerators significantly outperform the integrated RK3588S NPU for most vision tasks.

## Recommendations by Use Case

### Recommended: Raspberry Pi 5 + Hailo AI Kit (~$150)

Best for:
- Projects using existing robotics kits (SunFounder, etc.)
- Teams wanting maximum community support
- Production deployments requiring long-term stability
- Computer vision applications requiring high inference rates

### Budget Option: Raspberry Pi 5 Alone (~$80)

Best for:
- Learning and prototyping
- Simple sensor integration
- Projects with minimal AI requirements
- CPU-based inference for lighter models

### Power User: RPi 5 + AI HAT+ 26 TOPS (~$190)

Best for:
- Complex multi-model inference pipelines
- Real-time video analytics
- Edge LLM experimentation
- Maximum AI performance requirements

### Alternative: Orange Pi 5 Pro (~$109)

Consider only when:
- Budget is the primary constraint
- More than 8GB RAM is required (up to 32GB available)
- Building custom hardware (not using existing kits)
- Team has experience with Rockchip platforms
- Willing to adapt software and handle compatibility issues

**Note**: Requires Orange Pi 5 Pro or 5 Plus for 40-pin GPIO compatibility.

## Sample Build: Raspberry Pi 5 AI Robotics Platform

The following is a recommended parts list for building a Raspberry Pi 5-based AI robotics platform with NVMe storage and 26 TOPS AI acceleration.

> **Note**: These Amazon links are provided for convenience only. These components may be available for less through other vendors such as Adafruit, SparkFun, The Pi Hut, or direct from manufacturers.

| Component | Description | Est. Price | Amazon Link |
|-----------|-------------|------------|-------------|
| Raspberry Pi 5 (16GB) | Main SBC with 16GB RAM | ~$145 | [B0DSPYPKRG](https://www.amazon.com/dp/B0DSPYPKRG) |
| Waveshare Hailo-8 M.2 Module | 26 TOPS AI Accelerator (module only) | ~$99 | [B0D928WG5L](https://www.amazon.com/dp/B0D928WG5L) |
| GeeekPi Armor Lite V5 Cooler | Active cooler with aluminum heatsink | ~$15 | [B0CNVFCWQR](https://www.amazon.com/dp/B0CNVFCWQR) |
| SunFounder Dual NVMe Raft | PCIe to dual M.2 HAT for RPi 5 | ~$40 | [B0FC27KH3X](https://www.amazon.com/dp/B0FC27KH3X) |
| WD Black SN7100 500GB | M.2 2230 NVMe SSD | ~$60 | [B0DN7JK8T4](https://www.amazon.com/dp/B0DN7JK8T4) |

### Estimated Total Cost

| Configuration | Components | Est. Total |
|---------------|------------|------------|
| **Full AI Build** | All above | **~$359** |
| **Without NVMe storage** | Pi 5 + Hailo-8 + Cooler | ~$259 |
| **Basic (no AI accelerator)** | Pi 5 + Cooler + NVMe | ~$260 |

### Notes on Component Selection

1. **Raspberry Pi 5 16GB**: The 16GB variant provides headroom for AI workloads and future expansion. The 8GB variant ($80) is sufficient for most robotics applications.

2. **Waveshare Hailo-8**: The module-only version requires a compatible M.2 HAT. The SunFounder Dual NVMe Raft supports both NVMe storage and Hailo-8L in its dual M.2 slots.

3. **Active Cooling**: Essential for sustained AI inference workloads. The Armor Lite V5 provides excellent thermal performance in a compact form factor.

4. **NVMe Storage**: The WD Black SN7100 in 2230 form factor provides fast storage while maintaining compatibility with compact robotics builds. A 2280 form factor drive can also be used with the Dual NVMe Raft.

5. **Power Supply**: Not included above. Use the official Raspberry Pi 27W USB-C power supply (~$12) for reliable operation with AI accelerator and NVMe storage.

## Battery System Design

For mobile robotics applications, battery power is essential. This section analyzes power requirements and recommends battery solutions for the Raspberry Pi 5 AI robotics platform.

### Power Requirements Analysis

| Component | Typical Power | Peak Power |
|-----------|---------------|------------|
| Raspberry Pi 5 (16GB) | 8W | 12W (under load) |
| Hailo-8 M.2 Module | 2.5W | 4W (active inference) |
| NVMe SSD (WD Black SN7100) | 1W | 3W (sustained write) |
| Active Cooler Fan | 0.5W | 0.5W |
| **Total** | **12W** | **19.5W** |

**Energy Budget for 1 Hour:**
- Typical usage: 12Wh
- Heavy AI workload: ~15-16Wh
- With safety margin (20%): ~19Wh capacity needed

### Battery Options

> **Note**: These Amazon links are provided for convenience only. These components may be available for less through other vendors.

#### Option 1: USB-C PD Power Bank (Simplest)

A high-quality USB-C Power Delivery power bank that supports 20V/3A output provides the simplest solution for portable power.

| Component | Description | Est. Price | Amazon Link |
|-----------|-------------|------------|-------------|
| Anker 737 Power Bank (24,000mAh) | 140W USB-C PD, 24Ah @ 3.7V = ~89Wh | ~$110 | [B09VPHVT2Z](https://www.amazon.com/dp/B09VPHVT2Z) |

**Runtime estimate**: 89Wh ÷ 15W = ~5-6 hours typical use

This exceeds the 1-hour minimum requirement significantly, providing excellent margin for extended development sessions.

#### Option 2: Compact PD Power Bank (Budget)

| Component | Description | Est. Price | Amazon Link |
|-----------|-------------|------------|-------------|
| Anker 325 Power Bank (20,000mAh) | 65W USB-C PD, ~74Wh | ~$50 | [B0BYP5LZCX](https://www.amazon.com/dp/B0BYP5LZCX) |

**Runtime estimate**: 74Wh ÷ 15W = ~4-5 hours

#### Option 3: Pi-Specific UPS HAT (Best for Robotics Integration)

For cleaner integration and proper shutdown handling, a UPS HAT with replaceable cells is ideal for production robotics.

| Component | Description | Est. Price | Amazon Link |
|-----------|-------------|------------|-------------|
| Waveshare UPS HAT (D) | 5V/5A output, accepts 4× 18650 cells | ~$25 | [B0D5ZXJB9W](https://www.amazon.com/dp/B0D5ZXJB9W) |
| Samsung 35E 18650 Cells (4×) | 3500mAh per cell, high quality | ~$28 | [B0BN25JJ7P](https://www.amazon.com/dp/B0BN25JJ7P) |

**Capacity**: 4 × 3.5Ah × 3.7V = ~52Wh
**Runtime estimate**: 52Wh × 0.85 (efficiency) ÷ 15W = ~3 hours

**Advantages:**
- Integrates with GPIO for battery monitoring
- Supports graceful shutdown on low battery
- Replaceable/swappable cells for field use
- Lower center of gravity for mobile robots

### Power Delivery Requirements

The Raspberry Pi 5 requires **5V/5A (25W)** for stable operation under load. Ensure your solution provides:

- USB-C PD with 5V/5A profile, OR
- A HAT/board rated for 5A continuous output

### Hailo-8 Power Note

The Hailo-8 draws power from the PCIe slot. The SunFounder Dual NVMe Raft passes through PCIe power from the Pi's main supply—no separate power connection is needed. Ensure your total power budget accounts for the AI accelerator's consumption.

### UPS HAT Wiring Diagram

```
┌─────────────────┐
│  4× 18650 Cells │
│  (14.8V nom)    │
└────────┬────────┘
         │
         ▼
┌─────────────────┐
│ Waveshare UPS   │
│ HAT (D)         │
│ 5V/5A regulated │
└────────┬────────┘
         │ GPIO Header
         ▼
┌─────────────────┐
│ Raspberry Pi 5  │◄── Powers NVMe HAT + Hailo-8
│                 │    via PCIe
└─────────────────┘
```

### Battery Recommendation Summary

| Use Case | Recommended Solution | Est. Cost | Runtime |
|----------|---------------------|-----------|---------|
| Development/prototyping | Anker 325 Power Bank | ~$50 | 4-5 hours |
| Extended sessions | Anker 737 Power Bank | ~$110 | 5-6 hours |
| Production robot | UPS HAT + 18650 cells | ~$53 | ~3 hours |

**For development and prototyping**: Option 2 (Anker 325) provides a simple, reliable solution with plenty of runtime.

**For a finished robot**: Option 3 (UPS HAT + 18650s) offers proper integration, battery monitoring via GPIO, graceful shutdown support, and swappable cells for field use.

## Migration Considerations

If migrating from Raspberry Pi to Orange Pi:

1. **GPIO code** will require rewriting for OPi.GPIO or WiringOP
2. **Camera interfaces** are different and require driver changes
3. **HATs and shields** will not physically fit (5/5B) or may not function correctly
4. **I2C/SPI device addresses** may need reconfiguration
5. **PWM pins** are mapped differently
6. **Power requirements** may differ

## Conclusion

For Gorai robotics projects, we recommend the **Raspberry Pi 5 with Hailo AI Kit** as the primary platform. The combination offers:

- Full compatibility with robotics kits and HATs
- Superior AI inference performance (13-26 TOPS vs 6 TOPS)
- Extensive documentation and community support
- Long-term software support from Raspberry Pi Foundation
- Proven ecosystem for robotics applications

The Orange Pi 5 series remains a viable option for custom builds where cost is critical and compatibility requirements are minimal, but it should not be considered a drop-in replacement for Raspberry Pi in robotics applications.

## References

- [Orange Pi 5 Official Specifications](http://www.orangepi.org/html/hardWare/computerAndMicrocontrollers/details/Orange-Pi-5.html)
- [Raspberry Pi 5 vs Orange Pi 5 Pro Comparison](https://www.xda-developers.com/raspberry-pi-5-vs-orange-pi-5-pro/)
- [Google Coral TPU Project Status](https://github.com/blakeblackshear/frigate/discussions/18564)
- [Raspberry Pi AI Kit](https://www.raspberrypi.com/products/ai-kit/)
- [Raspberry Pi AI HAT+](https://www.raspberrypi.com/news/raspberry-pi-ai-hat/)
- [Hailo AI Kit Testing by Jeff Geerling](https://www.jeffgeerling.com/blog/2024/testing-raspberry-pis-ai-kit-13-tops-70)
- [RKNN-Toolkit2 GitHub](https://github.com/airockchip/rknn-toolkit2)
- [Rockchip RKNN Integration with Ultralytics](https://docs.ultralytics.com/integrations/rockchip-rknn/)
- [SunFounder PiCar-X Documentation](https://docs.sunfounder.com/projects/picar-x/en/latest/)
- [Orange Pi 5 GPIO Testing](https://www.learningtopi.com/templates/orangepi-5-testing-with-sbc_gpio/)
- [Coral TPU on Raspberry Pi 5](https://www.jeffgeerling.com/blog/2023/pcie-coral-tpu-finally-works-on-raspberry-pi-5)
