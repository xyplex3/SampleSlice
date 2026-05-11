package mpc_test

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"sampleslice/mpc"
	"sampleslice/slice"
)

// buildXPMSlices creates n AudioSlice values with note "C3".
func buildXPMSlices(n int) []slice.AudioSlice {
	s := make([]slice.AudioSlice, n)
	for i := range s {
		s[i] = slice.AudioSlice{
			StartSample: i * 100,
			EndSample:   (i + 1) * 100,
			Note:        "C3",
			Filename:    fmt.Sprintf("s_%d.wav", i),
		}
	}
	return s
}

// buildXPMSamples creates n int16 slices of length perSlice.
func buildXPMSamples(n, perSlice int) [][]int16 {
	out := make([][]int16, n)
	for i := range out {
		out[i] = make([]int16, perSlice)
		for j := range out[i] {
			out[i][j] = int16((j + 1) * 10)
		}
	}
	return out
}

// TestGenerateXPMProgram_XPMFileValid verifies the .xpm file exists and parses as valid XML.
func TestGenerateXPMProgram_XPMFileValid(t *testing.T) {
	dir := t.TempDir()
	_, err := mpc.GenerateXPMProgram(buildXPMSlices(3), "myprog", dir, 48, buildXPMSamples(3, 100), 44100)
	if err != nil {
		t.Fatalf("GenerateXPMProgram() error: %v", err)
	}

	xpmPath := filepath.Join(dir, "myprog.xpm")
	data, err := os.ReadFile(xpmPath)
	if err != nil {
		t.Fatalf("reading .xpm: %v", err)
	}

	var doc mpc.XPMDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("invalid XML in .xpm: %v", err)
	}
}

// TestGenerateXPMProgram_SamplesDir verifies Samples/ subdirectory has the correct WAV count.
func TestGenerateXPMProgram_SamplesDir(t *testing.T) {
	dir := t.TempDir()
	n := 4
	_, err := mpc.GenerateXPMProgram(buildXPMSlices(n), "prog", dir, 48, buildXPMSamples(n, 50), 44100)
	if err != nil {
		t.Fatalf("GenerateXPMProgram() error: %v", err)
	}

	wavs, err := filepath.Glob(filepath.Join(dir, "Samples", "*.wav"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(wavs) != n {
		t.Errorf("WAV count = %d, want %d", len(wavs), n)
	}
}

// TestGenerateXPMProgram_PadCountMatchesSlices verifies pad count in XML equals input slice count.
func TestGenerateXPMProgram_PadCountMatchesSlices(t *testing.T) {
	dir := t.TempDir()
	n := 5
	_, err := mpc.GenerateXPMProgram(buildXPMSlices(n), "prog", dir, 48, buildXPMSamples(n, 50), 44100)
	if err != nil {
		t.Fatalf("GenerateXPMProgram() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "prog.xpm"))
	if err != nil {
		t.Fatalf("reading xpm: %v", err)
	}
	var doc mpc.XPMDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshalling xpm: %v", err)
	}

	if len(doc.Instrument.Pads.Pad) != n {
		t.Errorf("pad count = %d, want %d", len(doc.Instrument.Pads.Pad), n)
	}
}

// TestGenerateXPMProgram_SampleFilePaths verifies SampleFile paths are relative and match files on disk.
func TestGenerateXPMProgram_SampleFilePaths(t *testing.T) {
	dir := t.TempDir()
	n := 3
	_, err := mpc.GenerateXPMProgram(buildXPMSlices(n), "prog", dir, 48, buildXPMSamples(n, 50), 44100)
	if err != nil {
		t.Fatalf("GenerateXPMProgram() error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "prog.xpm"))
	if err != nil {
		t.Fatalf("reading xpm: %v", err)
	}
	var doc mpc.XPMDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		t.Fatalf("unmarshalling xpm: %v", err)
	}

	for i, pad := range doc.Instrument.Pads.Pad {
		relPath := pad.Layer.SampleFile
		absPath := filepath.Join(dir, filepath.FromSlash(relPath))
		if _, err := os.Stat(absPath); os.IsNotExist(err) {
			t.Errorf("pad %d: SampleFile %q not found on disk at %s", i, relPath, absPath)
		}
	}
}

// TestGenerateXPMProgram_NoXPJFile verifies no .xpj is written by the XPM path.
func TestGenerateXPMProgram_NoXPJFile(t *testing.T) {
	dir := t.TempDir()
	_, err := mpc.GenerateXPMProgram(buildXPMSlices(2), "prog", dir, 48, buildXPMSamples(2, 50), 44100)
	if err != nil {
		t.Fatalf("GenerateXPMProgram() error: %v", err)
	}

	if _, err := os.Stat(filepath.Join(dir, "prog.xpj")); !os.IsNotExist(err) {
		t.Error(".xpj file must not be written by GenerateXPMProgram")
	}
}

// TestGenerateXPMProgram_NoSlices verifies an error is returned for empty input.
func TestGenerateXPMProgram_NoSlices(t *testing.T) {
	dir := t.TempDir()
	_, err := mpc.GenerateXPMProgram(nil, "prog", dir, 48, nil, 44100)
	if err == nil {
		t.Fatal("expected error for empty slices, got nil")
	}
}
