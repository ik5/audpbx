package vorbis

import (
	"fmt"
	"io"

	"github.com/ik5/audpbx/audio"
	"github.com/jfreymuth/oggvorbis"
)

type source struct {
    dec        *oggvorbis.Reader
    sampleRate int
    channels   int
    // working buffer in float32 frames
    tmp []float32
}

func (s *source) SampleRate() int { return s.sampleRate }
func (s *source) Channels() int   { return s.channels }
func (s *source) Close() error    { return nil }
func (s *source) BufSize() int { return cap(s.tmp) }

func (s *source) ReadSamples(dst []float32) (int, error) {
    frames := len(dst) / s.channels
    if frames == 0 {
        return 0, nil
    }
    if len(s.tmp) < frames*s.channels {
        s.tmp = make([]float32, frames*s.channels)
    }
    n, err := s.dec.Read(s.tmp)
    if n == 0 && err != nil {
        return 0, fmt.Errorf("%w", err)
    }
    // Copy as float32
    copy(dst[:n*s.channels], s.tmp[:n*s.channels])
    return n * s.channels, err
}

type Decoder struct{}

func (Decoder) Decode(r io.Reader) (audio.Source, error) {
    dec, err := oggvorbis.NewReader(r)
    if err != nil {
        return nil, err
    }
    return &source{
        dec:        dec,
        sampleRate: dec.SampleRate(),
        channels:   dec.Channels(),
        tmp:        make([]float32, 4096),
    }, nil
}
