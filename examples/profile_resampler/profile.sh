#!/bin/bash
# SPDX-License-Identifier: EPL-2.0
#
# Automated profiling script for audpbx resampler
# Usage: ./profile.sh <input.wav> [output.wav]

set -e

if [ $# -lt 1 ]; then
    echo "Usage: $0 <input.wav> [output.wav]"
    echo ""
    echo "This script will:"
    echo "  1. Run basic timing analysis"
    echo "  2. Generate CPU profile"
    echo "  3. Generate memory profile"
    echo "  4. Open interactive profile viewer"
    exit 1
fi

INPUT="$1"
OUTPUT="${2:-output.wav}"

if [ ! -f "$INPUT" ]; then
    echo "Error: Input file '$INPUT' not found"
    exit 1
fi

echo "======================================"
echo "Profiling Audio Resampler"
echo "======================================"
echo "Input:  $INPUT"
echo "Output: $OUTPUT"
echo ""

# Create profiles directory
mkdir -p profiles

# Step 1: Basic timing
echo ">>> Step 1: Running basic timing analysis..."
echo ""
go run main.go "$INPUT" "$OUTPUT"
echo ""

# Step 2: CPU profile
echo ">>> Step 2: Generating CPU profile..."
CPU_PROF="profiles/cpu_$(date +%Y%m%d_%H%M%S).prof"
go run main.go "$INPUT" "$OUTPUT" --cpu-profile="$CPU_PROF" > /dev/null 2>&1
echo "CPU profile saved: $CPU_PROF"
echo ""

# Step 3: Memory profile
echo ">>> Step 3: Generating memory profile..."
MEM_PROF="profiles/mem_$(date +%Y%m%d_%H%M%S).prof"
go run main.go "$INPUT" "$OUTPUT" --mem-profile="$MEM_PROF" > /dev/null 2>&1
echo "Memory profile saved: $MEM_PROF"
echo ""

# Step 4: Quick analysis
echo ">>> Step 4: Top CPU consumers..."
echo ""
go tool pprof -top -cum "$CPU_PROF" | head -20
echo ""

# Step 5: Show sample allocations
echo ">>> Step 5: Top memory allocations..."
echo ""
go tool pprof -top -alloc_space "$MEM_PROF" | head -15
echo ""

echo "======================================"
echo "Profiling Complete!"
echo "======================================"
echo ""
echo "Next steps:"
echo ""
echo "  # View CPU profile in browser (best visualization):"
echo "  go tool pprof -http=:8080 $CPU_PROF"
echo ""
echo "  # View memory profile:"
echo "  go tool pprof -http=:8080 $MEM_PROF"
echo ""
echo "  # Interactive command-line mode:"
echo "  go tool pprof $CPU_PROF"
echo "  > top"
echo "  > list CubicInterpolate"
echo "  > list Resampler.ReadSamples"
echo ""
echo "  # Generate PDF call graph (requires graphviz):"
echo "  go tool pprof -pdf $CPU_PROF > profiles/callgraph.pdf"
echo ""

# Offer to open browser
read -p "Open CPU profile in browser now? (y/n) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Opening browser at http://localhost:8080 ..."
    echo "Press Ctrl+C to stop the server when done"
    go tool pprof -http=:8080 "$CPU_PROF"
fi
