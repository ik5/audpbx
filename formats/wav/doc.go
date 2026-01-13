// SPDX-License-Identifier: EPL-2.0

// Package wav provides WAV audio file decoding and encoding.
//
// This package supports reading and writing WAV files in PCM 16-bit format.
// It uses the github.com/go-audio library for robust WAV file handling.
//
// # Supported Formats
//
// Currently supported:
//   - PCM 16-bit (most common WAV format)
//   - Mono and stereo
//   - Any sample rate
//
// # Decoding WAV Files
//
// Use the Decoder to read WAV files:
//
//	decoder := wav.Decoder{}
//	file, _ := os.Open("audio.wav")
//	source, err := decoder.Decode(file)
//	if err != nil {
//	    // Handle error
//	}
//
//	// Read samples
//	buf := make([]float32, 4096)
//	n, err := source.ReadSamples(buf)
//
// The decoder returns an audio.Source that provides samples as float32
// values in the range [-1.0, 1.0].
//
// # Writing WAV Files
//
// Use WriteWAV16 to create WAV files:
//
//	samples := []int16{100, -100, 200, -200}
//	file, _ := os.Create("output.wav")
//	err := wav.WriteWAV16(file, 8000, samples)
//
// The function writes a complete WAV file with proper headers.
//
// # Error Handling
//
// The package defines several error types:
//   - ErrNotWavFile: The input is not a valid WAV file
//   - ErrOnlyPCM16bitSupported: Only 16-bit PCM is supported
//   - ErrUnsupportedWavLayout: Unsupported WAV file structure
//
// Example:
//
//	source, err := decoder.Decode(file)
//	if err == wav.ErrNotWavFile {
//	    fmt.Println("Not a WAV file")
//	}
//
// # Performance
//
// The WAV encoder is highly optimized:
//   - Near-zero allocations (5-11 allocations per file)
//   - Chunked writing for large files
//   - Pre-allocated header buffer
//
// The decoder provides:
//   - Minimal allocations (2 per read)
//   - Efficient buffer management
//   - Stream-based reading for memory efficiency
//
// # File Format
//
// WAV files consist of:
//   - RIFF header (12 bytes)
//   - fmt chunk (24 bytes): audio format, sample rate, channels, bit depth
//   - data chunk: actual audio samples
//
// The WriteWAV16 function handles all format details automatically.
package wav
