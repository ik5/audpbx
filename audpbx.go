// SPDX-License-Identifier: EPL-2.0

package audpbx

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/formats/wav"
)

// SplitToMonoSources splits multi-channel audio into separate mono sources.
//
// Multi-channel audio (like stereo or 5.1 surround) has multiple independent audio
// streams (channels) mixed together. This function separates them into individual
// mono (single-channel) sources that can be processed independently.
//
// For example, stereo audio has two channels (left and right). After splitting:
//   - sources[0] = left channel only
//   - sources[1] = right channel only
//
// Returns sources in channel bit order (FL, FR, FC, LFE, BL, BR for 5.1 surround).
// Equivalent to pydub's split_to_mono function.
func SplitToMonoSources(src audio.Source) ([]audio.Source, error) {
	channels, _, err := audio.SplitChannels(src, 0) // 0 = auto-detect layout
	if err != nil {
		return nil, err
	}

	result := make([]audio.Source, len(channels))
	for i := range channels {
		result[i] = &channels[i]
	}
	return result, nil
}

// ProcessAndSaveChannel resamples a mono channel and saves it to a WAV file.
//
// Resampling changes the sample rate of audio. Sample rate is how many times per
// second the audio is measured (sampled). Common rates are:
//   - 44100 Hz (CD quality)
//   - 48000 Hz (professional audio)
//   - 16000 Hz (voice calls)
//   - 8000 Hz (telephone quality)
//
// For example, converting 44100 Hz to 16000 Hz reduces file size and is suitable
// for voice applications like telephony.
//
// The source must be mono (single channel). For multi-channel audio, use
// SplitToMonoSources first to separate channels, then process each independently.
func ProcessAndSaveChannel(mono audio.Source, rate int, path string) error {
	if mono.Channels() != 1 {
		return fmt.Errorf("source must be mono (1 channel), got %d channels", mono.Channels())
	}

	// Resample if needed
	var src audio.Source = mono
	if mono.SampleRate() != rate {
		src = audio.NewResampler(mono, rate)
	}

	// Read all samples
	samples, err := readAllSamplesAsInt16(src)
	if err != nil {
		return fmt.Errorf("failed to read samples: %w", err)
	}

	// Create output file
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer f.Close()

	// Write as WAV
	if err := wav.WriteWAV16(f, rate, samples); err != nil {
		return fmt.Errorf("failed to write WAV: %w", err)
	}

	return nil
}

// ProcessAndSaveChannelWriter resamples a mono channel and saves it to a writer as WAV.
func ProcessAndSaveChannelWriter(mono audio.Source, rate int, w io.Writer) error {
	if mono.Channels() != 1 {
		return fmt.Errorf("source must be mono (1 channel), got %d channels", mono.Channels())
	}

	// Resample if needed
	var src audio.Source = mono
	if mono.SampleRate() != rate {
		src = audio.NewResampler(mono, rate)
	}

	// Read all samples
	samples, err := readAllSamplesAsInt16(src)
	if err != nil {
		return fmt.Errorf("failed to read samples: %w", err)
	}

	// Write as WAV
	if err := wav.WriteWAV16(w, rate, samples); err != nil {
		return fmt.Errorf("failed to write WAV: %w", err)
	}

	return nil
}

// ProcessingError represents an error that occurred during channel processing.
type ProcessingError struct {
	Channel  audio.Channel
	Activity string // "extract", "resample", "process", "save", etc.
	Err      error
}

func (e *ProcessingError) Error() string {
	return fmt.Sprintf("%s error on channel %s: %v", e.Activity, e.Channel, e.Err)
}

// ProcessingResult contains the results of channel processing.
type ProcessingResult struct {
	Errors []ProcessingError
}

// HasErrors returns true if any errors occurred during processing.
func (r *ProcessingResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// ChannelOption configures channel processing behavior.
type ChannelOption func(*channelProcessor)

// channelProcessor holds configuration for processing channels.
type channelProcessor struct {
	// Channel selection
	selectedChannels map[audio.Channel]bool
	selectAll        bool

	// Resampling
	resampleFunc func(ch audio.Channel) int

	// Custom processing
	processorFunc func(ch audio.Channel, src audio.Source) (audio.Source, error)

	// Output
	saveToFileFunc   func(ch audio.Channel) string
	saveToWriterFunc func(ch audio.Channel) io.Writer

	// Channel mapping
	channelMapping map[audio.Channel]audio.Channel

	// Mixdown
	mixdown      bool
	mixdownPath  string
	mixdownWrite io.Writer

	// Normalization
	normalizeFunc func(ch audio.Channel) bool

	// Gain adjustment
	gainFunc func(ch audio.Channel) float32

	// Concurrency
	maxConcurrent int

	// Error handling
	errorHandler func(ch audio.Channel, activity string, err error) error

	// Progress
	progressCallback func(ch audio.Channel, progress float64)

	// Buffer size
	bufferSize int
}

// WithChannels specifies which channels to process.
// If not called, all channels will be processed.
func WithChannels(channels ...audio.Channel) ChannelOption {
	return func(p *channelProcessor) {
		if p.selectedChannels == nil {
			p.selectedChannels = make(map[audio.Channel]bool)
		}
		for _, ch := range channels {
			p.selectedChannels[ch] = true
		}
		p.selectAll = false
	}
}

// WithResample configures per-channel resampling.
//
// Resampling changes the sample rate (number of samples per second) of audio.
// This is useful when you need audio at a specific sample rate for your application.
// Higher sample rates preserve more audio detail but use more storage and bandwidth.
//
// Example use cases:
//   - Converting CD audio (44.1 kHz) to telephone quality (8 kHz) for voice calls
//   - Upsampling to match professional equipment requirements (48 kHz, 96 kHz)
//   - Reducing file size by lowering sample rate for streaming
//
// The function receives a channel and returns the desired sample rate in Hz.
// Return 0 or negative to skip resampling for that channel.
func WithResample(rateFunc func(ch audio.Channel) int) ChannelOption {
	return func(p *channelProcessor) {
		p.resampleFunc = rateFunc
	}
}

// WithSaveToFile configures saving channels to files.
// The function receives a channel and returns the file path.
// Return empty string to skip saving that channel.
func WithSaveToFile(pathFunc func(ch audio.Channel) string) ChannelOption {
	return func(p *channelProcessor) {
		p.saveToFileFunc = pathFunc
	}
}

// WithSaveToWriter configures saving channels to writers.
// The function receives a channel and returns an io.Writer.
// Return nil to skip saving that channel.
func WithSaveToWriter(writerFunc func(ch audio.Channel) io.Writer) ChannelOption {
	return func(p *channelProcessor) {
		p.saveToWriterFunc = writerFunc
	}
}

// WithProcessor applies custom processing to each channel.
// The function receives a channel and its source, and returns a processed source.
func WithProcessor(procFunc func(ch audio.Channel, src audio.Source) (audio.Source, error)) ChannelOption {
	return func(p *channelProcessor) {
		p.processorFunc = procFunc
	}
}

// WithChannelMapping remaps channels during processing.
//
// Channel mapping allows you to swap or redirect channels to different positions.
// This is useful when:
//   - Correcting incorrectly labeled channels
//   - Swapping left and right channels
//   - Routing channels to different outputs
//
// Example - swap left and right channels:
//
//	mapping := map[audio.Channel]audio.Channel{
//	    audio.ChannelFrontLeft:  audio.ChannelFrontRight,
//	    audio.ChannelFrontRight: audio.ChannelFrontLeft,
//	}
//	result, _ := ProcessChannels(src, audio.LayoutStereo,
//	    WithChannelMapping(mapping),
//	    WithSaveToFile(...))
//
// Example - route center channel to front-left:
//
//	mapping := map[audio.Channel]audio.Channel{
//	    audio.ChannelFrontCenter: audio.ChannelFrontLeft,
//	}
func WithChannelMapping(mapping map[audio.Channel]audio.Channel) ChannelOption {
	return func(p *channelProcessor) {
		p.channelMapping = mapping
	}
}

// WithMixdown mixes all channels into a single mono channel and saves it.
//
// Mixdown (also called "downmixing") combines multiple audio channels into a single
// mono (one-channel) output. This is done by averaging the audio from all channels.
//
// For example, mixing stereo (2 channels) to mono:
//   - Left channel: [0.5, 0.3, 0.8]
//   - Right channel: [0.4, 0.6, 0.2]
//   - Mixed mono: [(0.5+0.4)/2, (0.3+0.6)/2, (0.8+0.2)/2] = [0.45, 0.45, 0.5]
//
// Common uses:
//   - Creating a single-speaker version from stereo/surround audio
//   - Reducing file size by converting to mono where stereo isn't needed
//   - Preparing audio for systems that only support mono playback
//
// Specify a file path to save the mixed audio. The path can be empty if using
// WithMixdownWriter instead.
func WithMixdown(pathOrNil string) ChannelOption {
	return func(p *channelProcessor) {
		p.mixdown = true
		p.mixdownPath = pathOrNil
	}
}

// WithMixdownWriter mixes all channels into a single mono channel and saves to writer.
func WithMixdownWriter(w io.Writer) ChannelOption {
	return func(p *channelProcessor) {
		p.mixdown = true
		p.mixdownWrite = w
	}
}

// WithNormalize configures per-channel normalization.
//
// Normalization adjusts the audio volume so it uses the full available dynamic range
// without clipping (distortion from being too loud). It finds the loudest point in
// the audio and scales everything proportionally so that point reaches (but doesn't
// exceed) the maximum safe level.
//
// Example - before normalization:
//   - Peak level: 0.3 (only using 30% of available volume range)
//   - Sample values: [0.1, 0.3, -0.2, 0.15]
//
// After normalization (scaled to use ~95% of range to prevent clipping):
//   - Peak level: 0.95 (using 95% of available range)
//   - Sample values: [0.317, 0.95, -0.633, 0.475]
//
// Common uses:
//   - Making quiet recordings louder without manual adjustment
//   - Ensuring consistent volume levels across multiple audio files
//   - Preventing clipping when applying other effects like gain
//
// Note: Normalization requires reading the entire audio into memory to find the peak.
//
// The function receives a channel and returns true to normalize that channel.
func WithNormalize(normalizeFunc func(ch audio.Channel) bool) ChannelOption {
	return func(p *channelProcessor) {
		p.normalizeFunc = normalizeFunc
	}
}

// WithGain applies gain adjustment to channels.
//
// Gain is a volume multiplier applied to audio. It makes the audio louder or quieter
// by multiplying every sample by the gain value.
//
// Gain values:
//   - 1.0 = no change (original volume)
//   - 2.0 = double the volume (+6 dB)
//   - 0.5 = half the volume (-6 dB)
//   - 1.5 = 50% louder
//   - 0.1 = 10% of original (very quiet)
//
// Example with gain of 2.0:
//   - Original: [0.1, 0.3, -0.2, 0.5]
//   - After gain: [0.2, 0.6, -0.4, 1.0]
//
// Common uses:
//   - Boosting quiet channels (gain > 1.0)
//   - Reducing loud channels to prevent clipping (gain < 1.0)
//   - Balancing volume between different channels
//   - Applying consistent volume adjustments
//
// Warning: Gain values above 1.0 can cause clipping (distortion) if the audio
// becomes too loud. Consider using WithNormalize first, or keep gain below the
// headroom available in your audio.
//
// The function receives a channel and returns a gain multiplier.
func WithGain(gainFunc func(ch audio.Channel) float32) ChannelOption {
	return func(p *channelProcessor) {
		p.gainFunc = gainFunc
	}
}

// WithConcurrency sets the maximum number of channels to process concurrently.
// Default is 1 (sequential processing). Use 0 for unlimited concurrency.
func WithConcurrency(maxConcurrent int) ChannelOption {
	return func(p *channelProcessor) {
		p.maxConcurrent = maxConcurrent
	}
}

// WithErrorHandler sets a custom error handler.
// The handler receives the channel, activity name, and error.
// Return nil to continue processing other channels, or an error to stop.
func WithErrorHandler(handler func(ch audio.Channel, activity string, err error) error) ChannelOption {
	return func(p *channelProcessor) {
		p.errorHandler = handler
	}
}

// WithProgress sets a progress callback.
// Called periodically with channel and progress (0.0 to 1.0).
func WithProgress(callback func(ch audio.Channel, progress float64)) ChannelOption {
	return func(p *channelProcessor) {
		p.progressCallback = callback
	}
}

// WithBufferSize sets the buffer size for reading/writing audio data.
// Must be at least 256 samples. Returns error if too small or too large.
func WithBufferSize(size int) ChannelOption {
	return func(p *channelProcessor) {
		p.bufferSize = size
	}
}

// ProcessChannels splits the source and applies the specified options to each channel.
//
// This is the main high-level function for advanced audio processing. It provides
// a flexible way to extract, process, and save individual channels from multi-channel
// audio with options for resampling, gain adjustment, normalization, and more.
//
// Basic workflow:
//  1. Split the source into individual channels based on the layout
//  2. Apply processing to each channel (resample, gain, normalize, etc.)
//  3. Save processed channels to files or writers
//
// Example - Extract and save stereo channels:
//
//	result, err := ProcessChannels(src, audio.LayoutStereo,
//	    WithSaveToFile(func(ch audio.Channel) string {
//	        return fmt.Sprintf("%s.wav", ch)
//	    }))
//
// Example - Resample specific channels:
//
//	result, err := ProcessChannels(src, audio.Layout5Point1,
//	    WithChannels(audio.ChannelFrontLeft, audio.ChannelFrontRight),
//	    WithResample(func(ch audio.Channel) int { return 48000 }),
//	    WithSaveToFile(func(ch audio.Channel) string {
//	        return fmt.Sprintf("output_%s.wav", ch)
//	    }))
//
// Available options:
//   - WithChannels: Select specific channels to process
//   - WithResample: Change sample rate
//   - WithGain: Adjust volume
//   - WithNormalize: Auto-adjust volume to maximize loudness without clipping
//   - WithMixdown: Combine all channels to mono
//   - WithSaveToFile/WithSaveToWriter: Specify where to save output
//   - WithConcurrency: Process multiple channels simultaneously
//   - WithErrorHandler: Handle errors during processing
//   - WithChannelMapping: Remap/swap channels
//   - WithProcessor: Apply custom processing
//
// Returns a result containing any errors that occurred during processing.
// Individual channel errors don't stop processing of other channels.
func ProcessChannels(src audio.Source, layout audio.Channel, opts ...ChannelOption) (*ProcessingResult, error) {
	// Apply options
	proc := &channelProcessor{
		selectAll:     true,
		maxConcurrent: 1,
		bufferSize:    4096,
	}
	for _, opt := range opts {
		opt(proc)
	}

	// Validate buffer size
	if proc.bufferSize < 256 {
		return nil, fmt.Errorf("buffer size too small: %d (minimum 256)", proc.bufferSize)
	}
	if proc.bufferSize > 1024*1024 {
		return nil, fmt.Errorf("buffer size too large: %d (maximum 1048576)", proc.bufferSize)
	}

	// Check if we have any output configured
	if proc.saveToFileFunc == nil && proc.saveToWriterFunc == nil && !proc.mixdown {
		return nil, fmt.Errorf("no output configured: use WithSaveToFile, WithSaveToWriter, or WithMixdown")
	}

	result := &ProcessingResult{}

	// Extract channels
	var channels []audio.ChannelSource
	var err error

	if proc.selectAll {
		channels, _, err = audio.SplitChannels(src, layout)
	} else {
		// Build list of selected channels
		selected := make([]audio.Channel, 0, len(proc.selectedChannels))
		for ch := range proc.selectedChannels {
			selected = append(selected, ch)
		}
		channels, err = audio.ExtractChannels(src, layout, selected...)
	}

	if err != nil {
		return result, fmt.Errorf("failed to extract channels: %w", err)
	}

	// Handle mixdown if requested
	if proc.mixdown {
		if err := proc.processMixdown(src, layout); err != nil {
			result.Errors = append(result.Errors, ProcessingError{
				Channel:  0,
				Activity: "mixdown",
				Err:      err,
			})
		}
	}

	// Process channels
	if proc.maxConcurrent <= 1 {
		// Sequential processing
		for i := range channels {
			proc.processChannel(&channels[i], result)
		}
	} else {
		// Concurrent processing
		proc.processChannelsConcurrent(channels, result)
	}

	return result, nil
}

// processChannel processes a single channel with all configured options.
func (p *channelProcessor) processChannel(ch *audio.ChannelSource, result *ProcessingResult) {
	channel := ch.Channel()

	// Apply channel mapping if configured
	if p.channelMapping != nil {
		if mapped, ok := p.channelMapping[channel]; ok {
			channel = mapped
		}
	}

	var src audio.Source = ch

	// Apply custom processor
	if p.processorFunc != nil {
		processed, err := p.processorFunc(channel, src)
		if err != nil {
			p.handleError(channel, "process", err, result)
			return
		}
		src = processed
	}

	// Apply resampling
	if p.resampleFunc != nil {
		rate := p.resampleFunc(channel)
		if rate > 0 && rate != src.SampleRate() {
			src = audio.NewResampler(src, rate)
		}
	}

	// Apply gain
	if p.gainFunc != nil {
		gain := p.gainFunc(channel)
		if gain != 1.0 {
			src = applyGain(src, gain)
		}
	}

	// Apply normalization
	if p.normalizeFunc != nil && p.normalizeFunc(channel) {
		var err error
		src, err = normalize(src)
		if err != nil {
			p.handleError(channel, "normalize", err, result)
			return
		}
	}

	// Save to file
	if p.saveToFileFunc != nil {
		path := p.saveToFileFunc(channel)
		if path != "" {
			if err := p.saveToFile(src, path, channel, result); err != nil {
				return
			}
		}
	}

	// Save to writer
	if p.saveToWriterFunc != nil {
		writer := p.saveToWriterFunc(channel)
		if writer != nil {
			if err := p.saveToWriter(src, writer, channel, result); err != nil {
				return
			}
		}
	}
}

// processChannelsConcurrent processes channels concurrently.
func (p *channelProcessor) processChannelsConcurrent(channels []audio.ChannelSource, result *ProcessingResult) {
	var wg sync.WaitGroup
	var mu sync.Mutex

	// Create semaphore for limiting concurrency
	sem := make(chan struct{}, p.maxConcurrent)
	if p.maxConcurrent == 0 {
		sem = nil // Unlimited
	}

	for i := range channels {
		wg.Add(1)
		go func(ch *audio.ChannelSource) {
			defer wg.Done()

			if sem != nil {
				sem <- struct{}{}        // Acquire
				defer func() { <-sem }() // Release
			}

			// Create a local result to avoid mutex on every error
			localResult := &ProcessingResult{}
			p.processChannel(ch, localResult)

			// Merge results
			if len(localResult.Errors) > 0 {
				mu.Lock()
				result.Errors = append(result.Errors, localResult.Errors...)
				mu.Unlock()
			}
		}(&channels[i])
	}

	wg.Wait()
}

// processMixdown mixes all channels to mono and saves.
func (p *channelProcessor) processMixdown(src audio.Source, layout audio.Channel) error {
	mono := audio.NewMonoMixer(src)

	if p.mixdownPath != "" {
		return ProcessAndSaveChannel(mono, mono.SampleRate(), p.mixdownPath)
	}
	if p.mixdownWrite != nil {
		return ProcessAndSaveChannelWriter(mono, mono.SampleRate(), p.mixdownWrite)
	}
	return fmt.Errorf("mixdown requested but no output path or writer provided")
}

// saveToFile saves a source to a WAV file.
func (p *channelProcessor) saveToFile(src audio.Source, path string, channel audio.Channel, result *ProcessingResult) error {
	// Read all samples
	samples, err := readAllSamplesAsInt16(src)
	if err != nil {
		p.handleError(channel, "save", err, result)
		return err
	}

	f, err := os.Create(path)
	if err != nil {
		p.handleError(channel, "save", err, result)
		return err
	}
	defer f.Close()

	if err := wav.WriteWAV16(f, src.SampleRate(), samples); err != nil {
		p.handleError(channel, "save", err, result)
		return err
	}

	return nil
}

// saveToWriter saves a source to a writer as WAV.
func (p *channelProcessor) saveToWriter(src audio.Source, w io.Writer, channel audio.Channel, result *ProcessingResult) error {
	// Read all samples
	samples, err := readAllSamplesAsInt16(src)
	if err != nil {
		p.handleError(channel, "save", err, result)
		return err
	}

	if err := wav.WriteWAV16(w, src.SampleRate(), samples); err != nil {
		p.handleError(channel, "save", err, result)
		return err
	}
	return nil
}

// handleError records an error and calls the error handler if configured.
func (p *channelProcessor) handleError(ch audio.Channel, activity string, err error, result *ProcessingResult) {
	procErr := ProcessingError{
		Channel:  ch,
		Activity: activity,
		Err:      err,
	}
	result.Errors = append(result.Errors, procErr)

	if p.errorHandler != nil {
		p.errorHandler(ch, activity, err)
	}
}

// applyGain creates a source that applies gain multiplication.
func applyGain(src audio.Source, gain float32) audio.Source {
	return &gainSource{src: src, gain: gain}
}

type gainSource struct {
	src  audio.Source
	gain float32
}

func (g *gainSource) ReadSamples(dst []float32) (int, error) {
	n, err := g.src.ReadSamples(dst)
	for i := 0; i < n; i++ {
		dst[i] *= g.gain
	}
	return n, err
}

func (g *gainSource) SampleRate() int       { return g.src.SampleRate() }
func (g *gainSource) Channels() int         { return g.src.Channels() }
func (g *gainSource) BufSize() int          { return g.src.BufSize() }
func (g *gainSource) Close() error          { return g.src.Close() }

// normalize reads all audio, finds peak, and applies normalization.
// This requires buffering the entire audio in memory.
func normalize(src audio.Source) (audio.Source, error) {
	// Pre-allocate buffers
	bufSize := src.BufSize()
	tempBuf := make([]float32, bufSize)
	buf := make([]float32, 0, bufSize*100)

	// Track position for efficient growth
	pos := 0

	for {
		n, err := src.ReadSamples(tempBuf)
		if n > 0 {
			// Ensure capacity
			if pos+n > cap(buf) {
				newCap := max(cap(buf)*2, pos+n)
				newBuf := make([]float32, pos, newCap)
				copy(newBuf, buf[:pos])
				buf = newBuf
			}

			// Extend slice and copy samples
			buf = buf[:pos+n]
			copy(buf[pos:], tempBuf[:n])
			pos += n
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	buf = buf[:pos]

	// Find peak
	var peak float32
	for i := range buf {
		sample := buf[i]
		if sample < 0 {
			sample = -sample
		}
		if sample > peak {
			peak = sample
		}
	}

	// Avoid division by zero
	if peak == 0 {
		peak = 1.0
	}

	// Normalize to 0.95 to avoid clipping
	gain := 0.95 / peak
	for i := range buf {
		buf[i] *= gain
	}

	// Return a source that reads from the normalized buffer
	return &bufferSource{
		data:       buf,
		sampleRate: src.SampleRate(),
		channels:   src.Channels(),
	}, nil
}

type bufferSource struct {
	data       []float32
	pos        int
	sampleRate int
	channels   int
}

func (b *bufferSource) ReadSamples(dst []float32) (int, error) {
	if b.pos >= len(b.data) {
		return 0, io.EOF
	}

	n := copy(dst, b.data[b.pos:])
	b.pos += n

	if b.pos >= len(b.data) {
		return n, io.EOF
	}
	return n, nil
}

func (b *bufferSource) SampleRate() int { return b.sampleRate }
func (b *bufferSource) Channels() int   { return b.channels }
func (b *bufferSource) BufSize() int    { return 4096 }
func (b *bufferSource) Close() error    { return nil }

// readAllSamplesAsInt16 reads all samples from a source and converts them to int16.
func readAllSamplesAsInt16(src audio.Source) ([]int16, error) {
	// Pre-allocate based on known size if possible
	bufSize := src.BufSize()
	tempBuf := make([]float32, bufSize)
	result := make([]int16, 0, bufSize*100)

	// Track position for efficient appending
	pos := 0

	for {
		n, err := src.ReadSamples(tempBuf)
		if n > 0 {
			// Ensure capacity
			if pos+n > cap(result) {
				newCap := max(cap(result)*2, pos+n)
				newResult := make([]int16, pos, newCap)
				copy(newResult, result[:pos])
				result = newResult
			}

			// Extend slice to accommodate new samples
			result = result[:pos+n]

			// Convert samples in place
			for i := 0; i < n; i++ {
				// Clamp to [-1.0, 1.0]
				sample := tempBuf[i]
				if sample > 1.0 {
					sample = 1.0
				} else if sample < -1.0 {
					sample = -1.0
				}
				// Convert to int16
				result[pos+i] = int16(sample * 32767.0)
			}
			pos += n
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return result[:pos], nil
}
