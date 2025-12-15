#!/usr/bin/env python3
"""
Gorai Person Detector Service

An external service that subscribes to camera frames via NATS, runs YOLOX
person detection (using Hailo NPU or ONNX Runtime), draws bounding boxes,
and publishes annotated images plus detection results.

This service is configured via a Service RDL file and receives its
configuration through environment variables from the robot runtime.
"""

import asyncio
import json
import logging
import os
import signal
import sys
import time
from dataclasses import dataclass
from typing import Optional

import nats
from nats.aio.client import Client as NATS

from config.settings import Settings
from inference.hailo_backend import HailoBackend
from processing.postprocess import postprocess_detections
from annotate.draw_boxes import draw_bounding_boxes


# Configure logging
logging.basicConfig(
    level=os.environ.get("LOG_LEVEL", "INFO").upper(),
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger("person-detector")


@dataclass
class DetectionResult:
    """A single detection result."""
    class_name: str
    confidence: float
    bbox: tuple[float, float, float, float]  # x1, y1, x2, y2 normalized


class PersonDetectorService:
    """Main service class for person detection."""

    def __init__(self, settings: Settings):
        self.settings = settings
        self.nc: Optional[NATS] = None
        self.backend: Optional[HailoBackend] = None
        self.running = False
        self.frame_count = 0
        self.detection_count = 0
        self.start_time: Optional[float] = None
        self.last_inference_ms: float = 0.0
        self.fps: float = 0.0
        self._fps_frame_count = 0
        self._fps_last_time: Optional[float] = None

    async def start(self) -> None:
        """Start the service."""
        logger.info(f"Starting person detector service: {self.settings.service_name}")
        logger.info(f"Input topic: {self.settings.input_topic}")
        logger.info(f"Output topics: {self.settings.output_topic_annotated}, {self.settings.output_topic_detections}")

        # Connect to NATS
        self.nc = await nats.connect(self.settings.nats_url)
        logger.info(f"Connected to NATS at {self.settings.nats_url}")

        # Initialize inference backend
        self.backend = HailoBackend(
            model_path=self.settings.model_path,
            confidence_threshold=self.settings.confidence_threshold,
        )
        await self.backend.initialize()
        logger.info(f"Initialized Hailo backend with model: {self.settings.model_path}")

        # Subscribe to input topic
        self.running = True
        self.start_time = time.time()
        await self.nc.subscribe(
            self.settings.input_topic,
            cb=self._handle_frame,
        )
        logger.info(f"Subscribed to {self.settings.input_topic}")

        # Start heartbeat task
        logger.info(f"Starting heartbeat publisher on topic: {self.settings.heartbeat_topic}")
        heartbeat_task = asyncio.create_task(self._publish_heartbeats())

        # Keep running
        while self.running:
            await asyncio.sleep(1)

        # Cancel heartbeat task
        heartbeat_task.cancel()
        try:
            await heartbeat_task
        except asyncio.CancelledError:
            pass

    async def stop(self) -> None:
        """Stop the service gracefully."""
        logger.info("Stopping person detector service...")
        self.running = False

        if self.backend:
            await self.backend.shutdown()

        if self.nc:
            await self.nc.drain()
            await self.nc.close()

        logger.info(f"Service stopped. Processed {self.frame_count} frames, {self.detection_count} detections")

    async def _publish_heartbeats(self) -> None:
        """Publish periodic heartbeat messages for dashboard monitoring."""
        while self.running:
            try:
                uptime = time.time() - self.start_time if self.start_time else 0.0
                heartbeat = {
                    "name": self.settings.service_name,
                    "type": "service",
                    "subtype": "object_detection",
                    "status": "running",
                    "metrics": {
                        "frames_processed": self.frame_count,
                        "total_detections": self.detection_count,
                        "fps": round(self.fps, 1),
                        "inference_ms": round(self.last_inference_ms, 1),
                        "uptime_seconds": round(uptime, 1),
                    }
                }
                await self.nc.publish(
                    self.settings.heartbeat_topic,
                    json.dumps(heartbeat).encode(),
                )
                logger.info(f"Published heartbeat: fps={self.fps:.1f}, frames={self.frame_count}")
            except Exception as e:
                logger.warning(f"Failed to publish heartbeat: {e}")

            await asyncio.sleep(5)  # Publish every 5 seconds

    async def _handle_frame(self, msg) -> None:
        """Handle incoming camera frame."""
        try:
            frame_start = time.time()
            self.frame_count += 1

            # Update FPS calculation
            now = time.time()
            if self._fps_last_time is None:
                self._fps_last_time = now
                self._fps_frame_count = 0
            else:
                self._fps_frame_count += 1
                elapsed = now - self._fps_last_time
                if elapsed >= 1.0:  # Update FPS every second
                    self.fps = self._fps_frame_count / elapsed
                    self._fps_frame_count = 0
                    self._fps_last_time = now

            # Decode JPEG image
            jpeg_data = msg.data

            # Run inference
            inference_start = time.time()
            raw_detections = await self.backend.infer(jpeg_data)
            self.last_inference_ms = (time.time() - inference_start) * 1000

            # Post-process detections
            detections = postprocess_detections(
                raw_detections,
                classes=self.settings.classes,
                confidence_threshold=self.settings.confidence_threshold,
            )

            # Create detection results
            results = []
            for det in detections:
                results.append({
                    "class": det["class"],
                    "confidence": det["confidence"],
                    "bbox": {
                        "x1": det["bbox"][0],
                        "y1": det["bbox"][1],
                        "x2": det["bbox"][2],
                        "y2": det["bbox"][3],
                    }
                })
                self.detection_count += 1

            # Draw bounding boxes on image
            if self.settings.draw_boxes and detections:
                annotated_jpeg = draw_bounding_boxes(
                    jpeg_data,
                    detections,
                    color=self.settings.box_color,
                    thickness=self.settings.box_thickness,
                    draw_labels=self.settings.draw_labels,
                )
            else:
                annotated_jpeg = jpeg_data

            # Publish annotated image
            await self.nc.publish(
                self.settings.output_topic_annotated,
                annotated_jpeg,
            )

            # Publish detection results as JSON
            detection_msg = {
                "timestamp": msg.headers.get("timestamp") if msg.headers else None,
                "frame_id": self.frame_count,
                "detections": results,
            }
            await self.nc.publish(
                self.settings.output_topic_detections,
                json.dumps(detection_msg).encode(),
            )

            if self.frame_count % 100 == 0:
                logger.debug(f"Processed {self.frame_count} frames, {len(results)} detections in this frame")

        except Exception as e:
            logger.error(f"Error processing frame: {e}", exc_info=True)


async def main():
    """Main entry point."""
    # Load settings from environment
    settings = Settings.from_environment()

    logger.info("=" * 60)
    logger.info("Gorai Person Detector Service")
    logger.info("=" * 60)
    logger.info(f"Robot: {settings.robot_name}")
    logger.info(f"Service: {settings.service_name}")
    logger.info(f"Model: {settings.model_path}")
    logger.info(f"Confidence threshold: {settings.confidence_threshold}")
    logger.info(f"Classes: {settings.classes}")
    logger.info("=" * 60)

    # Create service
    service = PersonDetectorService(settings)

    # Handle shutdown signals
    loop = asyncio.get_event_loop()

    def shutdown_handler():
        logger.info("Received shutdown signal")
        asyncio.create_task(service.stop())

    for sig in (signal.SIGTERM, signal.SIGINT):
        loop.add_signal_handler(sig, shutdown_handler)

    # Start service
    try:
        await service.start()
    except Exception as e:
        logger.error(f"Service error: {e}", exc_info=True)
        await service.stop()
        sys.exit(1)


if __name__ == "__main__":
    asyncio.run(main())
