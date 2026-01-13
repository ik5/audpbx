// SPDX-License-Identifier: EPL-2.0

package utils

import (
	"math"
	"testing"
)

func TestCubicInterpolate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		y0, y1, y2, y3 float32
		x              float32
		want           float32
		tolerance      float32
	}{
		{
			name:      "interpolate at start (x=0)",
			y0:        0.0,
			y1:        1.0,
			y2:        2.0,
			y3:        3.0,
			x:         0.0,
			want:      1.0, // Should return y1
			tolerance: 0.001,
		},
		{
			name:      "interpolate at end (x=1)",
			y0:        0.0,
			y1:        1.0,
			y2:        2.0,
			y3:        3.0,
			x:         1.0,
			want:      2.0, // Should return y2
			tolerance: 0.001,
		},
		{
			name:      "interpolate midpoint (x=0.5)",
			y0:        0.0,
			y1:        1.0,
			y2:        2.0,
			y3:        3.0,
			x:         0.5,
			want:      1.5, // Should be close to average
			tolerance: 0.1,
		},
		{
			name:      "linear data produces linear result",
			y0:        1.0,
			y1:        2.0,
			y2:        3.0,
			y3:        4.0,
			x:         0.25,
			want:      2.25,
			tolerance: 0.01,
		},
		{
			name:      "negative values",
			y0:        -1.0,
			y1:        -0.5,
			y2:        0.5,
			y3:        1.0,
			x:         0.5,
			want:      0.0,
			tolerance: 0.1,
		},
		{
			name:      "audio waveform peak",
			y0:        0.5,
			y1:        0.9,
			y2:        0.7,
			y3:        0.3,
			x:         0.3,
			want:      0.85,
			tolerance: 0.1,
		},
		{
			name:      "zero values",
			y0:        0.0,
			y1:        0.0,
			y2:        0.0,
			y3:        0.0,
			x:         0.5,
			want:      0.0,
			tolerance: 0.001,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := CubicInterpolate(tt.y0, tt.y1, tt.y2, tt.y3, tt.x)
			diff := float32(math.Abs(float64(got - tt.want)))

			if diff > tt.tolerance {
				t.Errorf("CubicInterpolate() = %v, want %v (tolerance %v, diff %v)",
					got, tt.want, tt.tolerance, diff)
			}
		})
	}
}

// TestCubicInterpolateBounds verifies behavior at boundaries
func TestCubicInterpolateBounds(t *testing.T) {
	t.Parallel()

	// Test that x=0 always returns y1
	for i := range 100 {
		y0, y1, y2, y3 := float32(i), float32(i+1), float32(i+2), float32(i+3)
		result := CubicInterpolate(y0, y1, y2, y3, 0.0)

		if result != y1 {
			t.Errorf("x=0 should return y1=%v, got %v", y1, result)
		}
	}

	// Test that x=1 always returns y2
	for i := range 100 {
		y0, y1, y2, y3 := float32(i), float32(i+1), float32(i+2), float32(i+3)
		result := CubicInterpolate(y0, y1, y2, y3, 1.0)
		if result != y2 {
			t.Errorf("x=1 should return y2=%v, got %v", y2, result)
		}
	}
}

// TestCubicInterpolateMonotonic tests that monotonic input produces reasonable output
func TestCubicInterpolateMonotonic(t *testing.T) {
	t.Parallel()

	// For monotonically increasing values, result should be between y1 and y2
	y0, y1, y2, y3 := float32(1.0), float32(2.0), float32(3.0), float32(4.0)

	for x := float32(0.0); x <= 1.0; x += 0.1 {
		result := CubicInterpolate(y0, y1, y2, y3, x)

		if result < y1-0.5 || result > y2+0.5 {
			t.Errorf("x=%v: result %v outside reasonable range [%v, %v]",
				x, result, y1-0.5, y2+0.5)
		}
	}
}

// BenchmarkCubicInterpolate tests performance and allocations
func BenchmarkCubicInterpolate(b *testing.B) {
	var result float32
	y0, y1, y2, y3 := float32(0.5), float32(1.0), float32(0.8), float32(0.3)
	x := float32(0.5)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		result = CubicInterpolate(y0, y1, y2, y3, x)
	}

	// Prevent compiler optimization
	_ = result
}

// BenchmarkCubicInterpolateRealistic simulates real usage in resampling
func BenchmarkCubicInterpolateRealistic(b *testing.B) {
	// Simulate processing 1 second of stereo audio at 44.1kHz -> 8kHz
	// That's approximately 8000 output samples
	samples := make([]float32, 8000)
	y0, y1, y2, y3 := float32(0.1), float32(0.5), float32(0.3), float32(-0.2)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		for j := range samples {
			// Vary x slightly to prevent constant folding
			x := float32(j%100) / 100.0
			samples[j] = CubicInterpolate(y0, y1, y2, y3, x)
		}
	}
}

// TestCubicInterpolate_ZeroAllocs verifies no heap allocations
func TestCubicInterpolate_ZeroAllocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping allocation test in short mode")
	}

	allocs := testing.AllocsPerRun(1000, func() {
		_ = CubicInterpolate(0.5, 1.0, 0.8, 0.3, 0.5)
	})

	if allocs > 0 {
		t.Errorf("CubicInterpolate allocated %v times, want 0", allocs)
	}
}
