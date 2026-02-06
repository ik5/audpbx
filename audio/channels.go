// SPDX-License-Identifier: EPL-2.0

package audio

import (
	"fmt"
	"io"
)

// Channel represents a speaker position in a multi-channel audio stream.
// Values are bitmask flags following the Microsoft WAVEFORMATEXTENSIBLE
// channel mask specification. Multiple channels can be combined with bitwise OR.
type Channel uint32

// Channel types based on Audio standards
const (
	// ChannelFrontLeft is the front left speaker (FL).
	ChannelFrontLeft Channel = 1 << iota // 0x1
	// ChannelFrontRight is the front right speaker (FR).
	ChannelFrontRight // 0x2
	// ChannelFrontCenter is the front center speaker (FC).
	ChannelFrontCenter // 0x4
	// ChannelLowFrequency is the low-frequency effects channel (LFE/subwoofer).
	ChannelLowFrequency // 0x8
	// ChannelBackLeft is the back left (surround left) speaker (BL).
	ChannelBackLeft // 0x10
	// ChannelBackRight is the back right (surround right) speaker (BR).
	ChannelBackRight // 0x20
	// ChannelFrontLeftOfCenter is the front left of center speaker (FLC).
	ChannelFrontLeftOfCenter // 0x40
	// ChannelFrontRightOfCenter is the front right of center speaker (FRC).
	ChannelFrontRightOfCenter // 0x80
	// ChannelBackCenter is the back center (surround center) speaker (BC).
	ChannelBackCenter // 0x100
	// ChannelSideLeft is the side left speaker (SL).
	ChannelSideLeft // 0x200
	// ChannelSideRight is the side right speaker (SR).
	ChannelSideRight // 0x400
	// ChannelTopCenter is the top center (overhead) speaker (TC).
	ChannelTopCenter // 0x800
	// ChannelTopFrontLeft is the top front left speaker (TFL).
	ChannelTopFrontLeft // 0x1000
	// ChannelTopFrontCenter is the top front center speaker (TFC).
	ChannelTopFrontCenter // 0x2000
	// ChannelTopFrontRight is the top front right speaker (TFR).
	ChannelTopFrontRight // 0x4000
	// ChannelTopBackLeft is the top back left speaker (TBL).
	ChannelTopBackLeft // 0x8000
	// ChannelTopBackCenter is the top back center speaker (TBC).
	ChannelTopBackCenter // 0x10000
	// ChannelTopBackRight is the top back right speaker (TBR).
	ChannelTopBackRight // 0x20000
)

// maxDefinedChannels is the number of named speaker positions in the
// WAVEFORMATEXTENSIBLE specification.
const maxDefinedChannels = 18

// ChannelLayout represents a predefined combination of speaker positions.
type ChannelLayout = Channel

// Layout types based on channels
const (
	// LayoutMono is a single center channel.
	LayoutMono ChannelLayout = ChannelFrontCenter
	// LayoutStereo is standard left/right stereo.
	LayoutStereo ChannelLayout = ChannelFrontLeft | ChannelFrontRight
	// Layout2Point1 is stereo plus a subwoofer.
	Layout2Point1 ChannelLayout = LayoutStereo | ChannelLowFrequency
	// LayoutQuad is four-corner surround.
	LayoutQuad ChannelLayout = ChannelFrontLeft | ChannelFrontRight | ChannelBackLeft | ChannelBackRight
	// Layout5Point1 is 5.1 surround sound.
	Layout5Point1 ChannelLayout = ChannelFrontLeft | ChannelFrontRight | ChannelFrontCenter | ChannelLowFrequency | ChannelBackLeft | ChannelBackRight
	// Layout7Point1 is 7.1 surround sound (5.1 plus side speakers).
	Layout7Point1 ChannelLayout = Layout5Point1 | ChannelSideLeft | ChannelSideRight
)

// channelNames maps each Channel flag to its short human-readable name.
var channelNames = map[Channel]string{
	ChannelFrontLeft:          "FL",
	ChannelFrontRight:         "FR",
	ChannelFrontCenter:        "FC",
	ChannelLowFrequency:       "LFE",
	ChannelBackLeft:           "BL",
	ChannelBackRight:          "BR",
	ChannelFrontLeftOfCenter:  "FLC",
	ChannelFrontRightOfCenter: "FRC",
	ChannelBackCenter:         "BC",
	ChannelSideLeft:           "SL",
	ChannelSideRight:          "SR",
	ChannelTopCenter:          "TC",
	ChannelTopFrontLeft:       "TFL",
	ChannelTopFrontCenter:     "TFC",
	ChannelTopFrontRight:      "TFR",
	ChannelTopBackLeft:        "TBL",
	ChannelTopBackCenter:      "TBC",
	ChannelTopBackRight:       "TBR",
}

// String returns the short name of a single channel flag (e.g. "FL", "LFE").
// For combined layouts it returns a pipe-separated list (e.g. "FL|FR").
// For unnamed channel bits it returns "ch<N>".
func (c Channel) String() string {
	if c == 0 {
		return "none"
	}

	if name, ok := channelNames[c]; ok {
		return name
	}

	// Handle combined layouts or unnamed bits.
	result := ""
	for bit := Channel(1); bit != 0; bit <<= 1 {
		if c&bit == 0 {
			continue
		}
		if result != "" {
			result += "|"
		}
		if name, ok := channelNames[bit]; ok {
			result += name
		} else {
			result += fmt.Sprintf("ch%d", BitIndex(bit))
		}
	}
	return result
}

// ChannelCount returns the number of active channels (set bits) in the mask.
func (c Channel) ChannelCount() int {
	count := 0
	for v := c; v != 0; v &= v - 1 {
		count++
	}
	return count
}

// Contains reports whether the channel mask includes all of the given channels.
func (c Channel) Contains(ch Channel) bool {
	return c&ch == ch
}

// Index returns the zero-based interleave index of ch within the layout c.
// The index corresponds to the position of ch's samples in interleaved PCM data.
// Returns -1 if ch is not part of the layout.
func (c Channel) Index(ch Channel) int {
	if c&ch == 0 {
		return -1
	}
	idx := 0
	for bit := Channel(1); bit < ch; bit <<= 1 {
		if c&bit != 0 {
			idx++
		}
	}
	return idx
}

// BitIndex returns the zero-based bit position of a single-bit value.
func BitIndex(v Channel) int {
	n := 0
	for v >>= 1; v != 0; v >>= 1 {
		n++
	}
	return n
}

// ChannelManager extracts a single channel from a multi-channel Source,
// producing a new mono Source containing only the selected channel's samples.
//
// ChannelManager implements the Source interface.
type ChannelManager struct {
	src     Source
	channel Channel // the target channel to extract
	layout  Channel // the full channel layout of src
	chIndex int     // interleave index of the target channel
	buf     []float32
}

// NewChannelManager creates a ChannelManager that extracts the specified channel
// from src. The layout describes the channel assignment of src's interleaved
// samples (e.g. LayoutStereo, Layout5Point1). If layout is 0, a default layout
// is inferred from src.Channels():
//   - 1 channel:  LayoutMono
//   - 2 channels: LayoutStereo
//   - otherwise:  channels are assigned in WAVEFORMATEXTENSIBLE bit order
//
// Returns an error if the requested channel is not present in the layout,
// or if the layout's channel count does not match src.Channels().
func NewChannelManager(src Source, channel Channel, layout Channel) (*ChannelManager, error) {
	if layout == 0 {
		layout = defaultLayout(src.Channels())
	}

	if layout.ChannelCount() != src.Channels() {
		return nil, fmt.Errorf(
			"layout has %d channels but source has %d",
			layout.ChannelCount(), src.Channels(),
		)
	}

	idx := layout.Index(channel)
	if idx < 0 {
		return nil, fmt.Errorf(
			"channel %s is not present in layout %s",
			channel, layout,
		)
	}

	return &ChannelManager{
		src:     src,
		channel: channel,
		layout:  layout,
		chIndex: idx,
	}, nil
}

// defaultLayout returns a channel layout mask for the given number of channels
// using the standard WAVEFORMATEXTENSIBLE bit order.
func defaultLayout(channels int) Channel {
	switch channels {
	case 1:
		return LayoutMono
	case 2:
		return LayoutStereo
	case 3:
		return LayoutStereo | ChannelLowFrequency
	case 4:
		return LayoutQuad
	case 6:
		return Layout5Point1
	case 8:
		return Layout7Point1
	default:
		// Assign channels in bit order up to n channel.
		var layout Channel
		count := 0
		for bit := Channel(1); bit != 0 && count < channels; bit <<= 1 {
			layout |= bit
			count++
		}
		return layout
	}
}

// ReadSamples reads de-interleaved mono samples for the selected channel
// into dst. Returns the number of mono samples written.
func (cm *ChannelManager) ReadSamples(dst []float32) (int, error) {
	totalCh := cm.src.Channels()

	// We need totalCh interleaved samples for each output sample.
	// Allocate or grow the internal buffer as needed.
	needed := len(dst) * totalCh
	if len(cm.buf) < needed {
		cm.buf = make([]float32, needed)
	}

	n, err := cm.src.ReadSamples(cm.buf[:needed])
	if n == 0 {
		return 0, err
	}

	// n is the total number of float32 values read (interleaved).
	// Extract every totalCh-th sample starting at chIndex.
	frames := n / totalCh
	for i := range frames {
		dst[i] = cm.buf[i*totalCh+cm.chIndex]
	}

	if err != nil && err != io.EOF {
		return frames, err
	}
	if err == io.EOF {
		return frames, io.EOF
	}
	return frames, nil
}

// SampleRate returns the sample rate of the underlying source.
func (cm *ChannelManager) SampleRate() int {
	return cm.src.SampleRate()
}

// Channels returns 1, since the output is always a single extracted channel.
func (cm *ChannelManager) Channels() int {
	return 1
}

// BufSize returns the recommended buffer size.
func (cm *ChannelManager) BufSize() int {
	return cm.src.BufSize() / cm.src.Channels()
}

// Close releases the underlying source.
func (cm *ChannelManager) Close() error {
	return cm.src.Close()
}

// Layout returns the channel layout mask used by this ChannelManager.
func (cm *ChannelManager) Layout() Channel {
	return cm.layout
}

// ExtractedChannel returns which channel is being extracted.
func (cm *ChannelManager) ExtractedChannel() Channel {
	return cm.channel
}
