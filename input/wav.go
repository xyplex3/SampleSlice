// Package input provides WAV file reading and PCM decoding.
// It supports 16-bit, 24-bit, and 32-bit PCM; stereo files are downmixed
// to mono. All sample data is normalized to the range [-1, 1].
package input

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// WAVFile represents a parsed WAV file with metadata and PCM sample data.
type WAVFile struct {
	SampleRate  uint32
	BitDepth    uint16
	Channels    uint16
	Data        []float64 // mono PCM data normalized to [-1, 1]
	DurationSec float64
}

// ReadWAV reads and parses a WAV file, returning a WAVFile struct.
// Supports 16-bit, 24-bit, and 32-bit PCM. Stereo is downmixed to mono.
func ReadWAV(path string) (*WAVFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer f.Close()

	return parseWAV(f)
}

// parseWAV reads WAV format chunks from an io.Reader and returns a parsed WAVFile.
// It processes "fmt " and "data" chunks, skipping unknown chunks.
func parseWAV(r io.Reader) (*WAVFile, error) {
	// Read RIFF header
	header := make([]byte, 12)
	if _, err := io.ReadFull(r, header); err != nil {
		return nil, fmt.Errorf("failed to read RIFF header: %w", err)
	}

	// Validate RIFF and WAV format
	if string(header[0:4]) != "RIFF" {
		return nil, fmt.Errorf("not a RIFF file: got %s", header[0:4])
	}
	if string(header[8:12]) != "WAVE" {
		return nil, fmt.Errorf("not a WAVE file: got %s", header[8:12])
	}

	// Read chunks to find fmt and data
	var fmtData []byte
	var audioData []byte
	var sampleRate uint32
	var bitsPerSample uint16
	var numChannels uint16

	for {
		// Read chunk header (8 bytes)
		chunkHeader := make([]byte, 8)
		n, err := io.ReadFull(r, chunkHeader)
		if n == 0 && err == io.EOF {
			break
		}
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("failed to read chunk header: %w", err)
		}

		chunkID := string(chunkHeader[0:4])
		chunkSize := binary.LittleEndian.Uint32(chunkHeader[4:8])

		switch chunkID {
		case "fmt ":
			fmtData = make([]byte, chunkSize)
			if _, err := io.ReadFull(r, fmtData); err != nil {
				return nil, fmt.Errorf("failed to read fmt chunk: %w", err)
			}
			// Parse fmt chunk
			if len(fmtData) < 16 {
				return nil, fmt.Errorf("fmt chunk too small")
			}
			audioFormat := binary.LittleEndian.Uint16(fmtData[0:2])
			if audioFormat != 1 {
				return nil, fmt.Errorf("unsupported audio format: %d (expected 1 for PCM)", audioFormat)
			}
			numChannels = binary.LittleEndian.Uint16(fmtData[2:4])
			sampleRate = binary.LittleEndian.Uint32(fmtData[4:8])
			bitsPerSample = binary.LittleEndian.Uint16(fmtData[14:16])

		case "data":
			audioData = make([]byte, chunkSize)
			if _, err := io.ReadFull(r, audioData); err != nil {
				return nil, fmt.Errorf("failed to read data chunk: %w", err)
			}

		default:
			// Skip unknown chunks
			if _, err := io.CopyN(io.Discard, r, int64(chunkSize)); err != nil {
				return nil, fmt.Errorf("failed to skip chunk %s: %w", chunkID, err)
			}
		}
	}

	if len(audioData) == 0 {
		return nil, fmt.Errorf("no audio data found")
	}

	// Convert PCM data to float64 samples (normalized to [-1, 1])
	samples, err := decodePCM(audioData, numChannels, bitsPerSample)
	if err != nil {
		return nil, err
	}

	duration := float64(len(samples)) / float64(sampleRate)

	return &WAVFile{
		SampleRate:  sampleRate,
		BitDepth:    bitsPerSample,
		Channels:    numChannels,
		Data:        samples,
		DurationSec: duration,
	}, nil
}

// decodePCM converts raw PCM bytes to normalized float64 mono samples.
// Stereo channels are downmixed by averaging.
func decodePCM(data []byte, channels uint16, bitsPerSample uint16) ([]float64, error) {
	bytesPerSample := bitsPerSample / 8
	bytesPerFrame := int(channels) * int(bytesPerSample)

	if len(data)%bytesPerFrame != 0 {
		return nil, fmt.Errorf("audio data length %d is not a multiple of frame size %d", len(data), bytesPerFrame)
	}

	numFrames := len(data) / bytesPerFrame
	samples := make([]float64, numFrames)

	switch bitsPerSample {
	case 16:
		for i := 0; i < numFrames; i++ {
			offset := i * bytesPerFrame
			var sum float64
			for ch := 0; ch < int(channels); ch++ {
				val := int16(binary.LittleEndian.Uint16(data[offset+ch*2 : offset+ch*2+2]))
				sum += float64(val) / 32768.0
			}
			samples[i] = sum / float64(channels)
		}

	case 24:
		for i := 0; i < numFrames; i++ {
			offset := i * bytesPerFrame
			var sum float64
			for ch := 0; ch < int(channels); ch++ {
				base := offset + ch*3
				// 24-bit little-endian: bytes 0,1,2 -> bits 0-7, 8-15, 16-23
				raw := uint32(data[base]) | (uint32(data[base+1]) << 8) | (uint32(data[base+2]) << 16)
				// Sign-extend 24-bit to int32 via arithmetic shift.
				signed := int32(raw<<8) >> 8
				sum += float64(signed) / 8388608.0
			}
			samples[i] = sum / float64(channels)
		}

	case 32:
		for i := 0; i < numFrames; i++ {
			offset := i * bytesPerFrame
			var sum float64
			for ch := 0; ch < int(channels); ch++ {
				base := offset + ch*4
				val := int32(binary.LittleEndian.Uint32(data[base : base+4]))
				sum += float64(val) / 2147483648.0
			}
			samples[i] = sum / float64(channels)
		}

	default:
		return nil, fmt.Errorf("unsupported bit depth: %d bits", bitsPerSample)
	}

	return samples, nil
}

// Stats holds read-only summary statistics for a WAV file.
// Use [WAVFile.GetStats] to obtain a Stats from a parsed [WAVFile].
type Stats struct {
	Path        string  `json:"path"`
	SampleRate  uint32  `json:"sample_rate"`
	BitDepth    uint16  `json:"bit_depth"`
	Channels    uint16  `json:"channels"`
	DurationSec float64 `json:"duration_sec"`
	SampleCount int     `json:"sample_count"`
}

// GetStats returns a [Stats] summary for the WAV file, using path as the
// reported file path (typically the original file argument).
func (w *WAVFile) GetStats(path string) Stats {
	return Stats{
		Path:        path,
		SampleRate:  w.SampleRate,
		BitDepth:    w.BitDepth,
		Channels:    w.Channels,
		DurationSec: w.DurationSec,
		SampleCount: len(w.Data),
	}
}

// PrintStats writes the WAV file statistics as indented JSON to stdout.
func (w *WAVFile) PrintStats(path string) {
	stats := w.GetStats(path)
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(stats); err != nil {
		fmt.Fprintf(os.Stderr, "error writing stats: %v\n", err)
	}
}
