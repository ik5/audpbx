// SPDX-License-Identifier: EPL-2.0

package audio

import (
	"fmt"
	"io"

	"github.com/ik5/audpbx/utils"
)

// resamplerBlockFrames is how many source frames the resampler pulls from the
// underlying Source per read.
//
// This constant is the difference between "fast" and "unusably slow". Pulling
// one frame at a time means one Source.ReadSamples call — and, for a decoder
// sitting on an unbuffered file, one read(2) syscall plus a handful of
// allocations — for every single source frame. A 100 second 44.1 kHz stereo
// file is 4.4 million frames, so the per-call overhead completely dominates the
// arithmetic. Reading in blocks amortises all of it away.
const resamplerBlockFrames = 8192

// resamplerHistoryFrames is the number of frames kept behind the current
// position so cubic interpolation always has its y0 neighbour.
const resamplerHistoryFrames = 1

// resamplerLookaheadFrames is the number of frames kept ahead of the current
// position so cubic interpolation always has its y2 and y3 neighbours.
const resamplerLookaheadFrames = 2

// resamplerFilterPrimeFrames is the number of leading source frames that bypass
// the anti-aliasing filter while its state primes.
const resamplerFilterPrimeFrames = 3

// downsampleRatioThreshold is the srcRate/dstRate ratio above which the output
// rate can no longer carry the source bandwidth, so anti-alias filtering is
// needed. A ratio at or below unity is upsampling, which cannot alias.
const downsampleRatioThreshold = 1.0

// defaultFilterAlpha is the coefficient of the one-pole anti-aliasing low-pass
// applied when downsampling: out = alpha*in + (1-alpha)*prev.
//
// It is a fixed coefficient rather than one derived from the resampling ratio,
// so the cutoff does not track the output Nyquist frequency and attenuates far
// less than a proper decimation filter would at large ratios.
const defaultFilterAlpha float32 = 0.5

// unityGain is the full-scale filter coefficient, used to split the one-pole
// low-pass between its input and its retained state.
const unityGain float32 = 1.0

// Resampler streams from src to target sample rate using cubic interpolation.
//
// Resampling changes the sample rate (number of audio measurements per second) of
// audio without changing the pitch or speed. This is essential when:
//   - Converting between different audio standards (44.1 kHz CD to 48 kHz pro audio)
//   - Reducing bandwidth for streaming (downsampling to 8 kHz for voice calls)
//   - Matching hardware requirements (some devices only support specific rates)
//
// The resampler uses cubic interpolation (Catmull-Rom spline) for high quality:
//   - Upsampling (e.g., 22 kHz → 44 kHz): Creates new samples between existing ones
//   - Downsampling (e.g., 48 kHz → 16 kHz): Selects and interpolates fewer samples
//
// Features:
//   - Works on interleaved samples; preserves channel count
//   - Includes basic anti-aliasing filtering when downsampling to prevent artifacts
//   - Reads the source in large blocks, so decoder call overhead is amortised
//   - Zero allocations after initialization for optimal performance
//
// Example:
//
//	// Resample from 44.1 kHz to 16 kHz
//	resampler := audio.NewResampler(source, 16000)
//	buf := make([]float32, 4096)
//	n, err := resampler.ReadSamples(buf)
type Resampler struct {
	src      Source
	srcRate  float64
	dstRate  float64
	ratio    float64 // srcRate / dstRate - how many source samples per output sample
	channels int

	// in is a sliding window of interleaved source frames. The frame at buffer
	// offset b holds absolute source frame inBase+b.
	//
	// inBase starts at -1, and offset 0 is seeded with a copy of source frame 0.
	// That synthetic frame is the y0 neighbour for the very first output frame,
	// which keeps the interpolation loop free of start-of-stream branches.
	in       []float32
	inFrames int   // frames currently resident in in
	inBase   int64 // absolute source frame index of in's first frame

	// srcFramesRead counts real frames pulled from src, excluding the synthetic
	// history frame and any end-of-stream padding.
	srcFramesRead int64

	// outFrame is the index of the next output frame to generate.
	outFrame int64

	primed bool
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
	useFilter := ratio > downsampleRatioThreshold
	var filterAlpha float32
	if useFilter {
		// Simple one-pole low-pass filter
		filterAlpha = defaultFilterAlpha
	}

	// The window has to hold a full block plus the interpolator's neighbours on
	// either side, so a refill never has to split a block.
	windowFrames := resamplerBlockFrames + resamplerHistoryFrames + resamplerLookaheadFrames

	return &Resampler{
		src:         src,
		srcRate:     float64(src.SampleRate()),
		dstRate:     float64(dstRate),
		ratio:       ratio,
		channels:    channels,
		in:          make([]float32, windowFrames*channels),
		inBase:      -resamplerHistoryFrames,
		useFilter:   useFilter,
		filterAlpha: filterAlpha,
		filterState: make([]float32, channels),
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

// windowLast returns the highest absolute source frame index resident in in.
func (r *Resampler) windowLast() int64 {
	return r.inBase + int64(r.inFrames) - 1
}

// pull appends source frames to the free space at the end of in, applying the
// anti-aliasing filter to each new frame. It reads until the window is full or
// the source is exhausted, so a single call replaces thousands of one-frame
// reads.
func (r *Resampler) pull() error {
	channels := r.channels
	capFrames := len(r.in) / channels

	for r.inFrames < capFrames && !r.eof {
		offset := r.inFrames * channels

		n, err := r.src.ReadSamples(r.in[offset:])

		// A source that hands back a partial frame would desynchronise the
		// interleaving, so leave the remainder for the next read.
		n -= n % channels

		if n > 0 {
			frames := n / channels
			r.filterFrames(r.inFrames, frames)
			r.inFrames += frames
			r.srcFramesRead += int64(frames)
		}

		if err == io.EOF {
			r.eof = true
			break
		}

		if err != nil {
			return err
		}

		if n == 0 {
			// The source produced nothing yet did not report EOF. Treat it as
			// the end of the stream rather than spinning forever.
			r.eof = true
			break
		}
	}

	return nil
}

// filterFrames applies the one-pole anti-aliasing low-pass to frames
// [atFrame, atFrame+count) of the window, in place.
//
// The first resamplerFilterPrimeFrames source frames pass through unfiltered
// and only frame 0 seeds the filter state, which is what the original
// frame-at-a-time implementation did.
func (r *Resampler) filterFrames(atFrame, count int) {
	if !r.useFilter || count == 0 {
		return
	}

	channels := r.channels
	absolute := r.inBase + int64(atFrame)

	// Walk past the unfiltered prime region, seeding the state from frame 0.
	for count > 0 && absolute < resamplerFilterPrimeFrames {
		if absolute == 0 {
			offset := atFrame * channels
			copy(r.filterState, r.in[offset:offset+channels])
		}
		absolute++
		atFrame++
		count--
	}

	if count == 0 {
		return
	}

	alpha := r.filterAlpha
	beta := unityGain - alpha
	offset := atFrame * channels
	end := offset + count*channels

	// Mono and stereo are the overwhelmingly common cases and are worth keeping
	// out of the inner per-channel loop.
	switch channels {
	case LayoutMono.ChannelCount():
		state := r.filterState[0]
		for i := offset; i < end; i++ {
			state = alpha*r.in[i] + beta*state
			r.in[i] = state
		}
		r.filterState[0] = state
	case LayoutStereo.ChannelCount():
		left, right := r.filterState[0], r.filterState[1]
		for i := offset; i < end; i += LayoutStereo.ChannelCount() {
			left = alpha*r.in[i] + beta*left
			right = alpha*r.in[i+1] + beta*right
			r.in[i] = left
			r.in[i+1] = right
		}
		r.filterState[0], r.filterState[1] = left, right
	default:
		for i := offset; i < end; i += channels {
			for c := range channels {
				state := alpha*r.in[i+c] + beta*r.filterState[c]
				r.in[i+c] = state
				r.filterState[c] = state
			}
		}
	}
}

// prime loads the first block and seeds the synthetic history frame at offset 0.
func (r *Resampler) prime() error {
	r.primed = true

	// Reserve offset 0 for the history copy of source frame 0; pull fills in
	// from offset 1 onward.
	r.inFrames = resamplerHistoryFrames

	if err := r.pull(); err != nil {
		return err
	}

	if r.srcFramesRead == 0 {
		return io.EOF
	}

	channels := r.channels
	first := resamplerHistoryFrames * channels
	copy(r.in[:first], r.in[first:first+channels])

	return nil
}

// slide repositions the window so that absolute source frames idx-1 through
// idx+2 — the four samples cubic interpolation needs — are all resident.
func (r *Resampler) slide(idx int64) error {
	channels := r.channels
	need := idx + resamplerLookaheadFrames

	if need <= r.windowLast() {
		return nil
	}

	first := idx - resamplerHistoryFrames

	// With an extreme ratio a single output step can jump clean past the
	// window. Discard whole windows of source until the target is reachable.
	for !r.eof && first > r.windowLast() {
		r.inBase += int64(r.inFrames)
		r.inFrames = 0
		if err := r.pull(); err != nil {
			return err
		}
	}

	// Drop the frames we have moved past, making room for the next block.
	if drop := first - r.inBase; drop > 0 {
		if int(drop) >= r.inFrames {
			r.inBase += int64(r.inFrames)
			r.inFrames = 0
		} else {
			copy(r.in, r.in[int(drop)*channels:r.inFrames*channels])
			r.inFrames -= int(drop)
			r.inBase = first
		}
	}

	if err := r.pull(); err != nil {
		return err
	}

	// At end of stream, clamp: repeat the last real frame so the interpolator
	// still sees four samples.
	if r.eof && r.inFrames > 0 {
		last := (r.inFrames - 1) * channels
		capFrames := len(r.in) / channels
		for r.windowLast() < need && r.inFrames < capFrames {
			offset := r.inFrames * channels
			copy(r.in[offset:offset+channels], r.in[last:last+channels])
			r.inFrames++
		}
	}

	return nil
}

// ReadSamples produces dst samples at r.dstRate.
// dst length should be a multiple of r.channels.
func (r *Resampler) ReadSamples(dst []float32) (int, error) {
	channels := r.channels

	if len(dst)%channels != 0 {
		return 0, ErrInvalidDstSize
	}

	if !r.primed {
		if err := r.prime(); err != nil {
			return 0, err
		}
	}

	framesNeeded := len(dst) / channels
	written := 0

	for written < framesNeeded {
		// Calculate which source position this output frame should sample from.
		// Integer part is the source frame index, fractional part is position
		// within it.
		srcPosFloat := float64(r.outFrame) * r.ratio
		srcFrameIdx := int64(srcPosFloat)
		alpha := float32(srcPosFloat - float64(srcFrameIdx))

		if err := r.slide(srcFrameIdx); err != nil {
			if written == 0 {
				return 0, err
			}
			return written * channels, err
		}

		// Once the source is exhausted, stop as soon as the read position walks
		// past the last real frame.
		if r.eof && srcPosFloat >= float64(r.srcFramesRead) {
			if written == 0 {
				return 0, io.EOF
			}
			return written * channels, io.EOF
		}

		// Offset of y1 (the frame at srcFrameIdx). y0 sits one frame behind it,
		// which the synthetic history frame guarantees is in range.
		y1 := int(srcFrameIdx-r.inBase) * channels
		y0 := y1 - channels
		y2 := y1 + channels
		y3 := y2 + channels
		out := written * channels

		// Perform cubic interpolation
		for c := range channels {
			dst[out+c] = utils.CubicInterpolate(
				r.in[y0+c], r.in[y1+c], r.in[y2+c], r.in[y3+c], alpha,
			)
		}

		written++
		r.outFrame++
	}

	return written * channels, nil
}
