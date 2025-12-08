# Part III: Development

> **In This Part:** Set up your development environment, build your first complete GoRAI project, create custom components, and master testing strategies.

## Overview

Part III is where theory meets practice. You'll set up a complete development environment, work through the "Hello Sensor" tutorial step-by-step, learn to create your own custom components, and develop a testing strategy that catches bugs before they reach your robot.

This is the most hands-on part of the book. Expect to write code, run tests, and see real output.

## Chapters in This Part

| Chapter | Title | What You'll Learn |
|---------|-------|-------------------|
| [Chapter 10](ch10-devenv/_index.md) | Development Environment | Go setup, NATS, tools, hardware configuration |
| [Chapter 11](ch11-hello-sensor/_index.md) | Hello Sensor Tutorial | Complete walkthrough of a real GoRAI component |
| [Chapter 12](ch12-custom/_index.md) | Custom Components | Creating your own sensors, motors, and services |
| [Chapter 13](ch13-testing/_index.md) | Testing | Unit tests, component tests, integration tests, fakes |

## Key Concepts Introduced

By the end of Part III, you'll understand:

- **Development workflow**: Edit, test, run cycle with GoRAI
- **Project structure**: How to organize GoRAI code
- **Platform-specific code**: Build tags for Linux, macOS, etc.
- **The fake pattern**: Test doubles for every component
- **Testing pyramid**: Unit → Component → Integration → System → Hardware
- **Continuous integration**: Automated testing for robot code

## Prerequisites

Part III assumes you've read:
- Part I (mental model and architecture)
- Chapter 3 (NATS messaging) from Part II
- At least one component chapter (4, 5, or 6) from Part II

You'll also need:
- A computer with Go 1.21+ installed
- Basic terminal/command-line skills
- Optionally: A Raspberry Pi or similar SBC for hardware testing

## Hands-On Projects

| Chapter | Project | Outcome |
|---------|---------|---------|
| 10 | Environment Setup | Working GoRAI development environment |
| 11 | Hello Sensor | Complete temperature sensor with NATS publishing |
| 12 | Custom Motor | Your own motor driver with fake implementation |
| 13 | Test Suite | Comprehensive tests for a GoRAI component |

<!-- book-only -->
*Work through Chapter 11 completely before attempting Chapter 12. The Hello Sensor example establishes patterns you'll use repeatedly.*
<!-- /book-only -->

<!-- website-only -->
!!! tip "Start Here for Hands-On Learning"
    If you learn by doing, start with [Chapter 10: Development Environment](ch10-devenv/_index.md) to get set up, then work through the [Hello Sensor Tutorial](ch11-hello-sensor/_index.md).
<!-- /website-only -->
