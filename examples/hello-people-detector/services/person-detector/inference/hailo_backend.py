"""
Hailo NPU inference backend for YOLOX object detection.

This backend supports:
- Hailo-8L NPU (using HailoRT)
- Fallback to ONNX Runtime for development/testing
"""

import asyncio
import logging
import os
from typing import Any, Dict, List, Optional

import numpy as np

logger = logging.getLogger(__name__)

# Try to import Hailo runtime
try:
    from hailo_platform import HEF, VDevice, ConfigureParams, InferVStreams, InputVStreamParams, OutputVStreamParams
    HAILO_AVAILABLE = True
except ImportError:
    HAILO_AVAILABLE = False
    logger.warning("Hailo runtime not available, using ONNX fallback")

# Fallback to ONNX Runtime
try:
    import onnxruntime as ort
    ONNX_AVAILABLE = True
except ImportError:
    ONNX_AVAILABLE = False


class HailoBackend:
    """Hailo NPU inference backend with ONNX fallback."""

    def __init__(
        self,
        model_path: str,
        confidence_threshold: float = 0.5,
        input_size: tuple = (640, 640),
    ):
        self.model_path = model_path
        self.confidence_threshold = confidence_threshold
        self.input_size = input_size

        self._hef = None
        self._vdevice = None
        self._network_group = None
        self._onnx_session = None
        self._use_hailo = False

    async def initialize(self) -> None:
        """Initialize the inference backend."""
        # Run in executor to avoid blocking
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self._initialize_sync)

    def _initialize_sync(self) -> None:
        """Synchronous initialization."""
        if self.model_path.endswith(".hef") and HAILO_AVAILABLE:
            self._initialize_hailo()
        elif self.model_path.endswith(".onnx") and ONNX_AVAILABLE:
            self._initialize_onnx()
        else:
            # Create mock backend for development
            logger.warning(f"No inference backend available for {self.model_path}")
            logger.warning("Using mock backend - will return empty detections")

    def _initialize_hailo(self) -> None:
        """Initialize Hailo NPU backend."""
        logger.info(f"Initializing Hailo backend with {self.model_path}")

        # Load HEF model
        self._hef = HEF(self.model_path)

        # Create virtual device
        self._vdevice = VDevice()

        # Configure network
        configure_params = ConfigureParams.create_from_hef(self._hef, interface=HailoStreamInterface.PCIE)
        self._network_group = self._vdevice.configure(self._hef, configure_params)[0]

        # Get input/output stream info
        input_vstreams_info = self._network_group.get_input_vstream_infos()
        output_vstreams_info = self._network_group.get_output_vstream_infos()

        self._input_vstream_params = InputVStreamParams.make(self._network_group, quantized=False)
        self._output_vstream_params = OutputVStreamParams.make(self._network_group, quantized=False)

        self._use_hailo = True
        logger.info("Hailo backend initialized successfully")

    def _initialize_onnx(self) -> None:
        """Initialize ONNX Runtime backend."""
        logger.info(f"Initializing ONNX backend with {self.model_path}")

        # Create ONNX Runtime session
        providers = ["CUDAExecutionProvider", "CPUExecutionProvider"]
        self._onnx_session = ort.InferenceSession(self.model_path, providers=providers)

        logger.info("ONNX backend initialized successfully")

    async def infer(self, jpeg_data: bytes) -> List[Dict[str, Any]]:
        """Run inference on JPEG image data.

        Args:
            jpeg_data: Raw JPEG bytes

        Returns:
            List of raw detection dictionaries with bbox, confidence, class_id
        """
        # Run in executor to avoid blocking
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(None, self._infer_sync, jpeg_data)

    def _infer_sync(self, jpeg_data: bytes) -> List[Dict[str, Any]]:
        """Synchronous inference."""
        # Decode JPEG to numpy array
        import cv2
        nparr = np.frombuffer(jpeg_data, np.uint8)
        image = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

        if image is None:
            logger.error("Failed to decode JPEG image")
            return []

        # Preprocess image
        input_tensor = self._preprocess(image)

        # Run inference
        if self._use_hailo:
            raw_output = self._infer_hailo(input_tensor)
        elif self._onnx_session is not None:
            raw_output = self._infer_onnx(input_tensor)
        else:
            # Mock backend - return empty detections
            return []

        # Parse raw output to detections
        detections = self._parse_yolox_output(raw_output, image.shape)

        return detections

    def _preprocess(self, image: np.ndarray) -> np.ndarray:
        """Preprocess image for YOLOX inference."""
        import cv2

        # Resize to model input size
        h, w = image.shape[:2]
        target_h, target_w = self.input_size

        # Calculate scale
        scale = min(target_w / w, target_h / h)
        new_w, new_h = int(w * scale), int(h * scale)

        # Resize
        resized = cv2.resize(image, (new_w, new_h))

        # Pad to target size
        padded = np.full((target_h, target_w, 3), 114, dtype=np.uint8)
        padded[:new_h, :new_w] = resized

        # Convert to float and normalize
        input_tensor = padded.astype(np.float32)

        # Transpose to NCHW format
        input_tensor = input_tensor.transpose(2, 0, 1)

        # Add batch dimension
        input_tensor = np.expand_dims(input_tensor, 0)

        return input_tensor

    def _infer_hailo(self, input_tensor: np.ndarray) -> np.ndarray:
        """Run inference on Hailo NPU."""
        with InferVStreams(
            self._network_group,
            self._input_vstream_params,
            self._output_vstream_params,
        ) as infer_pipeline:
            input_data = {self._hef.get_input_vstream_infos()[0].name: input_tensor}
            output_data = infer_pipeline.infer(input_data)
            return list(output_data.values())[0]

    def _infer_onnx(self, input_tensor: np.ndarray) -> np.ndarray:
        """Run inference with ONNX Runtime."""
        input_name = self._onnx_session.get_inputs()[0].name
        output_name = self._onnx_session.get_outputs()[0].name
        output = self._onnx_session.run([output_name], {input_name: input_tensor})
        return output[0]

    def _parse_yolox_output(
        self,
        output: np.ndarray,
        original_shape: tuple,
    ) -> List[Dict[str, Any]]:
        """Parse YOLOX output tensor to detection list.

        Args:
            output: Raw model output tensor
            original_shape: Original image shape (H, W, C)

        Returns:
            List of detection dicts with class_id, confidence, bbox
        """
        detections = []

        # YOLOX output format: [batch, num_detections, 85]
        # 85 = 4 (bbox) + 1 (objectness) + 80 (class scores)
        if len(output.shape) == 3:
            output = output[0]  # Remove batch dimension

        # Calculate scale factors for bbox coordinates
        orig_h, orig_w = original_shape[:2]
        target_h, target_w = self.input_size
        scale = min(target_w / orig_w, target_h / orig_h)

        for detection in output:
            # Extract components
            bbox = detection[:4]  # cx, cy, w, h
            objectness = detection[4]
            class_scores = detection[5:]

            # Calculate confidence
            class_id = np.argmax(class_scores)
            confidence = objectness * class_scores[class_id]

            if confidence < self.confidence_threshold:
                continue

            # Convert from center format to corner format
            cx, cy, w, h = bbox
            x1 = (cx - w / 2) / scale
            y1 = (cy - h / 2) / scale
            x2 = (cx + w / 2) / scale
            y2 = (cy + h / 2) / scale

            # Clip to image bounds
            x1 = max(0, min(x1, orig_w))
            y1 = max(0, min(y1, orig_h))
            x2 = max(0, min(x2, orig_w))
            y2 = max(0, min(y2, orig_h))

            # Normalize coordinates
            detections.append({
                "class_id": int(class_id),
                "confidence": float(confidence),
                "bbox": [
                    float(x1 / orig_w),
                    float(y1 / orig_h),
                    float(x2 / orig_w),
                    float(y2 / orig_h),
                ],
            })

        return detections

    async def shutdown(self) -> None:
        """Shutdown the backend."""
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self._shutdown_sync)

    def _shutdown_sync(self) -> None:
        """Synchronous shutdown."""
        if self._network_group:
            self._network_group = None
        if self._vdevice:
            self._vdevice = None
        if self._hef:
            self._hef = None
        if self._onnx_session:
            self._onnx_session = None
        logger.info("Backend shutdown complete")
