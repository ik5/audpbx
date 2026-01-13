// SPDX-License-Identifier: EPL-2.0

package utils

import (
	"math"
	"testing"
)

func TestFloat32ToInt16(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input float32
		want  int16
	}{
		{
			name:  "zero",
			input: 0.0,
			want:  0,
		},
		{
			name:  "max positive",
			input: 1.0,
			want:  math.MaxInt16,
		},
		{
			name:  "max negative",
			input: -1.0,
			want:  math.MinInt16,
		},
		{
			name:  "half positive",
			input: 0.5,
			want:  16383, // math.MaxInt16 * 0.5 ≈ 16383.5
		},
		{
			name:  "half negative",
			input: -0.5,
			want:  -16383,
		},
		{
			name:  "quarter positive",
			input: 0.25,
			want:  8191, // math.MaxInt16 * 0.25 ≈ 8191.75
		},
		{
			name:  "small positive",
			input: 0.001,
			want:  32, // math.MaxInt16 * 0.001 ≈ 32.767
		},
		{
			name:  "small negative",
			input: -0.001,
			want:  -32,
		},
		{
			name:  "clamp over max",
			input: 1.5,
			want:  math.MaxInt16, // Should clamp to 1.0
		},
		{
			name:  "clamp over min",
			input: -1.5,
			want:  math.MinInt16, // Should clamp to -1.0
		},
		{
			name:  "clamp way over max",
			input: 100.0,
			want:  math.MaxInt16,
		},
		{
			name:  "clamp way under min",
			input: -100.0,
			want:  math.MinInt16,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := Float32ToInt16(tt.input)
			// Allow for rounding differences of ±1
			diff := int16(math.Abs(float64(got - tt.want)))

			if diff > 1 {
				t.Errorf("Float32ToInt16(%v) = %v, want %v (diff %v)",
					tt.input, got, tt.want, diff)
			}
		})
	}
}

// TestFloat32ToInt16Range tests full range conversion
func TestFloat32ToInt16Range(t *testing.T) {
	t.Parallel()

	var result int32

	// Test that values in [-1, 1] produce valid int16 values
	for f := -1.0; f <= 1.0; f += 0.01 {
		result = int32(Float32ToInt16(float32(f)))

		// Result should be in valid int16 range (note: math.MinInt16 is valid for int16)
		if result < math.MinInt16 || result > math.MaxInt16 {
			t.Errorf("Float32ToInt16(%v) = %v, outside valid range [-32768, 32767]",
				f, result)
		}

		// Result should be proportional to input (using 32768 as multiplier)
		expected := int32(f * 32768.0)
		diff := int16(math.Abs(float64(result - expected)))

		if diff > 1 {
			t.Errorf("Float32ToInt16(%v) = %v, want ≈%v (diff %v)",
				f, result, expected, diff)
		}
	}
}

// TestFloat32ToInt16Symmetry tests that conversion is symmetric
func TestFloat32ToInt16Symmetry(t *testing.T) {
	t.Parallel()

	testVals := []float32{0.1, 0.25, 0.5, 0.75, 0.9, 0.99, 1.0}

	for _, val := range testVals {
		pos := Float32ToInt16(val)
		neg := Float32ToInt16(-val)

		// Absolute values should be equal (within rounding)
		if math.Abs(float64(pos+neg)) > 1 {
			t.Errorf("Float32ToInt16 not symmetric: +%v=%v, -%v=%v",
				val, pos, val, neg)
		}
	}
}

// TestFloat32ToInt16Monotonic tests that function is monotonic
func TestFloat32ToInt16Monotonic(t *testing.T) {
	t.Parallel()

	prev := Float32ToInt16(-1.0)

	for f := -0.99; f <= 1.0; f += 0.01 {
		curr := Float32ToInt16(float32(f))
		if curr < prev {
			t.Errorf("Float32ToInt16 not monotonic: f=%v gives %v, but previous was %v",
				f, curr, prev)
		}
		prev = curr
	}
}

// BenchmarkFloat32ToInt16 tests performance and allocations
func BenchmarkFloat32ToInt16(b *testing.B) {
	var result int16
	input := float32(0.5)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		result = Float32ToInt16(input)
	}

	// Prevent compiler optimization
	_ = result
}

// BenchmarkFloat32ToInt16Realistic simulates converting audio buffer
func BenchmarkFloat32ToInt16Realistic(b *testing.B) {
	// Simulate converting 1 second of mono audio at 8kHz
	floatSamples := make([]float32, 8000)
	int16Samples := make([]int16, 8000)

	// Fill with realistic audio data
	for i := range floatSamples {
		floatSamples[i] = float32(math.Sin(float64(i) * 0.1))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		for j := range floatSamples {
			int16Samples[j] = Float32ToInt16(floatSamples[j])
		}
	}
}

// BenchmarkFloat32ToInt16WithClamping tests performance with out-of-range values
func BenchmarkFloat32ToInt16WithClamping(b *testing.B) {
	var result int16
	inputs := []float32{-2.0, -1.0, 0.0, 1.0, 2.0}

	b.ResetTimer()
	b.ReportAllocs()

	for i := range b.N {
		result = Float32ToInt16(inputs[i%len(inputs)])
	}

	_ = result
}

// TestFloat32ToInt16_ZeroAllocs verifies no heap allocations
func TestFloat32ToInt16_ZeroAllocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping allocation test in short mode")
	}

	allocs := testing.AllocsPerRun(1000, func() {
		_ = Float32ToInt16(0.5)
	})

	if allocs > 0 {
		t.Errorf("Float32ToInt16 allocated %v times, want 0", allocs)
	}
}

// TestFloat32ToInt16_BatchZeroAllocs tests batch conversion allocations
func TestFloat32ToInt16_BatchZeroAllocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping allocation test in short mode")
	}

	floatBuf := make([]float32, 1024)
	int16Buf := make([]int16, 1024)

	allocs := testing.AllocsPerRun(100, func() {
		for i := range floatBuf {
			int16Buf[i] = Float32ToInt16(floatBuf[i])
		}
	})

	if allocs > 0 {
		t.Errorf("Float32ToInt16 batch conversion allocated %v times, want 0", allocs)
	}
}
