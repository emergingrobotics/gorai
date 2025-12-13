"""
YOLOX output postprocessing.

Decodes raw model output into bounding boxes and class predictions.
Applies non-maximum suppression (NMS) to filter overlapping detections.
"""

from dataclasses import dataclass
from typing import Dict, List, Optional

import numpy as np


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


@dataclass
class BoundingBox:
    """Bounding box coordinates."""
    x: int      # Top-left x
    y: int      # Top-left y
    width: int
    height: int

    def to_dict(self) -> dict:
        return {
            "x": self.x,
            "y": self.y,
            "width": self.width,
            "height": self.height
        }


@dataclass
class Detection:
    """A single detection result."""
    class_id: int
    class_name: str
    confidence: float
    bbox: BoundingBox

    def to_dict(self) -> dict:
        return {
            "class_id": self.class_id,
            "class_name": self.class_name,
            "confidence": round(self.confidence, 3),
            "bbox": self.bbox.to_dict()
        }


def decode_yolox_output(
    outputs: Dict[str, np.ndarray],
    image_width: int,
    image_height: int,
    confidence_threshold: float = 0.5,
    nms_threshold: float = 0.45,
    target_classes: Optional[List[str]] = None
) -> List[Detection]:
    """
    Decode YOLOX model output to detection objects.

    Args:
        outputs: Model outputs dictionary
        image_width: Original image width
        image_height: Original image height
        confidence_threshold: Minimum confidence threshold
        nms_threshold: NMS IoU threshold
        target_classes: List of class names to detect (None = all)

    Returns:
        List of Detection objects
    """
    # Get output tensor (assume single output)
    output_name = list(outputs.keys())[0]
    predictions = outputs[output_name]

    # Expected shape: (batch, num_anchors, 5 + num_classes)
    if len(predictions.shape) == 3:
        predictions = predictions[0]  # Remove batch dimension

    # Get target class indices
    target_indices = None
    if target_classes:
        target_indices = set()
        for class_name in target_classes:
            if class_name in COCO_CLASSES:
                target_indices.add(COCO_CLASSES.index(class_name))

    # Parse predictions
    boxes = []
    scores = []
    class_ids = []

    # Model input size (assumed 640x640)
    model_input_size = 640

    # Scale factors
    scale_x = image_width / model_input_size
    scale_y = image_height / model_input_size

    for pred in predictions:
        # pred: [x, y, w, h, obj_conf, class_probs...]
        obj_conf = pred[4]

        if obj_conf < confidence_threshold:
            continue

        # Get class with highest probability
        class_probs = pred[5:]
        class_id = np.argmax(class_probs)
        class_conf = class_probs[class_id]

        # Combined confidence
        confidence = obj_conf * class_conf

        if confidence < confidence_threshold:
            continue

        # Filter by target classes
        if target_indices and class_id not in target_indices:
            continue

        # Get bounding box (center format to corner format)
        cx, cy, w, h = pred[:4]

        # Scale to original image size
        cx *= scale_x
        cy *= scale_y
        w *= scale_x
        h *= scale_y

        # Convert to corner format
        x1 = int(cx - w / 2)
        y1 = int(cy - h / 2)
        x2 = int(cx + w / 2)
        y2 = int(cy + h / 2)

        # Clip to image bounds
        x1 = max(0, min(x1, image_width - 1))
        y1 = max(0, min(y1, image_height - 1))
        x2 = max(0, min(x2, image_width))
        y2 = max(0, min(y2, image_height))

        boxes.append([x1, y1, x2, y2])
        scores.append(float(confidence))
        class_ids.append(int(class_id))

    if not boxes:
        return []

    # Apply NMS
    boxes = np.array(boxes)
    scores = np.array(scores)
    class_ids = np.array(class_ids)

    keep_indices = nms(boxes, scores, nms_threshold)

    # Build detection results
    detections = []
    for idx in keep_indices:
        x1, y1, x2, y2 = boxes[idx]
        class_id = class_ids[idx]

        det = Detection(
            class_id=class_id,
            class_name=COCO_CLASSES[class_id] if class_id < len(COCO_CLASSES) else f"class_{class_id}",
            confidence=scores[idx],
            bbox=BoundingBox(
                x=int(x1),
                y=int(y1),
                width=int(x2 - x1),
                height=int(y2 - y1)
            )
        )
        detections.append(det)

    return detections


def nms(boxes: np.ndarray, scores: np.ndarray, threshold: float) -> List[int]:
    """
    Non-maximum suppression.

    Args:
        boxes: Array of boxes (N, 4) in [x1, y1, x2, y2] format
        scores: Array of confidence scores (N,)
        threshold: IoU threshold

    Returns:
        List of indices to keep
    """
    if len(boxes) == 0:
        return []

    # Sort by score (descending)
    order = scores.argsort()[::-1]

    x1 = boxes[:, 0]
    y1 = boxes[:, 1]
    x2 = boxes[:, 2]
    y2 = boxes[:, 3]

    areas = (x2 - x1) * (y2 - y1)

    keep = []

    while len(order) > 0:
        i = order[0]
        keep.append(i)

        if len(order) == 1:
            break

        # Compute IoU with remaining boxes
        xx1 = np.maximum(x1[i], x1[order[1:]])
        yy1 = np.maximum(y1[i], y1[order[1:]])
        xx2 = np.minimum(x2[i], x2[order[1:]])
        yy2 = np.minimum(y2[i], y2[order[1:]])

        w = np.maximum(0, xx2 - xx1)
        h = np.maximum(0, yy2 - yy1)

        intersection = w * h
        union = areas[i] + areas[order[1:]] - intersection
        iou = intersection / (union + 1e-6)

        # Keep boxes with IoU below threshold
        mask = iou <= threshold
        order = order[1:][mask]

    return keep
