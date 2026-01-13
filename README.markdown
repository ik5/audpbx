# audpbx

[![Go Reference](https://pkg.go.dev/badge/github.com/ik5/audpbx.svg)](https://pkg.go.dev/github.com/ik5/audpbx)
[![Go Report Card](https://goreportcard.com/badge/github.com/ik5/audpbx)](https://goreportcard.com/report/github.com/ik5/audpbx)

**audpbx** is a high-performance Go library for audio processing, specializing in format conversion, resampling, and channel mixing. Designed for telephony and VoIP applications, it provides efficient tools for converting various audio formats to mono 16-bit PCM at any sample rate.

## Features

- **Multiple Format Support**: Decode WAV, MP3, Ogg Vorbis, and AIFF audio files
- **High-Quality Resampling**: Cubic interpolation for sample rate conversion with minimal artifacts
- **Channel Mixing**: Convert stereo/multi-channel audio to mono
- **Performance Optimized**: Near-zero allocations, optimized for throughput
- **Simple API**: Clean, idiomatic Go interfaces
- **Streaming Support**: Process audio without loading entire files into memory
- **Comprehensive Testing**: Extensive unit tests and benchmarks

## Installation

```bash
go get github.com/ik5/audpbx
```

## Quick Start

### Basic Usage: Decode and Resample Audio

```go
package main

import (
    "log"
    "os"
    
    "github.com/ik5/audpbx"
    "github.com/ik5/audpbx/formats/wav"
)

func main() {
    // Open audio file
    file, err := os.Open("input.wav")
    if err != nil {
        log.Fatal(err)
    }
    defer file.Close()
    
    // Decode the audio
    decoder := wav.Decoder{}
    src, err := decoder.Decode(file)
    if err != nil {
        log.Fatal(err)
    }
    defer src.Close()
    
    // Resample to 8kHz mono, 16-bit PCM
    pcm16, rate, err := audpbx.ResampleToMono16(src, 8000, 4096)
    if err != nil {
        log.Fatal(err)
    }
    
    // Write output
    output, err := os.Create("output.wav")
    if err != nil {
        log.Fatal(err)
    }
    defer output.Close()
    
    if err := wav.WriteWAV16(output, rate, pcm16); err != nil {
        log.Fatal(err)
    }
}
```

### Convert MP3 to WAV

```go
package main

import (
    "log"
    "os"
    
    "github.com/ik5/audpbx"
    "github.com/ik5/audpbx/formats/mp3"
    "github.com/ik5/audpbx/formats/wav"
)

func main() {
    // Decode MP3
    mp3File, _ := os.Open("input.mp3")
    defer mp3File.Close()
    
    decoder := mp3.Decoder{}
    src, _ := decoder.Decode(mp3File)
    defer src.Close()
    
    // Convert to 16kHz mono
    pcm16, rate, _ := audpbx.ResampleToMono16(src, 16000, 4096)
    
    // Write WAV
    wavFile, _ := os.Create("output.wav")
    defer wavFile.Close()
    
    wav.WriteWAV16(wavFile, rate, pcm16)
}
```

## Supported Formats

| Format | Decoder | Encoder | Notes |
|--------|---------|---------|-------|
| WAV | ✅ | ✅ | PCM 16-bit, powered by [go-audio/wav](https://github.com/go-audio/wav) |
| MP3 | ✅ | ❌ | Decode-only, powered by [hajimehoshi/go-mp3](https://github.com/hajimehoshi/go-mp3) |
| Ogg Vorbis | ✅ | ❌ | Decode-only, powered by [jfreymuth/oggvorbis](https://github.com/jfreymuth/oggvorbis) |
| AIFF | ✅ | ❌ | PCM 16-bit decode-only, powered by [go-audio/aiff](https://github.com/go-audio/aiff) |

## Architecture

The library is organized into three main layers:

### 1. High-Level API (`audpbx` package)

Convenient functions for common tasks:

```go
// Resample audio to target rate, convert to mono, return int16 PCM
pcm16, rate, err := audpbx.ResampleToMono16(src, targetRate, bufferSize)
```

### 2. Audio Processing (`audio` package)

Low-level building blocks for custom pipelines:

```go
// Build custom processing chain
resampler := audio.NewResampler(source, 16000)
mixer := audio.NewMonoMixer(resampler)

// Read samples
buf := make([]float32, 4096)
n, err := mixer.ReadSamples(buf)
```

### 3. Format Support (`formats/*` packages)

Decoders for various audio formats:

```go
// Each format provides a Decoder
wavDecoder := wav.Decoder{}
mp3Decoder := mp3.Decoder{}
vorbisDecoder := vorbis.Decoder{}
aiffDecoder := aiff.Decoder{}

// All decoders implement the same interface
src, err := decoder.Decode(reader)
```

## Advanced Usage

### Custom Processing Pipeline

```go
package main

import (
    "github.com/ik5/audpbx/audio"
    "github.com/ik5/audpbx/formats/wav"
    "io"
    "log"
    "os"
)

func main() {
    // Open source audio
    file, _ := os.Open("input.wav")
    defer file.Close()
    
    decoder := wav.Decoder{}
    src, _ := decoder.Decode(file)
    defer src.Close()
    
    // Build processing pipeline:
    // 1. Resample to 8kHz
    resampled := audio.NewResampler(src, 8000)
    
    // 2. Convert to mono
    mono := audio.NewMonoMixer(resampled)
    
    // 3. Process in chunks
    buf := make([]float32, 4096)
    for {
        n, err := mono.ReadSamples(buf)
        if n > 0 {
            // Process samples in buf[0:n]
            processSamples(buf[:n])
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Fatal(err)
        }
    }
}

func processSamples(samples []float32) {
    // Your custom audio processing here
}
```

### Streaming Large Files

Process audio files without loading them entirely into memory:

```go
func streamProcess(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close()
    
    decoder := wav.Decoder{}
    src, err := decoder.Decode(file)
    if err != nil {
        return err
    }
    defer src.Close()
    
    // Process in small chunks - memory efficient
    const chunkSize = 4096
    buf := make([]float32, chunkSize)
    
    for {
        n, err := src.ReadSamples(buf)
        if n > 0 {
            // Stream processing - only chunkSize samples in memory
            processChunk(buf[:n])
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
    }
    
    return nil
}
```

### Working with Different Sample Rates

```go
// Common telephony rates
pcm8k, _, _ := audpbx.ResampleToMono16(src, 8000, 4096)   // 8 kHz (G.711)
pcm16k, _, _ := audpbx.ResampleToMono16(src, 16000, 4096) // 16 kHz (wideband)

// CD quality
pcm44k, _, _ := audpbx.ResampleToMono16(src, 44100, 4096) // 44.1 kHz (CD)
pcm48k, _, _ := audpbx.ResampleToMono16(src, 48000, 4096) // 48 kHz (professional)
```

## Performance

The library is designed for high performance with minimal allocations:

### Benchmarks

```
BenchmarkResampler_44100to8000-8      	   12693	     89207 ns/op	      96 B/op	       2 allocs/op
BenchmarkResampler_48000to16000-8     	   12842	     92584 ns/op	      96 B/op	       2 allocs/op
BenchmarkMonoMixer_Stereo-8           	 1000000	      1044 ns/op	       0 B/op	       0 allocs/op
BenchmarkWAVWriter-8                  	   25880	     45842 ns/op	      11 B/op	       0 allocs/op
BenchmarkAIFFDecoder-8                	  158761	      7536 ns/op	       0 B/op	       0 allocs/op
```

### Optimization Tips

1. **Buffer Size**: Larger buffers (4096-16384 samples) reduce function call overhead
2. **Reuse Buffers**: Allocate buffers once and reuse them
3. **Batch Processing**: Process audio in chunks for better cache performance
4. **Avoid Allocations**: The library is designed for near-zero allocations in hot paths

## API Reference

### Core Types

#### `audio.Source` Interface

All audio sources implement this interface:

```go
type Source interface {
    SampleRate() int                                // Sample rate in Hz
    Channels() int                                  // Number of channels
    ReadSamples(dst []float32) (n int, err error)  // Read samples
    BufSize() int                                   // Recommended buffer size
    Close() error                                   // Release resources
}
```

Samples are float32 values in the range [-1.0, 1.0].

#### `audio.Decoder` Interface

Format decoders implement this interface:

```go
type Decoder interface {
    Decode(r io.Reader) (Source, error)
}
```

### Main Functions

#### `audpbx.ResampleToMono16()`

High-level function for common audio processing:

```go
func ResampleToMono16(src audio.Source, targetRate int, bufferSize int) ([]int16, int, error)
```

- **src**: Input audio source
- **targetRate**: Target sample rate (e.g., 8000, 16000, 44100)
- **bufferSize**: Processing buffer size (typical: 4096)
- **Returns**: PCM samples, actual sample rate, error

### Format-Specific APIs

#### WAV Format

```go
// Decode WAV file
decoder := wav.Decoder{}
src, err := decoder.Decode(reader)

// Encode WAV file (mono 16-bit PCM only)
err := wav.WriteWAV16(writer, sampleRate, samples)
```

#### MP3 Format

```go
decoder := mp3.Decoder{}
src, err := decoder.Decode(reader)
// Note: MP3 decoder always outputs stereo
```

#### Ogg Vorbis Format

```go
decoder := vorbis.Decoder{}
src, err := decoder.Decode(reader)
```

#### AIFF Format

```go
decoder := aiff.Decoder{}
src, err := decoder.Decode(reader)
```

## Error Handling

The library provides specific error types for better error handling:

```go
src, err := decoder.Decode(file)
if err != nil {
    switch err {
    case wav.ErrNotWavFile:
        // Not a WAV file
    case wav.ErrOnlyPCM16bitSupported:
        // Unsupported WAV format
    case aiff.ErrNotAiffFile:
        // Not an AIFF file
    default:
        // Other errors
    }
}
```

## Testing

Run the full test suite:

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Run specific benchmark
go test -bench=BenchmarkResampler ./audio
```

## Examples

The repository includes complete examples in the `examples/` directory:

```bash
# Run the resampler example
go run examples/resampler/main.go input.wav output.wav 8000
```

More examples in the [documentation](https://pkg.go.dev/github.com/ik5/audpbx).

## Use Cases

- **VoIP/Telephony**: Convert audio to 8kHz mono for G.711 codec
- **Speech Recognition**: Prepare audio for ASR systems (typically 16kHz mono)
- **Audio Streaming**: Convert various formats to a common format for streaming
- **Podcast Processing**: Normalize audio files to consistent format
- **Audio Analysis**: Preprocess audio for feature extraction and ML pipelines

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Write tests for your changes
4. Ensure all tests pass (`go test ./...`)
5. Commit your changes (`git commit -am 'Add amazing feature'`)
6. Push to the branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

## Dependencies

- [github.com/go-audio/audio](https://github.com/go-audio/audio) - Audio buffer utilities
- [github.com/go-audio/wav](https://github.com/go-audio/wav) - WAV file support
- [github.com/go-audio/aiff](https://github.com/go-audio/aiff) - AIFF file support
- [github.com/hajimehoshi/go-mp3](https://github.com/hajimehoshi/go-mp3) - MP3 decoder
- [github.com/jfreymuth/oggvorbis](https://github.com/jfreymuth/oggvorbis) - Ogg Vorbis decoder

## License

This project is licensed under the **Eclipse Public License 2.0** — see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Built with Go 1.23+ features (range-over-int, b.Loop())
- Inspired by telephony and VoIP audio processing requirements
- Uses industry-standard audio libraries for format support
