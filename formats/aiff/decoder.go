package aiff

import (
	"fmt"
	"io"

	"github.com/go-audio/aiff"
	goaudio "github.com/go-audio/audio"
	"github.com/ik5/audpbx/audio"
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
}

func (s *source) SampleRate() int { return s.sampleRate }
func (s *source) Channels() int   { return s.channels }
func (s *source) Close() error    { return nil }
func (s *source) BufSize() int {
	if s.intBuf != nil {
		return cap(s.intBuf.Data)
	}
	return 4096
}

func (s *source) ReadSamples(dst []float32) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}

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
	case 8:
		maxVal = 128.0
	case 16:
		maxVal = 32768.0
	case 24:
		maxVal = 8388608.0
	case 32:
		maxVal = 2147483648.0
	default:
		maxVal = 32768.0 // Default to 16-bit
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
	if dec.BitDepth != 16 {
		return nil, ErrOnlyPCM16bitSupported
	}

	format := dec.Format()
	if format == nil {
		return nil, ErrUnsupportedAiffLayout
	}

	return &source{
		dec:        dec,
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
