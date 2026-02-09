# Profiling and Debugging Guide for ResampleToMono16

This guide helps you identify bottlenecks in the audio resampling pipeline.

## Quick Start

### 1. Basic Timing Breakdown

Run the profiling tool to see where time is spent:

```bash
cd examples/profile_resampler
go run main.go your_file.wav output.wav
```

This will show you:
- File I/O time
- Decode setup time
- **Resample/Mix processing time** ← Main focus
- Output write time
- Memory usage and GC statistics

### 2. CPU Profiling (Detailed Function-Level Analysis)

Generate a CPU profile to see which functions consume the most CPU:

```bash
go run main.go your_file.wav output.wav --cpu-profile=cpu.prof
```

Analyze the profile:

```bash
# Interactive mode
go tool pprof -http=:8080 cpu.prof

# Or command-line mode
go tool pprof cpu.prof
# Then type: top20, list ResampleToMono16, list ReadSamples, etc.
```

### 3. Memory Profiling

Check memory allocations:

```bash
go run main.go your_file.wav output.wav --mem-profile=mem.prof
go tool pprof -http=:8080 mem.prof
```

Look for:
- Large allocations in the resampling loop
- Unexpected buffer growth
- GC pressure

## Expected Bottlenecks

Based on the code structure, the likely bottlenecks are:

### 1. **Resampler: Cubic Interpolation** (High CPU)

**Location:** `audio/resampler.go:ReadSamples()`

**What it does:**
- For each output sample, calculates position in source stream
- Performs cubic interpolation using 4 surrounding samples
- Calls `utils.CubicInterpolate()` per channel per sample

**Why it's slow:**
- Catmull-Rom spline requires 9 multiplications + 6 additions per sample
- Called for every output sample × every channel
- For 1.6MB WAV: ~160K samples × 2 channels = 320K interpolations

**Debug it:**
```bash
go tool pprof cpu.prof
(pprof) list CubicInterpolate
(pprof) list Resampler.ReadSamples
```

**Optimization ideas:**
- Use SIMD instructions for batch interpolation
- Use faster interpolation (linear instead of cubic)
- Pre-compute interpolation coefficients

### 2. **WAV Decoder: Int→Float32 Conversion** (Medium CPU)

**Location:** `formats/wav/decoder.go:source.ReadSamples()`

**What it does:**
- Reads PCM samples as int from go-audio library
- Converts each sample: `float32(intSample) / 32768.0`
- Division per sample

**Why it's slow:**
- Division is ~10-20x slower than multiplication
- Not vectorized

**Debug it:**
```bash
(pprof) list wav.source.ReadSamples
```

**Optimization ideas:**
- Replace division with multiplication: `float32(intSample) * (1.0/32768.0)`
- Batch convert with SIMD
- Stream directly as float32 if possible

### 3. **MonoMixer: Channel Averaging** (Low-Medium CPU)

**Location:** `audio/mono_mixer.go:ReadSamples()`

**What it does:**
- For stereo: `output = (left + right) * 0.5`

**Why it might be slow:**
- Requires reading 2x data from resampler
- Additional memory copy

**Debug it:**
```bash
(pprof) list MonoMixer.ReadSamples
```

### 4. **ResampleToMono16: Float32→Int16 Conversion** (Low CPU)

**Location:** `resample.go:ResampleToMono16()`

**What it does:**
- Clamps samples to [-1, 1]
- Multiplies by 32768 and converts to int16

**Why it's usually fast:**
- Simple arithmetic
- Good cache locality

## Using Zed Editor for Debugging

### Setup Go Debugging in Zed

1. Install delve debugger:
```bash
go install github.com/go-delve/delve/cmd/dlv@latest
```

2. In Zed, you can:
   - Set breakpoints by clicking line numbers
   - Use `dlv debug` in terminal
   - Run with `dlv exec`

### Manual Instrumentation

Add timing points in the code:

```go
import "time"

// In resampler.go ReadSamples():
start := time.Now()
// ... interpolation code ...
fmt.Printf("Interpolation took: %v\n", time.Since(start))
```

### Tracing Specific Functions

Add logging to understand the flow:

```go
// In resampler.go
func (r *Resampler) ReadSamples(dst []float32) (int, error) {
    fmt.Printf("[Resampler] Reading %d samples, srcPos=%d, ratio=%.2f\n", 
               len(dst), r.srcPosition, r.ratio)
    // ... rest of code
}
```

## Benchmark Specific Components

Create targeted benchmarks:

```bash
# Benchmark just the resampler
go test -bench=BenchmarkResampler -benchmem -cpuprofile=resample_cpu.prof

# Benchmark cubic interpolation
go test -bench=BenchmarkCubicInterpolate -benchmem ./utils/
```

## Common Performance Issues

### Issue 1: Small Buffer Size
**Symptom:** Many ReadSamples calls
**Fix:** Increase buffer size from 4096 to 8192 or 16384

### Issue 2: Memory Allocations in Hot Loop
**Symptom:** High allocation count in pprof
**Fix:** Pre-allocate buffers, avoid growing slices

### Issue 3: GC Pressure
**Symptom:** High "NumGC" in memory stats
**Fix:** Reduce allocations, increase initial capacity estimates

### Issue 4: Decode Overhead
**Symptom:** High time in wav.Decoder.PCMBuffer
**Fix:** Buffer more data from decoder, reduce calls

## Testing with Different File Sizes

```bash
# Small file (should be fast)
go run profile_main.go small_100kb.wav out.wav

# Medium file (your 1.6MB case)
go run profile_main.go Daniel_Bautista.wav out.wav

# Large file (your 96MB case)
go run profile_main.go large_96mb.wav out.wav --cpu-profile=large.prof
```

## Expected Performance

**Rough estimates for modern CPU (single-threaded):**
- Decode: ~200-400 MB/s
- Resample with cubic: ~50-100 MB/s
- Simple operations: ~500+ MB/s

**Your results:**
- 1.6 MB in 3s = **0.53 MB/s** ← Too slow! 
- 96 MB in 40s = **2.4 MB/s** ← Too slow!

This suggests the bottleneck is in resampling (cubic interpolation) or possibly the decoder.

## Next Steps

1. **Run the profiler** to get baseline numbers
2. **Generate CPU profile** to see the hottest functions
3. **Focus on the top 3 functions** in the profile
4. **Try different buffer sizes** (8192, 16384)
5. **Consider optimization strategies** based on findings

## Advanced: pprof Commands

```bash
# Show top CPU consumers
go tool pprof -top cpu.prof

# Show call graph
go tool pprof -pdf cpu.prof > graph.pdf

# Focus on specific function
go tool pprof -focus=Resampler cpu.prof

# Compare two profiles
go tool pprof -base=old.prof new.prof

# Show assembly
go tool pprof cpu.prof
(pprof) disasm CubicInterpolate
```

## Contact

If you find specific bottlenecks, consider:
- Switching to linear interpolation for speed vs. quality trade-off
- Using assembly/SIMD for hot paths
- Parallelizing if processing multiple files
- Using different decoder library
