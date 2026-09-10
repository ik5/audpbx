# Audio Resampler Profiling Tools

Tools for profiling and debugging the performance of `audpbx.ResampleToMono16`.

## Start here

**[PROFILING_GUIDE.md](PROFILING_GUIDE.md)** — the case study of the 40x
slowdown that used to affect this pipeline: what was actually slow (it was not
the interpolation math), how the profile settled it, what the numbers are now,
and what is still slow and why.

## Current performance

103 second 44.1 kHz stereo source to 8 kHz mono, Intel i7-6820HQ:

| Format | audpbx    | ffmpeg | allocations |
|--------|-----------|--------|-------------|
| WAV    | **59 ms** | 166 ms | 71          |
| AIFF   | **58 ms** | 151 ms | 70          |
| Ogg    | 467 ms    | 213 ms | 58 K        |
| MP3    | 2043 ms   | 198 ms | 552 K       |

WAV and AIFF are faster than ffmpeg. MP3 and Ogg are bound inside the
third-party codecs (`go-mp3`, `jfreymuth/vorbis`) rather than in this library.

## Quick start

Measure the pipeline against real files — this is the measurement that matters:

```bash
go test ./internal/perfbench/ -bench . -benchtime 5x -benchmem
```

Profile one file with a stage breakdown:

```bash
./profile.sh ../testdata/Daniel_Bautista_-_Capricerie_No._5_\(Bach\,_Paganini\).wav

# or
make profile INPUT=../testdata/some_file.wav

# or manually
go run main.go input.wav output.wav --cpu-profile=cpu.prof
go tool pprof -http=:8080 cpu.prof
```

## What's here

- **`main.go`** — profiling tool with a per-stage timing breakdown
- **`profile.sh`** — automated script that runs all profiling steps
- **`Makefile`** — convenience targets
- **`benchmark_test.go`** — component benchmarks (in-memory source; see caveat)
- **`instrumented_resample.go`** — `ResampleToMono16` with internal timing points
- **`PROFILING_GUIDE.md`** — case study, methodology, remaining bottlenecks
- **`QUICK_DEBUG.md`** — command reference
- **`README_PROFILING.md`** — walkthrough of the tools and their output

## Caveat: component benchmarks cannot find pipeline bottlenecks

`benchmark_test.go` uses an in-memory `mockSource`: no syscalls, no per-call
allocations. It reported healthy numbers during the entire period when the real
pipeline ran 40x slower than ffmpeg, because the cost was in decoder call
overhead and I/O that a mock does not have.

Use it to compare inner-loop implementations. Use
[`internal/perfbench`](../../internal/perfbench) to measure the pipeline.

## Example output

```
=== Performance Breakdown ===
File open:          0.029 ms (  0.0%)
Decode setup:       0.061 ms (  0.1%)
Resample/Mix:      61.165 ms ( 97.0%) ← MAIN PROCESSING
Write output:       1.465 ms (  2.3%)
---
TOTAL:             63.032 ms

=== Memory Statistics ===
Num GC:          0
Output samples:  825636 (1.57 MB)

=== Throughput ===
Speed:           275.45 MB/s
Processing rate: 1637.33x realtime
```

## Running tests and benchmarks

```bash
# Real-file pipeline benchmarks (from the repository root)
go test ./internal/perfbench/ -bench . -benchtime 5x -benchmem

# Component benchmarks (this directory)
make bench

# Buffer size comparison — expect little variation, the resampler
# reads its source in fixed blocks regardless of the caller's buffer
make test-buffers

# View a profile
make view-cpu
```

## Requirements

- Go 1.25+
- An input audio file (WAV, MP3, Ogg, AIFF)

Optional:

- [graphviz](https://graphviz.org/) for call-graph output
- `ffmpeg` for reference comparisons

## See also

- [../resampler](../resampler) — the basic resampler example
- [../../resample.go](../../resample.go) — source for `ResampleToMono16`
- [../../audio/resampler.go](../../audio/resampler.go) — the block-reading resampler
