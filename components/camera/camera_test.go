package camera_test

import (
	"context"
	"testing"
	"time"

	"github.com/emergingrobotics/gorai/components"
	"github.com/emergingrobotics/gorai/components/camera"
	"github.com/emergingrobotics/gorai/components/camera/fake"
	"github.com/emergingrobotics/gorai/pkg/resource"
)

func TestCamera_IsComponent(t *testing.T) {
	// Camera must implement component.Component
	var _ component.Component = (camera.Camera)(nil)
}

func TestFakeCamera_Image(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "camera", "test")
	c := fake.NewWithName(name, 640, 480)

	img, err := c.Image(ctx)
	if err != nil {
		t.Fatalf("Image failed: %v", err)
	}

	bounds := img.Bounds()
	if bounds.Dx() != 640 || bounds.Dy() != 480 {
		t.Errorf("image size = %dx%d, want 640x480", bounds.Dx(), bounds.Dy())
	}
}

func TestFakeCamera_Properties(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "camera", "test")
	c := fake.NewWithName(name, 1280, 720)

	props, err := c.Properties(ctx)
	if err != nil {
		t.Fatalf("Properties failed: %v", err)
	}

	if props.Width != 1280 {
		t.Errorf("Width = %d, want 1280", props.Width)
	}
	if props.Height != 720 {
		t.Errorf("Height = %d, want 720", props.Height)
	}
	if props.FrameRate != 30 {
		t.Errorf("FrameRate = %v, want 30", props.FrameRate)
	}
}

func TestFakeCamera_Stream(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	name := resource.NewComponentName("gorai", "camera", "test")
	c := fake.NewWithName(name, 320, 240)

	stream, err := c.Stream(ctx)
	if err != nil {
		t.Fatalf("Stream failed: %v", err)
	}

	// Read at least one frame
	select {
	case img := <-stream:
		if img == nil {
			t.Error("received nil image from stream")
		}
		bounds := img.Bounds()
		if bounds.Dx() != 320 || bounds.Dy() != 240 {
			t.Errorf("streamed image size = %dx%d, want 320x240", bounds.Dx(), bounds.Dy())
		}
	case <-ctx.Done():
		t.Error("timeout waiting for stream frame")
	}
}

func TestFakeCamera_Name(t *testing.T) {
	name := resource.NewComponentName("gorai", "camera", "front_cam")
	c := fake.NewWithName(name, 0, 0)

	if c.Name().String() != "gorai:component:camera/front_cam" {
		t.Errorf("Name() = %q, want 'gorai:component:camera/front_cam'", c.Name().String())
	}
}

func TestFakeCamera_DefaultResolution(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "camera", "test")
	c := fake.NewWithName(name, 0, 0) // Use defaults

	props, err := c.Properties(ctx)
	if err != nil {
		t.Fatalf("Properties failed: %v", err)
	}

	if props.Width != 640 {
		t.Errorf("default Width = %d, want 640", props.Width)
	}
	if props.Height != 480 {
		t.Errorf("default Height = %d, want 480", props.Height)
	}
}

func TestFakeCamera_Reconfigure(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "camera", "test")
	c := fake.NewWithName(name, 640, 480)

	// Reconfigure to new resolution
	conf := resource.NewConfig(map[string]any{
		"width":  800.0,
		"height": 600.0,
	})
	if err := c.Reconfigure(ctx, resource.EmptyDependencies(), conf); err != nil {
		t.Fatalf("Reconfigure failed: %v", err)
	}

	props, err := c.Properties(ctx)
	if err != nil {
		t.Fatalf("Properties failed: %v", err)
	}

	if props.Width != 800 {
		t.Errorf("Width after reconfigure = %d, want 800", props.Width)
	}
	if props.Height != 600 {
		t.Errorf("Height after reconfigure = %d, want 600", props.Height)
	}
}

func TestFakeCamera_DoCommand(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "camera", "test")
	c := fake.NewWithName(name, 640, 480)

	// Test get_resolution command
	result, err := c.DoCommand(ctx, map[string]any{
		"command": "get_resolution",
	})
	if err != nil {
		t.Fatalf("DoCommand get_resolution failed: %v", err)
	}

	if result["width"].(int) != 640 {
		t.Errorf("width = %v, want 640", result["width"])
	}
	if result["height"].(int) != 480 {
		t.Errorf("height = %v, want 480", result["height"])
	}

	// Test set_resolution command
	_, err = c.DoCommand(ctx, map[string]any{
		"command": "set_resolution",
		"width":   1920.0,
		"height":  1080.0,
	})
	if err != nil {
		t.Fatalf("DoCommand set_resolution failed: %v", err)
	}

	result, _ = c.DoCommand(ctx, map[string]any{"command": "get_resolution"})
	if result["width"].(int) != 1920 {
		t.Errorf("width after set = %v, want 1920", result["width"])
	}

	// Test unknown command
	_, err = c.DoCommand(ctx, map[string]any{
		"command": "unknown",
	})
	if err == nil {
		t.Error("expected error for unknown command")
	}
}

func TestFakeCamera_Close(t *testing.T) {
	ctx := context.Background()
	name := resource.NewComponentName("gorai", "camera", "test")
	c := fake.NewWithName(name, 640, 480)

	// Close should succeed
	if err := c.Close(ctx); err != nil {
		t.Fatalf("Close failed: %v", err)
	}
}
