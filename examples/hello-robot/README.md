# Hello Robot Example

A minimal example demonstrating NATS pub/sub messaging between two containers in Gorai.

> **📋 New to local testing?** See [SETUP.md](SETUP.md) for installing prerequisites on Linux/macOS
>
> **📋 Ready to test?** See [TESTING.md](TESTING.md) for detailed testing instructions
>
> **🔧 Need production features?** See [../hello-robot-production](../hello-robot-production/) for a version with full health checks and monitoring

## Architecture

```
hello-robot namespace
├── NATS (StatefulSet)
│   └── Message broker on port 4222
├── Publisher (Deployment)
│   └── Publishes "Hello #N" every second to "hello.messages"
└── Subscriber (Deployment)
    └── Receives and prints messages from "hello.messages"
```

## Quick Start

### Local Testing (Native)

```bash
# Terminal 1: Start NATS
nats-server

# Terminal 2: Build and run publisher
make build-native
./bin/publisher

# Terminal 3: Run subscriber
./bin/subscriber

# You should see:
# 2025-01-11T22:00:00Z Received: Hello #1
# 2025-01-11T22:00:01Z Received: Hello #2
# ...
```

### Local K3s/K3d

```bash
# Build containers
make build

# Deploy to K3s
kubectl apply -f deploy/

# Watch logs
kubectl logs -n hello-robot -l app=subscriber -f

# Cleanup
kubectl delete namespace hello-robot
```

### Raspberry Pi Deployment

```bash
# Build for ARM64
make build-arm64

# Push to Pi
podman save hello-robot-publisher:latest | ssh pi@robot1.local sudo k3s ctr images import -
podman save hello-robot-subscriber:latest | ssh pi@robot1.local sudo k3s ctr images import -

# Deploy (from dev machine with KUBECONFIG set to Pi)
kubectl apply -f deploy/

# View logs
kubectl logs -n hello-robot -l app=subscriber -f
```

## Files

- `publisher/main.go` - Publisher application
- `subscriber/main.go` - Subscriber application
- `publisher/Containerfile` - Publisher container image
- `subscriber/Containerfile` - Subscriber container image
- `deploy/namespace.yaml` - Kubernetes namespace
- `deploy/nats.yaml` - NATS StatefulSet
- `deploy/publisher.yaml` - Publisher Deployment
- `deploy/subscriber.yaml` - Subscriber Deployment
- `Makefile` - Build automation

## Message Flow

1. Publisher connects to NATS at `nats://nats:4222`
2. Every second, publisher sends `{"text": "Hello #N", "count": N}`
3. Subscriber receives messages and prints them
4. All communication happens via NATS topic `hello.messages`
