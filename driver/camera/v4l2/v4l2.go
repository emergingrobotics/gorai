// Package v4l2 provides a V4L2 camera driver for Gorai.
package v4l2

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"log/slog"
	"sync"
	"time"

	"github.com/blackjack/webcam"
	"github.com/gorai/gorai/component/camera"
)

// Common V4L2 pixel format FourCC codes
const (
	FormatYUYV  = 0x56595559 // YUYV 4:2:2
	FormatMJPEG = 0x47504A4D // Motion-JPEG
)

// Config holds configuration for the V4L2 camera.
type Config struct {
	// Device path (e.g., "/dev/video0")
	Device string
	// Width in pixels
	Width uint32
	// Height in pixels
	Height uint32
	// FrameRate in frames per second
	FrameRate float64
	// JPEGQuality for encoding (1-100)
	JPEGQuality int
}

// DefaultConfig returns a default configuration.
func DefaultConfig() *Config {
	return &Config{
		Device:      "/dev/video0",
		Width:       640,
		Height:      480,
		FrameRate:   30,
		JPEGQuality: 80,
	}
}

// Camera implements the camera.Camera interface using V4L2.
type Camera struct {
	cfg    *Config
	logger *slog.Logger

	cam    *webcam.Webcam
	format webcam.PixelFormat

	mu       sync.RWMutex
	running  bool
	frameCh  chan []byte
	stopCh   chan struct{}
	doneCh   chan struct{}

	// Frame callback for publishing
	onFrame func(jpeg []byte, timestamp time.Time)
}

// Option configures a Camera.
type Option func(*Camera)

// WithLogger sets the logger.
func WithLogger(logger *slog.Logger) Option {
	return func(c *Camera) {
		c.logger = logger
	}
}

// WithOnFrame sets a callback for each captured frame.
func WithOnFrame(fn func(jpeg []byte, timestamp time.Time)) Option {
	return func(c *Camera) {
		c.onFrame = fn
	}
}

// New creates a new V4L2 camera.
func New(cfg *Config, opts ...Option) (*Camera, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	c := &Camera{
		cfg:     cfg,
		logger:  slog.Default(),
		frameCh: make(chan []byte, 2),
		stopCh:  make(chan struct{}),
		doneCh:  make(chan struct{}),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// Open opens the camera device.
func (c *Camera) Open() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	cam, err := webcam.Open(c.cfg.Device)
	if err != nil {
		return fmt.Errorf("failed to open camera %s: %w", c.cfg.Device, err)
	}
	c.cam = cam

	// Get supported formats
	formatDesc := cam.GetSupportedFormats()
	c.logger.Debug("Supported formats", "formats", formatDesc)

	// Prefer MJPEG if available, otherwise YUYV
	var selectedFormat webcam.PixelFormat
	for format := range formatDesc {
		if format == FormatMJPEG {
			selectedFormat = format
			break
		}
		if format == FormatYUYV {
			selectedFormat = format
		}
	}

	if selectedFormat == 0 {
		// Just use the first available format
		for format := range formatDesc {
			selectedFormat = format
			break
		}
	}

	if selectedFormat == 0 {
		cam.Close()
		return fmt.Errorf("no supported pixel format found")
	}

	c.format = selectedFormat
	c.logger.Info("Selected format", "format", formatDesc[selectedFormat])

	// Set format and size
	_, w, h, err := cam.SetImageFormat(selectedFormat, c.cfg.Width, c.cfg.Height)
	if err != nil {
		cam.Close()
		return fmt.Errorf("failed to set image format: %w", err)
	}

	c.logger.Info("Camera configured",
		"device", c.cfg.Device,
		"width", w,
		"height", h,
		"format", formatDesc[selectedFormat])

	return nil
}

// Start begins capturing frames.
func (c *Camera) Start(ctx context.Context) error {
	c.mu.Lock()
	if c.running {
		c.mu.Unlock()
		return fmt.Errorf("camera already running")
	}
	if c.cam == nil {
		c.mu.Unlock()
		return fmt.Errorf("camera not opened")
	}

	if err := c.cam.StartStreaming(); err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to start streaming: %w", err)
	}

	c.running = true
	c.stopCh = make(chan struct{})
	c.doneCh = make(chan struct{})
	c.mu.Unlock()

	// Start capture goroutine
	go c.captureLoop(ctx)

	c.logger.Info("Camera streaming started", "device", c.cfg.Device)
	return nil
}

// captureLoop continuously captures frames from the camera.
func (c *Camera) captureLoop(ctx context.Context) {
	defer close(c.doneCh)

	frameInterval := time.Duration(float64(time.Second) / c.cfg.FrameRate)
	ticker := time.NewTicker(frameInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			if err := c.captureFrame(); err != nil {
				c.logger.Warn("Frame capture error", "error", err)
			}
		}
	}
}

// captureFrame captures a single frame.
func (c *Camera) captureFrame() error {
	c.mu.RLock()
	cam := c.cam
	format := c.format
	c.mu.RUnlock()

	if cam == nil {
		return fmt.Errorf("camera not open")
	}

	// Read frame with timeout
	err := cam.WaitForFrame(uint32(time.Second / 10))
	if err != nil {
		return fmt.Errorf("wait for frame: %w", err)
	}

	frame, err := cam.ReadFrame()
	if err != nil {
		return fmt.Errorf("read frame: %w", err)
	}
	if len(frame) == 0 {
		return nil // Empty frame, skip
	}

	timestamp := time.Now()

	// Convert to JPEG if needed
	var jpegData []byte
	if format == FormatMJPEG {
		// Already JPEG
		jpegData = make([]byte, len(frame))
		copy(jpegData, frame)
	} else {
		// Convert YUYV to JPEG
		var err error
		jpegData, err = c.yuyvToJPEG(frame)
		if err != nil {
			return fmt.Errorf("convert to JPEG: %w", err)
		}
	}

	// Call frame callback if set
	if c.onFrame != nil {
		c.onFrame(jpegData, timestamp)
	}

	// Send to frame channel (non-blocking)
	select {
	case c.frameCh <- jpegData:
	default:
		// Channel full, drop frame
	}

	return nil
}

// yuyvToJPEG converts YUYV frame data to JPEG.
func (c *Camera) yuyvToJPEG(yuyv []byte) ([]byte, error) {
	width := int(c.cfg.Width)
	height := int(c.cfg.Height)

	// Create RGBA image
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Convert YUYV to RGBA
	for y := 0; y < height; y++ {
		for x := 0; x < width; x += 2 {
			i := (y*width + x) * 2
			if i+3 >= len(yuyv) {
				break
			}

			y0 := int(yuyv[i])
			u := int(yuyv[i+1])
			y1 := int(yuyv[i+2])
			v := int(yuyv[i+3])

			// Convert YUV to RGB for first pixel
			r0, g0, b0 := yuvToRGB(y0, u, v)
			offset0 := (y*width + x) * 4
			img.Pix[offset0] = r0
			img.Pix[offset0+1] = g0
			img.Pix[offset0+2] = b0
			img.Pix[offset0+3] = 255

			// Convert YUV to RGB for second pixel
			r1, g1, b1 := yuvToRGB(y1, u, v)
			offset1 := (y*width + x + 1) * 4
			img.Pix[offset1] = r1
			img.Pix[offset1+1] = g1
			img.Pix[offset1+2] = b1
			img.Pix[offset1+3] = 255
		}
	}

	// Encode to JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: c.cfg.JPEGQuality}); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// yuvToRGB converts YUV values to RGB.
func yuvToRGB(y, u, v int) (uint8, uint8, uint8) {
	// Standard YUV to RGB conversion
	c := y - 16
	d := u - 128
	e := v - 128

	r := clamp((298*c + 409*e + 128) >> 8)
	g := clamp((298*c - 100*d - 208*e + 128) >> 8)
	b := clamp((298*c + 516*d + 128) >> 8)

	return uint8(r), uint8(g), uint8(b)
}

// clamp clamps a value to 0-255 range.
func clamp(x int) int {
	if x < 0 {
		return 0
	}
	if x > 255 {
		return 255
	}
	return x
}

// Stop stops capturing frames.
func (c *Camera) Stop() error {
	c.mu.Lock()
	if !c.running {
		c.mu.Unlock()
		return nil
	}
	c.running = false
	close(c.stopCh)
	c.mu.Unlock()

	// Wait for capture loop to finish
	<-c.doneCh

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cam != nil {
		c.cam.StopStreaming()
	}

	c.logger.Info("Camera streaming stopped", "device", c.cfg.Device)
	return nil
}

// Close closes the camera device.
func (c *Camera) Close() error {
	c.Stop()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.cam != nil {
		c.cam.Close()
		c.cam = nil
	}

	c.logger.Info("Camera closed", "device", c.cfg.Device)
	return nil
}

// Frame returns a channel that receives JPEG frames.
func (c *Camera) Frame() <-chan []byte {
	return c.frameCh
}

// Image implements camera.Camera interface.
func (c *Camera) Image(ctx context.Context) (image.Image, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case frame := <-c.frameCh:
		return jpeg.Decode(bytes.NewReader(frame))
	}
}

// Stream implements camera.Camera interface.
func (c *Camera) Stream(ctx context.Context) (<-chan image.Image, error) {
	imgCh := make(chan image.Image, 2)

	go func() {
		defer close(imgCh)
		for {
			select {
			case <-ctx.Done():
				return
			case frame, ok := <-c.frameCh:
				if !ok {
					return
				}
				img, err := jpeg.Decode(bytes.NewReader(frame))
				if err != nil {
					c.logger.Warn("Failed to decode frame", "error", err)
					continue
				}
				select {
				case imgCh <- img:
				default:
				}
			}
		}
	}()

	return imgCh, nil
}

// Properties implements camera.Camera interface.
func (c *Camera) Properties(ctx context.Context) (camera.Properties, error) {
	return camera.Properties{
		Width:     int(c.cfg.Width),
		Height:    int(c.cfg.Height),
		FrameRate: c.cfg.FrameRate,
	}, nil
}

// Name returns the component name.
func (c *Camera) Name() string {
	return c.cfg.Device
}
