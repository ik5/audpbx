# Audio Resampler Performance Analysis Tools

Tools for measuring and profiling `audpbx.ResampleToMono16`.

## Status

The large performance problem these tools were built to chase has been found and
fixed. It was **not** the interpolation math — two thirds of CPU time was
`read(2)`, because the resampler pulled one frame at a time from the decoder and
the decoder issued one syscall per 4 bytes.

103 second 44.1 kHz stereo source to 8 kHz mono, Intel i7-6820HQ:

| Format | Before  | After     | ffmpeg  | Speedup |
|--------|---------|-----------|---------|---------|
| WAV    | 6985 ms | **59 ms** | 166 ms  | 118x    |
| AIFF   | 7235 ms | **58 ms** | 151 ms  | 124x    |
| Ogg    | 705 ms  | 467 ms    | 213 ms  | 1.5x    |
| MP3    | 2191 ms | 2043 ms   | 198 ms  | 1.07x   |

WAV and AIFF now beat ffmpeg. MP3 and Ogg are bound inside the third-party
codecs rather than in this library.

The full write-up, including the profile output and the root cause, is in
**[PROFILING_GUIDE.md](PROFILING_GUIDE.md)**.

## Files in this directory

| File | Purpose |
|------|---------|
| `main.go` | Profiling tool — per-stage timing breakdown for one file |
| `profile.sh` | Automated script that runs the profiling steps |
| `Makefile` | Convenience targets for the above |
| `benchmark_test.go` | Component benchmarks (in-memory source — see caveat below) |
| `instrumented_resample.go` | `ResampleToMono16` with internal timing points |
| `PROFILING_GUIDE.md` | Case study, methodology, and what is still slow |
| `QUICK_DEBUG.md` | Command reference |

## Important caveat about `benchmark_test.go`

Those benchmarks use an in-memory `mockSource`. It performs no syscalls and
allocates nothing per call, so it **cannot** see the costs that dominate real
runs. They reported healthy numbers throughout the period when the real pipeline
was 40x slower than ffmpeg.

Use them to compare two implementations of an inner loop. To measure the
pipeline, use the real-file benchmarks:

```bash
go test ./internal/perfbench/ -bench . -benchtime 5x -benchmem
```

## Quick start

### Real-file pipeline benchmark

```bash
go test ./internal/perfbench/ -bench . -benchtime 5x -benchmem
```

### With profiles

```bash
go test ./internal/perfbench/ -run XXX -bench . -benchtime 1x \
    -cpuprofile cpu.prof -memprofile mem.prof

go tool pprof -top -nodecount=25 cpu.prof
go tool pprof -sample_index=alloc_objects -top mem.prof
go tool pprof -http=:8080 cpu.prof
```

### Single file, stage breakdown

```bash
cd examples/profile_resampler

./profile.sh ../testdata/Daniel_Bautista_-_Capricerie_No._5_\(Bach\,_Paganini\).wav

# or manually
go run main.go input.wav output.wav
go run main.go input.wav output.wav --cpu-profile=cpu.prof
```

### Component benchmarks

```bash
cd examples/profile_resampler
go test -bench=. -benchmem
```

## What you will see

### Stage breakdown

`main.go` reports where wall-clock time goes. For the 17.4 MB WAV test file:

```
Input file: Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini).wav (17.36 MB)
Source: 44100 Hz, 2 channels

=== Performance Breakdown ===
File open:          0.029 ms (  0.0%)
Decode setup:       0.061 ms (  0.1%)
Resample/Mix:      61.165 ms ( 97.0%) ← MAIN PROCESSING
Write output:       1.465 ms (  2.3%)
---
TOTAL:             63.032 ms

=== Memory Statistics ===
Alloc delta:     3.41 MB
Total alloc:     3.41 MB
Num GC:          0
Output samples:  825636 (1.57 MB)

=== Throughput ===
Speed:           275.45 MB/s
Processing rate: 1637.33x realtime
```

`Num GC: 0` is the point: the whole conversion no longer triggers a single
garbage collection. Before the fix this stage allocated 943 MB across four
files.

### CPU profile

For WAV and AIFF the run is short enough to be dominated by startup. For MP3 and
Ogg the codec's own DSP is on top, which is expected:

```
      flat  flat%   sum%        cum   cum%
     770ms 25.67% 25.67%      780ms 26.00%  go-mp3/internal/frame.(*Frame).subbandSynthesis
     260ms  8.67% 34.33%      320ms 10.67%  go-mp3/internal/imdct.Win
     170ms  5.67% 40.00%      170ms  5.67%  math.archLog
     140ms  4.67% 44.67%      140ms  4.67%  math.archExp
```

`Syscall6` should be near-invisible. If it is high, reads have become too small
— that is the original bug returning.

### Memory

The WAV pipeline allocates about **71 times total** for a 103 second file, and
`Resampler.ReadSamples` allocates nothing in steady state.

```
BenchmarkPipelineWAV-8    5    59360662 ns/op    3574768 B/op    71 allocs/op
BenchmarkPipelineMP3-8    5  2042892190 ns/op  226424755 B/op  552284 allocs/op
```

MP3's 552K allocations are inside `go-mp3` (`imdct.Win` returns a fresh slice per
call), not in this library.

## Interpreting results

### Where to look

1. Is `Syscall6` high? Reads are too small.
2. Do allocation counts scale with *frames* or with *blocks*? They should scale
   with blocks.
3. Is `go-audio/wav.PCMBuffer` present? The WAV direct-decode path was bypassed.
4. Otherwise, is the remaining time inside a third-party codec? Then it is not
   this library's to fix.

### What is not worth chasing

- **Interpolation cost.** `CubicInterpolate` did not appear in the top twenty
  even when the pipeline was 40x too slow.
- **Output buffer size.** The resampler reads its source in fixed blocks, so a
  64 sample caller buffer measures the same as a 4096 sample one.

### Check the tests before reaching for a profile

If throughput regressed, the cause is probably already named by a failing test.
These assert call patterns and allocation counts rather than elapsed time, so
they are deterministic:

```bash
go test -run 'ReadsSourceInBlocks|AllocsDoNotScale|ZeroAllocsSteadyState' ./...
```

Note that the block-read check is not redundant with the allocation checks.
Reverting the resampler to frame-at-a-time reads costs 41x throughput while
leaving output and allocation counts unchanged, so only the call-pattern test
notices. See
[PROFILING_GUIDE.md](PROFILING_GUIDE.md#guarding-against-regressions).

## Requirements

- Go 1.25+
- An input audio file (WAV, MP3, Ogg, AIFF)

Optional:

- [graphviz](https://graphviz.org/) for `pprof -pdf` and `web`
- [delve](https://github.com/go-delve/delve) for stepping through
- `ffmpeg` for reference comparisons

## See also

- [PROFILING_GUIDE.md](PROFILING_GUIDE.md) — the case study, and the quality caveat
- [QUICK_DEBUG.md](QUICK_DEBUG.md) — command reference
- [../resampler](../resampler) — the basic conversion example
- [../../internal/perfbench](../../internal/perfbench) — real-file benchmarks
