// SPDX-License-Identifier: EPL-2.0

package audio

import (
	"sync"
	"testing"
)

// TestExtractChannels_Selective tests extracting only specific channels
func TestExtractChannels_Selective(t *testing.T) {
	t.Parallel()

	// Create a 5.1 source where each channel has a unique value
	src := newMockSource(48000, 6, 100, func(sample int, channel int) float32 {
		return float32(channel) * 0.1 // ch0=0.0, ch1=0.1, ch2=0.2, etc.
	})

	// Extract only FL, FR, and BR
	channels, err := ExtractChannels(src, Layout5Point1,
		ChannelFrontLeft,
		ChannelFrontRight,
		ChannelBackRight)

	if err != nil {
		t.Fatalf("ExtractChannels failed: %v", err)
	}

	// Should have exactly 3 channels
	if len(channels) != 3 {
		t.Fatalf("got %d channels, want 3", len(channels))
	}

	// Verify channel identities in order
	if channels[0].Channel() != ChannelFrontLeft {
		t.Errorf("channels[0] = %s, want FL", channels[0].Channel())
	}
	if channels[1].Channel() != ChannelFrontRight {
		t.Errorf("channels[1] = %s, want FR", channels[1].Channel())
	}
	if channels[2].Channel() != ChannelBackRight {
		t.Errorf("channels[2] = %s, want BR", channels[2].Channel())
	}

	// Read samples and verify values
	buf := make([]float32, 10)

	// FL should have value 0.0 (channel index 0)
	n, _ := channels[0].ReadSamples(buf)
	if n > 0 && buf[0] != 0.0 {
		t.Errorf("FL first sample = %v, want 0.0", buf[0])
	}

	// FR should have value 0.1 (channel index 1)
	n, _ = channels[1].ReadSamples(buf)
	if n > 0 && buf[0] != 0.1 {
		t.Errorf("FR first sample = %v, want 0.1", buf[0])
	}

	// BR should have value 0.5 (channel index 5)
	n, _ = channels[2].ReadSamples(buf)
	if n > 0 && buf[0] != 0.5 {
		t.Errorf("BR first sample = %v, want 0.5", buf[0])
	}
}

// TestExtractChannels_CustomOrder tests extracting channels in custom order
func TestExtractChannels_CustomOrder(t *testing.T) {
	t.Parallel()

	src := newSilentSource(44100, 2, 100)

	// Extract both channels in custom order (FR first, then FL)
	channels, err := ExtractChannels(src, LayoutStereo,
		ChannelFrontRight,
		ChannelFrontLeft)

	if err != nil {
		t.Fatalf("ExtractChannels failed: %v", err)
	}

	if len(channels) != 2 {
		t.Fatalf("got %d channels, want 2", len(channels))
	}

	// Order should match the order we specified
	if channels[0].Channel() != ChannelFrontRight {
		t.Errorf("channels[0] = %s, want FR", channels[0].Channel())
	}
	if channels[1].Channel() != ChannelFrontLeft {
		t.Errorf("channels[1] = %s, want FL", channels[1].Channel())
	}
}

// TestExtractChannels_InvalidChannel tests error handling
func TestExtractChannels_InvalidChannel(t *testing.T) {
	t.Parallel()

	src := newSilentSource(44100, 2, 100)

	// Try to extract a channel that's not in stereo layout
	_, err := ExtractChannels(src, LayoutStereo, ChannelBackLeft)

	if err == nil {
		t.Error("expected error for channel not in layout, got nil")
	}
}

// TestExtractChannels_NoChannels tests error handling for empty wanted list
func TestExtractChannels_NoChannels(t *testing.T) {
	t.Parallel()

	src := newSilentSource(44100, 2, 100)

	_, err := ExtractChannels(src, LayoutStereo)

	if err == nil {
		t.Error("expected error for no channels specified, got nil")
	}
}

// TestExtractChannels_SingleChannel tests extracting just one channel
func TestExtractChannels_SingleChannel(t *testing.T) {
	t.Parallel()

	src := newSilentSource(48000, 6, 100)

	// Extract only the subwoofer
	channels, err := ExtractChannels(src, Layout5Point1, ChannelLowFrequency)

	if err != nil {
		t.Fatalf("ExtractChannels failed: %v", err)
	}

	if len(channels) != 1 {
		t.Fatalf("got %d channels, want 1", len(channels))
	}

	if channels[0].Channel() != ChannelLowFrequency {
		t.Errorf("channels[0] = %s, want LFE", channels[0].Channel())
	}

	// Verify it reads correctly
	buf := make([]float32, 10)
	n, _ := channels[0].ReadSamples(buf)
	if n == 0 {
		t.Error("failed to read from LFE channel")
	}
}

// TestExtractChannels_MemoryEfficiency verifies only requested channels buffer data
func TestExtractChannels_MemoryEfficiency(t *testing.T) {
	t.Parallel()

	// Create a 5.1 source
	src := newSilentSource(48000, 6, 10000)

	// Extract only 2 out of 6 channels
	channels, err := ExtractChannels(src, Layout5Point1,
		ChannelFrontLeft,
		ChannelFrontRight)

	if err != nil {
		t.Fatalf("ExtractChannels failed: %v", err)
	}

	// Read a lot of data from both channels
	buf := make([]float32, 5000)
	for i := 0; i < 2; i++ {
		channels[0].ReadSamples(buf)
		channels[1].ReadSamples(buf)
	}

	// The key test: we should only have 2 channel sources, not 6
	if len(channels) != 2 {
		t.Errorf("got %d channel sources, want 2 (memory efficiency check)", len(channels))
	}

	// The splitter should only have 2 outputs
	if channels[0].splitter.nOut != 2 {
		t.Errorf("splitter has %d outputs, want 2", channels[0].splitter.nOut)
	}
}

// Benchmarks for ExtractChannels

func BenchmarkExtractChannels_Stereo(b *testing.B) {
	const frames = 4096
	buf := make([]float32, frames)

	b.ReportAllocs()
	for b.Loop() {
		src := newSilentSource(44100, 2, frames)
		channels, err := ExtractChannels(src, LayoutStereo,
			ChannelFrontLeft, ChannelFrontRight)
		if err != nil {
			b.Fatal(err)
		}
		channels[0].ReadSamples(buf)
		channels[1].ReadSamples(buf)
	}
}

func BenchmarkExtractChannels_5Point1_Selective(b *testing.B) {
	const frames = 4096
	buf := make([]float32, frames)

	b.ReportAllocs()
	for b.Loop() {
		src := newSilentSource(48000, 6, frames)
		channels, err := ExtractChannels(src, Layout5Point1,
			ChannelFrontLeft, ChannelFrontRight, ChannelBackRight)
		if err != nil {
			b.Fatal(err)
		}
		for i := range channels {
			channels[i].ReadSamples(buf)
		}
	}
}

func BenchmarkExtractChannels_5Point1_Single(b *testing.B) {
	const frames = 4096
	buf := make([]float32, frames)

	b.ReportAllocs()
	for b.Loop() {
		src := newSilentSource(48000, 6, frames)
		channels, err := ExtractChannels(src, Layout5Point1,
			ChannelLowFrequency)
		if err != nil {
			b.Fatal(err)
		}
		channels[0].ReadSamples(buf)
	}
}

// Benchmark comparing ExtractChannels vs SplitChannels for selective reading
func BenchmarkExtractChannels_vs_SplitChannels_Selective(b *testing.B) {
	const frames = 4096
	buf := make([]float32, frames)

	b.Run("ExtractChannels", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			src := newSilentSource(48000, 6, frames)
			channels, err := ExtractChannels(src, Layout5Point1,
				ChannelFrontLeft, ChannelFrontRight)
			if err != nil {
				b.Fatal(err)
			}
			channels[0].ReadSamples(buf)
			channels[1].ReadSamples(buf)
		}
	})

	b.Run("SplitChannels", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			src := newSilentSource(48000, 6, frames)
			channels, _, err := SplitChannels(src, Layout5Point1)
			if err != nil {
				b.Fatal(err)
			}
			// Close unused channels
			for i := 2; i < len(channels); i++ {
				channels[i].Close()
			}
			channels[0].ReadSamples(buf)
			channels[1].ReadSamples(buf)
		}
	})
}

// Benchmark steady-state reading (reuse extracted channels)
func BenchmarkExtractChannels_ReuseChannels_Stereo(b *testing.B) {
	const frames = 4096
	src := newSilentSource(44100, 2, frames*b.N)
	channels, err := ExtractChannels(src, LayoutStereo,
		ChannelFrontLeft, ChannelFrontRight)
	if err != nil {
		b.Fatal(err)
	}

	buf := make([]float32, frames)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		channels[0].ReadSamples(buf)
		channels[1].ReadSamples(buf)
	}
}

func BenchmarkExtractChannels_ReuseChannels_5Point1_Selective(b *testing.B) {
	const frames = 4096
	src := newSilentSource(48000, 6, frames*b.N)
	channels, err := ExtractChannels(src, Layout5Point1,
		ChannelFrontLeft, ChannelFrontRight, ChannelBackRight)
	if err != nil {
		b.Fatal(err)
	}

	buf := make([]float32, frames)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		for j := range channels {
			channels[j].ReadSamples(buf)
		}
	}
}

// Benchmark memory efficiency - selective vs all channels
func BenchmarkExtractChannels_MemoryUsage(b *testing.B) {
	const frames = 4096
	buf := make([]float32, frames)

	b.Run("Extract_2_of_6", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			src := newSilentSource(48000, 6, frames*10)
			channels, err := ExtractChannels(src, Layout5Point1,
				ChannelFrontLeft, ChannelFrontRight)
			if err != nil {
				b.Fatal(err)
			}
			// Read multiple times to see ring buffer behavior
			for range 10 {
				channels[0].ReadSamples(buf)
				channels[1].ReadSamples(buf)
			}
		}
	})

	b.Run("Split_All_6_Read_2", func(b *testing.B) {
		b.ReportAllocs()
		for b.Loop() {
			src := newSilentSource(48000, 6, frames*10)
			channels, _, err := SplitChannels(src, Layout5Point1)
			if err != nil {
				b.Fatal(err)
			}
			// Close unused
			for i := 2; i < 6; i++ {
				channels[i].Close()
			}
			// Read multiple times
			for range 10 {
				channels[0].ReadSamples(buf)
				channels[1].ReadSamples(buf)
			}
		}
	})
}

// Benchmark concurrent reading from extracted channels
func BenchmarkExtractChannels_ConcurrentRead(b *testing.B) {
	const frames = 4096

	b.ReportAllocs()
	for b.Loop() {
		src := newSilentSource(44100, 2, frames)
		channels, err := ExtractChannels(src, LayoutStereo,
			ChannelFrontLeft, ChannelFrontRight)
		if err != nil {
			b.Fatal(err)
		}

		var wg sync.WaitGroup
		wg.Add(2)
		for i := range 2 {
			go func(ch *ChannelSource) {
				defer wg.Done()
				buf := make([]float32, frames)
				ch.ReadSamples(buf)
			}(&channels[i])
		}
		wg.Wait()
	}
}
