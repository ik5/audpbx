// SPDX-License-Identifier: EPL-2.0

// Package audpbx provides high-level audio processing utilities for Go applications.
//
// This package offers convenient functions for common audio processing tasks such as
// resampling, format conversion, and decoding various audio formats. It's designed to
// be simple to use while maintaining good performance.
//
// # Supported Formats
//
// The package supports decoding the following audio formats:
//   - WAV (PCM 16-bit) via formats/wav
//   - MP3 via formats/mp3
//   - Ogg Vorbis via formats/vorbis
//   - AIFF (PCM 16-bit) via formats/aiff
//
// # Quick Start
//
// The simplest way to process audio is using ResampleToMono16:
//
//	// Decode an audio file
//	decoder := wav.Decoder{}
//	file, _ := os.Open("audio.wav")
//	src, _ := decoder.Decode(file)
//
//	// Resample to 8kHz mono, 16-bit PCM
//	samples, rate, _ := audpbx.ResampleToMono16(src, 8000, 4096)
//
//	// samples is now []int16 at 8kHz mono
//
// # Audio Processing Pipeline
//
// For more control, you can build custom audio processing pipelines using the
// audio subpackage:
//
//	// Create a resampler
//	resampler := audio.NewResampler(source, 16000)
//
//	// Convert to mono
//	mono := audio.NewMonoMixer(resampler)
//
//	// Read samples
//	buf := make([]float32, 4096)
//	n, err := mono.ReadSamples(buf)
//
// # Format Decoders
//
// Each format has its own decoder:
//
//	// WAV
//	wavDecoder := wav.Decoder{}
//	src, _ := wavDecoder.Decode(reader)
//
//	// MP3
//	mp3Decoder := mp3.Decoder{}
//	src, _ := mp3Decoder.Decode(reader)
//
//	// Vorbis
//	vorbisDecoder := vorbis.Decoder{}
//	src, _ := vorbisDecoder.Decode(reader)
//
//	// AIFF
//	aiffDecoder := aiff.Decoder{}
//	src, _ := aiffDecoder.Decode(reader)
//
// All decoders return an audio.Source interface which can be used with
// the audio processing functions.
//
// # Writing WAV Files
//
// The package can write PCM WAV files:
//
//	samples := []int16{100, -100, 200, -200}
//	file, _ := os.Create("output.wav")
//	wav.WriteWAV16(file, 8000, samples)
//
// # Performance
//
// The package is optimized for performance with minimal allocations:
//   - Resampling uses cubic interpolation for quality
//   - Buffer reuse minimizes GC pressure
//   - Batch conversions reduce per-sample overhead
//
// See the individual subpackages for more detailed documentation.
package audpbx
