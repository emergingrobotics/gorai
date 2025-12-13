"""Configuration loading for object detection service."""

import json
import os
from dataclasses import dataclass
from typing import List, Optional


@dataclass
class ServiceConfig:
    """Configuration for the object detection service."""

    # Robot info
    robot_name: str
    service_name: str

    # NATS connection
    nats_url: str

    # Model settings
    model_path: str
    confidence_threshold: float
    classes: List[str]

    # Topics
    input_topic: str
    output_topic_annotated: str
    output_topic_detections: str


def load_config(config_path: str) -> ServiceConfig:
    """
    Load service configuration from RDL file and environment.

    Args:
        config_path: Path to robot.json configuration file

    Returns:
        ServiceConfig with all settings
    """
    # Load RDL configuration
    with open(config_path, 'r') as f:
        rdl = json.load(f)

    robot_name = rdl["robot"]["name"]

    # Find person_detector service configuration
    service_config = None
    for svc in rdl.get("services", []):
        if svc.get("type") == "object_detection":
            service_config = svc
            break

    if service_config is None:
        raise ValueError("No object_detection service found in configuration")

    service_name = service_config["name"]
    attrs = service_config.get("attributes", {})

    # Get NATS URL from environment or config
    nats_url = os.environ.get("NATS_URL")
    if not nats_url:
        nats_config = rdl.get("nats", {})
        nats_url = nats_config.get("url", "nats://localhost:4222")

    # Get model path from environment or config
    model_path = os.environ.get("MODEL_PATH")
    if not model_path:
        model_path = attrs.get("model_path", "/models/yolox_s_leaky.hef")

    # Get confidence threshold from environment or config
    confidence_threshold = float(os.environ.get("CONFIDENCE_THRESHOLD", "0"))
    if confidence_threshold == 0:
        confidence_threshold = attrs.get("confidence_threshold", 0.5)

    # Get classes to detect
    classes = attrs.get("classes", ["person"])

    # Get topics
    input_topic = attrs.get(
        "input_topic",
        f"gorai.{robot_name}.main_camera.data"
    )
    output_topic_annotated = attrs.get(
        "output_topic_annotated",
        f"gorai.{robot_name}.{service_name}.annotated"
    )
    output_topic_detections = attrs.get(
        "output_topic_detections",
        f"gorai.{robot_name}.{service_name}.detections"
    )

    return ServiceConfig(
        robot_name=robot_name,
        service_name=service_name,
        nats_url=nats_url,
        model_path=model_path,
        confidence_threshold=confidence_threshold,
        classes=classes,
        input_topic=input_topic,
        output_topic_annotated=output_topic_annotated,
        output_topic_detections=output_topic_detections,
    )
