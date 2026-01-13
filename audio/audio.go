// SPDX-License-Identifier: EPL-2.0

package audio

import (
	"io"
	"sync"
)

type Source interface {
    // SampleRate of the PCM stream in Hz.
    SampleRate() int
    // Channels count (e.g., 1=mono, 2=stereo).
    Channels() int
    // ReadSamples fills dst with interleaved float32 samples in [-1,1].
    // Returns number of float32 values written (not frames). When n == 0 with err == io.EOF, the stream is finished.
    ReadSamples(dst []float32) (n int, err error)

    BufSize() int

    // Close releases any resources.
    Close() error
}

// Decoder constructs a Source from an input reader.
type Decoder interface {
    Decode(r io.Reader) (Source, error)
}

// Registry for decoders by format key (e.g., "wav", "mp3", "ogg vorbis").
type Registry struct {
    codecs map[string]Decoder

    mtx *sync.Mutex
}

func NewRegistry() *Registry {
	return &Registry{
		codecs: make(map[string]Decoder),
		mtx: &sync.Mutex{},
    }
}

func (r *Registry) Register(format string, d Decoder) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

	r.codecs[format] = d
}

func (r *Registry) Get(format string) (Decoder, bool) {
	r.mtx.Lock()
	defer r.mtx.Unlock()

    d, ok := r.codecs[format]
    return d, ok
}
