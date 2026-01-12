.PHONY: help build build-native build-arm64 deploy undeploy logs clean test test-health

# Default target
help:
	@echo "Hello Robot Production Example - Makefile targets:"
	@echo ""
	@echo "  build          - Build container images for local architecture"
	@echo "  build-native   - Build native binaries (no containers)"
	@echo "  build-arm64    - Build container images for ARM64 (Raspberry Pi)"
	@echo "  deploy         - Deploy to K3s cluster"
	@echo "  undeploy       - Remove deployment from K3s"
	@echo "  logs           - View subscriber logs"
	@echo "  test-health    - Test health endpoints (requires port-forward)"
	@echo "  clean          - Clean build artifacts"
	@echo "  test           - Run local test with native binaries"
	@echo ""

# Build container images for local architecture
build:
	@echo "Building container images for local architecture..."
	podman build -t hello-robot-production-publisher:latest -f publisher/Containerfile publisher/
	podman build -t hello-robot-production-subscriber:latest -f subscriber/Containerfile subscriber/
	@echo "Done! Images built:"
	@echo "  - hello-robot-production-publisher:latest"
	@echo "  - hello-robot-production-subscriber:latest"

# Build native binaries (no containers)
build-native:
	@echo "Building native binaries..."
	@mkdir -p bin
	cd publisher && go build -o ../bin/publisher main.go
	cd subscriber && go build -o ../bin/subscriber main.go
	@echo "Done! Binaries in bin/"
	@echo "  - bin/publisher"
	@echo "  - bin/subscriber"

# Build for ARM64 (Raspberry Pi)
build-arm64:
	@echo "Building container images for ARM64..."
	podman build --platform linux/arm64 -t hello-robot-production-publisher:latest -f publisher/Containerfile publisher/
	podman build --platform linux/arm64 -t hello-robot-production-subscriber:latest -f subscriber/Containerfile subscriber/
	@echo "Done! ARM64 images built:"
	@echo "  - hello-robot-production-publisher:latest"
	@echo "  - hello-robot-production-subscriber:latest"
	@echo ""
	@echo "To push to Raspberry Pi:"
	@echo '  podman save hello-robot-production-publisher:latest | ssh pi@robot1.local sudo k3s ctr images import -'
	@echo '  podman save hello-robot-production-subscriber:latest | ssh pi@robot1.local sudo k3s ctr images import -'

# Deploy to K3s
deploy:
	@echo "Deploying to K3s..."
	kubectl apply -f deploy/
	@echo ""
	@echo "Deployment started. Watch status with:"
	@echo "  kubectl get pods -n hello-robot -w"
	@echo ""
	@echo "View logs with:"
	@echo "  make logs"
	@echo ""
	@echo "Test health endpoints with:"
	@echo "  make test-health"

# Remove deployment
undeploy:
	@echo "Removing deployment..."
	kubectl delete namespace hello-robot --ignore-not-found=true
	@echo "Done!"

# View subscriber logs
logs:
	@echo "Streaming subscriber logs (Ctrl+C to stop)..."
	kubectl logs -n hello-robot -l app=subscriber -f

# Test health endpoints
test-health:
	@echo "========================================="
	@echo "Testing Health Endpoints"
	@echo "========================================="
	@echo ""
	@echo "This requires port-forwarding to be set up."
	@echo "In another terminal, run:"
	@echo "  kubectl port-forward -n hello-robot deployment/publisher 8081:8080"
	@echo "  kubectl port-forward -n hello-robot deployment/subscriber 8082:8080"
	@echo ""
	@echo "Then test:"
	@echo "  curl http://localhost:8081/healthz  # Publisher liveness"
	@echo "  curl http://localhost:8081/readyz   # Publisher readiness"
	@echo "  curl http://localhost:8082/healthz  # Subscriber liveness"
	@echo "  curl http://localhost:8082/readyz   # Subscriber readiness"
	@echo ""
	@echo "Expected: All should return 200 OK"
	@echo "========================================="

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	@echo "Done!"

# Run local test with native binaries
test: build-native
	@echo "========================================="
	@echo "Local Test Instructions:"
	@echo "========================================="
	@echo ""
	@echo "1. In Terminal 1, start NATS:"
	@echo "   nats-server"
	@echo ""
	@echo "2. In Terminal 2, run publisher:"
	@echo "   ./bin/publisher"
	@echo ""
	@echo "3. In Terminal 3, run subscriber:"
	@echo "   ./bin/subscriber"
	@echo ""
	@echo "4. In Terminal 4, test health endpoints:"
	@echo "   curl http://localhost:8080/healthz  # Should return: OK"
	@echo "   curl http://localhost:8080/readyz   # Should return: Ready"
	@echo ""
	@echo "You should see messages being published and received."
	@echo ""
	@echo "Press Ctrl+C in each terminal to stop."
	@echo "========================================="
