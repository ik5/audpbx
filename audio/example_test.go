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

// Example_channelLayout demonstrates working with channel layouts and metadata.
func Example_channelLayout() {
	// Understanding channel layouts
	layout := audio.Layout5Point1
	
	fmt.Printf("Layout: %s\n", layout)
	fmt.Printf("Channel count: %d\n", layout.ChannelCount())
	
	// Check if layout contains specific channels
	hasLFE := layout.Contains(audio.ChannelLowFrequency)
	fmt.Printf("Has subwoofer: %t\n", hasLFE)
	
	// Find position of a channel in interleaved data
	flIndex := layout.Index(audio.ChannelFrontLeft)
	frIndex := layout.Index(audio.ChannelFrontRight)
	fmt.Printf("FL index: %d\n", flIndex)
	fmt.Printf("FR index: %d\n", frIndex)
	
	// Output:
	// Layout: FL|FR|FC|LFE|BL|BR
	// Channel count: 6
	// Has subwoofer: true
	// FL index: 0
	// FR index: 1
}

// Example_splitChannels demonstrates splitting all channels from multi-channel audio.
func Example_splitChannels() {
	// Create a stereo source
	source := audiotest.NewSineSource(44100, 2, 44100, 440.0)
	
	// Split into individual mono channels
	channels, layout, err := audio.SplitChannels(source, audio.LayoutStereo)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Layout: %s\n", layout)
	fmt.Printf("Split into %d channels\n", len(channels))
	
	// Each channel is independent
	fmt.Printf("Channel 0: %s (%d Hz, %d channels)\n", 
		channels[0].Channel(), channels[0].SampleRate(), channels[0].Channels())
	fmt.Printf("Channel 1: %s (%d Hz, %d channels)\n",
		channels[1].Channel(), channels[1].SampleRate(), channels[1].Channels())
	
	// Read from left channel
	buf := make([]float32, 100)
	n, _ := channels[0].ReadSamples(buf)
	fmt.Printf("Read %d samples from left channel\n", n)
	
	// Output:
	// Layout: FL|FR
	// Split into 2 channels
	// Channel 0: FL (44100 Hz, 1 channels)
	// Channel 1: FR (44100 Hz, 1 channels)
	// Read 100 samples from left channel
}

// Example_extractChannels demonstrates extracting only specific channels.
func Example_extractChannels() {
	// Create a 5.1 surround source
	source := audiotest.NewConstantSource(48000, 6, 48000, 0.5)
	
	// Extract only front left, front right, and subwoofer
	channels, err := audio.ExtractChannels(source, audio.Layout5Point1,
		audio.ChannelFrontLeft,
		audio.ChannelFrontRight,
		audio.ChannelLowFrequency)
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// You get exactly 3 channels in the order specified
	fmt.Printf("Extracted %d channels\n", len(channels))
	fmt.Printf("Channel 0: %s\n", channels[0].Channel())
	fmt.Printf("Channel 1: %s\n", channels[1].Channel())
	fmt.Printf("Channel 2: %s\n", channels[2].Channel())
	
	// Read from each channel
	buf := make([]float32, 100)
	
	n1, _ := channels[0].ReadSamples(buf) // Front Left
	n2, _ := channels[1].ReadSamples(buf) // Front Right
	n3, _ := channels[2].ReadSamples(buf) // Subwoofer
	
	fmt.Printf("Read %d + %d + %d samples\n", n1, n2, n3)
	
	// Output:
	// Extracted 3 channels
	// Channel 0: FL
	// Channel 1: FR
	// Channel 2: LFE
	// Read 100 + 100 + 100 samples
}

// Example_extractSingleChannel demonstrates extracting just one channel.
func Example_extractSingleChannel() {
	// Create a 5.1 source
	source := audiotest.NewSineSource(48000, 6, 48000, 100.0)
	
	// Extract only the subwoofer (LFE) channel
	channels, err := audio.ExtractChannels(source, audio.Layout5Point1,
		audio.ChannelLowFrequency)
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	fmt.Printf("Extracted %d channel\n", len(channels))
	fmt.Printf("Channel: %s\n", channels[0].Channel())
	
	// Process only the subwoofer
	buf := make([]float32, 1024)
	total := 0
	
	for total < 48000 {
		n, err := channels[0].ReadSamples(buf)
		total += n
		if err == io.EOF {
			break
		}
	}
	
	fmt.Printf("Processed %d LFE samples\n", total)
	
	// Output:
	// Extracted 1 channel
	// Channel: LFE
	// Processed 48000 LFE samples
}

// Example_channelIdentification shows how to identify which channel is which.
func Example_channelIdentification() {
	source := audiotest.NewSilentSource(48000, 6, 1000)
	
	// Extract channels in custom order
	channels, err := audio.ExtractChannels(source, audio.Layout5Point1,
		audio.ChannelBackRight,
		audio.ChannelFrontLeft,
		audio.ChannelFrontCenter)
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// Method 1: Access by index (in the order you specified)
	br := &channels[0] // Back Right (first requested)
	fl := &channels[1] // Front Left (second requested)
	fc := &channels[2] // Front Center (third requested)
	
	fmt.Printf("By index: BR=%s, FL=%s, FC=%s\n", 
		br.Channel(), fl.Channel(), fc.Channel())
	
	// Method 2: Create a map for named access
	chMap := make(map[audio.Channel]*audio.ChannelSource)
	for i := range channels {
		chMap[channels[i].Channel()] = &channels[i]
	}
	
	fmt.Printf("By name: FL=%s, FC=%s, BR=%s\n",
		chMap[audio.ChannelFrontLeft].Channel(),
		chMap[audio.ChannelFrontCenter].Channel(),
		chMap[audio.ChannelBackRight].Channel())
	
	// Output:
	// By index: BR=BR, FL=FL, FC=FC
	// By name: FL=FL, FC=FC, BR=BR
}

// Example_channelResampling shows resampling individual channels independently.
func Example_channelResampling() {
	// Create a stereo source at 44.1kHz
	source := audiotest.NewSineSource(44100, 2, 44100, 440.0)
	
	// Extract both channels
	channels, err := audio.ExtractChannels(source, audio.LayoutStereo,
		audio.ChannelFrontLeft,
		audio.ChannelFrontRight)
	
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	
	// Resample each channel independently to 16kHz
	leftResampled := audio.NewResampler(&channels[0], 16000)
	rightResampled := audio.NewResampler(&channels[1], 16000)
	
	fmt.Printf("Original: %d Hz, 2 channels\n", source.SampleRate())
	fmt.Printf("Left resampled: %d Hz, %d channel\n", 
		leftResampled.SampleRate(), leftResampled.Channels())
	fmt.Printf("Right resampled: %d Hz, %d channel\n",
		rightResampled.SampleRate(), rightResampled.Channels())
	
	// Read resampled data
	buf := make([]float32, 1000)
	nl, _ := leftResampled.ReadSamples(buf)
	
	fmt.Printf("Read %d samples from each channel\n", nl)
	
	// Output:
	// Original: 44100 Hz, 2 channels
	// Left resampled: 16000 Hz, 1 channel
	// Right resampled: 16000 Hz, 1 channel
	// Read 1000 samples from each channel
}
