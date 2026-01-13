// SPDX-License-Identifier: EPL-2.0

// Package mp3 provides MP3 audio file decoding.
//
// This package uses github.com/hajimehoshi/go-mp3 to decode MP3 files.
// It provides a simple interface for reading MP3 audio as PCM samples.
//
// # Supported Formats
//
// The decoder supports:
//   - MP3 (MPEG-1 Audio Layer 3)
//   - Various bitrates
//   - Stereo output (most MP3 files)
//
// # Decoding MP3 Files
//
// Use the Decoder to read MP3 files:
//
//	decoder := mp3.Decoder{}
//	file, _ := os.Open("audio.mp3")
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
// MP3 decoder output:
//   - Sample format: float32 in range [-1.0, 1.0]
//   - Channels: 2 (stereo)
//   - Sample rate: Depends on the MP3 file (typically 44.1kHz or 48kHz)
//
// To convert to mono or resample, use the audio package:
//
//	// Convert stereo MP3 to mono 8kHz
//	mp3Source, _ := decoder.Decode(file)
//	resampled := audio.NewResampler(mp3Source, 8000)
//	mono := audio.NewMonoMixer(resampled)
//
// # Performance
//
// The MP3 decoder:
//   - Streams data efficiently
//   - Minimal allocations during reading
//   - Suitable for real-time processing
//
// # Limitations
//
// Note:
//   - MP3 writing is not supported (decoding only)
//   - Output is always stereo (use MonoMixer to convert)
//   - Requires reading entire frames for decoding
//
// # Use Cases
//
// Common applications:
//   - Playing MP3 files
//   - Converting MP3 to WAV
//   - Audio analysis
//   - Voice processing pipelines
//
// Example converting MP3 to WAV:
//
//	mp3File, _ := os.Open("input.mp3")
//	mp3Decoder := mp3.Decoder{}
//	source, _ := mp3Decoder.Decode(mp3File)
//
//	// Resample and convert to mono
//	pcm16, rate, _ := audpbx.ResampleToMono16(source, 8000, 4096)
//
//	// Write as WAV
//	wavFile, _ := os.Create("output.wav")
//	wav.WriteWAV16(wavFile, rate, pcm16)
package mp3
