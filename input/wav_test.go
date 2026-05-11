package input_test

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"testing"

	"sampleslice/input"
)

// buildMinimalWAV constructs a minimal valid WAV file in memory.
func buildMinimalWAV(t *testing.T, samples []int16, numChannels uint16) []byte {
	t.Helper()

	// Build PCM data
	var pcm bytes.Buffer
	for i := 0; i < len(samples); i += int(numChannels) {
		for ch := 0; ch < int(numChannels); ch++ {
			idx := i + ch
			if idx < len(samples) {
				binary.Write(&pcm, binary.LittleEndian, samples[idx])
			}
		}
	}

	dataSize := uint32(pcm.Len())
	fmtSize := uint32(16) // standard PCM fmt chunk size
	fileSize := 4 + fmtSize + 8 + dataSize // RIFF size = everything after this uint32

	var buf bytes.Buffer
	// RIFF header
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, fileSize)
	buf.WriteString("WAVE")

	// fmt chunk
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, fmtSize)
	binary.Write(&buf, binary.LittleEndian, uint16(1))        // PCM format
	binary.Write(&buf, binary.LittleEndian, numChannels)
	binary.Write(&buf, binary.LittleEndian, uint32(44100))     // sample rate
	binary.Write(&buf, binary.LittleEndian, uint32(44100*2))   // byte rate (mono 16-bit)
	binary.Write(&buf, binary.LittleEndian, uint16(2))          // block align (mono 16-bit)
	binary.Write(&buf, binary.LittleEndian, uint16(16))         // bits per sample

	// data chunk
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataSize)
	buf.Write(pcm.Bytes())

	return buf.Bytes()
}

// TestReadWAV_NonExistent verifies that a missing file returns an error.
func TestReadWAV_NonExistent(t *testing.T) {
	_, err := input.ReadWAV("/nonexistent/file.wav")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

// TestReadWAV_NotAWAV verifies that an invalid file returns an error.
func TestReadWAV_NotAWAV(t *testing.T) {
	tmpFile := t.TempDir() + "/fake.wav"
	if err := os.WriteFile(tmpFile, []byte("not a wav file"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := input.ReadWAV(tmpFile)
	if err == nil {
		t.Fatal("expected error for non-WAV file, got nil")
	}
}

// TestReadWAV_Mono16Bit verifies reading a basic mono 16-bit WAV file.
func TestReadWAV_Mono16Bit(t *testing.T) {
	samples := []int16{0, 100, -100, 32767, -32768}
	wavData := buildMinimalWAV(t, samples, 1)

	tmpFile := t.TempDir() + "/mono.wav"
	if err := os.WriteFile(tmpFile, wavData, 0644); err != nil {
		t.Fatalf("failed to write test wav: %v", err)
	}

	wav, err := input.ReadWAV(tmpFile)
	if err != nil {
		t.Fatalf("ReadWAV() error = %v", err)
	}

	if wav.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", wav.SampleRate)
	}
	if wav.BitDepth != 16 {
		t.Errorf("BitDepth = %d, want 16", wav.BitDepth)
	}
	if wav.Channels != 1 {
		t.Errorf("Channels = %d, want 1", wav.Channels)
	}
	if len(wav.Data) != len(samples) {
		t.Errorf("sample count = %d, want %d", len(wav.Data), len(samples))
	}

	// Verify first sample is 0
	if wav.Data[0] != 0 {
		t.Errorf("Data[0] = %f, want 0", wav.Data[0])
	}

	// Verify max sample (32767) normalizes close to 1
	if wav.Data[3] < 0.99 {
		t.Errorf("Data[3] = %f, want ~1.0", wav.Data[3])
	}

	// Verify min sample (-32768) normalizes close to -1
	if wav.Data[4] > -0.99 {
		t.Errorf("Data[4] = %f, want ~-1.0", wav.Data[4])
	}
}

// TestReadWAV_Stereo16Bit verifies reading and downmixing a stereo WAV.
func TestReadWAV_Stereo16Bit(t *testing.T) {
	// Left channel: 32767, Right channel: -32768 -> downmixed should be ~0
	samples := []int16{32767, -32768}
	wavData := buildMinimalWAV(t, samples, 2)

	tmpFile := t.TempDir() + "/stereo.wav"
	if err := os.WriteFile(tmpFile, wavData, 0644); err != nil {
		t.Fatalf("failed to write test wav: %v", err)
	}

	wav, err := input.ReadWAV(tmpFile)
	if err != nil {
		t.Fatalf("ReadWAV() error = %v", err)
	}

	if wav.Channels != 2 {
		t.Errorf("Channels = %d, want 2", wav.Channels)
	}
	if len(wav.Data) != 1 {
		t.Errorf("sample count = %d, want 1", len(wav.Data))
	}

	// Downmix of max positive and max negative should be near zero
	if math.Abs(wav.Data[0]) > 0.01 {
		t.Errorf("downmixed sample = %f, want ~0", wav.Data[0])
	}
}

// TestGetStats verifies the GetStats method returns correct values.
func TestGetStats(t *testing.T) {
	samples := []int16{100, 200, 300}
	wavData := buildMinimalWAV(t, samples, 1)

	tmpFile := t.TempDir() + "/stats.wav"
	if err := os.WriteFile(tmpFile, wavData, 0644); err != nil {
		t.Fatalf("failed to write test wav: %v", err)
	}

	wav, err := input.ReadWAV(tmpFile)
	if err != nil {
		t.Fatalf("ReadWAV() error = %v", err)
	}

	stats := wav.GetStats(tmpFile)

	if stats.Path != tmpFile {
		t.Errorf("Path = %q, want %q", stats.Path, tmpFile)
	}
	if stats.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", stats.SampleRate)
	}
	if stats.SampleCount != 3 {
		t.Errorf("SampleCount = %d, want 3", stats.SampleCount)
	}
	expectedDuration := float64(3) / 44100
	if stats.DurationSec != expectedDuration {
		t.Errorf("DurationSec = %f, want %f", stats.DurationSec, expectedDuration)
	}
}

// TestReadWAV_TruncatedHeader verifies error handling for truncated files.
func TestReadWAV_TruncatedHeader(t *testing.T) {
	tmpFile := t.TempDir() + "/truncated.wav"
	// Only 6 bytes instead of 12 for RIFF header
	if err := os.WriteFile(tmpFile, []byte("RIFFxx"), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	_, err := input.ReadWAV(tmpFile)
	if err == nil {
		t.Fatal("expected error for truncated header, got nil")
	}
}

// TestReadWAV_UnknownChunk verifies that unknown chunks are skipped gracefully.
func TestReadWAV_UnknownChunk(t *testing.T) {
	samples := []int16{100, 200}
	wavData := buildMinimalWAV(t, samples, 1)

	// Insert an unknown chunk before the data chunk
	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, uint32(0)) // placeholder
	buf.WriteString("WAVE")

	// Unknown chunk
	buf.WriteString("IGNR")
	binary.Write(&buf, binary.LittleEndian, uint32(4))
	buf.WriteString("test")

	// Append the rest of the wav data after "WAVE"
	rest := wavData[12:] // skip "RIFF" + 4 bytes + "WAVE"
	buf.Write(rest)

	// Fix the RIFF size field
	finalData := buf.Bytes()
	actualSize := uint32(len(finalData) - 8)
	binary.LittleEndian.PutUint32(finalData[4:8], actualSize)

	tmpFile := t.TempDir() + "/unknown_chunk.wav"
	if err := os.WriteFile(tmpFile, finalData, 0644); err != nil {
		t.Fatalf("failed to write test wav: %v", err)
	}

	wav, err := input.ReadWAV(tmpFile)
	if err != nil {
		t.Fatalf("ReadWAV() with unknown chunk error = %v", err)
	}

	if len(wav.Data) != 2 {
		t.Errorf("sample count = %d, want 2", len(wav.Data))
	}
}

// BenchmarkReadWAV measures WAV reading performance.
func BenchmarkReadWAV(b *testing.B) {
	// Generate 1 second of 16-bit mono audio at 44100 Hz
	samples := make([]int16, 44100)
	for i := range samples {
		samples[i] = int16(i % 32768)
	}
	wavData := buildMinimalWAV(&testing.T{}, samples, 1)

	tmpFile := b.TempDir() + "/benchmark.wav"
	if err := os.WriteFile(tmpFile, wavData, 0644); err != nil {
		b.Fatalf("failed to write benchmark wav: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = input.ReadWAV(tmpFile)
	}
}

// buildWAVWithBitDepth constructs a mono WAV in memory with the given bit depth.
// pcmBytes is the raw PCM data to embed.
func buildWAVWithBitDepth(t *testing.T, pcmBytes []byte, bitsPerSample uint16) []byte {
	t.Helper()
	const sampleRate = uint32(44100)
	const numChannels = uint16(1)
	blockAlign := numChannels * (bitsPerSample / 8)
	byteRate := uint32(sampleRate) * uint32(blockAlign)
	dataSize := uint32(len(pcmBytes))
	fmtSize := uint32(16)
	fileSize := 4 + 8 + fmtSize + 8 + dataSize

	var buf bytes.Buffer
	buf.WriteString("RIFF")
	binary.Write(&buf, binary.LittleEndian, fileSize)
	buf.WriteString("WAVE")
	buf.WriteString("fmt ")
	binary.Write(&buf, binary.LittleEndian, fmtSize)
	binary.Write(&buf, binary.LittleEndian, uint16(1)) // PCM
	binary.Write(&buf, binary.LittleEndian, numChannels)
	binary.Write(&buf, binary.LittleEndian, sampleRate)
	binary.Write(&buf, binary.LittleEndian, byteRate)
	binary.Write(&buf, binary.LittleEndian, blockAlign)
	binary.Write(&buf, binary.LittleEndian, bitsPerSample)
	buf.WriteString("data")
	binary.Write(&buf, binary.LittleEndian, dataSize)
	buf.Write(pcmBytes)
	return buf.Bytes()
}

// TestReadWAV_24Bit verifies 24-bit PCM decoding and normalization.
func TestReadWAV_24Bit(t *testing.T) {
	// Build three 24-bit little-endian samples: 0, max (+8388607), min (-8388608).
	var pcm bytes.Buffer
	for _, v := range []int32{0, 8388607, -8388608} {
		pcm.WriteByte(byte(v))
		pcm.WriteByte(byte(v >> 8))
		pcm.WriteByte(byte(v >> 16))
	}
	wavBytes := buildWAVWithBitDepth(t, pcm.Bytes(), 24)

	tmp := t.TempDir() + "/test24.wav"
	if err := os.WriteFile(tmp, wavBytes, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	wav, err := input.ReadWAV(tmp)
	if err != nil {
		t.Fatalf("ReadWAV() error = %v", err)
	}
	if len(wav.Data) != 3 {
		t.Fatalf("sample count = %d, want 3", len(wav.Data))
	}
	if wav.Data[0] != 0 {
		t.Errorf("Data[0] = %f, want 0", wav.Data[0])
	}
	if wav.Data[1] < 0.999 {
		t.Errorf("Data[1] = %f, want ~1.0", wav.Data[1])
	}
	if wav.Data[2] > -0.999 {
		t.Errorf("Data[2] = %f, want ~-1.0", wav.Data[2])
	}
}

// TestReadWAV_32Bit verifies 32-bit PCM decoding and normalization.
func TestReadWAV_32Bit(t *testing.T) {
	// Three 32-bit samples: 0, max (+2147483647), min (-2147483648).
	var pcm bytes.Buffer
	for _, v := range []int32{0, 2147483647, -2147483648} {
		binary.Write(&pcm, binary.LittleEndian, v)
	}
	wavBytes := buildWAVWithBitDepth(t, pcm.Bytes(), 32)

	tmp := t.TempDir() + "/test32.wav"
	if err := os.WriteFile(tmp, wavBytes, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	wav, err := input.ReadWAV(tmp)
	if err != nil {
		t.Fatalf("ReadWAV() error = %v", err)
	}
	if len(wav.Data) != 3 {
		t.Fatalf("sample count = %d, want 3", len(wav.Data))
	}
	if wav.Data[0] != 0 {
		t.Errorf("Data[0] = %f, want 0", wav.Data[0])
	}
	if wav.Data[1] < 0.999 {
		t.Errorf("Data[1] = %f, want ~1.0", wav.Data[1])
	}
	if wav.Data[2] > -0.999 {
		t.Errorf("Data[2] = %f, want ~-1.0", wav.Data[2])
	}
}

// TestReadWAV_UnsupportedBitDepth verifies that an unsupported bit depth returns an error.
func TestReadWAV_UnsupportedBitDepth(t *testing.T) {
	// 8-bit PCM is not supported by the decoder.
	pcm := []byte{128, 0, 255} // three 8-bit samples
	wavBytes := buildWAVWithBitDepth(t, pcm, 8)

	tmp := t.TempDir() + "/test8.wav"
	if err := os.WriteFile(tmp, wavBytes, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := input.ReadWAV(tmp)
	if err == nil {
		t.Fatal("expected error for unsupported 8-bit depth, got nil")
	}
}

// TestPrintStats_Smoke verifies PrintStats does not panic on a valid WAVFile.
func TestPrintStats_Smoke(t *testing.T) {
	samples := []int16{100, 200, -100}
	wavBytes := buildMinimalWAV(t, samples, 1)

	tmp := t.TempDir() + "/stats.wav"
	if err := os.WriteFile(tmp, wavBytes, 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	wav, err := input.ReadWAV(tmp)
	if err != nil {
		t.Fatalf("ReadWAV(): %v", err)
	}
	// Redirect stdout so test output stays clean.
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	wav.PrintStats(tmp)
	w.Close()
	os.Stdout = old
	r.Close()
}

func Example_readWAV() {
	fmt.Println("ReadWAV example")
	// Output: ReadWAV example
}