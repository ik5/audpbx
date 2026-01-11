package wav

import (
	"bytes"
	"encoding/binary"
	"io"
	"testing"
)

func TestWriteWAV16_ValidFile(t *testing.T) {
	t.Parallel()

	samples := []int16{0, 100, -100, 200, -200}
	buf := new(bytes.Buffer)

	err := WriteWAV16(buf, 8000, samples)
	if err != nil {
		t.Fatalf("WriteWAV16() error = %v, want nil", err)
	}

	// Verify RIFF header
	if buf.Len() < 44 {
		t.Fatalf("WAV file too small: %d bytes", buf.Len())
	}

	data := buf.Bytes()
	if string(data[0:4]) != "RIFF" {
		t.Errorf("RIFF marker = %q, want \"RIFF\"", string(data[0:4]))
	}

	if string(data[8:12]) != "WAVE" {
		t.Errorf("WAVE marker = %q, want \"WAVE\"", string(data[8:12]))
	}
}

func TestWriteWAV16_EmptySamples(t *testing.T) {
	t.Parallel()

	samples := []int16{}
	buf := new(bytes.Buffer)

	err := WriteWAV16(buf, 8000, samples)
	if err != nil {
		t.Fatalf("WriteWAV16() error = %v, want nil", err)
	}

	// Should still create valid WAV header
	if buf.Len() != 44 {
		t.Errorf("WAV file size = %d, want 44 (header only)", buf.Len())
	}
}

func TestWriteWAV16_SingleSample(t *testing.T) {
	t.Parallel()

	samples := []int16{12345}
	buf := new(bytes.Buffer)

	err := WriteWAV16(buf, 16000, samples)
	if err != nil {
		t.Fatalf("WriteWAV16() error = %v", err)
	}

	expectedSize := 44 + 2 // header + 1 int16 sample
	if buf.Len() != expectedSize {
		t.Errorf("WAV file size = %d, want %d", buf.Len(), expectedSize)
	}
}

func TestWriteWAV16_CorrectHeader(t *testing.T) {
	t.Parallel()

	samples := []int16{100, 200, 300, 400}
	buf := new(bytes.Buffer)

	err := WriteWAV16(buf, 44100, samples)
	if err != nil {
		t.Fatalf("WriteWAV16() error = %v", err)
	}

	data := buf.Bytes()

	// Verify fmt chunk marker
	if string(data[12:16]) != "fmt " {
		t.Errorf("fmt marker = %q, want \"fmt \"", string(data[12:16]))
	}

	// Verify fmt chunk size (should be 16 for PCM)
	fmtSize := binary.LittleEndian.Uint32(data[16:20])
	if fmtSize != 16 {
		t.Errorf("fmt chunk size = %d, want 16", fmtSize)
	}

	// Verify audio format (1 = PCM)
	audioFormat := binary.LittleEndian.Uint16(data[20:22])
	if audioFormat != 1 {
		t.Errorf("audio format = %d, want 1 (PCM)", audioFormat)
	}

	// Verify number of channels (1 = mono)
	numChannels := binary.LittleEndian.Uint16(data[22:24])
	if numChannels != 1 {
		t.Errorf("num channels = %d, want 1", numChannels)
	}

	// Verify sample rate
	sampleRate := binary.LittleEndian.Uint32(data[24:28])
	if sampleRate != 44100 {
		t.Errorf("sample rate = %d, want 44100", sampleRate)
	}

	// Verify bits per sample (should be 16)
	bitsPerSample := binary.LittleEndian.Uint16(data[34:36])
	if bitsPerSample != 16 {
		t.Errorf("bits per sample = %d, want 16", bitsPerSample)
	}

	// Verify data chunk marker
	if string(data[36:40]) != "data" {
		t.Errorf("data marker = %q, want \"data\"", string(data[36:40]))
	}

	// Verify data size
	dataSize := binary.LittleEndian.Uint32(data[40:44])
	expectedDataSize := uint32(len(samples) * 2)
	if dataSize != expectedDataSize {
		t.Errorf("data size = %d, want %d", dataSize, expectedDataSize)
	}
}

func TestWriteWAV16_SampleData(t *testing.T) {
	t.Parallel()

	samples := []int16{100, -200, 300, -400}
	buf := new(bytes.Buffer)

	err := WriteWAV16(buf, 8000, samples)
	if err != nil {
		t.Fatalf("WriteWAV16() error = %v", err)
	}

	data := buf.Bytes()

	// Sample data starts at byte 44
	for i, expected := range samples {
		offset := 44 + (i * 2)
		actual := int16(binary.LittleEndian.Uint16(data[offset : offset+2]))
		if actual != expected {
			t.Errorf("sample[%d] = %d, want %d", i, actual, expected)
		}
	}
}

func TestWriteWAV16_RoundTrip(t *testing.T) {
	t.Parallel()

	// Write samples to WAV
	originalSamples := []int16{0, 100, -100, 32767, -32768, 12345, -6789}
	buf := new(bytes.Buffer)

	err := WriteWAV16(buf, 16000, originalSamples)
	if err != nil {
		t.Fatalf("WriteWAV16() error = %v", err)
	}

	// Decode and read back
	decoder := Decoder{}
	src, err := decoder.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	if src.SampleRate() != 16000 {
		t.Errorf("SampleRate() = %d, want 16000", src.SampleRate())
	}

	if src.Channels() != 1 {
		t.Errorf("Channels() = %d, want 1", src.Channels())
	}

	// Read samples as float32
	dst := make([]float32, len(originalSamples))
	n, err := src.ReadSamples(dst)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != len(originalSamples) {
		t.Errorf("ReadSamples() n = %d, want %d", n, len(originalSamples))
	}

	// Verify samples match (allowing for float conversion rounding)
	const maxInt16 float32 = 32768.0
	for i, original := range originalSamples {
		expectedFloat := float32(original) / maxInt16
		diff := dst[i] - expectedFloat
		if diff < -0.0001 || diff > 0.0001 {
			t.Errorf("sample[%d] = %v, want ≈%v (original=%d)", i, dst[i], expectedFloat, original)
		}
	}
}

func TestWriteWAV16_VariousSampleRates(t *testing.T) {
	t.Parallel()

	sampleRates := []int{8000, 16000, 22050, 44100, 48000, 96000}

	for _, rate := range sampleRates {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			samples := []int16{100, 200, 300}
			buf := new(bytes.Buffer)

			err := WriteWAV16(buf, rate, samples)
			if err != nil {
				t.Fatalf("WriteWAV16(%d) error = %v", rate, err)
			}

			data := buf.Bytes()
			actualRate := binary.LittleEndian.Uint32(data[24:28])
			if actualRate != uint32(rate) {
				t.Errorf("sample rate in header = %d, want %d", actualRate, rate)
			}
		})
	}
}

func TestWriteWAV16_LargeFile(t *testing.T) {
	t.Parallel()

	// Create 10 seconds of audio at 44.1kHz
	numSamples := 44100 * 10
	samples := make([]int16, numSamples)
	for i := range samples {
		samples[i] = int16(i % 1000)
	}

	buf := new(bytes.Buffer)
	err := WriteWAV16(buf, 44100, samples)

	if err != nil {
		t.Fatalf("WriteWAV16() error = %v", err)
	}

	expectedSize := 44 + (numSamples * 2)
	if buf.Len() != expectedSize {
		t.Errorf("WAV file size = %d, want %d", buf.Len(), expectedSize)
	}
}

func TestWriteWAV16_ByteOrder(t *testing.T) {
	t.Parallel()

	// Test that multi-byte values are written in little-endian
	samples := []int16{0x1234}
	buf := new(bytes.Buffer)

	err := WriteWAV16(buf, 8000, samples)
	if err != nil {
		t.Fatalf("WriteWAV16() error = %v", err)
	}

	data := buf.Bytes()
	// Sample should be at byte 44, little-endian: 0x34, 0x12
	if data[44] != 0x34 || data[45] != 0x12 {
		t.Errorf("sample bytes = [%02x %02x], want [34 12]", data[44], data[45])
	}
}

func TestWriteWAV16_RIFFSize(t *testing.T) {
	t.Parallel()

	samples := []int16{100, 200, 300, 400}
	buf := new(bytes.Buffer)

	err := WriteWAV16(buf, 8000, samples)
	if err != nil {
		t.Fatalf("WriteWAV16() error = %v", err)
	}

	data := buf.Bytes()
	riffSize := binary.LittleEndian.Uint32(data[4:8])

	// RIFF size should be file size - 8 (for "RIFF" and size field)
	expectedRiffSize := uint32(buf.Len() - 8)
	if riffSize != expectedRiffSize {
		t.Errorf("RIFF size = %d, want %d", riffSize, expectedRiffSize)
	}
}

func TestWriteWAV16_BlockAlign(t *testing.T) {
	t.Parallel()

	samples := []int16{100}
	buf := new(bytes.Buffer)

	err := WriteWAV16(buf, 8000, samples)
	if err != nil {
		t.Fatalf("WriteWAV16() error = %v", err)
	}

	data := buf.Bytes()
	blockAlign := binary.LittleEndian.Uint16(data[32:34])

	// Block align = channels * (bitsPerSample / 8)
	// For mono 16-bit: 1 * 2 = 2
	if blockAlign != 2 {
		t.Errorf("block align = %d, want 2", blockAlign)
	}
}

func TestWriteWAV16_ByteRate(t *testing.T) {
	t.Parallel()

	sampleRate := 44100
	samples := []int16{100}
	buf := new(bytes.Buffer)

	err := WriteWAV16(buf, sampleRate, samples)
	if err != nil {
		t.Fatalf("WriteWAV16() error = %v", err)
	}

	data := buf.Bytes()
	byteRate := binary.LittleEndian.Uint32(data[28:32])

	// Byte rate = sample rate * channels * (bitsPerSample / 8)
	// For 44100 Hz mono 16-bit: 44100 * 1 * 2 = 88200
	expectedByteRate := uint32(sampleRate * 1 * 2)
	if byteRate != expectedByteRate {
		t.Errorf("byte rate = %d, want %d", byteRate, expectedByteRate)
	}
}

// BenchmarkWriteWAV16 benchmarks writing WAV files
func BenchmarkWriteWAV16(b *testing.B) {
	samples := make([]int16, 44100) // 1 second at 44.1kHz
	for i := range samples {
		samples[i] = int16(i % 1000)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buf := new(bytes.Buffer)
		_ = WriteWAV16(buf, 44100, samples)
	}
}

// BenchmarkWriteWAV16_SmallFile benchmarks small files
func BenchmarkWriteWAV16_SmallFile(b *testing.B) {
	samples := make([]int16, 1000)
	for i := range samples {
		samples[i] = int16(i)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buf := new(bytes.Buffer)
		_ = WriteWAV16(buf, 8000, samples)
	}
}

// BenchmarkWriteWAV16_LargeFile benchmarks large files
func BenchmarkWriteWAV16_LargeFile(b *testing.B) {
	samples := make([]int16, 441000) // 10 seconds at 44.1kHz
	for i := range samples {
		samples[i] = int16(i % 10000)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buf := new(bytes.Buffer)
		_ = WriteWAV16(buf, 44100, samples)
	}
}

// BenchmarkWriteWAV16_RoundTrip benchmarks write+decode
func BenchmarkWriteWAV16_RoundTrip(b *testing.B) {
	samples := make([]int16, 8000) // 1 second at 8kHz
	for i := range samples {
		samples[i] = int16(i % 1000)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		buf := new(bytes.Buffer)
		_ = WriteWAV16(buf, 8000, samples)

		decoder := Decoder{}
		_, _ = decoder.Decode(bytes.NewReader(buf.Bytes()))
	}
}
