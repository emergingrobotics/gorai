"""
Post-processing for YOLOX detections.

Includes:
- Non-maximum suppression (NMS)
- Class filtering
- Confidence thresholding
"""

from typing import Any, Dict, List

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


def nms(
    boxes: np.ndarray,
    scores: np.ndarray,
    iou_threshold: float = 0.45,
) -> List[int]:
    """Apply non-maximum suppression.

    Args:
        boxes: Array of shape (N, 4) with x1, y1, x2, y2 coordinates
        scores: Array of shape (N,) with confidence scores
        iou_threshold: IoU threshold for suppression

    Returns:
        List of indices to keep
    """
    if len(boxes) == 0:
        return []

    # Convert to float
    boxes = boxes.astype(np.float32)
    scores = scores.astype(np.float32)

    # Get coordinates
    x1 = boxes[:, 0]
    y1 = boxes[:, 1]
    x2 = boxes[:, 2]
    y2 = boxes[:, 3]

    # Calculate areas
    areas = (x2 - x1) * (y2 - y1)

    # Sort by score
    order = scores.argsort()[::-1]

    keep = []
    while len(order) > 0:
        i = order[0]
        keep.append(i)

        if len(order) == 1:
            break

        # Calculate IoU with remaining boxes
        xx1 = np.maximum(x1[i], x1[order[1:]])
        yy1 = np.maximum(y1[i], y1[order[1:]])
        xx2 = np.minimum(x2[i], x2[order[1:]])
        yy2 = np.minimum(y2[i], y2[order[1:]])

        w = np.maximum(0, xx2 - xx1)
        h = np.maximum(0, yy2 - yy1)
        intersection = w * h

        iou = intersection / (areas[i] + areas[order[1:]] - intersection)

        # Keep boxes with IoU below threshold
        inds = np.where(iou <= iou_threshold)[0]
        order = order[inds + 1]

    return keep


def postprocess_detections(
    raw_detections: List[Dict[str, Any]],
    classes: List[str] = None,
    confidence_threshold: float = 0.5,
    nms_threshold: float = 0.45,
) -> List[Dict[str, Any]]:
    """Post-process raw detections.

    Args:
        raw_detections: List of raw detection dicts with class_id, confidence, bbox
        classes: List of class names to keep (None = keep all)
        confidence_threshold: Minimum confidence to keep
        nms_threshold: IoU threshold for NMS

    Returns:
        Filtered and processed detection list
    """
    if not raw_detections:
        return []

    # Filter by confidence
    filtered = [d for d in raw_detections if d["confidence"] >= confidence_threshold]

    if not filtered:
        return []

    # Convert class IDs to names and filter by class
    class_ids_to_keep = None
    if classes:
        class_ids_to_keep = set()
        for cls_name in classes:
            if cls_name.lower() in [c.lower() for c in COCO_CLASSES]:
                idx = [c.lower() for c in COCO_CLASSES].index(cls_name.lower())
                class_ids_to_keep.add(idx)

    processed = []
    for det in filtered:
        class_id = det["class_id"]

        # Skip if class not in filter list
        if class_ids_to_keep is not None and class_id not in class_ids_to_keep:
            continue

        # Get class name
        class_name = COCO_CLASSES[class_id] if class_id < len(COCO_CLASSES) else f"class_{class_id}"

        processed.append({
            "class": class_name,
            "class_id": class_id,
            "confidence": det["confidence"],
            "bbox": det["bbox"],
        })

    if not processed:
        return []

    # Apply NMS
    boxes = np.array([d["bbox"] for d in processed])
    scores = np.array([d["confidence"] for d in processed])

    keep_indices = nms(boxes, scores, nms_threshold)

    return [processed[i] for i in keep_indices]
