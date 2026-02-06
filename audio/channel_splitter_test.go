// SPDX-License-Identifier: EPL-2.0

package audio

import (
	"errors"
	"io"
	"sync"
	"testing"
)

// ---------------------------------------------------------------------------
// SplitChannels — construction
// ---------------------------------------------------------------------------

func TestSplitChannels_Stereo(t *testing.T) {
	src := newMockSource(44100, 2, 64, stereoIdentityWaveform)
	channels, layout, err := SplitChannels(src, LayoutStereo)
	if err != nil {
		t.Fatalf("SplitChannels: %v", err)
	}
	if len(channels) != 2 {
		t.Fatalf("got %d channels, want 2", len(channels))
	}
	if layout != LayoutStereo {
		t.Errorf("layout = %s, want FL|FR", layout)
	}
	if channels[0].Channel() != ChannelFrontLeft {
		t.Errorf("channels[0] = %s, want FL", channels[0].Channel())
	}
	if channels[1].Channel() != ChannelFrontRight {
		t.Errorf("channels[1] = %s, want FR", channels[1].Channel())
	}
	for i := range channels {
		if channels[i].Channels() != 1 {
			t.Errorf("channels[%d].Channels() = %d, want 1", i, channels[i].Channels())
		}
		if channels[i].SampleRate() != 44100 {
			t.Errorf("channels[%d].SampleRate() = %d, want 44100", i, channels[i].SampleRate())
		}
	}
}

func TestSplitChannels_5Point1(t *testing.T) {
	src := newMockSource(48000, 6, 32, multiChannelWaveform(6))
	channels, layout, err := SplitChannels(src, Layout5Point1)
	if err != nil {
		t.Fatalf("SplitChannels: %v", err)
	}
	if len(channels) != 6 {
		t.Fatalf("got %d channels, want 6", len(channels))
	}
	if layout != Layout5Point1 {
		t.Errorf("layout = %s, want FL|FR|FC|LFE|BL|BR", layout)
	}

	wantChannels := []Channel{
		ChannelFrontLeft, ChannelFrontRight, ChannelFrontCenter,
		ChannelLowFrequency, ChannelBackLeft, ChannelBackRight,
	}
	for i, want := range wantChannels {
		if channels[i].Channel() != want {
			t.Errorf("channels[%d] = %s, want %s", i, channels[i].Channel(), want)
		}
	}
}

func TestSplitChannels_AutoLayout(t *testing.T) {
	src := newSilentSource(44100, 2, 100)
	channels, layout, err := SplitChannels(src, 0)
	if err != nil {
		t.Fatalf("SplitChannels: %v", err)
	}
	if layout != LayoutStereo {
		t.Errorf("auto layout = %s, want FL|FR", layout)
	}
	if len(channels) != 2 {
		t.Fatalf("got %d channels, want 2", len(channels))
	}
}

func TestSplitChannels_LayoutMismatch(t *testing.T) {
	src := newSilentSource(44100, 2, 100)
	_, _, err := SplitChannels(src, Layout5Point1) // src has 2 ch, layout wants 6
	if err == nil {
		t.Fatal("expected error for layout/source mismatch, got nil")
	}
}

// ---------------------------------------------------------------------------
// SplitChannels — reading stereo data
// ---------------------------------------------------------------------------

func TestSplitChannels_ReadStereoData(t *testing.T) {
	const frames = 64
	src := newMockSource(44100, 2, frames, stereoIdentityWaveform)
	channels, _, err := SplitChannels(src, LayoutStereo)
	if err != nil {
		t.Fatal(err)
	}

	leftBuf := make([]float32, frames)
	rightBuf := make([]float32, frames)

	// Read left channel.
	nL, _ := channels[0].ReadSamples(leftBuf)
	if nL != frames {
		t.Fatalf("left: got %d, want %d", nL, frames)
	}

	// Read right channel.
	nR, _ := channels[1].ReadSamples(rightBuf)
	if nR != frames {
		t.Fatalf("right: got %d, want %d", nR, frames)
	}

	for i := range frames {
		if leftBuf[i] != 0.0 {
			t.Errorf("left[%d] = %f, want 0.0", i, leftBuf[i])
		}
		if rightBuf[i] != 1.0 {
			t.Errorf("right[%d] = %f, want 1.0", i, rightBuf[i])
		}
	}
}

// ---------------------------------------------------------------------------
// SplitChannels — reading 5.1 data
// ---------------------------------------------------------------------------

func TestSplitChannels_Read5Point1Data(t *testing.T) {
	const frames = 32
	src := newMockSource(48000, 6, frames, multiChannelWaveform(6))
	channels, _, err := SplitChannels(src, Layout5Point1)
	if err != nil {
		t.Fatal(err)
	}

	for chIdx := range channels {
		t.Run(channels[chIdx].Channel().String(), func(t *testing.T) {
			buf := make([]float32, frames)
			n, _ := channels[chIdx].ReadSamples(buf)
			if n != frames {
				t.Fatalf("read %d, want %d", n, frames)
			}

			for f := range n {
				want := float32(f*6 + chIdx)
				if buf[f] != want {
					t.Errorf("frame %d: got %f, want %f", f, buf[f], want)
				}
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SplitChannels — EOF propagation
// ---------------------------------------------------------------------------

func TestSplitChannels_EOF(t *testing.T) {
	src := newSilentSource(44100, 2, 8)
	channels, _, err := SplitChannels(src, LayoutStereo)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]float32, 16) // larger than available frames

	n, err := channels[0].ReadSamples(buf)
	if n != 8 {
		t.Fatalf("got %d, want 8", n)
	}
	if err != io.EOF {
		t.Fatalf("err = %v, want io.EOF", err)
	}

	// Subsequent reads should also return EOF.
	n, err = channels[0].ReadSamples(buf)
	if n != 0 || err != io.EOF {
		t.Errorf("post-EOF: n=%d err=%v, want 0, io.EOF", n, err)
	}
}

// ---------------------------------------------------------------------------
// SplitChannels — Close behaviour
// ---------------------------------------------------------------------------

func TestSplitChannels_CloseOneChannel(t *testing.T) {
	const frames = 32
	src := newMockSource(44100, 2, frames, stereoIdentityWaveform)
	channels, _, err := SplitChannels(src, LayoutStereo)
	if err != nil {
		t.Fatal(err)
	}

	// Close left, right should still be readable.
	if err := channels[0].Close(); err != nil {
		t.Fatalf("Close left: %v", err)
	}

	buf := make([]float32, frames)
	n, _ := channels[1].ReadSamples(buf)
	if n != frames {
		t.Errorf("right after left closed: got %d, want %d", n, frames)
	}
}

func TestSplitChannels_ReadAfterClose(t *testing.T) {
	src := newSilentSource(44100, 2, 32)
	channels, _, err := SplitChannels(src, LayoutStereo)
	if err != nil {
		t.Fatal(err)
	}

	channels[0].Close()

	buf := make([]float32, 8)
	_, err = channels[0].ReadSamples(buf)
	if err != io.ErrClosedPipe {
		t.Errorf("read after close: err = %v, want io.ErrClosedPipe", err)
	}
}

func TestSplitChannels_CloseAllChannels(t *testing.T) {
	src := newSilentSource(44100, 2, 32)
	channels, _, err := SplitChannels(src, LayoutStereo)
	if err != nil {
		t.Fatal(err)
	}

	// Close both — underlying source should be closed (no panic).
	for i := range channels {
		if err := channels[i].Close(); err != nil {
			t.Errorf("Close channel %d: %v", i, err)
		}
	}
}

// ---------------------------------------------------------------------------
// SplitChannels — BufSize
// ---------------------------------------------------------------------------

func TestSplitChannels_BufSize(t *testing.T) {
	src := newSilentSource(44100, 2, 100) // BufSize() = 4096
	channels, _, err := SplitChannels(src, LayoutStereo)
	if err != nil {
		t.Fatal(err)
	}

	// BufSize should be src.BufSize()/channels = 4096/2 = 2048.
	if got := channels[0].BufSize(); got != 2048 {
		t.Errorf("BufSize() = %d, want 2048", got)
	}
}

// ---------------------------------------------------------------------------
// SplitChannels — chunked / partial reads
// ---------------------------------------------------------------------------

func TestSplitChannels_ChunkedRead(t *testing.T) {
	const totalFrames = 100
	src := newMockSource(44100, 2, totalFrames, stereoIdentityWaveform)
	channels, _, err := SplitChannels(src, LayoutStereo)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]float32, 10)
	var right []float32

	for {
		n, err := channels[1].ReadSamples(buf)
		right = append(right, buf[:n]...)
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	if len(right) != totalFrames {
		t.Fatalf("collected %d samples, want %d", len(right), totalFrames)
	}
	for i, v := range right {
		if v != 1.0 {
			t.Errorf("right[%d] = %f, want 1.0", i, v)
		}
	}
}

// ---------------------------------------------------------------------------
// SplitChannels — concurrent reads
// ---------------------------------------------------------------------------

func TestSplitChannels_ConcurrentRead(t *testing.T) {
	const frames = 256
	src := newMockSource(44100, 2, frames, stereoIdentityWaveform)
	channels, _, err := SplitChannels(src, LayoutStereo)
	if err != nil {
		t.Fatal(err)
	}

	type result struct {
		samples []float32
		err     error
	}

	leftCh := make(chan result, 1)
	rightCh := make(chan result, 1)

	readAll := func(cs *ChannelSource) result {
		var all []float32
		buf := make([]float32, 32)
		for {
			n, err := cs.ReadSamples(buf)
			all = append(all, buf[:n]...)
			if err == io.EOF {
				return result{all, nil}
			}
			if err != nil {
				return result{all, err}
			}
		}
	}

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		leftCh <- readAll(&channels[0])
	}()
	go func() {
		defer wg.Done()
		rightCh <- readAll(&channels[1])
	}()
	wg.Wait()
	close(leftCh)
	close(rightCh)

	lr := <-leftCh
	rr := <-rightCh

	if lr.err != nil {
		t.Fatalf("left read error: %v", lr.err)
	}
	if rr.err != nil {
		t.Fatalf("right read error: %v", rr.err)
	}

	if len(lr.samples) != frames {
		t.Errorf("left: got %d samples, want %d", len(lr.samples), frames)
	}
	if len(rr.samples) != frames {
		t.Errorf("right: got %d samples, want %d", len(rr.samples), frames)
	}

	for i, v := range lr.samples {
		if v != 0.0 {
			t.Errorf("left[%d] = %f, want 0.0", i, v)
		}
	}
	for i, v := range rr.samples {
		if v != 1.0 {
			t.Errorf("right[%d] = %f, want 1.0", i, v)
		}
	}
}

// ---------------------------------------------------------------------------
// SplitChannels — ring buffer growth
// ---------------------------------------------------------------------------

func TestSplitChannels_RingBufferGrowth(t *testing.T) {
	// Push >16384 frames (the initial ring size) through a single channel
	// to force ring buffer expansion.
	const frames = 20000
	src := newConstantSource(44100, 2, frames, 0.42)
	channels, _, err := SplitChannels(src, LayoutStereo)
	if err != nil {
		t.Fatal(err)
	}

	buf := make([]float32, frames)
	n, _ := channels[0].ReadSamples(buf)
	if n != frames {
		t.Fatalf("got %d, want %d", n, frames)
	}

	for i := range n {
		if buf[i] != 0.42 {
			t.Errorf("sample[%d] = %f, want 0.42", i, buf[i])
		}
	}
}

// ---------------------------------------------------------------------------
// ProcessChannels
// ---------------------------------------------------------------------------

// identityProcessFunc returns the source unchanged.
func identityProcessFunc(src Source, _ Channel) (Source, error) {
	return src, nil
}

func TestProcessChannels_Identity(t *testing.T) {
	const frames = 32
	src := newMockSource(44100, 2, frames, stereoIdentityWaveform)

	result, err := ProcessChannels(src, LayoutStereo, identityProcessFunc)
	if err != nil {
		t.Fatalf("ProcessChannels: %v", err)
	}
	if len(result) != 2 {
		t.Fatalf("got %d processed channels, want 2", len(result))
	}

	if result[0].Channel != ChannelFrontLeft {
		t.Errorf("result[0].Channel = %s, want FL", result[0].Channel)
	}
	if result[1].Channel != ChannelFrontRight {
		t.Errorf("result[1].Channel = %s, want FR", result[1].Channel)
	}

	// Read and verify data.
	buf := make([]float32, frames)
	n, _ := result[0].Source.ReadSamples(buf)
	if n != frames {
		t.Fatalf("left read: got %d, want %d", n, frames)
	}
	for i := range n {
		if buf[i] != 0.0 {
			t.Errorf("left[%d] = %f, want 0.0", i, buf[i])
		}
	}

	n, _ = result[1].Source.ReadSamples(buf)
	if n != frames {
		t.Fatalf("right read: got %d, want %d", n, frames)
	}
	for i := range n {
		if buf[i] != 1.0 {
			t.Errorf("right[%d] = %f, want 1.0", i, buf[i])
		}
	}
}

func TestProcessChannels_ErrorRollback(t *testing.T) {
	src := newSilentSource(44100, 6, 32)
	callCount := 0
	errProcessFunc := func(src Source, ch Channel) (Source, error) {
		callCount++
		if callCount == 3 {
			return nil, errors.New("processing failed")
		}
		return src, nil
	}

	_, err := ProcessChannels(src, Layout5Point1, errProcessFunc)
	if err == nil {
		t.Fatal("expected error from ProcessChannels, got nil")
	}
}

func TestProcessChannels_LayoutMismatch(t *testing.T) {
	src := newSilentSource(44100, 2, 32)
	_, err := ProcessChannels(src, Layout5Point1, identityProcessFunc)
	if err == nil {
		t.Fatal("expected error for layout mismatch, got nil")
	}
}

// ---------------------------------------------------------------------------
// Benchmarks — b.Loop() (Go 1.24+)
// ---------------------------------------------------------------------------

func BenchmarkSplitChannels_Stereo(b *testing.B) {
	const frames = 4096
	buf := make([]float32, frames)

	b.ReportAllocs()
	for b.Loop() {
		src := newSilentSource(44100, 2, frames)
		channels, _, err := SplitChannels(src, LayoutStereo)
		if err != nil {
			b.Fatal(err)
		}
		channels[0].ReadSamples(buf)
		channels[1].ReadSamples(buf)
	}
}

func BenchmarkSplitChannels_5Point1(b *testing.B) {
	const frames = 4096
	buf := make([]float32, frames)

	b.ReportAllocs()
	for b.Loop() {
		src := newSilentSource(48000, 6, frames)
		channels, _, err := SplitChannels(src, Layout5Point1)
		if err != nil {
			b.Fatal(err)
		}
		for i := range channels {
			channels[i].ReadSamples(buf)
		}
	}
}

func BenchmarkSplitChannels_ConcurrentRead(b *testing.B) {
	const frames = 4096

	b.ReportAllocs()
	for b.Loop() {
		src := newSilentSource(44100, 2, frames)
		channels, _, err := SplitChannels(src, LayoutStereo)
		if err != nil {
			b.Fatal(err)
		}

		var wg sync.WaitGroup
		wg.Add(2)
		for i := range 2 {
			go func(ch *ChannelSource) {
				defer wg.Done()
				buf := make([]float32, frames)
				ch.ReadSamples(buf)
			}(&channels[i])
		}
		wg.Wait()
	}
}

func BenchmarkProcessChannels_Stereo(b *testing.B) {
	const frames = 4096

	b.ReportAllocs()
	for b.Loop() {
		src := newSilentSource(44100, 2, frames)
		result, err := ProcessChannels(src, LayoutStereo, identityProcessFunc)
		if err != nil {
			b.Fatal(err)
		}
		buf := make([]float32, frames)
		for i := range result {
			result[i].Source.ReadSamples(buf)
		}
	}
}
