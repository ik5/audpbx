# audpbx Examples

This directory contains working examples, integration tests, and performance benchmarks for the `audpbx` audio processing library.

> **Note:** These are not traditional "examples" in the Go sense—they are practical demonstrations, integration tests, and performance analysis tools that show how to use both the high-level and low-level APIs in real-world scenarios.

## Purpose

The examples serve multiple purposes:

1. **📚 Learning Resource** - Understand how to use the library APIs
2. **✅ Integration Tests** - Verify end-to-end functionality with real audio files
3. **⚡ Performance Testing** - Profile and optimize audio processing pipelines
4. **🔧 Implementation Reference** - See best practices for common tasks
5. **📊 Benchmarking** - Measure throughput and identify bottlenecks

## Directory Structure

```
examples/
├── README.md                       # This file
├── TESTDATA_ORGANIZATION.md        # Test file management guide
│
├── testdata/                       # Shared test audio files
│   ├── LICENSE                     # Attribution for test files
│   ├── README.md                   # Test file documentation
│   ├── METADATA_PRESERVATION.md    # Metadata handling guide
│   ├── SCRIPT_USAGE.md             # Test file management script docs
│   ├── manage_testdata.sh          # Automated test file management
│   └── [audio files in multiple formats]
│
├── resampler/                      # Basic resampling example
│   ├── main.go                     # Simple format conversion
│   └── go.mod                      # Module dependencies
│
└── profile_resampler/              # Performance profiling tools
    ├── main.go                     # Profiling with timing breakdown
    ├── instrumented_resample.go    # ResampleToMono16 with timing points
    ├── benchmark_test.go           # Component benchmarks (in-memory source)
    ├── README.md                   # Directory index
    ├── README_PROFILING.md         # Walkthrough of the tools
    ├── QUICK_DEBUG.md              # Command reference
    ├── PROFILING_GUIDE.md          # Case study and bottleneck analysis
    ├── profile.sh                  # Automated profiling script
    ├── Makefile                    # Convenient make commands
    └── go.mod                      # Module dependencies
```

End-to-end pipeline benchmarks live outside this directory, in
[`internal/perfbench`](../internal/perfbench), because they need real decoders
reading real files rather than an in-memory source.

## Examples Overview

### 1. `resampler/` - Basic Audio Conversion

**Type:** Integration test + Tutorial example

**What it demonstrates:**
- Opening audio files (WAV, MP3, OGG, AIFF)
- Decoding with format-specific decoders
- Using the high-level `ResampleToMono16()` API
- Writing WAV output files
- Multi-format support via registry pattern

**Use case:** Learning the basic workflow of audio format conversion.

**Run it:**
```bash
# From the repository root
go run ./examples/resampler input.wav output.wav

# Works with any supported format; the decoder is picked from the extension
go run ./examples/resampler song.mp3 output.wav
go run ./examples/resampler audio.ogg output.wav

# Target sample rate is optional and defaults to 8000 Hz
go run ./examples/resampler song.mp3 output.wav 16000
```

**What it tests:**
- ✅ End-to-end format conversion
- ✅ Decoder integration
- ✅ Resampling to an arbitrary rate (8 kHz telephony by default)
- ✅ Mono mixdown from stereo/multi-channel
- ✅ WAV file writing

**APIs demonstrated:**
- **High-level:** `audpbx.ResampleToMono16()`
- **Format support:** `audio.Registry`, decoder interfaces
- **Output:** `wav.WriteWAV16()`

### 2. `profile_resampler/` - Performance Analysis Tools

**Type:** Performance testing + Profiling toolkit + Integration test

**What it demonstrates:**
- CPU and memory profiling
- Timing breakdown of processing stages
- Bottleneck identification
- Performance optimization strategies
- Benchmarking individual components

**Use case:** Analyzing performance issues and optimizing audio processing pipelines.

**Run it:**
```bash
cd profile_resampler

# Basic timing analysis
go run main.go input.wav output.wav

# With CPU profiling
go run main.go input.wav output.wav --cpu-profile=cpu.prof
go tool pprof -http=:8080 cpu.prof

# Automated profiling with script
./profile.sh input.wav output.wav

# Run benchmarks
go test -bench=. -benchmem
```

**What it tests:**
- ⚡ Processing speed and throughput
- 📊 Memory allocation patterns
- 🔍 CPU hotspots (in practice: decoder overhead, not interpolation)
- 💾 Memory usage with large files
- 🎯 Buffer size impact — expect almost none; the resampler reads its source in
  fixed internal blocks regardless of the caller's buffer size

**APIs demonstrated:**
- **Low-level pipeline:** `audio.NewResampler()`, `audio.NewMonoMixer()`
- **Streaming:** Chunk-based processing with buffers
- **Profiling:** Go's `runtime/pprof` integration

**Tools provided:**
- `main.go` - Profiling tool with detailed timing
- `benchmark_test.go` - Micro-benchmarks
- `profile.sh` - Automated profiling workflow
- `Makefile` - Quick commands (`make profile`, `make bench`)
- Comprehensive documentation (3 guide files)

## Test Data

All examples share test audio files from the `testdata/` directory. These files:

- **Have proper licensing** - All CC-BY 4.0 from Jamendo and Freesound
- **Preserve metadata** - Artist, title, license info embedded
- **Cover multiple formats** - OGG, MP3, WAV, AIFF
- **Vary in size** - From 1.4 MB to 167 MB for different test scenarios
- **Include documentation** - Complete attribution and usage guides

### Available Test Files

| File | Format | Size | Duration | Best For |
|------|--------|------|----------|----------|
| Capricerie (Daniel Bautista) | OGG | 1.4 MB | 1:43 | Quick tests, format validation |
| Sneakers (GloryToTheMachine) | MP3 | 2.1 MB | 1:30 | Standard integration tests |
| Bells Drone (kevp888) | WAV | 167 MB | 16:29 | **Performance testing, large files** |

All files are available in 4 formats (OGG, MP3, WAV, AIFF) with preserved
metadata, except that the two largest `bells_drone` renderings (WAV and AIFF)
are excluded from version control; fetch them with
[testdata/manage_testdata.sh](testdata/manage_testdata.sh).

The `Capricerie` file is what `internal/perfbench` measures against, in all four
formats.

**See:** [testdata/README.md](testdata/README.md) for complete details.

**Manage test files:** [testdata/manage_testdata.sh](testdata/manage_testdata.sh) script.

## Usage Patterns

### Pattern 1: Basic Format Conversion (High-Level API)

**Example:** `resampler/main.go`

```go
// 1. Register decoders
reg := audio.NewRegistry()
reg.Register("wav", wav.Decoder{})
reg.Register("mp3", mp3.Decoder{})

// 2. Open and decode
decoder, _ := reg.Get("mp3")
src, _ := decoder.Decode(file)

// 3. Process (high-level API)
pcm16, rate, _ := audpbx.ResampleToMono16(src, 8000, 4096)

// 4. Write output
wav.WriteWAV16(output, rate, pcm16)
```

**When to use:**
- Simple format conversion tasks
- Quick prototyping
- You want the library to handle everything

### Pattern 2: Custom Processing Pipeline (Low-Level API)

**Example:** `profile_resampler/` (streaming mode)

```go
// 1. Build custom pipeline
resampler := audio.NewResampler(src, 16000)
mono := audio.NewMonoMixer(resampler)

// 2. Stream processing
buf := make([]float32, 4096)
for {
    n, err := mono.ReadSamples(buf)
    if n > 0 {
        // Custom processing on buf[:n]
        processCustom(buf[:n])
    }
    if err == io.EOF {
        break
    }
}
```

**When to use:**
- Streaming large files (memory efficient)
- Custom audio processing (effects, analysis)
- Need fine-grained control
- Building complex pipelines

### Pattern 3: Performance Analysis

**Example:** `profile_resampler/`

```go
// Profile specific components
go test -bench=BenchmarkCubicInterpolate -cpuprofile=cpu.prof
go tool pprof cpu.prof

// Profile end-to-end
go run main.go largefile.wav output.wav --cpu-profile=cpu.prof --mem-profile=mem.prof
```

**When to use:**
- Identifying performance bottlenecks
- Optimizing processing pipelines
- Analyzing memory usage
- Testing with different buffer sizes

## Running Examples

### Prerequisites

```bash
# Ensure audpbx is installed
go get github.com/ik5/audpbx

# Navigate to example directory
cd examples/resampler  # or profile_resampler
```

### Basic Execution

```bash
# Resampler example, from the repository root
go run ./examples/resampler \
    examples/testdata/Daniel_Bautista_-_Capricerie_No._5_\(Bach\,_Paganini\).ogg \
    output.wav

# With an explicit target sample rate
go run ./examples/resampler examples/testdata/some_file.mp3 output.wav 16000

# Profile example (its own module, so run it from its directory)
cd examples/profile_resampler
go run main.go ../testdata/844152__kevp888__020a_100111_0243_exp02_bells_drone.wav output.wav
```

Note that the large `bells_drone` WAV and AIFF files are excluded from version
control (see `.gitignore`); use `testdata/manage_testdata.sh` to fetch them.

### With Profiling

```bash
cd profile_resampler

# CPU profiling
go run main.go ../testdata/largefile.wav output.wav --cpu-profile=cpu.prof
go tool pprof -http=:8080 cpu.prof

# Memory profiling
go run main.go ../testdata/largefile.wav output.wav --mem-profile=mem.prof
go tool pprof -http=:8080 mem.prof

# Automated (recommended)
./profile.sh ../testdata/largefile.wav
```

### Run Benchmarks

End-to-end, through real decoders reading real files. This is the measurement
that reflects actual performance:

```bash
# From the repository root
go test ./internal/perfbench/ -bench . -benchtime 5x -benchmem
```

Component benchmarks, against an in-memory source. Useful for comparing two
implementations of an inner loop, but blind to decoder overhead and I/O:

```bash
cd examples/profile_resampler

# All benchmarks
go test -bench=. -benchmem

# Specific component
go test -bench=BenchmarkCubicInterpolate -benchmem

# With CPU profile
go test -bench=. -benchmem -cpuprofile=bench.prof
```

## What These Examples Test

### Integration Testing

The examples serve as integration tests by:

1. **Format Decoding**
   - WAV, MP3, OGG, AIFF support
   - Error handling for unsupported formats
   - Metadata handling

2. **Audio Processing**
   - Resampling (upsampling and downsampling)
   - Channel mixing (stereo → mono)
   - Sample format conversion (float32 → int16)

3. **File I/O**
   - Reading various audio formats
   - Writing WAV files
   - Streaming large files

4. **API Contracts**
   - High-level API simplicity
   - Low-level API flexibility
   - Error handling patterns

### Performance Regression Testing

Separately from the profiling tools, these properties are asserted by tests in
the main module, each verified by reintroducing the defect it guards:

| Test | Catches |
|------|---------|
| `audio.TestResampler_ReadsSourceInBlocks` | Frame-at-a-time source reads (41x slowdown) |
| `TestResampleToMono16_AllocsDoNotScaleWithLength` | A per-frame allocation anywhere in the pipeline |
| `wav`/`aiff` `TestSource_ReadSamples_*ZeroAllocs*` | Decoders allocating per call |
| `aiff.TestSource_ReadSamples_RawBigEndian` | AIFF sample byte order |

### Performance Testing

The profiling tools measure:

1. **Throughput**
   - MB/s processing rate
   - Realtime ratio (how fast vs. audio duration)

2. **CPU Hotspots**
   - Decoder cost, which dominates for MP3 and Ogg
   - Syscall time, which should be negligible and is the first thing to check
     if throughput regresses
   - Resampling and channel mixing, both small in practice

3. **Memory**
   - Allocation patterns (count matters more than size here — a per-call
     allocation of a few bytes is invisible by size and glaring by count)
   - GC pressure
   - Buffer efficiency

4. **Scalability**
   - Performance with different file sizes
   - Impact of buffer size (expect almost none)
   - Streaming efficiency

## Understanding Performance

### Typical Performance Characteristics

From `profile_resampler/` on the 17.4 MB WAV test file (103 s, 44.1 kHz stereo
to 8 kHz mono):

```
=== Performance Breakdown ===
File open:          0.029 ms (  0.0%)
Decode setup:       0.061 ms (  0.1%)
Resample/Mix:      61.165 ms ( 97.0%) ← Main processing
Write output:       1.465 ms (  2.3%)
---
TOTAL:             63.032 ms

Num GC:          0
Speed:           275.45 MB/s
Processing rate: 1637.33x realtime
```

End-to-end comparison against `ffmpeg` for the same conversion:

| Source format | audpbx    | ffmpeg | Allocations |
|---------------|-----------|--------|-------------|
| WAV           | **59 ms** | 166 ms | 71          |
| AIFF          | **58 ms** | 151 ms | 70          |
| Ogg Vorbis    | 467 ms    | 213 ms | 58 K        |
| MP3           | 2043 ms   | 198 ms | 552 K       |

### Where the time actually goes

This is worth reading, because the intuitive answer is wrong.

Cubic interpolation is **not** a bottleneck. It never appeared in the top twenty
of a CPU profile even when this pipeline ran 40x slower than ffmpeg. The real
cost then was `read(2)`, at 67% of all CPU time, because the resampler pulled
one frame at a time from the decoder and the decoder issued one syscall per
4 bytes. That has been fixed by reading the source in blocks.

What remains, by format:

1. **WAV and AIFF** — bounded by memory bandwidth. Little left to win.
2. **MP3** (~2 s) — almost entirely inside `hajimehoshi/go-mp3`: subband
   synthesis, its IMDCT window (a fresh allocation per call), and `math.Pow` in
   requantization.
3. **Ogg** (~470 ms) — almost entirely inside `jfreymuth/vorbis`.
4. **Resampling and mixing themselves** — a small fraction in every case.

**See:** [profile_resampler/PROFILING_GUIDE.md](profile_resampler/PROFILING_GUIDE.md)
for the full case study, including profile output and the root cause.

### Benchmark with real files, not mocks

`profile_resampler/benchmark_test.go` uses an in-memory source. It performs no
syscalls and allocates nothing per call, so it cannot see the costs that
dominate real runs — it reported healthy numbers throughout the period when the
pipeline was 40x too slow.

Use it to compare inner-loop implementations. To measure the pipeline, use the
real-file benchmarks in [`internal/perfbench`](../internal/perfbench):

```bash
go test ./internal/perfbench/ -bench . -benchtime 5x -benchmem
```

### Regressions are caught by tests, not by these benchmarks

A benchmark cannot fail, so the properties that must hold are asserted by
ordinary tests. They check call patterns and allocation counts, which are
deterministic, rather than elapsed time, which would flake:

```bash
go test -run 'ReadsSourceInBlocks|AllocsDoNotScale|ZeroAllocsSteadyState' ./...
```

The most important of these is `audio.TestResampler_ReadsSourceInBlocks`, and
the reason is worth understanding: making the resampler read frame-at-a-time
again costs 41x throughput while changing neither the output nor the allocation
count. Correctness tests pass, allocation tests pass — only the number and size
of reads into the source gives it away.

See
[profile_resampler/PROFILING_GUIDE.md](profile_resampler/PROFILING_GUIDE.md#guarding-against-regressions)
for the full list and what each one catches.

## API Level Demonstrations

### High-Level API (audpbx package)

**Demonstrated in:** `resampler/main.go`

```go
// One-line audio processing
pcm16, rate, err := audpbx.ResampleToMono16(src, 8000, 4096)
```

**Benefits:**
- Simple, concise
- Handles common cases
- Good for quick tasks
- Beginner-friendly

**Use when:**
- Converting formats simply
- Don't need custom processing
- Prototyping
- Learning the library

### Low-Level API (audio package)

**Demonstrated in:** `profile_resampler/` internals, custom pipelines

```go
// Build custom pipeline
resampler := audio.NewResampler(src, targetRate)
mixer := audio.NewMonoMixer(resampler)
// ... custom processing
```

**Benefits:**
- Full control
- Memory efficient streaming
- Custom processing steps
- Performance optimization

**Use when:**
- Processing large files
- Custom audio effects
- Need fine-grained control
- Building complex pipelines

### Format APIs (formats/* packages)

**Demonstrated in:** Both examples

```go
// Each format has a decoder
wavDec := wav.Decoder{}
mp3Dec := mp3.Decoder{}
oggDec := vorbis.Decoder{}

// All implement the same interface
src, err := decoder.Decode(reader)
```

**Benefits:**
- Consistent interface
- Easy to add formats
- Format-agnostic code

## Learning Path

### 1. Start with Basic Example
```bash
# From the repository root
go run ./examples/resampler \
    examples/testdata/Daniel_Bautista_-_Capricerie_No._5_\(Bach\,_Paganini\).ogg \
    output.wav
```

**Learn:**
- Basic workflow
- High-level API usage
- Format support

### 2. Read the Code
- Study `resampler/main.go` (~115 lines)
- Understand decoder registration
- See error handling patterns

### 3. Experiment
- Try different formats
- Change the target sample rate (third argument)
- Try different buffer sizes — and notice how little difference they make

### 4. Profile Performance
```bash
cd profile_resampler
./profile.sh ../testdata/844152__kevp888__020a_100111_0243_exp02_bells_drone.wav
```

**Learn:**
- Where time is spent
- How to optimize
- Profiling techniques

### 5. Study Low-Level API
- Look at `profile_resampler/` internals
- See streaming implementation
- Understand component benchmarks

### 6. Build Custom Solutions
- Create your own processing pipeline
- Add custom effects
- Optimize for your use case

## Adding New Examples

To add a new example:

1. **Create directory:**
   ```bash
   mkdir examples/my_example
   cd examples/my_example
   ```

2. **Initialize module:**
   ```bash
   go mod init github.com/ik5/audpbx/examples/my_example
   go get github.com/ik5/audpbx
   ```

3. **Write your example:**
   ```go
   package main
   
   import "github.com/ik5/audpbx"
   
   func main() {
       // Your example code
   }
   ```

4. **Use test data:**
   ```go
   testFile := "../testdata/Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).ogg"
   ```

5. **Document it:**
   - Add README.md explaining what it demonstrates
   - Include usage examples
   - Add to this file's directory structure section

## Best Practices

### 1. Use Test Data
- Always reference `../testdata/` files
- Ensures reproducible results
- Proper licensing (CC-BY 4.0)

### 2. Error Handling
```go
if err != nil {
    log.Fatalf("failed to process: %v", err)
}
```

### 3. Resource Cleanup
```go
defer file.Close()
defer src.Close()
```

### 4. Buffer Sizing
```go
// Good default for most cases
const bufferSize = 4096

// Larger for better throughput
const bufferSize = 16384

// Smaller for lower latency
const bufferSize = 1024
```

### 5. Choose the Right API
- **High-level** for simple tasks
- **Low-level** for streaming/custom processing
- **Profile** to identify bottlenecks

## Troubleshooting

### Example Won't Build

```bash
# Ensure dependencies are downloaded
go mod tidy
go mod download
```

### File Not Found

```bash
# Use proper path to testdata
go run main.go ../testdata/filename.ogg output.wav

# From examples/ directory:
go run resampler/main.go testdata/filename.ogg output.wav
```

### Performance Issues

```bash
cd profile_resampler

# Profile to find bottlenecks
./profile.sh large_file.wav

# View results
go tool pprof -http=:8080 profiles/cpu_latest.prof
```

### Memory Issues with Large Files

Use streaming (low-level API):
```go
// Don't load entire file
// Process in chunks
buf := make([]float32, 4096)
for {
    n, _ := src.ReadSamples(buf)
    // Process buf[:n]
}
```

## Related Documentation

- **[Main README](../README.markdown)** - Library overview and API reference
- **[Test Data Guide](TESTDATA_ORGANIZATION.md)** - Test file management
- **[Test Data README](testdata/README.md)** - Available test files
- **[Profiling Guide](profile_resampler/README_PROFILING.md)** - Performance analysis
- **[pkg.go.dev](https://pkg.go.dev/github.com/ik5/audpbx)** - API documentation

## Contributing Examples

We welcome new examples! Good examples demonstrate:

1. **Real-world use cases**
2. **Best practices**
3. **Performance considerations**
4. **Error handling**
5. **Both high-level and low-level APIs**

Please ensure your example:
- ✅ Builds and runs successfully
- ✅ Uses test data from `testdata/`
- ✅ Includes documentation
- ✅ Follows Go best practices
- ✅ Has proper error handling

## Summary

The `examples/` directory is more than just tutorials—it's a comprehensive testing and performance analysis suite that demonstrates:

- ✅ **Integration testing** with real audio files
- ✅ **Performance profiling** and optimization
- ✅ **High-level API** simplicity (ResampleToMono16)
- ✅ **Low-level API** flexibility (streaming pipelines)
- ✅ **Format support** (WAV, MP3, OGG, AIFF)
- ✅ **Best practices** for audio processing in Go
- ✅ **Profiling techniques** for identifying bottlenecks
- ✅ **Real-world scenarios** (telephony, VoIP, streaming)

Whether you're learning the library, testing functionality, or optimizing performance, the examples provide the tools and guidance you need.
