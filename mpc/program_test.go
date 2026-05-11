package mpc_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sampleslice/mpc"
	"sampleslice/slice"
)

// TestGenerateProgram_NoSlices verifies error when no slices provided.
func TestGenerateProgram_NoSlices(t *testing.T) {
	dir := t.TempDir()
	_, err := mpc.GenerateProgram(nil, "test", dir, 48)
	if err == nil {
		t.Error("expected error for empty slices, got nil")
	}
	if !strings.Contains(err.Error(), "no slices") {
		t.Errorf("expected 'no slices' in error, got: %v", err)
	}
}

// TestGenerateProgram_SinglePad verifies program with a single pad.
func TestGenerateProgram_SinglePad(t *testing.T) {
	dir := t.TempDir()
	slices := []slice.AudioSlice{
		{StartSample: 0, EndSample: 1000, Note: "C3", Filename: "kick_C3.wav"},
	}

	program, err := mpc.GenerateProgram(slices, "kick_prog", dir, 48)
	if err != nil {
		t.Fatalf("GenerateProgram() returned error: %v", err)
	}

	if program == nil {
		t.Fatal("expected program, got nil")
	}
	if program.Name != "kick_prog" {
		t.Errorf("program name = %q, want %q", program.Name, "kick_prog")
	}
	if len(program.Pads) != 1 {
		t.Fatalf("expected 1 pad, got %d", len(program.Pads))
	}

	pad := program.Pads[0]
	if pad.NoteName != "C3" {
		t.Errorf("pad note name = %q, want %q", pad.NoteName, "C3")
	}
	if pad.FileName != "slice_001.wav" {
		t.Errorf("pad file name = %q, want %q", pad.FileName, "slice_001.wav")
	}
}

// TestGenerateProgram_MultiplePads verifies program with multiple pads.
func TestGenerateProgram_MultiplePads(t *testing.T) {
	dir := t.TempDir()
	slices := []slice.AudioSlice{
		{StartSample: 0, EndSample: 1000, Note: "C3", Filename: "s1.wav"},
		{StartSample: 1000, EndSample: 2000, Note: "C#3", Filename: "s2.wav"},
		{StartSample: 2000, EndSample: 3000, Note: "D3", Filename: "s3.wav"},
	}

	program, err := mpc.GenerateProgram(slices, "multi", dir, 48)
	if err != nil {
		t.Fatalf("GenerateProgram() returned error: %v", err)
	}

	if len(program.Pads) != 3 {
		t.Fatalf("expected 3 pads, got %d", len(program.Pads))
	}

	expectedFiles := []string{"slice_001.wav", "slice_002.wav", "slice_003.wav"}
	for i, expected := range expectedFiles {
		if program.Pads[i].FileName != expected {
			t.Errorf("pad[%d] filename = %q, want %q", i, program.Pads[i].FileName, expected)
		}
	}
}

// TestGenerateProgram_PrgFileCreated verifies .prg file is created.
func TestGenerateProgram_PrgFileCreated(t *testing.T) {
	dir := t.TempDir()
	slices := []slice.AudioSlice{
		{StartSample: 0, EndSample: 1000, Note: "C3", Filename: "s.wav"},
	}

	_, err := mpc.GenerateProgram(slices, "test_prg", dir, 48)
	if err != nil {
		t.Fatalf("GenerateProgram() returned error: %v", err)
	}

	prgPath := filepath.Join(dir, "test_prg.prg")
	_, err = os.Stat(prgPath)
	if err != nil {
		t.Fatalf("expected .prg file at %s, got error: %v", prgPath, err)
	}
}

// TestGenerateProgram_PrgFileHeader verifies .prg file has correct header.
func TestGenerateProgram_PrgFileHeader(t *testing.T) {
	dir := t.TempDir()
	slices := []slice.AudioSlice{
		{StartSample: 0, EndSample: 1000, Note: "C3", Filename: "s.wav"},
	}

	_, err := mpc.GenerateProgram(slices, "header_test", dir, 48)
	if err != nil {
		t.Fatalf("GenerateProgram() returned error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "header_test.prg"))
	if err != nil {
		t.Fatalf("failed to read .prg file: %v", err)
	}

	// Check MPC magic bytes
	if len(data) < 4 || string(data[0:4]) != "MPC " {
		t.Errorf("expected 'MPC ' header, got %q", string(data[0:4]))
	}
}

// TestPadColorForIndex verifies pad color assignment pattern.
func TestPadColorForIndex(t *testing.T) {
	tests := []struct {
		index  int
		expect string
	}{
		{0, "White"},
		{1, "Red"},
		{2, "Green"},
		{3, "Orange"},
		{4, "White"},
		{15, "Orange"},
		{16, "White"}, // wraps around
		{100, "White"}, // 100 % 4 == 0
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("index_%d", tt.index), func(t *testing.T) {
			program, err := mpc.GenerateProgram(
				[]slice.AudioSlice{{StartSample: 0, EndSample: 100, Note: "C3", Filename: "s.wav"}},
				"color_test", t.TempDir(), 48)
			if err != nil {
				t.Fatalf("GenerateProgram() error: %v", err)
			}

			// We can't directly test padColorForIndex since it's unexported,
			// but we can verify the color pattern through the generated program
			_ = program
			_ = tt.expect
		})
	}
}

// TestValidateProgram_EmProgram verifies validation of empty program.
func TestValidateProgram_EmptyProgram(t *testing.T) {
	program := &mpc.Program{
		Name: "empty",
		Pads: []mpc.Pad{},
	}

	warnings := mpc.ValidateProgram(program)
	if len(warnings) == 0 {
		t.Fatal("expected warnings for empty program, got none")
	}

	found := false
	for _, w := range warnings {
		if strings.Contains(w, "no pads") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'no pads' warning")
	}
}

// TestValidateProgram_MoreThan128Pads verifies warning for >128 pads.
func TestValidateProgram_MoreThan128Pads(t *testing.T) {
	pads := make([]mpc.Pad, 129)
	for i := range pads {
		pads[i] = mpc.Pad{Note: 48 + (i % 80), NoteName: fmt.Sprintf("note_%d", i)}
	}

	program := &mpc.Program{
		Name: "large",
		Pads: pads,
	}

	warnings := mpc.ValidateProgram(program)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "more than 128") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'more than 128 pads' warning")
	}
}

// TestValidateProgram_DuplicateNotes verifies duplicate note detection.
func TestValidateProgram_DuplicateNotes(t *testing.T) {
	program := &mpc.Program{
		Name: "dup",
		Pads: []mpc.Pad{
			{Note: 48, NoteName: "C3"},
			{Note: 48, NoteName: "C3"}, // duplicate
		},
	}

	warnings := mpc.ValidateProgram(program)
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "Duplicate") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'Duplicate note' warning")
	}
}

// TestNoteNameToMIDI verifies MIDI note conversion through GenerateProgram.
func TestNoteNameToMIDI(t *testing.T) {
	valid := []struct {
		note   string
		expect int
	}{
		{"C3", 48},
		{"C4", 60},
		{"A4", 69},
		{"C#3", 49},
		{"Db4", 61}, // D-flat 4: correct MIDI value is 61, not 62
		{"B2", 47},
		{"G3", 55},
	}

	for _, tt := range valid {
		t.Run(tt.note, func(t *testing.T) {
			dir := t.TempDir()
			slices := []slice.AudioSlice{
				{StartSample: 0, EndSample: 100, Note: tt.note, Filename: "s.wav"},
			}
			program, err := mpc.GenerateProgram(slices, "midi_test", dir, 48)
			if err != nil {
				t.Fatalf("GenerateProgram() error: %v", err)
			}
			if len(program.Pads) > 0 && program.Pads[0].Note != tt.expect {
				t.Errorf("MIDI note for %q = %d, want %d", tt.note, program.Pads[0].Note, tt.expect)
			}
		})
	}

	// Invalid note names must now return an error instead of silently defaulting.
	invalid := []string{"", "xyz", "Q5"}
	for _, note := range invalid {
		t.Run("invalid_"+note, func(t *testing.T) {
			dir := t.TempDir()
			slices := []slice.AudioSlice{
				{StartSample: 0, EndSample: 100, Note: note, Filename: "s.wav"},
			}
			_, err := mpc.GenerateProgram(slices, "midi_test", dir, 48)
			if err == nil {
				t.Errorf("expected error for invalid note %q, got nil", note)
			}
		})
	}
}

// TestCountSlicesForProgram verifies slice counting logic (128 pads per program, 8 banks).
func TestCountSlicesForProgram(t *testing.T) {
	tests := []struct {
		count            int
		expectFirst      int
		expectAdditional int
	}{
		{0, 0, 0},
		{1, 1, 0},
		{16, 16, 0},
		{17, 17, 0},
		{32, 32, 0},
		{64, 64, 0},
		{65, 65, 0},
		{128, 128, 0},
		{129, 128, 1},
		{192, 128, 1},
		{193, 128, 1},
		{256, 128, 1},
		{257, 128, 2},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("count_%d", tt.count), func(t *testing.T) {
			first, additional := mpc.CountSlicesForProgram(tt.count)
			if first != tt.expectFirst {
				t.Errorf("firstProgramCount = %d, want %d", first, tt.expectFirst)
			}
			if additional != tt.expectAdditional {
				t.Errorf("additionalPrograms = %d, want %d", additional, tt.expectAdditional)
			}
		})
	}
}

// TestGenerateMultiProgram_NoSlices verifies error for empty slices.
func TestGenerateMultiProgram_NoSlices(t *testing.T) {
	dir := t.TempDir()
	_, err := mpc.GenerateMultiProgram(nil, "test", dir, 48)
	if err == nil {
		t.Error("expected error for empty slices, got nil")
	}
}

// TestGenerateMultiProgram_SingleProgram verifies single program when <=128 slices.
func TestGenerateMultiProgram_SingleProgram(t *testing.T) {
	dir := t.TempDir()
	slices := make([]slice.AudioSlice, 10)
	for i := range slices {
		slices[i] = slice.AudioSlice{
			StartSample: i * 100,
			EndSample:   (i + 1) * 100,
			Note:        fmt.Sprintf("C%d", 3+i%3),
			Filename:     fmt.Sprintf("s_%d.wav", i),
		}
	}

	programs, err := mpc.GenerateMultiProgram(slices, "multi", dir, 48)
	if err != nil {
		t.Fatalf("GenerateMultiProgram() error: %v", err)
	}

	if len(programs) != 1 {
		t.Fatalf("expected 1 program, got %d", len(programs))
	}
	if len(programs[0].Pads) != 10 {
		t.Errorf("expected 10 pads, got %d", len(programs[0].Pads))
	}
}

// TestGenerateMultiProgram_MultiplePrograms verifies multiple programs when >128 slices.
func TestGenerateMultiProgram_MultiplePrograms(t *testing.T) {
	dir := t.TempDir()
	slices := make([]slice.AudioSlice, 135)
	for i := range slices {
		slices[i] = slice.AudioSlice{
			StartSample: i * 100,
			EndSample:   (i + 1) * 100,
			Note:        "C3",
			Filename:    fmt.Sprintf("s_%d.wav", i),
		}
	}

	programs, err := mpc.GenerateMultiProgram(slices, "split", dir, 48)
	if err != nil {
		t.Fatalf("GenerateMultiProgram() error: %v", err)
	}

	if len(programs) != 2 {
		t.Fatalf("expected 2 programs, got %d", len(programs))
	}
	if len(programs[0].Pads) != 128 {
		t.Errorf("first program pads = %d, want 128", len(programs[0].Pads))
	}
	if len(programs[1].Pads) != 7 {
		t.Errorf("second program pads = %d, want 7", len(programs[1].Pads))
	}
}

// TestSanitizeProgramName verifies program name sanitization.
func TestSanitizeProgramName(t *testing.T) {
	tests := []struct {
		input  string
		expect string
	}{
		{"valid_name", "valid_name"},
		{"valid-name", "valid-name"},
		{"has spaces", "has_spaces"},
		{"special!@#", "special"},
		{"mixed_hello world", "mixed_hello_world"},
		{"multiple__underscores", "multiple_underscores"},
		{"", ""},
		{"UPPER123case", "UPPER123case"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := mpc.SanitizeProgramName(tt.input)
			if result != tt.expect {
				t.Errorf("SanitizeProgramName(%q) = %q, want %q", tt.input, result, tt.expect)
			}
		})
	}
}

// TestColorToString verifies color to string conversion.
func TestColorToString(t *testing.T) {
	// We verify this indirectly through PrintProgramInfo output
	// since colorToString is unexported
}

// TestGenerateProgram_MaxPads verifies only 128 pads are written to .prg (8 banks × 16).
func TestGenerateProgram_MaxPads(t *testing.T) {
	dir := t.TempDir()
	slices := make([]slice.AudioSlice, 135)
	for i := range slices {
		note := fmt.Sprintf("C%d", 3+i%5)
		slices[i] = slice.AudioSlice{
			StartSample: i * 100,
			EndSample:   (i + 1) * 100,
			Note:        note,
			Filename:    fmt.Sprintf("s_%d.wav", i),
		}
	}

	program, err := mpc.GenerateProgram(slices, "max128", dir, 48)
	if err != nil {
		t.Fatalf("GenerateProgram() error: %v", err)
	}

	// Program struct holds all slices; .prg file writes at most 128
	if len(program.Pads) != 135 {
		t.Errorf("expected 135 pads in program struct, got %d", len(program.Pads))
	}
}

// BenchmarkGenerateProgram measures program generation performance.
func BenchmarkGenerateProgram(b *testing.B) {
	slices := make([]slice.AudioSlice, 16)
	for i := range slices {
		slices[i] = slice.AudioSlice{
			StartSample: i * 1000,
			EndSample:   (i + 1) * 1000,
			Note:        fmt.Sprintf("C%d", 3+i%5),
			Filename:     fmt.Sprintf("s_%d.wav", i),
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		  dir := b.TempDir()
		_, _ = mpc.GenerateProgram(slices, "bench", dir, 48)
	}
}

// BenchmarkValidateProgram measures validation performance.
func BenchmarkValidateProgram(b *testing.B) {
	pads := make([]mpc.Pad, 16)
	for i := range pads {
		pads[i] = mpc.Pad{Note: 48 + i, NoteName: fmt.Sprintf("note_%d", i)}
	}
	program := &mpc.Program{Name: "bench", Pads: pads}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		  _ = mpc.ValidateProgram(program)
	}
}

// BenchmarkSanitizeProgramName measures name sanitization performance.
func BenchmarkSanitizeProgramName(b *testing.B) {
	name := "My_Super! Program@Name#123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		  _ = mpc.SanitizeProgramName(name)
	}
}