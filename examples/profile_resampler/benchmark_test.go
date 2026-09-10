// SPDX-License-Identifier: EPL-2.0

// These benchmarks measure individual components against an in-memory source.
//
// IMPORTANT: they cannot detect the pipeline's most likely performance problem.
// A mock source that copies from a slice performs no syscalls and allocates
// nothing per call, so it hides exactly the costs that dominate real runs:
// per-call decoder overhead and read(2) traffic. These benchmarks reported
// healthy numbers while the real pipeline ran ~40x slower than ffmpeg, because
// the resampler was pulling one frame at a time from the decoder and the
// decoder was issuing one syscall per 4 bytes.
//
// Use these to compare component-level changes. To measure the pipeline, use
// the real-file benchmarks in internal/perfbench, which decode actual files:
//
//	go test ./internal/perfbench/ -bench . -benchtime 3x
//
// See PROFILING_GUIDE.md for the full case study.
package main

import (
	"io"
	"testing"

	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/utils"
)

// mockSource is an in-memory audio.Source. See the package comment above for
// why benchmarks built on it cannot see I/O or decoder call overhead.
type mockSource struct {
	rate     int
	channels int
	samples  []float32
	pos      int
}

func (m *mockSource) SampleRate() int { return m.rate }
func (m *mockSource) Channels() int   { return m.channels }
func (m *mockSource) BufSize() int    { return 4096 }
func (m *mockSource) Close() error    { return nil }

func (m *mockSource) ReadSamples(dst []float32) (int, error) {
	if m.pos >= len(m.samples) {
		return 0, io.EOF
	}
	n := copy(dst, m.samples[m.pos:])
	m.pos += n
	if m.pos >= len(m.samples) {
		return n, io.EOF
	}
	return n, nil
}

func newMockSource(rate, channels, numFrames int) *mockSource {
	samples := make([]float32, numFrames*channels)
	// Fill with sine wave
	for i := range samples {
		samples[i] = 0.5
	}
	return &mockSource{
		rate:     rate,
		channels: channels,
		samples:  samples,
	}
}

// Benchmark cubic interpolation
func BenchmarkCubicInterpolate(b *testing.B) {
	y0, y1, y2, y3 := float32(0.1), float32(0.5), float32(0.8), float32(0.3)
	x := float32(0.5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = utils.CubicInterpolate(y0, y1, y2, y3, x)
	}
}

// Benchmark resampler with different scenarios
func BenchmarkResampler_Downsample_Stereo(b *testing.B) {
	// 48kHz stereo -> 8kHz (typical voice processing)
	src := newMockSource(48000, 2, 48000) // 1 second of audio
	buf := make([]float32, 4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src.pos = 0 // Reset
		resampler := audio.NewResampler(src, 8000)
		for {
			_, err := resampler.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}

func BenchmarkResampler_Upsample_Mono(b *testing.B) {
	// 8kHz mono -> 48kHz
	src := newMockSource(8000, 1, 8000) // 1 second of audio
	buf := make([]float32, 4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src.pos = 0
		resampler := audio.NewResampler(src, 48000)
		for {
			_, err := resampler.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}

func BenchmarkResampler_CDQuality(b *testing.B) {
	// 44.1kHz stereo -> 48kHz (CD to pro audio)
	src := newMockSource(44100, 2, 44100)
	buf := make([]float32, 4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src.pos = 0
		resampler := audio.NewResampler(src, 48000)
		for {
			_, err := resampler.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}

// Benchmark mono mixer
func BenchmarkMonoMixer_Stereo(b *testing.B) {
	src := newMockSource(48000, 2, 48000)
	buf := make([]float32, 4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src.pos = 0
		mixer := audio.NewMonoMixer(src)
		for {
			_, err := mixer.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}

func BenchmarkMonoMixer_Quad(b *testing.B) {
	src := newMockSource(48000, 4, 48000)
	buf := make([]float32, 4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src.pos = 0
		mixer := audio.NewMonoMixer(src)
		for {
			_, err := mixer.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}

// Benchmark full pipeline
func BenchmarkFullPipeline_48kStereoTo8kMono(b *testing.B) {
	src := newMockSource(48000, 2, 48000*10) // 10 seconds
	buf := make([]float32, 4096)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src.pos = 0
		resampler := audio.NewResampler(src, 8000)
		mixer := audio.NewMonoMixer(resampler)

		pcm16 := make([]int16, 0, 8000*10)
		const maxInt16 float32 = 32768.0

		for {
			n, err := mixer.ReadSamples(buf)
			if n > 0 {
				for j := 0; j < n; j++ {
					x := buf[j]
					if x > 1 {
						x = 1
					} else if x < -1 {
						x = -1
					}
					pcm16 = append(pcm16, int16(x*maxInt16))
				}
			}
			if err == io.EOF {
				break
			}
		}
	}
}

// Benchmark with different buffer sizes
func BenchmarkBufferSize_1024(b *testing.B)  { benchmarkBufferSize(b, 1024) }
func BenchmarkBufferSize_2048(b *testing.B)  { benchmarkBufferSize(b, 2048) }
func BenchmarkBufferSize_4096(b *testing.B)  { benchmarkBufferSize(b, 4096) }
func BenchmarkBufferSize_8192(b *testing.B)  { benchmarkBufferSize(b, 8192) }
func BenchmarkBufferSize_16384(b *testing.B) { benchmarkBufferSize(b, 16384) }

func benchmarkBufferSize(b *testing.B, bufSize int) {
	src := newMockSource(48000, 2, 48000*2) // 2 seconds
	buf := make([]float32, bufSize)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		src.pos = 0
		resampler := audio.NewResampler(src, 8000)
		mixer := audio.NewMonoMixer(resampler)

		for {
			_, err := mixer.ReadSamples(buf)
			if err == io.EOF {
				break
			}
		}
	}
}
