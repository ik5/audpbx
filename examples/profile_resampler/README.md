# Audio Resampler Profiling Tools

This directory contains tools to profile and debug the performance of `audpbx.ResampleToMono16`.

## Quick Start

```bash
# Automated profiling (easiest)
./profile.sh path/to/your_file.wav

# Or with make
make profile INPUT=path/to/your_file.wav

# Or manually
go run main.go path/to/your_file.wav output.wav --cpu-profile=cpu.prof
go tool pprof -http=:8080 cpu.prof
```

## What's Here

- **`main.go`** - Profiling tool with detailed timing breakdown
- **`profile.sh`** - Automated script that runs all profiling steps
- **`Makefile`** - Convenient make commands
- **`benchmark_test.go`** - Benchmarks for individual components
- **`instrumented_resample.go`** - Detailed internal timing version
- **`README_PROFILING.md`** - Complete guide (start here!)
- **`QUICK_DEBUG.md`** - Quick reference for commands
- **`PROFILING_GUIDE.md`** - Detailed explanations of bottlenecks

## Documentation

**Start with:** [README_PROFILING.md](README_PROFILING.md)

This comprehensive guide explains:
- How to run the profiling tools
- What the results mean
- Expected bottlenecks and how to fix them
- Integration with Zed editor
- Optimization strategies

## Example Output

```
=== Performance Breakdown ===
File open:          0.234 ms (  0.1%)
Decode setup:      12.450 ms (  0.4%)
Resample/Mix:    2847.234 ms ( 94.8%) ← MAIN PROCESSING
Write output:     145.670 ms (  4.7%)
---
TOTAL:           3005.588 ms
```

The tools will help you identify exactly where time is being spent in the audio processing pipeline.

## Running Tests

```bash
# Run all benchmarks
make bench

# Test different buffer sizes
make test-buffers

# View results
make view-cpu
```

## Requirements

- Go 1.21+
- Input audio file (WAV, MP3, OGG, AIFF)

Optional:
- [graphviz](https://graphviz.org/) for generating call graph PDFs

## Usage Examples

```bash
# Basic timing
go run main.go input.wav output.wav

# With CPU profiling
go run main.go input.wav output.wav --cpu-profile=cpu.prof

# With memory profiling
go run main.go input.wav output.wav --mem-profile=mem.prof

# Both
go run main.go input.wav output.wav \
  --cpu-profile=cpu.prof \
  --mem-profile=mem.prof
```

## See Also

- [../resampler](../resampler) - The basic resampler example
- [../../resample.go](../../resample.go) - Source code for ResampleToMono16
