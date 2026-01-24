# K3s Installation Guide for Gorai

**Version:** 1.0
**Status:** Active
**Last Updated:** 2025-01-11

## 1. Overview

Gorai uses **K3s** (Lightweight Kubernetes) for container orchestration across all platforms. K3s provides:

- **Consistent deployment model** from single robots to multi-robot fleets
- **Container isolation** with reproducible builds
- **Cloud-native patterns** (Deployments, Services, ConfigMaps)
- **Minimal overhead** (~512MB RAM, ~50MB binary)
- **Production-grade** orchestration without complexity

### 1.1 Why K3s for Gorai

**"AI at the edge requires capable hardware. Capable hardware can run K3s."**

Modern edge AI workloads (vision, SLAM, ML inference) require 4GB+ RAM regardless of orchestration. With this baseline:

- K3s overhead (~512MB) is acceptable
- One deployment model scales from 1 to 100+ robots
- Every robot is fleet-ready from day one
- Container benefits: versioned artifacts, reproducible builds, isolated dependencies

### 1.2 What K3s Provides

| Component | Purpose | Gorai Usage |
|-----------|---------|-------------|
| **containerd** | Container runtime | Runs gorai-core, services |
| **SQLite** | Control plane database | Stores cluster state |
| **CoreDNS** | Service discovery | Internal DNS for pods |
| **Traefik** | Ingress controller | Disabled (gorai dashboard instead) |
| **ServiceLB** | Load balancer | Exposes services on host |
| **Flannel** | Pod networking | VXLAN overlay network |

---

## 2. Prerequisites

### 2.1 Hardware Requirements

| Platform | RAM | Storage | Notes |
|----------|-----|---------|-------|
| **Raspberry Pi 5 (8GB)** | 8 GB | NVMe SSD required | Primary platform |
| **Jetson Orin Nano Super** | 8 GB | NVMe SSD required | Performance platform |
| **Orange Pi 5B (8GB)** | 8 GB | 64GB+ eMMC acceptable | Budget AI platform |
| **Raspberry Pi 4 (4GB+)** | 4 GB minimum | USB SSD required | Minimum platform |
| **x86_64** | 4 GB minimum | SSD recommended | Development |

**CRITICAL: SSD or eMMC Required**

K3s uses SQLite for the control plane database. SD cards provide insufficient random I/O (10-30 IOPS) and will cause:
- Database corruption
- Pod scheduling failures
- System instability

**Required:** NVMe SSD, USB 3.0 SSD, or built-in eMMC (Orange Pi 5B).

### 2.2 Operating System Requirements

| Platform | OS | Kernel | Notes |
|----------|----|---------||-------|
| **Raspberry Pi 5** | Raspberry Pi OS 64-bit (Bookworm) | 6.1+ | Official Pi OS recommended |
| **Raspberry Pi 4** | Raspberry Pi OS 64-bit (Bookworm) | 6.1+ | Official Pi OS recommended |
| **Orange Pi 5B** | Armbian Bookworm | 5.10+ (vendor kernel) | Use Armbian server image |
| **Jetson Orin** | JetPack 6.0+ (Ubuntu 22.04) | 5.15+ | JetPack required for GPU |
| **x86_64** | Ubuntu 22.04/24.04, Fedora 39+ | 5.10+ | Any modern Linux |

### 2.3 Kernel Requirements

K3s requires:

```bash
# Check kernel version (minimum 5.10)
uname -r

# Required kernel features
# - cgroups v2
# - iptables
# - br_netfilter
# - overlay filesystem
```

Most modern distributions have these enabled by default.

---

## 3. Installation by Platform

### 3.1 Raspberry Pi 5 (Primary Platform)

**Prerequisites:**
- Raspberry Pi OS 64-bit (Bookworm or newer)
- NVMe SSD via HAT or USB 3.0 SSD
- 8GB RAM model recommended
- Internet connection for initial install

**Installation Steps:**

```bash
# 1. Update system
sudo apt update && sudo apt upgrade -y

# 2. Install K3s (single-node server)
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644

# 3. Verify installation
sudo systemctl status k3s

# 4. Check node is ready
sudo k3s kubectl get nodes

# Expected output:
# NAME    STATUS   ROLES                  AGE   VERSION
# pi5     Ready    control-plane,master   30s   v1.28.x+k3s1

# 5. Verify storage
sudo k3s kubectl get storageclass

# Expected output:
# NAME                   PROVISIONER             RECLAIMPOLICY
# local-path (default)   rancher.io/local-path   Delete
```

**Configuration:**

K3s is installed as a systemd service:
```bash
# Service status
sudo systemctl status k3s

# View logs
sudo journalctl -u k3s -f

# Restart K3s
sudo systemctl restart k3s
```

**Resource Usage:**

```bash
# Check K3s memory usage
ps aux | grep k3s

# Expected: ~300-500MB RSS
```

### 3.2 Orange Pi 5B (Budget AI Platform)

**Prerequisites:**
- Armbian Bookworm (server image recommended)
- Built-in 64GB+ eMMC
- 8GB RAM model
- Active cooling (fan required)

**Special Considerations:**

Orange Pi 5B uses a vendor-specific kernel for NPU support. Ensure kernel compatibility:

```bash
# Check kernel version
uname -r
# Should be 5.10+ (Rockchip vendor kernel)

# Check cgroups v2 support
cat /proc/filesystems | grep cgroup
# Should show: cgroup2
```

**Installation Steps:**

```bash
# 1. Update Armbian
sudo apt update && sudo apt upgrade -y

# 2. Enable cgroups v2 (if needed)
# Edit /boot/armbianEnv.txt
sudo nano /boot/armbianEnv.txt
# Add: extraargs=systemd.unified_cgroup_hierarchy=1

# Reboot if changed
sudo reboot

# 3. Install K3s
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644

# 4. Verify installation
sudo systemctl status k3s
sudo k3s kubectl get nodes
```

**NPU Access in Containers:**

For RK3588 NPU access, containers need device passthrough:

```yaml
# Deployment manifest example
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: detector
    volumeMounts:
    - name: dev-dri
      mountPath: /dev/dri
  volumes:
  - name: dev-dri
    hostPath:
      path: /dev/dri
```

### 3.3 NVIDIA Jetson Orin Nano Super (Performance Platform)

**Prerequisites:**
- JetPack 6.0+ (includes Ubuntu 22.04)
- NVMe SSD (M.2 2280)
- nvidia-container-toolkit installed

**Installation Steps:**

```bash
# 1. Verify JetPack version
cat /etc/nv_tegra_release
# Should be: R36.2 or newer (JetPack 6.0+)

# 2. Install nvidia-container-toolkit (if not present)
sudo apt update
sudo apt install -y nvidia-container-toolkit

# 3. Install K3s with NVIDIA runtime
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644 \
  --container-runtime-endpoint /var/run/containerd/containerd.sock

# 4. Configure containerd for NVIDIA runtime
sudo mkdir -p /var/lib/rancher/k3s/agent/etc/containerd/
sudo tee /var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl > /dev/null <<EOF
[plugins."io.containerd.grpc.v1.cri".containerd]
  default_runtime_name = "nvidia"

[plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia]
  runtime_type = "io.containerd.runc.v2"
  [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.nvidia.options]
    BinaryName = "/usr/bin/nvidia-container-runtime"
EOF

# 5. Restart K3s to apply containerd config
sudo systemctl restart k3s

# 6. Verify GPU access
sudo k3s kubectl run gpu-test --rm -it --restart=Never \
  --image=nvcr.io/nvidia/l4t-base:r36.2.0 -- nvidia-smi

# Expected: NVIDIA GPU information displayed
```

**GPU Device Plugin:**

For automatic GPU scheduling:

```bash
# Install NVIDIA device plugin
sudo k3s kubectl apply -f https://raw.githubusercontent.com/NVIDIA/k8s-device-plugin/v0.14.0/nvidia-device-plugin.yml

# Verify plugin running
sudo k3s kubectl get pods -n kube-system | grep nvidia
```

**Power Modes:**

Jetson power modes affect K3s performance:

```bash
# Set to MAXN SUPER (25W, 67 TOPS)
sudo nvpmodel -m 0

# Verify
nvpmodel -q
```

### 3.4 Raspberry Pi 4 (Minimum Platform)

**Prerequisites:**
- Raspberry Pi OS 64-bit (Bookworm)
- USB 3.0 SSD (blue USB port)
- 4GB or 8GB RAM model
- EEPROM updated for USB boot

**Installation Steps:**

```bash
# 1. Update EEPROM (for USB boot)
sudo rpi-eeprom-update -a
sudo reboot

# 2. Update system
sudo apt update && sudo apt upgrade -y

# 3. Install K3s
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644

# 4. Verify installation
sudo systemctl status k3s
sudo k3s kubectl get nodes
```

**Performance Notes:**

Pi 4 has limited resources:
- Enable swap for stability (2GB recommended)
- Limit concurrent pods (use resource quotas)
- Expect slower deployment times

```bash
# Add swap (if needed)
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# Make persistent
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

### 3.5 x86_64 (Development Platform)

**Prerequisites:**
- Ubuntu 22.04/24.04 or Fedora 39+
- 4GB+ RAM
- 20GB+ available storage

**Installation Steps:**

```bash
# Ubuntu/Debian
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644

# Fedora (may need firewall rules)
sudo firewall-cmd --permanent --add-port=6443/tcp
sudo firewall-cmd --permanent --add-port=10250/tcp
sudo firewall-cmd --reload

curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644

# Verify
sudo systemctl status k3s
sudo k3s kubectl get nodes
```

---

## 4. Post-Installation Configuration

### 4.1 kubectl Access

**Root access (default):**

```bash
sudo k3s kubectl get nodes
```

**Non-root access:**

```bash
# Copy kubeconfig to user directory
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $USER:$USER ~/.kube/config

# Set restrictive permissions
chmod 600 ~/.kube/config

# Now use kubectl without sudo
kubectl get nodes
```

**Install standalone kubectl (optional):**

```bash
# Ubuntu/Debian
sudo apt install -y kubectl

# Or via snap
sudo snap install kubectl --classic
```

### 4.2 Verify Installation

```bash
# 1. Check node status
kubectl get nodes

# Expected output:
# NAME     STATUS   ROLES                  AGE   VERSION
# robot1   Ready    control-plane,master   5m    v1.28.x+k3s1

# 2. Check system pods
kubectl get pods -A

# Expected: coredns, metrics-server, local-path-provisioner running

# 3. Check storage class
kubectl get storageclass

# Expected: local-path (default)

# 4. Test deployment
kubectl run nginx --image=nginx:alpine
kubectl get pods

# Expected: nginx pod running

# Cleanup
kubectl delete pod nginx
```

### 4.3 Configure Resource Limits

For robots with limited resources:

```bash
# Edit K3s config
sudo nano /etc/rancher/k3s/config.yaml
```

Add:
```yaml
# /etc/rancher/k3s/config.yaml
kubelet-arg:
  - "max-pods=30"               # Limit total pods (default 110)
  - "eviction-hard=memory.available<200Mi"  # OOM protection
  - "eviction-soft=memory.available<300Mi"
  - "eviction-soft-grace-period=memory.available=1m"
```

Restart K3s:
```bash
sudo systemctl restart k3s
```

---

## 5. Gorai Integration

### 5.1 Deploy First Robot

Once K3s is installed:

```bash
# 1. Install gorai CLI (if not installed)
curl -sfL https://get.gorai.dev | sh

# 2. Create robot configuration
cat > robot.yaml <<EOF
apiVersion: gorai.dev/v1
kind: Robot
metadata:
  name: my-robot
spec:
  components:
  - name: main_camera
    type: camera
    model: v4l2
    attributes:
      device: /dev/video0
  dashboard:
    enabled: true
EOF

# 3. Deploy to K3s
gorai deploy robot.yaml

# 4. Check deployment
kubectl get pods -n gorai-my-robot

# Expected: gorai-core, nats, dashboard pods running

# 5. Access dashboard
# Get service URL
kubectl get svc -n gorai-my-robot
# Open in browser: http://<node-ip>:<port>
```

### 5.2 View Logs

```bash
# All robot pods
kubectl logs -n gorai-my-robot --all-containers -f

# Specific pod
kubectl logs -n gorai-my-robot gorai-core-xxxxx -f
```

### 5.3 Uninstall Robot

```bash
# Remove robot deployment
gorai undeploy my-robot

# Or manually
kubectl delete namespace gorai-my-robot
```

---

## 6. Troubleshooting

### 6.1 K3s Won't Start

**Symptom:** `sudo systemctl status k3s` shows failed

**Solutions:**

```bash
# 1. Check detailed logs
sudo journalctl -u k3s -n 100 --no-pager

# 2. Check for port conflicts
sudo ss -tulpn | grep -E '6443|10250'

# 3. Check SQLite database
sudo ls -lh /var/lib/rancher/k3s/server/db/state.db

# If corrupted (0 bytes), reinstall:
sudo /usr/local/bin/k3s-uninstall.sh
# Reinstall (see platform-specific steps above)
```

### 6.2 Pods Stuck in Pending

**Symptom:** `kubectl get pods` shows pods in Pending state

**Solutions:**

```bash
# 1. Describe pod to see events
kubectl describe pod <pod-name>

# Common causes:
# - Insufficient resources
# - Image pull failure
# - Storage provisioning failure

# 2. Check node resources
kubectl describe node

# Look for: Allocatable resources vs Allocated

# 3. Check storage
kubectl get pvc  # Persistent Volume Claims
kubectl get pv   # Persistent Volumes
```

### 6.3 Container Image Pull Failures

**Symptom:** ImagePullBackOff or ErrImagePull

**Solutions:**

```bash
# 1. Check image name is correct
kubectl describe pod <pod-name> | grep -A 5 "Events:"

# 2. Pull image manually to test
sudo crictl pull <image-name>

# 3. Check containerd
sudo systemctl status containerd

# 4. Check network connectivity
ping -c 3 registry-1.docker.io
```

### 6.4 Out of Memory (OOM)

**Symptom:** Pods killed with OOMKilled status

**Solutions:**

```bash
# 1. Check memory usage
free -h

# 2. Add resource limits to deployments
# Edit deployment:
kubectl edit deployment <deployment-name>

# Add:
spec:
  containers:
  - resources:
      limits:
        memory: "512Mi"
      requests:
        memory: "256Mi"

# 3. Add swap (Raspberry Pi)
sudo fallocate -l 2G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
```

### 6.5 Orange Pi 5B Specific: cgroups v2

**Symptom:** K3s fails with "cgroups not found" error

**Solution:**

```bash
# Enable cgroups v2 in kernel boot params
sudo nano /boot/armbianEnv.txt

# Add or modify:
extraargs=systemd.unified_cgroup_hierarchy=1

# Save and reboot
sudo reboot

# Verify after reboot
cat /proc/cmdline | grep cgroup
# Should contain: systemd.unified_cgroup_hierarchy=1
```

### 6.6 Jetson: GPU Not Accessible in Pods

**Symptom:** nvidia-smi fails inside containers

**Solution:**

```bash
# 1. Verify nvidia-container-runtime is installed
dpkg -l | grep nvidia-container

# 2. Check containerd config
sudo cat /var/lib/rancher/k3s/agent/etc/containerd/config.toml.tmpl

# Should have nvidia runtime configured

# 3. Restart K3s
sudo systemctl restart k3s

# 4. Test GPU access
kubectl run gpu-test --rm -it --restart=Never \
  --image=nvcr.io/nvidia/l4t-base:r36.2.0 -- nvidia-smi
```

### 6.7 Storage Issues (SD Card)

**Symptom:** Random pod failures, database corruption

**Solution:**

**You MUST use SSD or eMMC for K3s.**

If you're on SD card:
1. Migrate to USB SSD (Raspberry Pi 4/5)
2. Use built-in eMMC (Orange Pi 5B)
3. Install NVMe SSD (Jetson, Pi 5 with HAT)

SD cards do not provide sufficient random I/O for K3s SQLite database.

---

## 7. Advanced Configuration

### 7.1 Multi-Node Cluster (Fleet)

For multi-robot fleets:

**Server node (control plane):**

```bash
# Install K3s server
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644 \
  --token my-shared-secret

# Get server URL and token
sudo cat /var/lib/rancher/k3s/server/node-token
# Save this token

# Server is at: https://<server-ip>:6443
```

**Agent nodes (robots):**

```bash
# On each robot, join the cluster
curl -sfL https://get.k3s.io | K3S_URL=https://<server-ip>:6443 \
  K3S_TOKEN=<token-from-server> sh -
```

### 7.2 Air-Gapped Installation

For offline installations:

```bash
# 1. Download K3s binary and images (on internet-connected machine)
wget https://github.com/k3s-io/k3s/releases/download/v1.28.5+k3s1/k3s-arm64
wget https://github.com/k3s-io/k3s/releases/download/v1.28.5+k3s1/k3s-airgap-images-arm64.tar.gz

# 2. Transfer files to robot

# 3. Install on robot (offline)
sudo mkdir -p /var/lib/rancher/k3s/agent/images/
sudo cp k3s-airgap-images-arm64.tar.gz /var/lib/rancher/k3s/agent/images/
sudo cp k3s-arm64 /usr/local/bin/k3s
sudo chmod +x /usr/local/bin/k3s

# 4. Run install script (won't download)
INSTALL_K3S_SKIP_DOWNLOAD=true ./install.sh
```

### 7.3 Custom Data Directory

To use alternate storage location:

```bash
# Install with custom data dir
curl -sfL https://get.k3s.io | sh -s - \
  --data-dir /mnt/nvme/k3s \
  --disable traefik \
  --write-kubeconfig-mode 644
```

---

## 8. Uninstallation

### 8.1 Server (Single-Node)

```bash
# Stop and uninstall K3s
sudo /usr/local/bin/k3s-uninstall.sh

# This removes:
# - K3s binary
# - systemd service
# - Data directory (/var/lib/rancher/k3s)
# - Container images
# - Network configuration
```

### 8.2 Agent (Multi-Node)

```bash
# On agent nodes
sudo /usr/local/bin/k3s-agent-uninstall.sh
```

### 8.3 Manual Cleanup

If uninstall script fails:

```bash
# Stop K3s
sudo systemctl stop k3s

# Remove data
sudo rm -rf /var/lib/rancher/k3s
sudo rm -rf /etc/rancher

# Remove binaries
sudo rm /usr/local/bin/k3s
sudo rm /usr/local/bin/kubectl

# Remove service
sudo rm /etc/systemd/system/k3s.service
sudo systemctl daemon-reload

# Remove network interfaces
sudo ip link delete flannel.1
sudo ip link delete cni0
```

---

## 9. Performance Benchmarks

### 9.1 Resource Overhead

| Platform | Idle RAM | Idle CPU | Disk Usage |
|----------|----------|----------|------------|
| **Raspberry Pi 5** | ~450MB | ~3% | ~1.5GB |
| **Jetson Orin** | ~500MB | ~2% | ~1.8GB |
| **Orange Pi 5B** | ~420MB | ~4% | ~1.6GB |
| **Raspberry Pi 4** | ~480MB | ~8% | ~1.5GB |

### 9.2 Deployment Times

| Operation | Pi 5 (NVMe) | Pi 4 (USB SSD) | Jetson (NVMe) |
|-----------|-------------|----------------|---------------|
| K3s startup | ~20s | ~45s | ~25s |
| Pod deployment | ~10s | ~25s | ~8s |
| Image pull (100MB) | ~30s | ~60s | ~25s |

### 9.3 Network Performance

| Test | Throughput | Latency |
|------|------------|---------|
| Pod-to-Pod (same node) | ~10 Gbps | <1ms |
| Pod-to-Service | ~8 Gbps | ~1ms |
| NATS pub/sub (in-cluster) | 100k msg/s | ~0.5ms |

---

## 10. Next Steps

After K3s installation:

1. **Deploy first robot:** See [Quick Start Guide](../README.md#quick-start)
2. **Configure hardware access:** Device passthrough for cameras, sensors
3. **Set up AI accelerators:** Hailo NPU, Coral TPU, NVIDIA GPU
4. **Join fleet (optional):** Multi-robot coordination via NATS

## 11. Support

- **Documentation:** https://gorai.dev/docs
- **K3s docs:** https://docs.k3s.io
- **Issues:** https://github.com/gorai/gorai/issues
- **Community:** https://discord.gg/gorai (planned)

---

**End of K3s Installation Guide**
