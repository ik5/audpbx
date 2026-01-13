// SPDX-License-Identifier: EPL-2.0

package mp3_test

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"

	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/formats/mp3"
	"github.com/ik5/audpbx/formats/wav"
)

// Example demonstrates basic MP3 decoding and conversion to WAV.
func Example() {
	// Open MP3 file
	f, err := os.Open("testdata/sample.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Decode MP3 to audio source
	decoder := mp3.Decoder{}
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

// ExampleDecoder_Decode shows how to decode an MP3 file.
func ExampleDecoder_Decode() {
	// Create MP3 decoder
	decoder := mp3.Decoder{}

	// Open MP3 file
	f, err := os.Open("input.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Decode MP3 to audio source
	src, err := decoder.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Decoded MP3: %d Hz, %d channels\n",
		src.SampleRate(), src.Channels())
}

// ExampleDecoder_Decode_convertToWav demonstrates converting MP3 to WAV format.
func ExampleDecoder_Decode_convertToWav() {
	// Decode MP3
	mp3File, err := os.Open("input.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer mp3File.Close()

	mp3Decoder := mp3.Decoder{}
	src, err := mp3Decoder.Decode(mp3File)
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

	fmt.Println("MP3 converted to WAV")
}

// ExampleDecoder_Decode_resample demonstrates resampling MP3 audio.
func ExampleDecoder_Decode_resample() {
	// Decode MP3
	f, err := os.Open("input.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := mp3.Decoder{}
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

	fmt.Println("MP3 resampled to 16kHz mono")
}

// ExampleDecoder_Decode_errorHandling shows error handling for invalid MP3 files.
func ExampleDecoder_Decode_errorHandling() {
	decoder := mp3.Decoder{}

	// Try to decode invalid MP3 data
	invalidData := bytes.NewReader([]byte("not an mp3 file"))
	_, err := decoder.Decode(invalidData)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Println("MP3 decoded successfully")
}

// ExampleDecoder_Decode_streaming demonstrates streaming MP3 decoding.
func ExampleDecoder_Decode_streaming() {
	// Open MP3 file for streaming
	f, err := os.Open("input.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := mp3.Decoder{}
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

	fmt.Printf("Streamed %d samples from MP3\n", totalSamples)
}

// ExampleDecoder_Decode_metadata shows how MP3 decoding handles stereo output.
func ExampleDecoder_Decode_metadata() {
	// MP3 decoder always outputs stereo
	f, err := os.Open("input.mp3")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	decoder := mp3.Decoder{}
	src, err := decoder.Decode(f)
	if err != nil {
		log.Fatal(err)
	}

	// MP3 decoder provides stereo output
	if src.Channels() == 2 {
		fmt.Println("MP3 decoded as stereo")
	}

	// Use MonoMixer if mono output is needed
	mono := audio.NewMonoMixer(src)
	fmt.Printf("Converted to %d channel(s)\n", mono.Channels())

	// Output:
	// MP3 decoded as stereo
	// Converted to 1 channel(s)
}
