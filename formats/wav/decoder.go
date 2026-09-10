// SPDX-License-Identifier: EPL-2.0

package wav

import (
	"encoding/binary"
	"fmt"
	"io"

	"github.com/go-audio/wav"
	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/utils"
)

// pcmAudioFormat is the WAV format tag for uncompressed PCM.
const pcmAudioFormat = 1

// source wraps go-audio wav.Decoder to implement audio.Source
type source struct {
	dec        *wav.Decoder
	sampleRate int
	channels   int
	bitDepth   int

	// pcm reads the raw bytes of the PCM chunk.
	//
	// Decoding those bytes here rather than calling wav.Decoder.PCMBuffer is a
	// large win: PCMBuffer allocates a byte buffer, a bytes.Reader, a format
	// struct and a per-sample scratch buffer on every call, then decodes one
	// sample at a time through a closure into an []int that is four times wider
	// than the samples it carries. Reading the chunk directly turns the whole
	// thing into one read plus a tight decode loop.
	pcm io.Reader
	raw []byte
}

func (s *source) SampleRate() int { return s.sampleRate }
func (s *source) Channels() int   { return s.channels }
func (s *source) Close() error    { return nil }
func (s *source) BufSize() int    { return audio.DefaultBufSize }

func (s *source) ReadSamples(dst []float32) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}

	// Decode rejects anything that is not 16-bit PCM, so that is the only
	// layout that can reach here. Guard anyway: if the accepted formats are
	// ever widened, this loop has to be widened with them.
	if s.bitDepth != utils.BitDepth16 {
		return 0, ErrOnlyPCM16bitSupported
	}

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

	// WAV PCM is little-endian.
	for i := range samples {
		offset := i * utils.BytesPerSample16
		v := int16(binary.LittleEndian.Uint16(buf[offset:]))
		dst[i] = float32(v) / utils.SampleScale16
	}

	// If we got fewer samples than requested, we're at EOF
	if samples < len(dst) {
		return samples, io.EOF
	}

	return samples, err
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
			return nil, fmt.Errorf("reading wav data: %w", err)
		}
		rs = &readSeeker{data: data, offset: 0}
	}

	dec := wav.NewDecoder(rs)
	if !dec.IsValidFile() {
		return nil, ErrNotWavFile
	}

	// Only support PCM for now
	if dec.WavAudioFormat != pcmAudioFormat {
		return nil, fmt.Errorf("unsupported audio format: %d (only PCM supported)", dec.WavAudioFormat)
	}

	// Check bit depth
	if int(dec.BitDepth) != utils.BitDepth16 {
		return nil, ErrOnlyPCM16bitSupported
	}

	// Forward to PCM data
	if err := dec.FwdToPCM(); err != nil {
		return nil, fmt.Errorf("forwarding to PCM data: %w", err)
	}

	format := dec.Format()
	if format == nil {
		return nil, ErrUnsupportedWavLayout
	}

	// FwdToPCM leaves PCMChunk positioned at the start of the sample data, and
	// its reader is bounded to the chunk, so reads stop at the end of the audio.
	if dec.PCMChunk == nil || dec.PCMChunk.R == nil {
		return nil, ErrUnsupportedWavLayout
	}

	return &source{
		dec:        dec,
		sampleRate: format.SampleRate,
		channels:   format.NumChannels,
		bitDepth:   int(dec.BitDepth),
		pcm:        dec.PCMChunk.R,
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
		return 0, ErrNegativePosition
	}

	rs.offset = newOffset
	return newOffset, nil
}
