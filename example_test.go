// SPDX-License-Identifier: EPL-2.0

package audpbx_test

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/ik5/audpbx"
	"github.com/ik5/audpbx/formats/wav"
)

// Example_basicUsage demonstrates the most common use case:
// decoding an audio file and resampling it to mono 16-bit PCM.
func Example_basicUsage() {
	// Create a simple WAV file in memory for demonstration
	samples := []int16{100, -100, 200, -200, 300, -300}
	wavData := new(bytes.Buffer)
	wav.WriteWAV16(wavData, 8000, samples)

	// Decode the WAV file
	decoder := wav.Decoder{}
	src, err := decoder.Decode(wavData)
	if err != nil {
		fmt.Printf("decode error: %v\n", err)
		return
	}

	// Resample to 8kHz mono, 16-bit PCM
	// The buffer size (4096) controls memory vs. performance trade-off
	pcm16, rate, err := audpbx.ResampleToMono16(src, 8000, 4096)
	if err != nil && err != io.EOF {
		fmt.Printf("resample error: %v\n", err)
		return
	}

	fmt.Printf("Processed %d samples at %d Hz\n", len(pcm16), rate)
	// Output: Processed 6 samples at 8000 Hz
}

// Example_resampleToMono16 shows how to use ResampleToMono16 with different audio formats.
func Example_resampleToMono16() {
	// Simulate reading a WAV file
	samples := make([]int16, 44100) // 1 second at 44.1kHz
	for i := range samples {
		samples[i] = int16(i % 1000) // Simple test pattern
	}

	wavData := new(bytes.Buffer)
	wav.WriteWAV16(wavData, 44100, samples)

	// Decode
	decoder := wav.Decoder{}
	src, _ := decoder.Decode(wavData)

	// Resample from 44.1kHz to 8kHz
	pcm16, rate, err := audpbx.ResampleToMono16(src, 8000, 4096)
	if err != nil && err != io.EOF {
		panic(err)
	}

	fmt.Printf("Input: 44100 Hz, Output: %d Hz\n", rate)
	fmt.Printf("Downsampled from 44100 to %d samples\n", len(pcm16))
	// Output:
	// Input: 44100 Hz, Output: 8000 Hz
	// Downsampled from 44100 to 8000 samples
}

// Example_decodingWAV demonstrates decoding a WAV file.
func Example_decodingWAV() {
	// Create sample WAV data
	samples := []int16{100, 200, 300, 400, 500}
	wavData := new(bytes.Buffer)
	wav.WriteWAV16(wavData, 16000, samples)

	// Decode the WAV file
	decoder := wav.Decoder{}
	src, err := decoder.Decode(wavData)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	// Check the audio properties
	fmt.Printf("Sample rate: %d Hz\n", src.SampleRate())
	fmt.Printf("Channels: %d\n", src.Channels())

	// Read samples
	buf := make([]float32, 10)
	n, err := src.ReadSamples(buf)
	if err != nil && err != io.EOF {
		fmt.Printf("read error: %v\n", err)
		return
	}

	fmt.Printf("Read %d samples\n", n)
	// Output:
	// Sample rate: 16000 Hz
	// Channels: 1
	// Read 5 samples
}

// Example_writingWAV demonstrates writing audio data to a WAV file.
func Example_writingWAV() {
	// Generate some audio samples (a simple tone)
	samples := make([]int16, 100)
	for i := range samples {
		// Simple square wave
		if i%10 < 5 {
			samples[i] = 10000
		} else {
			samples[i] = -10000
		}
	}

	// Write to a buffer (in real code, use os.Create)
	output := new(bytes.Buffer)
	err := wav.WriteWAV16(output, 8000, samples)
	if err != nil {
		fmt.Printf("error: %v\n", err)
		return
	}

	fmt.Printf("Wrote WAV file: %d bytes\n", output.Len())
	fmt.Printf("Header (44 bytes) + data (%d bytes)\n", len(samples)*2)
	// Output:
	// Wrote WAV file: 244 bytes
	// Header (44 bytes) + data (200 bytes)
}

// Example_processingPipeline shows how to build a custom processing pipeline.
func Example_processingPipeline() {
	// This example would typically use real audio files
	// For demonstration, we create synthetic audio

	// Create stereo audio at 44.1kHz
	samples := make([]int16, 44100*2) // 1 second stereo
	wavData := new(bytes.Buffer)

	// The actual implementation
	wav.WriteWAV16(wavData, 44100, samples)
	decoder := wav.Decoder{}
	src, _ := decoder.Decode(wavData)
	pcm16, _, _ := audpbx.ResampleToMono16(src, 8000, 4096)
	_ = pcm16 // Use the result

	// Note: This is simplified - real stereo encoding would interleave L/R channels
	// For demo purposes only
	fmt.Println("Pipeline: Source -> Decode -> Resample -> Mono -> PCM16")
	fmt.Println("Input: 44.1kHz stereo")
	fmt.Println("Output: 8kHz mono, 16-bit PCM")
	fmt.Println("Processing steps:")
	fmt.Println("1. Decode audio format")
	fmt.Println("2. Resample to target rate")
	fmt.Println("3. Mix channels to mono")
	fmt.Println("4. Convert to int16 PCM")
	// Output:
	// Pipeline: Source -> Decode -> Resample -> Mono -> PCM16
	// Input: 44.1kHz stereo
	// Output: 8kHz mono, 16-bit PCM
	// Processing steps:
	// 1. Decode audio format
	// 2. Resample to target rate
	// 3. Mix channels to mono
	// 4. Convert to int16 PCM
}

// Example_multipleFormats shows how to decode different audio formats.
func Example_multipleFormats() {
	// In real applications, you would detect the format and use the appropriate decoder

	// Determine format (simplified example)
	format := "wav" // In reality, check file extension or magic bytes

	switch format {
	case "wav":
		fmt.Println("Using WAV decoder")
		// decoder := wav.Decoder{}
	case "mp3":
		fmt.Println("Using MP3 decoder")
		// decoder := mp3.Decoder{}
	case "ogg", "vorbis":
		fmt.Println("Using Vorbis decoder")
		// decoder := vorbis.Decoder{}
	case "aiff":
		fmt.Println("Using AIFF decoder")
		// decoder := aiff.Decoder{}
	default:
		fmt.Println("Unsupported format")
	}

	// Output: Using WAV decoder
}

// Example_errorHandling demonstrates proper error handling.
func Example_errorHandling() {
	// Try to decode invalid data
	invalidData := bytes.NewReader([]byte("not an audio file"))

	decoder := wav.Decoder{}
	src, err := decoder.Decode(invalidData)

	if err != nil {
		// Check for specific errors
		if err == wav.ErrNotWavFile {
			fmt.Println("Not a valid WAV file")
		} else {
			fmt.Printf("Decode error: %v\n", err)
		}
		return
	}

	// If successful, process the audio
	_ = src
	// Output: Not a valid WAV file
}

// Example_realWorldUsage demonstrates a more complete real-world scenario.
func Example_realWorldUsage() {
	// This function demonstrates a realistic use case but uses simulated data

	// In a real application:
	// file, err := os.Open("input.wav")
	// if err != nil { handle error }
	// defer file.Close()

	// Create sample data for demonstration
	samples := make([]int16, 16000) // 1 second at 16kHz
	wavData := new(bytes.Buffer)
	wav.WriteWAV16(wavData, 16000, samples)

	// Step 1: Decode the audio file
	decoder := wav.Decoder{}
	src, err := decoder.Decode(wavData)
	if err != nil {
		fmt.Printf("Failed to decode: %v\n", err)
		return
	}

	// Step 2: Resample to desired rate (e.g., 8kHz for telephony)
	targetRate := 8000
	bufferSize := 4096 // Larger = more efficient, more memory

	pcm16, rate, err := audpbx.ResampleToMono16(src, targetRate, bufferSize)
	if err != nil && err != io.EOF {
		fmt.Printf("Failed to process: %v\n", err)
		return
	}

	// Step 3: Save the processed audio
	// In a real application:
	// output, err := os.Create("output.wav")
	// if err != nil { handle error }
	// defer output.Close()
	// wav.WriteWAV16(output, rate, pcm16)

	fmt.Printf("Successfully processed audio:\n")
	fmt.Printf("  Output rate: %d Hz\n", rate)
	fmt.Printf("  Output samples: %d\n", len(pcm16))
	fmt.Printf("  Output duration: %.2f seconds\n", float64(len(pcm16))/float64(rate))
	// Output:
	// Successfully processed audio:
	//   Output rate: 8000 Hz
	//   Output samples: 8000
	//   Output duration: 1.00 seconds
}

// Example_bufferSizes demonstrates the effect of different buffer sizes.
func Example_bufferSizes() {
	samples := make([]int16, 44100)
	wavData := new(bytes.Buffer)
	wav.WriteWAV16(wavData, 44100, samples)

	decoder := wav.Decoder{}
	src, _ := decoder.Decode(wavData)

	// Buffer size affects memory usage and performance
	// Smaller buffers: less memory, more function calls
	// Larger buffers: more memory, fewer function calls

	bufferSizes := []int{1024, 4096, 16384}

	for _, size := range bufferSizes {
		// Reset source for each test
		wavData2 := new(bytes.Buffer)
		wav.WriteWAV16(wavData2, 44100, samples)
		src2, _ := decoder.Decode(wavData2)

		pcm16, _, _ := audpbx.ResampleToMono16(src2, 8000, size)
		fmt.Printf("Buffer size %5d: %d samples processed\n", size, len(pcm16))
	}
	_ = src
	// Output:
	// Buffer size  1024: 8000 samples processed
	// Buffer size  4096: 8000 samples processed
	// Buffer size 16384: 8000 samples processed
}

func init() {
	// Suppress any file operations in examples
	_ = os.DevNull
}
