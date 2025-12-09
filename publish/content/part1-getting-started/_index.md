# Part I: Getting Started

> **GoRAI** is pronounced "go-ray" (like "sting-ray")

> **In This Part:** Understand why GoRAI exists, who it's for, and how to think about robot software as distributed systems.

## Overview

Before writing any code, you need to understand *why* GoRAI exists and *how* it approaches robotics software differently than alternatives like ROS 2, YARP, or Viam. This foundation will make everything else click into place.

Part I establishes the mental model you'll use throughout your GoRAI development. We start with the landscape of robotics frameworks, explain GoRAI's design philosophy, and then dive into the architectural concepts that make everything work.

## Chapters in This Part

| Chapter | Title | What You'll Learn |
|---------|-------|-------------------|
| [Chapter 1](ch01-why-gorai/_index.md) | Why GoRAI? | The robotics landscape, GoRAI's philosophy, and who should use it |
| [Chapter 2](ch02-architecture/_index.md) | Architecture & Mental Model | Core concepts, distributed systems, and the NWS/NWC pattern |

## Key Concepts Introduced

By the end of Part I, you'll understand:

- **Why existing frameworks fall short** for modern robotics development
- **GoRAI's design principles**: Go-first, NATS-native, AI-optimized, modular
- **The three-layer architecture**: Primary compute, secondary nodes, microcontrollers
- **Nodes and Resources**: The fundamental building blocks
- **Network transparency**: How NWS/NWC makes location irrelevant
- **Configuration-driven design**: JSON config with hot reload

## Prerequisites

Part I assumes only:
- Basic programming knowledge (any language)
- Command-line familiarity
- Curiosity about robotics

No prior robotics experience required. If you've used ROS or similar frameworks, you'll find familiar concepts presented with fresh perspectives.

<!-- book-only -->
*Read these chapters sequentially. Chapter 2 builds directly on the context established in Chapter 1.*
<!-- /book-only -->

<!-- website-only -->
!!! tip "New to GoRAI?"
    If you want to jump straight to code, try the [Quick Start](../getting-started/quickstart.md) first, then return here to understand the concepts more deeply.
<!-- /website-only -->
