// SPDX-License-Identifier: EPL-2.0

package aiff_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/formats/aiff"
	"github.com/ik5/audpbx/formats/wav"
)

// Example demonstrates basic AIFF decoding and conversion to WAV.
func Example() {
	// Open AIFF file
	f, err := os.Open("testdata/sample.aiff")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Decode AIFF to audio source
	decoder := aiff.Decoder{}
	src, err := decoder.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	// Display audio properties
	fmt.Printf("Sample Rate: %d Hz\n", src.SampleRate())
	fmt.Printf("Channels: %d\n", src.Channels())

	// Read some samples
	buf := make([]float32, 4096)
	n, _ := src.ReadSamples(buf)
	fmt.Printf("Read %d samples\n", n)

	// Output:
	// Sample Rate: 44100 Hz
	// Channels: 2
	// Read 4096 samples
}

// ExampleDecoder_Decode shows how to decode an AIFF file.
func ExampleDecoder_Decode() {
	// Create AIFF decoder
	decoder := aiff.Decoder{}

	// Open AIFF file
	f, err := os.Open("input.aiff")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Decode AIFF to audio source
	src, err := decoder.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Decoded AIFF: %d Hz, %d channels\n",
		src.SampleRate(), src.Channels())
}

// ExampleDecoder_Decode_convertToWav demonstrates converting AIFF to WAV format.
func ExampleDecoder_Decode_convertToWav() {
	// Decode AIFF
	aiffFile, err := os.Open("input.aiff")
	if err != nil {
		log.Fatal(err)
	}
	defer aiffFile.Close()

	aiffDecoder := aiff.Decoder{}
	src, err := aiffDecoder.Decode(aiffFile)
	if err != nil {
		log.Fatal(err)
	}

	// Read all samples and convert to int16
	buf := make([]float32, 4096)
	var samples []int16
	for {
		n, err := src.ReadSamples(buf)
		if n > 0 {
			// Convert float32 to int16
			for i := 0; i < n; i++ {
				sample := buf[i]
				if sample > 1.0 {
					sample = 1.0
				} else if sample < -1.0 {
					sample = -1.0
				}
				samples = append(samples, int16(sample*32768.0))
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	// Write to WAV
	wavFile, err := os.Create("output.wav")
	if err != nil {
		log.Fatal(err)
	}
	defer wavFile.Close()

	if err := wav.WriteWAV16(wavFile, src.SampleRate(), samples); err != nil {
		log.Fatal(err)
	}

	fmt.Println("AIFF converted to WAV")
}

// ExampleDecoder_Decode_resample demonstrates resampling AIFF audio.
func ExampleDecoder_Decode_resample() {
	// Decode AIFF
	f, err := os.Open("input.aiff")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := aiff.Decoder{}
	src, err := decoder.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	// Resample to 16kHz mono
	resampler := audio.NewResampler(src, 16000)
	mixer := audio.NewMonoMixer(resampler)

	// Process resampled audio
	buf := make([]float32, 1024)
	for {
		n, err := mixer.ReadSamples(buf)
		if n > 0 {
			// Process samples in buf[:n]
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Println("AIFF resampled to 16kHz mono")
}

// ExampleDecoder_Decode_errorHandling shows error handling for invalid AIFF files.
func ExampleDecoder_Decode_errorHandling() {
	decoder := aiff.Decoder{}

	// Try to decode invalid AIFF data
	invalidData := bytes.NewReader([]byte("not an aiff file"))
	_, err := decoder.Decode(invalidData)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("AIFF decoded successfully")
}

// ExampleDecoder_Decode_streaming demonstrates streaming AIFF decoding.
func ExampleDecoder_Decode_streaming() {
	// Open AIFF file for streaming
	f, err := os.Open("input.aiff")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := aiff.Decoder{}
	src, err := decoder.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	// Stream in chunks
	chunkSize := 4096
	buf := make([]float32, chunkSize)

	var totalSamples int
	for {
		n, err := src.ReadSamples(buf)
		totalSamples += n

		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
	}

	fmt.Printf("Streamed %d samples from AIFF\n", totalSamples)
}

// ExampleDecoder_Decode_bigEndian demonstrates AIFF's big-endian format handling.
func ExampleDecoder_Decode_bigEndian() {
	// AIFF uses big-endian byte order (unlike WAV which uses little-endian)
	// The decoder handles byte order conversion transparently
	f, err := os.Open("input.aiff")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := aiff.Decoder{}
	src, err := decoder.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	// Output is always normalized float32 regardless of source byte order
	buf := make([]float32, 1024)
	n, _ := src.ReadSamples(buf)
	fmt.Printf("Read %d samples (byte order handled transparently)\n", n)
}
