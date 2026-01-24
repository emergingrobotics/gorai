// Package accel provides hardware acceleration interfaces for ML inference.
//
// The acceleration layer abstracts different hardware backends (CPU, GPU,
// TPU, NPU) behind a common interface, allowing models to run on the
// best available hardware.
package accel

import (
	"context"
)

// Accelerator provides hardware-accelerated tensor operations.
type Accelerator interface {
	// Name returns the accelerator name.
	Name() string

	// Device returns the device identifier.
	Device() string

	// Load loads a model from the given path.
	Load(ctx context.Context, modelPath string) (Model, error)

	// Close releases accelerator resources.
	Close() error
}

// Model represents a loaded ML model.
type Model interface {
	// Name returns the model name.
	Name() string

	// Metadata returns model metadata.
	Metadata() Metadata

	// Infer runs inference with the given inputs.
	Infer(ctx context.Context, inputs map[string]Tensor) (map[string]Tensor, error)

	// Close releases model resources.
	Close() error
}

// Metadata describes a model.
type Metadata struct {
	Name      string
	Version   string
	Framework string
	Inputs    []TensorSpec
	Outputs   []TensorSpec
}

// TensorSpec describes a tensor.
type TensorSpec struct {
	Name     string
	Shape    []int
	DataType DataType
}

// DataType represents tensor element types.
type DataType int

const (
	Float32 DataType = iota
	Float64
	Int32
	Int64
	Uint8
	Int8
)

// Tensor represents a multi-dimensional array.
type Tensor struct {
	Shape    []int
	DataType DataType
	Data     any // []float32, []float64, []int32, etc.
}

// NewTensor creates a new tensor with the given shape and data.
func NewTensor[T any](shape []int, data []T) Tensor {
	var dt DataType
	var d any = data

	switch any(data).(type) {
	case []float32:
		dt = Float32
	case []float64:
		dt = Float64
	case []int32:
		dt = Int32
	case []int64:
		dt = Int64
	case []uint8:
		dt = Uint8
	case []int8:
		dt = Int8
	}

	return Tensor{
		Shape:    shape,
		DataType: dt,
		Data:     d,
	}
}
