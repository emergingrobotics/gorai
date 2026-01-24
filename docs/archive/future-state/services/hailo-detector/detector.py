#!/usr/bin/env python3
"""
Hailo NPU Person Detector Service

Subscribes to camera frames from NATS, runs YOLOX inference on Hailo NPU,
draws bounding boxes around detected persons, and publishes annotated frames.

Environment Variables:
    NATS_URL: NATS server URL (default: nats://localhost:4222)
    GORAI_ROBOT_NAME: Robot name for topic namespace (default: robot)
    GORAI_SERVICE_NAME: Service name (default: person_detector)
    MODEL_PATH: Path to YOLOX HEF model (default: /models/yolox_s_leaky.hef)
    CONFIDENCE_THRESHOLD: Detection confidence threshold (default: 0.5)
    INPUT_TOPIC: Override input topic (optional)
    OUTPUT_TOPIC_ANNOTATED: Override annotated output topic (optional)
    OUTPUT_TOPIC_DETECTIONS: Override detections output topic (optional)
"""

import asyncio
import json
import logging
import os
import signal
import sys
import time
from datetime import datetime, timezone
from typing import List, Dict, Any, Optional

import cv2
import numpy as np

# Configure logging
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger('hailo-detector')

# Try to import Hailo SDK - fall back to mock for development
try:
    from hailo_platform import (
        HEF, VDevice, HailoStreamInterface,
        ConfigureParams, InputVStreamParams, OutputVStreamParams,
        InferVStreams
    )
    HAILO_AVAILABLE = True
    logger.info("Hailo SDK loaded successfully")
except ImportError:
    HAILO_AVAILABLE = False
    logger.warning("Hailo SDK not available - running in mock mode")

# Import NATS
try:
    import nats
    from nats.aio.client import Client as NATSClient
except ImportError:
    logger.error("nats-py not installed. Run: pip install nats-py")
    sys.exit(1)


# Configuration from environment
NATS_URL = os.getenv("NATS_URL", "nats://localhost:4222")
ROBOT_NAME = os.getenv("GORAI_ROBOT_NAME", "robot")
SERVICE_NAME = os.getenv("GORAI_SERVICE_NAME", "person_detector")
MODEL_PATH = os.getenv("MODEL_PATH", "/models/yolox_s_leaky.hef")
CONFIDENCE_THRESHOLD = float(os.getenv("CONFIDENCE_THRESHOLD", "0.5"))
NMS_THRESHOLD = float(os.getenv("NMS_THRESHOLD", "0.45"))
INPUT_SIZE = int(os.getenv("INPUT_SIZE", "640"))

# Topic configuration
INPUT_TOPIC = os.getenv("INPUT_TOPIC", f"gorai.{ROBOT_NAME}.main_camera.data")
OUTPUT_TOPIC_ANNOTATED = os.getenv(
    "OUTPUT_TOPIC_ANNOTATED",
    f"gorai.{ROBOT_NAME}.{SERVICE_NAME}.annotated"
)
OUTPUT_TOPIC_DETECTIONS = os.getenv(
    "OUTPUT_TOPIC_DETECTIONS",
    f"gorai.{ROBOT_NAME}.{SERVICE_NAME}.detections"
)

# COCO class names (80 classes)
COCO_CLASSES = [
    "person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck",
    "boat", "traffic light", "fire hydrant", "stop sign", "parking meter", "bench",
    "bird", "cat", "dog", "horse", "sheep", "cow", "elephant", "bear", "zebra",
    "giraffe", "backpack", "umbrella", "handbag", "tie", "suitcase", "frisbee",
    "skis", "snowboard", "sports ball", "kite", "baseball bat", "baseball glove",
    "skateboard", "surfboard", "tennis racket", "bottle", "wine glass", "cup",
    "fork", "knife", "spoon", "bowl", "banana", "apple", "sandwich", "orange",
    "broccoli", "carrot", "hot dog", "pizza", "donut", "cake", "chair", "couch",
    "potted plant", "bed", "dining table", "toilet", "tv", "laptop", "mouse",
    "remote", "keyboard", "cell phone", "microwave", "oven", "toaster", "sink",
    "refrigerator", "book", "clock", "vase", "scissors", "teddy bear", "hair drier",
    "toothbrush"
]

# Only detect these classes (person = 0)
DETECT_CLASSES = {0}  # person only


class Detection:
    """Represents a single detection."""

    def __init__(self, class_id: int, confidence: float, bbox: tuple):
        self.class_id = class_id
        self.class_name = COCO_CLASSES[class_id] if class_id < len(COCO_CLASSES) else f"class_{class_id}"
        self.confidence = confidence
        self.bbox = bbox  # (x, y, width, height)

    def to_dict(self) -> Dict[str, Any]:
        return {
            "class": self.class_name,
            "class_id": self.class_id,
            "confidence": round(self.confidence, 3),
            "bbox": {
                "x": int(self.bbox[0]),
                "y": int(self.bbox[1]),
                "width": int(self.bbox[2]),
                "height": int(self.bbox[3])
            }
        }


class HailoDetector:
    """YOLOX detector using Hailo NPU."""

    def __init__(self, model_path: str, input_size: int = 640):
        self.model_path = model_path
        self.input_size = input_size
        self.device = None
        self.hef = None
        self.network_group = None
        self.input_vstream_info = None
        self.output_vstream_infos = None

        if HAILO_AVAILABLE:
            self._init_hailo()
        else:
            logger.warning("Running in mock mode - no actual inference")

    def _init_hailo(self):
        """Initialize Hailo device and load model."""
        logger.info(f"Loading model from {self.model_path}")

        # Load HEF model
        self.hef = HEF(self.model_path)

        # Get device
        self.device = VDevice()

        # Configure network
        configure_params = ConfigureParams.create_from_hef(
            self.hef,
            interface=HailoStreamInterface.PCIe
        )
        self.network_group = self.device.configure(self.hef, configure_params)[0]

        # Get stream info
        self.input_vstream_info = self.hef.get_input_vstream_infos()[0]
        self.output_vstream_infos = self.hef.get_output_vstream_infos()

        logger.info(f"Model loaded successfully")
        logger.info(f"  Input shape: {self.input_vstream_info.shape}")
        logger.info(f"  Output layers: {len(self.output_vstream_infos)}")

    def preprocess(self, frame: np.ndarray) -> tuple:
        """
        Preprocess frame for YOLOX input.

        Returns:
            tuple: (preprocessed_image, scale_factor, padding)
        """
        h, w = frame.shape[:2]

        # Calculate scale to fit input size while maintaining aspect ratio
        scale = min(self.input_size / h, self.input_size / w)
        new_h, new_w = int(h * scale), int(w * scale)

        # Resize image
        resized = cv2.resize(frame, (new_w, new_h), interpolation=cv2.INTER_LINEAR)

        # Create padded image (letterbox)
        padded = np.full((self.input_size, self.input_size, 3), 114, dtype=np.uint8)
        pad_h = (self.input_size - new_h) // 2
        pad_w = (self.input_size - new_w) // 2
        padded[pad_h:pad_h + new_h, pad_w:pad_w + new_w] = resized

        # Normalize to [0, 1] and convert to float32
        normalized = padded.astype(np.float32) / 255.0

        return normalized, scale, (pad_w, pad_h)

    def postprocess(
        self,
        outputs: Dict[str, np.ndarray],
        scale: float,
        padding: tuple,
        original_shape: tuple
    ) -> List[Detection]:
        """
        Postprocess YOLOX outputs to detections.

        YOLOX output format varies by model. This handles the standard format.
        """
        detections = []
        pad_w, pad_h = padding
        orig_h, orig_w = original_shape[:2]

        # YOLOX typically outputs [batch, num_anchors, 85] for COCO
        # 85 = 4 (bbox) + 1 (objectness) + 80 (class scores)

        for output_name, output in outputs.items():
            # Flatten if needed
            if len(output.shape) > 2:
                output = output.reshape(-1, output.shape[-1])

            for detection in output:
                if len(detection) < 6:
                    continue

                # Extract values (format depends on model export)
                # Standard YOLOX: [cx, cy, w, h, obj_conf, class_scores...]
                cx, cy, w, h = detection[:4]
                obj_conf = detection[4]
                class_scores = detection[5:]

                # Skip low objectness
                if obj_conf < CONFIDENCE_THRESHOLD:
                    continue

                # Get best class
                class_id = int(np.argmax(class_scores))
                class_conf = class_scores[class_id]

                # Combined confidence
                confidence = obj_conf * class_conf

                if confidence < CONFIDENCE_THRESHOLD:
                    continue

                # Filter to detect classes only
                if DETECT_CLASSES and class_id not in DETECT_CLASSES:
                    continue

                # Convert to corner format and remove padding/scaling
                x1 = (cx - w / 2 - pad_w) / scale
                y1 = (cy - h / 2 - pad_h) / scale
                x2 = (cx + w / 2 - pad_w) / scale
                y2 = (cy + h / 2 - pad_h) / scale

                # Clip to image bounds
                x1 = max(0, min(x1, orig_w))
                y1 = max(0, min(y1, orig_h))
                x2 = max(0, min(x2, orig_w))
                y2 = max(0, min(y2, orig_h))

                # Convert to (x, y, w, h) format
                bbox = (x1, y1, x2 - x1, y2 - y1)

                if bbox[2] > 0 and bbox[3] > 0:
                    detections.append(Detection(class_id, confidence, bbox))

        # Apply NMS
        detections = self._nms(detections)

        return detections

    def _nms(self, detections: List[Detection]) -> List[Detection]:
        """Apply Non-Maximum Suppression."""
        if not detections:
            return []

        # Convert to arrays for OpenCV NMS
        boxes = np.array([[d.bbox[0], d.bbox[1], d.bbox[2], d.bbox[3]] for d in detections])
        scores = np.array([d.confidence for d in detections])

        # OpenCV NMS
        indices = cv2.dnn.NMSBoxes(
            boxes.tolist(),
            scores.tolist(),
            CONFIDENCE_THRESHOLD,
            NMS_THRESHOLD
        )

        if len(indices) == 0:
            return []

        # Handle different OpenCV versions
        if isinstance(indices, np.ndarray):
            indices = indices.flatten()
        else:
            indices = [i[0] if isinstance(i, (list, tuple)) else i for i in indices]

        return [detections[i] for i in indices]

    def infer(self, frame: np.ndarray) -> List[Detection]:
        """Run inference on a frame."""
        if not HAILO_AVAILABLE:
            return self._mock_infer(frame)

        # Preprocess
        preprocessed, scale, padding = self.preprocess(frame)

        # Run inference
        with InferVStreams(
            self.network_group,
            self.input_vstream_info,
            self.output_vstream_infos
        ) as pipeline:
            # Add batch dimension if needed
            input_data = np.expand_dims(preprocessed, 0)
            pipeline.send(input_data)
            outputs = pipeline.recv()

        # Postprocess
        detections = self.postprocess(outputs, scale, padding, frame.shape)

        return detections

    def _mock_infer(self, frame: np.ndarray) -> List[Detection]:
        """Mock inference for testing without Hailo hardware."""
        # Return a random detection occasionally for testing
        import random
        if random.random() < 0.3:  # 30% chance of detection
            h, w = frame.shape[:2]
            x = random.randint(0, w // 2)
            y = random.randint(0, h // 2)
            det_w = random.randint(50, min(150, w - x))
            det_h = random.randint(100, min(300, h - y))
            confidence = random.uniform(0.6, 0.95)
            return [Detection(0, confidence, (x, y, det_w, det_h))]
        return []

    def close(self):
        """Release Hailo resources."""
        if self.device:
            self.device = None
            logger.info("Hailo device released")


def draw_detections(
    frame: np.ndarray,
    detections: List[Detection],
    color: tuple = (0, 255, 0),
    thickness: int = 2
) -> np.ndarray:
    """Draw bounding boxes and labels on frame."""
    annotated = frame.copy()

    for det in detections:
        x, y, w, h = [int(v) for v in det.bbox]

        # Draw rectangle
        cv2.rectangle(annotated, (x, y), (x + w, y + h), color, thickness)

        # Draw label background
        label = f"{det.class_name} {det.confidence:.2f}"
        (label_w, label_h), baseline = cv2.getTextSize(
            label, cv2.FONT_HERSHEY_SIMPLEX, 0.5, 1
        )
        cv2.rectangle(
            annotated,
            (x, y - label_h - baseline - 5),
            (x + label_w, y),
            color,
            -1
        )

        # Draw label text
        cv2.putText(
            annotated,
            label,
            (x, y - baseline - 2),
            cv2.FONT_HERSHEY_SIMPLEX,
            0.5,
            (0, 0, 0),
            1,
            cv2.LINE_AA
        )

    return annotated


class DetectorService:
    """Main detector service that connects to NATS and processes frames."""

    def __init__(self):
        self.detector = None
        self.nc: Optional[NATSClient] = None
        self.running = False
        self.frame_count = 0
        self.total_inference_time = 0.0
        self.last_stats_time = time.time()
        self.stats_interval = 10.0  # Log stats every 10 seconds

    async def start(self):
        """Start the detector service."""
        logger.info("=" * 60)
        logger.info("Hailo Person Detector Service")
        logger.info("=" * 60)
        logger.info(f"NATS URL: {NATS_URL}")
        logger.info(f"Robot: {ROBOT_NAME}")
        logger.info(f"Service: {SERVICE_NAME}")
        logger.info(f"Model: {MODEL_PATH}")
        logger.info(f"Confidence threshold: {CONFIDENCE_THRESHOLD}")
        logger.info(f"Input topic: {INPUT_TOPIC}")
        logger.info(f"Output (annotated): {OUTPUT_TOPIC_ANNOTATED}")
        logger.info(f"Output (detections): {OUTPUT_TOPIC_DETECTIONS}")
        logger.info("=" * 60)

        # Initialize detector
        logger.info("Initializing Hailo detector...")
        self.detector = HailoDetector(MODEL_PATH, INPUT_SIZE)

        # Connect to NATS
        logger.info(f"Connecting to NATS at {NATS_URL}...")
        try:
            self.nc = await nats.connect(
                NATS_URL,
                name=f"hailo-detector-{ROBOT_NAME}",
                reconnect_time_wait=1,
                max_reconnect_attempts=-1,
            )
            logger.info("Connected to NATS")
        except Exception as e:
            logger.error(f"Failed to connect to NATS: {e}")
            raise

        # Subscribe to camera frames
        logger.info(f"Subscribing to {INPUT_TOPIC}...")
        await self.nc.subscribe(INPUT_TOPIC, cb=self._handle_frame)

        self.running = True
        logger.info("Detector service started - waiting for frames...")

    async def _handle_frame(self, msg):
        """Handle incoming camera frame."""
        if not self.running:
            return

        start_time = time.time()

        try:
            # Decode JPEG
            jpeg_data = msg.data
            nparr = np.frombuffer(jpeg_data, np.uint8)
            frame = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

            if frame is None:
                logger.warning("Failed to decode frame")
                return

            # Run inference
            detections = self.detector.infer(frame)
            inference_time = (time.time() - start_time) * 1000

            # Filter to persons only and above threshold
            person_detections = [
                d for d in detections
                if d.class_id == 0 and d.confidence >= CONFIDENCE_THRESHOLD
            ]

            # Draw bounding boxes
            annotated = draw_detections(frame, person_detections)

            # Encode annotated frame to JPEG
            encode_params = [cv2.IMWRITE_JPEG_QUALITY, 80]
            _, jpeg_annotated = cv2.imencode('.jpg', annotated, encode_params)

            # Publish annotated frame
            await self.nc.publish(OUTPUT_TOPIC_ANNOTATED, jpeg_annotated.tobytes())

            # Publish detections metadata
            detection_msg = {
                "timestamp": datetime.now(timezone.utc).isoformat(),
                "frame_id": self.frame_count,
                "robot": ROBOT_NAME,
                "service": SERVICE_NAME,
                "detections": [d.to_dict() for d in person_detections],
                "inference_ms": round(inference_time, 2),
                "input_shape": list(frame.shape[:2]),
            }
            await self.nc.publish(
                OUTPUT_TOPIC_DETECTIONS,
                json.dumps(detection_msg).encode()
            )

            # Update stats
            self.frame_count += 1
            self.total_inference_time += inference_time

            # Log periodic stats
            now = time.time()
            if now - self.last_stats_time >= self.stats_interval:
                avg_time = self.total_inference_time / max(1, self.frame_count)
                fps = self.frame_count / (now - self.last_stats_time)
                logger.info(
                    f"Stats: {self.frame_count} frames, "
                    f"avg inference: {avg_time:.1f}ms, "
                    f"fps: {fps:.1f}, "
                    f"detections this frame: {len(person_detections)}"
                )
                self.frame_count = 0
                self.total_inference_time = 0.0
                self.last_stats_time = now

        except Exception as e:
            logger.error(f"Error processing frame: {e}", exc_info=True)

    async def stop(self):
        """Stop the detector service."""
        logger.info("Stopping detector service...")
        self.running = False

        if self.nc:
            await self.nc.drain()
            await self.nc.close()

        if self.detector:
            self.detector.close()

        logger.info("Detector service stopped")

    async def run(self):
        """Run the service until stopped."""
        await self.start()

        # Wait for shutdown signal
        stop_event = asyncio.Event()

        def signal_handler():
            logger.info("Received shutdown signal")
            stop_event.set()

        loop = asyncio.get_event_loop()
        for sig in (signal.SIGTERM, signal.SIGINT):
            loop.add_signal_handler(sig, signal_handler)

        await stop_event.wait()
        await self.stop()


async def main():
    """Main entry point."""
    service = DetectorService()
    await service.run()


if __name__ == "__main__":
    try:
        asyncio.run(main())
    except KeyboardInterrupt:
        logger.info("Interrupted by user")
    except Exception as e:
        logger.error(f"Fatal error: {e}", exc_info=True)
        sys.exit(1)
