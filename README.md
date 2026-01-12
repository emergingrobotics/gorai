# Gorai

<img src="./images/gorai.png" width="25%">

**Professional robotics for prosumers — without the PhD**

*Pronounced "go-ray" (like "sting-ray")*

Gorai is a Go-based robotics framework designed for makers, citizen scientists, students, and small organizations who need real autonomy without ROS 2's complexity. Built on battle-tested cloud infrastructure (NATS, Prometheus), Gorai applies distributed systems patterns to robotics—making professional-grade software accessible to everyone.

We originally wanted to name this project "Gort" after the iconic robot from *The Day the Earth Stood Still*, but that name has been taken for years by DevOps tooling for the excellent [GoBot](https://gobot.io/) project. Gorai is a contraction of **Go + Robot + AI**—and it evokes the [eye ray gun of Gort](https://youtu.be/K6iF5sINVns?t=92)!

---

## Who Is Gorai For?

### You Should Use Gorai If:
- ✅ You want **real autonomy** (not educational toys, not simulations)
- ✅ You need **approachable software** (productive in days, not months)
- ✅ You value **modern developer experience** (AI-assisted coding, simple deployment)
- ✅ You're building **prosumer robots** (marine monitoring, land vehicles, research platforms)
- ✅ You want **cloud-native patterns** (NATS, Prometheus, containerization when needed)

### You Should Use ROS 2 If:
- ❌ You're in **enterprise/research** robotics (warehouse automation, autonomous vehicles)
- ❌ You need the **full ROS ecosystem** (thousands of packages, simulation, SLAM libraries)
- ❌ You have **months to invest** in learning complex toolchains
- ❌ You're in **academia** where ROS 2 is the standard

**We're not trying to replace ROS 2.** We're targeting a different market—prosumers who find Arduino too limiting and ROS 2 too complex. Think of Gorai as "ROS 2 for prosumers," not "ROS 2 killer."

## Why Gorai?

### The Cloud-Native Advantage

Gorai applies proven **distributed systems patterns from cloud infrastructure** to robotics:

| Cloud Pattern | ROS 2 Approach | Gorai Approach | Benefit |
|---------------|----------------|----------------|---------|
| **Message Broker** | Direct DDS peer-to-peer | NATS server | Decouples producers/consumers; easy monitoring |
| **Service Mesh** | Custom DDS discovery | NATS request/reply + queue groups | Automatic load balancing, failover |
| **Event Sourcing** | rosbag (manual) | JetStream (built-in) | Replay sensor streams, time-travel debugging |
| **Config Management** | Per-node params | NATS KV store | Global config, hot reload, version history |
| **Observability** | Custom diagnostics | Prometheus /metrics | Industry-standard dashboards, alerting |
| **Security** | DDS Security (complex) | NATS auth, TLS, JWT | Simpler, firewall-friendly |

**ROS 2's DDS is pre-cloud architecture** (designed for LANs in 2004). Gorai uses infrastructure that powers Coinbase, Mastercard, and Siemens—proven at global scale.

### Comparison Table

| Aspect | Gorai | ROS 2 | Viam | YARP |
|--------|-------|-------|------|------|
| **Target Market** | Prosumer | Enterprise/Research | Cloud-first | Research |
| **Language** | Go + TinyGo | C++/Python | Go | C++ |
| **Middleware** | NATS (cloud-native) | DDS (pre-cloud) | gRPC | Custom carriers |
| **Learning Curve** | Days | Months | Moderate | High |
| **Build System** | Go modules | CMake + ament + colcon | Go modules | CMake |
| **Deployment** | Native + systemd | Workspace sourcing | Cloud-dependent | Build artifacts |
| **AI/ML** | First-class + TPU/NPU | Package ecosystem | First-class | Minimal |
| **MCU Support** | TinyGo | micro-ROS | None | None |
| **ROS 2 Bridge** | Planned (Phase 3) | Native | None | Yes |
| **License** | Apache 2.0 | Apache 2.0 | AGPL | BSD-3 |

---

## Getting Started

This guide walks you through setting up Gorai on a **Raspberry Pi 5** and deploying your first robot from a development machine.

> **💻 Want to test locally first?** See [Quick Testing (Local Development)](#quick-testing-local-development) and the [Setup Guide](examples/hello-robot/SETUP.md).

### Prerequisites

**Hardware:**
- Raspberry Pi 5 (8GB recommended)
- NVMe SSD via HAT or USB 3.0 SSD (128GB minimum)
- SD card for initial OS installation
- Development machine (Linux or macOS)
- Network connection (Ethernet recommended for RPi)

**Software on Development Machine:**
- Go 1.22+ ([download](https://go.dev/dl/))
- Podman (for building container images)
- kubectl ([install](https://kubernetes.io/docs/tasks/tools/))
- ssh client

### Step 1: Prepare Raspberry Pi 5

**1.1 Install Raspberry Pi OS (64-bit)**

```bash
# On your development machine:
# Download Raspberry Pi Imager
# https://www.raspberrypi.com/software/

# Flash Raspberry Pi OS (64-bit, Bookworm) to SD card
# Enable SSH and configure WiFi/user in Imager before flashing
```

**1.2 Boot and Update**

```bash
# SSH into Pi (default: pi@raspberrypi.local or use IP address)
ssh pi@raspberrypi.local

# Update system
sudo apt update && sudo apt upgrade -y

# Set hostname
sudo hostnamectl set-hostname robot1

# Reboot
sudo reboot
```

**1.3 Set Up NVMe/USB SSD**

```bash
# SSH back in
ssh pi@robot1.local

# Check if SSD is detected
lsblk

# Format SSD (assuming /dev/nvme0n1 for NVMe or /dev/sda for USB)
sudo mkfs.ext4 /dev/nvme0n1  # OR: sudo mkfs.ext4 /dev/sda

# Create mount point
sudo mkdir -p /mnt/storage

# Mount SSD
sudo mount /dev/nvme0n1 /mnt/storage  # OR: sudo mount /dev/sda /mnt/storage

# Get UUID for persistent mounting
sudo blkid /dev/nvme0n1  # Note the UUID

# Add to /etc/fstab for automatic mounting
echo "UUID=<your-uuid-here> /mnt/storage ext4 defaults 0 2" | sudo tee -a /etc/fstab

# Test fstab
sudo umount /mnt/storage
sudo mount -a
df -h | grep storage  # Should show mounted
```

### Step 2: Install K3s on Raspberry Pi

```bash
# On the Raspberry Pi
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644 \
  --data-dir /mnt/storage/k3s

# Verify K3s is running
sudo systemctl status k3s

# Check node is ready
sudo k3s kubectl get nodes
# Should show: robot1   Ready   control-plane,master

# Copy kubeconfig for non-root access
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $USER:$USER ~/.kube/config
chmod 600 ~/.kube/config
```

### Step 3: Configure Development Machine

**3.1 Install Prerequisites**

**Linux:**
```bash
# Install Go (if not installed)
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Install Podman
sudo apt update
sudo apt install -y podman

# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
```

**macOS:**
```bash
# Install Homebrew (if not installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install prerequisites
brew install go
brew install podman
brew install kubectl

# Initialize and start Podman machine
podman machine init
podman machine start

# CRITICAL: Set DOCKER_HOST for k3d/Podman compatibility
# This tells k3d to use Podman instead of Docker
export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock
echo "export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock" >> ~/.zshrc
source ~/.zshrc

# Verify DOCKER_HOST is set
echo $DOCKER_HOST
```

> **⚠️ macOS Users:** The `DOCKER_HOST` variable is **required** for k3d to work with Podman. Without it, you'll see "Cannot connect to the Docker daemon" errors.

**3.2 Access Remote K3s Cluster**

```bash
# On development machine: Copy kubeconfig from Raspberry Pi
scp pi@robot1.local:~/.kube/config ~/.kube/robot1-config

# Edit the config to use robot's IP instead of localhost
sed -i 's/127.0.0.1/robot1.local/g' ~/.kube/robot1-config

# Use this kubeconfig
export KUBECONFIG=~/.kube/robot1-config

# Test connection
kubectl get nodes
# Should show: robot1   Ready   control-plane,master
```

### Step 4: Build and Deploy Hello Robot Example

**4.1 Clone Gorai Repository**

```bash
# On development machine
git clone https://github.com/gorai/gorai.git
cd gorai/examples/hello-robot
```

**4.2 Build Container Images**

```bash
# Build for ARM64 (Raspberry Pi architecture)
make build-arm64

# This builds:
# - hello-robot-publisher:latest
# - hello-robot-subscriber:latest
```

**4.3 Push Images to Raspberry Pi**

```bash
# Option 1: Save and load images
podman save hello-robot-publisher:latest | ssh pi@robot1.local sudo k3s ctr images import -

podman save hello-robot-subscriber:latest | ssh pi@robot1.local sudo k3s ctr images import -

# Option 2: Use a registry (for production)
# Set up a local registry or use a container registry
```

**4.4 Deploy to K3s**

```bash
# Deploy hello-robot
kubectl apply -f deploy/

# Watch deployment
kubectl get pods -n hello-robot -w

# Expected output:
# NAME                          READY   STATUS    RESTARTS   AGE
# nats-0                        1/1     Running   0          30s
# publisher-xxxxxxxxx-xxxxx     1/1     Running   0          30s
# subscriber-xxxxxxxxx-xxxxx    1/1     Running   0          30s
```

**4.5 View Logs**

```bash
# View subscriber logs (should show received messages)
kubectl logs -n hello-robot -l app=subscriber -f

# Expected output:
# 2025-01-11T22:00:00Z Received: Hello #1
# 2025-01-11T22:00:01Z Received: Hello #2
# 2025-01-11T22:00:02Z Received: Hello #3
# ...
```

**4.6 Clean Up**

```bash
# Remove deployment
kubectl delete namespace hello-robot
```

### Troubleshooting

**Can't connect to K3s on Raspberry Pi:**
- Check firewall: `sudo ufw allow 6443/tcp`
- Verify K3s is running: `ssh pi@robot1.local sudo systemctl status k3s`
- Check kubeconfig has correct IP address

**Images won't pull on Raspberry Pi:**
- Verify images are ARM64 architecture
- Check images are imported: `ssh pi@robot1.local sudo k3s ctr images ls | grep hello-robot`

**Pods stuck in Pending:**
- Check resources: `kubectl describe node robot1`
- View pod events: `kubectl describe pod -n hello-robot <pod-name>`

---

## Quick Testing (Local Development)

For rapid development and testing, you can run Gorai components locally on your Linux or macOS machine.

> **📋 For detailed installation instructions, see [Setup Guide: Local Testing Environment](examples/hello-robot/SETUP.md)**

### Prerequisites

**Both Linux and macOS:**
- Go 1.22+ installed
- Podman installed
- kubectl installed (for K3s/K3d testing)
- NATS server installed (for native testing)

If you don't have these installed, follow the [setup guide](examples/hello-robot/SETUP.md) for step-by-step installation instructions for your platform.

### Option 1: K3s Locally (Recommended)

**Linux:**

```bash
# Install K3s locally
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644

# Use local K3s
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

# Verify
kubectl get nodes
```

**macOS:**

```bash
# Install K3d (K3s in Podman)
brew install k3d

# Verify DOCKER_HOST is set (required for Podman)
echo $DOCKER_HOST
# Should output: unix:///Users/<username>/.local/share/containers/podman/machine/podman.sock

# Create local cluster
k3d cluster create gorai-dev

# Verify
kubectl get nodes
```

> **Note:** If you get "Cannot connect to the Docker daemon" errors, ensure `DOCKER_HOST` is set. See the [Setup Guide](examples/hello-robot/SETUP.md) for details.

### Option 2: Native NATS (Simpler, No K3s)

**Linux:**

```bash
# Install NATS server
curl -L https://github.com/nats-io/nats-server/releases/download/v2.10.7/nats-server-v2.10.7-linux-amd64.zip -o nats-server.zip
unzip nats-server.zip
sudo mv nats-server-v2.10.7-linux-amd64/nats-server /usr/local/bin/
rm -rf nats-server-v2.10.7-linux-amd64 nats-server.zip

# Run NATS server
nats-server &
```

**macOS:**

```bash
# Install NATS server
brew install nats-server

# Run NATS server
nats-server &
```

### Running Hello Robot Example Locally

**Using K3s/K3d (Containerized):**

```bash
# Navigate to hello-robot example
cd gorai/examples/hello-robot

# Build for local architecture (amd64)
make build

# IMPORTANT: Import images into K3s/K3d
# Linux (K3s):
podman save hello-robot-publisher:latest | sudo k3s ctr images import -
podman save hello-robot-subscriber:latest | sudo k3s ctr images import -

# macOS (K3d):
k3d image import hello-robot-publisher:latest -c gorai-dev
k3d image import hello-robot-subscriber:latest -c gorai-dev

# Deploy to local K3s
kubectl apply -f deploy/

# Watch logs
kubectl logs -n hello-robot -l app=subscriber -f
```

> **Note:** Images built locally must be imported into K3s/K3d. Skipping this causes "image can't be pulled" errors.

**Using Native NATS (No Containers):**

```bash
# Navigate to hello-robot example
cd gorai/examples/hello-robot

# Build binaries
make build-native

# Run publisher in one terminal
./bin/publisher

# Run subscriber in another terminal
./bin/subscriber

# You should see:
# Terminal 2 (subscriber):
# 2025-01-11T22:00:00Z Received: Hello #1
# 2025-01-11T22:00:01Z Received: Hello #2
# ...
```

### Quick Verification

```bash
# Install NATS CLI for debugging
# Linux
curl -L https://github.com/nats-io/natscli/releases/download/v0.1.1/nats-0.1.1-linux-amd64.zip -o nats.zip
unzip nats.zip
sudo mv nats-0.1.1-linux-amd64/nats /usr/local/bin/
rm -rf nats-0.1.1-linux-amd64 nats.zip

# macOS
brew install nats-io/nats-tools/nats

# Subscribe to messages
nats sub "hello.messages"

# You should see messages being published
```

### Cleanup

**K3s/K3d:**
```bash
# Remove deployment
kubectl delete namespace hello-robot

# Stop K3d cluster (macOS)
k3d cluster delete gorai-dev
```

**Native:**
```bash
# Stop publisher and subscriber (Ctrl+C in terminals)

# Stop NATS
killall nats-server
```

---

## Design Principles

Drawing from [our analysis](docs/general-designs.md) of ROS 2, Viam, and YARP, plus [strategic vision](docs/vision-analysis.md):

### What We Adopt

- **Resource-centric model** (from Viam): Unified abstraction for components and services
- **Named addressing** (from ROS 2, Viam, YARP): Hierarchical, human-readable identifiers
- **Message types** (from ROS 2): Proven sensor_msgs, geometry_msgs patterns
- **Transform trees** (from ROS 2 TF2): Coordinate frame management
- **Configuration-driven** (from Viam): JSON config with hot reload
- **Device interfaces** (from all): Clean separation of hardware from logic
- **NWS/NWC pattern** (from YARP): Transparent local/remote resource access

### What We Differentiate

- **NATS over DDS**: Cloud-native message broker vs. pre-cloud middleware
- **Go core + polyglot services**: Pragmatic language choices per component
- **Prometheus native**: Industry-standard observability, not custom diagnostics
- **JetStream**: Built-in event sourcing, not bolt-on database
- **Native deployment**: Components run in single process for simplicity and performance
- **External services**: AI/ML can run in containers with specialized hardware access
- **TinyGo support**: Unified language from microcontrollers to cloud
- **TPU/NPU focus**: Edge AI as primary concern, not afterthought
- **No cloud dependency**: Standalone-first, cloud-optional
- **Lower barrier**: Productive in days, not months

### What We Avoid

- PhD-level learning curves (CMake, colcon, ament, DDS QoS)
- Language purity dogma (use best tool for each job)
- Kubernetes complexity exposure (RDL abstracts it away)
- "Not Invented Here" syndrome (ROS 2 bridge planned for ecosystem access)
- Complex middleware that leaks implementation details

## Language Philosophy: Pragmatic C++ Use

Gorai is a Go-first framework, but we use C++ pragmatically when there are **technical reasons** that justify it. This philosophy reflects the realities of modern AI-assisted development.

### When We Use C++

C++ is justified when **at least one** of these conditions is true:

1. **Vendor-provided drivers too complex to port**
   - Hardware SDK with thousands of lines of low-level device control
   - Proprietary vendor libraries with no public specification
   - Real-time constraints requiring vendor-tuned implementations

2. **Performance-critical code requiring non-GC environment**
   - Sub-millisecond control loops (motor commutation, safety cutoffs)
   - Zero-allocation hot paths in hard real-time contexts
   - Direct hardware register manipulation

3. **Irreplaceable research implementations**
   - SLAM algorithms representing years of academic research (Cartographer, ORB-SLAM3)
   - When reimplementation would introduce novel bugs or lose proven stability

### When We Don't Use C++

**"It already exists in C++" is NOT sufficient justification.**

In the AI-assisted development era, source code has less intrinsic value than it once did. Modern AI coding assistants can:

- Port C++ implementations to Go with high accuracy
- Modernize architecture during porting (add NATS messaging, Prometheus metrics)
- Generate comprehensive tests during translation
- Refactor for clarity and Go idioms

### The AI-Assisted Porting Advantage

Many robotics libraries were written in C++ during an era when:
- Manual porting was prohibitively expensive
- Existing code represented years of developer time
- Rewriting meant high risk of introducing bugs

**This calculus has changed.** With AI assistance:

```
Traditional Development:
Port 10,000 lines C++ → Go manually
├── 4-6 weeks developer time
├── High bug introduction risk
└── Exhausting, error-prone work

AI-Assisted Development:
Port 10,000 lines C++ → Go with Claude/Copilot
├── 2-4 days developer time (review + iteration)
├── Comprehensive tests generated alongside
├── Modernize to use NATS, Prometheus during port
└── Often results in cleaner, more maintainable code
```

### Integration Pattern: Wrap Only When Necessary

When C++ is justified, we integrate it cleanly:

**Option 1: External Service (Preferred)**
```
┌─────────────────────────────────────────┐
│  C++ Service (containerized)            │
│  ├── Vendor SDK (C++)                   │
│  ├── NATS client (connects to broker)   │
│  └── Prometheus metrics                 │
└─────────────────────────────────────────┘
         │ NATS messaging
         ▼
┌─────────────────────────────────────────┐
│  Gorai Core (Go)                        │
│  └── Treats C++ service like any other  │
└─────────────────────────────────────────┘
```

**Option 2: CGo Wrapper (When External Process Impractical)**
```go
// Thin CGo wrapper in satellite repository
// github.com/gorai/gorai-driver-realsense

package realsense

// #cgo LDFLAGS: -lrealsense2
// #include <librealsense2/rs.h>
import "C"

type Camera struct {
    ctx C.rs2_context
    // ... minimal wrapper state
}

// Wrapper provides Go interface, delegates to C++
func (c *Camera) CaptureFrame() (image.Image, error) {
    // Minimal CGo calls
}
```

**Key Principle:** Keep C++ isolated. Core gorai repository remains pure Go.

### Examples in Gorai

| Component | Language | Justification |
|-----------|----------|---------------|
| **NATS messaging** | Pure Go | Native Go library, excellent performance |
| **Web dashboard** | Pure Go | stdlib http + templates sufficient |
| **GPS driver** | Pure Go | NMEA protocol simple to parse |
| **IMU sensor** | Pure Go | I2C protocol, well-documented registers |
| **Simple vision** | Python service | OpenCV ecosystem (port not cost-effective *yet*) |
| **RealSense camera** | C++ wrapper | Vendor SDK, complex low-level USB control |
| **Motor PID** | TinyGo on MCU | Real-time requirements, no GC pauses |
| **YOLO inference** | Python or ONNX | ML frameworks (porting detection model: no value) |
| **Cartographer SLAM** | C++ service | Years of research, proven stability |

### Decision Framework

When evaluating a potential dependency:

```
┌─────────────────────────────────────────────────────┐
│ Is there a Go implementation?                       │
│ ├─ Yes → Use it                                     │
│ └─ No → Continue...                                 │
└─────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────┐
│ Can AI port it in <1 week?                          │
│ ├─ Yes → Port it to Go, modernize architecture      │
│ └─ No → Continue...                                 │
└─────────────────────────────────────────────────────┘
         │
         ▼
┌─────────────────────────────────────────────────────┐
│ Does it have technical justification?               │
│ (vendor SDK complexity, real-time, research value)  │
│ ├─ Yes → Use C++ as external service or CGo wrapper │
│ └─ No → Build in Go from scratch                    │
└─────────────────────────────────────────────────────┘
```

### Why This Matters

**For users:** Simpler builds, fewer dependencies, easier debugging, AI-assisted customization

**For contributors:** Modern, readable code; AI coding assistants work better with Go than C++

**For the project:** Lower maintenance burden, faster feature velocity, wider contributor base

We're building a framework for the 2020s, not the 1990s. Language choices should reflect modern development realities.

## Architecture: K3s-Everywhere

Gorai uses a **K3s-everywhere architecture** where all robots deploy on Kubernetes (K3s), from simple single robots to multi-robot fleets. This provides a consistent deployment model that scales without architectural changes.

### Why K3s-Everywhere?

**"AI at the edge requires capable hardware. Capable hardware can run K3s."**

- Edge AI workloads (vision, SLAM) need 4GB+ RAM regardless of orchestration
- One deployment model to learn, debug, and maintain
- Every robot is fleet-ready from day one
- Container benefits: reproducible builds, versioned artifacts, isolated dependencies
- K3s is designed for edge/IoT: ~50MB binary, ~512MB RAM overhead

**See [K3s Installation Guide](specs/k3s-installation.md) for platform-specific installation instructions.**

### What Users See vs What Runs

```
┌─────────────────────────────────────────────────────────────────┐
│                    User Experience                               │
│                                                                  │
│   robot.json (RDL)  →  gorai deploy  →  Robot running           │
│                                                                  │
│   Users work with: Robot Definition Language (JSON)             │
│   Users never need: kubectl, manifests, pods, deployments       │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ gorai translates
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    What Actually Runs                            │
│                                                                  │
│   K3s Cluster (single-node or multi-node)                       │
│   ├── Namespace: gorai-{robot-name}                             │
│   ├── Pod: nats (message broker)                                │
│   ├── Pod: gorai-core (Go orchestration + components)           │
│   ├── Pod: {service} (vision, SLAM, navigation)                 │
│   └── ConfigMap, Services, PVCs (auto-generated)                │
└─────────────────────────────────────────────────────────────────┘
```

### Hardware Requirements

| Component | Minimum | Recommended |
|-----------|---------|-------------|
| **Compute** | Raspberry Pi 4 (4GB) | Raspberry Pi 5 (8GB) |
| **Storage** | USB 3.0 SSD (128GB) | NVMe SSD (256GB) |
| **Cost** | ~$105 | ~$145 |

**Not supported:** Pi 3, Pi Zero, Pi 4 (2GB), SD card-only deployments

**Why SSD required:** K3s uses SQLite which requires sustained random I/O. SD cards provide only 10-30 IOPS, causing database corruption and instability.

### Single Robot Deployment

```
┌─────────────────────────────────────────────────────────────────┐
│  Raspberry Pi 5 (8GB) + NVMe SSD                                │
│                                                                  │
│  K3s (containerd)                                               │
│  ├── nats pod           (message broker)                        │
│  ├── gorai-core pod     (orchestration, components)             │
│  ├── detector pod       (vision service)                        │
│  └── navigation pod     (path planning)                         │
│                                                                  │
│  Hardware: /dev/video0, /dev/i2c-1, /dev/hailo0                 │
└─────────────────────────────────────────────────────────────────┘
```

### Fleet Deployment

```
┌──────────────────┐  ┌──────────────────┐  ┌──────────────────┐
│     Robot 1      │  │     Robot 2      │  │     Robot 3      │
│  K3s (worker)    │  │  K3s (worker)    │  │  K3s (worker)    │
│  NATS Leaf Node  │  │  NATS Leaf Node  │  │  NATS Leaf Node  │
└────────┬─────────┘  └────────┬─────────┘  └────────┬─────────┘
         └─────────────────────┼─────────────────────┘
                               │
                    ┌──────────┴──────────┐
                    │   Control Plane      │
                    │   K3s Server         │
                    │   NATS Hub           │
                    │   ArgoCD (GitOps)    │
                    └─────────────────────┘
```

### Components vs Services

| Aspect | Components | Services |
|--------|------------|----------|
| **What** | Hardware abstractions | Software capabilities |
| **Implementation** | Native Go in gorai-core pod | Separate pods (any language) |
| **Hardware access** | Direct device passthrough | Via NATS messaging |
| **Examples** | Camera, Motor, IMU, GPIO | Vision, SLAM, Navigation |

All communication happens via NATS—components and services are logically separate but deployed as Kubernetes pods.

## Quick Start

### Prerequisites

- Raspberry Pi 4 (4GB+) or Pi 5 with external SSD
- Linux (Raspberry Pi OS 64-bit, Ubuntu)
- Internet connection for initial setup

### 1. Install K3s and Gorai

**First, install K3s on your platform:**

See the [K3s Installation Guide](specs/k3s-installation.md) for detailed, platform-specific instructions (Raspberry Pi 5, Orange Pi 5B, Jetson Orin, etc.).

**Quick install (Raspberry Pi 5):**

```bash
# Install K3s
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644

# Verify K3s is running
sudo systemctl status k3s
sudo k3s kubectl get nodes

# Install gorai CLI
curl -sfL https://get.gorai.dev | sh
```

### 2. Create a robot configuration

```json
{
  "$schema": "https://gorai.dev/schemas/rdl-v3.json",
  "version": "3",
  "robot": {
    "name": "my-robot",
    "description": "Example robot with camera"
  },
  "components": [
    {
      "name": "main_camera",
      "type": "camera",
      "model": "v4l2",
      "attributes": {
        "device": "/dev/video0",
        "width": 640,
        "height": 480
      }
    }
  ],
  "dashboard": {
    "enabled": true
  }
}
```

### 3. Deploy the robot

```bash
# Deploy to K3s cluster
gorai deploy robot.json

# Watch deployment progress
gorai status my-robot
```

### 4. Manage the robot

```bash
# Check status
gorai status my-robot

# View logs
gorai logs my-robot -f

# Open dashboard
gorai dashboard my-robot

# Undeploy
gorai undeploy my-robot
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `gorai cluster init` | Initialize K3s on this machine |
| `gorai cluster join` | Join an existing K3s cluster |
| `gorai cluster status` | Show cluster health and nodes |
| `gorai deploy` | Deploy robot to K3s cluster |
| `gorai undeploy` | Remove robot from cluster |
| `gorai status` | Show robot status and pods |
| `gorai logs` | View robot logs |
| `gorai dashboard` | Open web dashboard |
| `gorai validate` | Validate configuration file |
| `gorai build` | Build container images |

## Architecture

### Object Model

Every component (hardware) and service (software capability) is a Resource:

```go
type Resource interface {
    Name() string
    Reconfigure(ctx context.Context, config Config) error
    Close(ctx context.Context) error
}
```

### Component Categories

| Category | What It Does | Examples |
|----------|--------------|----------|
| **Sensor** | Observes the world (read-only) | Camera, GPS, temperature sensor |
| **Actuator** | Changes the world (does stuff) | Motor, robotic arm, gripper |
| **Power** | Manages energy | Battery, power supply |
| **Link** | Extra communication channel | Serial to MCU, radio telemetry |

### Communication via NATS

Everything communicates via NATS messaging:

```
gorai.{robot}.{node}.{topic}

Example: gorai.sentinel.camera_front.data
         │      │        │            │
         │      │        │            └─ the actual topic
         │      │        └─ which component
         │      └─ which robot
         └─ framework prefix
```

## External Services (AI/ML)

For compute-intensive workloads like ML inference, services can run as external processes or containers. External services can have their own **Service RDL** file that defines their behavior independently:

### Service RDL Pattern

External services can be modular, reusable components with their own definition files:

```
services/
+-- person-detector/
    +-- person-detector.rdl.json   # Service RDL (defines interface)
    +-- main.py                     # Service implementation
    +-- Containerfile               # Container build
```

**Service RDL** (`person-detector.rdl.json`):
```json
{
  "kind": "service",
  "service": {
    "type": "object_detection",
    "model": "yolox"
  },
  "topics": {
    "subscribe": [
      {"name": "input", "pattern": "gorai.{namespace}.{input_component}.data"}
    ],
    "publish": [
      {"name": "annotated", "pattern": "gorai.{namespace}.{service}.annotated"},
      {"name": "detections", "pattern": "gorai.{namespace}.{service}.detections"}
    ]
  },
  "attributes": {
    "confidence_threshold": {"type": "float", "default": 0.5}
  }
}
```

**Robot RDL** references the Service RDL:
```json
{
  "services": [
    {
      "name": "person_detector",
      "rdl": "./services/person-detector/person-detector.rdl.json",
      "attributes": {
        "input_component": "main_camera",
        "model_path": "/models/yolox_s.hef"
      },
      "external": {
        "enabled": true,
        "container": {
          "image": "localhost/person-detector:latest",
          "devices": ["/dev/hailo0"]
        },
        "managed": true
      }
    }
  ]
}
```

### Benefits of Service RDL

- **Modularity**: Services are self-contained packages
- **Reusability**: Same service definition works across robots
- **Separation of concerns**: Service authors define behavior, robot integrators configure deployment
- **Documentation**: Service RDL documents the interface (topics, attributes)
- **Validation**: Attributes are type-checked at load time

This pattern enables:
- Python-based ML frameworks
- Specialized hardware access (Hailo NPU, Coral TPU)
- Independent updates and scaling
- Running on different hosts

## AI/ML Integration

Gorai provides first-class support for edge AI with hardware acceleration.

### Services

- **Vision**: Object detection, classification, segmentation
- **ML Model**: Generic tensor inference with TPU/NPU acceleration
- **SLAM**: Localization and mapping
- **Navigation**: Waypoint and geospatial navigation

### Platform

Gorai targets **Linux-based systems** including:
- x86_64 servers and workstations
- ARM64 single-board computers (Raspberry Pi, Rockchip, NVIDIA Jetson)
- Microcontrollers via TinyGo (serial bridge to Linux node)

**Raspberry Pi 5** is our reference platform for testing and verification.

### Language Strategy: Pragmatic Polyglot

**Go for core framework** (NATS orchestration, node lifecycle, configuration, web UI):
- Native concurrency perfect for robotics
- Single binary deployment
- AI-assisted coding excellent with Go's clean syntax
- Cross-compilation trivial

**Polyglot services via NATS clients** (any language can be a GoRAI service):
- **Python**: Vision (YOLO, OpenCV), ML inference (PyTorch), path planning
- **C++**: SLAM (Cartographer, ORB-SLAM), point cloud processing
- **TinyGo**: Microcontroller peripherals (motor control, safety cutoffs)

NATS has clients for 40+ languages. Use the best tool for each job.

### Container Runtime

Gorai uses **K3s with containerd** for all deployments. The RDL abstracts container orchestration—users define robots in JSON, and `gorai deploy` handles Kubernetes manifests automatically.

**Container images** are OCI-compliant and can be built with Docker, Podman, or Buildah. K3s uses containerd internally (neither Docker nor Podman daemon required on the robot).

See [specs/deployment-k3s.md](specs/deployment-k3s.md) for deployment details.

### Hardware Acceleration

| Platform | Status | Library | Notes |
|----------|--------|---------|-------|
| Hailo NPU | **Working** | Container with HailoRT | 13-26 TOPS |
| Rockchip RK3588 NPU | **Working** | [go-rknnlite](https://github.com/swdee/go-rknnlite) | 6 TOPS |
| NVIDIA CUDA | **Working** | [onnxruntime_go](https://github.com/yalue/onnxruntime_go) | Requires CUDA 12.x |
| Google Coral TPU | Planned | - | CGo bindings needed |

### Inference Runtimes

- **ONNX Runtime**: Primary path for PyTorch/TensorFlow models
- **TensorFlow Lite**: Edge inference via [tflitego](https://github.com/nbortolotti/tflitego)
- **HailoRT**: For Hailo NPU inference (via container)

## Monitoring

### Prometheus Integration

Gorai exposes metrics for Prometheus:

```
gorai_sensor_value{robot="sentinel",sensor="temperature"} 42.5
gorai_component_state{robot="sentinel",component="motor_left"} 1
gorai_messages_total{robot="sentinel",direction="sent"} 15420
```

### Logging via journald

All logs go to systemd journal:

```bash
# Follow robot logs
gorai logs --config robot.json -f

# View last 100 lines
gorai logs --config robot.json --tail 100

# Direct journalctl
journalctl --user -u my-robot.service -f
```

## Documentation

### Core Documentation
- [Object Model Explained Simply](docs/simple-object-model.md) - Beginner-friendly introduction to Resources, Components, and Services
- [Framework Specification](specs/gorai-framework-specification.md) - Complete technical specification
- [Hello Sensor Design](specs/hello-sensor-design.md) - Example design document (CPU temperature sensor)
- [Code Organization](specs/code-organization.md) - Module structure and naming conventions
- [Robot Definition Language](specs/robot-definition-language.md) - RDL v3 configuration format
- [Deployment (K3s)](specs/deployment-k3s.md) - K3s-everywhere deployment guide
- [Hardware Requirements](specs/hardware-requirements.md) - Supported platforms and specs
- [Runtime Specification](specs/runtime.md) - Robot lifecycle and behavior

### Strategic Vision
- [Vision Analysis](docs/vision-analysis.md) - **Strategic architecture assessment**: Pure Go vs. hybrid, ROS 2 positioning, containerization strategy, market positioning
- [Strategic Summary](docs/STRATEGIC-SUMMARY.md) - Quick reference for strategic decisions
- [Design Comparison](docs/general-designs.md) - Analysis of ROS 2, Viam, and YARP
- [ROS 2 Design](docs/ros2-design.md) - ROS 2 architecture summary
- [Viam Design](docs/viam-design.md) - Viam architecture summary
- [YARP Design](docs/yarp-design.md) - YARP architecture summary

### AI/ML Integration
- [Go AI Ecosystem](docs/go-ai-material.md) - ML frameworks, inference runtimes, and hardware acceleration
- [Hailo NPU Integration](plans/hailo.md) - AI inference with Hailo

### Design Document Philosophy

The [Hello Sensor Design](specs/hello-sensor-design.md) document serves as a template for Gorai component designs. This level of detail is intentional: a well-written design document is both documentation for humans and a blueprint for AI-assisted implementation. It includes:

- Architecture diagrams and component relationships
- Complete Protocol Buffer definitions
- Platform-specific implementation details
- Verification steps and expected outputs
- Test specifications

When contributing new components or services, follow this format. The specificity enables AI coding assistants to implement and test designs with minimal ambiguity.

## Example Projects

### [hello-robot](examples/hello-robot/)
Minimal NATS pub/sub messaging example. Perfect for learning the basics of Gorai messaging.

### [hello-robot-production](examples/hello-robot-production/)
Production-ready NATS messaging example with full health checks, Kubernetes probes, and resilience features. Use this as a template for real deployments.

### [hello-camera](examples/hello-camera/)
Simple camera robot demonstrating V4L2 capture and web dashboard. A minimal example to get started.

### [hello-people-detector](examples/hello-people-detector/)
Camera robot with AI-based person detection using an external service. Demonstrates the **Service RDL** pattern for modular, reusable external services running on Hailo NPU.

### [Gorai-Sentinel](projects/project-pan-tilt.md)
Pan-tilt sensor fusion platform with camera, ToF depth sensor, and servo control. Validates multi-sensor synchronization, real-time control loops, and the action/service patterns.

**Hardware**: ~$150-350 | **Complexity**: Beginner

### [Gorai-Skimmer](projects/project-simple-boat.md)
Autonomous surface vehicle for bathymetry and water monitoring. Differential thrust propulsion, GPS navigation, Open Echo sonar, and optional underwater camera/hydrophone.

**Hardware**: ~$530 | **Complexity**: Intermediate

## Migration from v2

If you have a v2 configuration with tiered deployment:

```bash
# Initialize K3s cluster (required for v3)
gorai cluster init

# Update configuration version
# Change "version": "2" to "version": "3" in robot.json

# Deploy to K3s
gorai deploy robot.json

# Remove old systemd services (if any)
sudo systemctl disable gorai-*
```

See [specs/archive/](specs/archive/) for documentation on the previous tiered deployment model.

## Project Goals

1. **Target prosumer market** — makers, citizen scientists, students, small organizations
2. **Go core framework** — with pragmatic polyglot services via NATS
3. **Cloud-native patterns** — NATS, Prometheus, JetStream (proven distributed systems)
4. **AI-assisted coding** — designed for modern AI-powered development
5. **Low barrier to entry** — productive in days, not months
6. **ROS 2 compatibility** — bridge planned (Phase 3) for ecosystem access
7. **Modular architecture** — components, services, and resources all use same interface
8. **Edge AI focus** — TPU/NPU acceleration, ONNX Runtime, TensorFlow Lite
9. **No cloud dependency** — standalone-first, cloud-optional
10. **Have fun!** — robotics should be accessible and enjoyable

## Roadmap

### Phase 1: Core Framework (Months 1-6)
- ✅ NATS-based messaging (pub/sub, request/reply, actions)
- ✅ Resource model (components, services, unified interface)
- ✅ Configuration system (JSON, hot reload)
- ✅ Basic sensors (GPS, IMU, compass) in pure Go
- ✅ Motor control (I2C, PWM) in pure Go
- ✅ Web dashboard (Go stdlib http)
- ✅ Deploy to educational kit (PiCar-X) for validation

### Phase 2: First Product - Surf (Months 6-12)
- 🔄 Marine-specific sensors (GPS, compass, depth)
- 🔄 Waypoint navigation
- 🔄 Mission planner
- 🔄 LoRa telemetry
- 🔄 Hardware design finalized
- ⏸️ ROS 2 bridge (defer to Phase 3)
- ⏸️ K3s deployment (defer to Phase 3)

### Phase 3: Ecosystem (Months 12-18)
- ⏳ ROS 2 bridge MVP (one-way: ROS2 → NATS)
- ⏳ Gazebo simulation integration
- ⏳ Drive (wheeled robot) hardware
- ⏳ Community drivers (cameras, LIDAR)
- ⏳ Podman deployment templates

### Phase 4: Advanced Features (Months 18-24)
- ⏳ SLAM integration (Cartographer via C++ wrapper)
- ⏳ K3s edge-cloud deployment
- ⏳ Fleet management dashboard
- ⏳ Advanced ML (object tracking, semantic SLAM)

**Legend**: ✅ Complete | 🔄 In Progress | ⏳ Planned | ⏸️ Deferred

## AI-Assisted Development

Gorai is built entirely with [Claude Code](https://claude.ai/claude-code). Go's clarity, strong typing, and consistent idioms make it well-suited for AI-assisted development.

We believe AI-assisted software engineering is the future, and the entire project is organized to leverage it to the maximum extent possible: detailed specifications, clear interfaces, and comprehensive documentation that both humans and AI can reason about effectively.

## License

Apache 2.0
