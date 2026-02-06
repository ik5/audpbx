// SPDX-License-Identifier: EPL-2.0

package audio

import (
	"testing"
)

// ---------------------------------------------------------------------------
// Channel type — bitmask values
// ---------------------------------------------------------------------------

func TestChannel_BitValues(t *testing.T) {
	tests := []struct {
		ch   Channel
		want uint32
		name string
	}{
		{ChannelFrontLeft, 0x1, "FL"},
		{ChannelFrontRight, 0x2, "FR"},
		{ChannelFrontCenter, 0x4, "FC"},
		{ChannelLowFrequency, 0x8, "LFE"},
		{ChannelBackLeft, 0x10, "BL"},
		{ChannelBackRight, 0x20, "BR"},
		{ChannelFrontLeftOfCenter, 0x40, "FLC"},
		{ChannelFrontRightOfCenter, 0x80, "FRC"},
		{ChannelBackCenter, 0x100, "BC"},
		{ChannelSideLeft, 0x200, "SL"},
		{ChannelSideRight, 0x400, "SR"},
		{ChannelTopCenter, 0x800, "TC"},
		{ChannelTopFrontLeft, 0x1000, "TFL"},
		{ChannelTopFrontCenter, 0x2000, "TFC"},
		{ChannelTopFrontRight, 0x4000, "TFR"},
		{ChannelTopBackLeft, 0x8000, "TBL"},
		{ChannelTopBackCenter, 0x10000, "TBC"},
		{ChannelTopBackRight, 0x20000, "TBR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if uint32(tt.ch) != tt.want {
				t.Errorf("%s: got 0x%x, want 0x%x", tt.name, uint32(tt.ch), tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Channel.String
// ---------------------------------------------------------------------------

func TestChannel_String(t *testing.T) {
	tests := []struct {
		name string
		ch   Channel
		want string
	}{
		{"zero", 0, "none"},
		{"single FL", ChannelFrontLeft, "FL"},
		{"single LFE", ChannelLowFrequency, "LFE"},
		{"single TBR", ChannelTopBackRight, "TBR"},
		{"stereo", LayoutStereo, "FL|FR"},
		{"5.1", Layout5Point1, "FL|FR|FC|LFE|BL|BR"},
		{"7.1", Layout7Point1, "FL|FR|FC|LFE|BL|BR|SL|SR"},
		{"unnamed bit 18", Channel(1 << 18), "ch18"},
		{"mixed named+unnamed", ChannelFrontLeft | Channel(1<<20), "FL|ch20"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ch.String(); got != tt.want {
				t.Errorf("Channel(%#x).String() = %q, want %q", uint32(tt.ch), got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Channel.ChannelCount
// ---------------------------------------------------------------------------

func TestChannel_ChannelCount(t *testing.T) {
	tests := []struct {
		name  string
		ch    Channel
		count int
	}{
		{"zero", 0, 0},
		{"single", ChannelFrontLeft, 1},
		{"stereo", LayoutStereo, 2},
		{"5.1", Layout5Point1, 6},
		{"7.1", Layout7Point1, 8},
		{"all 18", Channel((1 << 18) - 1), 18},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ch.ChannelCount(); got != tt.count {
				t.Errorf("ChannelCount() = %d, want %d", got, tt.count)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Channel.Contains
// ---------------------------------------------------------------------------

func TestChannel_Contains(t *testing.T) {
	tests := []struct {
		name   string
		layout Channel
		ch     Channel
		want   bool
	}{
		{"stereo contains FL", LayoutStereo, ChannelFrontLeft, true},
		{"stereo contains FL|FR", LayoutStereo, LayoutStereo, true},
		{"stereo not contains LFE", LayoutStereo, ChannelLowFrequency, false},
		{"5.1 contains LFE", Layout5Point1, ChannelLowFrequency, true},
		{"7.1 superset of 5.1", Layout7Point1, Layout5Point1, true},
		{"zero contains zero", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.layout.Contains(tt.ch); got != tt.want {
				t.Errorf("Contains() = %v, want %v", got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Channel.Index
// ---------------------------------------------------------------------------

func TestChannel_Index(t *testing.T) {
	tests := []struct {
		name   string
		layout Channel
		ch     Channel
		want   int
	}{
		{"FL in stereo", LayoutStereo, ChannelFrontLeft, 0},
		{"FR in stereo", LayoutStereo, ChannelFrontRight, 1},
		{"FC in mono", LayoutMono, ChannelFrontCenter, 0},
		{"FC in 5.1", Layout5Point1, ChannelFrontCenter, 2},
		{"LFE in 5.1", Layout5Point1, ChannelLowFrequency, 3},
		{"BL in 5.1", Layout5Point1, ChannelBackLeft, 4},
		{"BR in 5.1", Layout5Point1, ChannelBackRight, 5},
		{"SL in 7.1", Layout7Point1, ChannelSideLeft, 6},
		{"SR in 7.1", Layout7Point1, ChannelSideRight, 7},
		{"missing LFE in stereo", LayoutStereo, ChannelLowFrequency, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.layout.Index(tt.ch); got != tt.want {
				t.Errorf("Index(%s) in %s = %d, want %d", tt.ch, tt.layout, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// BitIndex
// ---------------------------------------------------------------------------

func TestBitIndex(t *testing.T) {
	for i := range 32 {
		v := Channel(1 << i)
		if got := BitIndex(v); got != i {
			t.Errorf("BitIndex(1<<%d) = %d, want %d", i, got, i)
		}
	}
}

// ---------------------------------------------------------------------------
// Layout constants
// ---------------------------------------------------------------------------

func TestLayouts(t *testing.T) {
	tests := []struct {
		name   string
		layout Channel
		count  int
	}{
		{"Mono", LayoutMono, 1},
		{"Stereo", LayoutStereo, 2},
		{"2.1", Layout2Point1, 3},
		{"Quad", LayoutQuad, 4},
		{"5.1", Layout5Point1, 6},
		{"7.1", Layout7Point1, 8},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.layout.ChannelCount(); got != tt.count {
				t.Errorf("%s: ChannelCount() = %d, want %d", tt.name, got, tt.count)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// defaultLayout
// ---------------------------------------------------------------------------

func TestDefaultLayout(t *testing.T) {
	tests := []struct {
		nch  int
		want Channel
	}{
		{1, LayoutMono},
		{2, LayoutStereo},
		{4, LayoutQuad},
		{6, Layout5Point1},
		{8, Layout7Point1},
	}

	for _, tt := range tests {
		t.Run(tt.want.String(), func(t *testing.T) {
			if got := defaultLayout(tt.nch); got != tt.want {
				t.Errorf("defaultLayout(%d) = %s, want %s", tt.nch, got, tt.want)
			}
		})
	}
}

func TestDefaultLayout_ArbitraryCount(t *testing.T) {
	// For non-standard channel counts, defaultLayout should still return
	// a layout whose popcount matches the requested channel count.
	for _, nch := range []int{5, 7, 9, 12, 18} {
		layout := defaultLayout(nch)
		if got := layout.ChannelCount(); got != nch {
			t.Errorf("defaultLayout(%d).ChannelCount() = %d", nch, got)
		}
	}
}

// ---------------------------------------------------------------------------
// ChannelManager.ReadSamples — stereo extraction
// ---------------------------------------------------------------------------

// stereoIdentityWaveform produces value = float32(channel) per frame,
// so left channel = 0.0 and right channel = 1.0 for every frame.
func stereoIdentityWaveform(sample int, channel int) float32 {
	return float32(channel)
}

// multiChannelWaveform returns a waveform func producing a unique value
// per (frame, channel) pair: value = float32(sample*nch + channel).
func multiChannelWaveform(nch int) func(int, int) float32 {
	return func(sample int, channel int) float32 {
		return float32(sample*nch + channel)
	}
}

// ---------------------------------------------------------------------------
// Benchmarks — b.Loop() (Go 1.24+)
// ---------------------------------------------------------------------------

func BenchmarkChannel_String_Single(b *testing.B) {
	ch := ChannelFrontLeft
	for b.Loop() {
		_ = ch.String()
	}
}

func BenchmarkChannel_String_Layout(b *testing.B) {
	ch := Layout7Point1
	for b.Loop() {
		_ = ch.String()
	}
}

func BenchmarkChannel_ChannelCount(b *testing.B) {
	ch := Layout7Point1
	for b.Loop() {
		_ = ch.ChannelCount()
	}
}

func BenchmarkChannel_Index(b *testing.B) {
	layout := Layout7Point1
	for b.Loop() {
		_ = layout.Index(ChannelSideRight)
	}
}

func BenchmarkChannel_Contains(b *testing.B) {
	layout := Layout7Point1
	for b.Loop() {
		_ = layout.Contains(Layout5Point1)
	}
}

func BenchmarkDefaultLayout(b *testing.B) {
	for b.Loop() {
		_ = defaultLayout(6)
	}
}
