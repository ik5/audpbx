package audpbx

import (
	"fmt"
	"io"

	"github.com/ik5/audpbx/audio"
)

// ResampleToMono16 is a high-level convenience function that resamples audio to a target
// sample rate, converts it to mono, and collects all samples as 16-bit PCM data.
//
// This function creates a processing pipeline:
//   1. Resamples the source audio to targetRate using cubic interpolation
//   2. Converts the resampled audio to mono by averaging channels
//   3. Reads all samples from the pipeline
//   4. Converts float32 samples to int16 PCM format
//
// Parameters:
//   - src: The audio source to process (implements Source interface)
//   - targetRate: Target sample rate in Hz (e.g., 8000, 16000, 44100, 48000)
//   - bufferSize: Size of the buffer for reading samples (e.g., 4096)
//                 Larger buffers may be more efficient but use more memory
//
// Returns:
//   - []int16: Collected PCM samples as 16-bit signed integers
//   - int: The output sample rate (same as targetRate)
//   - error: Any error encountered during processing, or io.EOF when complete
//
// Note: This is a convenience function for common use cases. For more control over
// the audio processing pipeline, use NewResampler() and NewMonoMixer() directly.
//
// Example:
//
//	src, _ := decoder.Decode(file)
//	pcm16, rate, err := audio.ResampleToMono16(src, 8000, 4096)
//	if err != nil && err != io.EOF {
//	    panic(err)
//	}
//	// pcm16 now contains mono 16-bit PCM at 8kHz
func ResampleToMono16(src audio.Source, targetRate int, bufferSize int) ([]int16, int, error) {
	// Create the processing pipeline: resample -> mono
	resampler := audio.NewResampler(src, targetRate)
	mono := audio.NewMonoMixer(resampler)

	// Pre-allocate based on estimated output size to reduce allocations
	// Estimate: (source_rate / target_rate) * source_duration
	// We'll start with a reasonable default and grow if needed
	estimatedSamples := targetRate * 2 // Assume ~2 seconds initially
	pcm16 := make([]int16, 0, estimatedSamples)
	buf := make([]float32, bufferSize)

	for {
		n, err := mono.ReadSamples(buf)
		if n > 0 {
			// Ensure capacity before batch conversion
			if cap(pcm16)-len(pcm16) < n {
				// Grow by at least n samples, or double capacity
				newCap := len(pcm16) + max(n, cap(pcm16))
				newSlice := make([]int16, len(pcm16), newCap)
				copy(newSlice, pcm16)
				pcm16 = newSlice
			}

			// Batch convert float32 to int16 (inlined for performance)
			startIdx := len(pcm16)
			pcm16 = pcm16[:startIdx+n]
			const maxInt16 float32 = 32768.0
			for i := range n {
				x := buf[i]
				// Clamp to [-1, 1]
				if x > 1 {
					x = 1
				} else if x < -1 {
					x = -1
				}
				pcm16[startIdx+i] = int16(x * maxInt16)
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, targetRate, fmt.Errorf("%w", err)
		}
	}

	return pcm16, targetRate, nil
}
