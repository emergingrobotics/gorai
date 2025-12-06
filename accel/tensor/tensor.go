// Package tensor provides tensor utilities for ML inference.
package tensor

import (
	"fmt"

	"github.com/gorai/gorai/accel"
)

// Size returns the total number of elements in a tensor.
func Size(shape []int) int {
	if len(shape) == 0 {
		return 0
	}
	size := 1
	for _, dim := range shape {
		size *= dim
	}
	return size
}

// Reshape returns a new tensor with the given shape.
// The total number of elements must match.
func Reshape(t accel.Tensor, newShape []int) (accel.Tensor, error) {
	oldSize := Size(t.Shape)
	newSize := Size(newShape)

	if oldSize != newSize {
		return accel.Tensor{}, fmt.Errorf("cannot reshape tensor of size %d to size %d", oldSize, newSize)
	}

	return accel.Tensor{
		Shape:    newShape,
		DataType: t.DataType,
		Data:     t.Data,
	}, nil
}

// Zeros creates a tensor filled with zeros.
func Zeros(shape []int, dtype accel.DataType) accel.Tensor {
	size := Size(shape)

	var data any
	switch dtype {
	case accel.Float32:
		data = make([]float32, size)
	case accel.Float64:
		data = make([]float64, size)
	case accel.Int32:
		data = make([]int32, size)
	case accel.Int64:
		data = make([]int64, size)
	case accel.Uint8:
		data = make([]uint8, size)
	case accel.Int8:
		data = make([]int8, size)
	}

	return accel.Tensor{
		Shape:    shape,
		DataType: dtype,
		Data:     data,
	}
}

// Ones creates a tensor filled with ones.
func Ones(shape []int, dtype accel.DataType) accel.Tensor {
	size := Size(shape)

	var data any
	switch dtype {
	case accel.Float32:
		d := make([]float32, size)
		for i := range d {
			d[i] = 1
		}
		data = d
	case accel.Float64:
		d := make([]float64, size)
		for i := range d {
			d[i] = 1
		}
		data = d
	case accel.Int32:
		d := make([]int32, size)
		for i := range d {
			d[i] = 1
		}
		data = d
	case accel.Int64:
		d := make([]int64, size)
		for i := range d {
			d[i] = 1
		}
		data = d
	case accel.Uint8:
		d := make([]uint8, size)
		for i := range d {
			d[i] = 1
		}
		data = d
	case accel.Int8:
		d := make([]int8, size)
		for i := range d {
			d[i] = 1
		}
		data = d
	}

	return accel.Tensor{
		Shape:    shape,
		DataType: dtype,
		Data:     data,
	}
}
