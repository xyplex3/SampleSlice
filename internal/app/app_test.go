package app

import (
	"testing"

	"sampleslice/config"
	"sampleslice/krz"
)

// TestFloatToInt16 verifies float64-to-int16 conversion with clamping.
func TestFloatToInt16(t *testing.T) {
	tests := []struct {
		name    string
		input   []float64
		want    []int16
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
