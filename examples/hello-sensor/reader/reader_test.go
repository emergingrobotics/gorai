package reader_test

import (
	"testing"

	"github.com/gorai/gorai/examples/hello-sensor/reader"
)

func TestCelsiusToFahrenheit(t *testing.T) {
	tests := []struct {
		celsius    float64
		fahrenheit float64
	}{
		{0, 32},
		{100, 212},
		{-40, -40},
		{37, 98.6},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			got := reader.CelsiusToFahrenheit(tt.celsius)
			// Use approximate comparison for floating point
			if diff := got - tt.fahrenheit; diff > 0.1 || diff < -0.1 {
				t.Errorf("CelsiusToFahrenheit(%v) = %v, want %v", tt.celsius, got, tt.fahrenheit)
			}
		})
	}
}
