package mp3

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"
)

// mockMP3Reader simulates the gomp3.Decoder for testing
type mockMP3Reader struct {
	sampleRate   int
	samples      []int16 // PCM samples (16-bit)
	offset       int
	returnErrors bool
}

func (m *mockMP3Reader) SampleRate() int {
	return m.sampleRate
}

func (m *mockMP3Reader) Read(buf []byte) (int, error) {
	if m.returnErrors {
		return 0, io.ErrUnexpectedEOF
	}

	if m.offset >= len(m.samples) {
		return 0, io.EOF
	}

	// Calculate how many samples we can fit in the buffer
	bytesAvailable := (len(m.samples) - m.offset) * 2
	bytesToRead := len(buf)
	if bytesToRead > bytesAvailable {
		bytesToRead = bytesAvailable
	}

	// Ensure we read complete samples (even number of bytes)
	bytesToRead = (bytesToRead / 2) * 2
	samplesToRead := bytesToRead / 2

	// Write samples as little-endian int16
	for i := range samplesToRead {
		sample := m.samples[m.offset+i]
		binary.LittleEndian.PutUint16(buf[i*2:i*2+2], uint16(sample))
	}

	m.offset += samplesToRead

	if m.offset >= len(m.samples) {
		return bytesToRead, io.EOF
	}

	return bytesToRead, nil
}

func TestDecoder_InvalidInput(t *testing.T) {
	t.Parallel()

	// Invalid MP3 data
	invalidData := []byte("This is not MP3 data")

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

	mockReader := &mockMP3Reader{
		sampleRate: 44100,
		samples:    make([]int16, 100),
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		buf:        make([]byte, 8192),
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

	// Create test data: 8 samples (stereo: 4 frames)
	testSamples := []int16{0, 16384, 32767, -16384, -32768, 8192, -8192, 0}

	mockReader := &mockMP3Reader{
		sampleRate: 8000,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 8000,
		channels:   2,
		buf:        make([]byte, 8192),
	}

	dst := make([]float32, 8)
	n, err := src.ReadSamples(dst)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != 8 {
		t.Errorf("ReadSamples() n = %d, want 8", n)
	}

	// Verify int16 -> float32 conversion
	expected := []float32{0.0, 0.5, 1.0, -0.5, -1.0, 0.25, -0.25, 0.0}
	for i := range n {
		if math.Abs(float64(dst[i]-expected[i])) > 0.01 {
			t.Errorf("dst[%d] = %v, want ≈%v", i, dst[i], expected[i])
		}
	}
}

func TestSource_ReadSamples_EmptyBuffer(t *testing.T) {
	t.Parallel()

	mockReader := &mockMP3Reader{
		sampleRate: 8000,
		samples:    make([]int16, 100),
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 8000,
		channels:   2,
		buf:        make([]byte, 8192),
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

	testSamples := []int16{100, 200, 300, 400}

	mockReader := &mockMP3Reader{
		sampleRate: 8000,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 8000,
		channels:   2,
		buf:        make([]byte, 8192),
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

	// 10 samples total
	testSamples := make([]int16, 10)
	for i := range testSamples {
		testSamples[i] = int16(i * 1000)
	}

	mockReader := &mockMP3Reader{
		sampleRate: 8000,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 8000,
		channels:   2,
		buf:        make([]byte, 8192),
	}

	// Read in chunks
	dst := make([]float32, 4)
	n1, err1 := src.ReadSamples(dst)

	if err1 != nil && err1 != io.EOF {
		t.Fatalf("First ReadSamples() error = %v", err1)
	}

	if n1 != 4 {
		t.Errorf("First ReadSamples() n = %d, want 4", n1)
	}

	n2, err2 := src.ReadSamples(dst)

	if err2 != nil && err2 != io.EOF {
		t.Fatalf("Second ReadSamples() error = %v", err2)
	}

	if n2 != 4 {
		t.Errorf("Second ReadSamples() n = %d, want 4", n2)
	}

	// Last samples
	n3, err3 := src.ReadSamples(dst)

	if err3 != io.EOF {
		t.Errorf("Third ReadSamples() error = %v, want io.EOF", err3)
	}

	if n3 != 2 {
		t.Errorf("Third ReadSamples() n = %d, want 2", n3)
	}
}

func TestSource_ReadSamples_ConversionAccuracy(t *testing.T) {
	t.Parallel()

	// Test boundary values and precision
	testSamples := []int16{
		0,      // Zero
		1,      // Minimum positive
		-1,     // Minimum negative
		32767,  // Maximum positive
		-32768, // Maximum negative (exactly -1.0)
		16384,  // Quarter scale
		-16384, // Negative quarter
	}

	mockReader := &mockMP3Reader{
		sampleRate: 44100,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		buf:        make([]byte, 8192),
	}

	dst := make([]float32, len(testSamples))
	n, err := src.ReadSamples(dst)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != len(testSamples) {
		t.Errorf("ReadSamples() n = %d, want %d", n, len(testSamples))
	}

	// Verify conversion accuracy
	expected := []float32{0.0, 1.0 / 32768.0, -1.0 / 32768.0, 1.0, -1.0, 0.5, -0.5}
	for i := range n {
		diff := math.Abs(float64(dst[i] - expected[i]))
		if diff > 0.0001 {
			t.Errorf("dst[%d] = %v, want %v (diff = %v)", i, dst[i], expected[i], diff)
		}
	}
}

func TestSource_ReadSamples_LargeBuffer(t *testing.T) {
	t.Parallel()

	// Create a large sample set
	testSamples := make([]int16, 10000)
	for i := range testSamples {
		testSamples[i] = int16(i % 1000)
	}

	mockReader := &mockMP3Reader{
		sampleRate: 44100,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		buf:        make([]byte, 8192),
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

	testSamples := make([]int16, 100)
	for i := range testSamples {
		testSamples[i] = int16(i * 100)
	}

	mockReader := &mockMP3Reader{
		sampleRate: 8000,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 8000,
		channels:   2,
		buf:        make([]byte, 8192),
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

	mockReader := &mockMP3Reader{
		sampleRate: 44100,
		samples:    make([]int16, 100),
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		buf:        make([]byte, 8192),
	}

	err := src.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil", err)
	}
}

func TestSource_VariousSampleRates(t *testing.T) {
	t.Parallel()

	sampleRates := []int{8000, 11025, 16000, 22050, 32000, 44100, 48000}

	for _, rate := range sampleRates {
		t.Run("", func(t *testing.T) {
			t.Parallel()

			mockReader := &mockMP3Reader{
				sampleRate: rate,
				samples:    make([]int16, 100),
			}

			src := &source{
				dec:        mockReader,
				sampleRate: rate,
				channels:   2,
				buf:        make([]byte, 8192),
			}

			if src.SampleRate() != rate {
				t.Errorf("SampleRate() = %d, want %d", src.SampleRate(), rate)
			}
		})
	}
}

func TestSource_BufferResize(t *testing.T) {
	t.Parallel()

	testSamples := make([]int16, 1000)

	mockReader := &mockMP3Reader{
		sampleRate: 44100,
		samples:    testSamples,
	}

	// Start with small buffer
	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		buf:        make([]byte, 100),
	}

	initialCap := cap(src.buf)

	// Request more samples than buffer can hold
	dst := make([]float32, 1000)
	_, err := src.ReadSamples(dst)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	// Buffer should have grown
	if cap(src.buf) <= initialCap {
		t.Errorf("Buffer capacity = %d, want > %d (should have grown)", cap(src.buf), initialCap)
	}
}

func TestSource_StereoInterleaving(t *testing.T) {
	t.Parallel()

	// Stereo samples: L, R, L, R pattern
	testSamples := []int16{
		1000, 2000, // Frame 1: L=1000, R=2000
		3000, 4000, // Frame 2: L=3000, R=4000
		5000, 6000, // Frame 3: L=5000, R=6000
	}

	mockReader := &mockMP3Reader{
		sampleRate: 44100,
		samples:    testSamples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		buf:        make([]byte, 8192),
	}

	dst := make([]float32, 6)
	n, err := src.ReadSamples(dst)

	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != 6 {
		t.Errorf("ReadSamples() n = %d, want 6", n)
	}

	// Verify interleaving is preserved
	// Frame 1
	if dst[0] < dst[1] {
		// Left should be less than right for first frame
		// 1000/32768 < 2000/32768
	} else {
		t.Error("Stereo interleaving not preserved in frame 1")
	}
}

// BenchmarkSource_ReadSamples benchmarks reading samples
func BenchmarkSource_ReadSamples(b *testing.B) {
	samples := make([]int16, 44100*10) // 10 seconds
	for i := range samples {
		samples[i] = int16(i % 1000)
	}

	mockReader := &mockMP3Reader{
		sampleRate: 44100,
		samples:    samples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		buf:        make([]byte, 8192),
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
	samples := make([]int16, 44100)
	mockReader := &mockMP3Reader{
		sampleRate: 44100,
		samples:    samples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		buf:        make([]byte, 8192),
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
	samples := make([]int16, 441000)
	mockReader := &mockMP3Reader{
		sampleRate: 44100,
		samples:    samples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		buf:        make([]byte, 8192),
	}

	dst := make([]float32, 16384)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		mockReader.offset = 0
		_, _ = src.ReadSamples(dst)
	}
}

// BenchmarkSource_Conversion benchmarks int16->float32 conversion
func BenchmarkSource_Conversion(b *testing.B) {
	samples := make([]int16, 4096)
	for i := range samples {
		samples[i] = int16(i)
	}

	mockReader := &mockMP3Reader{
		sampleRate: 44100,
		samples:    samples,
	}

	src := &source{
		dec:        mockReader,
		sampleRate: 44100,
		channels:   2,
		buf:        make([]byte, 8192),
	}

	dst := make([]float32, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		mockReader.offset = 0
		_, _ = src.ReadSamples(dst)
	}
}

// BenchmarkSource_FullRead benchmarks reading entire audio stream
func BenchmarkSource_FullRead(b *testing.B) {
	samples := make([]int16, 44100) // 1 second
	for i := range samples {
		samples[i] = int16(i % 1000)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for b.Loop() {
		mockReader := &mockMP3Reader{
			sampleRate: 44100,
			samples:    samples,
		}

		src := &source{
			dec:        mockReader,
			sampleRate: 44100,
			channels:   2,
			buf:        make([]byte, 8192),
		}

		dst := make([]float32, 4096)
		for {
			_, err := src.ReadSamples(dst)
			if err == io.EOF {
				break
			}
		}
	}
}
