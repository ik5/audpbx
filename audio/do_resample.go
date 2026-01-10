package audio

import (
	"fmt"
	"io"

	"github.com/ik5/audpbx/utils"
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
func ResampleToMono16(src Source, targetRate int, bufferSize int) ([]int16, int, error) {
	// Create the processing pipeline: resample -> mono
	resampler := NewResampler(src, targetRate)
	mono := NewMonoMixer(resampler)

	// Collect all samples
	var pcm16 []int16
	buf := make([]float32, bufferSize)

	for {
		n, err := mono.ReadSamples(buf)
		if n > 0 {
			for i := range n {
				pcm16 = append(pcm16, utils.Float32ToInt16(buf[i]))
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
