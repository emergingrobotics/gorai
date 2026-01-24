"""
Draw bounding boxes and labels on images.
"""

from typing import Any, Dict, List, Tuple

import cv2
import numpy as np


def hex_to_bgr(hex_color: str) -> Tuple[int, int, int]:
    """Convert hex color to BGR tuple."""
    hex_color = hex_color.lstrip("#")
    r, g, b = tuple(int(hex_color[i:i+2], 16) for i in (0, 2, 4))
    return (b, g, r)  # OpenCV uses BGR


def draw_bounding_boxes(
    image_input,  # Can be bytes (JPEG) or np.ndarray (decoded image)
    detections: List[Dict[str, Any]],
    color: str = "#00FF00",
    thickness: int = 2,
    draw_labels: bool = True,
    font_scale: float = 0.6,
    jpeg_quality: int = 80,
) -> bytes:
    """Draw bounding boxes on image.

    Args:
        image_input: Input image - either JPEG bytes or decoded numpy array
        detections: List of detection dicts with class, confidence, bbox
        color: Box color in hex format
        thickness: Line thickness in pixels
        draw_labels: Whether to draw class labels
        font_scale: Font scale for labels
        jpeg_quality: Output JPEG quality (1-100)

    Returns:
        Annotated JPEG image as bytes
    """
    # Handle both JPEG bytes and numpy array input
    if isinstance(image_input, np.ndarray):
        image = image_input
    else:
        # Decode JPEG
        nparr = np.frombuffer(image_input, np.uint8)
        image = cv2.imdecode(nparr, cv2.IMREAD_COLOR)

    if image is None:
        return image_input if isinstance(image_input, bytes) else b''

    h, w = image.shape[:2]
    box_color = hex_to_bgr(color)

    for det in detections:
        bbox = det["bbox"]
        confidence = det["confidence"]
        class_name = det.get("class", "object")

        # Convert normalized coordinates to pixel coordinates
        x1 = int(bbox[0] * w)
        y1 = int(bbox[1] * h)
        x2 = int(bbox[2] * w)
        y2 = int(bbox[3] * h)

        # Draw rectangle
        cv2.rectangle(image, (x1, y1), (x2, y2), box_color, thickness)

        if draw_labels:
            # Prepare label text
            label = f"{class_name}: {confidence:.2f}"

            # Get text size
            (text_width, text_height), baseline = cv2.getTextSize(
                label,
                cv2.FONT_HERSHEY_SIMPLEX,
                font_scale,
                thickness,
            )

            # Draw label background
            label_y = max(y1, text_height + 10)
            cv2.rectangle(
                image,
                (x1, label_y - text_height - 10),
                (x1 + text_width + 10, label_y),
                box_color,
                -1,  # Filled
            )

            # Draw label text
            cv2.putText(
                image,
                label,
                (x1 + 5, label_y - 5),
                cv2.FONT_HERSHEY_SIMPLEX,
                font_scale,
                (255, 255, 255),  # White text
                thickness,
            )

    # Encode back to JPEG
    encode_params = [cv2.IMWRITE_JPEG_QUALITY, jpeg_quality]
    _, encoded = cv2.imencode(".jpg", image, encode_params)

    return encoded.tobytes()
