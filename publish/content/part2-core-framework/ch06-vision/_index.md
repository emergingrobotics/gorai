# Chapter 6: Vision, Links & More

> **In This Chapter:** Work with cameras, depth sensors, and other advanced components. Understand image data flow and computer vision integration.

## Overview

Vision gives robots the ability to see and interpret their environment. GoRAI's camera interface provides a consistent way to capture images from USB cameras, CSI cameras, IP cameras, and depth sensors.

This chapter covers the Camera interface, different camera types, how image data flows through the system, and integration with computer vision libraries like OpenCV.

## What You'll Learn

After reading this chapter, you'll understand:

- The Camera interface and image capture
- Different camera types and their characteristics
- Image data flow over NATS
- Computer vision integration patterns
- Depth sensing and point clouds

## Chapter Contents

| Section | Description |
|---------|-------------|
| [Camera Interface](camera.md) | `Image()`, properties, resolution, formats |
| [Camera Types](types.md) | USB, CSI, IP, depth cameras |
| [Data Flow](dataflow.md) | Image streaming over NATS, compression |
| [Computer Vision](cv.md) | OpenCV integration, processing pipelines |

## Key Takeaways

- **Cameras** return `image.Image` from Go's standard library
- **Properties** expose resolution, frame rate, intrinsics
- **Streaming** over NATS requires compression (JPEG, H.264)
- **Depth cameras** provide RGB-D data for 3D perception
- **OpenCV** integration uses GoCV bindings

## Prerequisites

This chapter assumes you've read:
- [Chapter 3: NATS](../ch03-nats/_index.md) — How image data flows
- [Chapter 4: Sensors](../ch04-sensors/_index.md) — Component interface patterns

## Quick Reference

```go
// The Camera interface
type Camera interface {
    Component
    Image(ctx context.Context) (image.Image, error)
    Properties(ctx context.Context) (Properties, error)
}

// Properties includes
type Properties struct {
    Width      int
    Height     int
    FrameRate  float64
    Intrinsics *Intrinsics // camera calibration
}
```

<!-- book-only -->
*Vision systems can consume significant bandwidth and processing power. The data flow patterns in this chapter are essential for building responsive robots.*
<!-- /book-only -->

<!-- website-only -->
!!! info "Bandwidth Considerations"
    Raw images are large. See the [Data Flow](dataflow.md) section for compression and streaming strategies.
<!-- /website-only -->
