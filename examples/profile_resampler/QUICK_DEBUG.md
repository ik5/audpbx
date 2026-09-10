# Quick Debug Reference

Command reference for profiling the resampling pipeline. For why the pipeline
looks the way it does, read [PROFILING_GUIDE.md](PROFILING_GUIDE.md).

## Run these first

### 1. Pipeline benchmark against real files

The measurement that matters. An in-memory source cannot see decoder call
overhead or I/O, which is where this pipeline's problems have historically been.

```bash
go test ./internal/perfbench/ -bench . -benchtime 5x -benchmem
```

### 2. CPU profile

```bash
go test ./internal/perfbench/ -run XXX -bench . -benchtime 1x -cpuprofile cpu.prof

go tool pprof -top -nodecount=25 cpu.prof
go tool pprof -http=:8080 cpu.prof          # flame graph, best view
```

Command-line exploration:

```bash
go tool pprof cpu.prof
> top20
> top -cum
> list ReadSamples
> peek PCMBuffer$
> web                                        # visual graph, needs graphviz
```

### 3. Memory profile

Use `alloc_objects`, not just `alloc_space`. A 4-byte allocation per call is
invisible by size and glaring by count.

```bash
go test ./internal/perfbench/ -run XXX -bench . -benchtime 1x -memprofile mem.prof

go tool pprof -sample_index=alloc_objects -top mem.prof
go tool pprof -sample_index=alloc_space   -top mem.prof
```

### 4. Single file, stage breakdown

```bash
cd examples/profile_resampler
go run main.go ../testdata/Daniel_Bautista_-_Capricerie_No._5_\(Bach\,_Paganini\).wav out.wav
```

### 5. Component benchmarks

For comparing two versions of an inner loop. These use an in-memory source and
will not reveal pipeline bottlenecks.

```bash
cd examples/profile_resampler
go test -bench=. -benchmem
go test -bench=BenchmarkCubicInterpolate -benchmem
go test -bench=BenchmarkBufferSize -benchmem
```

## Expected numbers

103 second 44.1 kHz stereo source to 8 kHz mono, Intel i7-6820HQ:

| Format | audpbx    | ffmpeg | allocations |
|--------|-----------|--------|-------------|
| WAV    | **59 ms** | 166 ms | 71          |
| AIFF   | **58 ms** | 151 ms | 70          |
| Ogg    | 467 ms    | 213 ms | 58 K        |
| MP3    | 2043 ms   | 198 ms | 552 K       |

WAV and AIFF are faster than ffmpeg. MP3 and Ogg are bound inside the
third-party codecs (`go-mp3`, `jfreymuth/vorbis`), not in this library.

If WAV is not in the tens of milliseconds for a file of this size, something has
regressed. Check the red flags below.

## What to look for

### Red flags

| Symptom | Almost certainly means | Test that should have caught it |
|---|---|---|
| `Syscall6` high in the CPU profile | Reads are too small — something is pulling frame-at-a-time | `audio.TestResampler_ReadsSourceInBlocks` |
| Allocation count scales with frame count | A per-sample or per-frame allocation crept into the hot path | `TestResampleToMono16_AllocsDoNotScaleWithLength` |
| `go-audio/wav.PCMBuffer` in the profile | The WAV direct-decode path was bypassed | `wav.TestSource_ReadSamples_ZeroAllocsSteadyState` |
| `go-audio/aiff.PCMBuffer` in the profile | The AIFF direct-decode path was bypassed | `aiff.TestSource_ReadSamples_RawZeroAllocsSteadyState` |
| `GC` functions in the top ten | Allocation pressure in the loop | the two allocation tests above |

If you are looking at a profile because throughput regressed, run the tests
first — one of them probably already names the cause:

```bash
go test -run 'ReadsSourceInBlocks|AllocsDoNotScale|ZeroAllocsSteadyState' ./...
```

### Healthy shape

- `Resampler.ReadSamples` allocates nothing in steady state
- Allocation count is proportional to *blocks*, not frames
- For MP3 and Ogg the codec's own DSP is on top; that is expected

## Common fixes

### Reads look too small

Check `resamplerBlockFrames` in `audio/resampler.go`. The resampler must pull
its source in blocks; one frame per call reintroduces the original 40x
slowdown.

### Output buffer size

Largely irrelevant now, and worth knowing so you do not chase it. The resampler
reads its source in fixed blocks regardless of the caller's buffer, so a 64
sample buffer and a 4096 sample buffer measure the same:

```
BenchmarkResampler_SmallBuffer-8    4899766 ns/op   # 64-sample buffer
BenchmarkResampler_Downsample-8     4972946 ns/op   # 4096-sample buffer
```

### Benchmark reports an implausibly small number

A `Resampler` keeps its own end-of-stream state. If you construct one outside
the timed loop and reset only the source, every iteration after the first
returns `io.EOF` immediately:

```go
// WRONG: times an early return, reported ~36 ns/op for 100K frames
resampler := NewResampler(src, 8000)
for b.Loop() {
    src.Reset()
    for { _, err := resampler.ReadSamples(buf); if err == io.EOF { break } }
}

// RIGHT: construct inside the loop
for b.Loop() {
    src.Reset()
    resampler := NewResampler(src, 8000)
    for { _, err := resampler.ReadSamples(buf); if err != nil { break } }
}
```

## Comparing against ffmpeg

```bash
time ffmpeg -v error -i input.wav -ar 8000 -ac 1 -c:a pcm_s16le -y ff.wav
time go run ./examples/resampler input.wav ours.wav 8000
```

Check anti-aliasing quality too, not just speed. A tone above the output Nyquist
should disappear:

```bash
ffmpeg -v error -f lavfi -i "sine=frequency=6000:sample_rate=44100:duration=5" \
    -ac 2 -c:a pcm_s16le -y tone6k.wav
go run ./examples/resampler tone6k.wav ours8k.wav 8000
ffmpeg -hide_banner -i ours8k.wav -af volumedetect -f null - 2>&1 | grep mean_volume
```

Currently about −27.9 dB where ffmpeg reaches −73.7 dB. See the quality note in
[PROFILING_GUIDE.md](PROFILING_GUIDE.md).

## Comparing two revisions

To confirm a change did not alter output, diff the bytes rather than trusting a
listen:

```bash
go run ./examples/resampler in.wav before.wav 8000
# apply the change
go run ./examples/resampler in.wav after.wav 8000
cmp before.wav after.wav && echo "bit-identical"
```

For profiles:

```bash
go tool pprof -base=before.prof after.prof
```

## Debugging with delve

```bash
go install github.com/go-delve/delve/cmd/dlv@latest

go build -o resampler_debug ./examples/resampler
dlv exec ./resampler_debug -- input.wav output.wav 8000

(dlv) break audio/resampler.go:pull
(dlv) continue
(dlv) print r.inFrames
(dlv) print r.inBase
```

## Execution trace

```bash
go test ./internal/perfbench/ -run XXX -bench BenchmarkPipelineWAV -benchtime 1x -trace trace.out
go tool trace trace.out
```
