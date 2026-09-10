// SPDX-License-Identifier: EPL-2.0

// Package audio provides low-level audio processing primitives.
//
// This package contains the core audio processing building blocks:
//   - Source interface for audio input
//   - Resampler for sample rate conversion
//   - MonoMixer for channel mixing
//   - Format registry for decoder registration
//
// # Source Interface
//
// The Source interface is the foundation of audio processing:
//
//	type Source interface {
//	    SampleRate() int
//	    Channels() int
//	    ReadSamples(dst []float32) (int, error)
//	    BufSize() int
//	    Close() error
//	}
//
// All audio decoders and processors implement this interface, allowing
// them to be chained together in processing pipelines.
//
// # Resampling
//
// The Resampler changes the sample rate of audio using cubic interpolation:
//
//	resampler := audio.NewResampler(source, 16000)
//	buf := make([]float32, 4096)
//	n, err := resampler.ReadSamples(buf)
//
// Resampling works for both upsampling and downsampling.
//
// The resampler pulls from its source in large blocks rather than frame by
// frame. This is what keeps it fast: a decoder sitting on an unbuffered file
// costs a syscall and several allocations per call, so reading a frame at a
// time makes that overhead, not the arithmetic, the dominant cost. Because of
// this, the size of the buffer passed to ReadSamples has very little effect on
// throughput, and ReadSamples allocates nothing once streaming has started.
//
// Treat the block reads as load-bearing rather than incidental. Reducing the
// block size costs roughly 40x throughput while changing neither the output nor
// the allocation count, so nothing but a call-pattern check notices. That check
// is TestResampler_ReadsSourceInBlocks; if you restructure how the source is
// read, make sure it still holds.
//
// When downsampling, a one-pole low-pass filter is applied for anti-aliasing.
// Be aware that its coefficient is fixed rather than derived from the
// resampling ratio, so at large ratios it attenuates considerably less than a
// proper decimation filter and content above the output Nyquist frequency will
// alias. See the package-level Limitations note in the parent audpbx package.
//
// # Channel Mixing
//
// The MonoMixer converts multi-channel audio to mono by averaging:
//
//	mono := audio.NewMonoMixer(source)
//	buf := make([]float32, 4096)
//	n, err := mono.ReadSamples(buf)
//
// Mono audio is often required for voice processing applications.
//
// # Format Registry
//
// The registry allows dynamic decoder registration:
//
//	registry := audio.NewRegistry()
//	registry.Register("wav", wav.Decoder{})
//	decoder, _ := registry.Get("wav")
//
// This is useful for applications that need to support multiple formats.
//
// # Sample Format
//
// Audio samples are represented as float32 in the range [-1.0, 1.0]:
//   - 0.0 represents silence
//   - 1.0 represents maximum positive amplitude
//   - -1.0 represents maximum negative amplitude
//
// This normalized format makes it easy to process audio without worrying
// about bit depths and ensures no clipping during intermediate processing.
//
// # Performance Considerations
//
// Resampler and MonoMixer both allocate nothing per call once running; their
// only allocations are one-time setup in the constructor. Sources are read in
// large blocks so that per-call decoder overhead is amortised.
//
// For best performance:
//   - Reuse buffers rather than allocating inside a read loop
//   - Process audio in streaming fashion rather than loading all in memory
//   - Do not bother tuning the buffer size for the resampler; it reads its
//     source in fixed internal blocks either way. DefaultBufSize is fine.
//
// Note that a Resampler tracks its own end-of-stream state and is not
// reusable across streams. Construct a new one per stream; resetting the
// underlying source is not enough, and in a benchmark loop it will make every
// iteration after the first return io.EOF immediately.
//
// # Error Handling
//
// Audio processing functions return io.EOF when no more data is available.
// Other errors indicate problems with the source or processing:
//
//	for {
//	    n, err := source.ReadSamples(buf)
//	    if err == io.EOF {
//	        break // Normal end of stream
//	    }
//	    if err != nil {
//	        return err // Processing error
//	    }
//	    // Process n samples from buf
//	}
package audio
