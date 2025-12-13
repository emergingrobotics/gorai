#!/usr/bin/env python3
"""
Gorai Object Detection Service

Subscribes to camera frames from NATS, runs YOLOx inference on Hailo NPU,
and publishes annotated frames with bounding boxes and detection metadata.

Usage:
    python main.py --config /etc/gorai/robot.json

Environment variables:
    NATS_URL: NATS server URL (default: nats://localhost:4222)
    MODEL_PATH: Path to HEF model file
    CONFIDENCE_THRESHOLD: Detection confidence threshold (default: 0.5)
    LOG_LEVEL: Logging level (debug, info, warn, error)
"""

import argparse
import asyncio
import io
import json
import logging
import os
import signal
import sys
import time
from datetime import datetime, timezone
from typing import Optional

import nats
from PIL import Image

from config.settings import load_config, ServiceConfig
from inference.hailo_backend import HailoInference
from processing.postprocess import decode_yolox_output
from annotate.draw_boxes import draw_detections

# Configure logging
def setup_logging(level: str = "info") -> logging.Logger:
    """Configure logging with JSON format."""
    log_level = getattr(logging, level.upper(), logging.INFO)

    handler = logging.StreamHandler(sys.stdout)
    handler.setFormatter(logging.Formatter(
        '{"time": "%(asctime)s", "level": "%(levelname)s", "message": "%(message)s", "module": "%(module)s"}'
    ))

    logger = logging.getLogger("gorai.object_detection")
    logger.setLevel(log_level)
    logger.addHandler(handler)

    return logger


class ObjectDetectionService:
    """Main object detection service class."""

    def __init__(self, config: ServiceConfig, logger: logging.Logger):
        self.config = config
        self.logger = logger
        self.nc: Optional[nats.NATS] = None
        self.inference: Optional[HailoInference] = None
        self.running = False
        self.frame_count = 0
        self.detection_count = 0
        self.start_time = time.time()

    async def connect(self):
        """Connect to NATS server."""
        self.logger.info(f"Connecting to NATS at {self.config.nats_url}")
        self.nc = await nats.connect(
            self.config.nats_url,
            name=f"gorai-{self.config.service_name}",
            reconnect_time_wait=2,
            max_reconnect_attempts=-1,
        )
        self.logger.info("Connected to NATS")

    def initialize_inference(self):
        """Initialize Hailo inference engine."""
        self.logger.info(f"Loading model from {self.config.model_path}")
        try:
            self.inference = HailoInference(
                self.config.model_path,
                batch_size=1
            )
            self.logger.info(f"Model loaded, input shape: {self.inference.input_shape}")
        except Exception as e:
            self.logger.error(f"Failed to load model: {e}")
            raise

    async def run(self):
        """Main service loop."""
        self.running = True

        async def message_handler(msg):
            """Process incoming camera frames."""
            try:
                await self.process_frame(msg.data)
            except Exception as e:
                self.logger.error(f"Error processing frame: {e}")

        # Subscribe to camera frames
        self.logger.info(f"Subscribing to {self.config.input_topic}")
        sub = await self.nc.subscribe(
            self.config.input_topic,
            cb=message_handler
        )

        self.logger.info("Object detection service running")

        # Publish heartbeat periodically
        while self.running:
            await self.publish_heartbeat()
            await asyncio.sleep(5)

        await sub.unsubscribe()

    async def process_frame(self, jpeg_data: bytes):
        """Process a single camera frame."""
        start_time = time.time()

        # Decode JPEG
        image = Image.open(io.BytesIO(jpeg_data))
        width, height = image.size

        # Preprocess for model
        input_tensor = self.preprocess(image)

        # Run inference
        inference_start = time.time()
        outputs = self.inference.infer(input_tensor)
        inference_ms = (time.time() - inference_start) * 1000

        # Decode detections
        detections = decode_yolox_output(
            outputs,
            image_width=width,
            image_height=height,
            confidence_threshold=self.config.confidence_threshold,
            target_classes=self.config.classes
        )

        # Update counters
        self.frame_count += 1
        self.detection_count += len(detections)

        # Draw bounding boxes on image
        annotated = draw_detections(image.copy(), detections)

        # Encode annotated image
        buffer = io.BytesIO()
        annotated.save(buffer, format='JPEG', quality=80)
        annotated_bytes = buffer.getvalue()

        # Publish annotated image
        await self.nc.publish(
            self.config.output_topic_annotated,
            annotated_bytes
        )

        # Publish detection metadata
        total_ms = (time.time() - start_time) * 1000
        detection_msg = {
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "frame_id": self.frame_count,
            "inference_time_ms": round(inference_ms, 2),
            "total_time_ms": round(total_ms, 2),
            "image_width": width,
            "image_height": height,
            "detections": [d.to_dict() for d in detections]
        }
        await self.nc.publish(
            self.config.output_topic_detections,
            json.dumps(detection_msg).encode()
        )

        # Log periodically
        if self.frame_count % 30 == 0:
            elapsed = time.time() - self.start_time
            fps = self.frame_count / elapsed if elapsed > 0 else 0
            self.logger.info(
                f"Processed {self.frame_count} frames, "
                f"{self.detection_count} detections, "
                f"{fps:.1f} FPS, "
                f"{inference_ms:.1f}ms inference"
            )

    def preprocess(self, image: Image.Image):
        """Preprocess image for model input."""
        import numpy as np

        # Get model input size
        input_h, input_w = self.inference.input_shape[1:3]

        # Resize image
        resized = image.resize((input_w, input_h), Image.BILINEAR)

        # Convert to numpy array
        img_array = np.array(resized, dtype=np.float32)

        # Normalize to [0, 1]
        img_array = img_array / 255.0

        # Add batch dimension
        img_array = np.expand_dims(img_array, axis=0)

        return img_array

    async def publish_heartbeat(self):
        """Publish service heartbeat."""
        if not self.nc or self.nc.is_closed:
            return

        elapsed = time.time() - self.start_time
        fps = self.frame_count / elapsed if elapsed > 0 else 0

        heartbeat = {
            "name": self.config.service_name,
            "type": "service",
            "subtype": "object_detection",
            "status": "running",
            "timestamp": datetime.now(timezone.utc).isoformat(),
            "metrics": {
                "frames_processed": self.frame_count,
                "total_detections": self.detection_count,
                "fps": round(fps, 1),
                "uptime_seconds": round(elapsed, 0)
            }
        }

        topic = f"gorai.{self.config.robot_name}._system.heartbeat"
        try:
            await self.nc.publish(topic, json.dumps(heartbeat).encode())
        except Exception as e:
            self.logger.warning(f"Failed to publish heartbeat: {e}")

    async def stop(self):
        """Stop the service."""
        self.running = False

        if self.nc and not self.nc.is_closed:
            await self.nc.drain()
            await self.nc.close()

        self.logger.info("Service stopped")


async def main():
    """Main entry point."""
    parser = argparse.ArgumentParser(description="Gorai Object Detection Service")
    parser.add_argument(
        "--config", "-c",
        default="/etc/gorai/robot.json",
        help="Path to robot configuration file"
    )
    args = parser.parse_args()

    # Setup logging
    log_level = os.environ.get("LOG_LEVEL", "info")
    logger = setup_logging(log_level)

    logger.info("Starting Gorai Object Detection Service")

    # Load configuration
    try:
        config = load_config(args.config)
    except Exception as e:
        logger.error(f"Failed to load configuration: {e}")
        sys.exit(1)

    logger.info(f"Configuration loaded for robot: {config.robot_name}")

    # Create service
    service = ObjectDetectionService(config, logger)

    # Handle shutdown signals
    loop = asyncio.get_event_loop()

    def signal_handler():
        logger.info("Shutdown signal received")
        asyncio.create_task(service.stop())

    for sig in (signal.SIGINT, signal.SIGTERM):
        loop.add_signal_handler(sig, signal_handler)

    try:
        # Connect to NATS
        await service.connect()

        # Initialize inference engine
        service.initialize_inference()

        # Run service
        await service.run()

    except Exception as e:
        logger.error(f"Service error: {e}")
        sys.exit(1)
    finally:
        await service.stop()


if __name__ == "__main__":
    asyncio.run(main())
