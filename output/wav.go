// Package output writes mono 16-bit PCM audio to WAV files.
package output

import (
	"encoding/binary"
	"fmt"
	"io"
	"os"
)

// WriteWAV writes mono 16-bit PCM samples to a WAV file at path.
// Each value in samples must be in the range [–1.0, 1.0]; values outside
// that range are clamped before encoding.
func WriteWAV(path string, samples []float64, sampleRate uint32) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer f.Close()

	return writeWAVToWriter(f, samples, sampleRate)
}

// writeWAVToWriter writes mono 16-bit PCM samples to an io.Writer in WAV format.
// All data is assembled into a single buffer and written in one call to avoid
// per-sample allocations on the hot path.
func writeWAVToWriter(w io.Writer, samples []float64, sampleRate uint32) error {
	dataSize := len(samples) * 2 // 16-bit = 2 bytes per sample
	buf := make([]byte, 44+dataSize)

	// RIFF header
	copy(buf[0:4], "RIFF")
	binary.LittleEndian.PutUint32(buf[4:8], uint32(36+dataSize)) // file size minus 8-byte RIFF header
	copy(buf[8:12], "WAVE")

	// fmt chunk
	copy(buf[12:16], "fmt ")
	binary.LittleEndian.PutUint32(buf[16:20], 16) // chunk size (PCM)
	binary.LittleEndian.PutUint16(buf[20:22], 1)  // audio format (PCM)
	binary.LittleEndian.PutUint16(buf[22:24], 1)  // channels (mono)
	binary.LittleEndian.PutUint32(buf[24:28], sampleRate)
	binary.LittleEndian.PutUint32(buf[28:32], sampleRate*2) // byte rate
	binary.LittleEndian.PutUint16(buf[32:34], 2)            // block align
	binary.LittleEndian.PutUint16(buf[34:36], 16)           // bits per sample

	// data chunk
	copy(buf[36:40], "data")
	binary.LittleEndian.PutUint32(buf[40:44], uint32(dataSize))

	// Sample data
	off := 44
	for _, s := range samples {
		if s > 1.0 {
			s = 1.0
		} else if s < -1.0 {
			s = -1.0
		}
		binary.LittleEndian.PutUint16(buf[off:off+2], uint16(int16(s*32767.0)))
		off += 2
	}

	_, err := w.Write(buf)
	return err
}
