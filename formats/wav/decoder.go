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
	const maxInt16 float32 = 32768.0 // 2^15 -> +32767

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

	var val int16

	for i := 0; i < samples; i++ {
		val = int16(binary.LittleEndian.Uint16(s.buf[2*i : 2*i+2]))
		dst[i] = float32(val) / maxInt16
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
	// Read RIFF header
	riffHeader := make([]byte, 12)
	if _, err := io.ReadFull(r, riffHeader); err != nil {
		return nil, fmt.Errorf("reading RIFF header: %w", err)
	}

	if !bytes.HasPrefix(riffHeader[:4], []byte("RIFF")) {
		return nil, ErrNotWavFile
	}

	if !bytes.HasPrefix(riffHeader[8:12], []byte("WAVE")) {
		return nil, ErrNotWavFile
	}

	var sampleRate, channels, bitsPerSample int
	var foundFmt, foundData bool
	var chunkID string
	var chunkSize uint32

	chunkHeader := make([]byte, 8)


	// Parse chunks until we find both fmt and data
	for {
		// Read chunk header (4 bytes ID + 4 bytes size)
		if _, err := io.ReadFull(r, chunkHeader); err != nil {
			if err == io.EOF && foundFmt && foundData {
				break
			}
			return nil, fmt.Errorf("reading chunk header: %w", err)
		}

		chunkID = string(chunkHeader[0:4])
		chunkSize = binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			if chunkSize < 16 {
				return nil, fmt.Errorf("fmt chunk too small: %d bytes", chunkSize)
			}

			// Read fmt chunk data
			fmtData := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, fmtData); err != nil {
				return nil, fmt.Errorf("reading fmt chunk: %w", err)
			}

			audioFormat := binary.LittleEndian.Uint16(fmtData[0:2])
			channels = int(binary.LittleEndian.Uint16(fmtData[2:4]))
			sampleRate = int(binary.LittleEndian.Uint32(fmtData[4:8]))
			bitsPerSample = int(binary.LittleEndian.Uint16(fmtData[14:16]))

			if audioFormat != 1 {
				return nil, fmt.Errorf("unsupported audio format: %d (only PCM supported)", audioFormat)
			}

			if bitsPerSample != 16 {
				return nil, ErrOnlyPCM16bitSupported
			}

			foundFmt = true

		case "data":
			if !foundFmt {
				return nil, fmt.Errorf("data chunk before fmt chunk")
			}

			foundData = true

			// The rest of the reader is PCM data
			// We don't need to track chunkSize, just read until EOF
			return &source{
				r:          r,
				sampleRate: sampleRate,
				channels:   channels,
				buf:        make([]byte, 8192),
			}, nil

		default:
			// Skip unknown chunks
			skipBuf := make([]byte, chunkSize)
			if _, err := io.ReadFull(r, skipBuf); err != nil {
				return nil, fmt.Errorf("skipping chunk %s: %w", chunkID, err)
			}

			// WAV chunks are word-aligned (2-byte boundary)
			if chunkSize%2 != 0 {
				padding := make([]byte, 1)
				io.ReadFull(r, padding) // ignore error, might be EOF
			}
		}
	}

	if !foundFmt {
		return nil, ErrUnsupportedWavLayout
	}

	if !foundData {
		return nil, ErrUnsupportedWavChunks
	}

	return nil, fmt.Errorf("unexpected end of WAV file")
}
