# SPDX-License-Identifier: EPL-2.0
# Makefile for profiling and benchmarking audio resampler

.PHONY: help profile bench cpu mem clean quick view-cpu view-mem

# Default test file - override with: make profile INPUT=yourfile.wav
INPUT ?= test.wav
OUTPUT ?= output.wav

help:
	@echo "Audio Resampler Profiling Tools"
	@echo ""
	@echo "Quick commands:"
	@echo "  make quick          - Basic timing (fastest)"
	@echo "  make profile        - Full automated profiling"
	@echo "  make cpu            - Generate CPU profile only"
	@echo "  make mem            - Generate memory profile only"
	@echo "  make bench          - Run benchmarks"
	@echo "  make view-cpu       - View latest CPU profile in browser"
	@echo "  make view-mem       - View latest memory profile"
	@echo "  make clean          - Remove generated files"
	@echo ""
	@echo "Usage:"
	@echo "  make profile INPUT=myfile.wav"
	@echo "  make cpu INPUT=myfile.wav"
	@echo ""

# Quick timing analysis (no profiling overhead)
quick:
	@echo "Running quick timing analysis..."
	@go run main.go $(INPUT) $(OUTPUT)

# Full automated profiling
profile:
	@./profile.sh $(INPUT) $(OUTPUT)

# CPU profiling only
cpu: profiles
	@echo "Generating CPU profile..."
	@go run main.go $(INPUT) $(OUTPUT) \
		--cpu-profile=profiles/cpu_latest.prof
	@echo ""
	@echo "CPU profile saved: profiles/cpu_latest.prof"
	@echo "View with: make view-cpu"

# Memory profiling only
mem: profiles
	@echo "Generating memory profile..."
	@go run main.go $(INPUT) $(OUTPUT) \
		--mem-profile=profiles/mem_latest.prof
	@echo ""
	@echo "Memory profile saved: profiles/mem_latest.prof"
	@echo "View with: make view-mem"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	@go test -bench=. -benchmem -benchtime=3s

# Benchmark with CPU profile
bench-cpu: profiles
	@echo "Running benchmarks with CPU profiling..."
	@go test -bench=. -benchmem -cpuprofile=profiles/bench_cpu.prof
	@echo ""
	@echo "Benchmark CPU profile: profiles/bench_cpu.prof"

# View CPU profile in browser
view-cpu:
	@if [ -f profiles/cpu_latest.prof ]; then \
		echo "Opening CPU profile in browser..."; \
		go tool pprof -http=:8080 profiles/cpu_latest.prof; \
	else \
		echo "No CPU profile found. Run: make cpu"; \
	fi

# View memory profile in browser
view-mem:
	@if [ -f profiles/mem_latest.prof ]; then \
		echo "Opening memory profile in browser..."; \
		go tool pprof -http=:8080 profiles/mem_latest.prof; \
	else \
		echo "No memory profile found. Run: make mem"; \
	fi

# Show CPU profile top functions
top-cpu:
	@if [ -f profiles/cpu_latest.prof ]; then \
		go tool pprof -top -cum profiles/cpu_latest.prof | head -20; \
	else \
		echo "No CPU profile found. Run: make cpu"; \
	fi

# Show memory allocations
top-mem:
	@if [ -f profiles/mem_latest.prof ]; then \
		go tool pprof -top -alloc_space profiles/mem_latest.prof | head -20; \
	else \
		echo "No memory profile found. Run: make mem"; \
	fi

# Create profiles directory
profiles:
	@mkdir -p profiles

# Clean generated files
clean:
	@echo "Cleaning generated files..."
	@rm -rf profiles/
	@rm -f output.wav cpu.prof mem.prof *.test
	@echo "Done."

# Build the profiler
build:
	@go build -o resampler_profile main.go
	@echo "Built: resampler_profile"

# Run with different buffer sizes (for testing)
test-buffers:
	@echo "Testing different buffer sizes..."
	@echo ""
	@for size in 1024 2048 4096 8192 16384; do \
		echo "=== Buffer size: $$size ==="; \
		go test -bench=BenchmarkBufferSize_$$size -benchtime=3s; \
		echo ""; \
	done

# Compare performance across file sizes
compare:
	@echo "Performance comparison..."
	@echo ""
	@echo "Small file test:"
	@time go run main.go small.wav out1.wav 2>&1 | grep "TOTAL:"
	@echo ""
	@echo "Medium file test:"
	@time go run main.go medium.wav out2.wav 2>&1 | grep "TOTAL:"
	@echo ""
	@echo "Large file test:"
	@time go run main.go large.wav out3.wav 2>&1 | grep "TOTAL:"

# Development: watch for changes and re-benchmark
watch:
	@echo "Watching for changes... (requires: brew install entr)"
	@find . -name '*.go' | entr -c make bench

# Show this help by default
.DEFAULT_GOAL := help
