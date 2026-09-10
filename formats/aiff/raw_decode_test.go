// SPDX-License-Identifier: EPL-2.0

package aiff

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"testing"
)

// The other tests in this package drive source through a mockAiffReader, which
// only implements PCMBuffer. That exercises the fallback path and leaves the
// direct SSND decode path — the one real files actually take — untested. These
// tests build genuine AIFF bytes so that path runs, including its byte order.

// encodeFloat80 encodes f as an 80-bit IEEE 754 extended float, the format AIFF
// uses to store sample rate in its COMM chunk.
func encodeFloat80(f float64) [10]byte {
	var out [10]byte

	if f == 0 {
		return out
	}

	sign := uint16(0)
	if f < 0 {
		sign = 1 << 15
		f = -f
	}

	// Normalize to 0.5 <= frac < 1, so f == frac * 2^exp.
	frac, exp := math.Frexp(f)

	// Extended layout stores mantissa with an explicit integer bit: the value is
	// mant/2^63 * 2^(e-16383) with 1 <= mant/2^63 < 2. Since frac*2^exp equals
	// (2*frac)*2^(exp-1), that gives e = exp + 16382 and mant = frac * 2^64.
	e := uint16(exp+16382) | sign
	mant := uint64(math.Ldexp(frac, 64))

	binary.BigEndian.PutUint16(out[0:2], e)
	binary.BigEndian.PutUint64(out[2:10], mant)

	return out
}

// createAIFFFile builds a minimal but valid 16-bit PCM AIFF file.
//
// samples are interleaved and are written big-endian, as AIFF requires.
func createAIFFFile(sampleRate, channels int, samples []int16) []byte {
	const (
		commChunkSize = 18
		ssndHeader    = 8 // offset + blockSize, ahead of the sample data
		bitDepth      = 16
	)

	frames := len(samples) / channels
	pcmBytes := len(samples) * 2
	ssndSize := ssndHeader + pcmBytes

	body := new(bytes.Buffer)

	// COMM chunk: channels, frame count, sample size, sample rate.
	body.WriteString("COMM")
	binary.Write(body, binary.BigEndian, uint32(commChunkSize))
	binary.Write(body, binary.BigEndian, uint16(channels))
	binary.Write(body, binary.BigEndian, uint32(frames))
	binary.Write(body, binary.BigEndian, uint16(bitDepth))
	rate := encodeFloat80(float64(sampleRate))
	body.Write(rate[:])

	// SSND chunk: offset and blockSize are both zero, then the samples.
	body.WriteString("SSND")
	binary.Write(body, binary.BigEndian, uint32(ssndSize))
	binary.Write(body, binary.BigEndian, uint32(0)) // offset
	binary.Write(body, binary.BigEndian, uint32(0)) // blockSize
	for _, s := range samples {
		binary.Write(body, binary.BigEndian, s)
	}

	// FORM wrapper. Size covers "AIFF" plus every chunk that follows.
	out := new(bytes.Buffer)
	out.WriteString("FORM")
	binary.Write(out, binary.BigEndian, uint32(4+body.Len()))
	out.WriteString("AIFF")
	out.Write(body.Bytes())

	return out.Bytes()
}

func TestCreateAIFFFile_IsAccepted(t *testing.T) {
	t.Parallel()

	data := createAIFFFile(44100, 2, []int16{1, 2, 3, 4})

	src, err := Decoder{}.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode() error = %v; the test AIFF builder produced invalid data", err)
	}

	if src.SampleRate() != 44100 {
		t.Errorf("SampleRate() = %d, want 44100", src.SampleRate())
	}

	if src.Channels() != 2 {
		t.Errorf("Channels() = %d, want 2", src.Channels())
	}
}

// TestSource_ReadSamples_RawBigEndian checks that the direct SSND decode path
// reads samples with AIFF's big-endian byte order.
//
// Without this, flipping the byte order in readRaw leaves every test in the
// package passing while producing byte-swapped garbage, because the rest of the
// suite only exercises the PCMBuffer fallback.
func TestSource_ReadSamples_RawBigEndian(t *testing.T) {
	t.Parallel()

	// Values whose two bytes differ, so a byte swap cannot go unnoticed.
	// 0x0102 swapped is 0x0201, and so on.
	samples := []int16{0x0102, 0x7F01, -0x0102, 0x0304, 0x1234, -0x5678}

	data := createAIFFFile(44100, 2, samples)

	src, err := Decoder{}.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	dst := make([]float32, len(samples))
	n, err := src.ReadSamples(dst)
	if err != nil && err != io.EOF {
		t.Fatalf("ReadSamples() error = %v", err)
	}

	if n != len(samples) {
		t.Fatalf("ReadSamples() n = %d, want %d", n, len(samples))
	}

	for i, want := range samples {
		wantFloat := float32(want) / 32768.0
		if dst[i] != wantFloat {
			t.Errorf("sample %d = %v (raw %d), want %v (raw %d)",
				i, dst[i], int16(dst[i]*32768), wantFloat, want)
		}
	}
}

// TestSource_ReadSamples_RawAcrossReads checks the direct path stays in sync
// when the caller reads in several smaller chunks.
func TestSource_ReadSamples_RawAcrossReads(t *testing.T) {
	t.Parallel()

	const channels = 2

	samples := make([]int16, 2000)
	for i := range samples {
		// Distinct and asymmetric between the two bytes.
		samples[i] = int16(i*7 + 1)
	}

	data := createAIFFFile(44100, channels, samples)

	src, err := Decoder{}.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	var got []float32
	dst := make([]float32, 64)

	for {
		n, err := src.ReadSamples(dst)
		got = append(got, dst[:n]...)

		if err == io.EOF {
			break
		}

		if err != nil {
			t.Fatalf("ReadSamples() error = %v", err)
		}
	}

	if len(got) != len(samples) {
		t.Fatalf("read %d samples in total, want %d", len(got), len(samples))
	}

	for i, want := range samples {
		if wantFloat := float32(want) / 32768.0; got[i] != wantFloat {
			t.Fatalf("sample %d = %v, want %v", i, got[i], wantFloat)
		}
	}
}

// TestSource_ReadSamples_RawNeverOverruns checks the Source contract that a
// read never reports more values than the caller's buffer can hold.
func TestSource_ReadSamples_RawNeverOverruns(t *testing.T) {
	t.Parallel()

	samples := make([]int16, 500)
	for i := range samples {
		samples[i] = int16(i)
	}

	data := createAIFFFile(22050, 2, samples)

	src, err := Decoder{}.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	for _, size := range []int{1, 2, 3, 17, 64, 999, 5000} {
		dst := make([]float32, size)

		n, err := src.ReadSamples(dst)
		if err != nil && err != io.EOF {
			t.Fatalf("ReadSamples(%d) error = %v", size, err)
		}

		if n > len(dst) {
			t.Fatalf("ReadSamples(%d) returned n = %d, more than the buffer holds",
				size, n)
		}

		if err == io.EOF {
			break
		}
	}
}

// TestSource_ReadSamples_RawZeroAllocsSteadyState guards the direct decode
// path's other purpose: it must not allocate per call.
//
// Reverting to PCMBuffer would reintroduce four allocations per call plus an
// []int intermediate, which is what made a full conversion allocate millions
// of times.
func TestSource_ReadSamples_RawZeroAllocsSteadyState(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping allocation test in short mode")
	}

	const readSize = 4096

	// Long enough that the sampled runs never reach the end of the stream.
	samples := make([]int16, readSize*200)
	for i := range samples {
		samples[i] = int16(i % 4096)
	}

	data := createAIFFFile(44100, 2, samples)

	src, err := Decoder{}.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	dst := make([]float32, readSize)

	// First read sizes the staging buffer; that one allocation is expected.
	if _, err := src.ReadSamples(dst); err != nil {
		t.Fatalf("priming ReadSamples() error = %v", err)
	}

	allocs := testing.AllocsPerRun(50, func() {
		if _, err := src.ReadSamples(dst); err != nil && err != io.EOF {
			t.Fatalf("ReadSamples() error = %v", err)
		}
	})

	if allocs > 0 {
		t.Errorf("ReadSamples allocated %v times per call in steady state, want 0", allocs)
	}
}
