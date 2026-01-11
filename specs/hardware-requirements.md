# Hardware Requirements Specification

**Version:** 2.0
**Status:** Active
**Last Updated:** 2024-12-25

## 1. Overview

Gorai requires capable hardware to support containerized robotics workloads and edge AI. This document specifies minimum and recommended hardware configurations.

### 1.1 Design Philosophy

**"Consistent deployment from edge to fleet."**

Gorai uses K3s (Lightweight Kubernetes), providing:
- Container isolation and reproducibility
- Consistent deployment model (~512 MB overhead)
- Fleet-ready from day one (single-node or multi-node)
- Production-grade orchestration without complexity

### 1.2 Platform Strategy

Gorai supports three platform tiers:

| Tier | Platform | AI Performance | Best For |
|------|----------|----------------|----------|
| **Primary** | Raspberry Pi 5 | External (Hailo, Coral) | Best ecosystem, beginners |
| **Performance** | Jetson Orin Nano Super | 67 TOPS (CUDA) | Multi-model AI, VLMs |
| **Budget AI** | Orange Pi 5B | 6 TOPS (built-in) | Lower cost AI builds |

All platforms use identical K3s deployment. Choose based on AI requirements and budget.

**See [K3s Installation Guide](k3s-installation.md) for platform-specific installation instructions.**

---

## 2. Minimum Requirements

### 2.1 Compute Platform

| Requirement | Specification |
|-------------|---------------|
| **Processor** | ARM64 (Cortex-A72 or better) or x86_64 |
| **RAM** | 4 GB minimum |
| **Cores** | 4 cores minimum |
| **Reference** | Raspberry Pi 4 Model B (4GB) |

### 2.2 Storage

| Requirement | Specification |
|-------------|---------------|
| **Type** | SSD recommended, SD card acceptable |
| **Interface** | USB 3.0, NVMe, eMMC, or SD card |
| **Capacity** | 32 GB minimum (64 GB recommended) |

**Storage recommendations by use case:**

| Use Case | Storage Type | Notes |
|----------|--------------|-------|
| Development | SSD or NVMe | Fast image pulls, frequent updates |
| Deployed robot | SSD or NVMe | Required for K3s SQLite database |
| AI workloads | SSD or NVMe | Model loading performance |
| Orange Pi 5B | Built-in eMMC | Built-in eMMC sufficient for K3s |

**CRITICAL:** K3s uses SQLite for the control plane database, which requires sustained random I/O. SD cards provide only 10-30 IOPS and will cause database corruption and instability. **SSD, NVMe, or eMMC required.**

### 2.3 Network

| Requirement | Specification |
|-------------|---------------|
| **Type** | Ethernet recommended |
| **Speed** | 100 Mbps minimum |
| **WiFi** | Acceptable for development/testing |

### 2.4 Power

| Platform | Power Supply |
|----------|--------------|
| Raspberry Pi 4 | 5V/3A USB-C, quality supply |
| Raspberry Pi 5 | 5V/5A USB-C, PD compatible |
| Orange Pi 5B | 5V/4A USB-C, quality supply |
| Jetson Orin | Per NVIDIA specifications |

**Undervoltage causes instability.** Use official or quality third-party power supplies.

---

## 3. Recommended Configurations

### 3.1 Primary Platform: Raspberry Pi 5 (8GB)

The Raspberry Pi 5 is Gorai's **primary reference platform** with the best documentation, largest community, and most beginner-friendly experience.

| Component | Specification | Est. Cost |
|-----------|---------------|-----------|
| Raspberry Pi 5 (8GB) | 4-core Cortex-A76, 8GB LPDDR4X | $80 |
| NVMe SSD (256GB) | PCIe Gen 2 x1, via HAT | $30 |
| NVMe HAT/Base | Pimoroni, Geekworm, etc. | $15-25 |
| Power Supply | 5V/5A USB-C PD | $15 |
| Heatsink/Case | Active cooling recommended | $10-20 |
| **Total** | | **$150-170** |

**For AI workloads**, add an external accelerator:
- Hailo-8L M.2 (+$70) → 13 TOPS
- Google Coral USB (+$60) → 4 TOPS

### 3.2 Alternative Platform: Orange Pi 5B (8GB)

The Orange Pi 5B is an **alternative platform** with a **built-in 6 TOPS NPU**, eliminating the need for an external AI accelerator. Recommended for users comfortable with Linux and seeking lower-cost AI capability.

| Component | Specification | Est. Cost |
|-----------|---------------|-----------|
| Orange Pi 5B (8GB/64GB) | 8-core RK3588S, 6 TOPS NPU, 64GB eMMC | $115 |
| Power Supply | 5V/4A USB-C | $15 |
| Heatsink/Fan | **Required** (board runs hot) | $10-15 |
| **Total** | | **$140-145** |

**Advantages:**
- Built-in 6 TOPS NPU (no add-on needed)
- Built-in 64GB eMMC (no SSD needed)
- Built-in WiFi 6 + Bluetooth 5.3
- 8 CPU cores (vs Pi 5's 4 cores)
- ~$50-90 cheaper than Pi 5 + Hailo for AI workloads

**Trade-offs:**
- Smaller community, less documentation
- Requires Armbian OS (official images less stable)
- NPU model conversion requires x86 Linux host
- More Linux experience required

### 3.3 Performance Platform: Jetson Orin Nano Super

The Jetson Orin Nano Super is Gorai's **performance platform** for AI-intensive workloads requiring multiple concurrent models, VLMs, or maximum inference throughput.

| Component | Specification | Est. Cost |
|-----------|---------------|-----------|
| Jetson Orin Nano Super | 6-core A78AE, 1024 CUDA cores, 67 TOPS | $249 |
| NVMe SSD (256GB) | M.2 2280 | $30 |
| WiFi Module | Intel AC8265 (M.2 Key E) | $20 |
| Power Supply | 5V/5A USB-C | $15 |
| Active Cooling | Fan + heatsink (required) | $20 |
| **Total** | | **~$335** |

**Advantages:**
- 67 TOPS AI performance (5x more than Hailo-8L)
- Full CUDA ecosystem (PyTorch, TensorFlow native)
- Run multiple AI models simultaneously
- 7ms inference latency for real-time control
- Can run on-robot LLMs/VLMs (LLaVA-7B, etc.)

**Trade-offs:**
- Higher power consumption (15-25W vs 5W for Pi)
- No built-in WiFi/storage (add ~$50)
- Requires active cooling for sustained loads
- NVIDIA ecosystem lock-in (JetPack OS only)
- Currently supply-constrained (check availability)

**Best for:**
- Multi-modal perception (vision + lidar + radar fusion)
- VLM-based navigation and interaction
- SLAM + path planning + vision in parallel
- Research requiring CUDA/TensorRT

### 3.4 Resource Budget (8GB Platform)

```
Total RAM:        8,192 MB
├── Linux OS:       300 MB
├── K3s:            512 MB
├── NATS:           512 MB
└── Available:    6,868 MB  ← For robot workloads

Total CPU:        4 cores @ 2.4 GHz
├── K3s:          ~3% overhead (idle)
└── Available:    ~97% (~3.9 cores)  ← For robot workloads
```

**K3s overhead is acceptable for edge AI:**
| Resource | K3s Overhead | Available (8GB) | Notes |
|----------|--------------|-----------------|-------|
| RAM | ~512 MB | ~6.9 GB | AI workloads need 4GB+ anyway |
| CPU | ~3% idle | ~97% | Minimal impact on inference |
| Disk | ~1.5 GB | Varies | SQLite database + images |

### 3.5 Performance Expectations

| Workload | Jetson Orin Super | Pi 5 + Hailo-8L | Orange Pi 5B (NPU) | Pi 5 (CPU) |
|----------|-------------------|-----------------|--------------------|--------------------|
| YOLO inference (640x480) | 60+ FPS | 30+ FPS | 25-30 FPS | 2-3 FPS |
| MobileNet classification | 200+ FPS | 100+ FPS | 50+ FPS | 10-15 FPS |
| Multiple models concurrent | ✅ Yes | ⚠️ Limited | ⚠️ Limited | ❌ No |
| On-robot LLM/VLM | ✅ Yes | ❌ No | ❌ No | ❌ No |
| Camera streaming | 30 FPS, 1080p | 30 FPS, 1080p | 30 FPS, 1080p | 30 FPS, 1080p |
| NATS messaging | 10,000+ msg/sec | 10,000+ msg/sec | 10,000+ msg/sec | 10,000+ msg/sec |
| Power consumption | 15-25W | ~5W | ~8W | ~5W |

---

## 4. Supported Platforms

### 4.1 Primary Platform (Reference)

| Platform | RAM | AI Acceleration | Storage | Support Level |
|----------|-----|-----------------|---------|---------------|
| **Raspberry Pi 5 (8GB)** | 8 GB | External (Hailo, Coral) | NVMe HAT | ★★★ Primary |
| **Raspberry Pi 5 (4GB)** | 4 GB | External (Hailo, Coral) | NVMe HAT | ★★★ |

### 4.2 Performance Platform (Maximum AI)

| Platform | RAM | AI Acceleration | Storage | Support Level |
|----------|-----|-----------------|---------|---------------|
| **Jetson Orin Nano Super** | 8 GB | 67 TOPS (CUDA) | NVMe required | ★★★ Performance |

### 4.3 Budget AI Platform (Built-in NPU)

| Platform | RAM | AI Acceleration | Storage | Support Level |
|----------|-----|-----------------|---------|---------------|
| **Orange Pi 5B (8GB)** | 8 GB | 6 TOPS built-in | 64-256GB eMMC | ★★★ Budget AI |
| **Orange Pi 5B (16GB)** | 16 GB | 6 TOPS built-in | 128-256GB eMMC | ★★★ Budget AI |

### 4.4 Minimum Baseline

| Platform | RAM | AI Acceleration | Storage | Support Level |
|----------|-----|-----------------|---------|---------------|
| **Raspberry Pi 4 (8GB)** | 8 GB | External (Coral) | USB SSD or SD | ★★☆ |
| **Raspberry Pi 4 (4GB)** | 4 GB | External (Coral) | USB SSD or SD | ★★☆ Minimum |

### 4.5 Other Supported Platforms

| Platform | RAM | Notes | Support Level |
|----------|-----|-------|---------------|
| **Radxa Rock 5B** | 4-16 GB | RK3588, 6 TOPS NPU | ★★☆ |
| **Orange Pi 5 Plus** | 4-32 GB | RK3588, dual 2.5GbE, NVMe | ★★☆ |
| **x86_64 (any)** | 4+ GB | Generic x86 systems | ★★☆ |

### 4.6 Not Supported

| Platform | Reason |
|----------|--------|
| Raspberry Pi 4 (2GB) | Insufficient RAM for containers + workloads |
| Raspberry Pi 3 (all) | Insufficient RAM (1GB max) |
| Raspberry Pi Zero (all) | Insufficient RAM for containers |
| Orange Pi 5B (4GB) | Tight RAM for AI workloads |

### 4.7 Platform Comparison

| Platform | Pros | Cons | Best For |
|----------|------|------|----------|
| **Pi 5 (8GB)** | Best ecosystem, NVMe, large community | No built-in NPU | Beginners, ecosystem access |
| **Jetson Orin Super** | 67 TOPS, CUDA, multi-model | Higher cost, power, supply issues | Maximum AI, VLMs |
| **Orange Pi 5B (8GB)** | Built-in NPU, lower cost, eMMC | Smaller community, Armbian | Budget AI builds |
| **Pi 4 (4GB)** | Lowest cost, proven platform | Slower, less RAM | Simple robots |
| **Rock 5B** | NPU, NVMe, more I/O | Software gaps | Advanced users |

---

## 5. AI Accelerators

### 5.1 Supported Accelerators

| Accelerator | Performance | Interface | Notes |
|-------------|-------------|-----------|-------|
| **RK3588 NPU** | 6 TOPS | Integrated | Built into Orange Pi 5B, Rock 5B |
| **Hailo-8** | 26 TOPS | M.2, USB | Recommended for Pi 5 |
| **Hailo-8L** | 13 TOPS | M.2 | Lower power, good for drones |
| **Google Coral TPU** | 4 TOPS | USB, M.2, PCIe | TensorFlow Lite models |
| **NVIDIA CUDA** | Varies | Integrated (Jetson) | Full CUDA ecosystem |

### 5.2 Accelerator Selection Guide

| Use Case | Recommended Solution |
|----------|---------------------|
| **Budget AI build** | Orange Pi 5B (built-in 6 TOPS NPU) |
| Object detection (YOLO) | Hailo-8 or RK3588 NPU |
| Multi-model inference | Hailo-8 (26 TOPS headroom) |
| TensorFlow Lite models | Coral TPU |
| PyTorch models | NVIDIA Jetson or Hailo |
| Maximum performance | NVIDIA Jetson Orin |

### 5.3 Cost Comparison for AI Workloads

| Configuration | Total Cost | AI Performance | Best For |
|---------------|------------|----------------|----------|
| Orange Pi 5B (8GB) | ~$145 | 6 TOPS | Budget AI |
| Pi 5 (8GB) + Coral USB | ~$210 | 4 TOPS | TFLite models |
| Pi 5 (8GB) + Hailo-8L | ~$240 | 13 TOPS | Efficient vision |
| Pi 5 (8GB) + Hailo-8 | ~$270 | 26 TOPS | Multi-model |
| **Jetson Orin Super** | ~$335 | **67 TOPS** | Maximum AI, VLMs |

### 5.4 RK3588 NPU Details (Orange Pi 5B)

The RK3588's built-in NPU provides good vision inference without external hardware:

**Specifications:**
- 3 NPU cores @ 1GHz
- 6 TOPS peak (INT8)
- Supports INT8 and FP16 quantization

**Supported Frameworks (via RKNN-Toolkit2):**
- TensorFlow / TensorFlow Lite
- PyTorch (via ONNX)
- ONNX (opset 12 recommended)
- Caffe, MXNet, Darknet

**Workflow:**
```
x86_64 Linux Host              Orange Pi 5B
┌─────────────────────┐        ┌─────────────────────┐
│ 1. Export to ONNX   │        │                     │
│ 2. Convert to .rknn │───────►│ 3. Run with         │
│    (RKNN-Toolkit2)  │        │    RKNN-Lite        │
└─────────────────────┘        └─────────────────────┘
```

**Note:** Model conversion requires an x86_64 Linux host. Models cannot be converted directly on the ARM board.

### 5.5 Without Accelerator

CPU-only inference is supported but limited:

| Model | Pi 5 CPU | Orange Pi 5B NPU | Pi 5 + Hailo-8L |
|-------|----------|------------------|-----------------|
| YOLOv8 (640) | 2-3 FPS | 25-30 FPS | 30+ FPS |
| MobileNet | 10-15 FPS | 50+ FPS | 100+ FPS |
| ResNet-50 | 1-2 FPS | 20+ FPS | 50+ FPS |

**Recommendation:** Use built-in NPU (Orange Pi 5B) or external accelerator (Hailo, Coral) for real-time vision.

---

## 6. Storage Configuration

### 6.1 Recommended Storage Setup

**Raspberry Pi 4:**
```
USB 3.0 SSD (blue port) or SD Card
├── Boot partition (256MB, FAT32)
├── Root partition (remaining, ext4)
└── Container data: /var/lib/containers
```

**Raspberry Pi 5:**
```
NVMe SSD (via HAT) — recommended
├── Boot partition (512MB, FAT32)
├── Root partition (remaining, ext4)
└── Container data: /var/lib/containers

Or USB 3.0 SSD, or SD Card (A2 class) for deployed robots
```

**Orange Pi 5B:**
```
Built-in eMMC (64-256GB)
├── Boot partition (512MB, FAT32)
├── Root partition (remaining, ext4)
└── Container data: /var/lib/containers

No external storage required
```

**Jetson Orin Nano Super:**
```
NVMe SSD (M.2 2280) — required
├── Boot partition (512MB, FAT32)
├── Root partition (remaining, ext4)
└── Container data: /var/lib/containers

No built-in storage; NVMe or USB SSD required
```

### 6.2 Storage Performance Requirements

| Metric | Minimum | Recommended |
|--------|---------|-------------|
| Sequential Read | 200 MB/s | 500+ MB/s |
| Sequential Write | 150 MB/s | 400+ MB/s |
| Random Read IOPS | 500 | 5,000+ |
| Random Write IOPS | 500 | 5,000+ |

### 6.3 Recommended SSDs

| Category | Examples |
|----------|----------|
| **NVMe (Pi 5)** | Samsung 980, WD SN570, Crucial P3 |
| **USB 3.0 SATA** | Samsung T7, SanDisk Extreme, Crucial X6 |
| **Budget** | Kingston A400 + enclosure |

**Avoid:** QLC drives for heavy write workloads, no-name brands with poor controllers.

---

## 7. Bill of Materials

### 7.1 Minimum Build (~$105)

| Component | Specification | Cost |
|-----------|---------------|------|
| Raspberry Pi 4 (4GB) | | $55 |
| USB 3.0 SSD (128GB) | Samsung T7 or similar | $30 |
| Power Supply | 5V/3A USB-C | $15 |
| SD Card (boot) | 16GB for initial setup | $5 |
| **Total** | | **~$105** |

### 7.2 Recommended Build (~$150)

| Component | Specification | Cost |
|-----------|---------------|------|
| Raspberry Pi 5 (8GB) | | $80 |
| NVMe SSD (256GB) | Samsung 980 or similar | $30 |
| NVMe HAT | Pimoroni NVMe Base | $20 |
| Power Supply | 5V/5A USB-C PD | $15 |
| Heatsink | Active cooling | $10 |
| **Total** | | **~$155** |

### 7.3 Budget AI Build (~$145) — Orange Pi 5B

| Component | Specification | Cost |
|-----------|---------------|------|
| Orange Pi 5B (8GB/64GB) | 8-core RK3588S, built-in 6 TOPS NPU | $115 |
| Power Supply | 5V/4A USB-C | $15 |
| Heatsink/Fan | **Required** | $15 |
| **Total** | | **~$145** |

**Note:** Built-in eMMC and NPU eliminate need for external SSD or AI accelerator.

### 7.4 AI-Ready Build (~$250) — Pi 5 + Hailo

| Component | Specification | Cost |
|-----------|---------------|------|
| Raspberry Pi 5 (8GB) | | $80 |
| NVMe SSD (256GB) | | $30 |
| NVMe HAT | With M.2 AI slot | $30 |
| Hailo-8L M.2 | 13 TOPS accelerator | $70 |
| Power Supply | 5V/5A USB-C PD | $15 |
| Heatsink | Active cooling | $10 |
| **Total** | | **~$235** |

### 7.5 Performance AI Build (~$335) — Jetson Orin Super

| Component | Specification | Cost |
|-----------|---------------|------|
| Jetson Orin Nano Super | 8GB, 67 TOPS (CUDA) | $249 |
| NVMe SSD (256GB) | M.2 2280 | $30 |
| WiFi Module | Intel AC8265 (M.2 Key E) | $20 |
| Power Supply | 5V/5A USB-C | $15 |
| Active Cooling | Fan + heatsink | $20 |
| **Total** | | **~$335** |

**Note:** 67 TOPS AI performance enables multi-model inference, VLMs, and on-robot LLMs.

---

## 8. Setup Checklist

### 8.1 Pre-Installation

- [ ] Verify platform meets minimum requirements (4GB RAM, ARM64/x86_64)
- [ ] Obtain storage (SSD recommended for development; SD card acceptable for deployed robots)
- [ ] Quality power supply available
- [ ] Ethernet connection available (recommended)

### 8.2 Storage Setup

**Raspberry Pi:**
- [ ] Flash Raspberry Pi OS (64-bit, Lite) to storage
- [ ] Configure boot from USB/NVMe (Pi 4 may need EEPROM update)
- [ ] Verify storage is recognized and bootable

**Orange Pi 5B:**
- [ ] Download Armbian (Debian-based, server image recommended)
- [ ] Flash Armbian to built-in eMMC (via USB-C or SD card installer)
- [ ] Verify eMMC boot and Armbian first-run setup

**Jetson Orin Nano Super:**
- [ ] Download JetPack 6.2+ from NVIDIA
- [ ] Flash JetPack to NVMe SSD (via SDK Manager or manual)
- [ ] Install WiFi module (M.2 Key E slot)
- [ ] Configure active cooling (fan required for sustained loads)

### 8.3 Network Setup

- [ ] Assign static IP or configure DHCP reservation
- [ ] Verify internet connectivity (for package installation)
- [ ] Configure hostname

### 8.4 Gorai Installation

```bash
# Install gorai CLI
curl -sfL https://get.gorai.dev | sh

# Initialize Gorai (installs Podman if needed, pulls base images)
gorai init

# Verify installation
gorai version
```

---

## 9. Troubleshooting

### 9.1 Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Pod won't start | Insufficient RAM | Upgrade to 4GB+ platform |
| Container image pull fails | Network/storage | Check connectivity, storage space |
| Throttling under load | Overheating | Add active cooling |
| USB SSD not detected | Power issues | Use powered USB hub or quality PSU |
| NPU not detected (Orange Pi) | Missing RKNN driver | Install rknpu2 kernel module |
| RKNN model fails to load | Wrong toolkit version | Match RKNN-Toolkit2 to board's runtime |
| Orange Pi thermal throttle | Board runs hot | **Required**: Add active fan cooling |
| Jetson GPU not in container | Runtime not configured | Install nvidia-container-toolkit |
| Jetson thermal throttle | Sustained 25W load | Improve cooling or use 15W mode |

### 9.2 Performance Verification

```bash
# Check memory
free -h

# Check storage performance
sudo hdparm -Tt /dev/sda  # Replace with your device

# Check CPU temperature
vcgencmd measure_temp  # Raspberry Pi
cat /sys/class/thermal/thermal_zone*/temp  # Jetson, Orange Pi

# Check container resource usage
podman stats
```

### 9.3 Storage Performance Test

```bash
# Install fio
sudo apt install fio

# Random read/write test
fio --name=random-rw --ioengine=libaio --iodepth=32 \
    --rw=randrw --bs=4k --direct=1 --size=1G \
    --numjobs=4 --runtime=60 --group_reporting

# Target: 500+ IOPS for both read and write
```

---

## Appendix A: Platform-Specific Notes

### A.1 Raspberry Pi 4

- **EEPROM Update:** Required for USB boot on older units
  ```bash
  sudo rpi-eeprom-update -a
  ```
- **USB 3.0:** Use blue USB ports for SSD
- **Power:** Genuine 5V/3A supply required; undervoltage causes instability

### A.2 Raspberry Pi 5

- **NVMe:** Native PCIe support via HAT/Base
- **Power:** Requires 5V/5A for full performance; PD power supply recommended
- **Cooling:** Active cooling recommended for sustained workloads

### A.3 NVIDIA Jetson Orin Nano Super

- **JetPack 6.2+:** Required for "Super" mode (67 TOPS)
  ```bash
  # Check JetPack version
  cat /etc/nv_tegra_release

  # Upgrade existing Orin Nano to Super mode
  sudo apt update && sudo apt upgrade
  ```
- **Power Modes:**
  - 7W (eco) — Battery operation
  - 15W (default) — Balanced
  - 25W MAXN SUPER — Maximum performance (67 TOPS)
  ```bash
  # Set power mode
  sudo nvpmodel -m 0  # MAXN SUPER (25W)
  sudo nvpmodel -m 1  # 15W
  sudo nvpmodel -m 2  # 7W

  # Check current mode
  nvpmodel -q
  ```
- **Storage:** NVMe SSD required (no built-in storage)
- **WiFi:** M.2 Key E module required (Intel AC8265 recommended)
- **Cooling:** Active fan required for 25W MAXN SUPER mode
- **Container GPU Access:**
  ```bash
  # Install NVIDIA Container Toolkit
  sudo apt install nvidia-container-toolkit
  sudo nvidia-ctk runtime configure --runtime=podman

  # Test GPU access in container
  podman run --rm --runtime=nvidia --device=nvidia.com/gpu=all \
    nvcr.io/nvidia/l4t-base:r36.2.0 nvidia-smi
  ```

### A.4 Orange Pi 5B

- **OS:** Use Armbian (Debian-based); official Orange Pi images less stable
  ```bash
  # Download from: https://www.armbian.com/orangepi-5b/
  # Recommended: Armbian Bookworm server image
  ```
- **eMMC Flashing:** Flash via USB-C (Maskrom mode) or SD card installer
- **Cooling:** **Active fan required** — RK3588S runs hot under sustained load
- **NPU Setup:**
  ```bash
  # Verify NPU is available
  ls /dev/dri/renderD*

  # Check RKNN runtime version
  cat /usr/lib/librknnrt.so | strings | grep version
  ```
- **Model Conversion:** Requires x86_64 Linux host with RKNN-Toolkit2
  - Cannot convert models directly on ARM board
  - Match toolkit version to runtime version on board

### A.5 Rockchip RK3588 (Rock 5B, Orange Pi 5 Plus)

- **NPU:** Requires RKNN toolkit for model conversion (same as Orange Pi 5B)
- **Vendor Images:** Use vendor-provided images for best hardware support
- **Mainline Linux:** Limited NPU support in mainline kernel
- **NVMe:** These boards support NVMe SSDs directly (unlike Orange Pi 5B)

---

## Appendix B: Upgrade Paths

### B.1 Pi 4 (4GB) → Pi 5 (8GB)

1. Backup robot configuration: `gorai backup rover1`
2. Install Pi 5 with new SSD
3. Install Gorai: `curl -sfL https://get.gorai.dev | sh`
4. Initialize cluster: `gorai cluster init`
5. Restore: `gorai restore rover1 backup.tar.gz`

### B.2 Pi 4 → Orange Pi 5B (for AI)

1. Backup robot configuration: `gorai backup rover1`
2. Install Armbian on Orange Pi 5B (eMMC boot)
3. Install Gorai: `curl -sfL https://get.gorai.dev | sh`
4. Initialize cluster: `gorai cluster init`
5. Restore: `gorai restore rover1 backup.tar.gz`
6. Update RDL to use `rk3588_npu` accelerator
7. Re-deploy: `gorai deploy rover1.yaml`

### B.3 Adding AI Accelerator

1. Power down robot
2. Install accelerator (USB Coral, M.2 Hailo, etc.)
3. Install drivers (per accelerator documentation)
4. Update RDL to reference accelerator
5. Deploy updated configuration

### B.4 Single Robot → Fleet

1. Designate control plane node (or use cloud VM)
2. Initialize cluster in HA mode: `gorai cluster init --ha`
3. Join existing robots: `gorai cluster join <url> --token <token>`
4. Update RDL for fleet topology
