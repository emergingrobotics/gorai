# Hailo Detector Service Makefile

IMAGE_NAME := localhost/hailo-detector
IMAGE_TAG := latest
CONTAINER_NAME := hailo-detector

# Default robot configuration
ROBOT_NAME ?= hello-camera
NATS_URL ?= nats://localhost:4222
MODEL_PATH ?= /models/yolox_s_leaky.hef
CONFIDENCE ?= 0.5

.PHONY: help build run run-mock stop logs shell clean install-service

help:
	@echo "Hailo Detector Service"
	@echo ""
	@echo "Build:"
	@echo "  make build          Build container image"
	@echo "  make clean          Remove container image"
	@echo ""
	@echo "Run:"
	@echo "  make run            Run with Hailo NPU"
	@echo "  make run-mock       Run without Hailo (mock mode)"
	@echo "  make stop           Stop running container"
	@echo "  make logs           View container logs"
	@echo "  make shell          Shell into running container"
	@echo ""
	@echo "Deploy:"
	@echo "  make install-service  Install systemd service"
	@echo ""
	@echo "Configuration (environment variables):"
	@echo "  ROBOT_NAME=$(ROBOT_NAME)"
	@echo "  NATS_URL=$(NATS_URL)"
	@echo "  MODEL_PATH=$(MODEL_PATH)"
	@echo "  CONFIDENCE=$(CONFIDENCE)"

build:
	@echo "Building $(IMAGE_NAME):$(IMAGE_TAG)..."
	podman build -t $(IMAGE_NAME):$(IMAGE_TAG) .
	@echo "Done. Run with: make run"

run:
	@echo "Starting Hailo detector..."
	podman run --rm -it \
		--name $(CONTAINER_NAME) \
		--device /dev/hailo0 \
		--network host \
		-e NATS_URL=$(NATS_URL) \
		-e GORAI_ROBOT_NAME=$(ROBOT_NAME) \
		-e GORAI_SERVICE_NAME=person_detector \
		-e MODEL_PATH=$(MODEL_PATH) \
		-e CONFIDENCE_THRESHOLD=$(CONFIDENCE) \
		-v $$(pwd)/models:/models:ro \
		$(IMAGE_NAME):$(IMAGE_TAG)

run-mock:
	@echo "Starting detector in mock mode (no Hailo)..."
	podman run --rm -it \
		--name $(CONTAINER_NAME) \
		--network host \
		-e NATS_URL=$(NATS_URL) \
		-e GORAI_ROBOT_NAME=$(ROBOT_NAME) \
		-e GORAI_SERVICE_NAME=person_detector \
		-e CONFIDENCE_THRESHOLD=$(CONFIDENCE) \
		$(IMAGE_NAME):$(IMAGE_TAG)

run-dev:
	@echo "Running detector directly (development)..."
	NATS_URL=$(NATS_URL) \
	GORAI_ROBOT_NAME=$(ROBOT_NAME) \
	GORAI_SERVICE_NAME=person_detector \
	MODEL_PATH=$(MODEL_PATH) \
	CONFIDENCE_THRESHOLD=$(CONFIDENCE) \
	python3 detector.py

stop:
	@echo "Stopping $(CONTAINER_NAME)..."
	-podman stop $(CONTAINER_NAME)

logs:
	podman logs -f $(CONTAINER_NAME)

shell:
	podman exec -it $(CONTAINER_NAME) /bin/bash

clean:
	@echo "Removing $(IMAGE_NAME):$(IMAGE_TAG)..."
	-podman rmi $(IMAGE_NAME):$(IMAGE_TAG)

install-service:
	@echo "Installing systemd service..."
	sudo cp deploy/hailo-detector.service /etc/systemd/system/
	sudo systemctl daemon-reload
	@echo "Service installed. Start with: sudo systemctl start hailo-detector"

download-model:
	@echo "Downloading YOLOX model..."
	@echo "Note: You may need to download from Hailo Model Zoo manually"
	@echo "Visit: https://hailo.ai/developer-zone/model-zoo/"
	@echo ""
	@echo "For Hailo-8L (M.2), download yolox_s_leaky.hef for hailo8l"
	@mkdir -p models
	@echo "Place model file in: models/yolox_s_leaky.hef"
