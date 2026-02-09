# Quick Debug Reference

## 🚀 Run These Commands First

### 1. Basic Performance Check
```bash
cd examples/profile_resampler
go run main.go Daniel_Bautista_-_Capricerie_No._5_\(Bach\,_Paganini\).wav output.wav
```

Look for the "MAIN PROCESSING" line - this is your bottleneck time.

### 2. CPU Profile (Most Important!)
```bash
go run main.go Daniel_Bautista_-_Capricerie_No._5_\(Bach\,_Paganini\).wav output.wav --cpu-profile=cpu.prof

# View in browser (best option)
go tool pprof -http=:8080 cpu.prof

# Or command line
go tool pprof cpu.prof
> top10
> list ResampleToMono16
> list CubicInterpolate
> web  # shows visual graph (requires graphviz)
```

### 3. Benchmark Individual Components
```bash
# Run all benchmarks
go test -bench=. -benchmem -cpuprofile=bench_cpu.prof

# Just cubic interpolation
go test -bench=BenchmarkCubicInterpolate -benchmem

# Just resampler
go test -bench=BenchmarkResampler -benchmem

# Test different buffer sizes
go test -bench=BenchmarkBufferSize -benchmem
```

### 4. Memory Profile
```bash
go run main.go your_file.wav output.wav --mem-profile=mem.prof
go tool pprof -http=:8080 mem.prof
```

## 🔍 What to Look For

### In CPU Profile (pprof)

**Expected hotspots:**
1. `CubicInterpolate` - Should be #1 (interpolation math)
2. `Resampler.ReadSamples` - Frame management
3. `wav.source.ReadSamples` - Int→Float conversion
4. `MonoMixer.ReadSamples` - Channel mixing

**Red flags:**
- GC functions in top 10 (means too many allocations)
- syscall functions (means I/O blocking)
- Unexpected functions in hot path

### In Benchmark Results

```
BenchmarkCubicInterpolate-8    50000000    25.3 ns/op
```
- Lower ns/op = better
- Compare buffer sizes: larger should have fewer ns/op per sample

### In Profile Output

```
=== Performance Breakdown ===
Resample/Mix:    2847.234 ms (94.8%) ← MAIN PROCESSING
```
- This should be 90%+ of total time
- If not, check file I/O or decode issues

## 🛠️ Common Fixes

### If CubicInterpolate is the bottleneck:
1. **Try linear interpolation** (trade quality for speed)
2. Consider SIMD optimization
3. Profile with different sample rates

### If ReadSamples calls are slow:
1. **Increase buffer size**: Try 8192 or 16384
2. Check memory allocations
3. Look for slice growth

### If decoder is slow:
1. Check if file is being read multiple times
2. Look at buffer sizes in decoder
3. Consider different WAV library

### If GC is frequent:
1. Pre-allocate larger initial slices
2. Reduce buffer copies
3. Reuse buffers

## 📊 Quick Comparison Test

Test with different settings:

```bash
# Baseline
time go run main.go input.wav output1.wav

# With profiling overhead (should be similar)
time go run profile_main.go input.wav output2.wav

# Try modifying buffer size in code (change 4096 to 8192)
# Edit main.go line with ResampleToMono16(..., 8192)
time go run main.go input.wav output3.wav
```

## 🎯 Expected Numbers for Your Files

**1.6 MB file:**
- Current: ~3 seconds
- Target: <0.5 seconds (6x faster)
- Best case: ~0.1 seconds (30x faster)

**96 MB file:**
- Current: ~40 seconds  
- Target: <10 seconds (4x faster)
- Best case: ~2 seconds (20x faster)

## 📝 Zed Editor Integration

### Run from Zed Terminal
```bash
# Open terminal in Zed (Ctrl+`)
cd examples/profile_resampler
go run main.go ../../testdata/your_file.wav output.wav
```

### Debug with Delve in Zed
```bash
# Build first
go build -o resampler_debug main.go

# Debug
dlv exec ./resampler_debug -- your_file.wav output.wav

# Set breakpoints
(dlv) break resample.go:45
(dlv) continue
(dlv) print n
(dlv) next
```

### View CPU Profile in Browser
```bash
go run main.go input.wav output.wav --cpu-profile=cpu.prof
go tool pprof -http=localhost:8080 cpu.prof
# Open http://localhost:8080 in browser
# Click "View" → "Flame Graph" for best visualization
```

## 🔬 Advanced: Trace Execution

For detailed execution trace:
```bash
go test -trace=trace.out -bench=BenchmarkFullPipeline
go tool trace trace.out
# Opens interactive trace viewer in browser
```

## 📞 Need Help?

1. Share your `top10` output from pprof
2. Share the "Performance Breakdown" from profile_main.go
3. Mention your CPU model for context
4. Note: 0.53 MB/s is ~100x slower than expected

Good luck! The profiler will tell you exactly where to focus.
