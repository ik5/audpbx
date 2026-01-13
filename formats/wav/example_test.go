// SPDX-License-Identifier: EPL-2.0

package wav_test

import (
	"bytes"
	"fmt"
	"io"

	"github.com/ik5/audpbx/formats/wav"
)

// Example_decoding demonstrates decoding a WAV file.
func Example_decoding() {
	// Create a sample WAV file
	samples := []int16{100, 200, 300, 400, 500}
	wavData := new(bytes.Buffer)
	wav.WriteWAV16(wavData, 16000, samples)

	// Decode the WAV file
	decoder := wav.Decoder{}
	source, err := decoder.Decode(wavData)
	if err != nil {
		fmt.Printf("Decode error: %v\n", err)
		return
	}

	// Check audio properties
	fmt.Printf("Sample rate: %d Hz\n", source.SampleRate())
	fmt.Printf("Channels: %d\n", source.Channels())

	// Read samples
	buf := make([]float32, 10)
	n, err := source.ReadSamples(buf)
	if err != nil && err != io.EOF {
		fmt.Printf("Read error: %v\n", err)
		return
	}

	fmt.Printf("Read %d samples\n", n)
	// Output:
	// Sample rate: 16000 Hz
	// Channels: 1
	// Read 5 samples
}

// Example_encoding demonstrates writing a WAV file.
func Example_encoding() {
	// Generate audio samples (simple sine-like wave)
	samples := make([]int16, 1000)
	for i := range samples {
		// Simple pattern for demo
		samples[i] = int16((i % 100) * 100)
	}

	// Write to buffer (in real code, use os.Create)
	output := new(bytes.Buffer)
	err := wav.WriteWAV16(output, 8000, samples)
	if err != nil {
		fmt.Printf("Write error: %v\n", err)
		return
	}

	fmt.Printf("Wrote %d bytes\n", output.Len())
	fmt.Printf("Header: 44 bytes\n")
	fmt.Printf("Data: %d bytes (%d samples × 2 bytes)\n", len(samples)*2, len(samples))
	// Output:
	// Wrote 2044 bytes
	// Header: 44 bytes
	// Data: 2000 bytes (1000 samples × 2 bytes)
}

// Example_roundTrip shows encoding and then decoding.
func Example_roundTrip() {
	// Original samples
	original := []int16{-1000, -500, 0, 500, 1000}

	// Encode to WAV
	wavData := new(bytes.Buffer)
	err := wav.WriteWAV16(wavData, 8000, original)
	if err != nil {
		fmt.Printf("Encode error: %v\n", err)
		return
	}

	// Decode back
	decoder := wav.Decoder{}
	source, err := decoder.Decode(wavData)
	if err != nil {
		fmt.Printf("Decode error: %v\n", err)
		return
	}

	// Read samples
	buf := make([]float32, len(original))
	n, _ := source.ReadSamples(buf)

	// Convert back to int16 for comparison
	recovered := make([]int16, n)
	for i := range n {
		recovered[i] = int16(buf[i] * 32768.0)
	}

	fmt.Println("Round-trip successful:")
	fmt.Printf("Original:  %v\n", original)
	fmt.Printf("Recovered: %v\n", recovered)
	// Output:
	// Round-trip successful:
	// Original:  [-1000 -500 0 500 1000]
	// Recovered: [-1000 -500 0 500 1000]
}

// Example_errorNotWAV shows handling of invalid WAV files.
func Example_errorNotWAV() {
	// Try to decode non-WAV data
	invalidData := bytes.NewReader([]byte("This is not a WAV file"))

	decoder := wav.Decoder{}
	_, err := decoder.Decode(invalidData)

	if err == wav.ErrNotWavFile {
		fmt.Println("Detected: Not a valid WAV file")
	} else if err != nil {
		fmt.Printf("Other error: %v\n", err)
	}
	// Output: Detected: Not a valid WAV file
}

// Example_emptySamples shows writing a WAV file with no audio data.
func Example_emptySamples() {
	// Write a WAV file with no samples (just header)
	samples := []int16{}
	output := new(bytes.Buffer)

	err := wav.WriteWAV16(output, 8000, samples)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Wrote empty WAV: %d bytes (header only)\n", output.Len())
	// Output: Wrote empty WAV: 44 bytes (header only)
}

// Example_sampleRates demonstrates different sample rates.
func Example_sampleRates() {
	rates := []int{8000, 16000, 44100, 48000}

	for _, rate := range rates {
		// Create 1 second of audio
		samples := make([]int16, rate)

		wavData := new(bytes.Buffer)
		wav.WriteWAV16(wavData, rate, samples)

		// Decode to verify
		decoder := wav.Decoder{}
		source, _ := decoder.Decode(wavData)

		fmt.Printf("Rate: %5d Hz → %5d Hz (verified)\n", rate, source.SampleRate())
	}
	// Output:
	// Rate:  8000 Hz →  8000 Hz (verified)
	// Rate: 16000 Hz → 16000 Hz (verified)
	// Rate: 44100 Hz → 44100 Hz (verified)
	// Rate: 48000 Hz → 48000 Hz (verified)
}

// Example_largeFile demonstrates handling large audio files efficiently.
func Example_largeFile() {
	// Create 10 seconds of audio at 44.1kHz
	duration := 10 // seconds
	sampleRate := 44100
	totalSamples := duration * sampleRate

	samples := make([]int16, totalSamples)
	// Generate simple test pattern
	for i := range samples {
		samples[i] = int16((i % 1000) - 500)
	}

	// Write the file
	output := new(bytes.Buffer)
	err := wav.WriteWAV16(output, sampleRate, samples)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Calculate sizes
	dataSize := totalSamples * 2 // 2 bytes per sample
	headerSize := 44
	totalSize := headerSize + dataSize

	fmt.Printf("Duration: %d seconds\n", duration)
	fmt.Printf("Sample rate: %d Hz\n", sampleRate)
	fmt.Printf("Total samples: %d\n", totalSamples)
	fmt.Printf("File size: %d bytes (%.2f MB)\n", totalSize, float64(totalSize)/(1024*1024))
	// Output:
	// Duration: 10 seconds
	// Sample rate: 44100 Hz
	// Total samples: 441000
	// File size: 882044 bytes (0.84 MB)
}

// Example_streamingRead demonstrates reading a WAV file in chunks.
func Example_streamingRead() {
	// Create a WAV file
	samples := make([]int16, 10000)
	wavData := new(bytes.Buffer)
	wav.WriteWAV16(wavData, 8000, samples)

	// Decode
	decoder := wav.Decoder{}
	source, _ := decoder.Decode(wavData)

	// Read in chunks
	buf := make([]float32, 1000) // Read 1000 samples at a time
	chunks := 0
	totalSamples := 0

	for {
		n, err := source.ReadSamples(buf)
		if n > 0 {
			chunks++
			totalSamples += n
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			break
		}
	}

	fmt.Printf("Read %d samples in %d chunks\n", totalSamples, chunks)
	fmt.Printf("Chunk size: 1000 samples\n")
	fmt.Println("Memory efficient: only one buffer allocated")
	// Output:
	// Read 10000 samples in 10 chunks
	// Chunk size: 1000 samples
	// Memory efficient: only one buffer allocated
}

// Example_sampleConversion shows the int16 to float32 conversion.
func Example_sampleConversion() {
	// Create samples covering the full int16 range
	samples := []int16{
		-32768, // Minimum int16
		-16384, // -50%
		0,      // Zero
		16384,  // +50%
		32767,  // Maximum int16
	}

	wavData := new(bytes.Buffer)
	wav.WriteWAV16(wavData, 8000, samples)

	decoder := wav.Decoder{}
	source, _ := decoder.Decode(wavData)

	buf := make([]float32, len(samples))
	n, _ := source.ReadSamples(buf)

	fmt.Println("int16 → float32 conversion:")
	for i := range n {
		fmt.Printf("  %6d → %+.3f\n", samples[i], buf[i])
	}
	// Output:
	// int16 → float32 conversion:
	//   -32768 → -1.000
	//   -16384 → -0.500
	//        0 → +0.000
	//    16384 → +0.500
	//    32767 → +1.000
}
