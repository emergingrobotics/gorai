// Package mlmodel defines the ML model inference service interface.
package mlmodel

import (
	"context"

	"github.com/emergingrobotics/gorai/services"
)

// Service provides ML model inference capabilities.
type Service interface {
	service.Service

	// Infer runs inference with the given input tensors.
	Infer(ctx context.Context, inputs map[string]Tensor) (map[string]Tensor, error)

	// Metadata returns model metadata.
	Metadata(ctx context.Context) (Metadata, error)
}

// Tensor represents a multi-dimensional array.
type Tensor struct {
	// Name of the tensor.
	Name string
	// Shape of the tensor.
	Shape []int
	// DataType of elements.
	DataType DataType
	// Data contains the flattened tensor data.
	Data any // []float32, []float64, []int32, []int64, []uint8, etc.
}

// DataType represents tensor element types.
type DataType int

const (
	DataTypeUnknown DataType = iota
	DataTypeFloat32
	DataTypeFloat64
	DataTypeInt32
	DataTypeInt64
	DataTypeUint8
	DataTypeInt8
	DataTypeBool
)

// Metadata describes an ML model.
type Metadata struct {
	// Name of the model.
	Name string
	// Version of the model.
	Version string
	// Framework used (onnx, tflite, rknn, etc.).
	Framework string
	// Inputs describes expected input tensors.
	Inputs []TensorSpec
	// Outputs describes output tensors.
	Outputs []TensorSpec
}

// TensorSpec describes a tensor's expected shape and type.
type TensorSpec struct {
	// Name of the tensor.
	Name string
	// Shape of the tensor (-1 for dynamic dimensions).
	Shape []int
	// DataType of elements.
	DataType DataType
	// Description of the tensor's purpose.
	Description string
}
