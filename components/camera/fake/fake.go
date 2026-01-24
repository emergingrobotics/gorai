// Package fake provides a fake camera implementation for testing.
package fake

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"sync"

	"github.com/gorai/gorai/components/camera"
	"github.com/gorai/gorai/pkg/registry"
	"github.com/gorai/gorai/pkg/resource"
)

func init() {
	registry.RegisterComponent("camera", "fake", New)
}

// Camera is a fake camera for testing.
type Camera struct {
	name   resource.Name
	mu     sync.RWMutex
	width  int
	height int
}

// New creates a new fake camera.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	nameStr, _ := conf["name"].(string)
	name := resource.NewComponentName("gorai", "camera", nameStr)

	width := 640
	height := 480

	if w, ok := conf["width"].(float64); ok {
		width = int(w)
	}
	if h, ok := conf["height"].(float64); ok {
		height = int(h)
	}

	return &Camera{
		name:   name,
		width:  width,
		height: height,
	}, nil
}

// NewWithName creates a fake camera with a specific resource name.
func NewWithName(name resource.Name, width, height int) *Camera {
	if width == 0 {
		width = 640
	}
	if height == 0 {
		height = 480
	}
	return &Camera{
		name:   name,
		width:  width,
		height: height,
	}
}

// Name returns the camera's resource name.
func (c *Camera) Name() resource.Name {
	return c.name
}

// Reconfigure updates the camera configuration.
func (c *Camera) Reconfigure(ctx context.Context, deps resource.Dependencies, conf resource.Config) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if w, ok := conf.GetInt("width"); ok {
		c.width = w
	}
	if h, ok := conf.GetInt("height"); ok {
		c.height = h
	}
	return nil
}

// DoCommand executes arbitrary commands for extensibility.
func (c *Camera) DoCommand(ctx context.Context, cmd map[string]any) (map[string]any, error) {
	if cmdName, ok := cmd["command"].(string); ok {
		switch cmdName {
		case "get_resolution":
			c.mu.RLock()
			defer c.mu.RUnlock()
			return map[string]any{
				"width":  c.width,
				"height": c.height,
			}, nil
		case "set_resolution":
			c.mu.Lock()
			defer c.mu.Unlock()
			if w, ok := cmd["width"].(float64); ok {
				c.width = int(w)
			}
			if h, ok := cmd["height"].(float64); ok {
				c.height = int(h)
			}
			return map[string]any{"status": "ok"}, nil
		}
	}
	return nil, fmt.Errorf("unknown command: %v", cmd)
}

// Close releases resources.
func (c *Camera) Close(ctx context.Context) error {
	return nil
}

// Image returns a test pattern image.
func (c *Camera) Image(ctx context.Context) (image.Image, error) {
	c.mu.RLock()
	width, height := c.width, c.height
	c.mu.RUnlock()

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	// Create a simple gradient test pattern
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / width),
				G: uint8(y * 255 / height),
				B: 128,
				A: 255,
			})
		}
	}

	return img, nil
}

// Stream returns a channel of images.
func (c *Camera) Stream(ctx context.Context) (<-chan image.Image, error) {
	ch := make(chan image.Image)

	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				img, _ := c.Image(ctx)
				select {
				case ch <- img:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return ch, nil
}

// Properties returns the camera properties.
func (c *Camera) Properties(ctx context.Context) (camera.Properties, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return camera.Properties{
		Width:     c.width,
		Height:    c.height,
		FrameRate: 30,
	}, nil
}

// Verify interface compliance.
var _ camera.Camera = (*Camera)(nil)
