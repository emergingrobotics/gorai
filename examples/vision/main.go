// Example vision demonstrates using the vision service interface.
package main

import (
	"fmt"
)

func main() {
	fmt.Println("Vision example")
	fmt.Println("==============")
	fmt.Println()
	fmt.Println("This example demonstrates the vision service interface.")
	fmt.Println("To use real vision capabilities, you need to:")
	fmt.Println()
	fmt.Println("1. Install a vision service implementation:")
	fmt.Println("   go get github.com/gorai/gorai-service-vision-yolo")
	fmt.Println()
	fmt.Println("2. Import it in your code:")
	fmt.Println("   import _ \"github.com/gorai/gorai-service-vision-yolo\"")
	fmt.Println()
	fmt.Println("3. Configure it in your robot config:")
	fmt.Println(`   {
     "services": [
       {
         "name": "detector",
         "type": "vision",
         "model": "yolo",
         "attributes": {
           "model_path": "/path/to/yolov8n.onnx",
           "confidence_threshold": 0.5
         }
       }
     ]
   }`)
	fmt.Println()
	fmt.Println("See docs/go-ai-material.md for available AI libraries.")
}
