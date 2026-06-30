// Package cpu provides a CPU-based reference implementation of the accelerator interface.
//
// This implementation is primarily for testing and development. For production
// inference, use hardware-accelerated backends (TPU, NPU, CUDA).
package cpu

import (
	"context"
	"fmt"

	"github.com/emergingrobotics/gorai/pkg/accel"
	"github.com/emergingrobotics/gorai/pkg/registry"
)

func init() {
	// Register the CPU accelerator
	registry.RegisterService("accelerator", "cpu", func(ctx context.Context, deps registry.Dependencies, conf registry.Config) (any, error) {
		return New(), nil
	})
}

// Accelerator is a CPU-based accelerator.
type Accelerator struct{}

// New creates a new CPU accelerator.
func New() *Accelerator {
	return &Accelerator{}
}

// Name returns the accelerator name.
func (a *Accelerator) Name() string {
	return "cpu"
}

// Device returns the device identifier.
func (a *Accelerator) Device() string {
	return "cpu:0"
}

// Load loads a model from the given path.
// Note: The CPU accelerator is a stub - real implementations would use
// ONNX Runtime, TFLite, or other inference engines.
func (a *Accelerator) Load(ctx context.Context, modelPath string) (accel.Model, error) {
	return nil, fmt.Errorf("CPU accelerator: model loading not implemented - use a hardware-specific accelerator")
}

// Close releases accelerator resources.
func (a *Accelerator) Close() error {
	return nil
}

// Verify interface compliance.
var _ accel.Accelerator = (*Accelerator)(nil)
