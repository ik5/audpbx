# Audio Resampler Performance Analysis Tools

This directory contains tools to help you debug and optimize the `audpbx.ResampleToMono16` function.

## 🎯 Your Performance Issue

**Current Performance:**
- 1.6 MB file: ~3 seconds (0.53 MB/s)
- 96 MB file: ~40 seconds (2.4 MB/s)

**Expected Performance:**
- Should be 50-200 MB/s (20-100x faster)

## 📁 Files in This Directory

| File | Purpose |
|------|---------|
| `main.go` | **Main profiling tool** - shows timing breakdown |
| `profile.sh` | **Automated script** - runs all profiling steps |
| `Makefile` | Quick make commands for all tools |
| `benchmark_test.go` | Benchmark individual components |
| `instrumented_resample.go` | Detailed timing version of ResampleToMono16 |
| `QUICK_DEBUG.md` | Quick reference commands |
| `PROFILING_GUIDE.md` | Detailed guide with explanations |

## 🚀 Quick Start (Pick One)

### Option 1: Automated (Easiest)
```bash
./profile.sh Daniel_Bautista_-_Capricerie_No._5_\(Bach\,_Paganini\).wav
```

This runs everything and offers to open the profile in your browser.

### Option 2: Manual (More Control)
```bash
# Basic timing
go run main.go your_file.wav output.wav

# With CPU profiling
go run main.go your_file.wav output.wav --cpu-profile=cpu.prof

# View the profile
go tool pprof -http=:8080 cpu.prof
```

### Option 3: Benchmarks
```bash
# Run all benchmarks
go test -bench=. -benchmem

# With CPU profile
go test -bench=. -benchmem -cpuprofile=bench.prof
```

## 🔍 What You'll Learn

### 1. Timing Breakdown
The profiler will show you:
```
=== Performance Breakdown ===
File open:          0.234 ms (  0.1%)
Decode setup:       12.45 ms (  0.4%)
Resample/Mix:     2847.23 ms ( 94.8%) ← MAIN PROCESSING
Write output:      145.67 ms (  4.7%)
---
TOTAL:            3005.59 ms
```

### 2. CPU Profile (Most Important!)
Shows which functions use the most CPU time:
```
Showing nodes accounting for 2650ms, 93.24% of 2842ms total
      flat  flat%   sum%        cum   cum%
    1850ms 65.09% 65.09%     1850ms 65.09%  CubicInterpolate
     450ms 15.83% 80.92%      800ms 28.14%  Resampler.ReadSamples
     200ms  7.04% 87.96%      350ms 12.31%  wav.source.ReadSamples
     150ms  5.28% 93.24%      200ms  7.04%  MonoMixer.ReadSamples
```

### 3. Memory Usage
```
=== Memory Statistics ===
Alloc delta:     12.45 MB
Total alloc:     18.23 MB
Num GC:          3
Output samples:  500000 (0.95 MB)
```

## 🎓 Understanding the Results

### Where to Look First

1. **"Resample/Mix" line** in timing breakdown → Should be 90%+ of time
2. **Top function in pprof** → Usually `CubicInterpolate` (the math)
3. **Number of GC runs** → Should be low (< 5 for small files)

### Likely Bottlenecks (in order)

#### 1. Cubic Interpolation (Expected #1 bottleneck)
- **Function:** `utils.CubicInterpolate`
- **Why slow:** 9 multiplications per sample
- **Called:** ~160K times for 1.6MB file
- **Fix ideas:**
  - Use linear interpolation (faster, lower quality)
  - SIMD vectorization
  - Assembly optimization

#### 2. WAV Decoder (Possible #2 bottleneck)
- **Function:** `wav.source.ReadSamples`  
- **Why slow:** Integer division for int→float32 conversion
- **Called:** Many times for small chunks
- **Fix ideas:**
  - Use multiplication instead of division
  - Larger buffer sizes
  - Different decoder library

#### 3. Frame Management (Overhead)
- **Function:** `Resampler.ReadSamples` (frame shifting)
- **Why slow:** Copying frames in sliding window
- **Fix ideas:**
  - Ring buffer instead of copying
  - Batch processing

## 🛠️ Optimization Strategies

### Try These in Order:

#### 1. Increase Buffer Size (Easy Win)
```go
// In your code, change from:
pcm16, rate, err := audpbx.ResampleToMono16(src, 8000, 4096)

// To:
pcm16, rate, err := audpbx.ResampleToMono16(src, 8000, 16384)
```

Run the benchmark to test:
```bash
go test -bench=BenchmarkBufferSize -benchmem
```

#### 2. Profile to Find Real Bottleneck
```bash
./profile.sh your_file.wav
# Look at the top function in pprof
```

#### 3. If CubicInterpolate is >60% of time:
Consider implementing linear interpolation as an option:
```go
// Linear is ~4x faster but lower quality
// y = y1 + (y2 - y1) * x
result := y1 + (y2-y1)*alpha
```

#### 4. If WAV decoder is slow:
Check if you can read larger chunks or use a different library.

## 📊 Benchmarking Components

Test individual parts to isolate the problem:

```bash
# Just the interpolation math
go test -bench=BenchmarkCubicInterpolate -benchmem

# Just the resampler
go test -bench=BenchmarkResampler_Downsample -benchmem

# Just the mixer
go test -bench=BenchmarkMonoMixer -benchmem

# Full pipeline
go test -bench=BenchmarkFullPipeline -benchmem
```

## 🎯 Zed Editor Integration

### Run from Zed
1. Open terminal in Zed: `Ctrl+` (backtick)
2. Navigate: `cd examples/profile_resampler`
3. Run: `./profile.sh ../../path/to/file.wav`

### View Results
The script opens the profile in your browser at `http://localhost:8080`

**Best views:**
- **Flame Graph** - Visual representation of time spent
- **Top** - List of hottest functions  
- **Source** - See source code with timing annotations

### Debugging in Zed
If you want to set breakpoints:
```bash
go build -o debug_resampler main.go
dlv exec ./debug_resampler -- input.wav output.wav

# In dlv:
(dlv) break resample.go:67
(dlv) continue
(dlv) print pcm16
(dlv) next
```

## 📈 Expected Improvements

| Optimization | Expected Speedup | Effort |
|--------------|------------------|--------|
| Larger buffer (8192→16384) | 10-20% | 1 minute |
| Fix int→float division | 5-10% | Easy |
| Linear interpolation | 3-5x | Medium |
| SIMD cubic interpolation | 4-8x | Hard |
| Parallel processing | 2-4x | Medium |

## 🐛 Common Issues

### Issue: "pprof command not found"
```bash
# pprof is included with Go, but try:
go install golang.org/x/tools/cmd/pprof@latest
```

### Issue: "Can't open browser"
```bash
# Manual mode instead:
go tool pprof cpu.prof
(pprof) top
(pprof) list CubicInterpolate
```

### Issue: Profile shows mostly GC time
- You have too many allocations
- Check memory profile: `go tool pprof mem.prof`
- Pre-allocate larger buffers

### Issue: Decoder appears slow
- File might be read multiple times
- Try with a file on tmpfs/RAM disk
- Check if seeking is involved

## 📚 Additional Resources

- [Go pprof documentation](https://go.dev/blog/pprof)
- [Profiling Go Programs](https://go.dev/blog/profiling-go-programs)
- [High Performance Go Workshop](https://dave.cheney.net/high-performance-go-workshop/gopherchina-2019.html)

## 💡 Tips

1. **Always profile before optimizing** - Don't guess!
2. **Focus on the top 1-2 functions** - They're usually 80%+ of time
3. **Test on real files** - Synthetic benchmarks can mislead
4. **Measure after each change** - Verify improvements
5. **Quality vs Speed** - Cubic is high quality, linear is fast

## 🤝 Getting Help

If you're stuck after profiling:

1. Run `./profile.sh your_file.wav`
2. Save the "Performance Breakdown" output
3. Run `go tool pprof -top cpu.prof` and save output
4. Share both outputs with your team

Good luck finding those bottlenecks! 🚀
