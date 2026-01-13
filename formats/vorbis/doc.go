// SPDX-License-Identifier: EPL-2.0

// Package vorbis provides Ogg Vorbis audio file decoding.
//
// This package uses github.com/jfreymuth/oggvorbis to decode Ogg Vorbis files.
// Vorbis is a free, open-source lossy audio compression format.
//
// # Supported Formats
//
// The decoder supports:
//   - Ogg Vorbis (.ogg files)
//   - Variable bitrates
//   - Mono and stereo
//   - Various sample rates
//
// # Decoding Vorbis Files
//
// Use the Decoder to read Ogg Vorbis files:
//
//	decoder := vorbis.Decoder{}
//	file, _ := os.Open("audio.ogg")
//	source, err := decoder.Decode(file)
//	if err != nil {
//	    // Handle error
//	}
//
//	// Read samples as float32 in range [-1.0, 1.0]
//	buf := make([]float32, 4096)
//	n, err := source.ReadSamples(buf)
//
// The decoder returns an audio.Source that provides samples as float32
// values normalized to the range [-1.0, 1.0].
//
// # Output Format
//
// Vorbis decoder output:
//   - Sample format: float32 in range [-1.0, 1.0]
//   - Channels: Depends on file (mono or stereo typically)
//   - Sample rate: Depends on file (commonly 44.1kHz or 48kHz)
//
// # Channel Layout
//
// For stereo files, samples are interleaved:
//
//	[L0, R0, L1, R1, L2, R2, ...]
//
// To convert to mono:
//
//	vorbisSource, _ := decoder.Decode(file)
//	mono := audio.NewMonoMixer(vorbisSource)
//
// # Performance
//
// The Vorbis decoder:
//   - Streams data efficiently
//   - Minimal allocations during reading
//   - Suitable for real-time playback
//
// # Limitations
//
// Note:
//   - Vorbis encoding is not supported (decoding only)
//   - Reading is frame-based (decode entire frames)
//
// # Use Cases
//
// Common applications:
//   - Playing Ogg Vorbis files
//   - Converting Vorbis to WAV
//   - Game audio (common format in games)
//   - Audio streaming
//
// # Quality vs. Compression
//
// Vorbis provides excellent quality at various bitrates:
//   - Low quality: ~64 kbps (voice/podcasts)
//   - Standard quality: ~128 kbps (music)
//   - High quality: ~192-256 kbps (archival)
//
// The decoder handles all quality levels transparently.
//
// # Example: Vorbis to WAV Conversion
//
//	// Read Ogg Vorbis file
//	oggFile, _ := os.Open("input.ogg")
//	vorbisDecoder := vorbis.Decoder{}
//	source, _ := vorbisDecoder.Decode(oggFile)
//
//	// Resample and convert to mono
//	pcm16, rate, _ := audpbx.ResampleToMono16(source, 16000, 4096)
//
//	// Write as WAV
//	wavFile, _ := os.Create("output.wav")
//	wav.WriteWAV16(wavFile, rate, pcm16)
package vorbis
