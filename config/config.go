// Package config provides configuration structures and validation
// for the SampleSlice application.
package config

import (
	"fmt"
	"strconv"
	"strings"
)

// OutputFormat specifies the output file format produced by the application.
type OutputFormat string

const (
	FormatMPC  OutputFormat = "mpc"  // Akai MPC .xpm XML program file
	FormatKRZ  OutputFormat = "krz"  // Kurzweil KRZ sample library file
	FormatBoth OutputFormat = "both" // Both MPC and KRZ outputs
	FormatXPM  OutputFormat = "xpm"  // Akai MPC .xpm XML program file
)

// VoiceMode specifies how a KRZ voice is triggered and pitched.
type VoiceMode string

const (
	VoiceModeDrum VoiceMode = "drum" // Fixed-pitch mono voice, one hit per note
	VoiceModePoly VoiceMode = "poly" // Multi-note polyphonic via sample rate modulation
)

// LayerConfig defines per-layer settings for multi-layer KRZ programs.
type LayerConfig struct {
	Name     string   // Layer name (for identification)
	Envelope Envelope // Per-layer envelope
	Priority uint8    // Per-layer priority (1-8)
	LoVel    uint8    // Minimum velocity (0-127)
	HiVel    uint8    // Maximum velocity (0-127)
	Pan      uint8    // Pan position (0=left, 64=center, 127=right)
	Output   uint8    // Output routing (1-8)
}

// Envelope holds a KRZ amplitude envelope shape, matching the real K2000
// AMPENV segment's own layout field-for-field (confirmed against Geoffrey
// Mayer's Kurzweil K2000/K2500/K2600 file format reference): three attack
// stages, one decay stage, and three release stages. Level fields are
// unsigned percentages (0-100); time fields are raw AMPENV time-table
// values (0 = instant, ~0.02s per step through the middle of the range,
// tapering non-linearly toward 60s at the top of the byte range). There is
// no separate Rel3 level — the envelope simply reaches silence by the end
// of Rel3Time.
type Envelope struct {
	Att1Level uint8 // Level reached at the end of attack stage 1 (0-100%)
	Att1Time  uint8 // Time for attack stage 1
	Att2Level uint8 // Level reached at the end of attack stage 2 (0-100%)
	Att2Time  uint8 // Time for attack stage 2
	Att3Level uint8 // Level reached at the end of attack stage 3 (0-100%)
	Att3Time  uint8 // Time for attack stage 3
	Dec1Level uint8 // Level reached at the end of the decay stage (0-100%)
	Dec1Time  uint8 // Time for the decay stage
	Rel1Level uint8 // Level reached at the end of release stage 1 (0-100%)
	Rel1Time  uint8 // Time for release stage 1
	Rel2Level uint8 // Level reached at the end of release stage 2 (0-100%)
	Rel2Time  uint8 // Time for release stage 2
	Rel3Time  uint8 // Time for the final release stage, down to silence
}

// IsEmpty reports whether all envelope fields are zero (the default unset
// state).
func (e *Envelope) IsEmpty() bool {
	return *e == Envelope{}
}

// EnvelopePresets defines standard envelope shapes, keyed by preset name.
// Valid keys are "drum", "perc", "pad", and "key"; see [GetEnvelopePreset].
var EnvelopePresets = map[string]Envelope{
	// Instant full-level attack, quick decay to silence: one-shot drum hits.
	"drum": {
		Att1Level: 100, Att1Time: 0,
		Dec1Level: 0, Dec1Time: 10,
	},
	// Like "drum" but with a slightly longer decay tail.
	"perc": {
		Att1Level: 100, Att1Time: 0,
		Dec1Level: 0, Dec1Time: 20,
	},
	// Slow attack, sustained body, long release: pads and sustained tones.
	"pad": {
		Att1Level: 100, Att1Time: 100,
		Dec1Level: 80, Dec1Time: 50,
		Rel1Level: 0, Rel1Time: 100,
	},
	// Moderate attack and decay: keyboard/mallet-style hits.
	"key": {
		Att1Level: 100, Att1Time: 10,
		Dec1Level: 70, Dec1Time: 30,
		Rel1Level: 0, Rel1Time: 40,
	},
}

// GetEnvelopePreset returns the named envelope preset and reports whether
// it exists. Valid names are "drum", "perc", "pad", and "key".
func GetEnvelopePreset(name string) (Envelope, bool) {
	e, ok := EnvelopePresets[name]
	return e, ok
}

// ParseEnvelope parses a comma-separated string of 13 uint8 values into an
// Envelope. Format: "att1level,att1time,att2level,att2time,att3level,
// att3time,dec1level,dec1time,rel1level,rel1time,rel2level,rel2time,
// rel3time" (values 0-255).
func ParseEnvelope(s string) (Envelope, error) {
	const fieldList = "att1level,att1time,att2level,att2time,att3level,att3time,dec1level,dec1time,rel1level,rel1time,rel2level,rel2time,rel3time"
	parts := strings.Split(s, ",")
	if len(parts) != 13 {
		return Envelope{}, fmt.Errorf("expected 13 comma-separated values (%s), got %d", fieldList, len(parts))
	}
	vals := make([]uint8, 13)
	for i, p := range parts {
		p = strings.TrimSpace(p)
		n, err := strconv.Atoi(p)
		if err != nil {
			return Envelope{}, fmt.Errorf("value %d (%q) is not a valid integer: %w", i+1, p, err)
		}
		if n < 0 || n > 255 {
			return Envelope{}, fmt.Errorf("value %d (%d) out of range 0-255", i+1, n)
		}
		vals[i] = uint8(n)
	}
	return Envelope{
		Att1Level: vals[0], Att1Time: vals[1],
		Att2Level: vals[2], Att2Time: vals[3],
		Att3Level: vals[4], Att3Time: vals[5],
		Dec1Level: vals[6], Dec1Time: vals[7],
		Rel1Level: vals[8], Rel1Time: vals[9],
		Rel2Level: vals[10], Rel2Time: vals[11],
		Rel3Time: vals[12],
	}, nil
}

// DetectionConfig groups transient detection parameters.
type DetectionConfig struct {
	Sensitivity  float64 // 0.0 (sensitive) to 1.0 (strict)
	MinInterval  int     // minimum ms between transients
	WindowSizeMs int     // energy window size in ms; 0 uses the 10ms default
}

// KRZConfig groups Kurzweil KRZ-specific output parameters.
//
// Envelope is wired into the generated Program's amplitude envelope (see
// package krz). VoiceMode, Priority, Stereo, and Compress are still
// accepted for CLI/API compatibility but do not yet change the generated
// KRZ bytes — see the Envelope doc comment in package krz.
type KRZConfig struct {
	Version        uint16        // KRZ file format version (default: 2000)
	Compress       bool          // Reserved; not yet wired to KRZ output
	VoiceMode      VoiceMode     // drum or poly
	Priority       uint8         // Voice priority 1-8
	Stereo         bool          // Enable stereo voice mode (for poly patches)
	Envelope       Envelope      // Custom ADSR envelope
	EnvelopePreset string        // Envelope preset name (drum, perc, pad, key)
	VASTFrom       string        // Path to an existing KRZ file to borrow a VAST tone from
	VASTProgram    string        // Name of the program within VASTFrom to borrow
	Layers         []LayerConfig // Per-layer overrides for multi-layer KRZ programs
}

// Config holds all configuration options for SampleSlice.
type Config struct {
	// Input/Output
	InputPath   string       // Path to the source WAV file
	OutputDir   string       // Destination directory for generated output
	ProgramName string       // User-supplied program name (combined with the input file's base name)
	Format      OutputFormat // Output format: mpc, krz, both, or xpm

	// Slicing / output settings
	RootNote            int      // Root MIDI note (0-127) assigned to the first slice
	PrePadding          int      // Pre-transient padding, in milliseconds
	PostPadding         int      // Post-transient padding, in milliseconds
	GMMap               bool     // Use the General MIDI drum note layout instead of sequential notes
	NoteMap             []string // Custom "index=note" assignments, e.g. "0=36,1=42"
	SimilarityThreshold float64  // 0=disabled; 0.95=remove near-duplicates

	// Post-extraction processing
	Normalize      bool    // Peak-normalize each slice to 0 dBFS
	AutoTrim       bool    // Strip leading/trailing silence from each slice
	TrimNoiseFloor float64 // linear amplitude, default 0.001 (~-60 dBFS)

	// Tempo-grid slicing (Feature 9)
	BPM         float64 // 0 = disabled; use transient detection
	LoopBars    int     // bars per grid slice (default 1)
	BeatsPerBar int     // time signature numerator (default 4)

	// Report
	ReportFormat string // "", "json", or "csv"

	// Sub-configs
	Detection DetectionConfig
	KRZ       KRZConfig
}

// Validate checks that the configuration is valid for processing.
// Returns an error if required fields are missing or values are out of range.
func (c *Config) Validate() error {
	if c.InputPath == "" {
		return fmt.Errorf("input path is required")
	}

	if c.Detection.Sensitivity < 0.0 || c.Detection.Sensitivity > 1.0 {
		return fmt.Errorf("sensitivity must be between 0.0 and 1.0")
	}

	if c.Detection.MinInterval < 0 {
		return fmt.Errorf("min-interval must be non-negative")
	}

	if c.Detection.WindowSizeMs < 0 {
		return fmt.Errorf("window-size must be non-negative")
	}

	if c.PrePadding < 0 {
		return fmt.Errorf("pre-padding must be non-negative")
	}

	if c.PostPadding < 0 {
		return fmt.Errorf("post-padding must be non-negative")
	}

	if c.BPM < 0 {
		return fmt.Errorf("bpm must be non-negative (0 disables tempo-grid mode)")
	}

	if c.RootNote < 0 || c.RootNote > 127 {
		return fmt.Errorf("root note must be a valid MIDI note (0-127)")
	}

	if c.KRZ.Priority != 0 && (c.KRZ.Priority < 1 || c.KRZ.Priority > 8) {
		return fmt.Errorf("priority must be between 1 and 8")
	}

	if c.KRZ.VoiceMode != VoiceModeDrum && c.KRZ.VoiceMode != VoiceModePoly && c.KRZ.VoiceMode != "" {
		return fmt.Errorf("voice mode must be 'drum' or 'poly'")
	}

	if (c.KRZ.VASTFrom == "") != (c.KRZ.VASTProgram == "") {
		return fmt.Errorf("--vast-from and --vast-program must be used together")
	}

	for i, layer := range c.KRZ.Layers {
		if layer.Priority != 0 && (layer.Priority < 1 || layer.Priority > 8) {
			return fmt.Errorf("layer %d: priority must be between 1 and 8", i)
		}
		if layer.LoVel > layer.HiVel {
			return fmt.Errorf("layer %d: LoVel (%d) must be <= HiVel (%d)", i, layer.LoVel, layer.HiVel)
		}
		if layer.Output != 0 && (layer.Output < 1 || layer.Output > 8) {
			return fmt.Errorf("layer %d: output must be between 1 and 8", i)
		}
	}

	return nil
}
