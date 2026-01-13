// SPDX-License-Identifier: EPL-2.0

package audio_test

import (
	"fmt"
	"io"

	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/internal/audiotest"
)

// Example_resampler demonstrates how to use the Resampler to change sample rates.
func Example_resampler() {
	// Create a test audio source at 44.1kHz
	source := audiotest.NewSineSource(44100, 1, 44100, 440.0) // 1 second, 440Hz tone

	// Create a resampler to convert to 16kHz
	resampler := audio.NewResampler(source, 16000)

	// Check the output properties
	fmt.Printf("Output sample rate: %d Hz\n", resampler.SampleRate())
	fmt.Printf("Channels: %d\n", resampler.Channels())

	// Read samples
	buf := make([]float32, 4096)
	totalSamples := 0

	for {
		n, err := resampler.ReadSamples(buf)
		totalSamples += n

		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error: %v\n", err)
			return
		}
	}

	fmt.Printf("Total samples read: %d\n", totalSamples)
	// Output:
	// Output sample rate: 16000 Hz
	// Channels: 1
	// Total samples read: 16000
}

// Example_monoMixer demonstrates converting stereo to mono.
func Example_monoMixer() {
	// Create a stereo audio source
	source := audiotest.NewSineSource(16000, 2, 16000, 440.0) // 1 second stereo

	// Create a mono mixer
	mono := audio.NewMonoMixer(source)

	// Check the output properties
	fmt.Printf("Input channels: %d\n", source.Channels())
	fmt.Printf("Output channels: %d\n", mono.Channels())
	fmt.Printf("Sample rate: %d Hz\n", mono.SampleRate())

	// Read some samples
	buf := make([]float32, 100)
	n, _ := mono.ReadSamples(buf)

	fmt.Printf("Read %d mono samples\n", n)
	// Output:
	// Input channels: 2
	// Output channels: 1
	// Sample rate: 16000 Hz
	// Read 100 mono samples
}

// Example_processingChain shows how to chain resampler and mono mixer.
func Example_processingChain() {
	// Start with stereo audio at 44.1kHz
	source := audiotest.NewSineSource(44100, 2, 44100, 440.0)

	// Step 1: Resample to 8kHz
	resampled := audio.NewResampler(source, 8000)

	// Step 2: Convert to mono
	mono := audio.NewMonoMixer(resampled)

	// Now we have 8kHz mono audio
	fmt.Printf("Final output:\n")
	fmt.Printf("  Sample rate: %d Hz\n", mono.SampleRate())
	fmt.Printf("  Channels: %d\n", mono.Channels())

	// Read all the samples
	buf := make([]float32, 4096)
	totalSamples := 0

	for {
		n, err := mono.ReadSamples(buf)
		totalSamples += n
		if err == io.EOF {
			break
		}
	}

	fmt.Printf("  Total samples: %d\n", totalSamples)
	fmt.Printf("  Duration: %.2f seconds\n", float64(totalSamples)/float64(mono.SampleRate()))
	// Output:
	// Final output:
	//   Sample rate: 8000 Hz
	//   Channels: 1
	//   Total samples: 8000
	//   Duration: 1.00 seconds
}

// mockDecoder is a simple decoder for testing the registry.
type mockDecoder struct{}

func (m mockDecoder) Decode(r io.Reader) (audio.Source, error) {
	return audiotest.NewSineSource(16000, 1, 1000, 440.0), nil
}

// Example_registry demonstrates the format registry.
func Example_registry() {
	// Create a new registry
	registry := audio.NewRegistry()

	// Register a decoder
	registry.Register("mock", mockDecoder{})

	// Retrieve the decoder
	decoder, ok := registry.Get("mock")
	if !ok {
		fmt.Println("Decoder not found")
		return
	}

	fmt.Printf("Retrieved decoder: %T\n", decoder)

	// Try to get an unregistered format
	_, ok = registry.Get("unknown")
	if !ok {
		fmt.Println("Unknown format not found in registry")
	}
	// Output:
	// Retrieved decoder: audio_test.mockDecoder
	// Unknown format not found in registry
}

// Example_sampleFormat explains the sample format used.
func Example_sampleFormat() {
	// Audio samples are float32 in range [-1.0, 1.0]

	// Create some example samples
	samples := []float32{
		0.0,   // Silence
		0.5,   // Half amplitude positive
		-0.5,  // Half amplitude negative
		1.0,   // Maximum positive
		-1.0,  // Maximum negative
	}

	fmt.Println("Sample format: float32 in range [-1.0, 1.0]")
	fmt.Println("Sample values:")
	for i, s := range samples {
		var description string
		switch {
		case s == 0:
			description = "silence"
		case s > 0 && s < 1:
			description = "positive amplitude"
		case s < 0 && s > -1:
			description = "negative amplitude"
		case s == 1:
			description = "maximum positive"
		case s == -1:
			description = "maximum negative"
		}
		fmt.Printf("  samples[%d] = %+.1f (%s)\n", i, s, description)
	}
	// Output:
	// Sample format: float32 in range [-1.0, 1.0]
	// Sample values:
	//   samples[0] = +0.0 (silence)
	//   samples[1] = +0.5 (positive amplitude)
	//   samples[2] = -0.5 (negative amplitude)
	//   samples[3] = +1.0 (maximum positive)
	//   samples[4] = -1.0 (maximum negative)
}

// Example_buffering demonstrates efficient buffer management.
func Example_buffering() {
	source := audiotest.NewSineSource(16000, 1, 16000, 440.0)

	// Reuse buffer to avoid allocations
	buf := make([]float32, 4096) // Allocate once

	readCount := 0
	for {
		n, err := source.ReadSamples(buf) // Reuse same buffer
		if n > 0 {
			readCount++
			// Process samples in buf[0:n]
		}
		if err == io.EOF {
			break
		}
	}

	fmt.Printf("Read audio in %d chunks with one buffer allocation\n", readCount)
	fmt.Printf("Buffer size: 4096 samples\n")
	fmt.Printf("Total allocations: 1 (the buffer)\n")
	// Output:
	// Read audio in 4 chunks with one buffer allocation
	// Buffer size: 4096 samples
	// Total allocations: 1 (the buffer)
}

// Example_upsampling shows upsampling (increasing sample rate).
func Example_upsampling() {
	// Start with 8kHz audio
	source := audiotest.NewSineSource(8000, 1, 8000, 440.0)

	// Upsample to 48kHz
	resampler := audio.NewResampler(source, 48000)

	fmt.Printf("Input rate: %d Hz\n", source.SampleRate())
	fmt.Printf("Output rate: %d Hz\n", resampler.SampleRate())
	fmt.Printf("Ratio: %.1fx (upsampling)\n", float64(48000)/float64(8000))

	// Count output samples
	buf := make([]float32, 4096)
	total := 0
	for {
		n, err := resampler.ReadSamples(buf)
		total += n
		if err == io.EOF {
			break
		}
	}

	fmt.Printf("Input samples: 8000\n")
	fmt.Printf("Output samples: %d\n", total)
	// Output:
	// Input rate: 8000 Hz
	// Output rate: 48000 Hz
	// Ratio: 6.0x (upsampling)
	// Input samples: 8000
	// Output samples: 48000
}

// Example_downsampling shows downsampling (decreasing sample rate).
func Example_downsampling() {
	// Start with 48kHz audio
	source := audiotest.NewSineSource(48000, 1, 48000, 440.0)

	// Downsample to 8kHz
	resampler := audio.NewResampler(source, 8000)

	fmt.Printf("Input rate: %d Hz\n", source.SampleRate())
	fmt.Printf("Output rate: %d Hz\n", resampler.SampleRate())
	fmt.Printf("Ratio: %.1fx (downsampling)\n", float64(48000)/float64(8000))

	// Count output samples
	buf := make([]float32, 4096)
	total := 0
	for {
		n, err := resampler.ReadSamples(buf)
		total += n
		if err == io.EOF {
			break
		}
	}

	fmt.Printf("Input samples: 48000\n")
	fmt.Printf("Output samples: %d\n", total)
	// Output:
	// Input rate: 48000 Hz
	// Output rate: 8000 Hz
	// Ratio: 6.0x (downsampling)
	// Input samples: 48000
	// Output samples: 8000
}

// Example_multiChannel demonstrates multi-channel mixing.
func Example_multiChannel() {
	// Create a 5.1 surround sound source (6 channels)
	source := audiotest.NewConstantSource(48000, 6, 48000, 0.5)

	fmt.Printf("Input: %d channels\n", source.Channels())

	// Mix to mono
	mono := audio.NewMonoMixer(source)

	fmt.Printf("Output: %d channel (mono)\n", mono.Channels())
	fmt.Println("All channels are averaged together")

	// Read a sample to verify
	buf := make([]float32, 1)
	n, _ := mono.ReadSamples(buf)
	if n > 0 {
		fmt.Printf("Output sample value: %.1f (average of 6 × 0.5)\n", buf[0])
	}
	// Output:
	// Input: 6 channels
	// Output: 1 channel (mono)
	// All channels are averaged together
	// Output sample value: 0.5 (average of 6 × 0.5)
}

// Example_errorHandling shows proper error handling in audio processing.
func Example_errorHandling() {
	source := audiotest.NewSineSource(16000, 1, 1000, 440.0) // Short audio

	buf := make([]float32, 4096)
	totalSamples := 0

	for {
		n, err := source.ReadSamples(buf)

		// Always process available samples first
		if n > 0 {
			totalSamples += n
			// Process buf[0:n] here
		}

		// Then handle errors
		if err == io.EOF {
			// Normal end of stream
			fmt.Println("Reached end of audio stream")
			break
		}
		if err != nil {
			// Other errors
			fmt.Printf("Error reading samples: %v\n", err)
			break
		}
	}

	fmt.Printf("Successfully processed %d samples\n", totalSamples)
	// Output:
	// Reached end of audio stream
	// Successfully processed 1000 samples
}
