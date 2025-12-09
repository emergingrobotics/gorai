# Chapter 1: Why GoRAI?

> **In This Chapter:** Understand the robotics software landscape, GoRAI's design philosophy, and whether GoRAI is right for your project.

## Overview

Before diving into code, let's establish *why* GoRAI exists. The robotics software world isn't lacking options—ROS 2, YARP, Viam, and others all have their place. So why create something new?

This chapter answers that question by examining what existing frameworks do well, where they fall short, and how GoRAI addresses those gaps. By the end, you'll understand GoRAI's design philosophy and be able to judge whether it's the right tool for your project.

## What You'll Learn

After reading this chapter, you'll understand:

- The history and evolution of robotics middleware
- Common pain points with existing frameworks
- GoRAI's core design principles
- Who should (and shouldn't) use GoRAI
- What you'll build throughout this book

## Chapter Contents

| Section | Description |
|---------|-------------|
| [The Robotics Landscape](landscape.md) | Brief history of ROS, ROS 2, YARP, Viam and their trade-offs |
| [Design Philosophy](philosophy.md) | GoRAI's core principles: Go-first, NATS-native, AI-optimized |
| [Target Audience](audience.md) | Who should use GoRAI and for what types of projects |
| [What You'll Build](whatyoullbuild.md) | Preview of the projects in this book |
| [Prerequisites](prerequisites.md) | Knowledge and tools you'll need |

## Key Takeaways

- **ROS 2** is powerful but complex; great for research, challenging for production
- **Viam** simplifies robotics but locks you into their ecosystem
- **GoRAI** aims for the sweet spot: powerful enough for real robots, simple enough to enjoy using
- GoRAI is opinionated—it chooses Go, NATS, and Protocol Buffers rather than offering infinite flexibility

<!-- book-only -->
*This chapter sets the stage for everything that follows. Even if you're eager to write code, understanding the "why" will make the "how" much clearer.*
<!-- /book-only -->

<!-- website-only -->
!!! tip "Already Convinced?"
    If you're already sold on GoRAI and want to start coding, jump to [Getting Started](../../getting-started/quickstart.md). Return here later to understand the deeper context.
<!-- /website-only -->
