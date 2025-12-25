package wav

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ik5/audpbx/audio"
)

type wavSource struct {
    r          io.Reader
    sampleRate int
    channels   int
    // assume PCM 16-bit
    buf        []byte
    tmp        []float32
}

func (s *wavSource) SampleRate() int { return s.sampleRate }
func (s *wavSource) Channels() int   { return s.channels }
func (s *wavSource) Close() error    { return nil }

func (s *wavSource) ReadSamples(dst []float32) (int, error) {
    // Read frames of int16 interleaved, convert to float32
    if len(s.buf) < len(dst)*2 {
        s.buf = make([]byte, len(dst)*2)
    }
    n, err := io.ReadFull(s.r, s.buf[:len(dst)*2])
    if err == io.ErrUnexpectedEOF {
        // Partial frame count
    } else if err != nil {
        if err == io.EOF || err == io.ErrUnexpectedEOF {
            // convert what we have
        } else {
            return 0, fmt.Errorf("%w", err)
        }
    }

    samples := n / 2

    for i := range samples {
        var v int16
        b := s.buf[2*i : 2*i+2]
        v = int16(binary.LittleEndian.Uint16(b))
        dst[i] = float32(v) / 32768.0
    }

    if samples == 0 && (err == io.EOF || err == io.ErrUnexpectedEOF) {
        return 0, io.EOF
    }
    return samples, nil
}

type Decoder struct{}

func (Decoder) Decode(r io.Reader) (audio.Source, error) {
    // Minimal WAV header parse: RIFF/WAVE + fmt/data chunks
    header := make([]byte, 44)

    if _, err := io.ReadFull(r, header); err != nil {
        return nil, fmt.Errorf("%w", err)
    }

    if !bytes.HasPrefix(header[:4], []byte("RIFF")) || !bytes.HasPrefix(header[8:12], []byte("WAVE")) {
        return nil, ErrNotWavFile
    }

    // Parse fmt chunk at 12.., assuming canonical layout
    if !bytes.HasPrefix(header[12:16], []byte("fmt ")) {
        return nil, ErrUnsupportedWavLayout
    }

    audioFormat := binary.LittleEndian.Uint16(header[20:22])
    channels := int(binary.LittleEndian.Uint16(header[22:24]))
    sampleRate := int(binary.LittleEndian.Uint32(header[24:28]))
    bitsPerSample := int(binary.LittleEndian.Uint16(header[34:36]))

    if audioFormat != 1 || bitsPerSample != 16 {
        return nil, ErrOnlyPCM16bitSupported
    }
    // Find "data" chunk start — here we assume 44-byte header with data chunk after fmt
    if !bytes.HasPrefix(header[36:40], []byte("data")) {
        return nil, ErrUnsupportedWavChunks
    }

    return &wavSource{
        r:          r,
        sampleRate: sampleRate,
        channels:   channels,
        buf:        make([]byte, 4096),
    }, nil
}
