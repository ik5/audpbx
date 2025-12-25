package audio

import (
	"errors"
	"fmt"
	"io"
	"math"
)

// Resampler streams from src to target sample rate using linear interpolation.
// Works on interleaved samples; preserves channel count.
type Resampler struct {
    src          Source
    srcRate      float64
    dstRate      float64
    ratio        float64
    channels     int
    // fractional position in source (in frames)
    pos          float64
    // small ring buffer per channel: previous and next samples
    // We keep two frames worth (prev,next) = 2*channels values.
    buf          []float32
    // internal frame buffer pulled from src
    inFrame      []float32
}

func NewResampler(src Source, dstRate int) *Resampler {
    return &Resampler{
        src:      src,
        srcRate:  float64(src.SampleRate()),
        dstRate:  float64(dstRate),
        ratio:    float64(src.SampleRate()) / float64(dstRate),
        channels: src.Channels(),
        buf:      make([]float32, 2*src.Channels()),
        inFrame:  make([]float32, 4096), // multiple of channels
    }
}

func (r *Resampler) SampleRate() int  { return int(r.dstRate) }
func (r *Resampler) Channels() int    { return r.channels }

func (r *Resampler) Close() error     {
	err := r.src.Close()
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

// ReadSamples produces dst samples at r.dstRate.
// dst length should be a multiple of r.channels.
func (r *Resampler) ReadSamples(dst []float32) (int, error) {
    if len(dst)%r.channels != 0 {
        return 0, ErrInvalidDstSize
    }

    // Ensure we have an initial frame for interpolation
    if r.pos == 0 {
        // Fill buf with first two frames (or duplicates if not enough)
        if err := r.fillInitialBuffer(); err != nil {
            if errors.Is(err, io.EOF) {
                return 0, io.EOF
            }
            return 0, fmt.Errorf("%w", err)
        }
    }

    written := 0
    dstFrames := len(dst) / r.channels

    for written < len(dst) {
        // Need to ensure we can interpolate at current pos:
        // pos is fractional frame index; floor gives current frame,
        // we need the following frame available too.
        for math.Floor(r.pos)+1 >= float64(len(r.inFrame)/r.channels) {
            // Shift pos down while we fetch more source data.
            r.pos -= float64(len(r.inFrame) / r.channels)
            // Pull more source data
            n, err := r.src.ReadSamples(r.inFrame[:cap(r.inFrame)])
            if n == 0 {
                if errors.Is(err, io.EOF) {
                    // If we cannot produce more, finalize
                    if written == 0 {
                        return 0, io.EOF
                    }
                    return written, nil
                }
                if err != nil {
                    return written, fmt.Errorf("%w", err)
                }
            }
            r.inFrame = r.inFrame[:n]
            if len(r.inFrame) == 0 {
                // No more data
                if written == 0 {
                    return 0, io.EOF
                }
                return written, nil
            }
        }

        // Interpolate one output frame
        srcFrameIndex := int(math.Floor(r.pos))
        alpha := float32(r.pos - float64(srcFrameIndex))
        for c := 0; c < r.channels; c++ {
            i0 := (srcFrameIndex*r.channels + c)
            i1 := i0 + r.channels
            s0 := r.inFrame[i0]
            // Guard for last frame: repeat s0 if i1 out of bounds
            var s1 float32
            if i1 < len(r.inFrame) {
                s1 = r.inFrame[i1]
            } else {
                s1 = s0
            }
            out := s0 + alpha*(s1-s0)
            dst[written+c] = out
        }
        written += r.channels
        r.pos += r.ratio
        if written/r.channels >= dstFrames {
            break
        }
    }
    return written, nil
}

func (r *Resampler) fillInitialBuffer() error {
    // Prime internal buffer with some source data
    n, err := r.src.ReadSamples(r.inFrame[:cap(r.inFrame)])
    if n == 0 && err != nil {
        return fmt.Errorf("%w",err)
    }
    r.inFrame = r.inFrame[:n]
    return nil
}
