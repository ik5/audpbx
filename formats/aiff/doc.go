// SPDX-License-Identifier: EPL-2.0

// Package aiff provides AIFF (Audio Interchange File Format) decoding.
//
// This package uses github.com/go-audio/aiff to decode AIFF files.
// AIFF is Apple's standard audio file format, commonly used on macOS.
//
// # Supported Formats
//
// Currently supported:
//   - AIFF (Audio Interchange File Format)
//   - PCM 16-bit (most common)
//   - Mono and multi-channel
//   - Any sample rate
//
// # Decoding AIFF Files
//
// Use the Decoder to read AIFF files:
//
//	decoder := aiff.Decoder{}
//	file, _ := os.Open("audio.aif")
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
// AIFF decoder output:
//   - Sample format: float32 in range [-1.0, 1.0]
//   - Channels: Depends on file (mono or stereo typically)
//   - Sample rate: Depends on file (commonly 44.1kHz or 48kHz)
//
// # Error Handling
//
// The package defines several error types:
//   - ErrNotAiffFile: The input is not a valid AIFF file
//   - ErrOnlyPCM16bitSupported: Only 16-bit PCM is currently supported
//   - ErrUnsupportedAiffLayout: Unsupported AIFF file structure
//
// Example:
//
//	source, err := decoder.Decode(file)
//	if err == aiff.ErrNotAiffFile {
//	    fmt.Println("Not an AIFF file")
//	}
//
// # AIFF vs. WAV
//
// AIFF is similar to WAV but:
//   - Uses big-endian byte order (WAV uses little-endian)
//   - Originated on Apple platforms (WAV on Windows)
//   - Stores sample rate as 80-bit float (WAV uses 32-bit int)
//   - Both are uncompressed PCM formats
//
// The decoder handles all format differences automatically.
//
// # Performance
//
// The AIFF decoder:
//   - Streams data efficiently
//   - Minimal allocations (2 per read)
//   - Efficient buffer management
//   - Zero allocations in benchmarks
//
// # Limitations
//
// Note:
//   - AIFF writing is not supported (decoding only)
//   - Only 16-bit PCM is supported (no 8-bit, 24-bit, or compressed formats)
//   - For other bit depths, you'll get ErrOnlyPCM16bitSupported
//
// # Use Cases
//
// Common applications:
//   - Playing AIFF files on macOS
//   - Converting AIFF to other formats
//   - Audio production workflows
//   - Professional audio applications
//
// # Example: AIFF to WAV Conversion
//
//	// Read AIFF file
//	aiffFile, _ := os.Open("input.aif")
//	aiffDecoder := aiff.Decoder{}
//	source, _ := aiffDecoder.Decode(aiffFile)
//
//	// Resample to 8kHz mono
//	pcm16, rate, _ := audpbx.ResampleToMono16(source, 8000, 4096)
//
//	// Write as WAV
//	wavFile, _ := os.Create("output.wav")
//	wav.WriteWAV16(wavFile, rate, pcm16)
//
// # File Extensions
//
// AIFF files typically use:
//   - .aif or .aiff for standard AIFF
//   - .aifc for AIFF-C (compressed, not supported)
//
// Always check for ErrOnlyPCM16bitSupported when opening files.
package aiff
