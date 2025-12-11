# Vision

Vision gives robots the ability to perceive and understand their environment. From simple obstacle detection to complex object recognition, cameras are increasingly central to robotic systems.

Gorai's camera interface provides a consistent way to capture images from USB cameras, CSI cameras, IP cameras, and depth sensors. This chapter covers the Camera interface, different camera types, how image data flows through the system, integration with computer vision libraries, and depth sensing.

## The Camera Interface

Cameras bridge the physical world and digital processing:

```go
type Camera interface {
    component.Component

    // Image captures and returns the current frame.
    Image(ctx context.Context) (image.Image, error)

    // Stream returns a channel of continuous frames.
    Stream(ctx context.Context) (<-chan image.Image, error)

    // Properties returns camera intrinsics and capabilities.
    Properties(ctx context.Context) (Properties, error)
}
```

### Image Capture

Single frame capture for on-demand processing:

```go
camera, _ := camera.New(node, cameraConfig)

// Capture a single frame
img, err := camera.Image(ctx)
if err != nil {
    log.Printf("capture failed: %v", err)
    return
}

// img is a Go standard library image.Image
bounds := img.Bounds()
log.Printf("Captured %dx%d image", bounds.Dx(), bounds.Dy())
```

### Image Formats

Go's `image.Image` interface supports multiple formats:

```go
switch img := img.(type) {
case *image.RGBA:
    // 8-bit RGBA
    pixel := img.RGBAAt(x, y)

case *image.Gray:
    // 8-bit grayscale
    pixel := img.GrayAt(x, y)

case *image.YCbCr:
    // YUV format (common from cameras)
    y := img.Y[img.YOffset(x, y)]
}
```

### Resolution and Frame Rate

Camera properties describe capabilities:

```go
type Properties struct {
    Width       int
    Height      int
    FrameRate   float64
    PixelFormat string    // "rgb8", "bgr8", "yuv422", etc.

    // Intrinsic parameters for 3D projection
    FocalLengthX  float64
    FocalLengthY  float64
    PrincipalX    float64
    PrincipalY    float64
    DistortionK   []float64  // Radial distortion
    DistortionP   []float64  // Tangential distortion
}
```

### Intrinsic Parameters

Camera intrinsics describe how 3D points project to 2D pixels:

```
┌─────────────────────────────────┐
│  Camera Intrinsic Matrix (K)    │
│                                 │
│  [ fx   0   cx ]                │
│  [  0  fy   cy ]                │
│  [  0   0    1 ]                │
│                                 │
│  fx, fy: Focal lengths (pixels) │
│  cx, cy: Principal point        │
└─────────────────────────────────┘
```

Use intrinsics to:

- Convert pixel coordinates to rays in 3D
- Undistort images
- Compute 3D positions from stereo or depth

```go
// Project 3D point to pixel
func (c *Camera) Project(point r3.Vector) (x, y float64) {
    props := c.Properties(ctx)
    x = props.FocalLengthX*point.X/point.Z + props.PrincipalX
    y = props.FocalLengthY*point.Y/point.Z + props.PrincipalY
    return
}
```

## Camera Types

Different camera technologies serve different needs.

### USB Cameras

The simplest option—plug and play via V4L2 (Video4Linux2):

```go
type USBCamera struct {
    device     string      // "/dev/video0"
    width      int
    height     int
    frameRate  int
    v4l2Device *v4l2.Device
}

func NewUSBCamera(device string, width, height, fps int) (*USBCamera, error) {
    dev, err := v4l2.Open(device)
    if err != nil {
        return nil, err
    }

    // Set format
    err = dev.SetFormat(v4l2.PixelFormatMJPEG, width, height)
    if err != nil {
        return nil, err
    }

    // Set frame rate
    err = dev.SetFrameRate(fps)
    if err != nil {
        return nil, err
    }

    return &USBCamera{
        device:     device,
        width:      width,
        height:     height,
        frameRate:  fps,
        v4l2Device: dev,
    }, nil
}
```

**Common USB cameras**:

- Logitech C920/C930: Good quality, wide compatibility
- ELP cameras: Inexpensive, various form factors
- Intel RealSense: RGB-D cameras

### CSI Cameras (Raspberry Pi)

Camera Serial Interface provides higher bandwidth:

```go
// Raspberry Pi camera via libcamera
type CSICamera struct {
    width     int
    height    int
    camera    *libcamera.Camera
}

func NewCSICamera(width, height int) (*CSICamera, error) {
    cam, err := libcamera.Open()
    if err != nil {
        return nil, err
    }

    config := libcamera.VideoConfiguration{
        Width:     width,
        Height:    height,
        PixelFmt:  libcamera.RGB888,
        BufferCnt: 4,
    }

    if err := cam.Configure(config); err != nil {
        return nil, err
    }

    return &CSICamera{
        width:  width,
        height: height,
        camera: cam,
    }, nil
}
```

**Advantages of CSI**:

- Lower CPU overhead (direct memory access)
- Higher frame rates (60+ fps)
- Lower latency
- GPU acceleration on Pi

**Common CSI cameras**:

- Raspberry Pi Camera Module v2/v3
- Arducam variety
- IMX477 (HQ Camera)

### IP Cameras

Network cameras for remote or distributed sensing:

```go
type IPCamera struct {
    url    string
    stream *gocv.VideoCapture
}

func NewIPCamera(rtspURL string) (*IPCamera, error) {
    stream, err := gocv.OpenVideoCapture(rtspURL)
    if err != nil {
        return nil, err
    }

    return &IPCamera{
        url:    rtspURL,
        stream: stream,
    }, nil
}

func (c *IPCamera) Image(ctx context.Context) (image.Image, error) {
    mat := gocv.NewMat()
    defer mat.Close()

    if ok := c.stream.Read(&mat); !ok {
        return nil, fmt.Errorf("failed to read frame")
    }

    return mat.ToImage()
}
```

**Use cases**:

- Remote monitoring
- Multi-camera systems
- PTZ (pan-tilt-zoom) cameras

### Depth Cameras (RGB-D)

Cameras that provide depth information:

```go
type DepthCamera interface {
    Camera

    // DepthImage returns per-pixel depth values.
    DepthImage(ctx context.Context) (*DepthMap, error)

    // PointCloud returns 3D point cloud.
    PointCloud(ctx context.Context) (*PointCloud, error)
}

type DepthMap struct {
    Width  int
    Height int
    Data   []float32  // Depth in meters per pixel
}
```

**Technologies**:

- **Stereo**: Two cameras, compute depth from disparity
- **Structured light**: Project pattern, measure deformation
- **ToF (Time of Flight)**: Measure light round-trip time
- **LiDAR**: Scanning laser measurement

**Common depth cameras**:

- Intel RealSense D415/D435/D455
- Azure Kinect
- Orbbec Astra
- OAK-D (with NPU)

## Image Data Flow

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

## Computer Vision Integration

Go isn't traditionally known for computer vision, but excellent bindings exist.

### OpenCV with Go (GoCV)

[GoCV](https://gocv.io/) provides comprehensive OpenCV bindings:

```go
import "gocv.io/x/gocv"

// Read image
mat := gocv.IMRead("image.jpg", gocv.IMReadColor)
defer mat.Close()

// Convert color space
gray := gocv.NewMat()
defer gray.Close()
gocv.CvtColor(mat, &gray, gocv.ColorBGRToGray)

// Edge detection
edges := gocv.NewMat()
defer edges.Close()
gocv.Canny(gray, &edges, 50, 100)

// Save result
gocv.IMWrite("edges.jpg", edges)
```

### Frame Processing Pipelines

Common vision operations:

```go
type VisionPipeline struct {
    camera Camera
    stages []ProcessingStage
}

type ProcessingStage interface {
    Process(img gocv.Mat) gocv.Mat
}

// Preprocessing stage
type Preprocessor struct {
    targetWidth  int
    targetHeight int
}

func (p *Preprocessor) Process(img gocv.Mat) gocv.Mat {
    // Resize
    resized := gocv.NewMat()
    gocv.Resize(img, &resized, image.Point{p.targetWidth, p.targetHeight}, 0, 0, gocv.InterpolationLinear)

    // Normalize
    normalized := gocv.NewMat()
    resized.ConvertTo(&normalized, gocv.MatTypeCV32F)
    normalized.DivideFloat(255.0)

    resized.Close()
    return normalized
}
```

### Common Operations

**Color detection**:

```go
func detectColor(img gocv.Mat, lower, upper gocv.Scalar) gocv.Mat {
    hsv := gocv.NewMat()
    defer hsv.Close()
    gocv.CvtColor(img, &hsv, gocv.ColorBGRToHSV)

    mask := gocv.NewMat()
    gocv.InRangeWithScalar(hsv, lower, upper, &mask)

    return mask
}

// Find red objects
redMask := detectColor(img,
    gocv.Scalar{Val1: 0, Val2: 100, Val3: 100},   // Lower HSV
    gocv.Scalar{Val1: 10, Val2: 255, Val3: 255})  // Upper HSV
```

**Feature detection**:

```go
func detectFeatures(img gocv.Mat) []gocv.KeyPoint {
    gray := gocv.NewMat()
    defer gray.Close()
    gocv.CvtColor(img, &gray, gocv.ColorBGRToGray)

    orb := gocv.NewORB()
    defer orb.Close()

    keypoints := orb.Detect(gray)
    return keypoints
}
```

**Contour finding**:

```go
func findContours(binary gocv.Mat) [][]image.Point {
    contours := gocv.FindContours(binary, gocv.RetrievalExternal, gocv.ChainApproxSimple)

    // Filter by area
    var significant [][]image.Point
    for _, contour := range contours {
        area := gocv.ContourArea(contour)
        if area > 100 {  // Minimum area threshold
            significant = append(significant, contour)
        }
    }
    return significant
}
```

## Depth Sensing

Depth cameras unlock 3D perception.

### Point Cloud Generation

Convert depth images to 3D points:

```go
func depthToPointCloud(depth *DepthMap, intrinsics *CameraIntrinsics) *PointCloud {
    points := make([]r3.Vector, 0, depth.Width*depth.Height)

    for y := 0; y < depth.Height; y++ {
        for x := 0; x < depth.Width; x++ {
            d := depth.At(x, y)
            if d <= 0 || d > 10.0 {  // Invalid or too far
                continue
            }

            // Back-project to 3D
            px := (float64(x) - intrinsics.Cx) / intrinsics.Fx * d
            py := (float64(y) - intrinsics.Cy) / intrinsics.Fy * d
            pz := d

            points = append(points, r3.Vector{X: px, Y: py, Z: pz})
        }
    }

    return &PointCloud{Points: points}
}
```

### Depth Image Formats

```go
type DepthMap struct {
    Width   int
    Height  int
    Data    []float32   // Meters per pixel
    MinDist float32     // Minimum valid distance
    MaxDist float32     // Maximum valid distance
}

func (d *DepthMap) At(x, y int) float32 {
    return d.Data[y*d.Width+x]
}

func (d *DepthMap) IsValid(x, y int) bool {
    v := d.At(x, y)
    return v >= d.MinDist && v <= d.MaxDist
}
```

### Registration with RGB

Aligning color and depth images:

```go
type RGBDFrame struct {
    Color     image.Image
    Depth     *DepthMap
    Transform mat4.Mat4  // Depth to color transform
}

// Project depth point to color pixel
func (f *RGBDFrame) DepthToColor(x, y int) (cx, cy int) {
    d := f.Depth.At(x, y)

    // 3D point in depth frame
    point := backProject(x, y, d, f.depthIntrinsics)

    // Transform to color frame
    colorPoint := f.Transform.MulVec(point)

    // Project to color image
    cx, cy = project(colorPoint, f.colorIntrinsics)
    return
}
```

---

With component types covered—sensors, actuators, and cameras—Chapter 8 explores services: the software capabilities that process component data and coordinate robot behavior.
