# Setup Guide: Local Testing Environment

This guide provides detailed instructions for installing all prerequisites needed to test Gorai locally on Linux or macOS.

> **⚠️ Important:** Gorai uses **Podman** (not Docker) for container operations. All instructions assume Podman is installed and configured.

## Overview

For local development and testing, you'll need:

1. **Go 1.22+** - For building native binaries
2. **Podman** - For building container images (required, not Docker)
3. **NATS Server** - For native message broker testing
4. **kubectl** - For Kubernetes cluster management
5. **K3s (Linux) or K3d (macOS)** - For local Kubernetes testing
6. **NATS CLI** (optional) - For debugging NATS messages

**Note:** On macOS, k3d requires the `DOCKER_HOST` environment variable to point to Podman's socket. This is covered in the installation steps below.

## Linux Installation

### 1. Install Go

**Ubuntu/Debian:**
```bash
# Download Go 1.22 (check https://go.dev/dl/ for latest version)
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz

# Remove any existing Go installation
sudo rm -rf /usr/local/go

# Extract to /usr/local
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify installation
go version
# Expected: go version go1.22.0 linux/amd64
```

**Fedora/RHEL:**
```bash
# Download Go
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz

# Remove existing installation
sudo rm -rf /usr/local/go

# Extract
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz

# Add to PATH
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify
go version
```

**Arch Linux:**
```bash
# Install from official repositories
sudo pacman -S go

# Verify
go version
```

### 2. Install Podman

**Ubuntu/Debian:**
```bash
# Update package list
sudo apt update

# Install Podman
sudo apt install -y podman

# Verify installation
podman --version
# Expected: podman version 3.x.x or higher

# Optional: Enable rootless Podman
echo "export DOCKER_HOST=unix:///run/user/$(id -u)/podman/podman.sock" >> ~/.bashrc
source ~/.bashrc
```

**Fedora/RHEL:**
```bash
# Install Podman (usually pre-installed on Fedora)
sudo dnf install -y podman

# Verify
podman --version
```

**Arch Linux:**
```bash
# Install Podman
sudo pacman -S podman

# Verify
podman --version
```

### 3. Install NATS Server

**All Linux Distributions:**
```bash
# Download NATS server (check https://github.com/nats-io/nats-server/releases for latest)
NATS_VERSION=2.10.7
curl -L https://github.com/nats-io/nats-server/releases/download/v${NATS_VERSION}/nats-server-v${NATS_VERSION}-linux-amd64.tar.gz -o nats-server.tar.gz

# Extract
tar -xzf nats-server.tar.gz

# Move to system bin
sudo mv nats-server-v${NATS_VERSION}-linux-amd64/nats-server /usr/local/bin/

# Cleanup
rm -rf nats-server-v${NATS_VERSION}-linux-amd64 nats-server.tar.gz

# Verify installation
nats-server --version
# Expected: nats-server: v2.10.7

# Test run (Ctrl+C to stop)
nats-server
```

### 4. Install kubectl

**All Linux Distributions:**
```bash
# Download latest stable kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"

# Make executable
chmod +x kubectl

# Move to system bin
sudo mv kubectl /usr/local/bin/

# Verify installation
kubectl version --client
# Expected: Client Version: v1.x.x
```

### 5. Install K3s (Local Kubernetes)

**All Linux Distributions:**
```bash
# Install K3s with minimal configuration
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644

# Wait for K3s to start (30-60 seconds)
sleep 30

# Verify K3s is running
sudo systemctl status k3s

# Set KUBECONFIG for current session
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml

# Add to bashrc for persistence
echo 'export KUBECONFIG=/etc/rancher/k3s/k3s.yaml' >> ~/.bashrc

# Verify cluster is ready
kubectl get nodes
# Expected:
# NAME       STATUS   ROLES                  AGE   VERSION
# hostname   Ready    control-plane,master   1m    v1.x.x+k3s1

# Check K3s pods are running
kubectl get pods -A
```

**Optional: Enable non-root kubectl access:**
```bash
# Copy K3s config to user directory
mkdir -p ~/.kube
sudo cp /etc/rancher/k3s/k3s.yaml ~/.kube/config
sudo chown $USER:$USER ~/.kube/config
chmod 600 ~/.kube/config

# Update KUBECONFIG in bashrc
sed -i 's|KUBECONFIG=/etc/rancher/k3s/k3s.yaml|KUBECONFIG=$HOME/.kube/config|' ~/.bashrc
source ~/.bashrc
```

**K3s Management Commands:**
```bash
# Start K3s
sudo systemctl start k3s

# Stop K3s
sudo systemctl stop k3s

# Restart K3s
sudo systemctl restart k3s

# Check status
sudo systemctl status k3s

# View K3s logs
sudo journalctl -u k3s -f

# Uninstall K3s (if needed)
# /usr/local/bin/k3s-uninstall.sh
```

### 6. Install NATS CLI (Optional)

**All Linux Distributions:**
```bash
# Download NATS CLI
NATS_CLI_VERSION=0.1.4
curl -L https://github.com/nats-io/natscli/releases/download/v${NATS_CLI_VERSION}/nats-${NATS_CLI_VERSION}-linux-amd64.tar.gz -o nats-cli.tar.gz

# Extract
tar -xzf nats-cli.tar.gz

# Move to system bin
sudo mv nats-${NATS_CLI_VERSION}-linux-amd64/nats /usr/local/bin/

# Cleanup
rm -rf nats-${NATS_CLI_VERSION}-linux-amd64 nats-cli.tar.gz

# Verify installation
nats --version
# Expected: nats version 0.1.4
```

---

## macOS Installation

> **⚠️ Critical Setup Requirement for macOS:**
>
> K3d requires the `DOCKER_HOST` environment variable to use Podman. This is covered in **Step 3 (Podman installation)** below and **MUST** be completed before running k3d commands. Without this, k3d will fail with "Cannot connect to the Docker daemon" errors.

### Quick Reference: DOCKER_HOST Setup (macOS)

After installing Podman (Step 3), you **MUST** run:

```bash
export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock
echo "export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock" >> ~/.zshrc
source ~/.zshrc
```

This tells k3d and other tools to use Podman instead of Docker. **This is not optional.**

### 1. Install Homebrew (if not installed)

```bash
# Install Homebrew
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Follow the on-screen instructions to add Homebrew to PATH
# Usually involves adding to ~/.zprofile or ~/.bash_profile

# Verify installation
brew --version
```

### 2. Install Go

```bash
# Install Go via Homebrew
brew install go

# Verify installation
go version
# Expected: go version go1.22.x darwin/amd64 (or darwin/arm64 on Apple Silicon)

# Add Go bin to PATH (if not already)
echo 'export PATH=$PATH:$HOME/go/bin' >> ~/.zshrc
source ~/.zshrc
```

### 3. Install Podman

```bash
# Install Podman via Homebrew
brew install podman

# Verify installation
podman --version
# Expected: podman version 4.x.x or higher

# Initialize Podman machine (required on macOS)
podman machine init

# Start Podman machine
podman machine start

# Verify Podman machine is running
podman machine list
# Expected:
# NAME                     VM TYPE     CREATED        LAST UP            CPUS        MEMORY      DISK SIZE
# podman-machine-default*  qemu        X minutes ago  Currently running  2           2GiB        100GiB

# ========================================
# CRITICAL: Set DOCKER_HOST for k3d/Podman compatibility
# ========================================
# This tells k3d and other tools to use Podman instead of Docker
export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock

# Make this permanent by adding to your shell profile
echo "export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock" >> ~/.zshrc

# Apply changes to current session
source ~/.zshrc

# Verify DOCKER_HOST is set correctly
echo $DOCKER_HOST
# Expected: unix:///Users/<your-username>/.local/share/containers/podman/machine/podman.sock

# Test Podman
podman run hello-world
```

**⚠️ IMPORTANT:** Without setting `DOCKER_HOST`, k3d will fail with "Cannot connect to the Docker daemon" errors. This step is **required**, not optional.

**Podman Machine Management:**
```bash
# Start Podman machine
podman machine start

# Stop Podman machine
podman machine stop

# Restart Podman machine
podman machine restart

# Check status
podman machine list

# SSH into machine (for debugging)
podman machine ssh
```

### 4. Install NATS Server

```bash
# Install NATS server via Homebrew
brew install nats-server

# Verify installation
nats-server --version
# Expected: nats-server: v2.10.x

# Test run (Ctrl+C to stop)
nats-server
```

### 5. Install kubectl

```bash
# Install kubectl via Homebrew
brew install kubectl

# Verify installation
kubectl version --client
# Expected: Client Version: v1.x.x
```

### 6. Install K3d (K3s in Podman)

> **Prerequisites:** Before installing k3d, you **MUST** have `DOCKER_HOST` set (from Step 3). K3d will not work with Podman without this variable.

```bash
# Install K3d via Homebrew
brew install k3d

# Verify installation
k3d --version
# Expected: k3d version v5.x.x

# ========================================
# VERIFY DOCKER_HOST IS SET (Required!)
# ========================================
echo $DOCKER_HOST
# Expected: unix:///Users/<your-username>/.local/share/containers/podman/machine/podman.sock

# If DOCKER_HOST is NOT set or empty, set it now:
# export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock
# echo "export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock" >> ~/.zshrc
# source ~/.zshrc

# Verify Podman machine is running
podman machine list
# Should show: podman-machine-default*  qemu  ...  Currently running

# If not running, start it:
podman machine start

# Create a local K3s cluster (uses Podman via DOCKER_HOST variable)
k3d cluster create gorai-dev

# If you see "Cannot connect to the Docker daemon" error, DOCKER_HOST is not set.
# Go back to Step 3 and set it, then try again.

# Verify cluster is ready
kubectl get nodes
# Expected:
# NAME                     STATUS   ROLES                  AGE   VERSION
# k3d-gorai-dev-server-0   Ready    control-plane,master   1m    v1.x.x+k3s1

# Check cluster info
kubectl cluster-info

# List all clusters
k3d cluster list
```

**K3d Management Commands:**
```bash
# Create cluster
k3d cluster create gorai-dev

# Delete cluster
k3d cluster delete gorai-dev

# Stop cluster
k3d cluster stop gorai-dev

# Start cluster
k3d cluster start gorai-dev

# List clusters
k3d cluster list

# Import local images into cluster
k3d image import <image-name> -c gorai-dev
```

### 7. Install NATS CLI (Optional)

```bash
# Install NATS CLI via Homebrew
brew install nats-io/nats-tools/nats

# Verify installation
nats --version
# Expected: nats version 0.1.x
```

---

## Verification and Testing

### Verify All Tools Are Installed

**Linux:**
```bash
echo "=== Checking Prerequisites ==="
echo -n "Go: " && go version
echo -n "Podman: " && podman --version
echo -n "NATS Server: " && nats-server --version
echo -n "kubectl: " && kubectl version --client --short
echo -n "K3s: " && sudo k3s --version
echo -n "NATS CLI: " && nats --version
echo "=== All tools installed! ==="
```

**macOS:**
```bash
echo "=== Checking Prerequisites ==="
echo -n "Go: " && go version
echo -n "Podman: " && podman --version
echo -n "NATS Server: " && nats-server --version
echo -n "kubectl: " && kubectl version --client --short
echo -n "K3d: " && k3d --version
echo -n "NATS CLI: " && nats --version
echo ""
echo "=== Checking DOCKER_HOST (Required for k3d + Podman) ==="
if [ -z "$DOCKER_HOST" ]; then
  echo "❌ DOCKER_HOST is NOT set - k3d will not work!"
  echo "   Run: export DOCKER_HOST=unix:///Users/\$(whoami)/.local/share/containers/podman/machine/podman.sock"
else
  echo "✅ DOCKER_HOST is set: $DOCKER_HOST"
fi
echo ""
echo "=== All tools installed! ==="
```

### Test NATS Server

```bash
# Terminal 1: Start NATS server
nats-server

# Terminal 2: Subscribe to test topic
nats sub test.topic

# Terminal 3: Publish test message
nats pub test.topic "Hello, NATS!"

# You should see the message in Terminal 2
# Press Ctrl+C in all terminals to stop
```

### Test Kubernetes Cluster

**Linux (K3s):**
```bash
# Check cluster is ready
kubectl get nodes

# Create test deployment
kubectl create deployment hello-k3s --image=nginx:alpine

# Wait for deployment
kubectl wait --for=condition=available --timeout=60s deployment/hello-k3s

# Check pod is running
kubectl get pods

# Cleanup
kubectl delete deployment hello-k3s
```

**macOS (K3d):**
```bash
# Check cluster is ready
kubectl get nodes

# Create test deployment
kubectl create deployment hello-k3d --image=nginx:alpine

# Wait for deployment
kubectl wait --for=condition=available --timeout=60s deployment/hello-k3d

# Check pod is running
kubectl get pods

# Cleanup
kubectl delete deployment hello-k3d
```

### Test Podman

```bash
# Test building an image
mkdir -p /tmp/podman-test
cat > /tmp/podman-test/Containerfile << 'EOF'
FROM alpine:latest
CMD ["echo", "Hello from Podman!"]
EOF

cd /tmp/podman-test
podman build -t test-image .

# Run the image
podman run --rm test-image
# Expected: Hello from Podman!

# Cleanup
podman rmi test-image
rm -rf /tmp/podman-test
```

---

## Troubleshooting

### Linux Issues

**Problem: K3s won't start**
```bash
# Check system logs
sudo journalctl -u k3s -n 50 --no-pager

# Check if port 6443 is in use
sudo netstat -tlnp | grep 6443

# Restart K3s
sudo systemctl restart k3s

# If all else fails, reinstall
/usr/local/bin/k3s-uninstall.sh
curl -sfL https://get.k3s.io | sh -s - --disable traefik --write-kubeconfig-mode 644
```

**Problem: Podman permission denied**
```bash
# Enable user namespaces
echo 'user.max_user_namespaces=28633' | sudo tee /etc/sysctl.d/userns.conf
sudo sysctl -p /etc/sysctl.d/userns.conf

# Set subuid/subgid
echo "$USER:100000:65536" | sudo tee -a /etc/subuid
echo "$USER:100000:65536" | sudo tee -a /etc/subgid

# Reboot may be required
```

**Problem: NATS server crashes**
```bash
# Check if another process is using port 4222
sudo netstat -tlnp | grep 4222

# Try starting with custom port
nats-server -p 4223
```

### macOS Issues

**Problem: Podman machine won't start**
```bash
# Check machine status
podman machine list

# Remove and recreate machine
podman machine stop
podman machine rm
podman machine init --cpus 2 --memory 4096 --disk-size 100
podman machine start

# Check logs
podman machine inspect
```

**Problem: K3d cluster won't create - "Cannot connect to the Docker daemon"**
```bash
# This happens when k3d can't find the Podman socket
# Solution: Set DOCKER_HOST environment variable

# First, verify Podman machine is running
podman machine list
# Should show "Currently running"

# If not running:
podman machine start

# Set DOCKER_HOST to point to Podman socket
export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock

# Verify the socket exists
ls -l ~/.local/share/containers/podman/machine/podman.sock

# Add to shell profile so it persists
echo "export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock" >> ~/.zshrc
source ~/.zshrc

# Now retry cluster creation
k3d cluster create gorai-dev
```

**Problem: K3d cluster won't create - other reasons**
```bash
# Make sure Podman machine is running
podman machine list

# Delete existing cluster if any
k3d cluster delete gorai-dev

# Create with more resources
k3d cluster create gorai-dev --agents 1 --servers 1

# Check k3d logs for errors
k3d cluster list
```

**Problem: Homebrew installation fails**
```bash
# Update Homebrew
brew update

# Fix permissions
sudo chown -R $(whoami) /usr/local/share/zsh /usr/local/share/zsh/site-functions

# Retry installation
brew install <package-name>
```

---

## Quick Start After Installation

Once all prerequisites are installed, test the hello-robot example:

```bash
# Clone the repository (if not already done)
git clone https://github.com/gorai/gorai.git
cd gorai/examples/hello-robot

# Follow the testing instructions
cat TESTING.md
```

---

## Next Steps

- [Quick Testing Guide](../../README.md#quick-testing-local-development) - Test Gorai locally
- [Hello Robot Testing](TESTING.md) - Detailed hello-robot testing
- [Getting Started with RPi 5](../../README.md#getting-started) - Deploy to Raspberry Pi
- [Framework Specification](../../specs/gorai-framework-specification.md) - Deep dive into Gorai

---

## Uninstallation

### Linux

```bash
# Uninstall K3s
/usr/local/bin/k3s-uninstall.sh

# Remove Podman
sudo apt remove podman  # Ubuntu/Debian
sudo dnf remove podman  # Fedora/RHEL

# Remove Go
sudo rm -rf /usr/local/go
# Remove PATH additions from ~/.bashrc manually

# Remove NATS server
sudo rm /usr/local/bin/nats-server

# Remove kubectl
sudo rm /usr/local/bin/kubectl

# Remove NATS CLI
sudo rm /usr/local/bin/nats
```

### macOS

```bash
# Delete K3d cluster
k3d cluster delete gorai-dev

# Uninstall via Homebrew
brew uninstall k3d
brew uninstall nats-server
brew uninstall nats-io/nats-tools/nats
brew uninstall kubectl
brew uninstall podman
brew uninstall go

# Remove Podman machine
podman machine stop
podman machine rm
```
