// SPDX-License-Identifier: EPL-2.0

package audpbx

import (
	"fmt"
	"io"

	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/utils"
)

// estimatedOutputSeconds is how many seconds of output ResampleToMono16
// pre-allocates before it starts reading.
//
// Source lengths are not discoverable through the Source interface, so this is
// a guess tuned to cover typical recordings in a single allocation. Being wrong
// is cheap in both directions: shorter inputs waste a little capacity, longer
// ones fall back to doubling.
const estimatedOutputSeconds = 30

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

	// Pre-allocate based on estimated output size to reduce allocations.
	//
	// Source lengths are not exposed by the Source interface, so the exact
	// output size is unknown up front. Starting at estimatedOutputSeconds of
	// output covers the great majority of recordings in one allocation;
	// anything longer doubles from there, so growth stays logarithmic rather
	// than copying the whole buffer every few blocks.
	estimatedSamples := targetRate * estimatedOutputSeconds
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
			out := pcm16[startIdx : startIdx+n]
			for i, x := range buf[:n] {
				out[i] = utils.Float32ToInt16(x)
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
