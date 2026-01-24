# Quick Fix: Image Pull Issues on K3d

## Diagnostic Steps

Run these commands to understand the issue:

```bash
# 1. What images do you have in Podman?
echo "=== Images in Podman ==="
podman images | grep hello-robot

# 2. What images are in k3d?
echo "=== Images in k3d cluster ==="
docker exec k3d-gorai-dev-server-0 ctr images ls | grep hello-robot

# 3. What image is the deployment trying to use?
echo "=== Deployment image references ==="
kubectl get deployment -n hello-robot publisher -o jsonpath='{.spec.template.spec.containers[0].image}'
echo ""
kubectl get deployment -n hello-robot subscriber -o jsonpath='{.spec.template.spec.containers[0].image}'
echo ""

# 4. What's the pod status?
echo "=== Pod status ==="
kubectl get pods -n hello-robot
```

## The Fix

Based on the deployment YAML, the pods are looking for:
- `hello-robot-publisher:latest`
- `hello-robot-subscriber:latest`

### Option A: Import images with correct tags (Recommended)

```bash
# Tag images without localhost prefix to match deployment
podman tag localhost/hello-robot-publisher:latest hello-robot-publisher:latest
podman tag localhost/hello-robot-subscriber:latest hello-robot-subscriber:latest

# Verify DOCKER_HOST is set
echo $DOCKER_HOST
# Should output: unix:///Users/<username>/.local/share/containers/podman/machine/podman.sock

# Now docker command should see these images
docker images | grep hello-robot

# Import into k3d using docker command
k3d image import hello-robot-publisher:latest -c gorai-dev
k3d image import hello-robot-subscriber:latest -c gorai-dev

# Delete the old pods to force recreation with new images
kubectl delete pod -n hello-robot -l app=publisher
kubectl delete pod -n hello-robot -l app=subscriber

# Watch new pods start
kubectl get pods -n hello-robot -w
```

### Option B: Use tar files (if Option A fails)

```bash
# Export from Podman
podman save localhost/hello-robot-publisher:latest -o /tmp/publisher.tar
podman save localhost/hello-robot-subscriber:latest -o /tmp/subscriber.tar

# Import tar files into k3d
k3d image import /tmp/publisher.tar -c gorai-dev
k3d image import /tmp/subscriber.tar -c gorai-dev

# Cleanup
rm /tmp/publisher.tar /tmp/subscriber.tar

# Delete pods to force recreation
kubectl delete pod -n hello-robot -l app=publisher
kubectl delete pod -n hello-robot -l app=subscriber

# Watch pods
kubectl get pods -n hello-robot -w
```

### Option C: Update deployment to use localhost/ prefix

```bash
# Update deployment image references
kubectl set image deployment/publisher publisher=localhost/hello-robot-publisher:latest -n hello-robot
kubectl set image deployment/subscriber subscriber=localhost/hello-robot-subscriber:latest -n hello-robot

# Import with localhost prefix
k3d image import localhost/hello-robot-publisher:latest -c gorai-dev
k3d image import localhost/hello-robot-subscriber:latest -c gorai-dev

# Watch pods restart automatically
kubectl get pods -n hello-robot -w
```

## Verification

Once pods show "Running" status:

```bash
# View subscriber logs
kubectl logs -n hello-robot -l app=subscriber -f

# You should see:
# 2026/01/11 XX:XX:XX Connecting to NATS at nats://nats:4222...
# 2026/01/11 XX:XX:XX Connected to NATS successfully
# 2026/01/11 XX:XX:XX Subscribed to 'hello.messages' (Ctrl+C to stop)...
# 2026/01/11 XX:XX:XX Received: Hello #1 (count=1)
# 2026/01/11 XX:XX:XX Received: Hello #2 (count=2)
```

## If Still Failing

Check the actual error:

```bash
kubectl describe pod -n hello-robot <pod-name>
```

Look for the "Events" section at the bottom for the specific error message.
