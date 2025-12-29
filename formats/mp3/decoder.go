package mp3

import (
	"fmt"
	"io"

	gomp3 "github.com/hajimehoshi/go-mp3"
	"github.com/ik5/audpbx/audio"
)

type source struct {
    dec       *gomp3.Decoder
    sampleRate int
    channels   int // mp3 decoder outputs stereo; treat as 2.
    buf        []byte
}

func (s *source) SampleRate() int { return s.sampleRate }
func (s *source) Channels() int   { return s.channels }
func (s *source) Close() error    { return nil }
func (s *source) BufSize() int { return cap(s.buf) }

func (s *source) ReadSamples(dst []float32) (int, error) {
    // go-mp3 returns 16-bit little-endian PCM bytes (stereo).
    if len(s.buf) < len(dst)*2 {
        s.buf = make([]byte, len(dst)*2)
    }
    n, err := s.dec.Read(s.buf[:len(dst)*2])
    if n == 0 && err != nil {
        return 0, fmt.Errorf("%w", err)
    }
    samples := n / 2
    for i := range samples {
        b := s.buf[2*i : 2*i+2]
        v := int16(uint16(b[0]) | (uint16(b[1]) << 8))
        dst[i] = float32(v) / 32768.0
    }
    return samples, err
}

type Decoder struct{}

func (Decoder) Decode(r io.Reader) (audio.Source, error) {
    dec, err := gomp3.NewDecoder(r)
    if err != nil {
        return nil,  fmt.Errorf("%w", err)
    }

    // go-mp3 exposes SampleRate() but not channels; assume 2 for most files
    return &source{
        dec:        dec,
        sampleRate: dec.SampleRate(),
        channels:   2,
        buf:        make([]byte, 8192),
    }, nil
}
