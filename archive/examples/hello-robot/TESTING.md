# Hello Robot Testing Instructions

This document provides detailed step-by-step instructions for testing the hello-robot example on your host machine.

## Prerequisites

Before testing, ensure you have the following installed on your host:

- Go 1.22+ (`go version`)
- Podman (`podman --version`)
- NATS server (`nats-server --version`)
- kubectl (for K3s testing) (`kubectl version --client`)

## Test Option 1: Native Binaries with NATS Server

This is the simplest test that doesn't require containers or Kubernetes.

### Step 1: Navigate to hello-robot directory

```bash
cd /er/gorai/examples/hello-robot
```

### Step 2: Build the binaries

```bash
make build-native
```

**Expected output:**
```
Building native binaries...
cd publisher && go build -o ../bin/publisher main.go
cd subscriber && go build -o ../bin/subscriber main.go
Done! Binaries in bin/
  - bin/publisher
  - bin/subscriber
```

### Step 3: Start NATS server

Open a new terminal (Terminal 1):

```bash
nats-server
```

**Expected output:**
```
[1742] 2026/01/11 23:14:53.812311 [INF] Starting nats-server
[1742] 2026/01/11 23:14:53.814198 [INF]   Version:  2.10.7
[1742] 2026/01/11 23:14:53.814248 [INF]   Git:      [fa8464d]
[1742] 2026/01/11 23:14:53.814306 [INF]   Name:     <random-name>
[1742] 2026/01/11 23:14:53.814357 [INF]   ID:       <random-id>
[1742] 2026/01/11 23:14:53.814714 [INF] Listening for client connections on 0.0.0.0:4222
[1742] 2026/01/11 23:14:53.815173 [INF] Server is ready
```

**Keep this terminal open!**

### Step 4: Start the publisher

Open a new terminal (Terminal 2):

```bash
cd /er/gorai/examples/hello-robot
./bin/publisher
```

**Expected output:**
```
2026/01/11 23:15:00 Connecting to NATS at nats://localhost:4222...
2026/01/11 23:15:00 Connected to NATS successfully
2026/01/11 23:15:00 Starting publisher (Ctrl+C to stop)...
2026/01/11 23:15:01 Published: Hello #1
2026/01/11 23:15:02 Published: Hello #2
2026/01/11 23:15:03 Published: Hello #3
...
```

**Keep this terminal open!**

### Step 5: Start the subscriber

Open a new terminal (Terminal 3):

```bash
cd /er/gorai/examples/hello-robot
./bin/subscriber
```

**Expected output:**
```
2026/01/11 23:15:10 Connecting to NATS at nats://localhost:4222...
2026/01/11 23:15:10 Connected to NATS successfully
2026/01/11 23:15:10 Subscribed to 'hello.messages' (Ctrl+C to stop)...
2026/01/11 23:15:11 Received: Hello #11 (count=11)
2026/01/11 23:15:12 Received: Hello #12 (count=12)
2026/01/11 23:15:13 Received: Hello #13 (count=13)
...
```

### Step 6: Verify and collect logs

**Success criteria:**
- Terminal 2 (publisher) shows "Published: Hello #N" messages every second
- Terminal 3 (subscriber) shows "Received: Hello #N" messages every second
- The count numbers should match between publisher and subscriber

**To collect logs, run in a new terminal:**

```bash
# Get last 20 lines from publisher (in Terminal 4)
# Copy output from Terminal 2 directly, or if running in background:
tail -20 /path/to/publisher.log

# Get last 20 lines from subscriber
# Copy output from Terminal 3 directly, or if running in background:
tail -20 /path/to/subscriber.log
```

### Step 7: Cleanup

Press `Ctrl+C` in each terminal to stop:
1. Terminal 3 (subscriber)
2. Terminal 2 (publisher)
3. Terminal 1 (nats-server)

## Test Option 2: Podman with Local K3s

This test uses containers and requires K3s or K3d to be installed.

### Prerequisites

**Linux:**
```bash
# Install K3s
curl -sfL https://get.k3s.io | sh -s - \
  --disable traefik \
  --write-kubeconfig-mode 644

# Set KUBECONFIG
export KUBECONFIG=/etc/rancher/k3s/k3s.yaml
```

**macOS:**
```bash
# Install K3d (K3s in containers)
brew install k3d

# Create local cluster
k3d cluster create gorai-dev

# KUBECONFIG is set automatically
```

### Step 1: Navigate to hello-robot directory

```bash
cd /er/gorai/examples/hello-robot
```

### Step 2: Build container images

```bash
make build
```

**Expected output:**
```
Building container images for local architecture...
podman build -t hello-robot-publisher:latest -f publisher/Containerfile publisher/
STEP 1/10: FROM golang:1.22-alpine AS builder
...
Successfully tagged localhost/hello-robot-publisher:latest
podman build -t hello-robot-subscriber:latest -f subscriber/Containerfile subscriber/
STEP 1/10: FROM golang:1.22-alpine AS builder
...
Successfully tagged localhost/hello-robot-subscriber:latest
Done! Images built:
  - hello-robot-publisher:latest
  - hello-robot-subscriber:latest
```

### Step 3: Verify images are built

```bash
podman images | grep hello-robot
```

**Expected output:**
```
localhost/hello-robot-publisher   latest      <image-id>   <time>   <size>
localhost/hello-robot-subscriber  latest      <image-id>   <time>   <size>
```

### Step 4: Import images into K3s

> **⚠️ CRITICAL:** Images built locally must be imported into K3s/K3d before deployment. Skipping this step will cause "image can't be pulled" errors.

**Linux:**
```bash
# Import publisher image
podman save hello-robot-publisher:latest | sudo k3s ctr images import -

# Import subscriber image
podman save hello-robot-subscriber:latest | sudo k3s ctr images import -

# Verify images are imported
sudo k3s ctr images ls | grep hello-robot
```

**macOS (K3d):**
```bash
# Import images from Podman into k3d cluster
k3d image import hello-robot-publisher:latest -c gorai-dev
k3d image import hello-robot-subscriber:latest -c gorai-dev

# Verify images are imported into k3d
docker exec k3d-gorai-dev-server-0 ctr images ls | grep hello-robot
# Should show both hello-robot-publisher and hello-robot-subscriber
```

> **Important:** K3d runs K3s inside a container, creating an isolated environment. Images in your local Podman registry are NOT automatically available inside k3d. You must explicitly import them with `k3d image import`.

### Step 5: Deploy to K3s

```bash
make deploy
```

**Expected output:**
```
Deploying to K3s...
kubectl apply -f deploy/
namespace/hello-robot created
service/nats created
statefulset.apps/nats created
deployment.apps/publisher created
deployment.apps/subscriber created

Deployment started. Watch status with:
  kubectl get pods -n hello-robot -w

View logs with:
  make logs
```

### Step 6: Watch pod status

```bash
kubectl get pods -n hello-robot -w
```

**Expected output (after 30-60 seconds):**
```
NAME                          READY   STATUS    RESTARTS   AGE
nats-0                        1/1     Running   0          30s
publisher-xxxxxxxxx-xxxxx     1/1     Running   0          30s
subscriber-xxxxxxxxx-xxxxx    1/1     Running   0          30s
```

**Wait until all pods show `1/1 Running`. Press Ctrl+C to stop watching.**

### Step 7: View subscriber logs

```bash
make logs
```

Or manually:

```bash
kubectl logs -n hello-robot -l app=subscriber -f
```

**Expected output:**
```
2026/01/11 23:20:00 Connecting to NATS at nats://nats:4222...
2026/01/11 23:20:00 Connected to NATS successfully
2026/01/11 23:20:00 Subscribed to 'hello.messages' (Ctrl+C to stop)...
2026/01/11 23:20:01 Received: Hello #1 (count=1)
2026/01/11 23:20:02 Received: Hello #2 (count=2)
2026/01/11 23:20:03 Received: Hello #3 (count=3)
...
```

**Press Ctrl+C to stop following logs.**

### Step 8: View publisher logs

```bash
kubectl logs -n hello-robot -l app=publisher
```

**Expected output:**
```
2026/01/11 23:20:00 Connecting to NATS at nats://nats:4222...
2026/01/11 23:20:00 Connected to NATS successfully
2026/01/11 23:20:00 Starting publisher (Ctrl+C to stop)...
2026/01/11 23:20:01 Published: Hello #1
2026/01/11 23:20:02 Published: Hello #2
2026/01/11 23:20:03 Published: Hello #3
...
```

### Step 9: Collect logs for verification

```bash
# Get all pod names
kubectl get pods -n hello-robot

# Get last 50 lines from publisher
kubectl logs -n hello-robot -l app=publisher --tail=50

# Get last 50 lines from subscriber
kubectl logs -n hello-robot -l app=subscriber --tail=50

# Get all pod events (for debugging)
kubectl get events -n hello-robot --sort-by='.lastTimestamp'
```

### Step 10: Cleanup

```bash
make undeploy
```

**Expected output:**
```
Removing deployment...
namespace "hello-robot" deleted
Done!
```

**Verify cleanup:**
```bash
kubectl get namespaces | grep hello-robot
# Should return nothing
```

## Troubleshooting

### Native Testing Issues

**Problem: NATS server won't start**
```bash
# Check if port 4222 is already in use
netstat -an | grep 4222
# or
lsof -i :4222

# Kill existing NATS server if needed
pkill nats-server
```

**Problem: Publisher/Subscriber can't connect to NATS**
```bash
# Verify NATS is listening
netstat -an | grep 4222
# Should show: tcp        0      0 0.0.0.0:4222            0.0.0.0:*               LISTEN

# Test with NATS CLI
nats pub hello.messages "test"
nats sub hello.messages
```

### K3s Testing Issues

**Problem: Pods stuck in Pending**
```bash
# Check pod details
kubectl describe pod -n hello-robot <pod-name>

# Check node resources
kubectl describe node

# Check for image pull issues
kubectl get events -n hello-robot
```

**Problem: Pods in ImagePullBackOff or "image can't be pulled"**

This is the most common error and means images weren't imported into K3s/K3d.

**Linux (K3s):**
```bash
# Verify images are in K3s
sudo k3s ctr images ls | grep hello-robot

# If not found, import them:
podman save hello-robot-publisher:latest | sudo k3s ctr images import -
podman save hello-robot-subscriber:latest | sudo k3s ctr images import -

# After importing, restart the deployment
kubectl rollout restart deployment -n hello-robot publisher
kubectl rollout restart deployment -n hello-robot subscriber
```

**macOS (K3d):**
```bash
# Check if images are in k3d
docker exec k3d-gorai-dev-server-0 ctr images ls | grep hello-robot

# If not found, import them:
k3d image import hello-robot-publisher:latest -c gorai-dev
k3d image import hello-robot-subscriber:latest -c gorai-dev

# After importing, restart the deployment
kubectl rollout restart deployment -n hello-robot publisher
kubectl rollout restart deployment -n hello-robot subscriber

# Watch pods restart
kubectl get pods -n hello-robot -w
```

**Problem: Pods in CrashLoopBackOff**
```bash
# Check pod logs for errors
kubectl logs -n hello-robot <pod-name>

# Check pod events
kubectl describe pod -n hello-robot <pod-name>

# Common issues:
# - NATS pod not ready yet (wait 30-60 seconds)
# - Wrong NATS_URL in deployment.yaml
```

## Success Criteria

The test is successful if:

1. ✅ Binaries build without errors (`make build-native`)
2. ✅ Container images build without errors (`make build`)
3. ✅ All pods reach `Running` state in K3s
4. ✅ Publisher logs show "Published: Hello #N" every second
5. ✅ Subscriber logs show "Received: Hello #N" every second
6. ✅ Message counts match between publisher and subscriber
7. ✅ No error messages in any logs

## Collecting Logs for Bug Reports

If you encounter issues and need to report them, collect the following:

```bash
# System info
uname -a
go version
podman --version
kubectl version --client

# Native test logs
./bin/publisher 2>&1 | tee publisher.log
./bin/subscriber 2>&1 | tee subscriber.log

# K3s test logs
kubectl get pods -n hello-robot -o wide
kubectl logs -n hello-robot -l app=publisher --tail=100 > publisher-k3s.log
kubectl logs -n hello-robot -l app=subscriber --tail=100 > subscriber-k3s.log
kubectl describe pod -n hello-robot <pod-name> > pod-describe.log
kubectl get events -n hello-robot > events.log
```
