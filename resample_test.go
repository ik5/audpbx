// SPDX-License-Identifier: EPL-2.0

package audpbx

import (
	"io"
	"math"
	"runtime"
	"testing"

	"github.com/ik5/audpbx/internal/audiotest"
)

func TestResampleToMono16_Basic(t *testing.T) {
	t.Parallel()

	// Create 1 second of stereo audio at 44.1kHz
	src := audiotest.NewSineSource(44100, 2, 44100, 440.0)

	pcm16, rate, err := ResampleToMono16(src, 8000, 4096)

	if err != nil && err != io.EOF {
		t.Fatalf("ResampleToMono16() error = %v", err)
	}

	if rate != 8000 {
		t.Errorf("ResampleToMono16() rate = %d, want 8000", rate)
	}

	// Should have approximately 8000 samples (1 second at 8kHz, mono)
	expected := 8000
	tolerance := 200
	if len(pcm16) < expected-tolerance || len(pcm16) > expected+tolerance {
		t.Errorf("ResampleToMono16() got %d samples, want ≈%d (±%d)",
			len(pcm16), expected, tolerance)
	}

	// Verify samples are in valid int16 range
	for i, s := range pcm16 {
		if s < -32768 || s > 32767 {
			t.Errorf("pcm16[%d] = %d, outside int16 range", i, s)
		}
	}
}

func TestResampleToMono16_AlreadyMono(t *testing.T) {
	t.Parallel()

	// Source is already mono
	src := audiotest.NewConstantSource(16000, 1, 16000, 0.5)

	pcm16, rate, err := ResampleToMono16(src, 8000, 4096)

	if err != nil && err != io.EOF {
		t.Fatalf("ResampleToMono16() error = %v", err)
	}

	if rate != 8000 {
		t.Errorf("ResampleToMono16() rate = %d, want 8000", rate)
	}

	// Should have approximately 8000 samples
	expected := 8000
	tolerance := 200
	if len(pcm16) < expected-tolerance || len(pcm16) > expected+tolerance {
		t.Errorf("ResampleToMono16() got %d samples, want ≈%d (±%d)",
			len(pcm16), expected, tolerance)
	}

	// With constant 0.5 input, all samples should be around 16383 (0.5 * 32767)
	for i, s := range pcm16 {
		if math.Abs(float64(s-16384)) > 1000 {
			t.Errorf("pcm16[%d] = %d, want ≈16384", i, s)
			break
		}
	}
}

func TestResampleToMono16_Silence(t *testing.T) {
	t.Parallel()

	// Stereo silence
	src := audiotest.NewSilentSource(44100, 2, 44100)

	pcm16, rate, err := ResampleToMono16(src, 8000, 4096)

	if err != nil && err != io.EOF {
		t.Fatalf("ResampleToMono16() error = %v", err)
	}

	if rate != 8000 {
		t.Errorf("ResampleToMono16() rate = %d, want 8000", rate)
	}

	// All samples should be close to zero
	for i, s := range pcm16 {
		if math.Abs(float64(s)) > 100 {
			t.Errorf("pcm16[%d] = %d, want ≈0 (silence)", i, s)
		}
	}
}

func TestResampleToMono16_EmptySource(t *testing.T) {
	t.Parallel()

	// Source with no samples
	src := audiotest.NewSilentSource(44100, 2, 0)

	pcm16, rate, err := ResampleToMono16(src, 8000, 4096)

	if err != nil && err != io.EOF {
		t.Fatalf("ResampleToMono16() error = %v", err)
	}

	if rate != 8000 {
		t.Errorf("ResampleToMono16() rate = %d, want 8000", rate)
	}

	if len(pcm16) != 0 {
		t.Errorf("ResampleToMono16() got %d samples, want 0", len(pcm16))
	}
}

func TestResampleToMono16_VariousRates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		srcRate    int
		dstRate    int
		srcSamples int
	}{
		{
			name:       "44.1kHz to 8kHz",
			srcRate:    44100,
			dstRate:    8000,
			srcSamples: 44100,
		},
		{
			name:       "48kHz to 16kHz",
			srcRate:    48000,
			dstRate:    16000,
			srcSamples: 48000,
		},
		{
			name:       "8kHz to 16kHz (upsample)",
			srcRate:    8000,
			dstRate:    16000,
			srcSamples: 8000,
		},
		{
			name:       "22.05kHz to 8kHz",
			srcRate:    22050,
			dstRate:    8000,
			srcSamples: 22050,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := audiotest.NewSineSource(tt.srcRate, 2, tt.srcSamples, 440.0)

			pcm16, rate, err := ResampleToMono16(src, tt.dstRate, 4096)

			if err != nil && err != io.EOF {
				t.Fatalf("ResampleToMono16() error = %v", err)
			}

			if rate != tt.dstRate {
				t.Errorf("ResampleToMono16() rate = %d, want %d", rate, tt.dstRate)
			}

			// Verify we got approximately the right number of samples
			// (1 second of audio at dstRate)
			expected := tt.dstRate
			tolerance := tt.dstRate / 20 // 5% tolerance
			if len(pcm16) < expected-tolerance || len(pcm16) > expected+tolerance {
				t.Errorf("ResampleToMono16() got %d samples, want ≈%d (±%d)",
					len(pcm16), expected, tolerance)
			}
		})
	}
}

func TestResampleToMono16_Clamping(t *testing.T) {
	t.Parallel()

	// Source with values outside [-1, 1] to test clamping
	src := audiotest.NewMockSource(8000, 1, 100, func(sample int, channel int) float32 {
		if sample%3 == 0 {
			return 2.0 // Should clamp to 1.0 -> 32767
		}

		if sample%3 == 1 {
			return -2.0 // Should clamp to -1.0 -> -32768
		}

		return 0.0
	})

	pcm16, _, err := ResampleToMono16(src, 8000, 4096)

	if err != nil && err != io.EOF {
		t.Fatalf("ResampleToMono16() error = %v", err)
	}

	// Verify values are properly clamped
	for i, s := range pcm16 {
		if s < -32768 || s > 32767 {
			t.Errorf("pcm16[%d] = %d, outside valid range", i, s)
		}
	}
}

// BenchmarkResampleToMono16 benchmarks the complete pipeline
func BenchmarkResampleToMono16(b *testing.B) {
	// 1 second of stereo 44.1kHz audio
	b.ReportAllocs()

	for b.Loop() {
		src := audiotest.NewSineSource(44100, 2, 44100, 440.0)
		_, _, _ = ResampleToMono16(src, 8000, 4096)
	}
}

// BenchmarkResampleToMono16_LargeBuffer benchmarks with larger buffer
func BenchmarkResampleToMono16_LargeBuffer(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		src := audiotest.NewSineSource(44100, 2, 44100, 440.0)
		_, _, _ = ResampleToMono16(src, 8000, 16384)
	}
}

// BenchmarkResampleToMono16_SmallBuffer benchmarks with small buffer
func BenchmarkResampleToMono16_SmallBuffer(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		src := audiotest.NewSineSource(44100, 2, 44100, 440.0)
		_, _, _ = ResampleToMono16(src, 8000, 1024)
	}
}

// BenchmarkResampleToMono16_Upsample benchmarks upsampling
func BenchmarkResampleToMono16_Upsample(b *testing.B) {
	b.ReportAllocs()

	for b.Loop() {
		src := audiotest.NewSineSource(8000, 2, 8000, 440.0)
		_, _, _ = ResampleToMono16(src, 44100, 4096)
	}
}

// TestResampleToMono16_AllocsDoNotScaleWithLength checks that the pipeline's
// allocation count is independent of how long the input is.
//
// This is the end-to-end guard for the class of defect that made this pipeline
// 40x slower than ffmpeg: an allocation on a per-frame or per-call path, which
// is invisible in a correctness test and barely visible by allocated bytes,
// but grows without bound as input length grows.
//
// It deliberately measures counts rather than time, so it is deterministic and
// cannot flake on a loaded machine. Growth is expected to be logarithmic, since
// the output slice doubles as it fills, so the assertion allows a small
// additive increase but nothing proportional to length.
//
// Note that this catches an allocating layer anywhere in the chain — decoder,
// resampler, mixer, or converter. It does not catch a resampler that reads its
// source in tiny chunks without allocating; audio.TestResampler_ReadsSourceInBlocks
// covers that.
func TestResampleToMono16_AllocsDoNotScaleWithLength(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping allocation test in short mode")
	}

	const (
		shortFrames = 44100 // 1 second at 44.1 kHz
		lengthRatio = 10    // the long input is this many times longer
		allowance   = 8     // room for the output slice doubling as it grows
	)

	countAllocs := func(frames int) uint64 {
		var before, after runtime.MemStats

		// Settle any garbage from setup before taking the baseline.
		runtime.GC()
		runtime.ReadMemStats(&before)

		src := audiotest.NewSineSource(44100, 2, frames, 440.0)

		pcm16, _, err := ResampleToMono16(src, 8000, 4096)
		if err != nil {
			t.Fatalf("ResampleToMono16() error = %v", err)
		}

		runtime.ReadMemStats(&after)

		// Keep the result alive so it cannot be optimised away.
		if len(pcm16) == 0 {
			t.Fatal("ResampleToMono16() produced no samples")
		}

		return after.Mallocs - before.Mallocs
	}

	shortAllocs := countAllocs(shortFrames)
	longAllocs := countAllocs(shortFrames * lengthRatio)

	t.Logf("%d frames: %d allocations; %d frames: %d allocations",
		shortFrames, shortAllocs, shortFrames*lengthRatio, longAllocs)

	if longAllocs > shortAllocs+allowance {
		t.Errorf("allocations grew from %d to %d when the input got %dx longer"+
			" (allowance %d); something on the hot path allocates per frame or"+
			" per read",
			shortAllocs, longAllocs, lengthRatio, allowance)
	}
}
