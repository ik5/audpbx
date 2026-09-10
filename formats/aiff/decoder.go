// SPDX-License-Identifier: EPL-2.0

package aiff

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/go-audio/aiff"
	goaudio "github.com/go-audio/audio"
	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/utils"
)

// aiffReader is an interface for aiff.Decoder to allow testing
type aiffReader interface {
	Format() *goaudio.Format
	PCMBuffer(buf *goaudio.IntBuffer) (int, error)
}

// source wraps go-audio aiff.Decoder to implement audio.Source
type source struct {
	dec        aiffReader
	sampleRate int
	channels   int
	bitDepth   int
	intBuf     *goaudio.IntBuffer

	// pcm reads the raw bytes of the SSND chunk when the underlying decoder
	// exposes it. Decoding those bytes here avoids PCMBuffer's four
	// allocations per call and its []int intermediate, which is four times
	// wider than the samples it carries. When pcm is nil (a decoder that only
	// offers PCMBuffer, such as a test double) ReadSamples falls back.
	pcm io.Reader
	raw []byte
}

func (s *source) SampleRate() int { return s.sampleRate }
func (s *source) Channels() int   { return s.channels }
func (s *source) Close() error    { return nil }
func (s *source) BufSize() int {
	if s.intBuf != nil {
		return cap(s.intBuf.Data)
	}
	return audio.DefaultBufSize
}

func (s *source) ReadSamples(dst []float32) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}

	if s.pcm != nil && s.bitDepth == utils.BitDepth16 {
		return s.readRaw(dst)
	}

	return s.readViaPCMBuffer(dst)
}

// readRaw decodes 16-bit big-endian samples straight out of the SSND chunk.
func (s *source) readRaw(dst []float32) (int, error) {
	need := len(dst) * utils.BytesPerSample16
	if cap(s.raw) < need {
		s.raw = make([]byte, need)
	}
	buf := s.raw[:need]

	n, err := io.ReadFull(s.pcm, buf)
	switch {
	case err == io.EOF || err == io.ErrUnexpectedEOF:
		err = io.EOF
	case err != nil:
		return 0, fmt.Errorf("reading pcm data: %w", err)
	}

	samples := n / utils.BytesPerSample16
	if samples == 0 {
		return 0, io.EOF
	}

	// AIFF PCM is big-endian.
	for i := range samples {
		offset := i * utils.BytesPerSample16
		v := int16(binary.BigEndian.Uint16(buf[offset:]))
		dst[i] = float32(v) / utils.SampleScale16
	}

	if samples < len(dst) {
		return samples, io.EOF
	}

	return samples, err
}

func (s *source) readViaPCMBuffer(dst []float32) (int, error) {
	// Resize buffer if needed
	if s.intBuf == nil || cap(s.intBuf.Data) < len(dst) {
		s.intBuf = &goaudio.IntBuffer{
			Data:   make([]int, len(dst)),
			Format: s.dec.Format(),
		}
	} else {
		s.intBuf.Data = s.intBuf.Data[:len(dst)]
	}

	// Read from decoder
	n, err := s.dec.PCMBuffer(s.intBuf)
	if n == 0 {
		if err != nil {
			return 0, err
		}
		return 0, io.EOF
	}

	// Convert int samples to float32
	// go-audio uses int format, we need to normalize based on bit depth
	var maxVal float32
	switch s.bitDepth {
	case utils.BitDepth8:
		maxVal = utils.SampleScale8
	case utils.BitDepth16:
		maxVal = utils.SampleScale16
	case utils.BitDepth24:
		maxVal = utils.SampleScale24
	case utils.BitDepth32:
		maxVal = utils.SampleScale32
	default:
		maxVal = utils.SampleScale16 // Default to 16-bit
	}

	for i := 0; i < n; i++ {
		dst[i] = float32(s.intBuf.Data[i]) / maxVal
	}

	// If we got fewer samples than requested and no error, we're at EOF
	if n < len(dst) && err == nil {
		return n, io.EOF
	}

	return n, err
}

type Decoder struct{}

func (Decoder) Decode(r io.Reader) (audio.Source, error) {
	// go-audio requires io.ReadSeeker
	rs, ok := r.(io.ReadSeeker)
	if !ok {
		// If not a ReadSeeker, we need to read all data into memory
		// This is a limitation of go-audio
		data, err := io.ReadAll(r)
		if err != nil {
			return nil, fmt.Errorf("reading aiff data: %w", err)
		}
		rs = &readSeeker{data: data, offset: 0}
	}

	dec := aiff.NewDecoder(rs)
	if !dec.IsValidFile() {
		return nil, ErrNotAiffFile
	}

	// Read file info
	dec.ReadInfo()

	// Check bit depth - only support 16-bit for now
	if int(dec.BitDepth) != utils.BitDepth16 {
		return nil, ErrOnlyPCM16bitSupported
	}

	format := dec.Format()
	if format == nil {
		return nil, ErrUnsupportedAiffLayout
	}

	// Advance to the sample data so the SSND chunk reader is available for the
	// direct decode path. PCMBuffer would do this lazily; doing it here lets
	// ReadSamples read the chunk itself. The chunk reader is bounded to the
	// chunk and is already positioned past the SSND header, so it yields
	// exactly the sample bytes. If this fails, pcm stays nil and ReadSamples
	// falls back to PCMBuffer.
	var pcm io.Reader
	if err := dec.FwdToPCM(); err == nil && dec.PCMChunk != nil {
		pcm = dec.PCMChunk.R
	}

	return &source{
		dec:        dec,
		pcm:        pcm,
		sampleRate: format.SampleRate,
		channels:   format.NumChannels,
		bitDepth:   int(dec.BitDepth),
	}, nil
}

// readSeeker implements io.ReadSeeker for in-memory data
type readSeeker struct {
	data   []byte
	offset int64
}

func (rs *readSeeker) Read(p []byte) (n int, err error) {
	if rs.offset >= int64(len(rs.data)) {
		return 0, io.EOF
	}
	n = copy(p, rs.data[rs.offset:])
	rs.offset += int64(n)
	return n, nil
}

func (rs *readSeeker) Seek(offset int64, whence int) (int64, error) {
	var newOffset int64
	switch whence {
	case io.SeekStart:
		newOffset = offset
	case io.SeekCurrent:
		newOffset = rs.offset + offset
	case io.SeekEnd:
		newOffset = int64(len(rs.data)) + offset
	default:
		return 0, fmt.Errorf("invalid whence: %d", whence)
	}

	if newOffset < 0 {
		return 0, fmt.Errorf("negative position")
	}

	rs.offset = newOffset
	return newOffset, nil
}
