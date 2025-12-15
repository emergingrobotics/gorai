"""
Settings configuration for the person detector service.

Configuration is loaded from environment variables, which are set by the
Gorai robot runtime based on the Service RDL and robot RDL.
"""

import json
import os
from dataclasses import dataclass, field
from typing import List


@dataclass
class Settings:
    """Service configuration settings."""

    # NATS connection
    nats_url: str = "nats://localhost:4222"

    # Robot/service identity
    robot_name: str = "hello-people-detector"
    service_name: str = "person_detector"
    namespace: str = "hello-people-detector"

    # Topics (resolved from Service RDL patterns)
    input_topic: str = ""
    output_topic_annotated: str = ""
    output_topic_detections: str = ""
    heartbeat_topic: str = ""

    # Model configuration
    model_path: str = "/models/yolox_s_leaky.hef"
    confidence_threshold: float = 0.5
    classes: List[str] = field(default_factory=lambda: ["person"])
    input_size: int = 640  # Model input size (320, 416, 640) - smaller = faster

    # Annotation settings
    draw_boxes: bool = True
    draw_labels: bool = True
    box_color: str = "#00FF00"
    box_thickness: int = 2

    @classmethod
    def from_environment(cls) -> "Settings":
        """Load settings from environment variables."""
        settings = cls()

        # NATS
        settings.nats_url = os.environ.get("NATS_URL", settings.nats_url)

        # Identity
        settings.robot_name = os.environ.get("GORAI_ROBOT_NAME", settings.robot_name)
        settings.service_name = os.environ.get("GORAI_SERVICE_NAME", settings.service_name)
        settings.namespace = os.environ.get("GORAI_NAMESPACE", settings.robot_name)

        # Topics - can be set directly or resolved from patterns
        input_component = os.environ.get("INPUT_COMPONENT", "main_camera")

        # Check for explicitly set topics first
        settings.input_topic = os.environ.get(
            "INPUT_TOPIC",
            f"gorai.{settings.namespace}.{input_component}.data"
        )
        settings.output_topic_annotated = os.environ.get(
            "OUTPUT_TOPIC_ANNOTATED",
            f"gorai.{settings.namespace}.{settings.service_name}.annotated"
        )
        settings.output_topic_detections = os.environ.get(
            "OUTPUT_TOPIC_DETECTIONS",
            f"gorai.{settings.namespace}.{settings.service_name}.detections"
        )
        settings.heartbeat_topic = os.environ.get(
            "HEARTBEAT_TOPIC",
            f"gorai.{settings.namespace}.system.heartbeat"
        )

        # Parse resolved topics from Gorai runtime (JSON format)
        if "GORAI_INPUT_TOPICS" in os.environ:
            try:
                topics = json.loads(os.environ["GORAI_INPUT_TOPICS"])
                if "input" in topics:
                    settings.input_topic = topics["input"]
            except json.JSONDecodeError:
                pass

        if "GORAI_OUTPUT_TOPICS" in os.environ:
            try:
                topics = json.loads(os.environ["GORAI_OUTPUT_TOPICS"])
                if "annotated" in topics:
                    settings.output_topic_annotated = topics["annotated"]
                if "detections" in topics:
                    settings.output_topic_detections = topics["detections"]
            except json.JSONDecodeError:
                pass

        # Model configuration
        settings.model_path = os.environ.get("MODEL_PATH", settings.model_path)
        settings.confidence_threshold = float(
            os.environ.get("CONFIDENCE_THRESHOLD", settings.confidence_threshold)
        )
        settings.input_size = int(
            os.environ.get("INPUT_SIZE", settings.input_size)
        )

        # Classes can be comma-separated or JSON array
        classes_env = os.environ.get("CLASSES", "")
        if classes_env:
            try:
                settings.classes = json.loads(classes_env)
            except json.JSONDecodeError:
                settings.classes = [c.strip() for c in classes_env.split(",")]

        # Annotation settings
        settings.draw_boxes = os.environ.get("DRAW_BOXES", "true").lower() == "true"
        settings.draw_labels = os.environ.get("DRAW_LABELS", "true").lower() == "true"
        settings.box_color = os.environ.get("BOX_COLOR", settings.box_color)
        settings.box_thickness = int(
            os.environ.get("BOX_THICKNESS", settings.box_thickness)
        )

        return settings
