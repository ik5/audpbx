package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ik5/audpbx/audio"
	"github.com/ik5/audpbx/formats/mp3"
	"github.com/ik5/audpbx/formats/vorbis"
	"github.com/ik5/audpbx/formats/wav"
	"github.com/ik5/audpbx/utils"
)

func main() {
    if len(os.Args) < 3 {
        fmt.Println("usage: resample <input.{wav|mp3|ogg}> <output.wav>")
        os.Exit(1)
    }
    inPath := os.Args[1]
    outPath := os.Args[2]

    // Registry
    reg := audio.NewRegistry()
    reg.Register("wav", wav.Decoder{})
    reg.Register("mp3", mp3.Decoder{})
    reg.Register("ogg", vorbis.Decoder{})

    ext := filepath.Ext(inPath)
    if len(ext) > 0 {
        ext = ext[1:] // drop dot
    }
    dec, ok := reg.Get(ext)
    if !ok {
        fmt.Println("unsupported format:", ext)
        os.Exit(1)
    }

    inFile, err := os.Open(inPath)
    if err != nil {
        panic(err)
    }
    defer inFile.Close()

    src, err := dec.Decode(inFile)
    if err != nil {
        panic(err)
    }
    defer src.Close()

    // Pipeline: resample -> mono
    res := audio.NewResampler(src, 8000)
    mono := audio.NewMonoMixer(res)

    // Stream and collect int16 samples
    var pcm16 []int16

    // Use a reasonable buffer size for mono output at 8kHz
    // 4096 samples = ~512ms of audio at 8kHz
    var buf = make([]float32, 4096)

    for {
        n, err := mono.ReadSamples(buf)
        if n > 0 {
            for i := range n {
                pcm16 = append(pcm16, utils.Float32ToInt16(buf[i]))
            }
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            panic(err)
        }
    }

    // Write WAV mono 16-bit @ 8 kHz
    outFile, err := os.Create(outPath)
    if err != nil {
        panic(err)
    }
    defer outFile.Close()

    if err := wav.WriteWAV16(outFile, 8000, pcm16); err != nil {
        panic(err)
    }

    fmt.Println("Wrote:", outPath)
}
