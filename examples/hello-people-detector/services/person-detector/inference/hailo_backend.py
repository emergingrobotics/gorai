"""
Hailo NPU inference backend for YOLOv8 object detection.

This backend supports:
- Hailo-8 NPU (using HailoRT) - ~50 fps for YOLOv8s
- Fallback to ONNX Runtime for development/testing - ~2 fps
"""

import asyncio
import logging
import os
from typing import Any, Dict, List, Optional

import numpy as np

logger = logging.getLogger(__name__)

# Try to import Hailo runtime
HAILO_AVAILABLE = False
try:
    from hailo_platform import HEF, VDevice, FormatType
    from hailo_platform.pyhailort import InferVStreams, ConfigureParams
    from hailo_platform.pyhailort import InputVStreamParams, OutputVStreamParams
    from hailo_platform.pyhailort import HailoStreamInterface
    HAILO_AVAILABLE = True
    logger.info("Hailo runtime AVAILABLE - HailoRT imported successfully")
except ImportError as e:
    logger.warning("=" * 50)
    logger.warning("HAILO RUNTIME NOT AVAILABLE")
    logger.warning(f"Import error: {e}")
    logger.warning("This means we will fall back to ONNX CPU inference")
    logger.warning("Expected performance: ~2 fps (vs ~50 fps with Hailo)")
    logger.warning("To fix: ensure HailoRT is installed and /dev/hailo0 is accessible")
    logger.warning("=" * 50)

# Fallback to ONNX Runtime
ONNX_AVAILABLE = False
try:
    import onnxruntime as ort
    ONNX_AVAILABLE = True
    logger.info("ONNX Runtime available as fallback")
except ImportError:
    logger.warning("ONNX Runtime not available - no fallback inference possible")


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
        self._network_group_params = None
        self._input_vstream_info = None
        self._output_vstream_info = None
        self._onnx_session = None
        self._use_hailo = False

    async def initialize(self) -> None:
        """Initialize the inference backend."""
        loop = asyncio.get_event_loop()
        await loop.run_in_executor(None, self._initialize_sync)

    def _initialize_sync(self) -> None:
        """Synchronous initialization."""
        logger.info("=" * 50)
        logger.info("INITIALIZING INFERENCE BACKEND")
        logger.info(f"Model path: {self.model_path}")
        logger.info(f"Model exists: {os.path.exists(self.model_path)}")
        logger.info(f"Hailo available: {HAILO_AVAILABLE}")
        logger.info(f"ONNX available: {ONNX_AVAILABLE}")
        logger.info("=" * 50)

        if self.model_path.endswith(".hef") and HAILO_AVAILABLE:
            try:
                logger.info("Attempting Hailo NPU initialization...")
                self._initialize_hailo()
                logger.info("SUCCESS: Using Hailo NPU backend (~50 fps expected)")
            except Exception as e:
                logger.error(f"Failed to initialize Hailo: {e}")
                logger.warning("Falling back to mock backend")
                logger.warning("NO INFERENCE WILL BE PERFORMED")
        elif self.model_path.endswith(".onnx") and ONNX_AVAILABLE:
            logger.warning("=" * 50)
            logger.warning("USING ONNX CPU BACKEND - SLOW PERFORMANCE EXPECTED!")
            logger.warning("Expected: ~2 fps (model file is .onnx)")
            logger.warning("For better performance, use a .hef model with Hailo NPU")
            logger.warning("=" * 50)
            self._initialize_onnx()
        elif ONNX_AVAILABLE:
            # Try ONNX fallback with .onnx extension
            onnx_path = self.model_path.replace(".hef", ".onnx")
            if os.path.exists(onnx_path):
                logger.warning("=" * 50)
                logger.warning("FALLING BACK TO ONNX CPU BACKEND")
                logger.warning(f"HEF model requested but using ONNX: {onnx_path}")
                logger.warning("Expected: ~2 fps")
                logger.warning("=" * 50)
                self.model_path = onnx_path
                self._initialize_onnx()
            else:
                logger.error(f"No inference backend available for {self.model_path}")
                logger.error(f"Tried ONNX fallback at: {onnx_path} - not found")
                logger.warning("Using mock backend - will return empty detections")
        else:
            logger.error(f"No inference backend available for {self.model_path}")
            logger.warning("Using mock backend - will return empty detections")

    def _initialize_hailo(self) -> None:
        """Initialize Hailo NPU backend."""
        logger.info(f"Initializing Hailo backend with {self.model_path}")

        # Load HEF model
        self._hef = HEF(self.model_path)
        logger.info(f"Loaded HEF model: {self.model_path}")

        # Create virtual device
        self._vdevice = VDevice()
        logger.info("Created VDevice")

        # Configure network group
        configure_params = ConfigureParams.create_from_hef(
            self._hef,
            interface=HailoStreamInterface.PCIe
        )
        network_groups = self._vdevice.configure(self._hef, configure_params)
        self._network_group = network_groups[0]
        logger.info(f"Configured network group: {self._network_group.name}")

        # Get input/output stream info
        input_vstreams_info = self._hef.get_input_vstream_infos()
        output_vstreams_info = self._hef.get_output_vstream_infos()

        if input_vstreams_info:
            self._input_vstream_info = input_vstreams_info[0]
            logger.info(f"Input: {self._input_vstream_info.name}, "
                       f"shape: {self._input_vstream_info.shape}, "
                       f"format: {self._input_vstream_info.format.type}")

        if output_vstreams_info:
            self._output_vstream_info = output_vstreams_info[0]
            logger.info(f"Output: {self._output_vstream_info.name}, "
                       f"shape: {self._output_vstream_info.shape}")

        # Store params for inference
        self._input_params = InputVStreamParams.make(
            self._network_group,
            format_type=FormatType.FLOAT32
        )
        self._output_params = OutputVStreamParams.make(
            self._network_group,
            format_type=FormatType.FLOAT32
        )

        self._use_hailo = True
        logger.info("Hailo backend initialized successfully")

    def _initialize_onnx(self) -> None:
        """Initialize ONNX Runtime backend."""
        logger.info(f"Initializing ONNX backend with {self.model_path}")

        # Create ONNX Runtime session
        providers = ["CPUExecutionProvider"]  # ARM64 typically CPU only
        self._onnx_session = ort.InferenceSession(self.model_path, providers=providers)

        input_info = self._onnx_session.get_inputs()[0]
        output_info = self._onnx_session.get_outputs()[0]
        logger.info(f"Input: {input_info.name}, shape: {input_info.shape}")
        logger.info(f"Output: {output_info.name}, shape: {output_info.shape}")
        logger.info("ONNX backend initialized successfully (CPU - expect ~2 fps)")

    async def infer(self, jpeg_data: bytes) -> tuple[List[Dict[str, Any]], Optional[np.ndarray]]:
        """Run inference on JPEG image data.

        Args:
            jpeg_data: Raw JPEG bytes

        Returns:
            Tuple of (detections list, decoded image array)
        """
        loop = asyncio.get_event_loop()
        return await loop.run_in_executor(None, self._infer_sync, jpeg_data)

    def _infer_sync(self, jpeg_data: bytes) -> tuple[List[Dict[str, Any]], Optional[np.ndarray]]:
        """Synchronous inference."""
        import cv2

        # Decode JPEG to numpy array
        nparr = np.frombuffer(jpeg_data, np.uint8)
        image = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

        if image is None:
            logger.error("Failed to decode JPEG image")
            return [], None

        # Preprocess image
        input_tensor = self._preprocess(image)

        # Run inference
        if self._use_hailo:
            raw_output = self._infer_hailo(input_tensor)
        elif self._onnx_session is not None:
            raw_output = self._infer_onnx(input_tensor)
        else:
            return [], image

        # Parse raw output to detections
        detections = self._parse_yolo_output(raw_output, image.shape)

        return detections, image

    def _preprocess(self, image: np.ndarray) -> np.ndarray:
        """Preprocess image for YOLO inference."""
        import cv2

        # Resize to model input size with letterboxing
        h, w = image.shape[:2]
        target_h, target_w = self.input_size

        # Calculate scale
        scale = min(target_w / w, target_h / h)
        new_w, new_h = int(w * scale), int(h * scale)

        # Resize
        resized = cv2.resize(image, (new_w, new_h))

        # Pad to target size (letterbox)
        padded = np.full((target_h, target_w, 3), 114, dtype=np.uint8)
        pad_h = (target_h - new_h) // 2
        pad_w = (target_w - new_w) // 2
        padded[pad_h:pad_h + new_h, pad_w:pad_w + new_w] = resized

        # Convert BGR to RGB
        rgb = cv2.cvtColor(padded, cv2.COLOR_BGR2RGB)

        # Normalize to 0-1 range
        input_tensor = rgb.astype(np.float32) / 255.0

        # Transpose to NCHW format for ONNX
        input_tensor = input_tensor.transpose(2, 0, 1)

        # Add batch dimension
        input_tensor = np.expand_dims(input_tensor, 0)

        return input_tensor

    def _infer_hailo(self, input_tensor: np.ndarray) -> np.ndarray:
        """Run inference on Hailo NPU."""
        # Hailo expects NHWC format, convert from NCHW
        input_nhwc = input_tensor.transpose(0, 2, 3, 1)

        # Create input dict
        input_name = self._input_vstream_info.name
        input_data = {input_name: input_nhwc}

        # Run inference
        with InferVStreams(self._network_group, self._input_params, self._output_params) as pipeline:
            output_data = pipeline.infer(input_data)

        # Get first output
        output_name = self._output_vstream_info.name
        return output_data[output_name]

    def _infer_onnx(self, input_tensor: np.ndarray) -> np.ndarray:
        """Run inference with ONNX Runtime."""
        input_name = self._onnx_session.get_inputs()[0].name
        output_name = self._onnx_session.get_outputs()[0].name
        output = self._onnx_session.run([output_name], {input_name: input_tensor})
        return output[0]

    def _parse_yolo_output(
        self,
        output: np.ndarray,
        original_shape: tuple,
    ) -> List[Dict[str, Any]]:
        """Parse YOLO output tensor to detection list.

        Supports YOLOv8 output format.

        Args:
            output: Raw model output tensor
            original_shape: Original image shape (H, W, C)

        Returns:
            List of detection dicts with class_id, confidence, bbox
        """
        detections = []

        logger.debug(f"Raw output shape: {output.shape}")

        # Calculate scale factors for bbox coordinates
        orig_h, orig_w = original_shape[:2]
        target_h, target_w = self.input_size
        scale = min(target_w / orig_w, target_h / orig_h)
        pad_h = (target_h - int(orig_h * scale)) // 2
        pad_w = (target_w - int(orig_w * scale)) // 2

        # Handle different output formats
        if len(output.shape) == 3:
            output = output[0]  # Remove batch dimension

        # YOLOv8 output: [84, num_detections] or [num_detections, 84]
        # 84 = 4 (bbox) + 80 (class scores)
        if output.shape[0] in [84, 85]:
            # Transpose from [84, N] to [N, 84]
            output = output.T
            logger.debug(f"Transposed output to {output.shape}")

        num_classes = output.shape[1] - 4
        logger.debug(f"Processing {len(output)} detections, {num_classes} classes")

        for detection in output:
            # YOLOv8: [x_center, y_center, width, height, class_scores...]
            bbox = detection[:4]
            class_scores = detection[4:]

            # Get best class
            class_id = np.argmax(class_scores)
            confidence = float(class_scores[class_id])

            if confidence < self.confidence_threshold:
                continue

            # Convert from center format to corner format
            cx, cy, w, h = bbox

            # Remove padding offset
            cx = cx - pad_w
            cy = cy - pad_h

            # Scale to original image
            x1 = (cx - w / 2) / scale
            y1 = (cy - h / 2) / scale
            x2 = (cx + w / 2) / scale
            y2 = (cy + h / 2) / scale

            # Clip to image bounds
            x1 = max(0, min(x1, orig_w))
            y1 = max(0, min(y1, orig_h))
            x2 = max(0, min(x2, orig_w))
            y2 = max(0, min(y2, orig_h))

            # Skip invalid boxes
            if x2 <= x1 or y2 <= y1:
                continue

            # Normalize coordinates to 0-1
            detections.append({
                "class_id": int(class_id),
                "confidence": confidence,
                "bbox": [
                    float(x1 / orig_w),
                    float(y1 / orig_h),
                    float(x2 / orig_w),
                    float(y2 / orig_h),
                ],
            })

        if detections:
            logger.debug(f"Found {len(detections)} detections above threshold {self.confidence_threshold}")

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
            self._vdevice.release()
            self._vdevice = None
        if self._hef:
            self._hef = None
        if self._onnx_session:
            self._onnx_session = None
        logger.info("Backend shutdown complete")
