// Example camera demonstrates using the camera component interface.
package main

import (
	"context"
	"fmt"
	"os"

	"github.com/gorai/gorai/component/camera/fake"
	"github.com/gorai/gorai/pkg/registry"
)

func main() {
	ctx := context.Background()

	// Create a fake camera for demonstration
	c, err := fake.New(ctx, nil, registry.Config{
		"name":   "demo_camera",
		"width":  640.0,
		"height": 480.0,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create camera: %v\n", err)
		os.Exit(1)
	}

	camera := c.(*fake.Camera)
	defer camera.Close(ctx)

	fmt.Println("Camera demo starting...")

	// Get properties
	props, _ := camera.Properties(ctx)
	fmt.Printf("Camera resolution: %dx%d @ %.1f fps\n", props.Width, props.Height, props.FrameRate)

	// Capture an image
	fmt.Println("Capturing image...")
	img, err := camera.Image(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to capture image: %v\n", err)
		os.Exit(1)
	}

	bounds := img.Bounds()
	fmt.Printf("Captured image: %dx%d pixels\n", bounds.Dx(), bounds.Dy())

	// Sample a pixel
	r, g, b, a := img.At(320, 240).RGBA()
	fmt.Printf("Pixel at center (320,240): R=%d G=%d B=%d A=%d\n", r>>8, g>>8, b>>8, a>>8)

	fmt.Println("Camera demo complete.")
}
