# Go AI Ecosystem for Robotics

This document provides a comprehensive overview of the Go ecosystem for AI, machine learning, computer vision, and hardware acceleration relevant to robotics applications. This serves as a reference for Gorai's AI integration strategy.

## Table of Contents

1. [Machine Learning Frameworks](#machine-learning-frameworks)
2. [Inference Runtimes](#inference-runtimes)
3. [Computer Vision](#computer-vision)
4. [LLM and Speech](#llm-and-speech)
5. [Hardware Acceleration](#hardware-acceleration)
6. [Vector Databases and Embeddings](#vector-databases-and-embeddings)
7. [Robotics-Specific Libraries](#robotics-specific-libraries)
8. [Image Generation](#image-generation)
9. [Assessment and Recommendations](#assessment-and-recommendations)

---

## Machine Learning Frameworks

### GoMLX - Accelerated ML Framework

[GoMLX](https://github.com/gomlx/gomlx) is the most promising full-featured ML framework for Go, using OpenXLA for hardware acceleration.

**Key Features:**
- Pure Go backend (runs everywhere including WASM/browser)
- OpenXLA/PJRT backend for GPU/TPU acceleration (same engine as JAX, TensorFlow, PyTorch/XLA)
- Automatic differentiation for training
- Distributed execution across multiple GPUs/TPUs
- Jupyter notebook support
- ONNX model conversion via [onnx-gomlx](https://github.com/gomlx/onnx-gomlx)

**Platform Support:**
- CPU PJRT on Linux/amd64, Windows+WSL, Darwin (macOS)
- Nvidia CUDA PJRT on Linux/amd64
- Google Cloud TPUs
- Apple Metal GPU support is desired but not yet implemented

**Use Case:** Full ML training and inference in Go without Python dependencies.

```go
// GoMLX example - defining a simple model
func MyModel(ctx *context.Context, inputs []*Node) *Node {
    x := inputs[0]
    x = layers.Dense(ctx.In("layer1"), x, 128, true)
    x = activations.Relu(x)
    x = layers.Dense(ctx.In("output"), x, 10, true)
    return x
}
```

### Gorgonia

[Gorgonia](https://github.com/gorgonia/gorgonia) is a library for building and training ML models using computational graphs.

**Key Features:**
- Similar design to Theano/TensorFlow
- GPU acceleration via CUDA
- Automatic differentiation and backpropagation
- Supports CNNs, RNNs, and generative models
- Performance comparable to PyTorch/TensorFlow CPU implementations

**Use Case:** Building custom neural networks in pure Go with CUDA support.

### Gonum

[Gonum](https://github.com/gonum/gonum) provides numerical computing foundations, similar to NumPy.

**Capabilities:**
- Matrix manipulation and linear algebra
- Optimization algorithms
- Statistical analysis
- Foundation for implementing custom ML algorithms

**Use Case:** Scientific computing, implementing ML algorithms from scratch.

### GoLearn

[GoLearn](https://github.com/sjwhitworth/golearn) is a beginner-friendly ML library.

**Algorithms:**
- Decision trees
- K-Nearest Neighbors (KNN)
- Support Vector Machines (SVM)
- Naive Bayes

**Use Case:** Traditional ML algorithms for classification/regression tasks.

---

## Inference Runtimes

### ONNX Runtime for Go

[onnxruntime_go](https://github.com/yalue/onnxruntime_go) - The most actively maintained ONNX runtime wrapper.

**Key Features:**
- Wraps Microsoft ONNX Runtime v1.22.0
- CUDA 12.x and cuDNN 9.x support for GPU acceleration
- Generic tensor support via Go generics
- Imported by 78+ packages

**Usage:**
```go
import ort "github.com/yalue/onnxruntime_go"

session, _ := ort.NewSession(modelPath, nil)
defer session.Destroy()

inputTensor, _ := ort.NewTensor(inputShape, inputData)
outputTensor, _ := ort.NewEmptyTensor[float32](outputShape)

session.Run([]ort.AnyTensor{inputTensor}, []ort.AnyTensor{outputTensor})
```

**Use Case:** Running PyTorch/TensorFlow models exported to ONNX format.

### TensorFlow Lite for Go

Multiple options exist:

1. **[tflitego](https://github.com/nbortolotti/tflitego)** - Clean Go bindings for TFLite C API
2. **[go-tflite](https://github.com/mattn/go-tflite)** - Alternative TFLite bindings
3. **[graft](https://github.com/wamuir/graft)** - Official TensorFlow Go bindings successor

**Platform Support:**
- Linux/X86_64
- Raspberry Pi (ARMv7)

**Use Case:** Running TFLite models on edge devices.

---

## Computer Vision

### GoCV (OpenCV Bindings)

[GoCV](https://gocv.io/) provides comprehensive OpenCV 4 bindings for Go.

**Key Features:**
- Full OpenCV 4.12.0 support
- CUDA GPU acceleration
- OpenVINO support for Intel hardware
- DNN module for neural network inference
- OpenCV Contrib modules
- Used by 1,595+ projects

**Installation:** Requires matching OpenCV version installed on system.

**Example - YOLO Object Detection:**
```go
net := gocv.ReadNetFromDarknet(cfg, weights)
defer net.Close()

net.SetPreferableBackend(gocv.NetBackendCUDA)
net.SetPreferableTarget(gocv.NetTargetCUDA)

blob := gocv.BlobFromImage(img, 1.0/255.0, image.Pt(416, 416),
    gocv.NewScalar(0, 0, 0, 0), true, false)
net.SetInput(blob, "")
output := net.Forward("")
```

### Pigo - Pure Go Face Detection

[Pigo](https://github.com/esimov/pigo) is a lightweight, pure Go face detection library.

**Key Features:**
- No OpenCV or external dependencies
- Face detection, pupil/eye localization, facial landmarks
- WebAssembly support (runs in browsers)
- Based on Pixel Intensity Comparison-based Object detection
- Very fast - no image preprocessing required

**Use Case:** Lightweight face detection without C dependencies.

### YOLO Implementations

1. **[wimspaargaren/yolov3](https://github.com/wimspaargaren/yolov3)** - YOLOv3 with CUDA support (50 FPS)
2. **[wimspaargaren/yolov5](https://github.com/wimspaargaren/yolov5)** - YOLOv5 implementation
3. **[yolov8_onnx_go](https://github.com/AndreyGermanov/yolov8_onnx_go)** - YOLOv8 via ONNX Runtime
4. **GoCV DNN** - Built-in YOLO detection example

---

## LLM and Speech

### llama.cpp Go Bindings

Multiple options for running local LLMs:

1. **[go-llama.cpp](https://github.com/go-skynet/go-llama.cpp)** - High-level bindings, GGUF model support
2. **[ollama/llama](https://pkg.go.dev/github.com/ollama/ollama/llama)** - Ollama's native bindings
3. **[llama-go](https://github.com/tcpipuk/llama-go)** - Production-ready with thread-safe inference

**GPU Support:**
- NVIDIA (CUDA)
- AMD (ROCm)
- Apple Silicon (Metal)
- Intel (SYCL)
- Vulkan, OpenCL

**Example:**
```go
import llama "github.com/go-skynet/go-llama.cpp"

model, _ := llama.New(modelPath, llama.SetContext(2048))
defer model.Free()

response, _ := model.Predict("What is robotics?",
    llama.SetTokens(256),
    llama.SetTemperature(0.7))
```

### Whisper Speech Recognition

[go-whisper](https://github.com/mutablelogic/go-whisper) - Speech-to-text using whisper.cpp.

**Features:**
- Transcription and translation
- OpenAI, ElevenLabs, GGML providers
- HTTP API server with streaming
- GPU acceleration (CUDA, Vulkan, Metal)
- Model management

**Official Bindings:** `github.com/ggerganov/whisper.cpp/bindings/go/pkg/whisper`

**Use Case:** Voice commands, audio transcription for robots.

---

## Hardware Acceleration

> **Note:** Gorai targets Linux-based systems. Hardware acceleration support varies by platform.

### Google Coral Edge TPU

**Status:** No Go bindings exist. This is **aspirational** for Gorai.

The [libedgetpu](https://github.com/google-coral/libedgetpu) C library provides the runtime driver but has no official Go bindings. The [edgetpu_c.h](https://github.com/google-coral/edgetpu/blob/master/libedgetpu/edgetpu_c.h) header could be used for CGo bindings.

**What would be required:**
- Create CGo bindings to libedgetpu C API
- Or use RPC/subprocess bridge to Python SDK
- TFLite Go bindings do not currently support EdgeTPU delegate

**Hardware:** USB Accelerator (~$60), M.2/PCIe Accelerator (~$25), Dev Board

### Raspberry Pi AI Kit (Hailo NPU)

**Status:** No Go bindings exist. This is **aspirational** for Gorai.

The Hailo-8L (13 TOPS) NPU uses the [HailoRT](https://github.com/hailo-ai/hailort) SDK with C/C++ and Python interfaces only.

**What would be required:**
- Create CGo bindings to HailoRT C API
- The SDK is well-documented but no community Go bindings exist

**Hardware:** Hailo-8L M.2 module (~$70), Raspberry Pi AI Kit

### Rockchip RK3588 NPU

**Status:** Working Go bindings exist. This is **production-ready** for Gorai.

[go-rknnlite](https://github.com/swdee/go-rknnlite) provides CGo bindings for RKNN-Toolkit2, announced April 2024 and actively maintained.

**Supported Chips:** RK3562, RK3566, RK3568, RK3576, RK3582, RK3588

**Key Features:**
- Full RKNN C API bindings
- Multi-core NPU support (RK3588 has 3 NPU cores, 6 TOPS total)
- Runtime pooling for parallel inference
- CPU affinity optimization for performance
- Tested on Radxa Rock 5B and other RK3588 SBCs

**Performance:** Single runtime ~7.9ms, Pool of 9 runtimes ~1.65ms per image (EfficientNet-Lite0)

**Requirements:**
- Linux (tested on Armbian, Radxa OS)
- RKNN-Toolkit2 installed (`/usr/include/rknn_api.h`, `/usr/lib/librknnrt.so`)

**Example:**
```go
import "github.com/swdee/go-rknnlite"

runtime, _ := rknnlite.NewRuntime(modelPath, rknnlite.NPUCoreAuto)
defer runtime.Close()

outputs, _ := runtime.Inference(inputData)
```

**Hardware:** Orange Pi 5, Radxa Rock 5B, Khadas Edge 2, and other RK3588 boards (~$80-150)

### Intel OpenVINO

**Status:** Partial Go support via GoCV. May need updates for latest OpenVINO.

GoCV includes OpenVINO support via the `gocv.io/x/gocv/openvino/ie` package, but the integration was built against **OpenVINO 2022.1 LTS** and **OpenCV 4.5.5**. Current OpenVINO is 2024.x.

```go
net.SetPreferableBackend(gocv.NetBackendOpenVINO)
net.SetPreferableTarget(gocv.NetTargetCPU)  // Or NetTargetOpenCLFP16 for GPU
```

**Limitations:**
- Intel Neural Compute Stick 2 (VPU) support ended in OpenVINO 2023.0
- GoCV's OpenVINO integration may not support latest OpenVINO 2024.x features
- Requires building OpenCV from source with OpenVINO support

**Hardware:** Intel CPUs, Intel Arc GPUs, Intel integrated graphics

### NVIDIA CUDA

**Status:** Working Go support via multiple libraries. This is **production-ready** for Gorai.

Multiple libraries support CUDA on Linux:
- **[onnxruntime_go](https://github.com/yalue/onnxruntime_go)** - CUDA 12.x execution provider (requires CUDA 12.x, cuDNN 9.x, and CUDA-enabled onnxruntime library)
- **[GoCV](https://gocv.io/)** - CUDA backend for DNN module
- **Gorgonia** - CUDA tensor operations
- **llama-go** - CUDA for LLM inference

**Requirements:**
- Linux with NVIDIA GPU
- CUDA 12.x and cuDNN 9.x (for onnxruntime_go v1.12.0+)
- CUDA-enabled shared libraries (not included by default in onnxruntime_go)

**Note:** The default onnxruntime_go package does not include CUDA support. You must obtain CUDA-enabled onnxruntime libraries separately.

### Apple Metal

**Status:** Experimental. macOS is **not a primary target** for Gorai (Linux-focused).

[go-metal](https://github.com/tsawler/go-metal) - Deep learning library for Apple Silicon.

**Features:**
- GPU-accelerated tensor operations
- Automatic differentiation
- PyTorch-inspired API

**Other Options:**
- [green-aloe/metal](https://pkg.go.dev/github.com/green-aloe/metal) - GPGPU compute
- [dmitri.shuralyov.com/gpu/mtl](https://pkg.go.dev/dmitri.shuralyov.com/gpu/mtl) - Low-level Metal bindings

**Note:** While Go code can run on macOS for development, Gorai's primary deployment targets are Linux-based robots and embedded systems.

---

## Vector Databases and Embeddings

### Embedded Vector Databases

1. **[chromem-go](https://github.com/philippgille/chromem-go)**
   - Embeddable, zero dependencies
   - In-memory with optional persistence
   - Cosine similarity search
   - 1,000 docs in 0.3ms, 100,000 docs in 40ms

2. **[kelindar/search](https://github.com/kelindar/search)**
   - Embedded vector search using llama.cpp
   - Semantic embeddings

### External Vector Databases

1. **[Milvus Go SDK](https://zilliz.com/product/integrations/Go)**
   - Billion-scale vector similarity search
   - Open-source, production-ready

2. **pgvector with Go**
   - PostgreSQL extension for vector operations
   - Standard database drivers work

### Use Case

RAG (Retrieval Augmented Generation) for robot knowledge bases, semantic search over documentation or sensor logs.

---

## Robotics-Specific Libraries

### Gobot

[Gobot](https://gobot.io/) - Framework for robotics, drones, and IoT.

**Supported Interfaces:**
- GPIO, I2C, SPI, UART
- Analog I/O, Bluetooth LE
- OpenCV integration via GoCV

**Platforms:** Arduino, Raspberry Pi, BeagleBone, Intel Edison, drones, and more.

### TinyGo

[TinyGo](https://tinygo.org/) - Go compiler for microcontrollers.

**Targets:** 100+ microcontroller boards including:
- Arduino Uno, Nano
- BBC micro:bit
- ESP32, ESP8266
- Nordic nRF52
- STM32 series
- Raspberry Pi Pico

**Features:**
- LLVM-based compiler
- WebAssembly output
- Small binary sizes

### Sensor Fusion Libraries

1. **[aykevl/fusion](https://pkg.go.dev/github.com/aykevl/fusion)** - Gyroscope + accelerometer fusion
2. **[attestimator](https://github.com/knei-knurow/attestimator)** - IMU sensor fusion (Madgwick/Mahony)
3. **[go-kalmanfilter](https://github.com/shantanubhadoria/go-kalmanfilter)** - Kalman filter for sensor fusion
4. **[goflying/ahrs](https://pkg.go.dev/github.com/skarppi/goflying/ahrs)** - Aircraft state estimation (IMU + GPS)

**Note:** Full SLAM implementations are uncommon in Go; most use C++ (ORB-SLAM, RTAB-Map) or Python.

---

## Image Generation

### Stable Diffusion

1. **[seasonjs/stable-diffusion](https://github.com/seasonjs/stable-diffusion)** - Pure Go bindings, cross-platform
2. **[mudler/go-stable-diffusion](https://github.com/mudler/go-stable-diffusion)** - Wrapper for SD-NCNN

---

## Assessment and Recommendations

> **Platform:** Gorai targets Linux-based systems for deployment. macOS may be used for development.

### Strengths of Go AI Ecosystem

| Area | Maturity | Key Libraries | Status |
|------|----------|---------------|--------|
| ONNX Inference | Strong | onnxruntime_go | Production-ready |
| OpenCV/Vision | Strong | GoCV | Production-ready |
| RK3588 NPU | Good | go-rknnlite | Production-ready |
| NVIDIA CUDA | Good | onnxruntime_go, GoCV | Production-ready (requires setup) |
| LLM Inference | Growing | go-llama.cpp, ollama | Production-ready |
| Numerical Computing | Solid | Gonum | Production-ready |
| Embedded/IoT | Strong | TinyGo, Gobot | Production-ready |

### Gaps and Challenges

| Area | Status | Effort Required |
|------|--------|-----------------|
| Google Coral TPU | **No Go bindings exist** | High - CGo wrapper to libedgetpu |
| Hailo NPU | **No Go bindings exist** | High - CGo wrapper to HailoRT |
| Intel OpenVINO | Partial (GoCV uses 2022.1) | Medium - may need updates |
| Apple Metal ML | Experimental | N/A - not a target platform |
| Full SLAM | Limited | High - use C++ via CGo or service |
| Training on GPU | Limited | Medium - GoMLX (CUDA), Gorgonia |

### Recommended Strategy for Gorai

1. **Primary Inference Path:** Use ONNX Runtime for model inference - widest model compatibility
2. **Edge Acceleration (working today):**
   - RK3588: Use go-rknnlite (production-ready)
   - NVIDIA: Use onnxruntime_go with CUDA (production-ready, requires setup)
3. **Edge Acceleration (future work):**
   - Coral TPU: Requires developing CGo bindings to libedgetpu
   - Hailo NPU: Requires developing CGo bindings to HailoRT
4. **Computer Vision:** GoCV for full OpenCV functionality, Pigo for lightweight face detection
5. **LLM Integration:** go-llama.cpp or Ollama API for local LLMs
6. **ML Framework:** GoMLX for any Go-native training needs
7. **Embedded:** TinyGo for microcontroller targets, Gobot for robotics abstractions

### Hardware Acceleration Priority

For Linux-based robotics AI workloads, prioritize by Go support availability:

| Priority | Platform | Status | Notes |
|----------|----------|--------|-------|
| 1 | **RK3588 NPU** | Ready | Best Go support via go-rknnlite, 6 TOPS, $80-150 boards |
| 2 | **NVIDIA GPU** | Ready | CUDA via onnxruntime_go, requires CUDA 12.x setup |
| 3 | **x86 CPU** | Ready | onnxruntime_go or GoCV, no special hardware |
| 4 | **Intel GPU** | Partial | GoCV OpenVINO may need updates |
| 5 | **Coral TPU** | Not ready | CGo wrapper development needed |
| 6 | **Hailo NPU** | Not ready | CGo wrapper development needed |

---

## References

### Core ML Libraries
- [GoMLX](https://github.com/gomlx/gomlx)
- [Gorgonia](https://github.com/gorgonia/gorgonia)
- [Gonum](https://github.com/gonum/gonum)
- [GoLearn](https://github.com/sjwhitworth/golearn)

### Inference Runtimes
- [onnxruntime_go](https://github.com/yalue/onnxruntime_go)
- [tflitego](https://github.com/nbortolotti/tflitego)
- [go-tflite](https://github.com/mattn/go-tflite)

### Computer Vision
- [GoCV](https://gocv.io/)
- [Pigo](https://github.com/esimov/pigo)
- [YOLOv5 Go](https://github.com/wimspaargaren/yolov5)

### LLM and Speech
- [go-llama.cpp](https://github.com/go-skynet/go-llama.cpp)
- [go-whisper](https://github.com/mutablelogic/go-whisper)
- [Ollama](https://github.com/ollama/ollama)

### Hardware Acceleration
- [go-rknnlite](https://github.com/swdee/go-rknnlite) (RK3588)
- [go-metal](https://github.com/tsawler/go-metal) (Apple Silicon)
- [libedgetpu](https://github.com/google-coral/libedgetpu) (Coral - no Go bindings)

### Robotics
- [Gobot](https://gobot.io/)
- [TinyGo](https://tinygo.org/)

### Vector Databases
- [chromem-go](https://github.com/philippgille/chromem-go)
- [Milvus Go SDK](https://github.com/milvus-io/milvus-sdk-go)
