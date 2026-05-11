// Package report builds and writes machine-readable transient/slice reports.
package report

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os"

	"sampleslice/midi"
	"sampleslice/slice"
)

// SliceEntry describes one detected slice in the report.
type SliceEntry struct {
	Index       int     `json:"index"`       // Zero-based slice index
	Note        string  `json:"note"`        // Assigned MIDI note name (e.g. "C3")
	MIDINote    int     `json:"midi_note"`   // MIDI note number (0-127)
	StartSample int     `json:"start_sample"` // Start sample index in the source audio
	EndSample   int     `json:"end_sample"`   // End sample index in the source audio
	StartSec    float64 `json:"start_sec"`   // Start time in seconds
	EndSec      float64 `json:"end_sec"`     // End time in seconds
	DurationSec float64 `json:"duration_sec"` // Duration in seconds
	Energy      float64 `json:"energy"`      // Normalized RMS energy at the transient onset
}

// Report is the top-level structure written by [WriteJSON] and [WriteCSV].
type Report struct {
	InputFile   string       `json:"input_file"`   // Path to the source WAV file
	SampleRate  uint32       `json:"sample_rate"`  // Sample rate of the source audio in Hz
	Sensitivity float64      `json:"sensitivity"`  // Transient detection sensitivity used (0.0-1.0)
	TotalSlices int          `json:"total_slices"` // Total number of slices produced
	Slices      []SliceEntry `json:"slices"`       // Per-slice detail entries
}

// Build constructs a [Report] from the provided slice list. sensitivity is
// recorded as-is in the report header; pass 0 when grid slicing is used
// (no detection threshold was applied).
func Build(inputPath string, sampleRate uint32, sensitivity float64, slices []slice.AudioSlice) *Report {
	entries := make([]SliceEntry, 0, len(slices))
	sr := float64(sampleRate)
	for i, s := range slices {
		startSec := float64(s.StartSample) / sr
		endSec := float64(s.EndSample) / sr
		midiNote, err := midi.NoteToMIDI(s.Note)
		if err != nil {
			midiNote = 0
		}
		entries = append(entries, SliceEntry{
			Index:       i,
			Note:        s.Note,
			MIDINote:    midiNote,
			StartSample: s.StartSample,
			EndSample:   s.EndSample,
			StartSec:    startSec,
			EndSec:      endSec,
			DurationSec: endSec - startSec,
			Energy:      s.Transient.Energy,
		})
	}
	return &Report{
		InputFile:   inputPath,
		SampleRate:  sampleRate,
		Sensitivity: sensitivity,
		TotalSlices: len(slices),
		Slices:      entries,
	}
}

// WriteJSON marshals r to indented JSON and writes it to path,
// creating or truncating the file as needed.
func WriteJSON(path string, r *Report) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshalling report JSON: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("writing report JSON %s: %w", path, err)
	}
	return nil
}

// WriteCSV writes a header row and one data row per slice to path,
// creating or truncating the file as needed.
func WriteCSV(path string, r *Report) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating report CSV %s: %w", path, err)
	}
	defer f.Close()

	w := csv.NewWriter(f)
	header := []string{
		"index", "note", "midi_note",
		"start_sample", "end_sample",
		"start_sec", "end_sec", "duration_sec", "energy",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, e := range r.Slices {
		row := []string{
			fmt.Sprintf("%d", e.Index),
			e.Note,
			fmt.Sprintf("%d", e.MIDINote),
			fmt.Sprintf("%d", e.StartSample),
			fmt.Sprintf("%d", e.EndSample),
			fmt.Sprintf("%.6f", e.StartSec),
			fmt.Sprintf("%.6f", e.EndSec),
			fmt.Sprintf("%.6f", e.DurationSec),
			fmt.Sprintf("%.6f", e.Energy),
		}
		if err := w.Write(row); err != nil {
			return err
		}
	}
	w.Flush()
	return w.Error()
}
