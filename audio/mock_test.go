package audio

import (
	"github.com/ik5/audpbx/internal/audiotest"
)

// Convenience wrappers for test helpers from internal/audiotest
// These maintain backward compatibility with existing tests in this package
// The return type is *audiotest.MockSource which implements audio.Source interface

func newMockSource(sampleRate, channels, totalSamples int, waveform func(sample int, channel int) float32) *audiotest.MockSource {
	return audiotest.NewMockSource(sampleRate, channels, totalSamples, waveform)
}

func newSilentSource(sampleRate, channels, totalSamples int) *audiotest.MockSource {
	return audiotest.NewSilentSource(sampleRate, channels, totalSamples)
}

func newSineSource(sampleRate, channels, totalSamples int, frequency float64) *audiotest.MockSource {
	return audiotest.NewSineSource(sampleRate, channels, totalSamples, frequency)
}

func newConstantSource(sampleRate, channels, totalSamples int, value float32) *audiotest.MockSource {
	return audiotest.NewConstantSource(sampleRate, channels, totalSamples, value)
}
