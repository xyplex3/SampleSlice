// Package app coordinates the end-to-end SampleSlice pipeline: reading a WAV
// file, detecting transients, slicing the audio, and writing the requested
// output format (Akai MPC .xpm, or Kurzweil .krz).
package app

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"sampleslice/config"
	"sampleslice/detect"
	"sampleslice/input"
	"sampleslice/krz"
	"sampleslice/midi"
	"sampleslice/mpc"
	"sampleslice/report"
	"sampleslice/slice"
)

// floatToInt16 converts normalized float64 samples to clamped int16 PCM.
func floatToInt16(samples []float64) []int16 {
	out := make([]int16, len(samples))
	for i, f := range samples {
		v := int32(f * 32768.0)
		if v > 32767 {
			v = 32767
		} else if v < -32768 {
			v = -32768
		}
		out[i] = int16(v)
	}
	return out
}

// Run executes the full SampleSlice pipeline for the given configuration.
// It reads the input WAV, detects transients, builds audio slices, and writes
// the output format specified by cfg.Format. Returns an error if any step fails.
func Run(cfg *config.Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	if _, err := os.Stat(cfg.InputPath); os.IsNotExist(err) {
		return fmt.Errorf("input file does not exist: %s", cfg.InputPath)
	}

	outputDir := cfg.OutputDir
	if outputDir == "" {
		dir := filepath.Dir(cfg.InputPath)
		baseName := filepath.Base(cfg.InputPath)
		nameWithoutExt := baseName[:len(baseName)-len(filepath.Ext(baseName))]
		outputDir = filepath.Join(dir, nameWithoutExt+"_sliced")
	}

	programName := buildProgramName(cfg.ProgramName, cfg.InputPath)

	startTime := time.Now()

	// Step 1: Read input WAV file
	fmt.Printf("Reading input file: %s\n", cfg.InputPath)
	wavData, err := input.ReadWAV(cfg.InputPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", cfg.InputPath, err)
	}
	fmt.Printf("  Sample rate: %d Hz\n", wavData.SampleRate)
	fmt.Printf("  Channels: %d\n", wavData.Channels)
	fmt.Printf("  Bits per sample: %d\n", wavData.BitDepth)
	fmt.Printf("  Duration: %.2f seconds\n", wavData.DurationSec)
	fmt.Printf("  Total samples: %d\n", len(wavData.Data))
	fmt.Println()

	samples := wavData.Data
	fmt.Printf("Mono samples: %d\n\n", len(samples))

	rootNoteName := midi.MIDIToNote(cfg.RootNote)

	// Step 2 & 3: Slice — either beat-grid or transient-detection mode
	var audioSlices []slice.AudioSlice
	reportSensitivity := cfg.Detection.Sensitivity

	if cfg.BPM > 0 {
		fmt.Printf("Slicing mode: tempo grid (%.1f BPM, %d-bar loops, %d beats/bar)\n",
			cfg.BPM, cfg.LoopBars, effectiveBeatsPerBar(cfg.BeatsPerBar))
		audioSlices = slice.SliceByGrid(samples, wavData.SampleRate, slice.GridConfig{
			BPM:         cfg.BPM,
			LoopBars:    cfg.LoopBars,
			BeatsPerBar: cfg.BeatsPerBar,
			Prefix:      programName,
			RootNote:    rootNoteName,
		})
		reportSensitivity = 0
		if len(audioSlices) == 0 {
			fmt.Println("Warning: No grid slices produced. Recording may be shorter than one bar.")
			fmt.Println("Exiting without creating output.")
			return nil
		}
		fmt.Printf("  Produced %d loop slices\n", len(audioSlices))
	} else {
		fmt.Printf("Slicing mode: transient detection (sensitivity: %.2f, min interval: %dms, window: %dms)\n",
			cfg.Detection.Sensitivity, cfg.Detection.MinInterval, effectiveWindowMs(cfg.Detection.WindowSizeMs))
		transients := detect.DetectTransients(
			samples, wavData.SampleRate,
			cfg.Detection.Sensitivity, cfg.Detection.MinInterval, cfg.Detection.WindowSizeMs,
		)
		fmt.Printf("  Detected %d transients\n", len(transients))
		if len(transients) > 0 {
			fmt.Printf("  First transient: %.3fs (energy: %.3f)\n", transients[0].TimeSec, transients[0].Energy)
			fmt.Printf("  Last transient: %.3fs (energy: %.3f)\n", transients[len(transients)-1].TimeSec, transients[len(transients)-1].Energy)
		}
		fmt.Println()

		if len(transients) == 0 {
			fmt.Println("Warning: No transients detected. Try lowering the sensitivity (-sensitivity flag).")
			fmt.Println("Exiting without creating output.")
			return nil
		}

		audioSlices = slice.SliceAudio(samples, transients, wavData.SampleRate, slice.SliceConfig{
			PrePaddingMs:  cfg.PrePadding,
			PostPaddingMs: cfg.PostPadding,
			RootNote:      rootNoteName,
		})
		fmt.Printf("  Created %d slices\n", len(audioSlices))
	}

	if cfg.SimilarityThreshold > 0 {
		before := len(audioSlices)
		audioSlices = slice.DeduplicateSlices(audioSlices, samples, cfg.SimilarityThreshold)
		fmt.Printf("  Deduplicated: %d → %d unique slices (threshold: %.2f)\n", before, len(audioSlices), cfg.SimilarityThreshold)
	}
	fmt.Println()

	_, additional := mpc.CountSlicesForProgram(len(audioSlices))
	if additional > 0 {
		fmt.Printf("  Note: %d slices detected, will create %d programs (up to %d pads each, 8 banks)\n",
			len(audioSlices), 1+additional, mpc.MaxPadsPerProg)
		fmt.Println()
	}

	// Step 4: Create output directory
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("creating output directory %s: %w", outputDir, err)
	}

	// Step 5: Precompute custom note map for O(1) lookup per slice
	customNoteMap := parseNoteMapConfig(cfg.NoteMap)

	// Step 5b: Write transient report if requested
	if cfg.ReportFormat != "" {
		r := report.Build(cfg.InputPath, wavData.SampleRate, reportSensitivity, audioSlices)
		reportPath := filepath.Join(outputDir, programName+"_report."+cfg.ReportFormat)
		switch cfg.ReportFormat {
		case "json":
			if err := report.WriteJSON(reportPath, r); err != nil {
				return err
			}
		case "csv":
			if err := report.WriteCSV(reportPath, r); err != nil {
				return err
			}
		default:
			return fmt.Errorf("unsupported report format: %s (use json or csv)", cfg.ReportFormat)
		}
		fmt.Printf("  Report: %s\n", reportPath)
	}

	// Step 6: Generate programs based on format
	outputFormat := cfg.Format
	if outputFormat == "" {
		outputFormat = config.FormatMPC
	}

	switch outputFormat {
	case config.FormatMPC, config.FormatXPM:
		err = generateXPMPrograms(audioSlices, samples, programName, outputDir, wavData, cfg, customNoteMap)
	case config.FormatKRZ:
		err = generateKRZPrograms(audioSlices, samples, programName, outputDir, cfg.RootNote, wavData, cfg, customNoteMap)
	case config.FormatBoth:
		err = generateXPMPrograms(audioSlices, samples, programName, outputDir, wavData, cfg, customNoteMap)
		if err == nil {
			err = generateKRZPrograms(audioSlices, samples, programName, outputDir, cfg.RootNote, wavData, cfg, customNoteMap)
		}
	default:
		return fmt.Errorf("unsupported output format: %s", outputFormat)
	}

	if err != nil {
		return err
	}

	elapsed := time.Since(startTime)
	fmt.Printf("\nComplete! Total time: %.2fs\n", elapsed.Seconds())
	if outputFormat != config.FormatKRZ {
		fmt.Println("To use in your MPC:")
		fmt.Println(" 1. Copy the output folder to your MPC's storage")
		fmt.Println(" 2. Load the .xpm program file in your MPC software")
	}

	return nil
}

// generateKRZPrograms creates KRZ program files and compressed samples
func generateKRZPrograms(audioSlices []slice.AudioSlice, samples []float64, programName, outputDir string, rootNote int, wavData *input.WAVFile, cfg *config.Config, customNoteMap map[int]int) error {
	fmt.Printf("Generating KRZ program(s) in: %s\n", outputDir)

	krzSlices := make([]krz.SliceData, len(audioSlices))
	for i, s := range audioSlices {
		sliceSamples := slice.ExtractSamples(samples, s)
		if cfg.AutoTrim {
			sliceSamples = slice.TrimSilence(sliceSamples, cfg.TrimNoiseFloor)
		}
		if cfg.Normalize {
			sliceSamples = slice.NormalizeSlice(sliceSamples)
		}
		int16Samples := floatToInt16(sliceSamples)

		startTime := float64(s.StartSample) / float64(wavData.SampleRate)
		endTime := float64(s.EndSample) / float64(wavData.SampleRate)

		note := resolveNote(i, s.Note, cfg, customNoteMap)

		krzSlices[i] = krz.SliceData{
			Samples: int16Samples,
			Note:    note,
			Start:   startTime,
			End:     endTime,
		}
	}

	voiceMode := krz.VoiceModeDrum
	if cfg.KRZ.VoiceMode == config.VoiceModePoly {
		voiceMode = krz.VoiceModePoly
	}

	envelope := krz.DefaultDrumEnvelope()
	if voiceMode == krz.VoiceModePoly {
		envelope = krz.DefaultPolyEnvelope()
	}
	if !cfg.KRZ.Envelope.IsEmpty() {
		envelope = cfg.KRZ.Envelope
	}

	krzVersion := cfg.KRZ.Version
	if krzVersion == 0 {
		krzVersion = 2000
	}

	if len(krzSlices) > 128 {
		slog.Warn("more than 128 slices, truncating to KRZ limit", "count", len(krzSlices), "limit", 128)
		krzSlices = krzSlices[:128]
	}

	krzData, err := krz.CreateFromSlices(krzSlices,
		krz.WithFileName(programName),
		krz.WithVersion(krzVersion),
		krz.WithCompression(cfg.KRZ.Compress),
		krz.WithVoiceMode(voiceMode),
		krz.WithPriority(cfg.KRZ.Priority),
		krz.WithStereo(cfg.KRZ.Stereo),
		krz.WithEnvelope(envelope),
	)
	if err != nil {
		return fmt.Errorf("creating KRZ file for %q: %w", programName, err)
	}

	krzPath := filepath.Join(outputDir, programName+".krz")
	if err := os.WriteFile(krzPath, krzData, 0644); err != nil {
		return fmt.Errorf("writing KRZ file %s: %w", krzPath, err)
	}

	fmt.Printf("  Created KRZ program: %s\n", krzPath)
	fmt.Printf("  Total parts: %d\n", len(krzSlices))
	fmt.Printf("  Sample rate: %d Hz\n", wavData.SampleRate)
	fmt.Printf("  Root note: %d (%s)\n", rootNote, midi.MIDIToNote(rootNote))
	fmt.Printf("  Voice mode: %s\n", voiceModeString(voiceMode))
	compression := "none"
	if cfg.KRZ.Compress {
		compression = "ADPCM"
	}
	fmt.Printf("  Compression: %s\n", compression)

	return nil
}

// generateXPMPrograms creates an Akai MPC-compatible .xpm XML program with
// individual WAV files written to outputDir/Samples/.
func generateXPMPrograms(audioSlices []slice.AudioSlice, samples []float64, programName, outputDir string, wavData *input.WAVFile, cfg *config.Config, customNoteMap map[int]int) error {
	fmt.Printf("Generating XPM program in: %s\n", outputDir)

	remappedSlices := make([]slice.AudioSlice, len(audioSlices))
	for i, s := range audioSlices {
		remappedSlices[i] = s
		remappedSlices[i].Note = midi.MIDIToNote(resolveNote(i, s.Note, cfg, customNoteMap))
	}

	sliceSamples := make([][]int16, len(audioSlices))
	for i, s := range audioSlices {
		raw := slice.ExtractSamples(samples, s)
		if raw == nil {
			sliceSamples[i] = []int16{}
			continue
		}
		if cfg.AutoTrim {
			raw = slice.TrimSilence(raw, cfg.TrimNoiseFloor)
		}
		if cfg.Normalize {
			raw = slice.NormalizeSlice(raw)
		}
		sliceSamples[i] = floatToInt16(raw)
	}

	prog, err := mpc.GenerateXPMProgram(remappedSlices, programName, outputDir, sliceSamples, wavData.SampleRate)
	if err != nil {
		return fmt.Errorf("generating XPM program: %w", err)
	}

	fmt.Printf("  Created XPM program: %s\n", filepath.Join(outputDir, programName+".xpm"))
	fmt.Printf("  WAV files written to: %s\n", filepath.Join(outputDir, "Samples"))
	fmt.Printf("  Total pads: %d\n", len(prog.Pads))
	fmt.Println()
	return nil
}

// effectiveWindowMs returns the window size that will actually be used (substituting the default).
func effectiveWindowMs(configured int) int {
	if configured <= 0 {
		return 10
	}
	return configured
}

// effectiveBeatsPerBar returns the beats-per-bar that will actually be used (substituting the default).
func effectiveBeatsPerBar(configured int) int {
	if configured < 1 {
		return 4
	}
	return configured
}

// voiceModeString returns a human-readable name for a VoiceMode
func voiceModeString(m krz.VoiceMode) string {
	switch m {
	case krz.VoiceModeDrum:
		return "drum (mono)"
	case krz.VoiceModePoly:
		return "poly (multi-note)"
	default:
		return "unknown"
	}
}

// parseNoteMapConfig parses "index=note" entries into a precomputed map[sliceIndex]midiNote.
// Malformed entries are skipped with a warning. Notes outside [0, 127] are clamped to that range.
func parseNoteMapConfig(noteMap []string) map[int]int {
	if len(noteMap) == 0 {
		return nil
	}
	result := make(map[int]int, len(noteMap))
	for _, entry := range noteMap {
		var idx, note int
		if _, err := fmt.Sscanf(entry, "%d=%d", &idx, &note); err != nil {
			slog.Warn("invalid note-map entry, skipping", "entry", entry)
			continue
		}
		if note < 0 || note > 127 {
			slog.Warn("note-map note out of range, clamping to [0,127]", "entry", entry, "note", note)
			if note < 0 {
				note = 0
			} else {
				note = 127
			}
		}
		result[idx] = note
	}
	return result
}

// buildProgramName combines the sanitized user-supplied program name with the
// source WAV file's base name (without extension). If the user name is empty
// after sanitization it falls back to "SampleSlice".
// Example: ("MyKit", "/path/to/drums.wav") → "MyKit_drums"
func buildProgramName(userProgramName, inputPath string) string {
	name := mpc.SanitizeProgramName(userProgramName)
	if name == "" {
		name = "SampleSlice"
	}
	base := filepath.Base(inputPath)
	ext := filepath.Ext(base)
	wavBase := mpc.SanitizeProgramName(base[:len(base)-len(ext)])
	if wavBase != "" {
		name = name + "_" + wavBase
	}
	return name
}

// resolveNote determines the MIDI note for a given slice using the precomputed note map.
// Modes: custom map (--note-map), GM drum map (--gm-map), or sequential (default).
func resolveNote(sliceIndex int, sliceNote string, cfg *config.Config, customNoteMap map[int]int) int {
	if len(customNoteMap) > 0 {
		if note, ok := customNoteMap[sliceIndex]; ok {
			return note
		}
	}

	if cfg.GMMap {
		if sliceIndex < len(midi.GMDrumNotes) {
			return midi.GMDrumNotes[sliceIndex]
		}
		return midi.GMDrumNotes[len(midi.GMDrumNotes)-1] + (sliceIndex - len(midi.GMDrumNotes))
	}

	return slice.NoteToMIDINote(sliceNote)
}
