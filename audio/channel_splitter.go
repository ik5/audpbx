// SPDX-License-Identifier: EPL-2.0

package audio

import (
	"fmt"
	"io"
	"sync"
)

// ChannelSource is a mono Source representing a single extracted channel
// from a SplitChannels call. It implements the Source interface.
type ChannelSource struct {
	splitter   *channelSplitter
	chIndex    int     // this channel's index in the interleaved data
	channel    Channel // which speaker position this represents
	sampleRate int
	bufSize    int

	// Per-channel ring buffer for decoupled reading.
	mu     sync.Mutex
	ring   []float32
	rPos   int // read position
	wPos   int // write position
	count  int // samples available
	closed bool
	eof    bool
}

// Channel returns which speaker position this source represents.
func (cs *ChannelSource) Channel() Channel {
	return cs.channel
}

// SampleRate returns the sample rate.
func (cs *ChannelSource) SampleRate() int {
	return cs.sampleRate
}

// Channels returns 1 (mono).
func (cs *ChannelSource) Channels() int {
	return 1
}

// BufSize returns the recommended buffer size for this channel.
func (cs *ChannelSource) BufSize() int {
	if cs.bufSize > 0 {
		return cs.bufSize
	}
	return 4096
}

// Close marks this channel source as closed.
func (cs *ChannelSource) Close() error {
	cs.mu.Lock()
	cs.closed = true
	cs.mu.Unlock()
	return cs.splitter.closeIfDone()
}

// ReadSamples reads mono samples for this channel into dst.
func (cs *ChannelSource) ReadSamples(dst []float32) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}

	cs.mu.Lock()
	if cs.closed {
		cs.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	cs.mu.Unlock()

	// Request the splitter to fill our ring buffer if needed.
	cs.splitter.fill(len(dst))

	cs.mu.Lock()
	defer cs.mu.Unlock()

	if cs.count == 0 && cs.eof {
		return 0, io.EOF
	}

	n := cs.drainInto(dst)
	if cs.count == 0 && cs.eof {
		return n, io.EOF
	}
	return n, nil
}

// drainInto copies up to len(dst) samples from the ring buffer into dst.
// Returns the number of samples copied.
// Caller must hold cs.mu.
func (cs *ChannelSource) drainInto(dst []float32) int {
	if cs.count == 0 {
		return 0
	}
	n := min(cs.count, len(dst))

	for i := range n {
		dst[i] = cs.ring[cs.rPos]
		cs.rPos = (cs.rPos + 1) % len(cs.ring)
	}
	cs.count -= n
	return n
}

// push appends a sample to this channel's ring buffer.
// Caller must hold cs.mu.
func (cs *ChannelSource) push(sample float32) {
	if cs.ring == nil {
		cs.ring = make([]float32, 16384)
	}
	// Grow if full.
	if cs.count == len(cs.ring) {
		newRing := make([]float32, len(cs.ring)*2)
		for i := 0; i < cs.count; i++ {
			newRing[i] = cs.ring[(cs.rPos+i)%len(cs.ring)]
		}
		cs.rPos = 0
		cs.wPos = cs.count
		cs.ring = newRing
	}
	cs.ring[cs.wPos] = sample
	cs.wPos = (cs.wPos + 1) % len(cs.ring)
	cs.count++
}

// channelSplitter reads from the shared source and fans samples out to
// individual ChannelSource ring buffers.
type channelSplitter struct {
	src     Source
	totalCh int
	nOut    int
	outputs []ChannelSource

	mu       sync.Mutex
	buf      []float32
	eof      bool
	srcClose sync.Once
}

// fill reads from the underlying source until each output channel has at
// least minPerCh samples available, or EOF is reached.
func (s *channelSplitter) fill(minPerCh int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.eof {
		return
	}

	if s.buf == nil {
		s.buf = make([]float32, s.totalCh*4096)
	}

	for {
		// Check if all non-closed outputs have enough.
		allSatisfied := true
		for i := range s.outputs {
			out := &s.outputs[i]
			out.mu.Lock()
			if !out.closed && out.count < minPerCh {
				allSatisfied = false
			}
			out.mu.Unlock()
		}
		if allSatisfied {
			return
		}

		n, err := s.src.ReadSamples(s.buf)
		if n > 0 {
			frames := n / s.totalCh
			for f := range frames {
				for i := range s.outputs {
					out := &s.outputs[i]
					out.mu.Lock()
					if !out.closed {
						// Use chIndex to read from correct position in interleaved data
						out.push(s.buf[f*s.totalCh+out.chIndex])
					}
					out.mu.Unlock()
				}
			}
		}

		if err == io.EOF || n == 0 {
			s.eof = true
			for i := range s.outputs {
				out := &s.outputs[i]
				out.mu.Lock()
				out.eof = true
				out.mu.Unlock()
			}
			return
		}
		if err != nil {
			s.eof = true
			for i := range s.outputs {
				out := &s.outputs[i]
				out.mu.Lock()
				out.eof = true
				out.mu.Unlock()
			}
			return
		}
	}
}

func (s *channelSplitter) closeIfDone() error {
	for i := range s.outputs {
		s.outputs[i].mu.Lock()
		closed := s.outputs[i].closed
		s.outputs[i].mu.Unlock()
		if !closed {
			return nil
		}
	}

	var err error
	s.srcClose.Do(func() {
		err = s.src.Close()
	})
	return err
}

// ExtractChannels extracts only the specified channels from src into separate mono Sources.
//
// Channels are independent audio streams in multi-channel audio. For example:
//   - Stereo: 2 channels (left speaker, right speaker)
//   - 5.1 surround: 6 channels (front-left, front-right, center, subwoofer, back-left, back-right)
//
// This function separates the interleaved multi-channel audio into individual mono
// sources for each requested channel, allowing you to process each channel independently.
//
// The returned slice contains only the requested channels, in the order specified in 'wanted'.
// The layout describes the channel mapping of src; if 0, a default is inferred
// from src.Channels(). Each channel in 'wanted' must be present in the layout.
//
// All returned Sources share the underlying src. Channels not in 'wanted' are
// never created or buffered, preventing memory waste - only the channels you
// request are extracted and stored.
//
// Example - extract only front speakers from 5.1 surround:
//
//	channels, err := audio.ExtractChannels(src5_1, audio.Layout5Point1,
//	    audio.ChannelFrontLeft,
//	    audio.ChannelFrontRight,
//	    audio.ChannelBackRight)
//	// Returns exactly 3 channels: FL, FR, BR
//	fl := &channels[0]  // FL
//	fr := &channels[1]  // FR
//	br := &channels[2]  // BR
func ExtractChannels(src Source, layout Channel, wanted ...Channel) ([]ChannelSource, error) {
	if layout == 0 {
		layout = defaultLayout(src.Channels())
	}

	if layout.ChannelCount() != src.Channels() {
		return nil, fmt.Errorf(
			"layout has %d channels but source has %d",
			layout.ChannelCount(), src.Channels(),
		)
	}

	if len(wanted) == 0 {
		return nil, fmt.Errorf("no channels specified")
	}

	totalCh := src.Channels()

	// Build a map of wanted channels for quick lookup
	wantedMap := make(map[Channel]bool, len(wanted))
	for _, ch := range wanted {
		if !layout.Contains(ch) {
			return nil, fmt.Errorf(
				"channel %s is not present in layout %s",
				ch, layout,
			)
		}
		wantedMap[ch] = true
	}

	// Create output sources in the order specified by 'wanted'
	sources := make([]ChannelSource, len(wanted))
	for i, ch := range wanted {
		chIndex := layout.Index(ch)
		sources[i] = ChannelSource{
			chIndex:    chIndex,
			channel:    ch,
			sampleRate: src.SampleRate(),
			bufSize:    src.BufSize() / totalCh,
		}
	}

	splitter := &channelSplitter{
		src:     src,
		totalCh: totalCh,
		nOut:    len(sources),
		outputs: sources,
	}

	// Link each source to the splitter
	for i := range sources {
		sources[i].splitter = splitter
	}

	return sources, nil
}

// SplitChannels extracts every channel from src into separate mono Sources.
// The returned slice is ordered by the channel's bit position (lowest bit first),
// matching the interleave order of the PCM data.
//
// Each returned Source can be read independently.
// The layout describes the channel mapping of src; if 0, a default is inferred
// from src.Channels().
//
// All returned Sources share the underlying src. Reading is coordinated
// internally — each call to ReadSamples on any output Source triggers a read
// from src and fans the data out to all channels.
//
// Example:
//
//	channels, layout, err := audio.SplitChannels(stereoSrc, audio.LayoutStereo)
//	left  := channels[0] // FL
//	right := channels[1] // FR
func SplitChannels(src Source, layout Channel) ([]ChannelSource, Channel, error) {
	if layout == 0 {
		layout = defaultLayout(src.Channels())
	}

	if layout.ChannelCount() != src.Channels() {
		return nil, 0, fmt.Errorf(
			"layout has %d channels but source has %d",
			layout.ChannelCount(), src.Channels(),
		)
	}

	totalCh := src.Channels()

	// Pre-allocate based on channel count
	sources := make([]ChannelSource, 0, totalCh)

	// Enumerate which channels are present, in bit order.
	for bit := Channel(1); bit != 0; bit <<= 1 {
		if layout&bit != 0 {
			sources = append(sources, ChannelSource{
				chIndex:    len(sources),
				channel:    bit,
				sampleRate: src.SampleRate(),
				bufSize:    src.BufSize() / totalCh,
			})
		}
	}

	splitter := &channelSplitter{
		src:     src,
		totalCh: totalCh,
		nOut:    len(sources),
		outputs: sources,
	}

	// Link each source to the splitter
	for i := range sources {
		sources[i].splitter = splitter
	}

	return sources, layout, nil
}

// ProcessedChannel pairs a processed mono Source with the channel it came from.
type ProcessedChannel struct {
	Source  Source
	Channel Channel
}

// ChannelProcessFunc is a function that takes a mono Source (a single channel)
// along with its channel identity, and returns a processed Source.
// The returned Source should be mono.
type ChannelProcessFunc func(src Source, ch Channel) (Source, error)

// ProcessChannels splits src into individual channels, applies fn to each,
// and returns the processed mono Sources. The layout describes the channel
// mapping; pass 0 to auto-detect from src.Channels().
//
// Example — resample each channel independently:
//
//	processed, err := audio.ProcessChannels(src5_1, audio.Layout5Point1,
//	    func(mono audio.Source, ch audio.Channel) (audio.Source, error) {
//	        return audio.NewResampler(mono, 8000), nil
//	    },
//	)
//	// processed[0] = FL resampled to 8kHz
//	// processed[1] = FR resampled to 8kHz
//	// ...
func ProcessChannels(src Source, layout Channel, fn ChannelProcessFunc) ([]ProcessedChannel, error) {
	channels, resolvedLayout, err := SplitChannels(src, layout)
	if err != nil {
		return nil, err
	}

	result := make([]ProcessedChannel, len(channels))
	for i := range channels {
		processed, err := fn(&channels[i], channels[i].channel)
		if err != nil {
			// Close already-processed sources on failure.
			for j := range i {
				result[j].Source.Close()
			}
			return nil, fmt.Errorf("processing channel %s: %w", channels[i].channel, err)
		}
		result[i] = ProcessedChannel{
			Source:  processed,
			Channel: channels[i].channel,
		}
	}

	_ = resolvedLayout // available if needed later
	return result, nil
}
