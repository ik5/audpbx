// SPDX-License-Identifier: EPL-2.0

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/ik5/audpbx"
	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/formats/aiff"
	"github.com/ik5/audpbx/formats/mp3"
	"github.com/ik5/audpbx/formats/vorbis"
	"github.com/ik5/audpbx/formats/wav"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("usage: profile_resampler <input.{wav|mp3|ogg}> <output.wav> [--cpu-profile=cpu.prof] [--mem-profile=mem.prof] [--trace]")
		os.Exit(1)
	}
	inPath := os.Args[1]
	outPath := os.Args[2]

	// Parse flags
	var cpuProfile, memProfile string
	enableTrace := false
	for i := 3; i < len(os.Args); i++ {
		arg := os.Args[i]
		if len(arg) > 14 && arg[:13] == "--cpu-profile" {
			cpuProfile = arg[14:]
		} else if len(arg) > 14 && arg[:13] == "--mem-profile" {
			memProfile = arg[14:]
		} else if arg == "--trace" {
			enableTrace = true
		}
	}

	// CPU profiling
	if cpuProfile != "" {
		f, err := os.Create(cpuProfile)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		if err := pprof.StartCPUProfile(f); err != nil {
			panic(err)
		}
		defer pprof.StopCPUProfile()
		fmt.Printf("CPU profiling enabled: %s\n", cpuProfile)
	}

	// Registry
	reg := audio.NewRegistry()
	reg.Register("wav", wav.Decoder{})
	reg.Register("mp3", mp3.Decoder{})
	reg.Register("ogg", vorbis.Decoder{})
	reg.Register("aif", aiff.Decoder{})
	reg.Register("aiff", aiff.Decoder{})

	ext := filepath.Ext(inPath)
	if len(ext) > 0 {
		ext = ext[1:] // drop dot
	}
	dec, ok := reg.Get(ext)
	if !ok {
		fmt.Println("unsupported format:", ext)
		os.Exit(1)
	}

	// Get file info
	fileInfo, err := os.Stat(inPath)
	if err != nil {
		panic(err)
	}
	fmt.Printf("Input file: %s (%.2f MB)\n", inPath, float64(fileInfo.Size())/1024/1024)

	// Timing breakdown
	totalStart := time.Now()

	// Stage 1: File open
	openStart := time.Now()
	inFile, err := os.Open(inPath)
	if err != nil {
		panic(err)
	}
	defer inFile.Close()
	openDuration := time.Since(openStart)

	// Stage 2: Decode
	decodeStart := time.Now()
	src, err := dec.Decode(inFile)
	if err != nil {
		panic(err)
	}
	defer src.Close()
	decodeDuration := time.Since(decodeStart)

	fmt.Printf("Source: %d Hz, %d channels\n", src.SampleRate(), src.Channels())

	// Memory snapshot before processing
	var memBefore runtime.MemStats
	runtime.ReadMemStats(&memBefore)

	// Stage 3: Resample + Mix + Convert
	resampleStart := time.Now()

	if enableTrace {
		// Manual timing points
		fmt.Println("\n--- Starting resampling with trace points ---")
	}

	pcm16, sampleRate, err := audpbx.ResampleToMono16(src, 8000, 4096)
	if err != nil {
		panic(err)
	}

	resampleDuration := time.Since(resampleStart)

	// Memory snapshot after processing
	var memAfter runtime.MemStats
	runtime.ReadMemStats(&memAfter)

	// Stage 4: Write output
	writeStart := time.Now()
	outFile, err := os.Create(outPath)
	if err != nil {
		panic(err)
	}
	defer outFile.Close()

	if err := wav.WriteWAV16(outFile, sampleRate, pcm16); err != nil {
		panic(err)
	}
	writeDuration := time.Since(writeStart)

	totalDuration := time.Since(totalStart)

	// Output statistics
	fmt.Printf("\n=== Performance Breakdown ===\n")
	fmt.Printf("File open:       %8.3f ms (%5.1f%%)\n", openDuration.Seconds()*1000, 100*openDuration.Seconds()/totalDuration.Seconds())
	fmt.Printf("Decode setup:    %8.3f ms (%5.1f%%)\n", decodeDuration.Seconds()*1000, 100*decodeDuration.Seconds()/totalDuration.Seconds())
	fmt.Printf("Resample/Mix:    %8.3f ms (%5.1f%%) ← MAIN PROCESSING\n", resampleDuration.Seconds()*1000, 100*resampleDuration.Seconds()/totalDuration.Seconds())
	fmt.Printf("Write output:    %8.3f ms (%5.1f%%)\n", writeDuration.Seconds()*1000, 100*writeDuration.Seconds()/totalDuration.Seconds())
	fmt.Printf("---\n")
	fmt.Printf("TOTAL:           %8.3f ms\n", totalDuration.Seconds()*1000)

	fmt.Printf("\n=== Memory Statistics ===\n")
	fmt.Printf("Alloc delta:     %.2f MB\n", float64(memAfter.Alloc-memBefore.Alloc)/1024/1024)
	fmt.Printf("Total alloc:     %.2f MB\n", float64(memAfter.TotalAlloc-memBefore.TotalAlloc)/1024/1024)
	fmt.Printf("Num GC:          %d\n", memAfter.NumGC-memBefore.NumGC)
	fmt.Printf("Output samples:  %d (%.2f MB)\n", len(pcm16), float64(len(pcm16)*2)/1024/1024)

	fmt.Printf("\n=== Throughput ===\n")
	inputMB := float64(fileInfo.Size()) / 1024 / 1024
	throughput := inputMB / totalDuration.Seconds()
	fmt.Printf("Speed:           %.2f MB/s\n", throughput)
	fmt.Printf("Processing rate: %.2fx realtime\n", calculateRealtimeRatio(src.SampleRate(), src.Channels(), len(pcm16), sampleRate, totalDuration))

	// Memory profiling
	if memProfile != "" {
		f, err := os.Create(memProfile)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		runtime.GC() // get up-to-date statistics
		if err := pprof.WriteHeapProfile(f); err != nil {
			panic(err)
		}
		fmt.Printf("\nMemory profile written to: %s\n", memProfile)
	}

	fmt.Printf("\nWrote: %s\n", outPath)
}

func calculateRealtimeRatio(srcRate, srcChannels int, outputSamples, outputRate int, duration time.Duration) float64 {
	// Calculate audio duration in seconds
	audioDuration := float64(outputSamples) / float64(outputRate)
	processingDuration := duration.Seconds()
	return audioDuration / processingDuration
}
