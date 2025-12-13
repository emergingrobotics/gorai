"""
Hailo NPU inference backend.

This module provides inference using the Hailo-8L NPU via HailoRT.
It loads HEF (Hailo Executable Format) models and runs inference.

For systems without Hailo hardware, a mock backend is used for development.
"""

import logging
import os
from typing import Dict, Tuple, Any

import numpy as np

logger = logging.getLogger("gorai.object_detection.inference")

# Try to import Hailo SDK
try:
    from hailo_platform import (
        HEF,
        VDevice,
        HailoStreamInterface,
        ConfigureParams,
        InferVStreams,
        InputVStreamParams,
        OutputVStreamParams,
        FormatType,
    )
    HAILO_AVAILABLE = True
    logger.info("Hailo SDK available")
except ImportError:
    HAILO_AVAILABLE = False
    logger.warning("Hailo SDK not available, using mock backend")


class HailoInference:
    """
    Hailo NPU inference engine.

    Loads a HEF model file and runs inference on the Hailo-8L NPU.
    Falls back to mock inference if Hailo hardware is not available.
    """

    def __init__(self, hef_path: str, batch_size: int = 1):
        """
        Initialize Hailo inference engine.

        Args:
            hef_path: Path to HEF model file
            batch_size: Batch size for inference
        """
        self.hef_path = hef_path
        self.batch_size = batch_size
        self.input_shape: Tuple[int, ...] = (1, 640, 640, 3)  # Default YOLO input

        if HAILO_AVAILABLE and os.path.exists(hef_path):
            self._init_hailo()
        else:
            self._init_mock()

    def _init_hailo(self):
        """Initialize real Hailo backend."""
        logger.info(f"Loading HEF model: {self.hef_path}")

        # Load HEF file
        self.hef = HEF(self.hef_path)

        # Create virtual device
        self.vdevice = VDevice()

        # Configure network group
        configure_params = ConfigureParams.create_from_hef(
            self.hef,
            interface=HailoStreamInterface.PCIe
        )
        self.network_group = self.vdevice.configure(
            self.hef,
            configure_params
        )[0]

        # Get input/output stream info
        self.input_vstream_infos = self.network_group.get_input_vstream_infos()
        self.output_vstream_infos = self.network_group.get_output_vstream_infos()

        # Get input shape
        input_info = self.input_vstream_infos[0]
        self.input_shape = input_info.shape
        self.input_name = input_info.name

        # Get output names
        self.output_names = [info.name for info in self.output_vstream_infos]

        logger.info(f"Model loaded: input={self.input_shape}, outputs={self.output_names}")

        self.is_mock = False

    def _init_mock(self):
        """Initialize mock backend for development."""
        logger.info("Using mock inference backend (no Hailo hardware)")

        # Default YOLOx input shape
        self.input_shape = (1, 640, 640, 3)
        self.input_name = "input"
        self.output_names = ["output"]

        self.is_mock = True

    def infer(self, image: np.ndarray) -> Dict[str, np.ndarray]:
        """
        Run inference on preprocessed image.

        Args:
            image: Preprocessed image array with shape (batch, height, width, channels)

        Returns:
            Dictionary mapping output names to output tensors
        """
        if self.is_mock:
            return self._mock_infer(image)

        return self._hailo_infer(image)

    def _hailo_infer(self, image: np.ndarray) -> Dict[str, np.ndarray]:
        """Run inference on Hailo hardware."""
        # Ensure correct dtype
        if image.dtype != np.float32:
            image = image.astype(np.float32)

        # Prepare input data
        input_data = {self.input_name: image}

        # Configure stream parameters
        input_vstream_params = InputVStreamParams.make_from_network_group(
            self.network_group,
            quantized=False,
            format_type=FormatType.FLOAT32
        )
        output_vstream_params = OutputVStreamParams.make_from_network_group(
            self.network_group,
            quantized=False,
            format_type=FormatType.FLOAT32
        )

        # Run inference
        with self.network_group.activate():
            with InferVStreams(
                self.network_group,
                input_vstream_params,
                output_vstream_params
            ) as infer_pipeline:
                results = infer_pipeline.infer(input_data)

        return results

    def _mock_infer(self, image: np.ndarray) -> Dict[str, np.ndarray]:
        """
        Mock inference for development without Hailo hardware.

        Returns random detections for testing.
        """
        # Simulate inference delay
        import time
        time.sleep(0.03)  # ~30ms like real inference

        batch_size = image.shape[0]

        # YOLOx output shape: (batch, num_anchors, 5 + num_classes)
        # For COCO: (1, 8400, 85) where 85 = 4 bbox + 1 obj + 80 classes
        num_anchors = 8400
        num_classes = 80

        # Generate mock output
        output = np.zeros((batch_size, num_anchors, 5 + num_classes), dtype=np.float32)

        # Add a few random "detections" for person class (class 0)
        import random
        num_detections = random.randint(0, 3)

        for i in range(num_detections):
            anchor_idx = random.randint(0, num_anchors - 1)

            # Random bbox (x, y, w, h in image coords)
            output[0, anchor_idx, 0] = random.uniform(0.1, 0.8) * 640  # x
            output[0, anchor_idx, 1] = random.uniform(0.1, 0.8) * 640  # y
            output[0, anchor_idx, 2] = random.uniform(50, 200)  # w
            output[0, anchor_idx, 3] = random.uniform(100, 400)  # h

            # Object confidence
            output[0, anchor_idx, 4] = random.uniform(0.6, 0.95)

            # Class probability (person = class 0)
            output[0, anchor_idx, 5] = random.uniform(0.7, 0.99)

        return {"output": output}

    def get_input_shape(self) -> Tuple[int, ...]:
        """Get model input shape."""
        return self.input_shape

    def close(self):
        """Release resources."""
        if not self.is_mock and hasattr(self, 'vdevice'):
            # VDevice cleanup handled by context manager
            pass
