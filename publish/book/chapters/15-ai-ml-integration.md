# AI/ML Integration

Modern robots increasingly rely on ML for perception and decision-making. Gorai treats AI as a first-class capability.

## The AI Opportunity in Robotics

**Edge inference vs cloud**:

- Edge: Low latency, works offline, privacy preserving
- Cloud: More compute, larger models, easier updates

Gorai focuses on edge inference—models running on the robot itself.

**Real-time requirements**:

- Object detection: 10-30 fps for navigation
- Voice commands: <500ms response
- Gesture recognition: <100ms for natural interaction

**Power constraints**:

- Mobile robots have limited battery
- Inference efficiency directly impacts runtime
- NPU/TPU provide 10x+ efficiency over CPU/GPU

## Hardware Accelerators

### NPU (Neural Processing Unit)

**RK3588 NPU** (6 TOPS):

- Found in Orange Pi 5, Rock 5B, Radxa
- Optimized for INT8 inference
- Gorai uses go-rknnlite bindings

```go
import "github.com/gorai/gorai/pkg/accel/rknn"

acc, err := rknn.New()
if err != nil {
    log.Fatal("NPU not available:", err)
}
defer acc.Close()

model, err := acc.Load(ctx, "yolov5s.rknn")
if err != nil {
    log.Fatal("Failed to load model:", err)
}
defer model.Close()
```

### GPU Acceleration

**NVIDIA CUDA** (Jetson):

```go
import "github.com/gorai/gorai/pkg/accel/cuda"

acc, err := cuda.New()
// Requires CUDA 12.x and cuDNN 9.x
```

**OpenCL** (generic):

```go
import "github.com/gorai/gorai/pkg/accel/opencl"

acc, err := opencl.New()
// Works on various GPUs
```

### TPU (Tensor Processing Unit)

**Google Coral**:

```go
import "github.com/gorai/gorai/pkg/accel/coral"

acc, err := coral.New()
// USB or M.2 Edge TPU
```

## Gorai's Acceleration Layer

The `accel` package provides a unified interface:

```go
// accel/accel.go
type Accelerator interface {
    Name() string
    Device() string
    Load(ctx context.Context, modelPath string) (Model, error)
    Close() error
}

type Model interface {
    Name() string
    Metadata() Metadata
    Infer(ctx context.Context, inputs map[string]Tensor) (map[string]Tensor, error)
    Close() error
}

type Tensor struct {
    Shape    []int
    DataType DataType  // Float32, Uint8, Int8
    Data     any
}
```

**Benefits**:

- Same code runs on different hardware
- Swap accelerators via configuration
- Fallback to CPU when accelerators unavailable

## Common ML Tasks

### Object Detection

Detect and locate objects in images:

```go
func detectObjects(img image.Image) (*vision.Detections, error) {
    // Preprocess
    input := preprocess(img)

    // Infer
    outputs, err := model.Infer(ctx, map[string]accel.Tensor{
        "images": input,
    })
    if err != nil {
        return nil, err
    }

    // Postprocess (NMS, decode boxes)
    detections := postprocess(outputs["output0"])
    return detections, nil
}

func preprocess(img image.Image) accel.Tensor {
    // Resize to model input size
    resized := resize.Resize(640, 640, img, resize.Lanczos3)

    // Convert to float32, normalize
    data := make([]float32, 640*640*3)
    for y := 0; y < 640; y++ {
        for x := 0; x < 640; x++ {
            r, g, b, _ := resized.At(x, y).RGBA()
            data[(y*640+x)*3+0] = float32(r>>8) / 255.0
            data[(y*640+x)*3+1] = float32(g>>8) / 255.0
            data[(y*640+x)*3+2] = float32(b>>8) / 255.0
        }
    }

    return accel.Tensor{
        Shape:    []int{1, 3, 640, 640},
        DataType: accel.Float32,
        Data:     data,
    }
}
```

### Classification

Identify what's in an image:

```go
func classify(img image.Image) (*vision.Classifications, error) {
    input := preprocessForClassification(img)

    outputs, _ := model.Infer(ctx, map[string]accel.Tensor{
        "input": input,
    })

    // Output is probability distribution
    probs := outputs["output"].([]float32)

    // Find top-k
    classifications := topK(probs, 5)
    return &vision.Classifications{
        Classifications: classifications,
    }, nil
}
```

### Pose Estimation

Detect human/object poses:

```go
type Pose struct {
    Keypoints []Keypoint
    Score     float64
}

type Keypoint struct {
    Name       string
    X, Y       float64
    Confidence float64
}

func estimatePose(img image.Image) ([]Pose, error) {
    input := preprocess(img)
    outputs, _ := model.Infer(ctx, inputs)

    // Decode keypoints
    poses := decodePoses(outputs)
    return poses, nil
}
```

## Model Deployment

### ONNX as Interchange

Convert models to ONNX for portability:

```python
# PyTorch to ONNX
torch.onnx.export(model, dummy_input, "model.onnx")

# TensorFlow to ONNX
python -m tf2onnx.convert --saved-model model_dir --output model.onnx
```

Then convert to target format:

```bash
# ONNX to RKNN (for RK3588 NPU)
python rknn_convert.py model.onnx model.rknn

# ONNX to TensorRT (for NVIDIA)
trtexec --onnx=model.onnx --saveEngine=model.trt
```

### Quantization for Edge

INT8 quantization reduces model size and speeds inference:

```python
# RKNN quantization
rknn.config(quantized_dtype='asymmetric_quantized-8')
rknn.load_onnx(model='model.onnx')
rknn.build(do_quantization=True, dataset='./calibration_data.txt')
```

Typical speedup: 2-4x over FP32, with <1% accuracy loss.

### Model Versioning

Track models like code:

```
models/
├── yolov5s/
│   ├── v1.0.0/
│   │   ├── model.rknn
│   │   ├── metadata.json
│   │   └── classes.txt
│   └── v1.1.0/
│       └── ...
└── mobilenet/
    └── ...
```

Configuration references version:

```json
{
    "model": "yolov5s",
    "version": "v1.0.0",
    "accelerator": "rknn"
}
```

## Vision Service Integration

Combine camera, accelerator, and model:

```go
type VisionService struct {
    camera Camera
    accel  accel.Accelerator
    model  accel.Model
}

func (v *VisionService) DetectObjects(ctx context.Context, img image.Image) (*Detections, error) {
    // Preprocess
    input := v.preprocess(img)

    // Infer
    outputs, err := v.model.Infer(ctx, map[string]accel.Tensor{
        "images": input,
    })
    if err != nil {
        return nil, err
    }

    // Postprocess
    return v.postprocess(outputs), nil
}

func (v *VisionService) DetectFromCamera(ctx context.Context) (*Detections, error) {
    img, err := v.camera.Image(ctx)
    if err != nil {
        return nil, err
    }
    return v.DetectObjects(ctx, img)
}
```

### Streaming Inference

Process frames continuously:

```go
func (v *VisionService) StreamDetections(ctx context.Context) (<-chan *Detections, error) {
    stream, err := v.camera.Stream(ctx)
    if err != nil {
        return nil, err
    }

    out := make(chan *Detections, 1)

    go func() {
        defer close(out)
        for img := range stream {
            detections, err := v.DetectObjects(ctx, img)
            if err != nil {
                continue
            }
            select {
            case out <- detections:
            default:
                // Drop if consumer is slow
            }
        }
    }()

    return out, nil
}
```

## Performance Optimization

### Batching

Process multiple images together:

```go
// Single image: 30ms
outputs, _ := model.Infer(ctx, inputs1)

// Batch of 4: 50ms total (12.5ms per image)
batchInputs := combineTensors(inputs1, inputs2, inputs3, inputs4)
outputs, _ := model.Infer(ctx, batchInputs)
```

### Async Inference

Don't block on slow models:

```go
type AsyncDetector struct {
    requests chan *DetectRequest
    results  chan *DetectResult
}

func (d *AsyncDetector) DetectAsync(img image.Image) <-chan *Detections {
    result := make(chan *Detections, 1)
    d.requests <- &DetectRequest{Image: img, Result: result}
    return result
}

func (d *AsyncDetector) worker(ctx context.Context) {
    for {
        select {
        case <-ctx.Done():
            return
        case req := <-d.requests:
            det, _ := d.detect(req.Image)
            req.Result <- det
            close(req.Result)
        }
    }
}
```

### Memory Management

Reuse buffers to avoid allocation:

```go
type InferencePool struct {
    inputBuffers  sync.Pool
    outputBuffers sync.Pool
}

func (p *InferencePool) GetInputBuffer() []float32 {
    if buf := p.inputBuffers.Get(); buf != nil {
        return buf.([]float32)
    }
    return make([]float32, 640*640*3)
}

func (p *InferencePool) PutInputBuffer(buf []float32) {
    p.inputBuffers.Put(buf)
}
```

## Choosing Hardware Accelerators

| Accelerator | TOPS | Power | Cost | Best For |
|-------------|------|-------|------|----------|
| RK3588 NPU | 6 | 5W | $100 | General edge AI |
| Coral TPU | 4 | 2W | $60 | Low power, USB |
| Jetson Orin Nano | 40 | 15W | $200 | Complex models |
| CPU (ARM Cortex-A76) | 0.5 | 3W | N/A | Fallback only |

**Selection criteria**:

- **Model size**: Larger models need more capable accelerators
- **Power budget**: Mobile robots need efficiency
- **Latency requirements**: Some accelerators have fixed overhead
- **Development ecosystem**: Consider tooling and documentation

---

Chapter 16 covers organizing your growing codebase as you add these AI capabilities.
