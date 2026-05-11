// Package slice divides an audio buffer into discrete regions and assigns
// sequential MIDI note names to each region.
//
// Two slicing modes are provided:
//   - Transient-based: [SliceAudio] segments audio at detected onset events.
//   - Beat-grid: [SliceByGrid] segments audio into equal-length bars aligned
//     to a given tempo.
//
// Additional helpers cover sample extraction ([ExtractSamples]),
// peak normalization ([NormalizeSlice]), silence trimming ([TrimSilence]),
// and energy-profile deduplication ([DeduplicateSlices]).
package slice

import (
	"fmt"
	"sort"

	"sampleslice/detect"
	"sampleslice/midi"
)

// SliceConfig holds the configuration for slicing audio.
type SliceConfig struct {
	PrePaddingMs  int    // Pre-transient padding in milliseconds
	PostPaddingMs int    // Post-transient padding in milliseconds
	RootNote      string // Root note for the first slice (e.g., "C3")
	Prefix        string // Filename prefix for exported slices
}

// AudioSlice represents a single sliced region of audio.
type AudioSlice struct {
	StartSample int              // Start sample index in the original audio
	EndSample   int              // End sample index in the original audio
	Note        string           // Assigned MIDI note name
	Filename    string           // Output filename (without path)
	Transient   detect.Transient // The transient that triggered this slice
}

// msPerSecond is the number of milliseconds in a second, used for padding calculations.
const msPerSecond = 1000

// SliceAudio takes audio samples and detected transients, then produces individual slices.
//
// Each slice starts at the transient position minus pre-padding and ends at the
// midpoint between consecutive transients plus post-padding. The last slice extends
// to the end of the audio buffer. Notes ascend chromatically from the root note.
//
// Parameters:
//   - samples: the full mono audio buffer normalized to [-1, 1]
//   - transients: detected transient positions
//   - sampleRate: audio sample rate in Hz
//   - config: slicing configuration
//
// Returns a slice of AudioSlice structs, each describing a region to export.
// Returns nil if no transients or no samples are provided.
func SliceAudio(samples []float64, transients []detect.Transient, sampleRate uint32, config SliceConfig) []AudioSlice {
	if len(samples) == 0 {
		return nil
	}
	if len(transients) == 0 {
		return nil
	}

	prePaddingSamples := (config.PrePaddingMs * int(sampleRate)) / msPerSecond
	postPaddingSamples := (config.PostPaddingMs * int(sampleRate)) / msPerSecond

	slices := make([]AudioSlice, 0, len(transients))
	totalSamples := len(samples)

	startNoteIdx, octave := midi.ParseNote(config.RootNote)

	for i, t := range transients {
		startSample := t.SamplePos - prePaddingSamples
		if startSample < 0 {
			startSample = 0
		}

		var endSample int
		if i == len(transients)-1 {
			endSample = totalSamples
		} else {
			midpoint := (t.SamplePos + transients[i+1].SamplePos) / 2
			endSample = midpoint + postPaddingSamples
			if endSample > totalSamples {
				endSample = totalSamples
			}
		}

		note := noteForIndex(i, startNoteIdx, octave)
		filename := config.Prefix + "_" + note + ".wav"

		slices = append(slices, AudioSlice{
			StartSample: startSample,
			EndSample:   endSample,
			Note:        note,
			Filename:    filename,
			Transient:   t,
		})
	}

	return slices
}

// ExtractSamples extracts the samples for a given slice from the full buffer.
// Returns nil if the slice starts beyond the buffer.
// The returned slice aliases the original buffer — callers must not modify it in place.
// The input AudioSlice is passed by value to avoid mutating the caller's data.
func ExtractSamples(samples []float64, s AudioSlice) []float64 {
	if s.StartSample >= len(samples) {
		return nil
	}
	if s.EndSample > len(samples) {
		s.EndSample = len(samples)
	}
	return samples[s.StartSample:s.EndSample]
}

// NoteToMIDINote converts a note name such as "C3" or "D#4" to a MIDI note
// number (A4 = 69). Returns 0 for unrecognized note names.
func NoteToMIDINote(note string) int {
	n, err := midi.NoteToMIDI(note)
	if err != nil {
		return 0
	}
	return n
}

// SortBySamplePos sorts AudioSlice values by StartSample position (ascending).
func SortBySamplePos(slices []AudioSlice) {
	sort.Slice(slices, func(i, j int) bool {
		return slices[i].StartSample < slices[j].StartSample
	})
}

// noteForIndex generates a note name for a given slice index, starting from the root note.
func noteForIndex(index int, startNoteIdx int, octave int) string {
	totalIdx := startNoteIdx + index
	currentOctave := octave + totalIdx/12
	noteIdx := totalIdx % 12
	return midi.NoteNames[noteIdx] + fmt.Sprintf("%d", currentOctave)
}
