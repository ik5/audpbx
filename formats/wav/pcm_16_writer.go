package wav

import (
	"encoding/binary"
	"fmt"
	"io"
)

// WriteWAV16 writes a mono 16-bit PCM WAV at sampleRate.  samples must be int16 PCM.
func WriteWAV16(w io.Writer, sampleRate int, samples []int16) error {
    numChannels := uint16(1)
    bitsPerSample := uint16(16)
    byteRate := uint32(sampleRate) * uint32(numChannels) * uint32(bitsPerSample/8)
    blockAlign := uint16(numChannels) * uint16(bitsPerSample/8)
    dataSize := uint32(len(samples) * 2)
    riffSize := 36 + dataSize

    // RIFF header
    if _, err := io.WriteString(w, "RIFF"); err != nil {
        return fmt.Errorf("%w", err)
    }
    if err := binary.Write(w, binary.LittleEndian, riffSize); err != nil {
        return fmt.Errorf("%w", err)
    }
    if _, err := io.WriteString(w, "WAVE"); err != nil {
        return fmt.Errorf("%w", err)
    }

    // fmt chunk
    if _, err := io.WriteString(w, "fmt "); err != nil {
        return fmt.Errorf("%w", err)
    }
    if err := binary.Write(w, binary.LittleEndian, uint32(16)); err != nil { // PCM fmt chunk size
        return fmt.Errorf("%w", err)
    }
    if err := binary.Write(w, binary.LittleEndian, uint16(1)); err != nil { // PCM format
        return fmt.Errorf("%w", err)
    }
    if err := binary.Write(w, binary.LittleEndian, numChannels); err != nil {
        return fmt.Errorf("%w", err)
    }
    if err := binary.Write(w, binary.LittleEndian, uint32(sampleRate)); err != nil {
        return fmt.Errorf("%w", err)
    }
    if err := binary.Write(w, binary.LittleEndian, byteRate); err != nil {
        return fmt.Errorf("%w", err)
    }
    if err := binary.Write(w, binary.LittleEndian, blockAlign); err != nil {
        return fmt.Errorf("%w", err)
    }
    if err := binary.Write(w, binary.LittleEndian, bitsPerSample); err != nil {
        return fmt.Errorf("%w", err)
    }

    // data chunk
    if _, err := io.WriteString(w, "data"); err != nil {
        return fmt.Errorf("%w", err)
    }
    if err := binary.Write(w, binary.LittleEndian, dataSize); err != nil {
        return fmt.Errorf("%w", err)
    }

    // samples
    for _, s := range samples {
        if err := binary.Write(w, binary.LittleEndian, s); err != nil {
            return fmt.Errorf("%w", err)
        }
    }
    return nil
}
