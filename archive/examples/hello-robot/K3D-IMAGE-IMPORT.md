# K3d Image Import Guide (Podman on macOS)

This guide explains how to import locally-built Podman images into k3d.

## Prerequisites

Before importing images, verify your setup:

### 1. Verify DOCKER_HOST is set

```bash
echo $DOCKER_HOST
# Expected: unix:///Users/<username>/.local/share/containers/podman/machine/podman.sock
```

If not set:
```bash
export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock
```

### 2. Verify Podman machine is running

```bash
podman machine list
# Expected to see: Currently running
```

If not running:
```bash
podman machine start
```

### 3. Verify images exist in Podman

```bash
podman images | grep hello-robot
```

**Expected output:**
```
localhost/hello-robot-publisher    latest      <image-id>   <time>   <size>
localhost/hello-robot-subscriber   latest      <image-id>   <time>   <size>
```

Note the `localhost/` prefix - this is important.

### 4. Verify k3d can see Podman

```bash
# K3d uses the DOCKER_HOST to talk to Podman
# This should list your Podman images
docker images | grep hello-robot
```

If this doesn't show your images, DOCKER_HOST isn't configured correctly.

## Method 1: Direct Import (Preferred)

When DOCKER_HOST is set correctly, k3d can import images directly:

```bash
# Import with localhost/ prefix
k3d image import localhost/hello-robot-publisher:latest -c gorai-dev
k3d image import localhost/hello-robot-subscriber:latest -c gorai-dev
```

**Note:** Use the full image name including `localhost/` prefix as shown by `podman images`.

## Method 2: Tag Without localhost Prefix

If Method 1 doesn't work, tag the images without the localhost prefix:

```bash
# Tag images for k3d
podman tag localhost/hello-robot-publisher:latest hello-robot-publisher:latest
podman tag localhost/hello-robot-subscriber:latest hello-robot-subscriber:latest

# Verify new tags
podman images | grep hello-robot

# Import into k3d
k3d image import hello-robot-publisher:latest -c gorai-dev
k3d image import hello-robot-subscriber:latest -c gorai-dev
```

## Method 3: Export/Import via Tar (Fallback)

If both methods above fail, use tar files as an intermediate:

```bash
# Export from Podman to tar
podman save localhost/hello-robot-publisher:latest -o /tmp/publisher.tar
podman save localhost/hello-robot-subscriber:latest -o /tmp/subscriber.tar

# Import tar into k3d
k3d image import /tmp/publisher.tar -c gorai-dev
k3d image import /tmp/subscriber.tar -c gorai-dev

# Cleanup
rm /tmp/publisher.tar /tmp/subscriber.tar
```

## Verify Import

After importing, verify images are in the k3d cluster:

```bash
# List images in k3d cluster
docker exec k3d-gorai-dev-server-0 ctr images ls | grep hello-robot
```

**Expected output:**
```
localhost/hello-robot-publisher:latest    application/vnd.docker.distribution.manifest.v2+json
localhost/hello-robot-subscriber:latest   application/vnd.docker.distribution.manifest.v2+json
```

Or:
```
docker.io/library/hello-robot-publisher:latest    application/vnd.docker.distribution.manifest.v2+json
docker.io/library/hello-robot-subscriber:latest   application/vnd.docker.distribution.manifest.v2+json
```

## Restart Deployments

After importing, restart the deployments to pick up the images:

```bash
kubectl rollout restart deployment -n hello-robot publisher
kubectl rollout restart deployment -n hello-robot subscriber

# Watch pods restart
kubectl get pods -n hello-robot -w
```

Wait for status to show `Running`, then check logs:

```bash
kubectl logs -n hello-robot -l app=subscriber -f
```

## Troubleshooting

### Error: "couldn't be found in the container runtime"

This means k3d can't see the image in Podman. Try:

1. **Check DOCKER_HOST:**
   ```bash
   echo $DOCKER_HOST
   # Must point to Podman socket
   ```

2. **Check if docker command sees Podman images:**
   ```bash
   docker images
   # Should list Podman images when DOCKER_HOST is set correctly
   ```

3. **Restart Podman machine:**
   ```bash
   podman machine stop
   podman machine start
   export DOCKER_HOST=unix:///Users/$(whoami)/.local/share/containers/podman/machine/podman.sock
   ```

4. **Try with localhost/ prefix:**
   ```bash
   k3d image import localhost/hello-robot-publisher:latest -c gorai-dev
   ```

5. **Last resort - use tar method (Method 3 above)**

### Error: "image already exists"

This is fine - it means the image is already imported. Restart deployments:

```bash
kubectl rollout restart deployment -n hello-robot publisher
kubectl rollout restart deployment -n hello-robot subscriber
```

### Pods still show ImagePullBackOff

1. **Check deployment YAML uses correct image name:**
   ```bash
   kubectl get deployment -n hello-robot publisher -o yaml | grep image:
   ```

2. **Update deployment if needed:**
   Edit `deploy/publisher.yaml` and `deploy/subscriber.yaml` to match the imported image name.

3. **Redeploy:**
   ```bash
   kubectl delete namespace hello-robot
   kubectl apply -f deploy/
   ```

## Quick Reference

**Full workflow for macOS k3d:**

```bash
# 1. Build images
cd /path/to/gorai/examples/hello-robot
make build

# 2. Verify images
podman images | grep hello-robot

# 3. Import to k3d (try this first)
k3d image import localhost/hello-robot-publisher:latest -c gorai-dev
k3d image import localhost/hello-robot-subscriber:latest -c gorai-dev

# 4. Deploy
kubectl apply -f deploy/

# 5. Watch pods
kubectl get pods -n hello-robot -w

# 6. View logs when running
kubectl logs -n hello-robot -l app=subscriber -f
```
