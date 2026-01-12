# Hello Robot Production Example

A **production-ready** example demonstrating NATS pub/sub messaging with full health checks, monitoring, and resilience features.

> **🔧 Production Features:**
> - ✅ HTTP health endpoints (`/healthz` and `/readyz`)
> - ✅ Kubernetes liveness and readiness probes
> - ✅ Connection state monitoring
> - ✅ Automatic failure detection and recovery
> - ✅ Graceful shutdown handling
> - ✅ Reconnection event handlers

## Differences from Basic Hello Robot

| Feature | Basic (hello-robot) | Production (this example) |
|---------|---------------------|---------------------------|
| **Health Checks** | ❌ None | ✅ HTTP endpoints on port 8080 |
| **Kubernetes Probes** | ❌ None | ✅ Liveness + Readiness probes |
| **Connection Monitoring** | ❌ Basic | ✅ Event handlers for disconnect/reconnect |
| **Failure Detection** | ❌ Manual | ✅ Automatic via health probes |
| **Auto-Recovery** | ❌ Process exit only | ✅ Pod restart + service removal |
| **Port Exposure** | ❌ None | ✅ Port 8080 for health |
| **Use Case** | Learning/Demo | Production deployment |

## Architecture

```
hello-robot namespace (Production)
├── NATS (StatefulSet)
│   └── Message broker with health checks
├── Publisher (Deployment)
│   ├── Publishes "Hello #N" every second
│   └── Health endpoints: /healthz, /readyz (port 8080)
└── Subscriber (Deployment)
    ├── Receives and prints messages
    └── Health endpoints: /healthz, /readyz (port 8080)
```

## Health Check Endpoints

### Publisher

- **`GET /healthz`** - Liveness probe
  - Returns `200 OK` if NATS connection is healthy
  - Returns `503 Service Unavailable` if NATS is disconnected

- **`GET /readyz`** - Readiness probe
  - Returns `200 OK` if ready to publish messages
  - Returns `503 Service Unavailable` if not ready

### Subscriber

- **`GET /healthz`** - Liveness probe
  - Returns `200 OK` if NATS connection and subscription are healthy
  - Returns `503 Service Unavailable` if unhealthy

- **`GET /readyz`** - Readiness probe
  - Returns `200 OK` if ready to receive messages
  - Returns `503 Service Unavailable` if not ready

## Kubernetes Probes Configuration

### Liveness Probe
```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 15  # Wait for startup
  periodSeconds: 10        # Check every 10s
  timeoutSeconds: 5        # 5s timeout
  failureThreshold: 3      # Restart after 3 failures
```

**Purpose:** Detect if the application is alive. If it fails, Kubernetes restarts the pod.

### Readiness Probe
```yaml
readinessProbe:
  httpGet:
    path: /readyz
    port: 8080
  initialDelaySeconds: 5   # Check earlier
  periodSeconds: 5         # Check every 5s
  timeoutSeconds: 3        # 3s timeout
  failureThreshold: 2      # Remove from service after 2 failures
```

**Purpose:** Detect if the application is ready to handle traffic. If it fails, Kubernetes removes the pod from service endpoints.

## Quick Start

### Prerequisites

See [../hello-robot/SETUP.md](../hello-robot/SETUP.md) for installing Go, Podman, NATS, kubectl, and K3s/K3d.

### Local Testing (Native)

```bash
# Terminal 1: Start NATS
nats-server

# Terminal 2: Build and run publisher
make build-native
./bin/publisher

# Terminal 3: Run subscriber
./bin/subscriber

# Terminal 4: Test health endpoints
curl http://localhost:8080/healthz
# Expected: OK

curl http://localhost:8080/readyz
# Expected: Ready
```

### Local K3s/K3d Testing

```bash
# Navigate to this directory
cd gorai/examples/hello-robot-production

# Build containers
make build

# Import into k3d (macOS)
podman save localhost/hello-robot-production-publisher:latest -o /tmp/pub.tar
podman save localhost/hello-robot-production-subscriber:latest -o /tmp/sub.tar
k3d image import /tmp/pub.tar -c gorai-dev
k3d image import /tmp/sub.tar -c gorai-dev
rm /tmp/pub.tar /tmp/sub.tar

# Deploy
make deploy

# Watch pods come up
kubectl get pods -n hello-robot -w

# View logs
make logs

# Test health endpoints (in another terminal)
kubectl port-forward -n hello-robot deployment/publisher 8081:8080
kubectl port-forward -n hello-robot deployment/subscriber 8082:8080

# Then test:
curl http://localhost:8081/healthz  # Publisher health
curl http://localhost:8082/healthz  # Subscriber health
```

## Testing Health Check Behavior

### Test Liveness Probe Failure

```bash
# Kill NATS to simulate failure
kubectl scale statefulset -n hello-robot nats --replicas=0

# Watch health endpoints fail
curl http://localhost:8081/healthz
# Should return: 503 Service Unavailable

# Watch Kubernetes restart the pods
kubectl get pods -n hello-robot -w
# Pods will show CrashLoopBackOff or Restarting

# Restore NATS
kubectl scale statefulset -n hello-robot nats --replicas=1

# Health checks recover automatically
curl http://localhost:8081/healthz
# Should return: 200 OK
```

### Test Readiness Probe

```bash
# View service endpoints
kubectl get endpoints -n hello-robot

# When healthy, pods are in endpoints
# When unhealthy, pods are removed from endpoints
```

## Production Deployment Features

### 1. Automatic Failure Detection

The publisher and subscriber continuously monitor their NATS connection:

```go
// Connection state handlers
nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
    log.Printf("NATS disconnected: %v", err)
    health.setConnected(nc, false)
})

nats.ReconnectHandler(func(nc *nats.Conn) {
    log.Printf("NATS reconnected")
    health.setConnected(nc, true)
})
```

### 2. Health Endpoint Implementation

Each component exposes HTTP health endpoints:

```go
http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
    if health.isHealthy() {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("NATS not connected"))
    }
})
```

### 3. Graceful Shutdown

Applications handle termination signals gracefully:

```go
case sig := <-sigChan:
    log.Printf("Received signal %v, shutting down gracefully...", sig)
    health.setConnected(nc, false)  // Mark unhealthy
    return                           // Clean exit
```

### 4. Resource Limits

All deployments have resource requests and limits:

```yaml
resources:
  requests:
    memory: "32Mi"
    cpu: "50m"
  limits:
    memory: "64Mi"
    cpu: "100m"
```

## Monitoring and Observability

### View Health Status

```bash
# Check if pods are ready
kubectl get pods -n hello-robot
# Look for READY column: 1/1 means healthy

# Check pod events
kubectl describe pod -n hello-robot <pod-name>
# Look for Liveness/Readiness probe failures

# View service endpoints
kubectl get endpoints -n hello-robot
# Shows which pods are receiving traffic
```

### View Logs

```bash
# Publisher logs
kubectl logs -n hello-robot -l app=publisher -f

# Subscriber logs
kubectl logs -n hello-robot -l app=subscriber -f

# Look for:
# - "Connected to NATS successfully"
# - "Health endpoints available at..."
# - "NATS reconnected" (during recovery)
```

## Troubleshooting

### Pods in CrashLoopBackOff

```bash
# Check pod logs
kubectl logs -n hello-robot <pod-name>

# Check events
kubectl describe pod -n hello-robot <pod-name>

# Common causes:
# - NATS not available
# - Health check failing
# - Resource limits exceeded
```

### Health Checks Failing

```bash
# Test health endpoint manually
kubectl port-forward -n hello-robot <pod-name> 8080:8080
curl http://localhost:8080/healthz

# Check NATS connectivity
kubectl exec -n hello-robot <pod-name> -- wget -O- nats:4222
```

### Pods Not Ready

```bash
# Check readiness probe
kubectl describe pod -n hello-robot <pod-name>

# Look for:
# Readiness probe failed: HTTP probe failed

# View probe configuration
kubectl get pod -n hello-robot <pod-name> -o yaml | grep -A 10 readinessProbe
```

## Files

```
hello-robot-production/
├── publisher/
│   ├── main.go          - Publisher with health checks
│   ├── Containerfile    - Docker image with port 8080
│   └── go.mod           - Go dependencies
├── subscriber/
│   ├── main.go          - Subscriber with health checks
│   ├── Containerfile    - Docker image with port 8080
│   └── go.mod           - Go dependencies
├── deploy/
│   ├── namespace.yaml   - Kubernetes namespace
│   ├── nats.yaml        - NATS StatefulSet
│   ├── publisher.yaml   - Publisher with probes
│   └── subscriber.yaml  - Subscriber with probes
├── Makefile             - Build automation
└── README.md            - This file
```

## When to Use This Example

**Use hello-robot-production when:**
- ✅ Deploying to production environments
- ✅ Need automatic failure recovery
- ✅ Require health monitoring integration
- ✅ Want Kubernetes-native resilience
- ✅ Need to detect and handle failures gracefully

**Use basic hello-robot when:**
- ✅ Learning NATS messaging concepts
- ✅ Quick prototyping or demos
- ✅ Don't need production features
- ✅ Want minimal complexity

## Best Practices Demonstrated

1. **Separation of Health and Business Logic**
   - Health checks run in separate goroutine
   - Don't block message processing

2. **Thread-Safe Health Status**
   - Use `sync.RWMutex` for concurrent access
   - Safe from multiple goroutines

3. **Proper Probe Configuration**
   - Liveness: Longer delays, restart on failure
   - Readiness: Shorter delays, remove from service

4. **Connection Event Handling**
   - Respond to disconnect/reconnect events
   - Update health status accordingly

5. **Graceful Shutdown**
   - Handle SIGTERM/SIGINT signals
   - Mark unhealthy before exit
   - Clean up resources

6. **Resource Limits**
   - Set requests for scheduling
   - Set limits to prevent resource exhaustion

## Next Steps

- Review [../hello-robot/HEALTH-CHECKS-ANALYSIS.md](../hello-robot/HEALTH-CHECKS-ANALYSIS.md) for detailed analysis
- Adapt health check patterns to your Gorai components
- Add metrics endpoints (Prometheus) for observability
- Implement distributed tracing for debugging
- Add alerting based on health check failures

## Cleanup

```bash
# Remove deployment
make undeploy

# Verify
kubectl get namespaces | grep hello-robot
# Should return nothing
```
