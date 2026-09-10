// SPDX-License-Identifier: EPL-2.0

// Command resampler converts an audio file of any supported format to a mono
// 16-bit PCM WAV file at a target sample rate.
//
// Usage:
//
//	resampler <input.{wav|mp3|ogg|aiff}> <output.wav> [target-rate]
//
// target-rate defaults to 8000 Hz, the rate used by G.711 telephony.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/ik5/audpbx"
	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/formats/aiff"
	"github.com/ik5/audpbx/formats/mp3"
	"github.com/ik5/audpbx/formats/vorbis"
	"github.com/ik5/audpbx/formats/wav"
)

// defaultTargetRate is the sample rate used when none is given: 8 kHz, the
// G.711 telephony rate this library is primarily aimed at.
const defaultTargetRate = 8000

// bufferSize is how many samples to pull from the pipeline per read.
//
// This is only the size of the caller's buffer. The resampler reads its source
// in blocks of its own choosing, so this value has very little effect on
// throughput; anything from a few hundred samples upward performs the same.
const bufferSize = 4096

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) < 2 {
		return fmt.Errorf(
			"usage: resampler <input.{wav|mp3|ogg|aiff}> <output.wav> [target-rate]")
	}

	inPath, outPath := args[0], args[1]

	targetRate := defaultTargetRate
	if len(args) > 2 {
		rate, err := strconv.Atoi(args[2])
		if err != nil {
			return fmt.Errorf("invalid target rate %q: %w", args[2], err)
		}
		if rate <= 0 {
			return fmt.Errorf("target rate must be positive, got %d", rate)
		}
		targetRate = rate
	}

	// Registry: maps a file extension to the decoder that handles it.
	reg := audio.NewRegistry()
	reg.Register("wav", wav.Decoder{})
	reg.Register("mp3", mp3.Decoder{})
	reg.Register("ogg", vorbis.Decoder{})
	reg.Register("aif", aiff.Decoder{})
	reg.Register("aiff", aiff.Decoder{})

	ext := filepath.Ext(inPath)
	if len(ext) > 0 {
		ext = ext[1:] // drop the dot
	}

	dec, ok := reg.Get(ext)
	if !ok {
		return fmt.Errorf("unsupported format: %q", ext)
	}

	inFile, err := os.Open(inPath)
	if err != nil {
		return fmt.Errorf("opening input: %w", err)
	}
	defer inFile.Close()

	src, err := dec.Decode(inFile)
	if err != nil {
		return fmt.Errorf("decoding %s: %w", inPath, err)
	}
	defer src.Close()

	// Resample to targetRate and mix down to mono in one pass.
	pcm16, sampleRate, err := audpbx.ResampleToMono16(src, targetRate, bufferSize)
	if err != nil {
		return fmt.Errorf("resampling: %w", err)
	}

	outFile, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("creating output: %w", err)
	}
	defer outFile.Close()

	if err := wav.WriteWAV16(outFile, sampleRate, pcm16); err != nil {
		return fmt.Errorf("writing wav: %w", err)
	}

	// Close explicitly so a write error is not swallowed by the deferred close.
	if err := outFile.Close(); err != nil {
		return fmt.Errorf("closing output: %w", err)
	}

	fmt.Printf("Wrote %s: %d Hz mono, %d samples\n", outPath, sampleRate, len(pcm16))

	return nil
}
