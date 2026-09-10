# Profiling Guide for ResampleToMono16

This guide covers how to profile the resampling pipeline, and documents the one
large bottleneck that has already been found and fixed. Read the case study
first — it is the most useful thing in this directory, because it shows how the
obvious hypothesis was wrong and how the profile settled it.

## Case Study: the 40x slowdown

### Symptom

A 103 second 44.1 kHz stereo WAV took **6.9 seconds** to convert to 8 kHz mono.
`ffmpeg` did the same job in 0.17 s.

### The wrong hypothesis

The natural guess was cubic interpolation. Catmull-Rom needs about nine
multiplies and six adds per sample per channel, and it runs for every output
sample, so it looks like the hot spot. Earlier revisions of this guide predicted
`utils.CubicInterpolate` would be the number one entry in the profile.

It was not in the top twenty.

### What the profile actually said

```
$ go test ./internal/perfbench/ -bench . -benchtime 1x -cpuprofile cpu.prof
$ go tool pprof -top -nodecount=6 cpu.prof

      flat  flat%   sum%        cum   cum%
    11.73s 67.38% 67.38%     11.73s 67.38%  internal/runtime/syscall/linux.Syscall6
     0.79s  4.54% 71.91%      0.81s  4.65%  go-mp3/internal/frame.(*Frame).subbandSynthesis
     0.35s  2.01% 73.92%      0.38s  2.18%  go-mp3/internal/imdct.Win
     0.19s  1.09% 75.01%     16.70s 95.92%  audio.(*Resampler).readSourceFrameInto
     0.18s  1.03% 76.05%      7.05s 40.49%  go-audio/aiff.(*Decoder).PCMBuffer
     0.17s  0.98% 79.09%      6.79s 39.00%  go-audio/wav.(*Decoder).PCMBuffer
```

**Two thirds of all CPU time was `read(2)`.** Interpolation did not register.

The memory profile agreed: **27.5 million allocations** and 943 MB allocated for
four files, essentially all of it inside `PCMBuffer` and `bytes.NewReader`.

### Root cause

`Resampler` pulled exactly one frame per call from its source:

```go
// audio/resampler.go, before
n, err := r.src.ReadSamples(r.srcBuf[:r.channels])   // 2 floats for stereo
```

A 103 second stereo file is 4.55 million frames, so that is 4.55 million calls.
Each one landed in `wav.Decoder.PCMBuffer`, which per call:

- allocates an `audio.Format`
- allocates `tmpBuf` — sized `len(buf.Data) * bytesPerSample`, i.e. **4 bytes**
- allocates a `bytes.Reader` and a per-sample scratch buffer
- issues one `Read` on the underlying `*os.File`

The file was not buffered, so that last step was **one syscall per 4 bytes of
audio**. The arithmetic was never the problem; the per-call overhead was.

### The fix

Two changes, no change to the interpolation math:

1. `Resampler` now reads its source in blocks of 8192 frames into a sliding
   window (`resamplerBlockFrames` in `audio/resampler.go`), keeping one frame of
   history and two of lookahead so cubic interpolation still has its four
   neighbours across refills.
2. The WAV and AIFF decoders decode 16-bit samples straight from the PCM chunk
   reader instead of going through `PCMBuffer`, which removes its four
   allocations per call and an `[]int` intermediate four times wider than the
   samples it carries.

### Result

103 second 44.1 kHz stereo source to 8 kHz mono, Intel i7-6820HQ:

| Format | Before  | After     | ffmpeg  | Speedup |
|--------|---------|-----------|---------|---------|
| WAV    | 6985 ms | **59 ms** | 166 ms  | 118x    |
| AIFF   | 7235 ms | **58 ms** | 151 ms  | 124x    |
| Ogg    | 705 ms  | 467 ms    | 213 ms  | 1.5x    |
| MP3    | 2191 ms | 2043 ms   | 198 ms  | 1.07x   |

Allocations for the whole WAV pipeline went from about **10.7 million to 71**, and
`Resampler.ReadSamples` is now zero-allocation in steady state.

WAV and AIFF are faster than ffmpeg. Syscall time dropped from 11.73 s to 50 ms.

### The methodology lesson

`benchmark_test.go` in this directory did **not** catch any of this, and could
not have. Its `mockSource` copies from an in-memory slice: no syscalls, no
per-call allocations. It reported healthy numbers throughout.

**Benchmark this pipeline through a real decoder reading a real file.** That is
what `internal/perfbench` is for.

## What is still slow

MP3 and Ogg are now bound inside the third-party codecs, not in this library's
code. Profiling MP3 shows the remaining time in:

- `go-mp3/internal/frame.(*Frame).subbandSynthesis`
- `go-mp3/internal/imdct.Win` — allocates a slice per call, 510K allocations and
  70 MB for one file
- `math.Pow` / `math.Exp` / `math.Log` inside `requantizeProcessLong`

These are inside `github.com/hajimehoshi/go-mp3`. Closing the remaining ~10x gap
to ffmpeg on MP3 means patching or replacing that decoder, not tuning the
resampler. The same applies to Ogg and `github.com/jfreymuth/vorbis`.

If your workload is WAV or AIFF — the usual case for PBX prompt conversion —
the pipeline is already faster than ffmpeg and there is little left to win.

## How to profile

### Pipeline, against real files

This is the measurement that matters:

```bash
go test ./internal/perfbench/ -bench . -benchtime 5x
```

With profiles:

```bash
go test ./internal/perfbench/ -run XXX -bench . -benchtime 1x \
    -cpuprofile cpu.prof -memprofile mem.prof

go tool pprof -top -nodecount=25 cpu.prof
go tool pprof -sample_index=alloc_objects -top mem.prof
go tool pprof -http=:8080 cpu.prof     # flame graph
```

### One file, with a stage breakdown

```bash
cd examples/profile_resampler
go run main.go input.wav output.wav
go run main.go input.wav output.wav --cpu-profile=cpu.prof
```

### Components, against an in-memory source

Useful for comparing two implementations of the same inner loop. Not useful for
finding pipeline bottlenecks — see the lesson above.

```bash
cd examples/profile_resampler
go test -bench=. -benchmem
```

## Reading a profile

### Expected shape now

For WAV or AIFF, total runtime is small enough that the profile is dominated by
startup. For MP3 and Ogg, expect the codec's own DSP functions on top.

You should **not** see:

- `Syscall6` high in the list — the regression this guide documents
- allocation counts scaling with frame count rather than with block count
- `PCMBuffer` in the WAV or AIFF path at all

### Useful commands

```bash
go tool pprof -top -nodecount=25 cpu.prof     # top consumers
go tool pprof -top -cum cpu.prof              # by cumulative time
go tool pprof -peek 'PCMBuffer$' cpu.prof     # callers and callees
go tool pprof -focus 'Syscall6' cpu.prof      # who is doing I/O
go tool pprof -list 'ReadSamples' cpu.prof    # line-level attribution
go tool pprof -base=old.prof new.prof         # compare two runs
```

### Allocation profiles

`alloc_objects` is usually more diagnostic than `alloc_space` for this kind of
problem: a per-call allocation of a few bytes barely shows up by size but stands
out immediately by count.

```bash
go tool pprof -sample_index=alloc_objects -top mem.prof
go tool pprof -sample_index=alloc_space   -top mem.prof
```

## Guarding against regressions

Each fix above has a test that fails if it is reverted. Every entry below was
confirmed by actually reintroducing the defect and watching the named test go
red.

| Test | Defect it catches |
|---|---|
| `audio.TestResampler_ReadsSourceInBlocks` | Frame-at-a-time source reads — the 40x slowdown itself |
| `TestResampleToMono16_AllocsDoNotScaleWithLength` | A per-frame or per-read allocation anywhere in the pipeline |
| `audio.TestResampler_ZeroAllocsSteadyState` | An allocation on the resampler's streaming path |
| `audio.TestResampler_BufferSizeIndependence` | Output changing with the caller's read size |
| `audio.TestResampler_IdentityRateIsExact` | The end-of-stream off-by-one, and dropped short sources |
| `wav.TestSource_ReadSamples_ZeroAllocsSteadyState` | A per-call allocation in the WAV decoder |
| `aiff.TestSource_ReadSamples_RawZeroAllocsSteadyState` | The AIFF decoder falling back to `PCMBuffer` |
| `aiff.TestSource_ReadSamples_RawBigEndian` | AIFF sample byte order |
| `wav`/`vorbis` `TestSource_ReadSamples_NeverOverruns`, `aiff.TestSource_ReadSamples_RawNeverOverruns` | A decoder reporting more values than the caller's buffer holds |
| `utils.TestFloat32ToInt16` | The full-scale int16 overflow |

### Why the call-pattern test is not redundant

It is tempting to assume the allocation tests cover the block-read fix. They do
not, and this was verified: setting `resamplerBlockFrames` to 1 makes the WAV
pipeline **41x slower — 59 ms to 2409 ms — with the entire suite still green.**

Two reasons:

- Block size does not change the output, so every correctness test still passes.
- The decoders reuse their staging buffers, so allocation counts stay flat
  whether the resampler asks for 2 samples or 16384.

Only the *pattern* of calls into the source reveals it, which is why
`TestResampler_ReadsSourceInBlocks` inspects the sizes and count of
`Source.ReadSamples` calls rather than timing or allocations. The lesson
generalises: **when the defect is per-call overhead, assert on the call pattern,
because neither output nor allocation counts will show it.**

### What is still not guarded

- **Throughput itself.** Nothing fails if the inner loop simply gets slower for
  a reason other than call pattern or allocation. That wants benchmark tracking
  in CI (`benchstat` against a stored baseline), not a unit test — a wall-clock
  threshold would flake on a loaded machine.
- **Anti-aliasing quality.** No test notices if the filter degrades. The 6 kHz
  tone measurement in the quality note below would make a reasonable assertion
  if that work is ever done.
- **MP3.** Its cost is entirely inside `go-mp3`; there is nothing of ours to
  guard.
- **Benchmarks that measure nothing.** A benchmark cannot fail, so the mistake
  described next stays silent by nature.

### A trap when writing benchmarks

A `Resampler` tracks its own end-of-stream state. Benchmarks must construct one
**inside** the timed loop; resetting only the source and reusing the resampler
makes every iteration after the first return `io.EOF` immediately, timing an
early return instead of the work. Four benchmarks in this repository did exactly
that and reported **36 ns/op for a 100,000-frame resample** — roughly 130,000x
faster than reality.

If a benchmark result looks too good, that is the first thing to check.

## A note on quality

Speed is not the only axis. The anti-aliasing filter applied when downsampling
is a one-pole low-pass with a fixed coefficient (`defaultFilterAlpha = 0.5`)
that does not track the resampling ratio, so it attenuates far less than a
proper decimation filter.

Measured: a 6 kHz tone resampled from 44.1 kHz to 8 kHz, where the 4 kHz output
Nyquist means it should be gone:

| | mean level |
|---|---|
| ffmpeg | −73.7 dB |
| audpbx | −27.9 dB |

The tone survives, aliased down to 2 kHz — about 46 dB worse than ffmpeg. Fixing
this properly means a polyphase FIR — the common telephony conversion 44.1 kHz
to 8 kHz is exactly 441/80, so it is a rational resample with a modest phase
count. That would cost some throughput, though there is ample headroom at 59 ms.

## See also

- [README.md](README.md) — what is in this directory
- [QUICK_DEBUG.md](QUICK_DEBUG.md) — command reference
- [../../internal/perfbench](../../internal/perfbench) — real-file benchmarks
- [../../resample.go](../../resample.go) — `ResampleToMono16`
- [../../audio/resampler.go](../../audio/resampler.go) — the block-reading resampler
