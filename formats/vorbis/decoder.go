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
	frameBuf   []float32 // buffer for reading frames from decoder
}

func (s *source) SampleRate() int { return s.sampleRate }
func (s *source) Channels() int   { return s.channels }
func (s *source) Close() error    { return nil }
func (s *source) BufSize() int    { return cap(s.frameBuf) }

func (s *source) ReadSamples(dst []float32) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}

	// oggvorbis.Reader.Read() expects a buffer sized in frames (not samples)
	// and returns the number of frames read
	framesRequested := len(dst) / s.channels

	// Ensure our frame buffer is large enough
	if cap(s.frameBuf) < framesRequested*s.channels {
		s.frameBuf = make([]float32, framesRequested*s.channels)
	}
	s.frameBuf = s.frameBuf[:framesRequested*s.channels]

	// Read frames from decoder
	// The oggvorbis library's Read method takes a []float32 and returns frames read
	framesRead, err := s.dec.Read(s.frameBuf)
	if framesRead == 0 {
		if err != nil {
			return 0, err
		}
		return 0, nil
	}

	// Copy the interleaved samples to dst
	samplesRead := framesRead * s.channels
	copy(dst, s.frameBuf[:samplesRead])

	return samplesRead, err
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
		frameBuf:   make([]float32, 4096),
	}, nil
}
