// SPDX-License-Identifier: EPL-2.0

package audio

import (
	"fmt"
	"io"

	"github.com/ik5/audpbx/utils"
)

// Resampler streams from src to target sample rate using cubic interpolation.
// Works on interleaved samples; preserves channel count.
// Includes basic anti-aliasing filtering when downsampling.
type Resampler struct {
	src      Source
	srcRate  float64
	dstRate  float64
	ratio    float64 // srcRate / dstRate - how many source samples per output sample
	channels int

	// Ring buffer holding 4 frames for cubic interpolation
	// frames[0] = t-1, frames[1] = t0, frames[2] = t+1, frames[3] = t+2
	frames   [4][]float32
	hasFrame [4]bool

	// Position within the current output stream (in source samples)
	pos float64

	// Buffer for reading from source
	srcBuf []float32
	eof    bool

	// Simple low-pass filter state for anti-aliasing (when downsampling)
	filterState []float32
	useFilter   bool
	filterAlpha float32
}

func NewResampler(src Source, dstRate int) *Resampler {
	channels := src.Channels()
	ratio := float64(src.SampleRate()) / float64(dstRate)

	// Enable simple low-pass filter when downsampling
	useFilter := ratio > 1.0
	var filterAlpha float32
	if useFilter {
		// Simple one-pole low-pass filter
		// Cutoff at Nyquist frequency of destination rate
		// This is a simplified filter - for production, use a proper FIR filter
		filterAlpha = 0.5
	}

	r := &Resampler{
		src:         src,
		srcRate:     float64(src.SampleRate()),
		dstRate:     float64(dstRate),
		ratio:       ratio,
		channels:    channels,
		srcBuf:      make([]float32, 4096),
		pos:         0,
		useFilter:   useFilter,
		filterAlpha: filterAlpha,
		filterState: make([]float32, channels),
	}

	// Initialize frame buffers
	for i := range r.frames {
		r.frames[i] = make([]float32, channels)
	}

	return r
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

// fetchNextFrame reads the next frame from source and shifts the frame buffer
func (r *Resampler) fetchNextFrame() error {
	if r.eof {
		return io.EOF
	}

	// Shift frames: [0,1,2,3] -> [1,2,3,?]
	copy(r.frames[0], r.frames[1])
	copy(r.frames[1], r.frames[2])
	copy(r.frames[2], r.frames[3])
	r.hasFrame[0] = r.hasFrame[1]
	r.hasFrame[1] = r.hasFrame[2]
	r.hasFrame[2] = r.hasFrame[3]

	// Try to read one frame into frames[3]
	n, err := r.src.ReadSamples(r.srcBuf[:r.channels])
	if n > 0 {
		copy(r.frames[3], r.srcBuf[:n])
		r.hasFrame[3] = true

		// Apply simple low-pass filter if downsampling
		if r.useFilter {
			for c := 0; c < r.channels; c++ {
				// One-pole low-pass: y[n] = alpha * x[n] + (1-alpha) * y[n-1]
				r.frames[3][c] = r.filterAlpha*r.frames[3][c] + (1-r.filterAlpha)*r.filterState[c]
				r.filterState[c] = r.frames[3][c]
			}
		}
	} else {
		r.hasFrame[3] = false
	}

	if err == io.EOF {
		r.eof = true
		if !r.hasFrame[3] {
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

	// Initialize frame buffer if needed
	if !r.hasFrame[1] {
		// Fill initial frames
		for i := 0; i < 4; i++ {
			n, err := r.src.ReadSamples(r.srcBuf[:r.channels])
			if n > 0 {
				copy(r.frames[i], r.srcBuf[:n])
				r.hasFrame[i] = true

				// Initialize filter state with first sample to avoid warm-up transients
				if i == 0 && r.useFilter {
					copy(r.filterState, r.srcBuf[:n])
				}
			}
			if err == io.EOF {
				r.eof = true
				if i == 0 {
					return 0, io.EOF
				}
				// Duplicate last valid frame for remaining slots
				for j := i; j < 4; j++ {
					if i > 0 {
						copy(r.frames[j], r.frames[i-1])
						r.hasFrame[j] = true
					}
				}
				break
			} else if err != nil {
				return 0, fmt.Errorf("%w", err)
			}
		}
	}

	written := 0
	framesNeeded := len(dst) / r.channels

	for written < framesNeeded {
		// Ensure we have frames for interpolation
		// pos should be in range [0, 1) for interpolation between frames[1] and frames[2]
		for r.pos >= 1.0 {
			r.pos -= 1.0
			if err := r.fetchNextFrame(); err != nil {
				if err == io.EOF {
					// Source exhausted - return what we have
					if written == 0 {
						return 0, io.EOF
					}
					return written * r.channels, io.EOF
				}
				return written * r.channels, err
			}
		}

		// Check if we have enough frames for cubic interpolation
		if !r.hasFrame[1] || !r.hasFrame[2] {
			// Not enough data
			if written == 0 {
				return 0, io.EOF
			}
			return written * r.channels, io.EOF
		}

		// Cubic interpolation between frames
		alpha := float32(r.pos)

		for c := 0; c < r.channels; c++ {
			var y0, y1, y2, y3 float32

			// Use available frames, duplicate edge frames if needed
			if r.hasFrame[0] {
				y0 = r.frames[0][c]
			} else {
				y0 = r.frames[1][c]
			}

			y1 = r.frames[1][c]
			y2 = r.frames[2][c]

			if r.hasFrame[3] {
				y3 = r.frames[3][c]
			} else {
				y3 = r.frames[2][c]
			}

			dst[written*r.channels+c] = utils.CubicInterpolate(y0, y1, y2, y3, alpha)
		}

		written++
		r.pos += r.ratio
	}

	return written * r.channels, nil
}
