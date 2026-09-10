# audpbx

[![Go Reference](https://pkg.go.dev/badge/github.com/ik5/audpbx.svg)](https://pkg.go.dev/github.com/ik5/audpbx)
[![Go Report Card](https://goreportcard.com/badge/github.com/ik5/audpbx)](https://goreportcard.com/report/github.com/ik5/audpbx)

**audpbx** is a high-performance Go library for audio processing, specializing in format conversion, resampling, and channel mixing. Designed for telephony and VoIP applications, it provides efficient tools for converting various audio formats to mono 16-bit PCM at any sample rate.

## Features

- **Multiple Format Support**: Decode WAV, MP3, Ogg Vorbis, and AIFF audio files
- **High-Quality Resampling**: Cubic interpolation for sample rate conversion with minimal artifacts
- **Advanced Channel Processing**: Split, extract, process individual channels from multi-channel audio
- **Channel Mixing**: Convert stereo/multi-channel audio to mono with customizable mixdown
- **Audio Effects**: Gain adjustment, normalization, channel mapping
- **High-Level API**: Powerful ProcessChannels function with flexible options (similar to pydub)
- **Performance Optimized**: Near-zero allocations, optimized for throughput
- **Simple API**: Clean, idiomatic Go interfaces
- **Streaming Support**: Process audio without loading entire files into memory
- **Comprehensive Testing**: Extensive unit tests and benchmarks
- **Beginner-Friendly Documentation**: Detailed explanations of audio concepts for newcomers

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
    mp3File, err := os.Open("input.mp3")
    if err != nil {
        log.Fatal(err)
    }
    defer mp3File.Close()

    decoder := mp3.Decoder{}
    src, err := decoder.Decode(mp3File)
    if err != nil {
        log.Fatal(err)
    }
    defer src.Close()

    // Convert to 16kHz mono
    pcm16, rate, err := audpbx.ResampleToMono16(src, 16000, 4096)
    if err != nil {
        log.Fatal(err)
    }

    // Write WAV
    wavFile, err := os.Create("output.wav")
    if err != nil {
        log.Fatal(err)
    }
    defer wavFile.Close()

    if err := wav.WriteWAV16(wavFile, rate, pcm16); err != nil {
        log.Fatal(err)
    }
}
```

### Process Multi-Channel Audio (New!)

Split stereo audio into separate channels and process them independently:

```go
package main

import (
    "fmt"
    "log"
    "os"
    
    "github.com/ik5/audpbx"
    "github.com/ik5/audpbx/audio"
    "github.com/ik5/audpbx/formats/wav"
)

func main() {
    // Open stereo audio file
    file, _ := os.Open("stereo.wav")
    defer file.Close()
    
    decoder := wav.Decoder{}
    src, _ := decoder.Decode(file)
    defer src.Close()
    
    // Split into separate mono channels, resample, and save
    result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
        audpbx.WithResample(func(ch audio.Channel) int {
            return 16000 // Resample all channels to 16kHz
        }),
        audpbx.WithGain(func(ch audio.Channel) float32 {
            return 1.2 // Boost volume by 20%
        }),
        audpbx.WithSaveToFile(func(ch audio.Channel) string {
            return fmt.Sprintf("channel_%s.wav", ch)
        }),
    )
    
    if err != nil {
        log.Fatal(err)
    }
    
    if result.HasErrors() {
        for _, e := range result.Errors {
            log.Printf("Error processing %s: %v", e.Channel, e.Err)
        }
    }
}
```

## Supported Formats

| Format | Decoder | Encoder | Notes |
|--------|---------|---------|-------|
| WAV | ✅ | ✅ | PCM 16-bit only, powered by [go-audio/wav](https://github.com/go-audio/wav) |
| MP3 | ✅ | ❌ | Decode-only, always outputs stereo, powered by [hajimehoshi/go-mp3](https://github.com/hajimehoshi/go-mp3) |
| Ogg Vorbis | ✅ | ❌ | Decode-only, powered by [jfreymuth/oggvorbis](https://github.com/jfreymuth/oggvorbis) |
| AIFF | ✅ | ❌ | PCM 16-bit decode-only, powered by [go-audio/aiff](https://github.com/go-audio/aiff) |

## Limitations

Worth knowing before you adopt this:

- **Anti-aliasing when downsampling is weak.** The filter is a single-pole
  low-pass with a fixed coefficient that does not track the resampling ratio, so
  content above the output Nyquist frequency is not properly removed. Measured
  on a 6 kHz tone resampled 44.1 kHz to 8 kHz, where it should vanish: audpbx
  leaves it at −27.9 dB, ffmpeg at −73.7 dB, a difference of about 46 dB. The
  surviving energy aliases down to 2 kHz. If you are downsampling music or
  wideband speech and care about artifacts, this matters; a polyphase FIR is the
  proper fix and is not yet implemented.
- **WAV supports only 16-bit PCM with format tag 1.** Files using
  `WAVE_FORMAT_EXTENSIBLE` (tag `0xFFFE`), which is what `ffmpeg` emits for more
  than two channels, are rejected with an "unsupported audio format" error. So
  are 8-, 24-, and 32-bit files.
- **AIFF supports only 16-bit PCM.** AIFF-C (`.aifc`) is not supported.
- **MP3 and Ogg decoding is slow** relative to ffmpeg — see
  [Performance](#performance). The cost is in the upstream decoders.
- **No encoding except WAV.** Output is mono 16-bit PCM WAV via
  `wav.WriteWAV16`.

## Architecture

The library is organized into three main layers:

### 1. High-Level API (`audpbx` package)

Convenient functions for common tasks:

```go
// Simple: Resample audio to target rate, convert to mono, return int16 PCM
pcm16, rate, err := audpbx.ResampleToMono16(src, targetRate, bufferSize)

// Advanced: Process multi-channel audio with flexible options
result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
    audpbx.WithChannels(audio.ChannelFrontLeft, audio.ChannelFrontRight),
    audpbx.WithResample(func(ch audio.Channel) int { return 48000 }),
    audpbx.WithGain(func(ch audio.Channel) float32 { return 1.5 }),
    audpbx.WithNormalize(func(ch audio.Channel) bool { return true }),
    audpbx.WithSaveToFile(func(ch audio.Channel) string {
        return fmt.Sprintf("%s.wav", ch)
    }),
)

// Split stereo/surround into individual mono channels
monoSources, err := audpbx.SplitToMonoSources(src)
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

## Audio Processing Concepts

For those new to audio processing, here's a quick guide to key concepts:

### Sample Rate
How many times per second audio is measured (in Hz). Higher rates capture more detail:
- **8000 Hz**: Telephone quality (minimum for speech)
- **16000 Hz**: Wideband speech (better quality calls)  
- **44100 Hz**: CD quality (music standard)
- **48000 Hz**: Professional audio/video

### Channels
Independent audio streams in multi-channel audio:
- **Mono (1)**: Single channel (one speaker)
- **Stereo (2)**: Two channels (left and right speakers)
- **5.1 Surround (6)**: Front-left, front-right, center, subwoofer, back-left, back-right

### Resampling
Changing the sample rate without affecting pitch or speed. Used to convert between standards or reduce bandwidth.

### Gain
Volume multiplier applied to audio:
- **1.0**: No change (original volume)
- **2.0**: Double the volume (+6 dB)
- **0.5**: Half the volume (-6 dB)

### Normalization
Automatically adjusting volume to use the full dynamic range without clipping (distortion). Finds the loudest point and scales everything proportionally.

### Mixdown/Downmixing
Combining multiple channels into mono by averaging. For example, stereo to mono: `output = (left + right) / 2`

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

### High-Level Channel Processing (New!)

Process individual channels with flexible options (similar to pydub):

```go
// Example 1: Extract and save only specific channels from 5.1 audio
result, err := audpbx.ProcessChannels(src, audio.Layout5Point1,
    // Select only front speakers
    audpbx.WithChannels(
        audio.ChannelFrontLeft,
        audio.ChannelFrontRight,
        audio.ChannelFrontCenter,
    ),
    audpbx.WithSaveToFile(func(ch audio.Channel) string {
        return fmt.Sprintf("channel_%s.wav", ch)
    }),
)

// Example 2: Resample different channels to different rates
result, err := audpbx.ProcessChannels(src, audio.Layout5Point1,
    audpbx.WithResample(func(ch audio.Channel) int {
        if ch == audio.ChannelLowFrequency {
            return 8000  // Subwoofer doesn't need high sample rate
        }
        return 44100 // Other channels at full quality
    }),
    audpbx.WithSaveToFile(func(ch audio.Channel) string {
        return fmt.Sprintf("%s.wav", ch)
    }),
)

// Example 3: Apply gain and normalization
result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
    // Boost volume by 50%
    audpbx.WithGain(func(ch audio.Channel) float32 {
        return 1.5
    }),
    // Then normalize to prevent clipping
    audpbx.WithNormalize(func(ch audio.Channel) bool {
        return true
    }),
    audpbx.WithSaveToFile(func(ch audio.Channel) string {
        return fmt.Sprintf("%s_processed.wav", ch)
    }),
)

// Example 4: Mix all channels to mono
result, err := audpbx.ProcessChannels(src, audio.Layout5Point1,
    audpbx.WithMixdown("mono_mix.wav"),
)

// Example 5: Process channels concurrently for better performance
result, err := audpbx.ProcessChannels(src, audio.Layout5Point1,
    audpbx.WithConcurrency(4), // Process 4 channels at a time
    audpbx.WithResample(func(ch audio.Channel) int { return 48000 }),
    audpbx.WithSaveToFile(func(ch audio.Channel) string {
        return fmt.Sprintf("%s_48k.wav", ch)
    }),
)

// Example 6: Swap left and right channels
result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
    audpbx.WithChannelMapping(map[audio.Channel]audio.Channel{
        audio.ChannelFrontLeft:  audio.ChannelFrontRight,
        audio.ChannelFrontRight: audio.ChannelFrontLeft,
    }),
    audpbx.WithSaveToFile(func(ch audio.Channel) string {
        return fmt.Sprintf("%s_swapped.wav", ch)
    }),
)
```

### Available Processing Options

The `ProcessChannels` function supports these options:

- **`WithChannels`**: Select specific channels to process
- **`WithResample`**: Change sample rate per channel
- **`WithGain`**: Adjust volume (multiply samples)
- **`WithNormalize`**: Auto-maximize volume without clipping
- **`WithMixdown`**: Combine all channels to mono
- **`WithSaveToFile`**: Save to files
- **`WithSaveToWriter`**: Save to io.Writer
- **`WithConcurrency`**: Process multiple channels simultaneously
- **`WithErrorHandler`**: Handle errors per channel
- **`WithChannelMapping`**: Swap/remap channels
- **`WithProcessor`**: Apply custom processing
- **`WithProgress`**: Track processing progress
- **`WithBufferSize`**: Configure buffer size

## Performance

### End-to-end conversion

Converting a 103 second 44.1 kHz stereo recording to 8 kHz mono 16-bit PCM,
measured on an Intel i7-6820HQ, with `ffmpeg` shown for reference:

| Source format | audpbx    | ffmpeg | Allocations |
|---------------|-----------|--------|-------------|
| WAV           | **59 ms** | 166 ms | 71          |
| AIFF          | **58 ms** | 151 ms | 70          |
| Ogg Vorbis    | 467 ms    | 213 ms | 58 K        |
| MP3           | 2043 ms   | 198 ms | 552 K       |

For PCM sources the pipeline is roughly 2.5-3x faster than ffmpeg and allocates
about 70 times for the whole file — the conversion completes without triggering
a single garbage collection.

MP3 and Ogg are bound inside their third-party decoders
(`hajimehoshi/go-mp3` and `jfreymuth/vorbis`), not in this library's resampling
code. Closing that gap means patching or replacing those decoders. See
[examples/profile_resampler/PROFILING_GUIDE.md](examples/profile_resampler/PROFILING_GUIDE.md).

Reproduce with:

```bash
go test ./internal/perfbench/ -bench . -benchtime 5x -benchmem
```

### Component benchmarks

```
# Resampling (100,000 frames, 44.1 kHz stereo, per full stream)
BenchmarkResampler_Downsample-8         4.97ms    73KB     3 allocs/op
BenchmarkResampler_Upsample-8           2.77ms    73KB     3 allocs/op
BenchmarkResampler_SmallBuffer-8        4.90ms    73KB     3 allocs/op
BenchmarkResampler_MultiChannel-8       5.84ms   270KB     3 allocs/op
BenchmarkResampler_ReadSamples-8       557.2µs      0B     0 allocs/op

# Mono mixing
BenchmarkMonoMixer_Passthrough-8       349.0µs      0B     0 allocs/op
BenchmarkMonoMixer_StereoToMono-8       4.34ms      0B     0 allocs/op

# High-level API
BenchmarkResampleToMono16-8             2.35ms   623KB     9 allocs/op
BenchmarkResampleToMono16_Upsample-8    1.75ms  2786KB     9 allocs/op
BenchmarkSplitToMonoSources-8          226.0ns    352B     3 allocs/op
BenchmarkProcessAndSaveChannel-8       452.6µs   884KB    15 allocs/op
BenchmarkProcessChannels_Stereo-8      905.7µs  1034KB    28 allocs/op
BenchmarkProcessChannels_5Point1-8      2.97ms  1447KB    69 allocs/op
BenchmarkProcessChannels_WithResample-8 1.07ms  1096KB    34 allocs/op
BenchmarkProcessChannels_Concurrent-8   2.20ms  1449KB    90 allocs/op
```

The three allocations per resampled stream are one-time setup in
`NewResampler`; `ReadSamples` itself allocates nothing once streaming has
started.

### Optimization Tips

1. **Buffer size barely matters.** `Resampler` reads its source in fixed
   internal blocks regardless of how much you ask for per call, so a 64-sample
   buffer and a 4096-sample buffer perform the same (4.90 ms vs 4.97 ms above).
   Pick whatever suits your call site; `audio.DefaultBufSize` is a reasonable
   default.
2. **Reuse buffers.** Allocate once outside your read loop and reuse.
3. **Stream rather than buffer whole files.** The pipeline is fully streaming;
   only `ResampleToMono16` collects the entire result in memory.
4. **Prefer WAV or AIFF sources** when you control the input format. Decoding,
   not resampling, is what costs for MP3 and Ogg.

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

#### Constants

```go
// audio: the default read size a Source reports from BufSize(), and a
// reasonable default for sizing your own read buffers.
audio.DefaultBufSize   // 4096
```

```go
// utils: PCM sample geometry, shared by the decoders and sample converters.
utils.BitsPerByte                                      // 8
utils.BitDepth8, BitDepth16, BitDepth24, BitDepth32    // 8, 16, 24, 32
utils.BytesPerSample8 ... BytesPerSample32             // 1, 2, 3, 4
utils.SampleScale8    ... SampleScale32                // 2^(bits-1)
```

`SampleScaleN` maps a normalized float sample in `[-1, 1]` onto the integer
range of that bit depth; dividing a raw integer sample by it yields a float in
`[-1, 1)`.

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

#### `audpbx.ProcessChannels()` (New!)

Advanced multi-channel processing with flexible options:

```go
func ProcessChannels(src audio.Source, layout audio.Channel, opts ...ChannelOption) (*ProcessingResult, error)
```

- **src**: Input audio source (stereo, 5.1, etc.)
- **layout**: Channel layout (e.g., `audio.LayoutStereo`, `audio.Layout5Point1`)
- **opts**: Processing options (WithResample, WithGain, WithNormalize, etc.)
- **Returns**: Processing result with any errors, error

Example:
```go
result, err := audpbx.ProcessChannels(src, audio.LayoutStereo,
    audpbx.WithResample(func(ch audio.Channel) int { return 48000 }),
    audpbx.WithGain(func(ch audio.Channel) float32 { return 1.5 }),
    audpbx.WithSaveToFile(func(ch audio.Channel) string {
        return fmt.Sprintf("%s.wav", ch)
    }),
)
```

#### `audpbx.SplitToMonoSources()` (New!)

Split multi-channel audio into separate mono sources:

```go
func SplitToMonoSources(src audio.Source) ([]audio.Source, error)
```

- **src**: Multi-channel audio source
- **Returns**: Slice of mono sources (one per channel), error

#### `audpbx.ProcessAndSaveChannel()` (New!)

Process and save a single mono channel:

```go
func ProcessAndSaveChannel(mono audio.Source, rate int, path string) error
```

- **mono**: Mono audio source (must be 1 channel)
- **rate**: Target sample rate
- **path**: Output file path
- **Returns**: error

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

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run component benchmarks
go test -bench=. ./...

# Run a specific benchmark
go test -bench=BenchmarkResampler ./audio
```

### Real-file benchmarks

Component benchmarks run against in-memory sources, which cannot see decoder
call overhead or file I/O — historically the dominant cost in this pipeline. To
measure end-to-end performance through real decoders reading real files:

```bash
go test ./internal/perfbench/ -bench . -benchtime 5x -benchmem
```

These skip automatically if the test assets under `examples/testdata/` are not
present. See
[examples/profile_resampler/PROFILING_GUIDE.md](examples/profile_resampler/PROFILING_GUIDE.md)
for why this distinction matters.

### Performance regression tests

Performance here is protected by ordinary `go test` assertions, not by timing.
Each was verified by reintroducing the defect and confirming the test fails:

| Test | Catches |
|------|---------|
| `audio.TestResampler_ReadsSourceInBlocks` | The resampler reading its source frame-at-a-time |
| `TestResampleToMono16_AllocsDoNotScaleWithLength` | A per-frame allocation anywhere in the pipeline |
| `audio.TestResampler_ZeroAllocsSteadyState` | An allocation on the resampler's streaming path |
| `wav.TestSource_ReadSamples_ZeroAllocsSteadyState` | A per-call allocation in the WAV decoder |
| `aiff.TestSource_ReadSamples_RawZeroAllocsSteadyState` | The AIFF decoder losing its direct-decode path |

They assert on **call patterns and allocation counts**, which are deterministic,
rather than on wall-clock time, which would flake. That choice matters: the
original 40x slowdown was per-call overhead, which changes neither the output
nor the allocation count — only how often the source is read. Reverting to
frame-at-a-time reads makes the WAV pipeline 41x slower while leaving every
correctness and allocation test green, so the call-pattern test is the only
thing that sees it.

Throughput itself is not asserted; that would need benchmark tracking against a
stored baseline in CI.

## Examples

The repository includes complete examples in the `examples/` directory:

```bash
# Convert to the default 8 kHz
go run ./examples/resampler input.wav output.wav

# Or specify a target sample rate
go run ./examples/resampler input.mp3 output.wav 16000
```

The converter accepts any supported input format and picks the decoder from the
file extension.

Performance analysis tools live in
[examples/profile_resampler](examples/profile_resampler).

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

- Requires Go 1.25+ (uses range-over-int and `b.Loop()`)
- Inspired by telephony and VoIP audio processing requirements
- Uses industry-standard audio libraries for format support

## Caveat

This project is a vibe coding-based package created for the following reasons:

1. Allows projects such as APIs to handle common audio formats and convert them to Open Source PBX-ready audio (such as [Asterisk PBX](https://www.asterisk.org/) and [FreeSWITCH](https://signalwire.com/freeswitch)).
2. Native Go code first (additional non-native formats will be added in the future).
3. Good documentation.
4. Comprehensive unit testing.
5. Zero-allocation code.
6. Testing Claude Code for vibe coding.

## TODO

The following features are planned:

* [ ] Support Opus format.
* [ ] Support AAC format (as binding with static linking, static building, dynamic library - building based on tags).
* [ ] Additional audio test files for each format.
* [ ] **Selectable resampling algorithm.** Allow the caller to choose the
      resampling method rather than hard-coding cubic interpolation, trading
      throughput against fidelity per use case:

  | Method | Characteristics |
  |--------|-----------------|
  | Linear | Cheapest; adequate when the source is already band-limited or when latency dominates |
  | Cubic (Catmull-Rom) | Current behaviour; good general-purpose default |
  | Polyphase FIR | Highest fidelity; proper band-limiting for large decimation ratios |

  Design notes for whoever picks this up:

  - Cubic should remain the default so existing callers are unaffected. A
    functional option on `NewResampler` (for example `WithInterpolator`) keeps
    the current signature valid.
  - The polyphase option also resolves the anti-aliasing shortfall described
    under [Limitations](#limitations), which cannot be addressed by retuning
    the existing one-pole filter. A polyphase FIR performs band-limiting and
    rate conversion in a single operation, so it replaces both the cubic
    interpolator and the one-pole filter rather than being layered on top of
    them.
  - Common telephony conversions are exact rational ratios (44.1 kHz to 8 kHz
    is 441/80), so a polyphase implementation can precompute one coefficient
    set per phase and evaluate only the output samples actually required.
  - Selecting a method changes output samples. The bit-exact comparisons and
    the quality measurements in
    [examples/profile_resampler/PROFILING_GUIDE.md](examples/profile_resampler/PROFILING_GUIDE.md)
    should be extended to cover each method independently.

  This is a design placeholder only; implementation is deliberately out of
  scope for the current branch.

* [ ] Accept `WAVE_FORMAT_EXTENSIBLE` WAV files, and bit depths other than 16.
* [ ] Close the MP3 and Ogg decode gap, which needs work in (or replacement of)
      the upstream decoders.
