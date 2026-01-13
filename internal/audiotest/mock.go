// SPDX-License-Identifier: EPL-2.0

package audiotest

import (
	"io"
	"math"
)

// MockSource is a test helper that generates audio data for testing.
// It implements the audio.Source interface (without importing it to avoid cycles).
type MockSource struct {
	sampleRate   int
	channels     int
	totalSamples int // Total samples to generate (per channel)
	generated    int // Samples generated so far (per channel)
	waveform     func(sample int, channel int) float32
}

// NewMockSource creates a new mock audio source.
// totalSamples is the total number of samples per channel to generate.
// waveform is a function that generates sample values given sample index and channel.
func NewMockSource(sampleRate, channels, totalSamples int, waveform func(sample int, channel int) float32) *MockSource {
	return &MockSource{
		sampleRate:   sampleRate,
		channels:     channels,
		totalSamples: totalSamples,
		generated:    0,
		waveform:     waveform,
	}
}

// NewSilentSource creates a mock source that generates silence (all zeros).
func NewSilentSource(sampleRate, channels, totalSamples int) *MockSource {
	return NewMockSource(sampleRate, channels, totalSamples, func(sample int, channel int) float32 {
		return 0.0
	})
}

// NewSineSource creates a mock source that generates a sine wave.
func NewSineSource(sampleRate, channels, totalSamples int, frequency float64) *MockSource {
	return NewMockSource(sampleRate, channels, totalSamples, func(sample int, channel int) float32 {
		t := float64(sample) / float64(sampleRate)
		return float32(math.Sin(2 * math.Pi * frequency * t))
	})
}

// NewConstantSource creates a mock source with constant value.
func NewConstantSource(sampleRate, channels, totalSamples int, value float32) *MockSource {
	return NewMockSource(sampleRate, channels, totalSamples, func(sample int, channel int) float32 {
		return value
	})
}

func (m *MockSource) SampleRate() int { return m.sampleRate }
func (m *MockSource) Channels() int   { return m.channels }
func (m *MockSource) BufSize() int    { return 4096 }
func (m *MockSource) Close() error    { return nil }

// Reset resets the generated sample counter to allow re-reading
func (m *MockSource) Reset() {
	m.generated = 0
}

func (m *MockSource) ReadSamples(dst []float32) (int, error) {
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
