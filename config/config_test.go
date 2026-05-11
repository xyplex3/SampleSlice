package config_test

import (
	"testing"

	"sampleslice/config"
)

// TestConfigValidate_EmptyConfig verifies that an empty config fails validation.
func TestConfigValidate_EmptyConfig(t *testing.T) {
	cfg := config.Config{}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for empty config, got nil")
	}
}

// TestConfigValidate_ValidConfig verifies that a fully populated config passes validation.
func TestConfigValidate_ValidConfig(t *testing.T) {
	cfg := config.Config{
		InputPath:   "./input.wav",
		OutputDir:   "./output",
		ProgramName: "TestDrums",
		RootNote:    36,
		PrePadding:  10,
		PostPadding: 20,
		Detection: config.DetectionConfig{
			Sensitivity: 0.5,
			MinInterval: 50,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

// TestConfigValidate_SensitivityBounds verifies sensitivity must be in [0.0, 1.0].
func TestConfigValidate_SensitivityBounds(t *testing.T) {
	tests := []struct {
		name        string
		sensitivity float64
		wantErr     bool
	}{
		{"negative sensitivity", -0.1, true},
		{"zero sensitivity", 0.0, false},
		{"mid sensitivity", 0.5, false},
		{"max sensitivity", 1.0, false},
		{"above max sensitivity", 1.1, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				InputPath: "./test.wav",
				Detection: config.DetectionConfig{Sensitivity: tt.sensitivity},
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("expected error for sensitivity %f, got nil", tt.sensitivity)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for sensitivity %f: %v", tt.sensitivity, err)
			}
		})
	}
}

// TestConfigValidate_MinInterval verifies min-interval must be non-negative.
func TestConfigValidate_MinInterval(t *testing.T) {
	cfg := config.Config{
		InputPath: "./test.wav",
		Detection: config.DetectionConfig{
			Sensitivity: 0.5,
			MinInterval: -1,
		},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for negative MinInterval, got nil")
	}
}

// TestConfigValidate_Padding verifies padding values must be non-negative.
func TestConfigValidate_Padding(t *testing.T) {
	tests := []struct {
		name        string
		prePadding  int
		postPadding int
		wantErr     bool
	}{
		{"negative pre-padding", -1, 0, true},
		{"negative post-padding", 0, -1, true},
		{"zero padding", 0, 0, false},
		{"positive padding", 10, 20, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				InputPath:   "./test.wav",
				PrePadding:  tt.prePadding,
				PostPadding: tt.postPadding,
				Detection:   config.DetectionConfig{Sensitivity: 0.5},
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("expected error for pre=%d post=%d, got nil", tt.prePadding, tt.postPadding)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for pre=%d post=%d: %v", tt.prePadding, tt.postPadding, err)
			}
		})
	}
}

// TestConfigValidate_RootNote verifies root note must be a valid MIDI note (0-127).
func TestConfigValidate_RootNote(t *testing.T) {
	tests := []struct {
		name     string
		rootNote int
		wantErr  bool
	}{
		{"negative note", -1, true},
		{"note zero", 0, false},
		{"mid note", 60, false},
		{"max note", 127, false},
		{"above max note", 128, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.Config{
				InputPath: "./test.wav",
				RootNote:  tt.rootNote,
				Detection: config.DetectionConfig{Sensitivity: 0.5},
			}
			err := cfg.Validate()
			if tt.wantErr && err == nil {
				t.Errorf("expected error for root note %d, got nil", tt.rootNote)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("unexpected error for root note %d: %v", tt.rootNote, err)
			}
		})
	}
}

// TestConfigValidate_InputPathRequired verifies that input path is required.
func TestConfigValidate_InputPathRequired(t *testing.T) {
	cfg := config.Config{
		Detection: config.DetectionConfig{Sensitivity: 0.5},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for missing input path, got nil")
	}
}

// TestConfigValidate_AllFieldsValid tests a config with all fields set to valid values.
func TestConfigValidate_AllFieldsValid(t *testing.T) {
	cfg := config.Config{
		InputPath:   "/path/to/input.wav",
		OutputDir:   "/path/to/output",
		ProgramName: "MyDrums",
		RootNote:    48,
		PrePadding:  50,
		PostPadding: 75,
		Detection: config.DetectionConfig{
			Sensitivity: 0.75,
			MinInterval: 100,
		},
	}

	err := cfg.Validate()
	if err != nil {
		t.Fatalf("expected valid config, got error: %v", err)
	}
}

// TestConfigValidate_BoundarySensitivity tests exact boundary values for sensitivity.
func TestConfigValidate_BoundarySensitivity(t *testing.T) {
	cfg := config.Config{
		InputPath: "./test.wav",
		Detection: config.DetectionConfig{Sensitivity: 0.0},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("sensitivity 0.0 should be valid, got error: %v", err)
	}

	cfg.Detection.Sensitivity = 1.0
	if err := cfg.Validate(); err != nil {
		t.Errorf("sensitivity 1.0 should be valid, got error: %v", err)
	}
}

// TestConfigValidate_BoundaryRootNote tests exact boundary values for root note.
func TestConfigValidate_BoundaryRootNote(t *testing.T) {
	cfg := config.Config{
		InputPath: "./test.wav",
		RootNote:  0,
		Detection: config.DetectionConfig{Sensitivity: 0.5},
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("root note 0 should be valid, got error: %v", err)
	}

	cfg.RootNote = 127
	if err := cfg.Validate(); err != nil {
		t.Errorf("root note 127 should be valid, got error: %v", err)
	}
}

// TestGetEnvelopePreset verifies preset lookup returns correct ok flag.
func TestGetEnvelopePreset(t *testing.T) {
	tests := []struct {
		name   string
		wantOK bool
	}{
		{"drum", true},
		{"perc", true},
		{"pad", true},
		{"key", true},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, ok := config.GetEnvelopePreset(tt.name)
			if ok != tt.wantOK {
				t.Errorf("GetEnvelopePreset(%q) ok = %v, want %v", tt.name, ok, tt.wantOK)
			}
		})
	}
}

// TestGetEnvelopePreset_DrumValues verifies the drum preset has expected field values.
func TestGetEnvelopePreset_DrumValues(t *testing.T) {
	e, ok := config.GetEnvelopePreset("drum")
	if !ok {
		t.Fatal("drum preset not found")
	}
	if e.Attack != 0 {
		t.Errorf("drum Attack = %d, want 0", e.Attack)
	}
	if e.Decay1 != 200 {
		t.Errorf("drum Decay1 = %d, want 200", e.Decay1)
	}
}

// TestParseEnvelope verifies parsing of comma-separated envelope strings.
func TestParseEnvelope(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    config.Envelope
		wantErr bool
	}{
		{
			name:  "all zeros",
			input: "0,0,0,0,0,0,0,0,0",
			want:  config.Envelope{},
		},
		{
			name:  "all max values",
			input: "255,255,255,255,255,255,255,255,255",
			want: config.Envelope{
				Attack: 255, Decay1: 255, Level1: 255,
				Decay2: 255, Level2: 255, Decay3: 255,
				Level3: 255, Sustain: 255, Release: 255,
			},
		},
		{
			name:  "drum-like values",
			input: "0,200,0,0,0,0,0,0,0",
			want:  config.Envelope{Decay1: 200},
		},
		{
			name:  "whitespace trimmed",
			input: "0, 150, 0, 0, 0, 0, 0, 0, 30",
			want:  config.Envelope{Decay1: 150, Release: 30},
		},
		{
			name:    "too few values",
			input:   "1,2,3",
			wantErr: true,
		},
		{
			name:    "too many values",
			input:   "1,2,3,4,5,6,7,8,9,10",
			wantErr: true,
		},
		{
			name:    "value below zero",
			input:   "-1,0,0,0,0,0,0,0,0",
			wantErr: true,
		},
		{
			name:    "value above 255",
			input:   "256,0,0,0,0,0,0,0,0",
			wantErr: true,
		},
		{
			name:    "non-numeric value",
			input:   "abc,0,0,0,0,0,0,0,0",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := config.ParseEnvelope(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseEnvelope(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
				return
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseEnvelope(%q) = %+v, want %+v", tt.input, got, tt.want)
			}
		})
	}
}

// TestEnvelopeIsEmpty verifies that IsEmpty detects all-zero envelopes.
func TestEnvelopeIsEmpty(t *testing.T) {
	tests := []struct {
		name string
		env  config.Envelope
		want bool
	}{
		{"all zeros", config.Envelope{}, true},
		{"attack non-zero", config.Envelope{Attack: 1}, false},
		{"release non-zero", config.Envelope{Release: 1}, false},
		{"sustain non-zero", config.Envelope{Sustain: 128}, false},
		{"all fields set", config.Envelope{
			Attack: 1, Decay1: 1, Level1: 1,
			Decay2: 1, Level2: 1, Decay3: 1,
			Level3: 1, Sustain: 1, Release: 1,
		}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.env.IsEmpty()
			if got != tt.want {
				t.Errorf("IsEmpty() = %v, want %v for %+v", got, tt.want, tt.env)
			}
		})
	}
}
