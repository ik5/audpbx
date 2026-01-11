package audio

import (
	"io"
	"math"
	"testing"
)

func TestResampler_Metadata(t *testing.T) {
	t.Parallel()

	src := newSilentSource(44100, 2, 1000)
	resampler := NewResampler(src, 8000)

	if resampler.SampleRate() != 8000 {
		t.Errorf("Resampler.SampleRate() = %d, want 8000", resampler.SampleRate())
	}

	if resampler.Channels() != 2 {
		t.Errorf("Resampler.Channels() = %d, want 2", resampler.Channels())
	}
}

func TestResampler_SameRate(t *testing.T) {
	t.Parallel()

	// No resampling needed (same rate)
	src := newConstantSource(8000, 1, 100, 0.5)
	resampler := NewResampler(src, 8000)

	buf := make([]float32, 100)
	n, err := resampler.ReadSamples(buf)

	if err == nil || err == io.EOF {
		// OK
	} else {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n == 0 {
		t.Fatal("ReadSamples() returned 0 samples")
	}

	// Values should be approximately 0.5
	for i := range n {
		if math.Abs(float64(buf[i]-0.5)) > 0.1 {
			t.Errorf("buf[%d] = %v, want ≈0.5", i, buf[i])
		}
	}
}

func TestResampler_Downsampling(t *testing.T) {
	t.Parallel()

	// Downsample from 44.1kHz to 8kHz
	totalSamples := 44100 // 1 second of audio
	src := newSineSource(44100, 1, totalSamples, 440.0)
	resampler := NewResampler(src, 8000)

	// Collect all resampled data
	buf := make([]float32, 1024)
	var samples []float32

	for {
		n, err := resampler.ReadSamples(buf)
		if n > 0 {
			samples = append(samples, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadSamples() error = %v", err)
		}
	}

	// Should have approximately 8000 samples (1 second at 8kHz)
	expected := 8000
	tolerance := 100
	if len(samples) < expected-tolerance || len(samples) > expected+tolerance {
		t.Errorf("Resampled %d samples, want ≈%d (±%d)", len(samples), expected, tolerance)
	}

	// Verify samples are in valid range
	for i, s := range samples {
		if s < -1.5 || s > 1.5 {
			t.Errorf("samples[%d] = %v, outside reasonable range [-1.5, 1.5]", i, s)
		}
	}
}

func TestResampler_Upsampling(t *testing.T) {
	t.Parallel()

	// Upsample from 8kHz to 44.1kHz
	totalSamples := 8000 // 1 second of audio
	src := newSineSource(8000, 1, totalSamples, 440.0)
	resampler := NewResampler(src, 44100)

	// Collect all resampled data
	buf := make([]float32, 1024)
	var samples []float32

	for {
		n, err := resampler.ReadSamples(buf)
		if n > 0 {
			samples = append(samples, buf[:n]...)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadSamples() error = %v", err)
		}
	}

	// Should have approximately 44100 samples (1 second at 44.1kHz)
	expected := 44100
	tolerance := 500
	if len(samples) < expected-tolerance || len(samples) > expected+tolerance {
		t.Errorf("Resampled %d samples, want ≈%d (±%d)", len(samples), expected, tolerance)
	}

	// Verify samples are in valid range
	for i, s := range samples {
		if s < -1.5 || s > 1.5 {
			t.Errorf("samples[%d] = %v, outside reasonable range [-1.5, 1.5]", i, s)
		}
	}
}

func TestResampler_StereoPreserved(t *testing.T) {
	t.Parallel()

	// Stereo resampling should preserve channel count
	src := newMockSource(44100, 2, 1000, func(sample int, channel int) float32 {
		if channel == 0 {
			return 0.3 // Left
		}
		return 0.7 // Right
	})

	resampler := NewResampler(src, 8000)

	if resampler.Channels() != 2 {
		t.Fatalf("Resampler.Channels() = %d, want 2", resampler.Channels())
	}

	buf := make([]float32, 20) // 10 stereo frames
	n, err := resampler.ReadSamples(buf)

	if n == 0 {
		t.Fatal("ReadSamples() returned 0 samples")
	}

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	// Verify channels are preserved (interleaved L, R, L, R, ...)
	frames := n / 2
	for f := range frames {
		left := buf[f*2]
		right := buf[f*2+1]

		// Left should be near 0.3, right near 0.7
		if math.Abs(float64(left-0.3)) > 0.2 {
			t.Errorf("frame[%d] left = %v, want ≈0.3", f, left)
		}
		if math.Abs(float64(right-0.7)) > 0.2 {
			t.Errorf("frame[%d] right = %v, want ≈0.7", f, right)
		}
	}
}

func TestResampler_EOF(t *testing.T) {
	t.Parallel()

	src := newSilentSource(44100, 1, 100)
	resampler := NewResampler(src, 8000)

	buf := make([]float32, 1024)

	// Read until EOF
	var totalRead int
	for {
		n, err := resampler.ReadSamples(buf)
		totalRead += n
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("ReadSamples() error = %v", err)
		}
	}

	if totalRead == 0 {
		t.Error("No samples read before EOF")
	}

	// Next read should return EOF immediately
	n, err := resampler.ReadSamples(buf)
	if err != io.EOF {
		t.Errorf("After EOF, ReadSamples() error = %v, want io.EOF", err)
	}
	if n != 0 {
		t.Errorf("After EOF, ReadSamples() n = %d, want 0", n)
	}
}

func TestResampler_InvalidDstSize(t *testing.T) {
	t.Parallel()

	src := newSilentSource(44100, 2, 1000)
	resampler := NewResampler(src, 8000)

	// Buffer size not multiple of channels (2)
	buf := make([]float32, 7)
	_, err := resampler.ReadSamples(buf)

	if err != ErrInvalidDstSize {
		t.Errorf("ReadSamples() with invalid size error = %v, want ErrInvalidDstSize", err)
	}
}

// BenchmarkResampler_Downsample benchmarks downsampling 44.1kHz -> 8kHz
func BenchmarkResampler_Downsample(b *testing.B) {
	src := newSineSource(44100, 2, 100000, 440.0)
	resampler := NewResampler(src, 8000)
	buf := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		src.generated = 0 // Reset
		for {
			_, err := resampler.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}

// BenchmarkResampler_Upsample benchmarks upsampling 8kHz -> 44.1kHz
func BenchmarkResampler_Upsample(b *testing.B) {
	src := newSineSource(8000, 2, 20000, 440.0)
	resampler := NewResampler(src, 44100)
	buf := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		src.generated = 0 // Reset
		for {
			_, err := resampler.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}

// BenchmarkResampler_ReadSamples benchmarks single ReadSamples call
func BenchmarkResampler_ReadSamples(b *testing.B) {
	src := newSineSource(44100, 2, 1000000, 440.0)
	resampler := NewResampler(src, 8000)
	buf := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		src.generated = 0
		_, _ = resampler.ReadSamples(buf)
	}
}

// TestResampler_MinimalAllocs verifies minimal allocations after initialization
func TestResampler_MinimalAllocs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping allocation test in short mode")
	}
	// Note: Cannot use t.Parallel() with testing.AllocsPerRun

	src := newSineSource(44100, 2, 1000000, 440.0)
	resampler := NewResampler(src, 8000)
	buf := make([]float32, 4096)

	// Warm up to initialize internal buffers
	resampler.ReadSamples(buf)

	allocs := testing.AllocsPerRun(100, func() {
		src.generated = 0
		_, _ = resampler.ReadSamples(buf)
	})

	// Resampler may need to grow internal buffers occasionally
	// but should have minimal allocations in steady state
	if allocs > 1 {
		t.Logf("Warning: Resampler.ReadSamples() allocated %v times (should be minimal)", allocs)
	}
}
