// SPDX-License-Identifier: EPL-2.0

// Package wav provides WAV audio file decoding and encoding.
//
// This package supports reading and writing WAV files in PCM 16-bit format.
// It uses the github.com/go-audio library for robust WAV file handling.
//
// # Supported Formats
//
// Currently supported:
//   - PCM 16-bit (most common WAV format), format tag 1
//   - Mono, stereo, and more channels provided the file uses format tag 1
//   - Any sample rate
//
// Not supported:
//   - Bit depths other than 16 (8, 24, and 32 are rejected)
//   - WAVE_FORMAT_EXTENSIBLE (format tag 0xFFFE), which is what ffmpeg writes
//     for files with more than two channels. Such files fail with an
//     "unsupported audio format" error.
//   - Compressed WAV payloads of any kind
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
// The decoder reads and converts 16-bit samples directly from the PCM chunk
// rather than going through wav.Decoder.PCMBuffer. PCMBuffer allocates a byte
// buffer, a bytes.Reader, a format struct and a per-sample scratch buffer on
// every call, and decodes one sample at a time through a closure into an []int
// four times wider than the samples it carries. Reading the chunk directly
// reduces that to a single read plus a tight decode loop, which reuses its
// staging buffer across calls.
//
// The result is that decoding is bounded by memory bandwidth rather than by
// per-call overhead, and a full file conversion allocates a handful of times in
// total rather than several times per audio frame.
//
// The decoder is stream-based, so memory use does not scale with file size.
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
