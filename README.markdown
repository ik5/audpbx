# audpbx

[![Go Reference](https://pkg.go.dev/badge/github.com/ik5/audpbx.svg)](https://pkg.go.dev/github.com/ik5/audpbx)
[![Go Report Card](https://goreportcard.com/badge/github.com/ik5/audpbx)](https://goreportcard.com/report/github.com/ik5/audpbx)

**audpbx** is a Go audio manipulation library: decode, encode (when an encoder exists), resample, mix, concat, and split — in-process, with no external commands (no ffmpeg, no sox).

Telecom tooling is first-class (G.7xx, Opus, GSM, raw/headerless codecs, PBX file dialects, and the helpers those need). PBX is one environment in that set, not the whole set. Non-telecom conversions are allowed. All usual sample rates, both endians, and basic PCM/container formatting are in scope.

## Scope

**audpbx is an audio manipulation library.** It resamples, filters, mixes,
splits, converts and analyses audio, and it provides the abstractions that let
audio flow through a processing pipeline.

**It does not implement codecs.** Codec implementations are external
dependencies that audpbx imports and wraps — see
[Dependencies](#dependencies). Adding support for a format or codec means
integrating an existing implementation behind a common interface, not writing
compression or decompression algorithms here.

What that means in practice:

| audpbx does | audpbx delegates |
|-------------|------------------|
| Resampling, mixing, gain, filtering, analysis | Codec compression and decompression |
| Container and header parsing, metadata | The signal processing inside a codec |
| Tone generation, composition, channel routing | |
| The `Source`, `Decoder` and registry abstractions | |
| Thin wrappers adapting third-party codecs to them | |

A consequence worth knowing: where decode throughput or codec quality is
limited, the limit usually lives in the upstream decoder rather than in audpbx.
See [Performance](#performance) and [Limitations](#limitations).

### Dependency policy: native Go first

**Native Go is the default, and an external (cgo) dependency is only acceptable
when both of these hold:**

1. **No suitable Go implementation exists** — neither a maintained third-party
   package nor a reasonable one to write; and
2. **Writing it in Go would be disproportionate effort** for a package of this
   scope.

Both tests must pass. "No Go package exists" is not on its own sufficient — if
the thing is tractable to implement, it gets implemented in Go. Equally,
"implementing it would be hard" is not sufficient either, if a usable Go package
is already available.

When a dependency does clear both tests, these rules apply:

- **The default build of this module does not use cgo.** Building `audpbx`
  must not require a C toolchain, a system library, or `pkg-config`.
- **Extra containers and codecs plug in from outside this tree.** Implement
  the interface and `Register` it in your own module (or an opt-in companion
  module). That is how this package grows without forking, and how a cgo or
  LGPL implementation stays out of this `go.mod`. Build tags inside *this*
  repo do not do that job.
- **Permissive licences are required for dependencies of this package.**
  GPL and other licences that would copyleft this library or its commercial
  users are out. LGPL, if used at all, lives only in a separate module the
  application imports and registers.
- **Vendored source beats a linked library.** A public-domain or
  CC0 single-header file compiled by cgo needs no system dependency, no
  presence check and no build-tag matrix, which makes it materially cheaper than
  linking.
- **Maintenance status is part of the decision.** A dependency's archival status
  and last release are checked before adoption, and re-checked as part of the
  pre-release audit — see
  [current_gaps.md](current_gaps.md#qa-8-no-pre-release-bug-and-security-audit).
- **An archived dependency with a known security defect is not acceptable.**
  Removing it takes precedence over feature work. Mitigating around it, or
  documenting it as a caveat, is not sufficient: depending on unmaintained code
  with a known unfixable flaw — and telling users about it — is worse advice
  than not depending on it at all. An archived upstream cannot be patched, so
  such a mitigation would be permanent rather than temporary.

Worked examples of the rule in practice, for audio codecs:

| Situation | Outcome |
|---|---|
| A maintained Go package exists | **Adopt it** — e.g. G.711 |
| No suitable Go package, but the codec is tractable | **Write it in Go**, as a separate module this library imports — e.g. G.726 |
| No Go implementation and the effort is disproportionate | **cgo in a separate module, registered** — e.g. G.729, where CS-ACELP is weeks of specialist DSP |

Note the second row: codecs written rather than adopted live in their own
modules, not inside `audpbx` — consistent with this library not implementing
codecs. See
[current_gaps.md](current_gaps.md#fmt-4-telco-and-voip-codecs-absent-user-2)
for how each codec has been decided and why.

## Features

- **Decode** WAV, MP3, Ogg Vorbis, and AIFF into a streaming `audio.Source`
- **Resample and mix** to a chosen rate and channel layout (cubic interpolation today; selectable algorithms are planned)
- **Channel processing**: split, extract, mixdown, gain, normalize via `ProcessChannels`
- **Encode** to mono 16-bit WAV today; other encoders plug in when they exist
- **Registry**: extra containers and codecs register from outside this tree
- **In-process Go**: no ffmpeg/sox; default build has no cgo

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
  wideband speech and care about artifacts, this matters. A polyphase FIR is the
  proper fix and is not yet implemented — resampling is audpbx's own work, so
  this one is in scope (see [Scope](#scope)).
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

Thin wrappers that adapt third-party codec libraries to the `audio.Source`
interface. Each package handles container parsing, sample-format conversion and
the `Source` contract; the codec itself lives in the dependency (see
[Scope](#scope) and [Dependencies](#dependencies)).

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
- **`WithProgress`**: Accepted, but **not yet implemented** — the callback is
  stored and never invoked, so it currently reports nothing
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

- **Telecom media and PBX**: Convert to and from rates and dialects a switch or media server will play (8 kHz / 16 kHz / 48 kHz, G.711, raw `.sln` / `.ulaw` / `.alaw`, and related formats). Asterisk, FreeSWITCH, WebRTC, and other telecom stacks are all in this set.
- **Recordings in and out**: Decode a capture from a media server and encode it to something a person or another system can keep (WAV, raw PCM, G.711 — when an encoder exists).
- **Join and split files**: Concatenate several recordings into one, or cut a longer recording into smaller ones (time range, markers, silence). Tooling only.
- **Programmatically native audio manipulation**: Filtering, resampling, mixing, gain, and the rest of the pipeline as a Go library — not only for the telecom world.
- **Ordinary audio conversion**: Any usual sample rate, both endians, basic PCM/containers — including work that is not a phone call. Speech recognition, podcasts, streaming, and ML preprocessing remain possible here; they are not headline use cases.

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

Codec and container implementations are delegated to these libraries; audpbx
wraps them behind `audio.Source`. See [Scope](#scope).

- [github.com/go-audio/audio](https://github.com/go-audio/audio) - Audio buffer utilities
- [github.com/go-audio/wav](https://github.com/go-audio/wav) - WAV container and PCM codec
- [github.com/go-audio/aiff](https://github.com/go-audio/aiff) - AIFF container and PCM codec
- [github.com/hajimehoshi/go-mp3](https://github.com/hajimehoshi/go-mp3) - MP3 decoder
- [github.com/jfreymuth/oggvorbis](https://github.com/jfreymuth/oggvorbis) - Ogg Vorbis decoder

Adding a format means adding a dependency and a wrapper, not implementing a
codec here.

## License

This project is licensed under the **Eclipse Public License 2.0** — see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Requires Go 1.25+ (uses range-over-int and `b.Loop()`)
- Inspired by telecommunication media and PBX audio processing requirements
- Uses industry-standard audio libraries for format support

## Caveat

This project is a vibe coding-based package created for the following reasons:

1. Lets a Go service manipulate audio in-process (no ffmpeg/sox), with first-class tooling for telecommunication media and PBX platforms such as [Asterisk](https://www.asterisk.org/) and [FreeSWITCH](https://signalwire.com/freeswitch).
2. Native Go code first — see [Dependency policy](#dependency-policy-native-go-first) for the rule and when an external dependency is acceptable.
3. Godoc for the API, with examples per API call by design.
4. Unit tests and benchmarks for implementations.
5. An attempt at zero-allocation code.
6. Testing Claude Code for vibe coding.

Planned work is not listed here. Decisions live in [roadmap_v1.md](roadmap_v1.md); the full inventory is [current_gaps.md](current_gaps.md).
