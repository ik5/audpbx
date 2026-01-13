// SPDX-License-Identifier: EPL-2.0

package wav

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"
)

// Helper function to create a minimal valid WAV file
func createWAVFile(sampleRate, channels, bitsPerSample int, samples []int16) []byte {
	buf := new(bytes.Buffer)

	numChannels := uint16(channels)
	bits := uint16(bitsPerSample)
	byteRate := uint32(sampleRate) * uint32(numChannels) * uint32(bits/8)
	blockAlign := uint16(numChannels) * uint16(bits/8)
	dataSize := uint32(len(samples) * 2)
	riffSize := 36 + dataSize

	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, riffSize)
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16)) // chunk size
	binary.Write(buf, binary.LittleEndian, uint16(1))  // PCM format
	binary.Write(buf, binary.LittleEndian, numChannels)
	binary.Write(buf, binary.LittleEndian, uint32(sampleRate))
	binary.Write(buf, binary.LittleEndian, byteRate)
	binary.Write(buf, binary.LittleEndian, blockAlign)
	binary.Write(buf, binary.LittleEndian, bits)

	// data chunk
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, dataSize)

	// Write samples
	for _, s := range samples {
		binary.Write(buf, binary.LittleEndian, s)
	}

	return buf.Bytes()
}

func TestDecoder_ValidWAVFile(t *testing.T) {
	t.Parallel()

	samples := []int16{0, 100, 200, -100, -200, 0}
	wavData := createWAVFile(8000, 1, 16, samples)

	decoder := Decoder{}
	src, err := decoder.Decode(bytes.NewReader(wavData))

	if err != nil {
		t.Fatalf("Decode() error = %v, want nil", err)
	}

	if src == nil {
		t.Fatal("Decode() returned nil source")
	}

	if src.SampleRate() != 8000 {
		t.Errorf("SampleRate() = %d, want 8000", src.SampleRate())
	}

	if src.Channels() != 1 {
		t.Errorf("Channels() = %d, want 1", src.Channels())
	}
}

func TestDecoder_StereoWAVFile(t *testing.T) {
	t.Parallel()

	samples := []int16{100, 200, 300, 400, 500, 600}
	wavData := createWAVFile(44100, 2, 16, samples)

	decoder := Decoder{}
	src, err := decoder.Decode(bytes.NewReader(wavData))

	if err != nil {
		t.Fatalf("Decode() error = %v, want nil", err)
	}

	if src.SampleRate() != 44100 {
		t.Errorf("SampleRate() = %d, want 44100", src.SampleRate())
	}

	if src.Channels() != 2 {
		t.Errorf("Channels() = %d, want 2", src.Channels())
	}
}

func TestDecoder_NotWAVFile(t *testing.T) {
	t.Parallel()

	// Invalid RIFF header
	invalidData := []byte("NOT A WAV FILE DATA")

	decoder := Decoder{}
	_, err := decoder.Decode(bytes.NewReader(invalidData))

	if err != ErrNotWavFile {
		t.Errorf("Decode() error = %v, want ErrNotWavFile", err)
	}
}

func TestDecoder_InvalidWAVEMarker(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(36))
	buf.WriteString("NOPE") // Invalid WAVE marker

	decoder := Decoder{}
	_, err := decoder.Decode(buf)

	if err != ErrNotWavFile {
		t.Errorf("Decode() error = %v, want ErrNotWavFile", err)
	}
}

func TestDecoder_TruncatedHeader(t *testing.T) {
	t.Parallel()

	// Only 5 bytes (less than 12 needed for RIFF header)
	truncatedData := []byte("RIFF\x00")

	decoder := Decoder{}
	_, err := decoder.Decode(bytes.NewReader(truncatedData))

	if err == nil {
		t.Error("Decode() error = nil, want error for truncated header")
	}
}

func TestDecoder_Non16BitPCM(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(36))
	buf.WriteString("WAVE")

	// fmt chunk with 8-bit PCM
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))
	binary.Write(buf, binary.LittleEndian, uint16(1))  // PCM
	binary.Write(buf, binary.LittleEndian, uint16(1))  // mono
	binary.Write(buf, binary.LittleEndian, uint32(8000))
	binary.Write(buf, binary.LittleEndian, uint32(8000))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint16(8)) // 8-bit

	// data chunk
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, uint32(0))

	decoder := Decoder{}
	_, err := decoder.Decode(buf)

	if err != ErrOnlyPCM16bitSupported {
		t.Errorf("Decode() error = %v, want ErrOnlyPCM16bitSupported", err)
	}
}

func TestDecoder_NonPCMFormat(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(36))
	buf.WriteString("WAVE")

	// fmt chunk with non-PCM format
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))
	binary.Write(buf, binary.LittleEndian, uint16(3))  // IEEE Float (not PCM)
	binary.Write(buf, binary.LittleEndian, uint16(1))  // mono
	binary.Write(buf, binary.LittleEndian, uint32(8000))
	binary.Write(buf, binary.LittleEndian, uint32(16000))
	binary.Write(buf, binary.LittleEndian, uint16(2))
	binary.Write(buf, binary.LittleEndian, uint16(16))

	decoder := Decoder{}
	_, err := decoder.Decode(buf)

	if err == nil {
		t.Error("Decode() error = nil, want error for non-PCM format")
	}
}

func TestDecoder_WithUnknownChunks(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(60))
	buf.WriteString("WAVE")

	// Custom chunk (should be skipped)
	buf.WriteString("INFO")
	binary.Write(buf, binary.LittleEndian, uint32(4))
	buf.Write([]byte{0, 0, 0, 0})

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint32(8000))
	binary.Write(buf, binary.LittleEndian, uint32(16000))
	binary.Write(buf, binary.LittleEndian, uint16(2))
	binary.Write(buf, binary.LittleEndian, uint16(16))

	// data chunk
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, uint32(4))
	binary.Write(buf, binary.LittleEndian, int16(100))
	binary.Write(buf, binary.LittleEndian, int16(200))

	decoder := Decoder{}
	src, err := decoder.Decode(buf)

	if err != nil {
		t.Fatalf("Decode() error = %v, want nil (should skip unknown chunks)", err)
	}

	if src == nil {
		t.Fatal("Decode() returned nil source")
	}
}

func TestDecoder_OddSizedChunkPadding(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	buf.WriteString("RIFF")
	binary.Write(buf, binary.LittleEndian, uint32(61))
	buf.WriteString("WAVE")

	// Odd-sized custom chunk
	buf.WriteString("INFO")
	binary.Write(buf, binary.LittleEndian, uint32(3))
	buf.Write([]byte{0, 0, 0})
	buf.WriteByte(0) // Padding byte

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(buf, binary.LittleEndian, uint32(16))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint16(1))
	binary.Write(buf, binary.LittleEndian, uint32(8000))
	binary.Write(buf, binary.LittleEndian, uint32(16000))
	binary.Write(buf, binary.LittleEndian, uint16(2))
	binary.Write(buf, binary.LittleEndian, uint16(16))

	// data chunk
	buf.WriteString("data")
	binary.Write(buf, binary.LittleEndian, uint32(4))
	binary.Write(buf, binary.LittleEndian, int16(100))
	binary.Write(buf, binary.LittleEndian, int16(200))

	decoder := Decoder{}
	src, err := decoder.Decode(buf)

	if err != nil {
		t.Fatalf("Decode() error = %v, want nil", err)
	}

	if src == nil {
		t.Fatal("Decode() returned nil source")
	}
}

func TestSource_ReadSamples(t *testing.T) {
	t.Parallel()

	samples := []int16{0, 16384, 32767, -16384, -32768}
	wavData := createWAVFile(8000, 1, 16, samples)

	decoder := Decoder{}
	src, err := decoder.Decode(bytes.NewReader(wavData))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	dst := make([]float32, 5)
	n, err := src.ReadSamples(dst)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != 5 {
		t.Errorf("ReadSamples() n = %d, want 5", n)
	}

	// Verify conversion from int16 to float32
	expected := []float32{0.0, 0.5, 1.0, -0.5, -1.0}
	for i := range n {
		if math.Abs(float64(dst[i]-expected[i])) > 0.01 {
			t.Errorf("dst[%d] = %v, want ≈%v", i, dst[i], expected[i])
		}
	}
}

func TestSource_ReadSamples_EmptyBuffer(t *testing.T) {
	t.Parallel()

	samples := []int16{100, 200, 300}
	wavData := createWAVFile(8000, 1, 16, samples)

	decoder := Decoder{}
	src, err := decoder.Decode(bytes.NewReader(wavData))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	dst := make([]float32, 0)
	n, err := src.ReadSamples(dst)

	if err != nil {
		t.Errorf("ReadSamples() with empty buffer error = %v, want nil", err)
	}

	if n != 0 {
		t.Errorf("ReadSamples() n = %d, want 0", n)
	}
}

func TestSource_ReadSamples_EOF(t *testing.T) {
	t.Parallel()

	samples := []int16{100, 200}
	wavData := createWAVFile(8000, 1, 16, samples)

	decoder := Decoder{}
	src, err := decoder.Decode(bytes.NewReader(wavData))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Read all samples
	dst := make([]float32, 2)
	n1, err1 := src.ReadSamples(dst)

	// Should get data, might get EOF on first or second read
	if err1 != nil && err1 != io.EOF {
		t.Errorf("ReadSamples() error = %v, want nil or io.EOF", err1)
	}

	if n1 != 2 {
		t.Errorf("ReadSamples() n = %d, want 2", n1)
	}

	// If we didn't get EOF yet, try to read more to get it
	if err1 != io.EOF {
		n2, err2 := src.ReadSamples(dst)
		if err2 != io.EOF {
			t.Errorf("Second ReadSamples() error = %v, want io.EOF", err2)
		}
		if n2 != 0 {
			t.Errorf("Second ReadSamples() n = %d, want 0", n2)
		}
	}

	// Subsequent reads should always return EOF with 0 samples
	n3, err3 := src.ReadSamples(dst)
	if err3 != io.EOF {
		t.Errorf("Final ReadSamples() error = %v, want io.EOF", err3)
	}
	if n3 != 0 {
		t.Errorf("Final ReadSamples() n = %d, want 0", n3)
	}
}

func TestSource_ReadSamples_PartialRead(t *testing.T) {
	t.Parallel()

	samples := []int16{100, 200, 300, 400, 500}
	wavData := createWAVFile(8000, 1, 16, samples)

	decoder := Decoder{}
	src, err := decoder.Decode(bytes.NewReader(wavData))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	// Read in chunks
	dst := make([]float32, 2)
	n1, err1 := src.ReadSamples(dst)

	if err1 != nil {
		t.Errorf("First ReadSamples() error = %v", err1)
	}

	if n1 != 2 {
		t.Errorf("First ReadSamples() n = %d, want 2", n1)
	}

	n2, err2 := src.ReadSamples(dst)

	if err2 != nil {
		t.Errorf("Second ReadSamples() error = %v", err2)
	}

	if n2 != 2 {
		t.Errorf("Second ReadSamples() n = %d, want 2", n2)
	}

	// Last sample
	dst3 := make([]float32, 2)
	n3, err3 := src.ReadSamples(dst3)

	if err3 != io.EOF {
		t.Errorf("Third ReadSamples() error = %v, want io.EOF", err3)
	}

	if n3 != 1 {
		t.Errorf("Third ReadSamples() n = %d, want 1", n3)
	}
}

func TestSource_BufSize(t *testing.T) {
	t.Parallel()

	samples := []int16{100, 200}
	wavData := createWAVFile(8000, 1, 16, samples)

	decoder := Decoder{}
	src, err := decoder.Decode(bytes.NewReader(wavData))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	bufSize := src.BufSize()
	if bufSize <= 0 {
		t.Errorf("BufSize() = %d, want positive value", bufSize)
	}
}

func TestSource_Close(t *testing.T) {
	t.Parallel()

	samples := []int16{100, 200}
	wavData := createWAVFile(8000, 1, 16, samples)

	decoder := Decoder{}
	src, err := decoder.Decode(bytes.NewReader(wavData))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	err = src.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestDecoder_VariousSampleRates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		sampleRate int
		channels   int
	}{
		{"8kHz Mono", 8000, 1},
		{"16kHz Mono", 16000, 1},
		{"22.05kHz Stereo", 22050, 2},
		{"44.1kHz Stereo", 44100, 2},
		{"48kHz Stereo", 48000, 2},
		{"96kHz Mono", 96000, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			samples := []int16{100, 200, 300}
			wavData := createWAVFile(tt.sampleRate, tt.channels, 16, samples)

			decoder := Decoder{}
			src, err := decoder.Decode(bytes.NewReader(wavData))

			if err != nil {
				t.Fatalf("Decode() error = %v", err)
			}

			if src.SampleRate() != tt.sampleRate {
				t.Errorf("SampleRate() = %d, want %d", src.SampleRate(), tt.sampleRate)
			}

			if src.Channels() != tt.channels {
				t.Errorf("Channels() = %d, want %d", src.Channels(), tt.channels)
			}
		})
	}
}

// BenchmarkDecoder_Decode benchmarks WAV file decoding
func BenchmarkDecoder_Decode(b *testing.B) {
	samples := make([]int16, 44100) // 1 second at 44.1kHz
	for i := range samples {
		samples[i] = int16(i % 1000)
	}
	wavData := createWAVFile(44100, 2, 16, samples)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		decoder := Decoder{}
		_, _ = decoder.Decode(bytes.NewReader(wavData))
	}
}

// BenchmarkSource_ReadSamples benchmarks reading samples
func BenchmarkSource_ReadSamples(b *testing.B) {
	samples := make([]int16, 44100*10) // 10 seconds
	for i := range samples {
		samples[i] = int16(i % 1000)
	}
	wavData := createWAVFile(44100, 2, 16, samples)

	decoder := Decoder{}
	src, _ := decoder.Decode(bytes.NewReader(wavData))
	dst := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = src.ReadSamples(dst)
	}
}

// BenchmarkSource_ReadSamples_SmallBuffer benchmarks with small buffers
func BenchmarkSource_ReadSamples_SmallBuffer(b *testing.B) {
	samples := make([]int16, 44100)
	wavData := createWAVFile(44100, 1, 16, samples)

	decoder := Decoder{}
	src, _ := decoder.Decode(bytes.NewReader(wavData))
	dst := make([]float32, 64)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = src.ReadSamples(dst)
	}
}

// BenchmarkSource_ReadSamples_LargeBuffer benchmarks with large buffers
func BenchmarkSource_ReadSamples_LargeBuffer(b *testing.B) {
	samples := make([]int16, 441000) // 10 seconds
	wavData := createWAVFile(44100, 1, 16, samples)

	decoder := Decoder{}
	src, _ := decoder.Decode(bytes.NewReader(wavData))
	dst := make([]float32, 16384)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		_, _ = src.ReadSamples(dst)
	}
}
