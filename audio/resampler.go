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

	// Current position in source stream (absolute frame index in source)
	// This represents which source frame we're currently at
	srcPosition int

	// Number of output frames generated so far
	outFramesGenerated int

	// Total source frames read from input
	srcFramesRead int

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
		filterAlpha = 0.5
	}

	r := &Resampler{
		src:         src,
		srcRate:     float64(src.SampleRate()),
		dstRate:     float64(dstRate),
		ratio:       ratio,
		channels:    channels,
		srcBuf:      make([]float32, 4096),
		srcPosition: 0,
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

// readSourceFrameInto reads one frame from source into dst, applies filtering
func (r *Resampler) readSourceFrameInto(dst []float32) (bool, error) {
	if r.eof {
		return false, io.EOF
	}

	n, err := r.src.ReadSamples(r.srcBuf[:r.channels])
	if n > 0 {
		copy(dst, r.srcBuf[:n])

		// Apply simple low-pass filter if downsampling
		if r.useFilter {
			for c := 0; c < r.channels; c++ {
				dst[c] = r.filterAlpha*dst[c] + (1-r.filterAlpha)*r.filterState[c]
				r.filterState[c] = dst[c]
			}
		}

		r.srcFramesRead++
		return true, err
	}

	if err == io.EOF {
		r.eof = true
	}
	return false, err
}

// ensureFrames makes sure we have frames loaded around the current source position
func (r *Resampler) ensureFrames() error {
	// We need frames for indices: srcPosition-1, srcPosition, srcPosition+1, srcPosition+2
	// These map to frames[0], frames[1], frames[2], frames[3]

	// On first call, load initial frames
	if !r.hasFrame[0] {
		// Load first 3 source frames
		// Note: We read directly here to initialize filter state properly
		for i := 0; i < 3; i++ {
			if r.eof {
				break
			}

			n, err := r.src.ReadSamples(r.srcBuf[:r.channels])
			if n > 0 {
				copy(r.frames[i+1], r.srcBuf[:n])
				r.hasFrame[i+1] = true
				r.srcFramesRead++

				// Initialize filter state with first sample (don't apply filter yet)
				if i == 0 && r.useFilter {
					copy(r.filterState, r.srcBuf[:n])
				}
			}
			if err == io.EOF {
				if i == 0 {
					return io.EOF // No data at all
				}
				// Pad with last valid frame
				for j := i + 1; j < 4; j++ {
					copy(r.frames[j], r.frames[i])
					r.hasFrame[j] = true
				}
				break
			} else if err != nil {
				return err
			}
		}

		// frames[0] is padding (duplicate of frames[1])
		copy(r.frames[0], r.frames[1])
		r.hasFrame[0] = true
		return nil
	}

	return nil
}

// shiftFramesAndLoad shifts frames left and loads the next source frame
func (r *Resampler) shiftFramesAndLoad() error {
	// Shift frames: [0,1,2,3] -> [1,2,3,?]
	copy(r.frames[0], r.frames[1])
	copy(r.frames[1], r.frames[2])
	copy(r.frames[2], r.frames[3])
	r.hasFrame[0] = r.hasFrame[1]
	r.hasFrame[1] = r.hasFrame[2]
	r.hasFrame[2] = r.hasFrame[3]

	// Try to load new frame into frames[3]
	ok, err := r.readSourceFrameInto(r.frames[3])
	if ok {
		r.hasFrame[3] = true
		return nil
	}

	// EOF or error - duplicate frames[2] into frames[3]
	if r.hasFrame[2] {
		copy(r.frames[3], r.frames[2])
		r.hasFrame[3] = true
	} else {
		r.hasFrame[3] = false
	}

	return err
}

// ReadSamples produces dst samples at r.dstRate.
// dst length should be a multiple of r.channels.
func (r *Resampler) ReadSamples(dst []float32) (int, error) {
	if len(dst)%r.channels != 0 {
		return 0, ErrInvalidDstSize
	}

	// Ensure initial frames are loaded
	if err := r.ensureFrames(); err != nil {
		return 0, err
	}

	written := 0
	framesNeeded := len(dst) / r.channels

	for written < framesNeeded {
		// Calculate which source position this output frame should sample from
		srcPosFloat := float64(r.outFramesGenerated) * r.ratio

		// Integer part is the source frame index, fractional part is position within
		srcFrameIdx := int(srcPosFloat)
		alpha := float32(srcPosFloat - float64(srcFrameIdx))

		// If we've moved to a new source frame, shift our window
		for srcFrameIdx > r.srcPosition && !r.eof {
			if err := r.shiftFramesAndLoad(); err != nil {
				if err != io.EOF {
					return written * r.channels, err
				}
				// EOF reached, stop shifting
				break
			}
			r.srcPosition++
		}

		// If at EOF and we need a source frame beyond what we've read, stop
		// We can interpolate up to the last frame we read
		if r.eof && srcFrameIdx >= r.srcFramesRead-1 {
			// Check if this position is actually beyond our data
			if srcPosFloat >= float64(r.srcFramesRead-1)+1.0 {
				if written == 0 {
					return 0, io.EOF
				}
				return written * r.channels, io.EOF
			}
		}

		// Check we have valid frames for interpolation
		if !r.hasFrame[1] || !r.hasFrame[2] {
			if written == 0 {
				return 0, io.EOF
			}
			return written * r.channels, io.EOF
		}

		// Perform cubic interpolation
		for c := 0; c < r.channels; c++ {
			y0 := r.frames[0][c]
			y1 := r.frames[1][c]
			y2 := r.frames[2][c]
			y3 := r.frames[3][c]

			dst[written*r.channels+c] = utils.CubicInterpolate(y0, y1, y2, y3, alpha)
		}

		written++
		r.outFramesGenerated++
	}

	return written * r.channels, nil
}
