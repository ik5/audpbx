package audio

import (
	"io"
	"math"
	"testing"
)

func TestMonoMixer_MonoPassthrough(t *testing.T) {
	t.Parallel()

	// Mono input should pass through unchanged
	src := newConstantSource(8000, 1, 100, 0.5)
	mixer := NewMonoMixer(src)

	if mixer.Channels() != 1 {
		t.Errorf("MonoMixer.Channels() = %d, want 1", mixer.Channels())
	}

	buf := make([]float32, 10)
	n, err := mixer.ReadSamples(buf)

	if err != nil {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != 10 {
		t.Errorf("ReadSamples() n = %d, want 10", n)
	}

	// All samples should be 0.5
	for i := range n {
		if buf[i] != 0.5 {
			t.Errorf("buf[%d] = %v, want 0.5", i, buf[i])
		}
	}
}

func TestMonoMixer_StereoToMono(t *testing.T) {
	t.Parallel()

	// Stereo source with different values per channel
	src := newMockSource(8000, 2, 100, func(sample int, channel int) float32 {
		if channel == 0 {
			return 0.4 // Left channel
		}
		return 0.6 // Right channel
	})

	mixer := NewMonoMixer(src)

	if mixer.Channels() != 1 {
		t.Errorf("MonoMixer.Channels() = %d, want 1", mixer.Channels())
	}

	buf := make([]float32, 10)
	n, err := mixer.ReadSamples(buf)

	if err != nil {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != 10 {
		t.Errorf("ReadSamples() n = %d, want 10", n)
	}

	// All samples should be average: (0.4 + 0.6) / 2 = 0.5
	expected := float32(0.5)
	for i := range n {
		if math.Abs(float64(buf[i]-expected)) > 0.001 {
			t.Errorf("buf[%d] = %v, want %v", i, buf[i], expected)
		}
	}
}

func TestMonoMixer_MultiChannel(t *testing.T) {
	t.Parallel()

	// 4-channel source
	src := newMockSource(8000, 4, 100, func(sample int, channel int) float32 {
		return float32(channel) / 10.0 // 0.0, 0.1, 0.2, 0.3
	})

	mixer := NewMonoMixer(src)

	buf := make([]float32, 10)
	n, err := mixer.ReadSamples(buf)

	if err != nil {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != 10 {
		t.Errorf("ReadSamples() n = %d, want 10", n)
	}

	// Average: (0.0 + 0.1 + 0.2 + 0.3) / 4 = 0.15
	expected := float32(0.15)
	for i := range n {
		diff := math.Abs(float64(buf[i] - expected))
		if diff > 0.001 {
			t.Errorf("buf[%d] = %v, want %v (diff %v)", i, buf[i], expected, diff)
		}
	}
}

func TestMonoMixer_EOF(t *testing.T) {
	t.Parallel()

	// Source with only 5 samples
	src := newSilentSource(8000, 2, 5)
	mixer := NewMonoMixer(src)

	buf := make([]float32, 10)
	n, err := mixer.ReadSamples(buf)

	if err != io.EOF {
		t.Errorf("ReadSamples() error = %v, want io.EOF", err)
	}

	if n != 5 {
		t.Errorf("ReadSamples() n = %d, want 5", n)
	}

	// Second read should return EOF immediately
	n, err = mixer.ReadSamples(buf)
	if err != io.EOF {
		t.Errorf("Second ReadSamples() error = %v, want io.EOF", err)
	}
	if n != 0 {
		t.Errorf("Second ReadSamples() n = %d, want 0", n)
	}
}

func TestMonoMixer_EmptyBuffer(t *testing.T) {
	t.Parallel()

	src := newSilentSource(8000, 2, 100)
	mixer := NewMonoMixer(src)

	buf := make([]float32, 0)
	n, err := mixer.ReadSamples(buf)

	if err != nil {
		t.Errorf("ReadSamples() with empty buffer error = %v, want nil", err)
	}

	if n != 0 {
		t.Errorf("ReadSamples() with empty buffer n = %d, want 0", n)
	}
}

func TestMonoMixer_PreservesMetadata(t *testing.T) {
	t.Parallel()

	src := newSilentSource(44100, 2, 100)
	mixer := NewMonoMixer(src)

	if mixer.SampleRate() != 44100 {
		t.Errorf("MonoMixer.SampleRate() = %d, want 44100", mixer.SampleRate())
	}

	if mixer.BufSize() != src.BufSize() {
		t.Errorf("MonoMixer.BufSize() = %d, want %d", mixer.BufSize(), src.BufSize())
	}
}

// BenchmarkMonoMixer_Passthrough benchmarks mono passthrough
func BenchmarkMonoMixer_Passthrough(b *testing.B) {
	src := newSilentSource(8000, 1, 100000)
	mixer := NewMonoMixer(src)
	buf := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		src.generated = 0 // Reset for next iteration
		for {
			_, err := mixer.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}

// BenchmarkMonoMixer_StereoToMono benchmarks stereo to mono conversion
func BenchmarkMonoMixer_StereoToMono(b *testing.B) {
	src := newSineSource(8000, 2, 100000, 440.0)
	mixer := NewMonoMixer(src)
	buf := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		src.generated = 0 // Reset for next iteration
		for {
			_, err := mixer.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}

// BenchmarkMonoMixer_ReadSamples benchmarks single ReadSamples call
func BenchmarkMonoMixer_ReadSamples(b *testing.B) {
	src := newSineSource(8000, 2, 1000000, 440.0)
	mixer := NewMonoMixer(src)
	buf := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		src.generated = 0
		_, _ = mixer.ReadSamples(buf)
	}
}

// BenchmarkMonoMixer_ZeroAllocs verifies no allocations after initialization
func BenchmarkMonoMixer_ZeroAllocs(b *testing.B) {
	if testing.Short() {
		b.Skip("skipping allocation test in short mode")
	}

	src := newSineSource(8000, 2, 100000, 440.0)
	mixer := NewMonoMixer(src)
	buf := make([]float32, 4096)

	// Warm up
	mixer.ReadSamples(buf)

	allocs := testing.AllocsPerRun(100, func() {
		src.generated = 0
		_, _ = mixer.ReadSamples(buf)
	})

	if allocs > 0 {
		b.Errorf("MonoMixer.ReadSamples() allocated %v times, want 0", allocs)
	}
}
