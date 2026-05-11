package output_test

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"

	"sampleslice/output"
)

// TestWriteWAV_EmptySamples verifies WAV file structure with zero samples.
func TestWriteWAV_EmptySamples(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.wav")

	err := output.WriteWAV(path, []float64{}, 44100)
	if err != nil {
		t.Fatalf("WriteWAV() returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read wav file: %v", err)
	}

	// Minimum WAV file: 44 bytes header + 0 data bytes
	if len(data) != 44 {
		t.Errorf("expected 44 bytes for empty WAV, got %d", len(data))
	}
}

// TestWriteWAV_SingleSample verifies WAV file with a single sample.
func TestWriteWAV_SingleSample(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "single.wav")

	samples := []float64{0.5}
	sampleRate := uint32(44100)

	err := output.WriteWAV(path, samples, sampleRate)
	if err != nil {
		t.Fatalf("WriteWAV() returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read wav file: %v", err)
	}

	// 44 bytes header + 2 bytes data (1 sample * 2 bytes)
	expectedSize := 46
	if len(data) != expectedSize {
		t.Errorf("expected %d bytes, got %d", expectedSize, len(data))
	}
}

// TestWriteWAV_MaxPositiveValue verifies clamping of maximum positive value.
func TestWriteWAV_MaxPositiveValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "max.wav")

	samples := []float64{1.0}
	err := output.WriteWAV(path, samples, 44100)
	if err != nil {
		t.Fatalf("WriteWAV() returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read wav file: %v", err)
	}

	val := int16(binary.LittleEndian.Uint16(data[len(data)-2:]))
	expected := int16(32767)
	if val != expected {
		t.Errorf("sample value = %d, want %d", val, expected)
	}
}

// TestWriteWAV_MaxNegativeValue verifies clamping of maximum negative value.
func TestWriteWAV_MaxNegativeValue(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "min.wav")

	samples := []float64{-1.0}
	err := output.WriteWAV(path, samples, 44100)
	if err != nil {
		t.Fatalf("WriteWAV() returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read wav file: %v", err)
	}

	val := int16(binary.LittleEndian.Uint16(data[len(data)-2:]))
	expected := int16(-32767)
	if val != expected {
		t.Errorf("sample value = %d, want %d", val, expected)
	}
}

// TestWriteWAV_Clamping verifies values outside [-1, 1] are clamped.
func TestWriteWAV_Clamping(t *testing.T) {
	tests := []struct {
		name   string
		input  float64
		expect int16
	}{
		{"aboveRange", 2.0, 32767},
		{"belowRange", -2.0, -32767},
		{"zero", 0.0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tt.name+".wav")

			err := output.WriteWAV(path, []float64{tt.input}, 44100)
			if err != nil {
				t.Fatalf("WriteWAV() returned error: %v", err)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read wav file: %v", err)
			}

			val := int16(binary.LittleEndian.Uint16(data[len(data)-2:]))
			if val != tt.expect {
				t.Errorf("sample value = %d, want %d", val, tt.expect)
			}
		})
	}
}

// TestWriteWAV_WAVHeader verifies the RIFF/WAVE header structure.
func TestWriteWAV_WAVHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "header.wav")

	samples := []float64{0.0, 0.5, -0.5}
	err := output.WriteWAV(path, samples, 44100)
	if err != nil {
		t.Fatalf("WriteWAV() returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read wav file: %v", err)
	}

	if string(data[0:4]) != "RIFF" {
		t.Errorf("expected RIFF header, got %q", string(data[0:4]))
	}
	if string(data[8:12]) != "WAVE" {
		t.Errorf("expected WAVE marker, got %q", string(data[8:12]))
	}
	if string(data[12:16]) != "fmt " {
		t.Errorf("expected fmt chunk, got %q", string(data[12:16]))
	}
	if string(data[36:40]) != "data" {
		t.Errorf("expected data chunk, got %q", string(data[36:40]))
	}
}

// TestWriteWAV_SampleRate verifies sample rate is correctly written.
func TestWriteWAV_SampleRate(t *testing.T) {
	tests := []struct {
		name       string
		sampleRate uint32
	}{
		{"44100hz", 44100},
		{"48000hz", 48000},
		{"96000hz", 96000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, tt.name+".wav")

			samples := []float64{0.1}
			err := output.WriteWAV(path, samples, tt.sampleRate)
			if err != nil {
				t.Fatalf("WriteWAV() returned error: %v", err)
			}

			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read wav file: %v", err)
			}

			writtenRate := binary.LittleEndian.Uint32(data[24:28])
			if writtenRate != tt.sampleRate {
				t.Errorf("sample rate = %d, want %d", writtenRate, tt.sampleRate)
			}
		})
	}
}

// TestWriteWAV_NestedDirectory verifies WriteWAV works when the parent directory exists.
func TestWriteWAV_NestedDirectory(t *testing.T) {
	dir := t.TempDir()
	nestedDir := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatalf("os.MkdirAll() returned error: %v", err)
	}

	path := filepath.Join(nestedDir, "test.wav")
	err := output.WriteWAV(path, []float64{0.0}, 44100)
	if err != nil {
		t.Fatalf("WriteWAV() to nested path returned error: %v", err)
	}

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("expected file to exist at %s, got error: %v", path, err)
	}
}

// TestWriteWAV_InvalidPath verifies error handling for invalid file paths.
func TestWriteWAV_InvalidPath(t *testing.T) {
	samples := []float64{0.5}
	err := output.WriteWAV("/nonexistent/directory/file.wav", samples, 44100)
	if err == nil {
		t.Error("expected error for invalid path, got nil")
	}
}

// TestWriteWAV_LargeFile verifies handling of large sample buffers.
func TestWriteWAV_LargeFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.wav")

	samples := make([]float64, 441000)
	for i := range samples {
		samples[i] = float64(i%1000)/1000.0 - 0.5
	}

	err := output.WriteWAV(path, samples, 44100)
	if err != nil {
		t.Fatalf("WriteWAV() returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read wav file: %v", err)
	}

	expectedSize := 44 + 441000*2
	if len(data) != expectedSize {
		t.Errorf("expected %d bytes, got %d", expectedSize, len(data))
	}
}

// TestWriteWAV_PiWave verifies a sine-like pattern is written without corruption.
func TestWriteWAV_PiWave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sine.wav")

	samples := make([]float64, 4410) // 0.1 second at 44100 Hz
	for i := range samples {
		samples[i] = math.Sin(2 * math.Pi * 440 * float64(i) / 44100)
	}

	err := output.WriteWAV(path, samples, 44100)
	if err != nil {
		t.Fatalf("WriteWAV() returned error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read wav file: %v", err)
	}

	dataChunkSize := binary.LittleEndian.Uint32(data[40:44])
	if dataChunkSize != uint32(4410*2) {
		t.Errorf("data chunk size = %d, want %d", dataChunkSize, 4410*2)
	}
}

// BenchmarkWriteWAV measures WAV writing performance.
func BenchmarkWriteWAV(b *testing.B) {
	samples := make([]float64, 44100) // 1 second
	for i := range samples {
		samples[i] = float64(i%1000)/1000.0 - 0.5
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = output.WriteWAV("/dev/null", samples, 44100)
	}
}
