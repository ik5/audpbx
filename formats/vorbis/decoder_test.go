// SPDX-License-Identifier: EPL-2.0

package vorbis

import (
	"bytes"
	"io"
	"testing"
)

// mockOggVorbisReader simulates the oggvorbis.Reader for testing
type mockOggVorbisReader struct {
	sampleRate   int
	channels     int
	samples      []float32
	offset       int
	returnErrors bool
}

func (m *mockOggVorbisReader) SampleRate() int {
	return m.sampleRate
}

func (m *mockOggVorbisReader) Channels() int {
	return m.channels
}

func (m *mockOggVorbisReader) Read(buf []float32) (int, error) {
	if m.returnErrors {
		return 0, io.ErrUnexpectedEOF
	}

	if m.offset >= len(m.samples) {
		return 0, io.EOF
	}

	// Calculate frames (not samples)
	framesRequested := len(buf) / m.channels
	samplesAvailable := len(m.samples) - m.offset
	framesAvailable := samplesAvailable / m.channels

	framesToRead := framesRequested
	if framesToRead > framesAvailable {
		framesToRead = framesAvailable
	}

	samplesToRead := framesToRead * m.channels
	copy(buf, m.samples[m.offset:m.offset+samplesToRead])
	m.offset += samplesToRead

	if m.offset >= len(m.samples) {
		return framesToRead, io.EOF
	}

	return framesToRead, nil
}

func TestDecoder_InvalidInput(t *testing.T) {
	t.Parallel()

	// Invalid Ogg Vorbis data
	invalidData := []byte("This is not Ogg Vorbis data")

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
		dec: &mockOggVorbisReader{
			sampleRate: 44100,
			channels:   2,
			samples:    make([]float32, 100),
		},
		sampleRate: 44100,
		channels:   2,
		frameBuf:   make([]float32, 4096),
	}

	if src.SampleRate() != 44100 {
		t.Errorf("SampleRate() = %d, want 44100", src.SampleRate())
	}

	if src.Channels() != 2 {
		t.Errorf("Channels() = %d, want 2", src.Channels())
	}

	if src.BufSize() <= 0 {
		t.Errorf("BufSize() = %d, want positive value", src.BufSize())
	}
}

func TestSource_ReadSamples(t *testing.T) {
	t.Parallel()

	// Create test data: stereo samples
	testSamples := []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8}

	mockReader := &mockOggVorbisReader{
		sampleRate: 8000,
		channels:   2,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 8000,
		channels:   2,
		frameBuf:   make([]float32, 4096),
	}

	dst := make([]float32, 8)
	n, err := src.ReadSamples(dst)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != 8 {
		t.Errorf("ReadSamples() n = %d, want 8", n)
	}

	// Verify samples match
	for i := range n {
		if dst[i] != testSamples[i] {
			t.Errorf("dst[%d] = %v, want %v", i, dst[i], testSamples[i])
		}
	}
}

func TestSource_ReadSamples_EmptyBuffer(t *testing.T) {
	t.Parallel()

	mockReader := &mockOggVorbisReader{
		sampleRate: 8000,
		channels:   1,
		samples:    make([]float32, 100),
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 8000,
		channels:   1,
		frameBuf:   make([]float32, 4096),
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

	testSamples := []float32{0.1, 0.2, 0.3, 0.4}

	mockReader := &mockOggVorbisReader{
		sampleRate: 8000,
		channels:   2,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 8000,
		channels:   2,
		frameBuf:   make([]float32, 4096),
	}

	// Read all samples
	dst := make([]float32, 4)
	n1, err1 := src.ReadSamples(dst)

	if err1 != io.EOF {
		t.Errorf("ReadSamples() error = %v, want io.EOF", err1)
	}

	if n1 != 4 {
		t.Errorf("ReadSamples() n = %d, want 4", n1)
	}

	// Try to read more
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

	// Create 6 samples (3 stereo frames)
	testSamples := []float32{0.1, 0.2, 0.3, 0.4, 0.5, 0.6}

	mockReader := &mockOggVorbisReader{
		sampleRate: 8000,
		channels:   2,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 8000,
		channels:   2,
		frameBuf:   make([]float32, 4096),
	}

	// Read in chunks
	dst := make([]float32, 4) // 2 frames
	n1, err1 := src.ReadSamples(dst)

	if err1 != nil && err1 != io.EOF {
		t.Fatalf("First ReadSamples() error = %v", err1)
	}

	if n1 != 4 {
		t.Errorf("First ReadSamples() n = %d, want 4", n1)
	}

	// Verify first chunk
	for i := range n1 {
		if dst[i] != testSamples[i] {
			t.Errorf("First chunk dst[%d] = %v, want %v", i, dst[i], testSamples[i])
		}
	}

	// Read remaining
	n2, err2 := src.ReadSamples(dst)

	if err2 != io.EOF {
		t.Errorf("Second ReadSamples() error = %v, want io.EOF", err2)
	}

	if n2 != 2 {
		t.Errorf("Second ReadSamples() n = %d, want 2", n2)
	}

	// Verify second chunk
	for i := range n2 {
		if dst[i] != testSamples[4+i] {
			t.Errorf("Second chunk dst[%d] = %v, want %v", i, dst[i], testSamples[4+i])
		}
	}
}

func TestSource_ReadSamples_Mono(t *testing.T) {
	t.Parallel()

	testSamples := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

	mockReader := &mockOggVorbisReader{
		sampleRate: 16000,
		channels:   1,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 16000,
		channels:   1,
		frameBuf:   make([]float32, 4096),
	}

	dst := make([]float32, 5)
	n, err := src.ReadSamples(dst)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != 5 {
		t.Errorf("ReadSamples() n = %d, want 5", n)
	}

	for i := range n {
		if dst[i] != testSamples[i] {
			t.Errorf("dst[%d] = %v, want %v", i, dst[i], testSamples[i])
		}
	}
}

func TestSource_ReadSamples_Stereo(t *testing.T) {
	t.Parallel()

	// Stereo: L, R, L, R pattern
	testSamples := []float32{0.1, 0.9, 0.2, 0.8, 0.3, 0.7}

	mockReader := &mockOggVorbisReader{
		sampleRate: 44100,
		channels:   2,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		frameBuf:   make([]float32, 4096),
	}

	dst := make([]float32, 6)
	n, err := src.ReadSamples(dst)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != 6 {
		t.Errorf("ReadSamples() n = %d, want 6", n)
	}

	// Verify interleaved pattern is preserved
	for i := range n {
		if dst[i] != testSamples[i] {
			t.Errorf("dst[%d] = %v, want %v", i, dst[i], testSamples[i])
		}
	}
}

func TestSource_ReadSamples_MultipleChannels(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		channels int
		samples  int
	}{
		{"Mono", 1, 100},
		{"Stereo", 2, 100},
		{"5.1 Surround", 6, 120},
		{"7.1 Surround", 8, 128},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			testSamples := make([]float32, tt.samples)
			for i := range testSamples {
				testSamples[i] = float32(i) / 1000.0
			}

			mockReader := &mockOggVorbisReader{
				sampleRate: 48000,
				channels:   tt.channels,
				samples:    testSamples,
			}

			src := &source{
				dec:        mockReader,
				sampleRate: 48000,
				channels:   tt.channels,
				frameBuf:   make([]float32, 4096),
			}

			if src.Channels() != tt.channels {
				t.Errorf("Channels() = %d, want %d", src.Channels(), tt.channels)
			}

			dst := make([]float32, tt.samples)
			n, err := src.ReadSamples(dst)

			if err != nil && err != io.EOF {
				t.Fatalf("ReadSamples() error = %v", err)
			}

			if n != tt.samples {
				t.Errorf("ReadSamples() n = %d, want %d", n, tt.samples)
			}
		})
	}
}

func TestSource_ReadSamples_LargeBuffer(t *testing.T) {
	t.Parallel()

	// Create a large sample set
	testSamples := make([]float32, 10000)
	for i := range testSamples {
		testSamples[i] = float32(i % 1000) / 1000.0
	}

	mockReader := &mockOggVorbisReader{
		sampleRate: 44100,
		channels:   2,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		frameBuf:   make([]float32, 4096),
	}

	dst := make([]float32, 10000)
	n, err := src.ReadSamples(dst)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != 10000 {
		t.Errorf("ReadSamples() n = %d, want 10000", n)
	}
}

func TestSource_ReadSamples_SmallReads(t *testing.T) {
	t.Parallel()

	testSamples := make([]float32, 100)
	for i := range testSamples {
		testSamples[i] = float32(i) / 100.0
	}

	mockReader := &mockOggVorbisReader{
		sampleRate: 8000,
		channels:   1,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 8000,
		channels:   1,
		frameBuf:   make([]float32, 4096),
	}

	// Read in very small chunks
	totalRead := 0
	for totalRead < 100 {
		dst := make([]float32, 5)
		n, err := src.ReadSamples(dst)

		if n > 0 {
			totalRead += n
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			t.Fatalf("ReadSamples() error = %v", err)
		}
	}

	if totalRead != 100 {
		t.Errorf("Total samples read = %d, want 100", totalRead)
	}
}

func TestSource_Close(t *testing.T) {
	t.Parallel()

	mockReader := &mockOggVorbisReader{
		sampleRate: 44100,
		channels:   2,
		samples:    make([]float32, 100),
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		frameBuf:   make([]float32, 4096),
	}

	err := src.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestSource_VariousSampleRates(t *testing.T) {
	t.Parallel()

	sampleRates := []int{8000, 16000, 22050, 44100, 48000, 96000}

	for _, rate := range sampleRates {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			mockReader := &mockOggVorbisReader{
				sampleRate: rate,
				channels:   2,
				samples:    make([]float32, 100),
			}

			src := &source{
				dec:        mockReader,
				sampleRate: rate,
				channels:   2,
				frameBuf:   make([]float32, 4096),
			}

			if src.SampleRate() != rate {
				t.Errorf("SampleRate() = %d, want %d", src.SampleRate(), rate)
			}
		})
	}
}

// BenchmarkSource_ReadSamples benchmarks reading samples
func BenchmarkSource_ReadSamples(b *testing.B) {
	samples := make([]float32, 44100*10) // 10 seconds
	for i := range samples {
		samples[i] = float32(i%1000) / 1000.0
	}

	mockReader := &mockOggVorbisReader{
		sampleRate: 44100,
		channels:   2,
		samples:    samples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		frameBuf:   make([]float32, 4096),
	}

	dst := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		mockReader.offset = 0
		_, _ = src.ReadSamples(dst)
	}
}

// BenchmarkSource_ReadSamples_SmallBuffer benchmarks small buffer reads
func BenchmarkSource_ReadSamples_SmallBuffer(b *testing.B) {
	samples := make([]float32, 44100)
	mockReader := &mockOggVorbisReader{
		sampleRate: 44100,
		channels:   1,
		samples:    samples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   1,
		frameBuf:   make([]float32, 4096),
	}

	dst := make([]float32, 64)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		mockReader.offset = 0
		_, _ = src.ReadSamples(dst)
	}
}

// BenchmarkSource_ReadSamples_LargeBuffer benchmarks large buffer reads
func BenchmarkSource_ReadSamples_LargeBuffer(b *testing.B) {
	samples := make([]float32, 441000)
	mockReader := &mockOggVorbisReader{
		sampleRate: 44100,
		channels:   2,
		samples:    samples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		frameBuf:   make([]float32, 4096),
	}

	dst := make([]float32, 16384)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		mockReader.offset = 0
		_, _ = src.ReadSamples(dst)
	}
}

// BenchmarkSource_ReadSamples_Mono benchmarks mono audio
func BenchmarkSource_ReadSamples_Mono(b *testing.B) {
	samples := make([]float32, 44100)
	mockReader := &mockOggVorbisReader{
		sampleRate: 44100,
		channels:   1,
		samples:    samples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   1,
		frameBuf:   make([]float32, 4096),
	}

	dst := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		mockReader.offset = 0
		_, _ = src.ReadSamples(dst)
	}
}

// BenchmarkSource_ReadSamples_Stereo benchmarks stereo audio
func BenchmarkSource_ReadSamples_Stereo(b *testing.B) {
	samples := make([]float32, 88200) // stereo
	mockReader := &mockOggVorbisReader{
		sampleRate: 44100,
		channels:   2,
		samples:    samples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		frameBuf:   make([]float32, 4096),
	}

	dst := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		mockReader.offset = 0
		_, _ = src.ReadSamples(dst)
	}
}
