// Package midi provides shared MIDI and note-name utilities used across SampleSlice.
package midi

import (
	"fmt"
	"strings"
)

// NoteNames is the chromatic scale (sharps) starting from C.
var NoteNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}

// NoteNameToIndex maps note names (sharps and flats) to chromatic indices 0-11.
var NoteNameToIndex = map[string]int{
	"C": 0, "C#": 1, "Db": 1,
	"D": 2, "D#": 3, "Eb": 3,
	"E": 4, "F": 5,
	"F#": 6, "Gb": 6,
	"G": 7, "G#": 8, "Ab": 8,
	"A": 9, "A#": 10, "Bb": 10,
	"B": 11,
}

// GMDrumNotes defines the sequential MIDI note assignments used when --gm-map is active.
// Each index corresponds to a detected slice position (kick=0, snare=1, …).
var GMDrumNotes = []int{
	36, // Kick
	38, // Rim
	40, // Snare 1
	41, // Tom 1
	42, // Snare 2
	43, // Tom 2
	44, // Clap
	45, // Tom 3
	46, // HiHat Close
	48, // HiHat Pedal
	49, // HiHat Open
	50, // Ride 1
	51, // Splash
	52, // Ride 2
	55, // Crash 1
	56, // Ride 3
	59, // Crash 2
	60, // Boom
	62, // Mid
	64, // Bongo
	65, // Shivari
	68, // Cuica
}

// ParseNote splits a note name like "C3" or "D#4" into a chromatic index (0-11) and octave.
// Unrecognized note names default to C (index 0).
func ParseNote(name string) (noteIdx, octave int) {
	var notePart strings.Builder
	for _, c := range name {
		if c >= '0' && c <= '9' {
			octave = octave*10 + int(c-'0')
		} else {
			notePart.WriteRune(c)
		}
	}
	noteIdx = NoteNameToIndex[notePart.String()]
	return noteIdx, octave
}

// NoteToMIDI converts a note name like "C3" or "D#4" to a MIDI note number (0-127).
// Returns an error if the note name portion is unrecognized.
func NoteToMIDI(name string) (int, error) {
	name = strings.TrimSpace(name)
	var notePart strings.Builder
	octave := 0
	for _, c := range name {
		if c >= '0' && c <= '9' {
			octave = octave*10 + int(c-'0')
		} else {
			notePart.WriteRune(c)
		}
	}
	np := notePart.String()
	if len(np) > 0 {
		// Normalize: uppercase first letter, lowercase suffix (handles "c#"→"C#", "db"→"Db")
		np = strings.ToUpper(np[:1]) + strings.ToLower(np[1:])
	}
	idx, ok := NoteNameToIndex[np]
	if !ok {
		return 0, fmt.Errorf("unrecognized note name %q", name)
	}
	return (octave+1)*12 + idx, nil
}

// MIDIToNote converts a MIDI note number to a note name like "C3".
func MIDIToNote(n int) string {
	octave := (n / 12) - 1
	noteIdx := n % 12
	return NoteNames[noteIdx] + fmt.Sprintf("%d", octave)
}
