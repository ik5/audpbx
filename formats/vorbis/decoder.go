// SPDX-License-Identifier: EPL-2.0

package vorbis

import (
	"fmt"
	"io"

	"github.com/ik5/audpbx/audio"
	"github.com/jfreymuth/oggvorbis"
)

// oggReader is an interface for oggvorbis.Reader to allow testing
type oggReader interface {
	SampleRate() int
	Channels() int
	Read([]float32) (int, error)
}

type source struct {
	dec        oggReader
	sampleRate int
	channels   int
	bufSize    int // preferred read size in samples, reported via BufSize
}

func (s *source) SampleRate() int { return s.sampleRate }
func (s *source) Channels() int   { return s.channels }
func (s *source) Close() error    { return nil }
func (s *source) BufSize() int    { return s.bufSize }

func (s *source) ReadSamples(dst []float32) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}

	// oggvorbis.Reader.Read returns the number of values decoded (frames *
	// channels), which is exactly what ReadSamples reports, and it writes
	// interleaved samples. So it can decode straight into dst with no staging
	// buffer and no copy.
	//
	// Read truncates its buffer to a whole number of frames on its own, so a
	// dst that is not frame-aligned is handled without desynchronising.
	n, err := s.dec.Read(dst)
	if n == 0 {
		if err != nil {
			return 0, err
		}
		return 0, nil
	}

	return n, err
}

type Decoder struct{}

func (Decoder) Decode(r io.Reader) (audio.Source, error) {
	dec, err := oggvorbis.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return &source{
		dec:        dec,
		sampleRate: dec.SampleRate(),
		channels:   dec.Channels(),
		bufSize:    audio.DefaultBufSize,
	}, nil
}
