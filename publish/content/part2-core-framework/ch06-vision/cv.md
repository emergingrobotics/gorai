## 6.4 Computer Vision Integration

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

*Cross-reference: See Chapter 12 for ML-based vision using the acceleration layer.*


## 6.5 Depth Sensing

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

With component types covered—sensors, actuators, and cameras—Chapter 7 explores services: the software capabilities that process component data and coordinate robot behavior.
