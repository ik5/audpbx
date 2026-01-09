package wav

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/ik5/audpbx/audio"
)

type source struct {
	r          io.Reader
	sampleRate int
	channels   int
	buf        []byte // byte buffer for reading PCM data
}

func (s *source) SampleRate() int { return s.sampleRate }
func (s *source) Channels() int   { return s.channels }
func (s *source) Close() error    { return nil }
func (s *source) BufSize() int    { return cap(s.buf) / 2 } // return sample capacity

func (s *source) ReadSamples(dst []float32) (int, error) {
	if len(dst) == 0 {
		return 0, nil
	}

	// Each sample is 2 bytes (int16 PCM)
	bytesNeeded := len(dst) * 2

	// Ensure buffer is large enough
	if cap(s.buf) < bytesNeeded {
		s.buf = make([]byte, bytesNeeded)
	}
	s.buf = s.buf[:bytesNeeded]

	// Read bytes from source
	n, err := io.ReadFull(s.r, s.buf)

	// Handle partial reads
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		// We might have partial data
		if n == 0 {
			return 0, io.EOF
		}
		// Ensure we have complete samples (even number of bytes)
		n = (n / 2) * 2
	} else if err != nil {
		return 0, fmt.Errorf("%w", err)
	}

	// Convert bytes to samples
	samples := n / 2
	for i := 0; i < samples; i++ {
		val := int16(binary.LittleEndian.Uint16(s.buf[2*i : 2*i+2]))
		dst[i] = float32(val) / 32768.0
	}

	// Return EOF only if we got no samples
	if samples == 0 {
		return 0, io.EOF
	}

	// If we got partial data, return it with EOF
	if err == io.ErrUnexpectedEOF || err == io.EOF {
		return samples, io.EOF
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

	// Parse fmt chunk at offset 12
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

	// Expect "data" chunk at offset 36
	if !bytes.HasPrefix(header[36:40], []byte("data")) {
		return nil, ErrUnsupportedWavChunks
	}

	// Data chunk size is at offset 40-44, but we'll just read until EOF
	// dataSize := binary.LittleEndian.Uint32(header[40:44])

	return &source{
		r:          r,
		sampleRate: sampleRate,
		channels:   channels,
		buf:        make([]byte, 8192),
	}, nil
}
