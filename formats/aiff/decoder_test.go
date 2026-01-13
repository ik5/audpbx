// SPDX-License-Identifier: EPL-2.0

package aiff

import (
	"bytes"
	"errors"
	"io"
	"testing"

	goaudio "github.com/go-audio/audio"
)

// mockAiffReader simulates the aiff.Decoder for testing
type mockAiffReader struct {
	sampleRate   int
	channels     int
	bitDepth     int
	samples      []int
	offset       int
	returnErrors bool
}

func (m *mockAiffReader) Format() *goaudio.Format {
	return &goaudio.Format{
		SampleRate:  m.sampleRate,
		NumChannels: m.channels,
	}
}

func (m *mockAiffReader) PCMBuffer(buf *goaudio.IntBuffer) (int, error) {
	if m.returnErrors {
		return 0, io.ErrUnexpectedEOF
	}

	if m.offset >= len(m.samples) {
		return 0, io.EOF
	}

	samplesToRead := len(buf.Data)
	if samplesToRead > len(m.samples)-m.offset {
		samplesToRead = len(m.samples) - m.offset
	}

	copy(buf.Data, m.samples[m.offset:m.offset+samplesToRead])
	m.offset += samplesToRead

	if m.offset >= len(m.samples) {
		return samplesToRead, io.EOF
	}

	return samplesToRead, nil
}

func TestDecoder_InvalidInput(t *testing.T) {
	t.Parallel()

	// Invalid AIFF data
	invalidData := []byte("This is not AIFF data")

	decoder := Decoder{}
	_, err := decoder.Decode(bytes.NewReader(invalidData))

	if err == nil {
		t.Error("Decode() error = nil, want error for invalid data")
	}
}

func TestDecoder_EmptyInput(t *testing.T) {
	t.Parallel()

	decoder := Decoder{}
	_, err := decoder.Decode(bytes.NewReader([]byte{}))

	if err == nil {
		t.Error("Decode() error = nil, want error for empty input")
	}
}

func TestSource_Metadata(t *testing.T) {
	t.Parallel()

	// Create a mock source
	src := &source{
		dec: &mockAiffReader{
			sampleRate: 44100,
			channels:   2,
			bitDepth:   16,
			samples:    make([]int, 100),
		},
		sampleRate: 44100,
		channels:   2,
		bitDepth:   16,
	}

	if src.SampleRate() != 44100 {
		t.Errorf("SampleRate() = %d, want 44100", src.SampleRate())
	}

	if src.Channels() != 2 {
		t.Errorf("Channels() = %d, want 2", src.Channels())
	}
}

func TestSource_Close(t *testing.T) {
	t.Parallel()

	src := &source{
		dec: &mockAiffReader{
			sampleRate: 44100,
			channels:   2,
			bitDepth:   16,
			samples:    make([]int, 100),
		},
		sampleRate: 44100,
		channels:   2,
		bitDepth:   16,
	}

	err := src.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestSource_ReadSamples(t *testing.T) {
	t.Parallel()

	// Create test samples (16-bit range: -32768 to 32767)
	testSamples := []int{0, 16384, -16384, 32767, -32768}

	src := &source{
		dec: &mockAiffReader{
			sampleRate: 44100,
			channels:   1,
			bitDepth:   16,
			samples:    testSamples,
		},
		sampleRate: 44100,
		channels:   1,
		bitDepth:   16,
	}

	dst := make([]float32, len(testSamples))
	n, err := src.ReadSamples(dst)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v, want nil or EOF", err)
	}

	if n != len(testSamples) {
		t.Errorf("ReadSamples() n = %d, want %d", n, len(testSamples))
	}

	// Verify conversion (int to float32 normalized by 32768.0)
	expected := []float32{0.0, 0.5, -0.5, 0.999969482, -1.0}
	for i := range n {
		if dst[i] < expected[i]-0.001 || dst[i] > expected[i]+0.001 {
			t.Errorf("ReadSamples() dst[%d] = %f, want ~%f", i, dst[i], expected[i])
		}
	}
}

func TestSource_ReadSamples_EmptyBuffer(t *testing.T) {
	t.Parallel()

	src := &source{
		dec: &mockAiffReader{
			sampleRate: 44100,
			channels:   2,
			bitDepth:   16,
			samples:    make([]int, 100),
		},
		sampleRate: 44100,
		channels:   2,
		bitDepth:   16,
	}

	dst := make([]float32, 0)
	n, err := src.ReadSamples(dst)

	if err != nil {
		t.Errorf("ReadSamples() error = %v, want nil", err)
	}

	if n != 0 {
		t.Errorf("ReadSamples() n = %d, want 0", n)
	}
}

func TestSource_ReadSamples_EOF(t *testing.T) {
	t.Parallel()

	src := &source{
		dec: &mockAiffReader{
			sampleRate: 44100,
			channels:   1,
			bitDepth:   16,
			samples:    []int{100, 200},
		},
		sampleRate: 44100,
		channels:   1,
		bitDepth:   16,
	}

	// First read - get all samples
	dst := make([]float32, 2)
	n1, err1 := src.ReadSamples(dst)

	if err1 != io.EOF {
		t.Errorf("First ReadSamples() error = %v, want io.EOF", err1)
	}

	if n1 != 2 {
		t.Errorf("First ReadSamples() n = %d, want 2", n1)
	}

	// Second read - should get EOF with 0 samples
	n2, err2 := src.ReadSamples(dst)

	if err2 != io.EOF {
		t.Errorf("Second ReadSamples() error = %v, want io.EOF", err2)
	}

	if n2 != 0 {
		t.Errorf("Second ReadSamples() n = %d, want 0", n2)
	}
}

func TestSource_ReadSamples_PartialRead(t *testing.T) {
	t.Parallel()

	src := &source{
		dec: &mockAiffReader{
			sampleRate: 44100,
			channels:   1,
			bitDepth:   16,
			samples:    []int{100, 200, 300, 400, 500},
		},
		sampleRate: 44100,
		channels:   1,
		bitDepth:   16,
	}

	// Read 2 samples at a time
	dst := make([]float32, 2)

	// First read
	n1, err1 := src.ReadSamples(dst)
	if err1 != nil {
		t.Errorf("First ReadSamples() error = %v, want nil", err1)
	}
	if n1 != 2 {
		t.Errorf("First ReadSamples() n = %d, want 2", n1)
	}

	// Second read
	n2, err2 := src.ReadSamples(dst)
	if err2 != nil {
		t.Errorf("Second ReadSamples() error = %v, want nil", err2)
	}
	if n2 != 2 {
		t.Errorf("Second ReadSamples() n = %d, want 2", n2)
	}

	// Third read - partial (only 1 sample left)
	n3, err3 := src.ReadSamples(dst)
	if err3 != io.EOF {
		t.Errorf("Third ReadSamples() error = %v, want io.EOF", err3)
	}
	if n3 != 1 {
		t.Errorf("Third ReadSamples() n = %d, want 1", n3)
	}
}

func TestSource_ReadSamples_MultipleReads(t *testing.T) {
	t.Parallel()

	totalSamples := 1000
	samples := make([]int, totalSamples)
	for i := range samples {
		samples[i] = i * 10
	}

	src := &source{
		dec: &mockAiffReader{
			sampleRate: 44100,
			channels:   1,
			bitDepth:   16,
			samples:    samples,
		},
		sampleRate: 44100,
		channels:   1,
		bitDepth:   16,
	}

	dst := make([]float32, 256)
	totalRead := 0

	for {
		n, err := src.ReadSamples(dst)
		totalRead += n

		if err == io.EOF {
			break
		}

		if err != nil {
			t.Fatalf("ReadSamples() unexpected error: %v", err)
		}

		if n == 0 {
			t.Fatal("ReadSamples() returned 0 samples without EOF")
		}
	}

	if totalRead != totalSamples {
		t.Errorf("Total samples read = %d, want %d", totalRead, totalSamples)
	}
}

func TestSource_ReadSamples_Error(t *testing.T) {
	t.Parallel()

	src := &source{
		dec: &mockAiffReader{
			sampleRate:   44100,
			channels:     1,
			bitDepth:     16,
			samples:      []int{100, 200},
			returnErrors: true,
		},
		sampleRate: 44100,
		channels:   1,
		bitDepth:   16,
	}

	dst := make([]float32, 10)
	_, err := src.ReadSamples(dst)

	if err == nil {
		t.Error("ReadSamples() error = nil, want error")
	}
}

func TestSource_BufSize(t *testing.T) {
	t.Parallel()

	src := &source{
		dec: &mockAiffReader{
			sampleRate: 44100,
			channels:   2,
			bitDepth:   16,
			samples:    make([]int, 100),
		},
		sampleRate: 44100,
		channels:   2,
		bitDepth:   16,
	}

	// Initially no buffer
	bufSize := src.BufSize()
	if bufSize != 4096 {
		t.Errorf("BufSize() = %d, want 4096 (default)", bufSize)
	}

	// After reading, buffer should be allocated
	dst := make([]float32, 100)
	src.ReadSamples(dst)

	bufSize = src.BufSize()
	if bufSize < 100 {
		t.Errorf("BufSize() = %d, want >= 100", bufSize)
	}
}

func TestSource_BitDepthNormalization(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bitDepth int
		input    int
		expected float32
	}{
		{"8-bit max", 8, 127, 127.0 / 128.0},
		{"8-bit min", 8, -128, -1.0},
		{"16-bit max", 16, 32767, 32767.0 / 32768.0},
		{"16-bit min", 16, -32768, -1.0},
		{"24-bit", 24, 8388607, 8388607.0 / 8388608.0},
		{"32-bit", 32, 2147483647, 2147483647.0 / 2147483648.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := &source{
				dec: &mockAiffReader{
					sampleRate: 44100,
					channels:   1,
					bitDepth:   tt.bitDepth,
					samples:    []int{tt.input},
				},
				sampleRate: 44100,
				channels:   1,
				bitDepth:   tt.bitDepth,
			}

			dst := make([]float32, 1)
			n, _ := src.ReadSamples(dst)

			if n != 1 {
				t.Fatalf("ReadSamples() n = %d, want 1", n)
			}

			tolerance := float32(0.001)
			if dst[0] < tt.expected-tolerance || dst[0] > tt.expected+tolerance {
				t.Errorf("ReadSamples() dst[0] = %f, want ~%f", dst[0], tt.expected)
			}
		})
	}
}

func TestErrors_AreErrors(t *testing.T) {
	t.Parallel()

	testErrors := []error{
		ErrNotAiffFile,
		ErrOnlyPCM16bitSupported,
		ErrUnsupportedAiffLayout,
		ErrUnsupportedAiffChunks,
	}

	for _, err := range testErrors {
		if err == nil {
			t.Error("Expected non-nil error")
		}

		if err.Error() == "" {
			t.Errorf("Error %v has empty message", err)
		}
	}
}

func TestErrors_IsComparison(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		err    error
		target error
		want   bool
	}{
		{"ErrNotAiffFile matches itself", ErrNotAiffFile, ErrNotAiffFile, true},
		{"ErrNotAiffFile doesn't match ErrOnlyPCM16bitSupported", ErrNotAiffFile, ErrOnlyPCM16bitSupported, false},
		{"ErrOnlyPCM16bitSupported matches itself", ErrOnlyPCM16bitSupported, ErrOnlyPCM16bitSupported, true},
		{"ErrUnsupportedAiffLayout matches itself", ErrUnsupportedAiffLayout, ErrUnsupportedAiffLayout, true},
		{"ErrUnsupportedAiffChunks matches itself", ErrUnsupportedAiffChunks, ErrUnsupportedAiffChunks, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if errors.Is(tt.err, tt.target) != tt.want {
				t.Errorf("errors.Is(%v, %v) = %v, want %v", tt.err, tt.target, !tt.want, tt.want)
			}
		})
	}
}

func TestErrors_Wrapping(t *testing.T) {
	t.Parallel()

	baseErrors := []error{
		ErrNotAiffFile,
		ErrOnlyPCM16bitSupported,
		ErrUnsupportedAiffLayout,
		ErrUnsupportedAiffChunks,
	}

	for _, baseErr := range baseErrors {
		t.Run(baseErr.Error(), func(t *testing.T) {
			wrapped := errors.Join(errors.New("context"), baseErr)

			if !errors.Is(wrapped, baseErr) {
				t.Errorf("Wrapped error doesn't match base error %v", baseErr)
			}
		})
	}
}

func TestErrors_Uniqueness(t *testing.T) {
	t.Parallel()

	errs := []error{
		ErrNotAiffFile,
		ErrOnlyPCM16bitSupported,
		ErrUnsupportedAiffLayout,
		ErrUnsupportedAiffChunks,
	}

	// Check that all error messages are unique
	messages := make(map[string]bool)
	for _, err := range errs {
		msg := err.Error()
		if messages[msg] {
			t.Errorf("Duplicate error message: %s", msg)
		}
		messages[msg] = true
	}
}

func TestErrors_Messages(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err     error
		message string
	}{
		{ErrNotAiffFile, "not an AIFF file"},
		{ErrOnlyPCM16bitSupported, "only 16-bit PCM AIFF is supported"},
		{ErrUnsupportedAiffLayout, "unsupported AIFF layout"},
		{ErrUnsupportedAiffChunks, "unsupported or malformed AIFF chunks"},
	}

	for _, tt := range tests {
		t.Run(tt.message, func(t *testing.T) {
			if tt.err.Error() != tt.message {
				t.Errorf("Error message = %q, want %q", tt.err.Error(), tt.message)
			}
		})
	}
}

// Benchmarks

func BenchmarkSource_ReadSamples(b *testing.B) {
	samples := make([]int, 4096)
	for i := range samples {
		samples[i] = i * 100
	}

	src := &source{
		dec: &mockAiffReader{
			sampleRate: 44100,
			channels:   2,
			bitDepth:   16,
			samples:    samples,
		},
		sampleRate: 44100,
		channels:   2,
		bitDepth:   16,
	}

	dst := make([]float32, 1024)

	b.ResetTimer()
	for b.Loop() {
		// Reset mock reader
		src.dec.(*mockAiffReader).offset = 0

		for {
			n, err := src.ReadSamples(dst)
			if err == io.EOF || n == 0 {
				break
			}
		}
	}
}

func BenchmarkSource_ReadSamples_SmallBuffer(b *testing.B) {
	samples := make([]int, 1024)
	for i := range samples {
		samples[i] = i * 50
	}

	src := &source{
		dec: &mockAiffReader{
			sampleRate: 44100,
			channels:   1,
			bitDepth:   16,
			samples:    samples,
		},
		sampleRate: 44100,
		channels:   1,
		bitDepth:   16,
	}

	dst := make([]float32, 32)

	b.ResetTimer()
	for b.Loop() {
		src.dec.(*mockAiffReader).offset = 0

		for {
			n, err := src.ReadSamples(dst)
			if err == io.EOF || n == 0 {
				break
			}
		}
	}
}

func BenchmarkSource_ReadSamples_LargeBuffer(b *testing.B) {
	samples := make([]int, 65536)
	for i := range samples {
		samples[i] = i
	}

	src := &source{
		dec: &mockAiffReader{
			sampleRate: 48000,
			channels:   2,
			bitDepth:   16,
			samples:    samples,
		},
		sampleRate: 48000,
		channels:   2,
		bitDepth:   16,
	}

	dst := make([]float32, 8192)

	b.ResetTimer()
	for b.Loop() {
		src.dec.(*mockAiffReader).offset = 0

		for {
			n, err := src.ReadSamples(dst)
			if err == io.EOF || n == 0 {
				break
			}
		}
	}
}
