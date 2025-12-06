// Package fake provides a fake camera implementation for testing.
package fake

import (
	"context"
	"image"
	"image/color"

	"github.com/gorai/gorai/component/camera"
	"github.com/gorai/gorai/pkg/registry"
)

func init() {
	registry.RegisterComponent("camera", "fake", New)
}

// Camera is a fake camera for testing.
type Camera struct {
	name   string
	width  int
	height int
}

// New creates a new fake camera.
func New(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
	name, _ := conf["name"].(string)
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

// Name returns the camera's name.
func (c *Camera) Name() string {
	return c.name
}

// Reconfigure updates the camera configuration.
func (c *Camera) Reconfigure(ctx context.Context, conf map[string]any) error {
	if w, ok := conf["width"].(float64); ok {
		c.width = int(w)
	}
	if h, ok := conf["height"].(float64); ok {
		c.height = int(h)
	}
	return nil
}

// Close releases resources.
func (c *Camera) Close(ctx context.Context) error {
	return nil
}

// Image returns a test pattern image.
func (c *Camera) Image(ctx context.Context) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, c.width, c.height))

	// Create a simple gradient test pattern
	for y := 0; y < c.height; y++ {
		for x := 0; x < c.width; x++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x * 255 / c.width),
				G: uint8(y * 255 / c.height),
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
	return camera.Properties{
		Width:     c.width,
		Height:    c.height,
		FrameRate: 30,
	}, nil
}

// Verify interface compliance.
var _ camera.Camera = (*Camera)(nil)
