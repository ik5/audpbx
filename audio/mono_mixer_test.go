// SPDX-License-Identifier: EPL-2.0

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
		src.Reset() // Reset for next iteration
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
		src.Reset() // Reset for next iteration
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

	for b.Loop() {
		src.Reset()
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
		src.Reset()
		_, _ = mixer.ReadSamples(buf)
	})

	if allocs > 0 {
		b.Errorf("MonoMixer.ReadSamples() allocated %v times, want 0", allocs)
	}
}

func TestMonoMixer_LargeBuffer(t *testing.T) {
	t.Parallel()

	src := newSineSource(8000, 2, 8000, 440.0)
	mixer := NewMonoMixer(src)

	// Very large buffer
	buf := make([]float32, 16384)
	n, err := mixer.ReadSamples(buf)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n < 0 || n > len(buf) {
		t.Errorf("ReadSamples() n = %d, should be in range [0, %d]", n, len(buf))
	}
}

func TestMonoMixer_MultipleChannels(t *testing.T) {
	t.Parallel()

	// 8-channel source with different values per channel
	src := newMockSource(8000, 8, 100, func(sample int, channel int) float32 {
		return float32(channel) * 0.1 // 0.0, 0.1, 0.2, ..., 0.7
	})

	mixer := NewMonoMixer(src)

	if mixer.Channels() != 1 {
		t.Fatalf("MonoMixer.Channels() = %d, want 1", mixer.Channels())
	}

	buf := make([]float32, 10)
	n, err := mixer.ReadSamples(buf)

	if n == 0 {
		t.Fatal("ReadSamples() returned 0 samples")
	}

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	// Average should be (0.0 + 0.1 + 0.2 + 0.3 + 0.4 + 0.5 + 0.6 + 0.7) / 8 = 2.8 / 8 = 0.35
	expected := float32(0.35)
	for i := range n {
		if math.Abs(float64(buf[i]-expected)) > 0.01 {
			t.Errorf("buf[%d] = %v, want ≈%v", i, buf[i], expected)
		}
	}
}

func TestMonoMixer_Close(t *testing.T) {
	t.Parallel()

	src := newSilentSource(8000, 2, 1000)
	mixer := NewMonoMixer(src)

	err := mixer.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestMonoMixer_PartialRead(t *testing.T) {
	t.Parallel()

	// Source with exactly 50 samples
	src := newSilentSource(8000, 2, 50)
	mixer := NewMonoMixer(src)

	// Request more than available
	buf := make([]float32, 100)
	n, err := mixer.ReadSamples(buf)

	if err != io.EOF {
		t.Errorf("ReadSamples() error = %v, want io.EOF", err)
	}

	if n != 50 {
		t.Errorf("ReadSamples() n = %d, want 50", n)
	}
}

func TestMonoMixer_SmallReads(t *testing.T) {
	t.Parallel()

	src := newConstantSource(8000, 2, 1000, 0.5)
	mixer := NewMonoMixer(src)

	// Multiple small reads
	for range 10 {
		buf := make([]float32, 5)
		n, err := mixer.ReadSamples(buf)

		if err != nil && err != io.EOF {
			t.Fatalf("ReadSamples() error = %v", err)
		}

		if n > 0 {
			// Values should be 0.5
			for i := range n {
				if math.Abs(float64(buf[i]-0.5)) > 0.01 {
					t.Errorf("buf[%d] = %v, want ≈0.5", i, buf[i])
				}
			}
		}

		if err == io.EOF {
			break
		}
	}
}

// BenchmarkMonoMixer_ManyChannels benchmarks mixing many channels
func BenchmarkMonoMixer_ManyChannels(b *testing.B) {
	src := newMockSource(8000, 16, 100000, func(sample int, channel int) float32 {
		return 0.0625 // 16 channels
	})
	mixer := NewMonoMixer(src)
	buf := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		src.Reset()
		for {
			_, err := mixer.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}

// BenchmarkMonoMixer_SmallReads benchmarks many small reads
func BenchmarkMonoMixer_SmallReads(b *testing.B) {
	src := newSineSource(8000, 2, 100000, 440.0)
	mixer := NewMonoMixer(src)
	buf := make([]float32, 32)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		src.Reset()
		for {
			_, err := mixer.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}
