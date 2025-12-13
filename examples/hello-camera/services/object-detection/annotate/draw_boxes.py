"""
Draw bounding boxes and labels on images.

Uses PIL (Pillow) for lightweight image annotation.
"""

from typing import List

from PIL import Image, ImageDraw, ImageFont

from processing.postprocess import Detection


# Color palette for different classes
CLASS_COLORS = [
    "#FF6B6B",  # Red - person
    "#4ECDC4",  # Teal
    "#45B7D1",  # Blue
    "#96CEB4",  # Green
    "#FFEAA7",  # Yellow
    "#DDA0DD",  # Plum
    "#98D8C8",  # Mint
    "#F7DC6F",  # Gold
    "#BB8FCE",  # Purple
    "#85C1E9",  # Light blue
]


def get_class_color(class_id: int) -> str:
    """Get consistent color for a class ID."""
    return CLASS_COLORS[class_id % len(CLASS_COLORS)]


def draw_detections(
    image: Image.Image,
    detections: List[Detection],
    line_width: int = 2,
    font_size: int = 14
) -> Image.Image:
    """
    Draw bounding boxes and labels on image.

    Args:
        image: PIL Image to annotate
        detections: List of Detection objects
        line_width: Bounding box line width
        font_size: Label font size

    Returns:
        Annotated PIL Image
    """
    draw = ImageDraw.Draw(image)

    # Try to load a font
    font = _get_font(font_size)

    for det in detections:
        bbox = det.bbox
        x1, y1 = bbox.x, bbox.y
        x2, y2 = x1 + bbox.width, y1 + bbox.height

        # Get color for this class
        color = get_class_color(det.class_id)

        # Draw bounding box
        draw.rectangle(
            [x1, y1, x2, y2],
            outline=color,
            width=line_width
        )

        # Prepare label text
        label = f"{det.class_name} {det.confidence:.0%}"

        # Get label dimensions
        text_bbox = draw.textbbox((x1, y1), label, font=font)
        text_width = text_bbox[2] - text_bbox[0]
        text_height = text_bbox[3] - text_bbox[1]

        # Position label above bounding box, or below if no room
        label_y = y1 - text_height - 4
        if label_y < 0:
            label_y = y2 + 2

        # Draw label background
        padding = 2
        draw.rectangle(
            [
                x1 - padding,
                label_y - padding,
                x1 + text_width + padding * 2,
                label_y + text_height + padding
            ],
            fill=color
        )

        # Draw label text
        draw.text(
            (x1 + padding, label_y),
            label,
            fill="white",
            font=font
        )

    return image


def _get_font(size: int) -> ImageFont.FreeTypeFont:
    """Get a font for label rendering."""
    # Try common font paths
    font_paths = [
        "/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
        "/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
        "/usr/share/fonts/TTF/DejaVuSans.ttf",
        "/usr/share/fonts/dejavu/DejaVuSans.ttf",
    ]

    for path in font_paths:
        try:
            return ImageFont.truetype(path, size)
        except (IOError, OSError):
            continue

    # Fall back to default font
    try:
        return ImageFont.load_default()
    except Exception:
        return None
