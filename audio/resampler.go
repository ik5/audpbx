package audio

import (
	"fmt"
	"io"
)

// Resampler streams from src to target sample rate using linear interpolation.
// Works on interleaved samples; preserves channel count.
type Resampler struct {
	src      Source
	srcRate  float64
	dstRate  float64
	ratio    float64 // srcRate / dstRate - how many source samples per output sample
	channels int

	// Ring buffer holding previous and current frame for interpolation
	prevFrame []float32 // previous frame (channels samples)
	currFrame []float32 // current frame (channels samples)
	hasPrev   bool      // whether prevFrame is valid
	hasCurr   bool      // whether currFrame is valid

	// Position within the current output stream (in source samples)
	pos float64

	// Buffer for reading from source
	srcBuf []float32
	eof    bool
}

func NewResampler(src Source, dstRate int) *Resampler {
	channels := src.Channels()
	return &Resampler{
		src:       src,
		srcRate:   float64(src.SampleRate()),
		dstRate:   float64(dstRate),
		ratio:     float64(src.SampleRate()) / float64(dstRate),
		channels:  channels,
		prevFrame: make([]float32, channels),
		currFrame: make([]float32, channels),
		srcBuf:    make([]float32, 4096),
		pos:       0,
	}
}

func (r *Resampler) SampleRate() int { return int(r.dstRate) }
func (r *Resampler) Channels() int   { return r.channels }
func (r *Resampler) BufSize() int    { return r.src.BufSize() }

func (r *Resampler) Close() error {
	err := r.src.Close()
	if err != nil {
		return fmt.Errorf("%w", err)
	}
	return nil
}

// fetchNextFrame reads the next frame from source into currFrame
func (r *Resampler) fetchNextFrame() error {
	if r.eof {
		return io.EOF
	}

	// Shift current to previous
	if r.hasCurr {
		copy(r.prevFrame, r.currFrame)
		r.hasPrev = true
	}

	// Try to read one frame
	n, err := r.src.ReadSamples(r.srcBuf[:r.channels])
	if n > 0 {
		copy(r.currFrame, r.srcBuf[:n])
		r.hasCurr = true
	} else {
		r.hasCurr = false
	}

	if err == io.EOF {
		r.eof = true
		if !r.hasCurr {
			return io.EOF
		}
	} else if err != nil {
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

	// Initialize if needed
	if !r.hasCurr {
		if err := r.fetchNextFrame(); err != nil {
			return 0, err
		}
	}

	written := 0
	framesNeeded := len(dst) / r.channels

	for written < framesNeeded {
		// Ensure we have current and next frame for interpolation
		// pos is in terms of source frame index
		// When pos >= 1.0, we need to advance to the next frame
		for r.pos >= 1.0 {
			r.pos -= 1.0
			if err := r.fetchNextFrame(); err != nil {
				if err == io.EOF && written > 0 {
					return written * r.channels, nil
				}
				return written * r.channels, err
			}
		}

		// Now interpolate between prevFrame and currFrame
		// If we don't have a previous frame, use current for both
		var frame0, frame1 []float32
		if r.hasPrev {
			frame0 = r.prevFrame
			frame1 = r.currFrame
		} else {
			// First frame - duplicate current
			frame0 = r.currFrame
			frame1 = r.currFrame
		}

		// Linear interpolation
		alpha := float32(r.pos)
		for c := 0; c < r.channels; c++ {
			s0 := frame0[c]
			s1 := frame1[c]
			dst[written*r.channels+c] = s0 + alpha*(s1-s0)
		}

		written++
		r.pos += r.ratio
	}

	return written * r.channels, nil
}
