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


# Configure logging - get level from environment (defaults to ERROR for production)
log_level = os.environ.get("LOG_LEVEL", "ERROR").upper()
logging.basicConfig(
    level=getattr(logging, log_level, logging.ERROR),
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger("person-detector")

# Also set level for submodules
logging.getLogger("inference").setLevel(getattr(logging, log_level, logging.ERROR))
logging.getLogger("inference.hailo_backend").setLevel(getattr(logging, log_level, logging.ERROR))


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
        self._processing = False  # Guard against queue buildup
        self._frames_skipped = 0

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
            input_size=(self.settings.input_size, self.settings.input_size),
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
        # Skip frame if already processing to prevent queue buildup
        if self._processing:
            self._frames_skipped += 1
            # Log every skip at DEBUG, and every 30 skips at INFO for visibility
            if self._frames_skipped % 30 == 0:
                logger.info(f"Frame skip count: {self._frames_skipped} (still processing previous frame)")
            else:
                logger.debug(f"Skipping frame - currently processing (total skipped: {self._frames_skipped})")
            return

        self._processing = True
        frame_start = time.time()
        logger.debug(f"Processing frame {self.frame_count + 1}, skipped so far: {self._frames_skipped}")
        try:
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

            # Get JPEG image data
            jpeg_data = msg.data
            jpeg_size_kb = len(jpeg_data) / 1024

            # Run inference - returns (detections, decoded_image) to avoid double decode
            inference_start = time.time()
            raw_detections, decoded_image = await self.backend.infer(jpeg_data)
            inference_ms = (time.time() - inference_start) * 1000
            self.last_inference_ms = inference_ms

            # Post-process detections
            postprocess_start = time.time()
            detections = postprocess_detections(
                raw_detections,
                classes=self.settings.classes,
                confidence_threshold=self.settings.confidence_threshold,
            )
            postprocess_ms = (time.time() - postprocess_start) * 1000

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

            # Draw bounding boxes on image - use decoded image to avoid re-decode
            draw_start = time.time()
            if decoded_image is not None:
                annotated_jpeg = draw_bounding_boxes(
                    decoded_image,  # Pass numpy array instead of JPEG bytes
                    detections if self.settings.draw_boxes else [],
                    color=self.settings.box_color,
                    thickness=self.settings.box_thickness,
                    draw_labels=self.settings.draw_labels,
                )
            else:
                annotated_jpeg = jpeg_data
            draw_ms = (time.time() - draw_start) * 1000

            # Publish annotated image
            publish_start = time.time()
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
            publish_ms = (time.time() - publish_start) * 1000

            # Total frame time
            total_ms = (time.time() - frame_start) * 1000

            # Log every frame at DEBUG level
            logger.debug(
                f"Frame {self.frame_count}: "
                f"total={total_ms:.1f}ms "
                f"(infer={inference_ms:.1f}ms, "
                f"post={postprocess_ms:.1f}ms, "
                f"draw={draw_ms:.1f}ms, "
                f"pub={publish_ms:.1f}ms) "
                f"| {len(results)} detections "
                f"| input={jpeg_size_kb:.1f}KB"
            )

            # Log timing summary at INFO level every frame (more frequent for debugging)
            # Also log warning if inference is slow (>100ms suggests ONNX fallback)
            if inference_ms > 100:
                logger.warning(
                    f"SLOW INFERENCE Frame {self.frame_count}: {inference_ms:.1f}ms "
                    f"(likely using ONNX CPU fallback instead of Hailo NPU)"
                )

            logger.info(
                f"Frame {self.frame_count}: "
                f"total={total_ms:.1f}ms "
                f"(infer={inference_ms:.1f}ms) "
                f"| {len(results)} detections "
                f"| fps={self.fps:.1f} "
                f"| skipped={self._frames_skipped}"
            )

        except Exception as e:
            logger.error(f"Error processing frame: {e}", exc_info=True)
        finally:
            self._processing = False


async def main():
    """Main entry point."""
    # Load settings from environment
    settings = Settings.from_environment()

    logger.info("=" * 60)
    logger.info("Gorai Person Detector Service")
    logger.info("=" * 60)
    logger.info(f"Log level: {log_level}")
    logger.info(f"Robot: {settings.robot_name}")
    logger.info(f"Service: {settings.service_name}")
    logger.info(f"NATS URL: {settings.nats_url}")
    logger.info(f"Input topic: {settings.input_topic}")
    logger.info(f"Output topics: {settings.output_topic_annotated}, {settings.output_topic_detections}")
    logger.info(f"Model path: {settings.model_path}")
    logger.info(f"Confidence threshold: {settings.confidence_threshold}")
    logger.info(f"Classes: {settings.classes}")
    logger.info(f"Draw boxes: {settings.draw_boxes}")

    # Log whether Hailo is expected
    if settings.model_path.endswith(".hef"):
        logger.info("Model type: Hailo HEF (expecting Hailo NPU)")
    elif settings.model_path.endswith(".onnx"):
        logger.warning("Model type: ONNX (will use CPU - expect ~2 fps!)")
        logger.warning("For better performance, use a .hef model with Hailo NPU")
    else:
        logger.warning(f"Unknown model type: {settings.model_path}")
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
