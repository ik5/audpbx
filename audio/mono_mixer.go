// SPDX-License-Identifier: EPL-2.0

package audio

import "fmt"

// MonoMixer converts multi-channel audio to mono by averaging all channels.
//
// Mono audio has a single channel, while multi-channel audio (stereo, surround)
// has multiple independent channels. This mixer combines them into one channel
// by averaging, which is useful when:
//   - You need single-channel audio for voice processing
//   - Reducing file size (mono is smaller than stereo)
//   - Targeting mono playback systems (some phones, intercoms)
//
// How it works:
//   - Stereo (2 ch): output = (left + right) / 2
//   - 5.1 (6 ch): output = (FL + FR + FC + LFE + BL + BR) / 6
//
// Example mixing stereo to mono:
//   Input: L=[0.8, 0.4], R=[0.2, 0.6]
//   Output: [(0.8+0.2)/2, (0.4+0.6)/2] = [0.5, 0.5]
//
// The mixer is efficient and performs no allocations after initialization
// when using reasonable buffer sizes.
type MonoMixer struct {
    src      Source
    tmp      []float32
}

// NewMonoMixer creates a mixer that converts src to mono.
// If src is already mono, samples pass through unchanged.
func NewMonoMixer(src Source) *MonoMixer {
    return &MonoMixer{
        src: src,
        tmp: make([]float32, 4096),
    }
}

func (m *MonoMixer) SampleRate() int { return m.src.SampleRate() }
func (m *MonoMixer) Channels() int   { return 1 }
func (m *MonoMixer) BufSize() int    { return m.src.BufSize() }
func (m *MonoMixer) Close() error    {
	err := m.src.Close()
	if err != nil {
		return fmt.Errorf("%w", err)
	}

	return nil
}

func (m *MonoMixer) ReadSamples(dst []float32) (int, error) {
    if len(dst) == 0 {
        return 0, nil
    }
    if m.src.Channels() == 1 {
        // Pass-through: read mono directly
        return m.src.ReadSamples(dst)
    }

    channels := m.src.Channels()
    // Calculate how many frames we can fit in dst
    maxFrames := len(dst)
    samplesNeeded := maxFrames * channels

    // Grow tmp buffer if needed (but don't shrink to avoid thrashing)
    if cap(m.tmp) < samplesNeeded {
        // Allocate with some headroom to reduce future reallocations
        newCap := samplesNeeded
        if newCap < 8192 {
            newCap = 8192 // Reasonable minimum
        }
        m.tmp = make([]float32, newCap)
    } else if len(m.tmp) < samplesNeeded {
        // Re-slice to needed size without reallocation
        m.tmp = m.tmp[:samplesNeeded]
    }

    // Only read what we need
    n, err := m.src.ReadSamples(m.tmp[:samplesNeeded])
    if n == 0 {
        return 0, err
    }
    frames := n / channels

    // Optimize: cache division result
    invChannels := float32(1.0) / float32(channels)

    // Unrolled loop for common cases
    switch channels {
    case 2: // Stereo (most common)
        for f := range frames {
            idx := f << 1 // f * 2
            dst[f] = (m.tmp[idx] + m.tmp[idx+1]) * 0.5
        }
    case 4: // Quad
        for f := range frames {
            idx := f << 2 // f * 4
            sum := m.tmp[idx] + m.tmp[idx+1] + m.tmp[idx+2] + m.tmp[idx+3]
            dst[f] = sum * 0.25
        }
    default: // Generic path
        for f := range frames {
            sum := float32(0)
            baseIdx := f * channels
            for c := range channels {
                sum += m.tmp[baseIdx+c]
            }
            dst[f] = sum * invChannels
        }
    }

    return frames, err
}
