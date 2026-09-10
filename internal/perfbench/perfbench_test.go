// SPDX-License-Identifier: EPL-2.0

// Package perfbench benchmarks the resampling pipeline against real files on
// disk.
//
// This exists because benchmarking against an in-memory source cannot catch the
// class of problem that actually matters here. The pipeline's dominant cost was
// never arithmetic — it was per-call overhead in the decoders and, for
// file-backed sources, one read(2) syscall per audio frame. A mock source that
// copies from a slice has no syscalls and no per-call allocations, so it reports
// a healthy number for a pipeline that takes seconds per minute of audio.
//
// Always benchmark this pipeline through a real decoder reading a real file.
package perfbench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ik5/audpbx"
	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/formats/aiff"
	"github.com/ik5/audpbx/formats/mp3"
	"github.com/ik5/audpbx/formats/vorbis"
	"github.com/ik5/audpbx/formats/wav"
)

// testdataDir is examples/testdata relative to this package.
const testdataDir = "../../examples/testdata"

// sampleStem is a ~103 second 44.1 kHz stereo recording available in every
// supported container.
const sampleStem = "Daniel_Bautista_-_Capricerie_No._5_(Bach,_Paganini)"

// targetRate is the telephony rate the pipeline is tuned for.
const targetRate = 8000

const bufferSize = 4096

func benchmarkPipeline(b *testing.B, ext string, decoder audio.Decoder) {
	b.Helper()

	path := filepath.Join(testdataDir, sampleStem+"."+ext)
	if _, err := os.Stat(path); err != nil {
		b.Skipf("test asset not available: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for range b.N {
		file, err := os.Open(path)
		if err != nil {
			b.Fatal(err)
		}

		src, err := decoder.Decode(file)
		if err != nil {
			b.Fatal(err)
		}

		pcm16, _, err := audpbx.ResampleToMono16(src, targetRate, bufferSize)
		if err != nil {
			b.Fatal(err)
		}

		if len(pcm16) == 0 {
			b.Fatal("pipeline produced no samples")
		}

		file.Close()
	}
}

func BenchmarkPipelineWAV(b *testing.B)  { benchmarkPipeline(b, "wav", wav.Decoder{}) }
func BenchmarkPipelineAIFF(b *testing.B) { benchmarkPipeline(b, "aiff", aiff.Decoder{}) }
func BenchmarkPipelineMP3(b *testing.B)  { benchmarkPipeline(b, "mp3", mp3.Decoder{}) }
func BenchmarkPipelineOGG(b *testing.B)  { benchmarkPipeline(b, "ogg", vorbis.Decoder{}) }
