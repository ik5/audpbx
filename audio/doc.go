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
// Resampling works for both upsampling and downsampling with high quality.
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
// The audio processing functions are optimized for performance:
//   - Minimal allocations (often zero after warmup)
//   - Efficient buffer management
//   - SIMD-friendly algorithms where possible
//
// For best performance:
//   - Reuse buffers when possible
//   - Use appropriate buffer sizes (4096 is a good default)
//   - Process audio in streaming fashion rather than loading all in memory
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
