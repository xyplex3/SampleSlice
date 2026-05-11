// Package config provides configuration structures and validation
// for the SampleSlice application.
package config

import (
	"fmt"
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

// Envelope holds ADSR envelope values for a KRZ layer.
type Envelope struct {
	Attack  uint8
	Decay1  uint8
	Level1  uint8
	Decay2  uint8
	Level2  uint8
	Decay3  uint8
	Level3  uint8
	Sustain uint8
	Release uint8
}

// IsEmpty reports whether all envelope fields are zero (the default unset state).
func (e *Envelope) IsEmpty() bool {
	return e.Attack == 0 && e.Decay1 == 0 && e.Level1 == 0 &&
		e.Decay2 == 0 && e.Level2 == 0 && e.Decay3 == 0 &&
		e.Level3 == 0 && e.Sustain == 0 && e.Release == 0
}

// EnvelopePresets defines standard envelope shapes
var EnvelopePresets = map[string]Envelope{
	"drum": {
		Attack:  0,
		Decay1:  200,
		Level1:  0,
		Decay2:  0,
		Level2:  0,
		Decay3:  0,
		Level3:  0,
		Sustain: 0,
		Release: 0,
	},
	"perc": {
		Attack:  0,
		Decay1:  100,
		Level1:  0,
		Decay2:  0,
		Level2:  0,
		Decay3:  0,
		Level3:  0,
		Sustain: 0,
		Release: 0,
	},
	"pad": {
		Attack:  100,
		Decay1:  150,
		Level1:  200,
		Decay2:  100,
		Level2:  180,
		Decay3:  50,
		Level3:  160,
		Sustain: 200,
		Release: 100,
	},
	"key": {
		Attack:  0,
		Decay1:  80,
		Level1:  180,
		Decay2:  50,
		Level2:  160,
		Decay3:  30,
		Level3:  140,
		Sustain: 180,
		Release: 50,
	},
}

// GetEnvelopePreset returns the named envelope preset and reports whether
// it exists. Valid names are "drum", "perc", "pad", and "key".
func GetEnvelopePreset(name string) (Envelope, bool) {
	e, ok := EnvelopePresets[name]
	return e, ok
}

// ParseEnvelope parses a comma-separated string of 9 uint8 values into an Envelope.
// Format: "attack,decay1,level1,decay2,level2,decay3,level3,sustain,release" (values 0-255)
func ParseEnvelope(s string) (Envelope, error) {
	parts := strings.Split(s, ",")
	if len(parts) != 9 {
		return Envelope{}, fmt.Errorf("expected 9 comma-separated values (attack,decay1,level1,decay2,level2,decay3,level3,sustain,release), got %d", len(parts))
	}
	vals := make([]uint8, 9)
	for i, p := range parts {
		p = strings.TrimSpace(p)
		var n int
		if _, err := fmt.Sscanf(p, "%d", &n); err != nil {
			return Envelope{}, fmt.Errorf("value %d (%q) is not a valid integer: %w", i+1, p, err)
		}
		if n < 0 || n > 255 {
			return Envelope{}, fmt.Errorf("value %d (%d) out of range 0-255", i+1, n)
		}
		vals[i] = uint8(n)
	}
	return Envelope{
		Attack:  vals[0],
		Decay1:  vals[1],
		Level1:  vals[2],
		Decay2:  vals[3],
		Level2:  vals[4],
		Decay3:  vals[5],
		Level3:  vals[6],
		Sustain: vals[7],
		Release: vals[8],
	}, nil
}

// DetectionConfig groups transient detection parameters.
type DetectionConfig struct {
	Sensitivity  float64 // 0.0 (sensitive) to 1.0 (strict)
	MinInterval  int     // minimum ms between transients
	WindowSizeMs int     // energy window size in ms; 0 uses the 10ms default
}

// KRZConfig groups Kurzweil KRZ-specific output parameters.
type KRZConfig struct {
	Version        uint16      // KRZ file format version (default: 2000)
	Compress       bool        // Use ADPCM compression for KRZ samples
	VoiceMode      VoiceMode   // drum or poly
	Priority       uint8       // Voice priority 1-8
	Stereo         bool        // Enable stereo voice mode (for poly patches)
	Envelope       Envelope    // Custom ADSR envelope
	EnvelopePreset string      // Envelope preset name (drum, perc, pad, key)
	Layers         []LayerConfig
}

// Config holds all configuration options for SampleSlice.
type Config struct {
	// Input/Output
	InputPath   string
	OutputDir   string
	ProgramName string
	Format      OutputFormat

	// Slicing / output settings
	RootNote            int
	PrePadding          int
	PostPadding         int
	GMMap               bool
	NoteMap             []string
	SimilarityThreshold float64 // 0=disabled; 0.95=remove near-duplicates

	// Post-extraction processing
	Normalize      bool
	AutoTrim       bool
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
