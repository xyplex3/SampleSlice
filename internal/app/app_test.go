package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"sampleslice/config"
	"sampleslice/krz"
	"sampleslice/output"
)

// TestFloatToInt16 verifies float64-to-int16 conversion with clamping.
func TestFloatToInt16(t *testing.T) {
	tests := []struct {
		name  string
		input []float64
		want  []int16
	}{
		{
			name:  "zero",
			input: []float64{0.0},
			want:  []int16{0},
		},
		{
			name:  "positive mid",
			input: []float64{0.5},
			want:  []int16{16384},
		},
		{
			name:  "negative mid",
			input: []float64{-0.5},
			want:  []int16{-16384},
		},
		{
			name:  "clips to max at 1.0",
			input: []float64{1.0},
			want:  []int16{32767},
		},
		{
			name:  "clips to min at -1.0",
			input: []float64{-1.0},
			want:  []int16{-32768},
		},
		{
			name:  "clips above 1.0",
			input: []float64{2.0},
			want:  []int16{32767},
		},
		{
			name:  "clips below -1.0",
			input: []float64{-2.0},
			want:  []int16{-32768},
		},
		{
			name:  "empty slice",
			input: []float64{},
			want:  []int16{},
		},
		{
			name:  "multiple values",
			input: []float64{0.0, 0.5, -0.5, 1.0},
			want:  []int16{0, 16384, -16384, 32767},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := floatToInt16(tt.input)
			if len(got) != len(tt.want) {
				t.Fatalf("len = %d, want %d", len(got), len(tt.want))
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("[%d] = %d, want %d", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestEffectiveBeatsPerBar verifies default substitution for zero/negative values.
func TestEffectiveBeatsPerBar(t *testing.T) {
	tests := []struct {
		name       string
		configured int
		want       int
	}{
		{"zero uses default", 0, 4},
		{"negative uses default", -1, 4},
		{"positive returns as-is", 3, 3},
		{"one returns as-is", 1, 1},
		{"seven returns as-is", 7, 7},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveBeatsPerBar(tt.configured)
			if got != tt.want {
				t.Errorf("effectiveBeatsPerBar(%d) = %d, want %d",
					tt.configured, got, tt.want)
			}
		})
	}
}

// TestEffectiveWindowMs verifies default substitution for zero/negative sizes.
func TestEffectiveWindowMs(t *testing.T) {
	tests := []struct {
		name       string
		configured int
		want       int
	}{
		{"zero uses default", 0, 10},
		{"negative uses default", -1, 10},
		{"positive returns as-is", 20, 20},
		{"one returns as-is", 1, 1},
		{"large value unchanged", 500, 500},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveWindowMs(tt.configured)
			if got != tt.want {
				t.Errorf("effectiveWindowMs(%d) = %d, want %d", tt.configured, got, tt.want)
			}
		})
	}
}

// TestVoiceModeString verifies human-readable labels for each mode.
func TestVoiceModeString(t *testing.T) {
	tests := []struct {
		name string
		mode krz.VoiceMode
		want string
	}{
		{"drum mode", krz.VoiceModeDrum, "drum (mono)"},
		{"poly mode", krz.VoiceModePoly, "poly (multi-note)"},
		{"unknown mode", krz.VoiceMode(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := voiceModeString(tt.mode)
			if got != tt.want {
				t.Errorf("voiceModeString(%v) = %q, want %q", tt.mode, got, tt.want)
			}
		})
	}
}

// TestParseNoteMapConfig verifies parsing of "index=note" entries.
func TestParseNoteMapConfig(t *testing.T) {
	t.Run("nil input returns nil", func(t *testing.T) {
		got := parseNoteMapConfig(nil)
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("empty slice returns nil", func(t *testing.T) {
		got := parseNoteMapConfig([]string{})
		if got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("valid entries parsed correctly", func(t *testing.T) {
		got := parseNoteMapConfig([]string{"0=36", "1=38", "2=42"})
		if got[0] != 36 {
			t.Errorf("got[0] = %d, want 36", got[0])
		}
		if got[1] != 38 {
			t.Errorf("got[1] = %d, want 38", got[1])
		}
		if got[2] != 42 {
			t.Errorf("got[2] = %d, want 42", got[2])
		}
	})

	t.Run("invalid entry is skipped", func(t *testing.T) {
		got := parseNoteMapConfig([]string{"0=36", "bad_entry", "2=42"})
		if len(got) != 2 {
			t.Errorf("len = %d, want 2 (bad entry skipped)", len(got))
		}
	})

	t.Run("note above 127 clamped to 127", func(t *testing.T) {
		got := parseNoteMapConfig([]string{"0=200"})
		if got[0] != 127 {
			t.Errorf("got[0] = %d, want 127 (clamped)", got[0])
		}
	})

	t.Run("note below 0 clamped to 0", func(t *testing.T) {
		got := parseNoteMapConfig([]string{"0=-5"})
		if got[0] != 0 {
			t.Errorf("got[0] = %d, want 0 (clamped)", got[0])
		}
	})

	t.Run("boundary note 0 accepted", func(t *testing.T) {
		got := parseNoteMapConfig([]string{"3=0"})
		if got[3] != 0 {
			t.Errorf("got[3] = %d, want 0", got[3])
		}
	})

	t.Run("boundary note 127 accepted", func(t *testing.T) {
		got := parseNoteMapConfig([]string{"5=127"})
		if got[5] != 127 {
			t.Errorf("got[5] = %d, want 127", got[5])
		}
	})
}

// TestBuildProgramName verifies that the output program name embeds the source
// WAV file's base name.
func TestBuildProgramName(t *testing.T) {
	tests := []struct {
		name        string
		programName string
		inputPath   string
		want        string
	}{
		{
			name:        "default name with simple wav",
			programName: "SampleSlice",
			inputPath:   "/path/to/drums.wav",
			want:        "SampleSlice_drums",
		},
		{
			name:        "custom name with simple wav",
			programName: "MyKit",
			inputPath:   "/path/to/kicks.wav",
			want:        "MyKit_kicks",
		},
		{
			name:        "empty name defaults to SampleSlice",
			programName: "",
			inputPath:   "/path/to/snare.wav",
			want:        "SampleSlice_snare",
		},
		{
			name:        "wav with hyphens preserved",
			programName: "Prog",
			inputPath:   "/path/to/my-loop.wav",
			want:        "Prog_my-loop",
		},
		{
			name:        "wav in nested path uses only filename",
			programName: "Beat",
			inputPath:   "/deep/path/to/file/beat_loop.wav",
			want:        "Beat_beat_loop",
		},
		{
			name:        "wav name with spaces sanitized",
			programName: "Kit",
			inputPath:   "/path/to/my loop.wav",
			want:        "Kit_my_loop",
		},
		{
			name:        "program name special chars sanitized",
			programName: "My Kit!",
			inputPath:   "/path/to/drums.wav",
			want:        "My_Kit_drums",
		},
		{
			name:        "wav without extension uses full basename",
			programName: "Kit",
			inputPath:   "/path/to/drums",
			want:        "Kit_drums",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildProgramName(tt.programName, tt.inputPath)
			if got != tt.want {
				t.Errorf("buildProgramName(%q, %q) = %q, want %q",
					tt.programName, tt.inputPath, got, tt.want)
			}
		})
	}
}

// TestResolveNote verifies note resolution priority: custom map, GM map, default.
func TestResolveNote(t *testing.T) {
	t.Run("custom map takes priority", func(t *testing.T) {
		cfg := &config.Config{GMMap: false}
		noteMap := map[int]int{0: 99}
		got := resolveNote(0, "C3", cfg, noteMap)
		if got != 99 {
			t.Errorf("got %d, want 99", got)
		}
	})

	t.Run("falls through to slice note when not in map", func(t *testing.T) {
		cfg := &config.Config{GMMap: false}
		noteMap := map[int]int{5: 99}
		// slice 0 not in map; "C3" = MIDI 48
		got := resolveNote(0, "C3", cfg, noteMap)
		if got != 48 {
			t.Errorf("got %d, want 48 (C3)", got)
		}
	})

	t.Run("gm map used when enabled", func(t *testing.T) {
		cfg := &config.Config{GMMap: true}
		// index 0 → kick = 36 in GM drum map
		got := resolveNote(0, "C3", cfg, nil)
		if got != 36 {
			t.Errorf("got %d, want 36 (GM kick)", got)
		}
	})

	t.Run("gm map index within range", func(t *testing.T) {
		cfg := &config.Config{GMMap: true}
		// index 1 → snare rim in standard GM drum notes
		got := resolveNote(1, "C3", cfg, nil)
		if got != 38 {
			t.Errorf("got %d, want 38 (GM snare)", got)
		}
	})

	t.Run("default uses slice note name", func(t *testing.T) {
		cfg := &config.Config{GMMap: false}
		got := resolveNote(0, "A4", cfg, nil)
		if got != 69 {
			t.Errorf("got %d, want 69 (A4)", got)
		}
	})

	t.Run("unknown note name returns 0", func(t *testing.T) {
		cfg := &config.Config{GMMap: false}
		got := resolveNote(0, "INVALID", cfg, nil)
		if got != 0 {
			t.Errorf("got %d, want 0 (unrecognized note)", got)
		}
	})
}

// writeSilentWAV writes a mono 16-bit WAV of numSamples all-zero samples to
// path, for use as deterministic Run() input: grid slicing only depends on
// sample count and sample rate, never on signal content.
func writeSilentWAV(t *testing.T, path string, sampleRate uint32, numSamples int) {
	t.Helper()
	if err := output.WriteWAV(path, make([]float64, numSamples), sampleRate); err != nil {
		t.Fatalf("writeSilentWAV: %v", err)
	}
}

// baseGridConfig returns a valid Config using beat-grid slicing (BPM > 0),
// which is deterministic and does not depend on transient-detection energy
// heuristics the way the default mode does.
func baseGridConfig(inputPath, outputDir string) *config.Config {
	return &config.Config{
		InputPath:   inputPath,
		OutputDir:   outputDir,
		ProgramName: "Test",
		Format:      config.FormatMPC,
		RootNote:    48,
		Detection:   config.DetectionConfig{Sensitivity: 0.5},
		BPM:         120,
		LoopBars:    1,
		BeatsPerBar: 4,
	}
}

// TestRun_MissingInputFile verifies Run reports a clear error when the input
// WAV path does not exist, without attempting to create any output.
func TestRun_MissingInputFile(t *testing.T) {
	dir := t.TempDir()
	cfg := baseGridConfig(filepath.Join(dir, "missing.wav"), filepath.Join(dir, "out"))

	err := Run(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "does not exist") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "does not exist")
	}
}

// TestRun_InvalidConfig verifies Run surfaces config validation errors before
// touching the filesystem.
func TestRun_InvalidConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := baseGridConfig(filepath.Join(dir, "in.wav"), filepath.Join(dir, "out"))
	cfg.Detection.Sensitivity = 2.0 // out of the valid 0.0-1.0 range

	err := Run(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "configuration error") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "configuration error")
	}
}

// TestRun_NoTransientsFoundReturnsNil verifies that silent audio in the
// default (transient-detection) mode produces no output and no error,
// matching the "nothing to do" warning path in Run.
func TestRun_NoTransientsFoundReturnsNil(t *testing.T) {
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "silent.wav")
	writeSilentWAV(t, wavPath, 8000, 4000)

	outDir := filepath.Join(dir, "out")
	cfg := baseGridConfig(wavPath, outDir)
	cfg.BPM = 0 // use transient detection, not grid mode

	if err := Run(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Errorf("expected output dir to not be created, stat err = %v", err)
	}
}

// TestRun_GridSlicesTooShortReturnsNil verifies that audio shorter than half
// a bar in grid mode produces no output and no error.
func TestRun_GridSlicesTooShortReturnsNil(t *testing.T) {
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "short.wav")
	// 120 BPM, 4/4 at 8000 Hz -> 1 bar = 16000 samples, half a bar = 8000.
	writeSilentWAV(t, wavPath, 8000, 1000)

	outDir := filepath.Join(dir, "out")
	cfg := baseGridConfig(wavPath, outDir)

	if err := Run(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(outDir); !os.IsNotExist(err) {
		t.Errorf("expected output dir to not be created, stat err = %v", err)
	}
}

// TestRun_UnsupportedFormat verifies Run rejects an unrecognized format
// after successfully slicing, rather than silently defaulting.
func TestRun_UnsupportedFormat(t *testing.T) {
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "in.wav")
	writeSilentWAV(t, wavPath, 8000, 20000)

	cfg := baseGridConfig(wavPath, filepath.Join(dir, "out"))
	cfg.Format = config.OutputFormat("bogus")

	err := Run(cfg)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported output format") {
		t.Errorf("error = %q, want to contain %q", err.Error(), "unsupported output format")
	}
}

// TestRun_GridModeProducesOutput verifies a full successful run in beat-grid
// mode for each output format, checking the expected files land on disk.
func TestRun_GridModeProducesOutput(t *testing.T) {
	tests := []struct {
		name       string
		format     config.OutputFormat
		wantSuffix string // expected file suffix under the output dir
	}{
		{"mpc format writes xpm program", config.FormatMPC, ".xpm"},
		{"krz format writes krz program", config.FormatKRZ, ".krz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			wavPath := filepath.Join(dir, "in.wav")
			// 120 BPM, 4/4 at 8000 Hz -> 1 bar = 16000 samples; use enough
			// samples for exactly one slice with no leftover tail.
			writeSilentWAV(t, wavPath, 8000, 20000)

			outDir := filepath.Join(dir, "out")
			cfg := baseGridConfig(wavPath, outDir)
			cfg.Format = tt.format

			if err := Run(cfg); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			entries, err := os.ReadDir(outDir)
			if err != nil {
				t.Fatalf("reading output dir: %v", err)
			}
			found := false
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), tt.wantSuffix) {
					found = true
					break
				}
			}
			if !found {
				names := make([]string, len(entries))
				for i, e := range entries {
					names[i] = e.Name()
				}
				t.Errorf("no file ending in %q found in output dir; got %v", tt.wantSuffix, names)
			}
		})
	}
}

// TestRun_ReportGeneratesFile verifies the --report option writes a report
// file alongside the program output.
func TestRun_ReportGeneratesFile(t *testing.T) {
	dir := t.TempDir()
	wavPath := filepath.Join(dir, "in.wav")
	writeSilentWAV(t, wavPath, 8000, 20000)

	outDir := filepath.Join(dir, "out")
	cfg := baseGridConfig(wavPath, outDir)
	cfg.ReportFormat = "json"

	if err := Run(cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	entries, err := os.ReadDir(outDir)
	if err != nil {
		t.Fatalf("reading output dir: %v", err)
	}
	found := false
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), "_report.json") {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected a _report.json file in the output dir")
	}
}
