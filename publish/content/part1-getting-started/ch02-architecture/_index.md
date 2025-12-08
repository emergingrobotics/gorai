# Chapter 2: Architecture & Mental Model

> **In This Chapter:** Learn how to think about GoRAI systems—nodes, resources, distributed architecture, and network transparency.

## Overview

Every framework has a mental model—a way of thinking about problems that, once internalized, makes everything easier. GoRAI's mental model comes from distributed systems engineering: robots are networks of communicating processes, and the communication patterns matter as much as the code.

This chapter establishes the conceptual foundation you'll use throughout your GoRAI development. We'll cover the three-layer architecture, the node and resource abstractions, configuration patterns, and the NWS/NWC mechanism that makes location transparent.

## What You'll Learn

After reading this chapter, you'll understand:

- GoRAI's three-layer architecture (primary, secondary, microcontroller)
- What nodes are and how they manage resources
- The difference between components and services
- How configuration drives robot behavior
- Network transparency through NWS/NWC

## Chapter Contents

| Section | Description |
|---------|-------------|
| [The Big Picture](bigpicture.md) | System architecture diagram and three-layer model |
| [Core Concepts](coreconcepts.md) | Nodes, resources, components, services |
| [Distributed Systems](distributed.md) | Why distributed matters, communication patterns |
| [Configuration](config.md) | JSON config, hot reload, dependency injection |
| [NWS/NWC Pattern](nwsnwc.md) | Network wrappers for location transparency |

## Key Takeaways

- **Three layers**: Primary compute (Linux SBCs), secondary nodes, microcontrollers (TinyGo)
- **Nodes** are the unit of deployment; they own and manage resources
- **Resources** are either **components** (hardware abstraction) or **services** (software capabilities)
- **Configuration** is JSON-based and supports hot reload
- **NWS/NWC** makes remote resources look like local ones

## Prerequisites

This chapter assumes you've read:
- [Chapter 1: Why GoRAI?](../ch01-why-gorai/_index.md)

No coding required yet—this is conceptual foundation.

<!-- book-only -->
*Take your time with this chapter. The mental model established here will make everything else in the book click into place. Consider re-reading after you've worked through Part III.*
<!-- /book-only -->

<!-- website-only -->
!!! abstract "Mental Model First"
    This chapter is conceptual—no code required. Understanding these concepts will make the hands-on chapters much easier.
<!-- /website-only -->
