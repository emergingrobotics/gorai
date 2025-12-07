# Chapter 6: Components - Vision

Vision gives robots the ability to perceive and understand their environment. From simple obstacle detection to complex object recognition, cameras are increasingly central to robotic systems.

## 6.1 The Camera Interface

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
