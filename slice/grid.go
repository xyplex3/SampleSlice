package slice

import (
	"math"

	"sampleslice/detect"
	"sampleslice/midi"
)

// GridConfig holds parameters for beat-grid slicing.
type GridConfig struct {
	BPM         float64 // beats per minute
	LoopBars    int     // bars per slice; 0 defaults to 1
	BeatsPerBar int     // time signature numerator; 0 defaults to 4
	Prefix      string  // filename prefix for exported slices
	RootNote    string  // root MIDI note for the first slice (e.g. "C3")
}

// SliceByGrid divides audio into equal-length segments aligned to the beat grid.
// Returns nil if samples is empty or BPM is zero or negative.
// Tail segments shorter than half a bar are discarded.
func SliceByGrid(samples []float64, sampleRate uint32, cfg GridConfig) []AudioSlice {
	if len(samples) == 0 || cfg.BPM <= 0 {
		return nil
	}

	loopBars := max(cfg.LoopBars, 1)
	beatsPerBar := cfg.BeatsPerBar
	if beatsPerBar < 1 {
		beatsPerBar = 4
	}

	samplesPerBeat := (float64(sampleRate) * 60.0) / cfg.BPM
	samplesPerBar := samplesPerBeat * float64(beatsPerBar)
	samplesPerSlice := max(int(math.Round(samplesPerBar*float64(loopBars))), 1)

	halfBar := int(math.Round(samplesPerBar / 2.0))

	startNoteIdx, octave := midi.ParseNote(cfg.RootNote)

	totalSamples := len(samples)
	var slices []AudioSlice

	for i := 0; ; i++ {
		start := i * samplesPerSlice
		if start >= totalSamples {
			break
		}
		end := start + samplesPerSlice
		if end > totalSamples {
			if totalSamples-start < halfBar {
				break
			}
			end = totalSamples
		}

		note := noteForIndex(i, startNoteIdx, octave)
		filename := cfg.Prefix + "_" + note + ".wav"

		slices = append(slices, AudioSlice{
			StartSample: start,
			EndSample:   end,
			Note:        note,
			Filename:    filename,
			Transient: detect.Transient{
				SamplePos: start,
				TimeSec:   float64(start) / float64(sampleRate),
				Energy:    0,
			},
		})
	}

	return slices
}
