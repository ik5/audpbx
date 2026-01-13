// SPDX-License-Identifier: EPL-2.0

package vorbis_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/formats/vorbis"
	"github.com/ik5/audpbx/formats/wav"
)

// Example demonstrates basic Ogg Vorbis decoding and conversion to WAV.
func Example() {
	// Open Ogg Vorbis file
	f, err := os.Open("testdata/sample.ogg")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Decode Ogg Vorbis to audio source
	decoder := vorbis.Decoder{}
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

// ExampleDecoder_Decode shows how to decode an Ogg Vorbis file.
func ExampleDecoder_Decode() {
	// Create Vorbis decoder
	decoder := vorbis.Decoder{}

	// Open Ogg Vorbis file
	f, err := os.Open("input.ogg")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Decode Ogg Vorbis to audio source
	src, err := decoder.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Decoded Vorbis: %d Hz, %d channels\n",
		src.SampleRate(), src.Channels())
}

// ExampleDecoder_Decode_convertToWav demonstrates converting Ogg Vorbis to WAV format.
func ExampleDecoder_Decode_convertToWav() {
	// Decode Ogg Vorbis
	vorbisFile, err := os.Open("input.ogg")
	if err != nil {
		log.Fatal(err)
	}
	defer vorbisFile.Close()

	vorbisDecoder := vorbis.Decoder{}
	src, err := vorbisDecoder.Decode(vorbisFile)
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

	fmt.Println("Ogg Vorbis converted to WAV")
}

// ExampleDecoder_Decode_resample demonstrates resampling Ogg Vorbis audio.
func ExampleDecoder_Decode_resample() {
	// Decode Ogg Vorbis
	f, err := os.Open("input.ogg")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := vorbis.Decoder{}
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

	fmt.Println("Ogg Vorbis resampled to 16kHz mono")
}

// ExampleDecoder_Decode_errorHandling shows error handling for invalid Ogg Vorbis files.
func ExampleDecoder_Decode_errorHandling() {
	decoder := vorbis.Decoder{}

	// Try to decode invalid Ogg Vorbis data
	invalidData := bytes.NewReader([]byte("not an ogg file"))
	_, err := decoder.Decode(invalidData)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("Ogg Vorbis decoded successfully")
}

// ExampleDecoder_Decode_streaming demonstrates streaming Ogg Vorbis decoding.
func ExampleDecoder_Decode_streaming() {
	// Open Ogg Vorbis file for streaming
	f, err := os.Open("input.ogg")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := vorbis.Decoder{}
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

	fmt.Printf("Streamed %d samples from Ogg Vorbis\n", totalSamples)
}

// ExampleDecoder_Decode_quality demonstrates handling different Vorbis quality settings.
func ExampleDecoder_Decode_quality() {
	// Ogg Vorbis supports various quality levels (lossy compression)
	// The decoder handles all quality levels transparently
	f, err := os.Open("input.ogg")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := vorbis.Decoder{}
	src, err := decoder.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	// Regardless of encoding quality, output is float32 samples
	fmt.Printf("Decoded Vorbis: %d Hz, %d channels\n",
		src.SampleRate(), src.Channels())
	fmt.Println("Quality level handled transparently")
}
