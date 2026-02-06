// SPDX-License-Identifier: EPL-2.0

package audpbx_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ik5/audpbx"
	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/internal/audiotest"
)

func TestSplitToMonoSources(t *testing.T) {
	src := audiotest.NewSineSource(44100, 2, 44100, 440.0)

	sources, err := audpbx.SplitToMonoSources(src)
	if err != nil {
		t.Fatalf("SplitToMonoSources failed: %v", err)
	}

	if len(sources) != 2 {
		t.Errorf("got %d sources, want 2", len(sources))
	}

	for i, src := range sources {
		if src.Channels() != 1 {
			t.Errorf("source[%d] has %d channels, want 1", i, src.Channels())
		}
		if src.SampleRate() != 44100 {
			t.Errorf("source[%d] has rate %d, want 44100", i, src.SampleRate())
		}
	}
}

func TestProcessAndSaveChannel(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test.wav")

	mono := audiotest.NewSineSource(44100, 1, 44100, 440.0)

	err := audpbx.ProcessAndSaveChannel(mono, 16000, path)
	if err != nil {
		t.Fatalf("ProcessAndSaveChannel failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Errorf("output file was not created")
	}
}

func TestProcessAndSaveChannelWriter(t *testing.T) {
	mono := audiotest.NewSineSource(44100, 1, 44100, 440.0)

	var buf bytes.Buffer
	err := audpbx.ProcessAndSaveChannelWriter(mono, 16000, &buf)
	if err != nil {
		t.Fatalf("ProcessAndSaveChannelWriter failed: %v", err)
	}

	if buf.Len() == 0 {
		t.Errorf("no data written to buffer")
	}
}

func TestProcessChannels_SaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	src := audiotest.NewSilentSource(44100, 2, 44100)

	result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
		audpbx.WithSaveToFile(func(ch audio.Channel) string {
			return filepath.Join(tmpDir, fmt.Sprintf("channel_%s.wav", ch))
		}))

	if err != nil {
		t.Fatalf("ProcessChannels failed: %v", err)
	}

	if result.HasErrors() {
		t.Errorf("processing had errors: %v", result.Errors)
	}

	// Verify files exist
	flPath := filepath.Join(tmpDir, "channel_FL.wav")
	frPath := filepath.Join(tmpDir, "channel_FR.wav")

	if _, err := os.Stat(flPath); os.IsNotExist(err) {
		t.Errorf("FL file not created")
	}
	if _, err := os.Stat(frPath); os.IsNotExist(err) {
		t.Errorf("FR file not created")
	}
}

func TestProcessChannels_WithChannels(t *testing.T) {
	tmpDir := t.TempDir()
	src := audiotest.NewSilentSource(48000, 6, 48000)

	result, err := audpbx.ProcessChannels(src, audio.Layout5Point1,
		audpbx.WithChannels(audio.ChannelFrontLeft, audio.ChannelFrontRight),
		audpbx.WithSaveToFile(func(ch audio.Channel) string {
			return filepath.Join(tmpDir, fmt.Sprintf("%s.wav", ch))
		}))

	if err != nil {
		t.Fatalf("ProcessChannels failed: %v", err)
	}

	if result.HasErrors() {
		t.Errorf("processing had errors: %v", result.Errors)
	}

	// Only FL and FR should be created
	flPath := filepath.Join(tmpDir, "FL.wav")
	frPath := filepath.Join(tmpDir, "FR.wav")
	fcPath := filepath.Join(tmpDir, "FC.wav")

	if _, err := os.Stat(flPath); os.IsNotExist(err) {
		t.Errorf("FL file not created")
	}
	if _, err := os.Stat(frPath); os.IsNotExist(err) {
		t.Errorf("FR file not created")
	}
	if _, err := os.Stat(fcPath); !os.IsNotExist(err) {
		t.Errorf("FC file should not be created")
	}
}

func TestProcessChannels_WithResample(t *testing.T) {
	tmpDir := t.TempDir()
	src := audiotest.NewSilentSource(44100, 2, 44100)

	result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
		audpbx.WithResample(func(ch audio.Channel) int {
			if ch == audio.ChannelFrontLeft {
				return 16000
			}
			return 8000
		}),
		audpbx.WithSaveToFile(func(ch audio.Channel) string {
			return filepath.Join(tmpDir, fmt.Sprintf("%s.wav", ch))
		}))

	if err != nil {
		t.Fatalf("ProcessChannels failed: %v", err)
	}

	if result.HasErrors() {
		t.Errorf("processing had errors: %v", result.Errors)
	}
}

func TestProcessChannels_WithGain(t *testing.T) {
	tmpDir := t.TempDir()
	src := audiotest.NewConstantSource(44100, 2, 44100, 0.5)

	result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
		audpbx.WithGain(func(ch audio.Channel) float32 {
			if ch == audio.ChannelFrontLeft {
				return 2.0 // Double left channel
			}
			return 1.0
		}),
		audpbx.WithSaveToFile(func(ch audio.Channel) string {
			return filepath.Join(tmpDir, fmt.Sprintf("%s.wav", ch))
		}))

	if err != nil {
		t.Fatalf("ProcessChannels failed: %v", err)
	}

	if result.HasErrors() {
		t.Errorf("processing had errors: %v", result.Errors)
	}
}

func TestProcessChannels_WithMixdown(t *testing.T) {
	tmpDir := t.TempDir()
	src := audiotest.NewSilentSource(44100, 2, 44100)
	mixPath := filepath.Join(tmpDir, "mixed.wav")

	result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
		audpbx.WithMixdown(mixPath))

	if err != nil {
		t.Fatalf("ProcessChannels failed: %v", err)
	}

	if result.HasErrors() {
		t.Errorf("processing had errors: %v", result.Errors)
	}

	if _, err := os.Stat(mixPath); os.IsNotExist(err) {
		t.Errorf("mixdown file not created")
	}
}

func TestProcessChannels_WithConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	src := audiotest.NewSilentSource(48000, 6, 48000)

	result, err := audpbx.ProcessChannels(src, audio.Layout5Point1,
		audpbx.WithConcurrency(3),
		audpbx.WithSaveToFile(func(ch audio.Channel) string {
			return filepath.Join(tmpDir, fmt.Sprintf("%s.wav", ch))
		}))

	if err != nil {
		t.Fatalf("ProcessChannels failed: %v", err)
	}

	if result.HasErrors() {
		t.Errorf("processing had errors: %v", result.Errors)
	}
}

func TestProcessChannels_NoOutputError(t *testing.T) {
	src := audiotest.NewSilentSource(44100, 2, 44100)

	_, err := audpbx.ProcessChannels(src, audio.LayoutStereo)

	if err == nil {
		t.Error("expected error when no output configured")
	}
}

func TestProcessChannels_BufferSizeValidation(t *testing.T) {
	tmpDir := t.TempDir()
	src := audiotest.NewSilentSource(44100, 2, 44100)

	// Too small
	_, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
		audpbx.WithBufferSize(100),
		audpbx.WithSaveToFile(func(ch audio.Channel) string {
			return filepath.Join(tmpDir, fmt.Sprintf("%s.wav", ch))
		}))

	if err == nil {
		t.Error("expected error for too small buffer")
	}

	// Too large
	_, err = audpbx.ProcessChannels(src, audio.LayoutStereo,
		audpbx.WithBufferSize(2*1024*1024),
		audpbx.WithSaveToFile(func(ch audio.Channel) string {
			return filepath.Join(tmpDir, fmt.Sprintf("%s.wav", ch))
		}))

	if err == nil {
		t.Error("expected error for too large buffer")
	}
}

func TestProcessChannels_WithErrorHandler(t *testing.T) {
	tmpDir := t.TempDir()
	src := audiotest.NewSilentSource(44100, 2, 44100)

	errorCount := 0
	result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
		audpbx.WithErrorHandler(func(ch audio.Channel, activity string, err error) error {
			errorCount++
			return nil // Continue processing
		}),
		audpbx.WithSaveToFile(func(ch audio.Channel) string {
			if ch == audio.ChannelFrontLeft {
				return "/invalid/path/file.wav" // Will fail
			}
			return filepath.Join(tmpDir, fmt.Sprintf("%s.wav", ch))
		}))

	if err != nil {
		t.Fatalf("ProcessChannels failed: %v", err)
	}

	if errorCount == 0 {
		t.Error("error handler was not called")
	}

	if !result.HasErrors() {
		t.Error("expected errors in result")
	}
}

// Benchmarks

func BenchmarkSplitToMonoSources(b *testing.B) {
	src := audiotest.NewSineSource(44100, 2, 44100, 440.0)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sources, err := audpbx.SplitToMonoSources(src)
		if err != nil {
			b.Fatal(err)
		}
		_ = sources
	}
}

func BenchmarkProcessAndSaveChannel(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mono := audiotest.NewSineSource(44100, 1, 8000, 440.0)
		path := filepath.Join(tmpDir, fmt.Sprintf("test_%d.wav", i))

		err := audpbx.ProcessAndSaveChannel(mono, 16000, path)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessAndSaveChannelWriter(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mono := audiotest.NewSineSource(44100, 1, 8000, 440.0)
		var buf bytes.Buffer

		err := audpbx.ProcessAndSaveChannelWriter(mono, 16000, &buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkProcessChannels_Stereo(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		src := audiotest.NewSilentSource(44100, 2, 8000)

		result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
			audpbx.WithSaveToFile(func(ch audio.Channel) string {
				return filepath.Join(tmpDir, fmt.Sprintf("%d_%s.wav", i, ch))
			}))

		if err != nil {
			b.Fatal(err)
		}
		if result.HasErrors() {
			b.Fatalf("errors: %v", result.Errors)
		}
	}
}

func BenchmarkProcessChannels_5Point1(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		src := audiotest.NewSilentSource(48000, 6, 8000)

		result, err := audpbx.ProcessChannels(src, audio.Layout5Point1,
			audpbx.WithSaveToFile(func(ch audio.Channel) string {
				return filepath.Join(tmpDir, fmt.Sprintf("%d_%s.wav", i, ch))
			}))

		if err != nil {
			b.Fatal(err)
		}
		if result.HasErrors() {
			b.Fatalf("errors: %v", result.Errors)
		}
	}
}

func BenchmarkProcessChannels_WithResample(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		src := audiotest.NewSilentSource(44100, 2, 8000)

		result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
			audpbx.WithResample(func(ch audio.Channel) int {
				return 16000
			}),
			audpbx.WithSaveToFile(func(ch audio.Channel) string {
				return filepath.Join(tmpDir, fmt.Sprintf("%d_%s.wav", i, ch))
			}))

		if err != nil {
			b.Fatal(err)
		}
		if result.HasErrors() {
			b.Fatalf("errors: %v", result.Errors)
		}
	}
}

func BenchmarkProcessChannels_WithGain(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		src := audiotest.NewConstantSource(44100, 2, 8000, 0.5)

		result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
			audpbx.WithGain(func(ch audio.Channel) float32 {
				return 1.5
			}),
			audpbx.WithSaveToFile(func(ch audio.Channel) string {
				return filepath.Join(tmpDir, fmt.Sprintf("%d_%s.wav", i, ch))
			}))

		if err != nil {
			b.Fatal(err)
		}
		if result.HasErrors() {
			b.Fatalf("errors: %v", result.Errors)
		}
	}
}

func BenchmarkProcessChannels_Concurrent(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		src := audiotest.NewSilentSource(48000, 6, 8000)

		result, err := audpbx.ProcessChannels(src, audio.Layout5Point1,
			audpbx.WithConcurrency(4),
			audpbx.WithSaveToFile(func(ch audio.Channel) string {
				return filepath.Join(tmpDir, fmt.Sprintf("%d_%s.wav", i, ch))
			}))

		if err != nil {
			b.Fatal(err)
		}
		if result.HasErrors() {
			b.Fatalf("errors: %v", result.Errors)
		}
	}
}

func BenchmarkProcessChannels_SelectiveChannels(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		src := audiotest.NewSilentSource(48000, 6, 8000)

		result, err := audpbx.ProcessChannels(src, audio.Layout5Point1,
			audpbx.WithChannels(audio.ChannelFrontLeft, audio.ChannelFrontRight),
			audpbx.WithSaveToFile(func(ch audio.Channel) string {
				return filepath.Join(tmpDir, fmt.Sprintf("%d_%s.wav", i, ch))
			}))

		if err != nil {
			b.Fatal(err)
		}
		if result.HasErrors() {
			b.Fatalf("errors: %v", result.Errors)
		}
	}
}

func BenchmarkProcessChannels_ComplexPipeline(b *testing.B) {
	tmpDir := b.TempDir()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		src := audiotest.NewSineSource(48000, 6, 8000, 440.0)

		result, err := audpbx.ProcessChannels(src, audio.Layout5Point1,
			audpbx.WithChannels(audio.ChannelFrontLeft, audio.ChannelFrontRight),
			audpbx.WithResample(func(ch audio.Channel) int {
				return 44100
			}),
			audpbx.WithGain(func(ch audio.Channel) float32 {
				return 1.2
			}),
			audpbx.WithSaveToFile(func(ch audio.Channel) string {
				return filepath.Join(tmpDir, fmt.Sprintf("%d_%s.wav", i, ch))
			}))

		if err != nil {
			b.Fatal(err)
		}
		if result.HasErrors() {
			b.Fatalf("errors: %v", result.Errors)
		}
	}
}
