# Appendix A: Glossary {.unnumbered}

This glossary defines key terms used throughout the book.

**Actuator**
: A component that produces physical motion or action, such as a motor, servo, or pneumatic cylinder.

**Behavior**
: A reusable unit of robot functionality that combines sensing and actuation to achieve a goal.

**Component**
: A fundamental building block in Gorai that interfaces with hardware (sensors, actuators) or provides services.

**Coordinator**
: A higher-level component that orchestrates multiple behaviors to achieve complex goals.

**JetStream**
: NATS's persistence layer that provides message history, replay, and exactly-once delivery.

**NATS**
: A lightweight, high-performance messaging system used by Gorai for all component communication.

**NWC (Node With Compute)**
: A Gorai node running on hardware with significant computing resources (Linux SBC, laptop, etc.).

**NWS (Node With Sensors)**
: A Gorai node running on a microcontroller that interfaces directly with sensors and actuators.

**Pub/Sub**
: A messaging pattern where publishers send messages to topics and subscribers receive messages from topics they're interested in.

**Request/Reply**
: A messaging pattern where a requester sends a message and waits for a response from a responder.

**Resource**
: The base abstraction in Gorai that all components and services implement.

**Sensor**
: A component that measures physical quantities (temperature, distance, orientation) and publishes data.

**Service**
: A component that provides capabilities not tied to specific hardware (navigation, vision processing).

**TinyGo**
: A Go compiler designed for microcontrollers and WebAssembly, used for Gorai's NWS nodes.

**Topic**
: A named channel for message routing in NATS. Components publish to and subscribe from topics.
