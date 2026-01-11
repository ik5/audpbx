package audio

import (
	"io"
	"math"
)

// mockSource is a test helper that generates audio data for testing.
// It implements the Source interface and can generate various waveforms.
type mockSource struct {
	sampleRate   int
	channels     int
	totalSamples int // Total samples to generate (per channel)
	generated    int // Samples generated so far (per channel)
	waveform     func(sample int, channel int) float32
}

// newMockSource creates a new mock audio source.
// totalSamples is the total number of samples per channel to generate.
// waveform is a function that generates sample values given sample index and channel.
func newMockSource(sampleRate, channels, totalSamples int, waveform func(sample int, channel int) float32) *mockSource {
	return &mockSource{
		sampleRate:   sampleRate,
		channels:     channels,
		totalSamples: totalSamples,
		generated:    0,
		waveform:     waveform,
	}
}

// newSilentSource creates a mock source that generates silence (all zeros).
func newSilentSource(sampleRate, channels, totalSamples int) *mockSource {
	return newMockSource(sampleRate, channels, totalSamples, func(sample int, channel int) float32 {
		return 0.0
	})
}

// newSineSource creates a mock source that generates a sine wave.
func newSineSource(sampleRate, channels, totalSamples int, frequency float64) *mockSource {
	return newMockSource(sampleRate, channels, totalSamples, func(sample int, channel int) float32 {
		t := float64(sample) / float64(sampleRate)
		return float32(math.Sin(2 * math.Pi * frequency * t))
	})
}

// newConstantSource creates a mock source with constant value.
func newConstantSource(sampleRate, channels, totalSamples int, value float32) *mockSource {
	return newMockSource(sampleRate, channels, totalSamples, func(sample int, channel int) float32 {
		return value
	})
}

func (m *mockSource) SampleRate() int { return m.sampleRate }
func (m *mockSource) Channels() int   { return m.channels }
func (m *mockSource) BufSize() int    { return 4096 }
func (m *mockSource) Close() error    { return nil }

func (m *mockSource) ReadSamples(dst []float32) (int, error) {
	if m.generated >= m.totalSamples {
		return 0, io.EOF
	}

	// Calculate how many frames we can write
	framesRequested := len(dst) / m.channels
	framesAvailable := m.totalSamples - m.generated
	framesToWrite := framesRequested
	if framesToWrite > framesAvailable {
		framesToWrite = framesAvailable
	}

	// Generate samples
	for frame := range framesToWrite {
		sampleIndex := m.generated + frame
		for ch := range m.channels {
			dst[frame*m.channels+ch] = m.waveform(sampleIndex, ch)
		}
	}

	m.generated += framesToWrite
	samplesWritten := framesToWrite * m.channels

	if m.generated >= m.totalSamples {
		return samplesWritten, io.EOF
	}

	return samplesWritten, nil
}
