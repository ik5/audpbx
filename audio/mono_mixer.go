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

    // Calculate how many frames we can fit in dst
    maxFrames := len(dst)
    samplesNeeded := maxFrames * m.src.Channels()

    // Ensure tmp can hold enough data from src
    if len(m.tmp) < samplesNeeded {
        m.tmp = make([]float32, samplesNeeded)
    }

    // Only read what we need
    n, err := m.src.ReadSamples(m.tmp[:samplesNeeded])
    if n == 0 {
        return 0, err
    }
    frames := n / m.src.Channels()
    for f := range frames {
        sum := float32(0)

        for c := range m.src.Channels() {
            sum += m.tmp[f*m.src.Channels()+c]
        }

        dst[f] = sum / float32(m.src.Channels())
    }

    return frames, err
}
