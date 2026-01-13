// SPDX-License-Identifier: EPL-2.0

package mp3

import (
	"fmt"
	"io"

	gomp3 "github.com/hajimehoshi/go-mp3"
	"github.com/ik5/audpbx/audio"
)

// mp3Reader is an interface for gomp3.Decoder to allow testing
type mp3Reader interface {
	Read([]byte) (int, error)
	SampleRate() int
}

type source struct {
	dec        mp3Reader
	sampleRate int
	channels   int
	buf        []byte
}

func (s *source) SampleRate() int { return s.sampleRate }
func (s *source) Channels() int   { return s.channels }
func (s *source) Close() error    { return nil }
func (s *source) BufSize() int    { return cap(s.buf) / 2 } // return sample capacity, not bytes

func (s *source) ReadSamples(dst []float32) (int, error) {
	// go-mp3 returns 16-bit little-endian PCM bytes (stereo interleaved)
	// Each sample is 2 bytes, so we need len(dst) * 2 bytes
	bytesNeeded := len(dst) * 2
	if cap(s.buf) < bytesNeeded {
		s.buf = make([]byte, bytesNeeded)
	}
	s.buf = s.buf[:bytesNeeded]

	n, err := s.dec.Read(s.buf)
	if n == 0 {
		if err != nil {
			return 0, err
		}
		return 0, nil
	}

	// Convert bytes to samples
	// Each sample is 2 bytes (int16 little-endian)
	samples := n / 2
	for i := range samples {
		// Read int16 little-endian
		low := uint16(s.buf[2*i])
		high := uint16(s.buf[2*i+1])
		val := int16(low | (high << 8))
		dst[i] = float32(val) / 32768.0
	}

	return samples, err
}

type Decoder struct{}

func (Decoder) Decode(r io.Reader) (audio.Source, error) {
	dec, err := gomp3.NewDecoder(r)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	// go-mp3 outputs stereo (2 channels) for most MP3 files
	return &source{
		dec:        dec,
		sampleRate: dec.SampleRate(),
		channels:   2,
		buf:        make([]byte, 8192),
	}, nil
}
