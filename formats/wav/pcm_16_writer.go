// SPDX-License-Identifier: EPL-2.0

package wav

import (
	"encoding/binary"
	"fmt"
	"io"
)

// WriteWAV16 writes a mono 16-bit PCM WAV at sampleRate.  samples must be int16 PCM.
// This uses an optimized implementation for minimal allocations.
func WriteWAV16(w io.Writer, sampleRate int, samples []int16) error {
	numChannels := uint16(1)
	bitsPerSample := uint16(16)
	byteRate := uint32(sampleRate) * uint32(numChannels) * uint32(bitsPerSample/8)
	blockAlign := uint16(numChannels) * uint16(bitsPerSample/8)
	dataSize := uint32(len(samples) * 2)
	riffSize := 36 + dataSize

	// Pre-allocate buffer for entire header (44 bytes)
	header := make([]byte, 44)

	// RIFF header (12 bytes)
	copy(header[0:4], "RIFF")
	binary.LittleEndian.PutUint32(header[4:8], riffSize)
	copy(header[8:12], "WAVE")

	// fmt chunk (24 bytes)
	copy(header[12:16], "fmt ")
	binary.LittleEndian.PutUint32(header[16:20], 16) // PCM fmt chunk size
	binary.LittleEndian.PutUint16(header[20:22], 1)  // PCM format
	binary.LittleEndian.PutUint16(header[22:24], numChannels)
	binary.LittleEndian.PutUint32(header[24:28], uint32(sampleRate))
	binary.LittleEndian.PutUint32(header[28:32], byteRate)
	binary.LittleEndian.PutUint16(header[32:34], blockAlign)
	binary.LittleEndian.PutUint16(header[34:36], bitsPerSample)

	// data chunk header (8 bytes)
	copy(header[36:40], "data")
	binary.LittleEndian.PutUint32(header[40:44], dataSize)

	// Write header in one operation
	if _, err := w.Write(header); err != nil {
		return fmt.Errorf("%w", err)
	}

	// Convert samples to bytes efficiently
	// For better performance with large files, write in chunks
	const chunkSize = 8192 // Write 8KB at a time
	if len(samples) == 0 {
		return nil
	}

	// Allocate buffer for chunk writes
	bufSize := min(len(samples), chunkSize)
	buf := make([]byte, bufSize*2)

	for i := 0; i < len(samples); i += chunkSize {
		end := min(i+chunkSize, len(samples))
		chunk := samples[i:end]

		// Resize buf if needed for last chunk
		if len(chunk)*2 > len(buf) {
			buf = buf[:len(chunk)*2]
		} else {
			buf = buf[:len(chunk)*2]
		}

		// Convert int16 samples to bytes
		for j, s := range chunk {
			binary.LittleEndian.PutUint16(buf[j*2:j*2+2], uint16(s))
		}

		if _, err := w.Write(buf); err != nil {
			return fmt.Errorf("%w", err)
		}
	}

	return nil
}
