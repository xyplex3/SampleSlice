package midi_test

import (
	"testing"

	"sampleslice/midi"
)

// TestParseNote verifies note name parsing into chromatic index and octave.
func TestParseNote(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantIdx    int
		wantOctave int
	}{
		{"C3", "C3", 0, 3},
		{"D#4", "D#4", 3, 4},
		{"A4", "A4", 9, 4},
		{"B2", "B2", 11, 2},
		{"Bb2 flat", "Bb2", 10, 2},
		{"Gb5 flat", "Gb5", 6, 5},
		{"C0 lowest octave", "C0", 0, 0},
		{"unrecognized note defaults to C", "Z3", 0, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotIdx, gotOctave := midi.ParseNote(tt.input)
			if gotIdx != tt.wantIdx {
				t.Errorf("ParseNote(%q) noteIdx = %d, want %d", tt.input, gotIdx, tt.wantIdx)
			}
			if gotOctave != tt.wantOctave {
				t.Errorf("ParseNote(%q) octave = %d, want %d", tt.input, gotOctave, tt.wantOctave)
			}
		})
	}
}

// TestNoteToMIDI verifies note name to MIDI number conversion.
func TestNoteToMIDI(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    int
		wantErr bool
	}{
		{"C3", "C3", 48, false},
		{"A4 standard tuning reference", "A4", 69, false},
		{"C4 middle C", "C4", 60, false},
		{"D#3 sharp", "D#3", 51, false},
		{"Bb3 flat", "Bb3", 58, false},
		{"Db4 flat", "Db4", 61, false},
		{"lowercase normalized", "c#3", 49, false},
		{"whitespace trimmed", " A4 ", 69, false},
		{"empty string", "", 0, true},
		{"invalid note name", "X5", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := midi.NoteToMIDI(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("NoteToMIDI(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("NoteToMIDI(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// TestMIDIToNote verifies MIDI number to note name conversion.
func TestMIDIToNote(t *testing.T) {
	tests := []struct {
		n    int
		want string
	}{
		{0, "C-1"},
		{12, "C0"},
		{48, "C3"},
		{49, "C#3"},
		{60, "C4"},
		{69, "A4"},
		{127, "G9"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := midi.MIDIToNote(tt.n)
			if got != tt.want {
				t.Errorf("MIDIToNote(%d) = %q, want %q", tt.n, got, tt.want)
			}
		})
	}
}

// TestMIDIToNote_RoundTrip verifies NoteToMIDI and MIDIToNote are inverses
// for standard note names.
func TestMIDIToNote_RoundTrip(t *testing.T) {
	notes := []string{"C3", "D#4", "A4", "G#2", "B5"}
	for _, note := range notes {
		n, err := midi.NoteToMIDI(note)
		if err != nil {
			t.Fatalf("NoteToMIDI(%q) unexpected error: %v", note, err)
		}
		got := midi.MIDIToNote(n)
		if got != note {
			t.Errorf("round-trip %q → %d → %q", note, n, got)
		}
	}
}
