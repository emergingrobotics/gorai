# Health Checks Analysis - Hello Robot Example

## Current Implementation Status

### ✅ NATS Server - HAS Health Checks

The NATS StatefulSet (`deploy/nats.yaml`) includes proper health checks:

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8222
  initialDelaySeconds: 10
  periodSeconds: 10
readinessProbe:
  httpGet:
    path: /healthz
    port: 8222
  initialDelaySeconds: 5
  periodSeconds: 5
```

**Status**: ✅ Properly configured
- NATS server provides built-in HTTP monitoring endpoint on port 8222
- Liveness probe ensures pod is restarted if NATS crashes
- Readiness probe ensures service doesn't route traffic until NATS is ready

### ❌ Publisher - NO Health Checks

The publisher deployment (`deploy/publisher.yaml`) has:
- ✅ Resource limits
- ✅ Environment variables
- ❌ **NO** livenessProbe
- ❌ **NO** readinessProbe

**What this means:**
- If publisher crashes or hangs, Kubernetes won't automatically restart it
- If NATS connection fails, pod keeps running but doesn't work
- No way to detect if the publisher is actually functioning

### ❌ Subscriber - NO Health Checks

The subscriber deployment (`deploy/subscriber.yaml`) has:
- ✅ Resource limits
- ✅ Environment variables
- ❌ **NO** livenessProbe
- ❌ **NO** readinessProbe

**What this means:**
- Same issues as publisher
- No way to know if subscriber is receiving messages
- Silent failures if NATS connection drops

## Impact Assessment

### For a "Hello World" Example
**Current state is ACCEPTABLE** because:
- ✅ This is a minimal demonstration, not production code
- ✅ The apps are simple and unlikely to hang
- ✅ Both apps have retry logic for NATS connections (30 attempts)
- ✅ Apps will exit if NATS connection fails after retries (container restarts automatically)
- ✅ Keeps the example simple and focused on NATS messaging

### For Production Use
**Health checks SHOULD be added** because:
- ❌ No way to detect application-level failures
- ❌ No way to verify NATS connectivity is healthy
- ❌ Can't implement graceful degradation or failover
- ❌ Monitoring tools can't accurately report service health
- ❌ Load balancers can't avoid unhealthy instances

## Recommended Health Check Implementation

If we wanted to make this production-ready, here's what should be added:

### 1. Add HTTP Health Endpoints to Go Code

**publisher/main.go:**
```go
import (
    "net/http"
    // ... existing imports
)

func main() {
    // ... existing NATS connection code

    // Add health check endpoint
    http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
        if nc != nil && nc.IsConnected() {
            w.WriteHeader(http.StatusOK)
            w.Write([]byte("OK"))
        } else {
            w.WriteHeader(http.StatusServiceUnavailable)
            w.Write([]byte("NATS not connected"))
        }
    })

    // Start HTTP server in background
    go http.ListenAndServe(":8080", nil)

    // ... existing publishing loop
}
```

**subscriber/main.go:**
```go
// Similar health endpoint checking if subscription is active
http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
    if nc != nil && nc.IsConnected() && sub != nil && sub.IsValid() {
        w.WriteHeader(http.StatusOK)
        w.Write([]byte("OK"))
    } else {
        w.WriteHeader(http.StatusServiceUnavailable)
        w.Write([]byte("Not ready"))
    }
})
go http.ListenAndServe(":8080", nil)
```

### 2. Add Probes to Deployment YAMLs

**deploy/publisher.yaml:**
```yaml
containers:
- name: publisher
  image: localhost/hello-robot-publisher:latest
  imagePullPolicy: IfNotPresent
  ports:
  - containerPort: 8080
    name: health
  env:
  - name: NATS_URL
    value: "nats://nats:4222"
  livenessProbe:
    httpGet:
      path: /healthz
      port: 8080
    initialDelaySeconds: 15
    periodSeconds: 10
    timeoutSeconds: 5
    failureThreshold: 3
  readinessProbe:
    httpGet:
      path: /healthz
      port: 8080
    initialDelaySeconds: 5
    periodSeconds: 5
    timeoutSeconds: 3
    failureThreshold: 2
  resources:
    # ... existing resources
```

**deploy/subscriber.yaml:**
```yaml
# Same additions as publisher
```

### 3. Update Containerfiles

Both Containerfiles would need to expose port 8080:

```dockerfile
# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/publisher .

# Expose health check port
EXPOSE 8080

# Run as non-root user
RUN adduser -D -u 1000 app
USER app

CMD ["./publisher"]
```

## Health Check Best Practices

### Liveness Probe
- **Purpose**: Detect if the application is alive and responding
- **Action**: Kubernetes restarts the container if it fails
- **Use when**: Application could hang, deadlock, or enter unrecoverable state
- **Typical checks**: Basic HTTP endpoint, process existence

### Readiness Probe
- **Purpose**: Detect if the application is ready to handle traffic
- **Action**: Kubernetes removes pod from service endpoints if it fails
- **Use when**: Application has dependencies (database, message queue)
- **Typical checks**: Verify all dependencies are reachable

### Key Parameters

```yaml
livenessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 15    # Wait for app startup
  periodSeconds: 10          # Check every 10s
  timeoutSeconds: 5          # Probe timeout
  failureThreshold: 3        # Restart after 3 consecutive failures

readinessProbe:
  httpGet:
    path: /healthz
    port: 8080
  initialDelaySeconds: 5     # Check earlier than liveness
  periodSeconds: 5           # Check more frequently
  timeoutSeconds: 3          # Shorter timeout
  failureThreshold: 2        # Remove from service faster
```

## Recommendations

### For Current "Hello World" Example
**Keep as-is** ✅
- The example focuses on NATS messaging concepts
- Adding health checks adds complexity that distracts from the learning goal
- The built-in retry logic and process exit on failure is sufficient for demos
- Document that health checks are omitted for simplicity

### For Production Gorai Components
**Implement full health checks** ✅
- All Gorai framework components should have health endpoints
- Follow the pattern shown above for NATS connectivity checks
- Add dependency checks for critical services
- Implement graceful degradation where possible

### For Documentation
**Add note to README.md** ✅
- Explain that health checks are intentionally omitted from hello-robot
- Reference this analysis document
- Provide a link to a "production-ready" example if/when created
- Show best practices for real Gorai deployments

## Testing Health Checks

If health checks were implemented, here's how to test them:

```bash
# Port-forward to the pod
kubectl port-forward -n hello-robot deployment/publisher 8080:8080

# Test health endpoint (in another terminal)
curl http://localhost:8080/healthz
# Should return: OK

# Test by killing NATS
kubectl scale statefulset -n hello-robot nats --replicas=0

# Health check should now fail
curl http://localhost:8080/healthz
# Should return: NATS not connected (HTTP 503)

# Watch pod get restarted by liveness probe
kubectl get pods -n hello-robot -w

# Restore NATS
kubectl scale statefulset -n hello-robot nats --replicas=1
```

## Conclusion

**Current Status:**
- NATS: ✅ Has health checks
- Publisher: ❌ No health checks
- Subscriber: ❌ No health checks

**Recommendation:**
- Keep hello-robot simple without health checks (it's a learning example)
- Document that production deployments should implement them
- Create a separate "production-ready" example with full health checks
- Ensure all real Gorai framework components implement proper health checks

**Priority:** Low for hello-robot, High for Gorai framework components
