## 6.2 Camera Types

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
