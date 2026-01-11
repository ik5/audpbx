package audio

import "fmt"

type MonoMixer struct {
    src      Source
    tmp      []float32
}

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
