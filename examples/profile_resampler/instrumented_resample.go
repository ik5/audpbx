// SPDX-License-Identifier: EPL-2.0

package main

import (
	"fmt"
	"io"
	"time"

	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/utils"
)

// estimatedOutputSeconds mirrors the constant of the same name in
// resample.go: how many seconds of output to pre-allocate before reading.
const estimatedOutputSeconds = 30

// InstrumentedResampleToMono16 is a copy of audpbx.ResampleToMono16 with
// per-stage timing added.
//
// Keep it in step with resample.go. Its whole purpose is to attribute time
// within that function, so if the two drift apart the numbers it reports stop
// describing the code that actually runs.
//
// Note that the read stage measured here covers the entire pipeline beneath it
// — mono mixing, resampling, and decoding — because those are pull-based and
// only do work when ReadSamples is called. A high "ReadSamples" figure is
// therefore expected; use a CPU profile to see which layer inside it is
// responsible.
func InstrumentedResampleToMono16(src audio.Source, targetRate int, bufferSize int) ([]int16, int, error) {
	timings := make(map[string]time.Duration)
	var totalStart = time.Now()

	// Stage 1: Setup pipeline
	setupStart := time.Now()
	resampler := audio.NewResampler(src, targetRate)
	mono := audio.NewMonoMixer(resampler)
	estimatedSamples := targetRate * estimatedOutputSeconds
	pcm16 := make([]int16, 0, estimatedSamples)
	buf := make([]float32, bufferSize)
	timings["setup"] = time.Since(setupStart)

	// Counters
	readCalls := 0
	totalSamplesRead := 0
	conversionTime := time.Duration(0)
	allocationTime := time.Duration(0)

	// Stage 2: Processing loop
	loopStart := time.Now()
	for {
		readStart := time.Now()
		n, err := mono.ReadSamples(buf)
		readDuration := time.Since(readStart)
		timings["read"] += readDuration
		readCalls++

		if n > 0 {
			totalSamplesRead += n

			// Allocation if needed
			allocStart := time.Now()
			if cap(pcm16)-len(pcm16) < n {
				newCap := len(pcm16) + max(n, cap(pcm16))
				newSlice := make([]int16, len(pcm16), newCap)
				copy(newSlice, pcm16)
				pcm16 = newSlice
			}
			allocationTime += time.Since(allocStart)

			// Conversion
			convStart := time.Now()
			startIdx := len(pcm16)
			pcm16 = pcm16[:startIdx+n]
			out := pcm16[startIdx : startIdx+n]
			for i, x := range buf[:n] {
				out[i] = utils.Float32ToInt16(x)
			}
			conversionTime += time.Since(convStart)
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, targetRate, fmt.Errorf("%w", err)
		}
	}
	timings["loop_total"] = time.Since(loopStart)
	timings["conversion"] = conversionTime
	timings["allocation"] = allocationTime

	totalDuration := time.Since(totalStart)

	// Print detailed breakdown
	fmt.Printf("\n=== Instrumented Timing Breakdown ===\n")
	fmt.Printf("Setup:           %8.3f ms (%5.1f%%)\n",
		timings["setup"].Seconds()*1000,
		100*timings["setup"].Seconds()/totalDuration.Seconds())
	fmt.Printf("ReadSamples:     %8.3f ms (%5.1f%%) [%d calls, %.2f ms/call]\n",
		timings["read"].Seconds()*1000,
		100*timings["read"].Seconds()/totalDuration.Seconds(),
		readCalls,
		timings["read"].Seconds()*1000/float64(readCalls))
	fmt.Printf("  └─ Allocation: %8.3f ms (%5.1f%%)\n",
		allocationTime.Seconds()*1000,
		100*allocationTime.Seconds()/totalDuration.Seconds())
	fmt.Printf("  └─ Conversion: %8.3f ms (%5.1f%%)\n",
		conversionTime.Seconds()*1000,
		100*conversionTime.Seconds()/totalDuration.Seconds())
	fmt.Printf("Loop overhead:   %8.3f ms (%5.1f%%)\n",
		(timings["loop_total"]-timings["read"]-allocationTime-conversionTime).Seconds()*1000,
		100*(timings["loop_total"]-timings["read"]-allocationTime-conversionTime).Seconds()/totalDuration.Seconds())
	fmt.Printf("---\n")
	fmt.Printf("TOTAL:           %8.3f ms\n", totalDuration.Seconds()*1000)
	fmt.Printf("\nSamples read:    %d (%.2f MB float32)\n",
		totalSamplesRead,
		float64(totalSamplesRead*4)/1024/1024)
	fmt.Printf("Output samples:  %d (%.2f MB int16)\n",
		len(pcm16),
		float64(len(pcm16)*2)/1024/1024)

	return pcm16, targetRate, nil
}
