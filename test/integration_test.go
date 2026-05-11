package test

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"

	"sampleslice/detect"
	"sampleslice/input"
	"sampleslice/output"
	"sampleslice/slice"
)

// createTestWAV creates a test WAV file with synthetic drum-like transients.
// Returns the file path and the number of transients embedded.
func createTestWAV(t *testing.T, dir string, numTransients int) (string, uint32) {
	sampleRate := uint32(44100)
	duration := 5.0 // 5 seconds
	totalSamples := int(float64(sampleRate) * duration)

	samples := make([]float64, totalSamples)

	// Place transients at regular intervals with a sine wave attack
	interval := float64(totalSamples) / float64(numTransients)
	for i := 0; i < numTransients; i++ {
		center := int(float64(i) * interval)
		// Create a transient: sharp attack followed by decay
		for j := 0; j < 500 && center+j < totalSamples; j++ {
			decay := math.Exp(-float64(j) / 200.0)
			samples[center+j] = decay * math.Sin(2*math.Pi*float64(j)*0.01)
		}
	}

	filename := filepath.Join(dir, "test_drum_loop.wav")
	err := output.WriteWAV(filename, samples, sampleRate)
	if err != nil {
		t.Fatalf("Failed to create test WAV: %v", err)
	}

	return filename, sampleRate
}

// TestFullPipeline tests the complete pipeline: read -> detect -> slice -> write
func TestFullPipeline(t *testing.T) {
	// Create temp directory
	tmpDir := t.TempDir()

	// Create test WAV with 8 transients
	wavPath, sampleRate := createTestWAV(t, tmpDir, 8)

	// Step 1: Read the WAV file
	wavData, err := input.ReadWAV(wavPath)
	if err != nil {
		t.Fatalf("Failed to read WAV: %v", err)
	}

	if len(wavData.Data) == 0 {
		t.Fatal("Expected non-empty audio data")
	}

	// Step 2: Detect transients
	transients := detect.DetectTransients(wavData.Data, wavData.SampleRate, 0.5, 50, 0)
	if len(transients) == 0 {
		t.Fatal("Expected to detect at least one transient")
	}

	fmt.Printf("Detected %d transients\n", len(transients))

	// Step 3: Slice audio
	config := slice.SliceConfig{
		PrePaddingMs:  20,
		PostPaddingMs: 200,
		RootNote:      "C3",
		Prefix:        "slice",
	}

	slices := slice.SliceAudio(wavData.Data, transients, wavData.SampleRate, config)
	if len(slices) == 0 {
		t.Fatal("Expected at least one slice")
	}

	fmt.Printf("Created %d slices\n", len(slices))

	// Step 4: Write each slice as a WAV file
	outputDir := filepath.Join(tmpDir, "sliced")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatalf("creating output dir: %v", err)
	}

	for i, s := range slices {
		sliceSamples := slice.ExtractSamples(wavData.Data, s)
		outputPath := filepath.Join(outputDir, s.Filename)
		err := output.WriteWAV(outputPath, sliceSamples, sampleRate)
		if err != nil {
			t.Errorf("Failed to write slice %d: %v", i, err)
			continue
		}

		// Verify the output file exists
		if _, err := os.Stat(outputPath); os.IsNotExist(err) {
			t.Errorf("Expected output file %s does not exist", outputPath)
		}
	}

	fmt.Printf("Successfully exported %d slice WAV files\n", len(slices))
}

// TestStereoDownmix verifies that stereo WAV files are correctly downmixed to mono
func TestStereoDownmix(t *testing.T) {
	tmpDir := t.TempDir()

	sampleRate := uint32(44100)
	totalSamples := 44100 // 1 second

	// Create stereo samples (L and R channels)
	leftChan := make([]float64, totalSamples)
	rightChan := make([]float64, totalSamples)

	for i := 0; i < totalSamples; i++ {
		leftChan[i] = 0.5 * math.Sin(2*math.Pi*float64(i)*440.0/float64(sampleRate))
		rightChan[i] = 0.3 * math.Sin(2*math.Pi*float64(i)*440.0/float64(sampleRate))
	}

	// Write as 16-bit stereo WAV
	filename := filepath.Join(tmpDir, "stereo_test.wav")

	// Create raw stereo PCM data
	rawData := make([]byte, totalSamples*4) // 2 channels * 2 bytes per sample
	for i := 0; i < totalSamples; i++ {
		leftVal := int16(leftChan[i] * 32767.0)
		rightVal := int16(rightChan[i] * 32767.0)

		// Little-endian left
		rawData[i*4] = byte(leftVal)
		rawData[i*4+1] = byte(leftVal >> 8)
		// Little-endian right
		rawData[i*4+2] = byte(rightVal)
		rawData[i*4+3] = byte(rightVal >> 8)
	}

	// Write WAV manually with stereo
	f, err := os.Create(filename)
	if err != nil {
		t.Fatalf("creating stereo WAV: %v", err)
	}
	defer f.Close()

	// Write RIFF header
	fmt.Fprint(f, "RIFF")
	dataSize := len(rawData)
	fileSize := 44 + dataSize - 8
	writeUint32LE(f, uint32(fileSize))
	fmt.Fprint(f, "WAVE")

	// fmt chunk
	fmt.Fprint(f, "fmt ")
	writeUint32LE(f, 16)
	writeUint16LE(f, 1) // PCM
	writeUint16LE(f, 2) // 2 channels
	writeUint32LE(f, sampleRate)
	writeUint32LE(f, sampleRate*4) // byte rate
	writeUint16LE(f, 4)            // block align
	writeUint16LE(f, 16)           // bits per sample

	// data chunk
	fmt.Fprint(f, "data")
	writeUint32LE(f, uint32(dataSize))
	if _, err := f.Write(rawData); err != nil {
		t.Fatalf("writing stereo PCM data: %v", err)
	}

	// Read back and verify mono conversion
	wavData, err := input.ReadWAV(filename)
	if err != nil {
		t.Fatalf("Failed to read stereo WAV: %v", err)
	}

	if wavData.Channels != 2 {
		t.Errorf("Expected 2 channels, got %d", wavData.Channels)
	}

	// The data should be downmixed to mono (length should equal mono sample count)
	expectedMonoSamples := totalSamples
	if len(wavData.Data) != expectedMonoSamples {
		t.Errorf("Expected %d mono samples, got %d", expectedMonoSamples, len(wavData.Data))
	}
}

// TestTransientDetectionEdgeCases tests transient detection with various inputs
func TestTransientDetectionEdgeCases(t *testing.T) {
	sampleRate := uint32(44100)

	// Test 1: Silent audio should produce no transients
	silent := make([]float64, sampleRate) // 1 second of silence
	transients := detect.DetectTransients(silent, sampleRate, 0.5, 50, 0)
	if len(transients) > 0 {
		t.Errorf("Expected no transients in silent audio, got %d", len(transients))
	}

	// Test 2: Single sharp transient
	signal := make([]float64, sampleRate)
	signal[sampleRate/2] = 1.0 // Single spike in the middle
	transients = detect.DetectTransients(signal, sampleRate, 0.3, 10, 0)
	if len(transients) == 0 {
		t.Error("Expected to detect at least one transient for a sharp spike")
	}
}

// TestSliceBoundaries verifies that slices don't exceed audio boundaries
func TestSliceBoundaries(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a short WAV with transients near the edges
	sampleRate := uint32(44100)
	duration := 2.0
	totalSamples := int(float64(sampleRate) * duration)

	samples := make([]float64, totalSamples)
	// Place transients near start and end
	samples[100] = 1.0
	samples[totalSamples-100] = 1.0

	filename := filepath.Join(tmpDir, "boundary_test.wav")
	if err := output.WriteWAV(filename, samples, sampleRate); err != nil {
		t.Fatalf("writing boundary WAV: %v", err)
	}

	wavData, err := input.ReadWAV(filename)
	if err != nil {
		t.Fatalf("reading boundary WAV: %v", err)
	}
	transients := detect.DetectTransients(wavData.Data, wavData.SampleRate, 0.3, 50, 0)

	config := slice.SliceConfig{
		PrePaddingMs:  20,
		PostPaddingMs: 200,
		RootNote:      "C3",
	}

	slices := slice.SliceAudio(wavData.Data, transients, wavData.SampleRate, config)

	for _, s := range slices {
		if s.StartSample < 0 {
			t.Error("Slice start sample is negative")
		}
		if s.EndSample > len(wavData.Data) {
			t.Error("Slice end sample exceeds audio length")
		}
		if s.EndSample <= s.StartSample {
			t.Error("Slice end is before start")
		}
	}
}

// Helper for writing WAV files
func writeUint32LE(f *os.File, v uint32) {
	buf := make([]byte, 4)
	for i := 0; i < 4; i++ {
		buf[i] = byte(v >> (8 * i))
	}
	f.Write(buf)
}

func writeUint16LE(f *os.File, v uint16) {
	buf := make([]byte, 2)
	buf[0] = byte(v)
	buf[1] = byte(v >> 8)
	f.Write(buf)
}
