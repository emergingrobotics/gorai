## 6.3 Image Data Flow

Images are large—a 1920x1080 RGB image is 6MB uncompressed. Efficient handling matters.

### Protocol Buffer Representation

From `vision.proto`:

```protobuf
message Image {
    std.Header header = 1;
    uint32 height = 2;
    uint32 width = 3;
    string encoding = 4;    // "rgb8", "bgr8", "mono8", "jpeg", etc.
    uint32 step = 5;        // Row length in bytes
    bytes data = 6;
}

message CompressedImage {
    std.Header header = 1;
    string format = 2;      // "jpeg", "png", "h264"
    bytes data = 3;
}
```

### Compression Considerations

**Raw images** for processing:
```go
// Full quality for vision algorithms
img, _ := camera.Image(ctx)
detections := detector.Detect(img)
```

**Compressed images** for transport:
```go
// JPEG for network transmission (10-20x smaller)
var buf bytes.Buffer
jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})

pub.Publish(ctx, &vision.CompressedImage{
    Format: "jpeg",
    Data:   buf.Bytes(),
})
```

**When to compress**:
- Sending over network (NATS, remote nodes)
- Logging/recording
- Display/streaming to operators

**When to keep raw**:
- Local processing pipeline
- Algorithms sensitive to compression artifacts
- Stereo matching, feature detection

### Streaming vs On-Demand

**On-demand** (pull model):
```go
// Get frame when needed
for {
    img, _ := camera.Image(ctx)
    processFrame(img)
    time.Sleep(100 * time.Millisecond)
}
```

Pros: Simple, process at your rate
Cons: May miss frames, inconsistent timing

**Streaming** (push model):
```go
// Subscribe to continuous frames
stream, _ := camera.Stream(ctx)
for img := range stream {
    processFrame(img)
}
```

Pros: All frames available, consistent timing
Cons: Must keep up or drop frames

### Frame Rate Management

Cameras produce frames faster than processing can handle:

```go
type FrameDropper struct {
    input  <-chan image.Image
    output chan image.Image
    latest image.Image
}

func (d *FrameDropper) Run(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return

        case img := <-d.input:
            // Always keep the latest
            d.latest = img

        case d.output <- d.latest:
            // Deliver when consumer is ready
            d.latest = nil
        }
    }
}
```

This pattern:
- Never blocks the camera
- Always provides the most recent frame
- Drops frames when consumer is slow

### Pipeline Architecture

Vision pipelines separate acquisition from processing:

```
┌──────────┐    ┌──────────┐    ┌──────────┐    ┌──────────┐
│  Camera  │───>│ Compress │───>│   NATS   │───>│ Detector │
│ (30 fps) │    │  (JPEG)  │    │ (topic)  │    │  (GPU)   │
└──────────┘    └──────────┘    └──────────┘    └──────────┘
     │
     │ (raw)
     ▼
┌──────────┐
│  Local   │
│ Display  │
└──────────┘
```

```go
// Camera node
func cameraNode(ctx context.Context) {
    camera := setupCamera()
    pub := pub.New[*vision.CompressedImage](node, "gorai.cameras.front.compressed")

    for img := range camera.Stream(ctx) {
        compressed := compress(img)
        pub.Publish(ctx, compressed)
    }
}

// Detector node (on GPU machine)
func detectorNode(ctx context.Context) {
    sub.New[*vision.CompressedImage](node, "gorai.cameras.front.compressed",
        func(msg *vision.CompressedImage) {
            img := decompress(msg)
            detections := model.Detect(img)
            detPub.Publish(ctx, detections)
        })
}
```
