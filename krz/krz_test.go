// Package krz tests cover KRZ file creation, serialization,
// envelope defaults, functional options, and MIDI note validation.
package krz

import (
	"bytes"
	"encoding/binary"
	"testing"
)

// -----------------------------------------------------------------------
// Tests for DefaultDrumEnvelope
// -----------------------------------------------------------------------

func TestDefaultDrumEnvelope(t *testing.T) {
	e := DefaultDrumEnvelope()

	expected := Envelope{
		Attack:  0,
		Decay1:  20,
		Level1:  70,
		Decay2:  30,
		Level2:  0,
		Decay3:  0,
		Level3:  0,
		Sustain: 0,
		Release: 5,
	}

	if e != expected {
		t.Errorf("DefaultDrumEnvelope() = %+v, want %+v", e, expected)
	}
}

// -----------------------------------------------------------------------
// Tests for DefaultPolyEnvelope
// -----------------------------------------------------------------------

func TestDefaultPolyEnvelope(t *testing.T) {
	e := DefaultPolyEnvelope()

	expected := Envelope{
		Attack:  5,
		Decay1:  40,
		Level1:  70,
		Decay2:  60,
		Level2:  30,
		Decay3:  80,
		Level3:  0,
		Sustain: 0,
		Release: 40,
	}

	if e != expected {
		t.Errorf("DefaultPolyEnvelope() = %+v, want %+v", e, expected)
	}
}

// -----------------------------------------------------------------------
// Tests for NewKRZFile
// -----------------------------------------------------------------------

func TestNewKRZFile(t *testing.T) {
	tests := []struct {
		name        string
		version     uint16
		wantVersion uint16
		wantModels  []uint16
	}{
		{
			name:        "version 1",
			version:     0x0001,
			wantVersion: 0x0001,
			wantModels:  []uint16{0x0064},
		},
		{
			name:        "version 0",
			version:     0,
			wantVersion: 0,
			wantModels:  []uint16{0x0064},
		},
		{
			name:        "version max",
			version:     0xFFFF,
			wantVersion: 0xFFFF,
			wantModels:  []uint16{0x0064},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := NewKRZFile(tc.version)
			if f.Version != tc.wantVersion {
				t.Errorf("Version = %d, want %d", f.Version, tc.wantVersion)
			}
			if len(f.Models) != len(tc.wantModels) {
				t.Fatalf("Models length = %d, want %d", len(f.Models), len(tc.wantModels))
			}
			for i, m := range tc.wantModels {
				if f.Models[i] != m {
					t.Errorf("Models[%d] = 0x%04x, want 0x%04x", i, f.Models[i], m)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests for KRZFile.AddKeymap
// -----------------------------------------------------------------------

func TestKRZFile_AddKeymap(t *testing.T) {
	f := NewKRZFile(1)
	km := NewKeymap(1, "TestKeymap")

	f.AddKeymap(km)

	if len(f.Keymaps) != 1 {
		t.Fatalf("Keymaps length = %d, want 1", len(f.Keymaps))
	}
	if f.Keymaps[0] != km {
		t.Error("Added keymap does not match original reference")
	}

	// Add a second keymap
	km2 := NewKeymap(2, "SecondKeymap")
	f.AddKeymap(km2)

	if len(f.Keymaps) != 2 {
		t.Fatalf("Keymaps length = %d, want 2", len(f.Keymaps))
	}
}

// -----------------------------------------------------------------------
// Tests for KRZFile.AddProgram
// -----------------------------------------------------------------------

func TestKRZFile_AddProgram(t *testing.T) {
	f := NewKRZFile(1)
	prog := NewProgram(1, "TestProg", VoiceModeDrum, 1, false, DefaultDrumEnvelope())

	f.AddProgram(prog)

	if len(f.Programs) != 1 {
		t.Fatalf("Programs length = %d, want 1", len(f.Programs))
	}
	if f.Programs[0] != prog {
		t.Error("Added program does not match original reference")
	}
}

// -----------------------------------------------------------------------
// Tests for KRZFile.Serialize
// -----------------------------------------------------------------------

func TestKRZFile_Serialize(t *testing.T) {
	tests := []struct {
		name      string
		setupFile func() *KRZFile
		wantErr   bool
		checkData func(t *testing.T, data []byte)
	}{
		{
			name: "no keymaps returns error",
			setupFile: func() *KRZFile {
				f := NewKRZFile(1)
				f.AddProgram(NewProgram(1, "P", VoiceModeDrum, 1, false, DefaultDrumEnvelope()))
				return f
			},
			wantErr: true,
		},
		{
			name: "no programs returns error",
			setupFile: func() *KRZFile {
				f := NewKRZFile(1)
				f.AddKeymap(NewKeymap(1, "K"))
				return f
			},
			wantErr: true,
		},
		{
			name: "minimal valid file",
			setupFile: func() *KRZFile {
				f := NewKRZFile(1)
				km := NewKeymap(1, "TestKM")
				km.AddSample([]int16{128, 256, 0}, 60, 60, 60, 8, false)
				f.AddKeymap(km)

				prog := NewProgram(1, "TestP", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
				part := Part{
					Enabled:   true,
					PartID:    1,
					KeymapRef: km.Hash,
					RootNote:  60,
					Velocity:  100,
					Pan:       64,
					Output:    1,
					LoKey:     0,
					HiKey:     127,
					LoVel:     0,
					HiVel:     127,
				}
				prog.AddPart(part)
				f.AddProgram(prog)
				return f
			},
			wantErr: false,
			checkData: func(t *testing.T, data []byte) {
				// Verify magic bytes
				if len(data) < 4 {
					t.Fatal("data too short for magic bytes")
				}
				expectedMagic := []byte{0x4B, 0x52, 0x5A, 0x00}
				if !bytes.Equal(data[:4], expectedMagic) {
					t.Errorf("magic bytes = %v, want %v", data[:4], expectedMagic)
				}

				// Verify version
				if len(data) < 6 {
					t.Fatal("data too short for version")
				}
				version := binary.BigEndian.Uint16(data[4:6])
				if version != 1 {
					t.Errorf("version = %d, want 1", version)
				}
			},
		},
		{
			name: "multiple keymaps and programs",
			setupFile: func() *KRZFile {
				f := NewKRZFile(1)
				for i := uint16(1); i <= 3; i++ {
					km := NewKeymap(i, "KM")
					km.AddSample([]int16{100}, 60, 60, 60, 8, false)
					f.AddKeymap(km)
				}
				for i := uint16(1); i <= 2; i++ {
					prog := NewProgram(i, "P", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
					prog.AddPart(Part{Enabled: true, PartID: 1, KeymapRef: f.Keymaps[0].Hash})
					f.AddProgram(prog)
				}
				return f
			},
			wantErr: false,
			checkData: func(t *testing.T, data []byte) {
				if len(data) == 0 {
					t.Error("serialized data is empty")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := tc.setupFile()
			data, err := f.Serialize()
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tc.checkData != nil {
				tc.checkData(t, data)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests for validateMidiNote
// -----------------------------------------------------------------------

func TestValidateMidiNote(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected uint16
	}{
		{"zero", 0, 0},
		{"mid", 60, 60},
		{"max", 127, 127},
		{"below_zero", -1, 0},
		{"far_below", -100, 0},
		{"above_max", 128, 127},
		{"far_above", 200, 127},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := validateMidiNote(tc.input)
			if result != tc.expected {
				t.Errorf("validateMidiNote(%d) = %d, want %d", tc.input, result, tc.expected)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests for validateMidiNoteUint
// -----------------------------------------------------------------------

func TestValidateMidiNoteUint(t *testing.T) {
	tests := []struct {
		name     string
		input    uint16
		expected uint16
	}{
		{"zero", 0, 0},
		{"mid", 60, 60},
		{"max", 127, 127},
		{"above_max", 128, 127},
		{"far_above", 200, 127},
		{"max_uint16", 0xFFFF, 127},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := validateMidiNoteUint(tc.input)
			if result != tc.expected {
				t.Errorf("validateMidiNoteUint(%d) = %d, want %d", tc.input, result, tc.expected)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests for truncateName
// -----------------------------------------------------------------------

func TestTruncateName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short_name", "Hi", 16, "Hi"},
		{"exact_length", "1234567890123456", 16, "1234567890123456"},
		{"too_long", "12345678901234567890", 16, "1234567890123456"},
		{"empty", "", 16, ""},
		{"zero_maxLen", "Hi", 0, ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := truncateName(tc.input, tc.maxLen)
			if result != tc.expected {
				t.Errorf("truncateName(%q, %d) = %q, want %q", tc.input, tc.maxLen, result, tc.expected)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests for CreateFromSlices
// -----------------------------------------------------------------------

func TestCreateFromSlices(t *testing.T) {
	tests := []struct {
		name    string
		slices  []SliceData
		opts    []CreateOption
		wantErr bool
	}{
		{
			name:    "empty slices returns error",
			slices:  []SliceData{},
			opts:    nil,
			wantErr: true,
		},
		{
			name: "single slice default options",
			slices: []SliceData{
				{Samples: []int16{100, 200, 150}, Note: 36, Start: 0.0, End: 0.5},
			},
			opts:    nil,
			wantErr: false,
		},
		{
			name: "multiple slices",
			slices: []SliceData{
				{Samples: []int16{100}, Note: 35, Start: 0, End: 0.1},
				{Samples: []int16{200}, Note: 38, Start: 0.1, End: 0.2},
				{Samples: []int16{150}, Note: 42, Start: 0.2, End: 0.3},
			},
			opts:    nil,
			wantErr: false,
		},
		{
			name: "with custom filename",
			slices: []SliceData{
				{Samples: []int16{100}, Note: 36},
			},
			opts:    []CreateOption{WithFileName("MyDrums")},
			wantErr: false,
		},
		{
			name: "with poly voice mode",
			slices: []SliceData{
				{Samples: []int16{100}, Note: 60},
			},
			opts:    []CreateOption{WithVoiceMode(VoiceModePoly)},
			wantErr: false,
		},
		{
			name: "with compression",
			slices: []SliceData{
				{Samples: []int16{100, 200}, Note: 36},
			},
			opts:    []CreateOption{WithCompression(true)},
			wantErr: false,
		},
		{
			name: "with custom priority",
			slices: []SliceData{
				{Samples: []int16{100}, Note: 36},
			},
			opts:    []CreateOption{WithPriority(8)},
			wantErr: false,
		},
		{
			name: "with stereo",
			slices: []SliceData{
				{Samples: []int16{100}, Note: 36},
			},
			opts:    []CreateOption{WithStereo(true)},
			wantErr: false,
		},
		{
			name: "note zero defaults to 60",
			slices: []SliceData{
				{Samples: []int16{100}, Note: 0},
			},
			opts:    nil,
			wantErr: false,
		},
		{
			name: "with custom envelope",
			slices: []SliceData{
				{Samples: []int16{100}, Note: 36},
			},
			opts: []CreateOption{WithEnvelope(Envelope{
				Attack:  10,
				Decay1:  20,
				Level1:  128,
				Sustain: 200,
				Release: 30,
			})},
			wantErr: false,
		},
		{
			name: "with custom version",
			slices: []SliceData{
				{Samples: []int16{100}, Note: 36},
			},
			opts:    []CreateOption{WithVersion(0x0002)},
			wantErr: false,
		},
		{
			name: "large slice exceeding 64 KB",
			slices: func() []SliceData {
				// 40000 int16 samples ≈ 80 KB of raw audio — well over the old 64 KB limit.
				samples := make([]int16, 40000)
				for i := range samples {
					samples[i] = int16(i % 32767)
				}
				return []SliceData{{Samples: samples, Note: 36}}
			}(),
			opts:    nil,
			wantErr: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			data, err := CreateFromSlices(tc.slices, tc.opts...)
			if tc.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(data) == 0 {
				t.Error("serialized data is empty")
			}
			// Verify magic bytes present
			if len(data) >= 4 {
				expectedMagic := []byte{0x4B, 0x52, 0x5A, 0x00}
				if !bytes.Equal(data[:4], expectedMagic) {
					t.Errorf("magic bytes = %v, want %v", data[:4], expectedMagic)
				}
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests for CreateOption functional options
// -----------------------------------------------------------------------

func TestCreateOptions(t *testing.T) {
	tests := []struct {
		name  string
		option CreateOption
		check  func(*testing.T, createConfig)
	}{
		{
			name:  "WithFileName",
			option: WithFileName("CustomName"),
			check: func(t *testing.T, c createConfig) {
				if c.fileName != "CustomName" {
					t.Errorf("fileName = %q, want %q", c.fileName, "CustomName")
				}
			},
		},
		{
			name:  "WithVersion",
			option: WithVersion(0x0005),
			check: func(t *testing.T, c createConfig) {
				if c.version != 0x0005 {
					t.Errorf("version = 0x%04x, want 0x0005", c.version)
				}
			},
		},
		{
			name:  "WithCompression true",
			option: WithCompression(true),
			check: func(t *testing.T, c createConfig) {
				if !c.compress {
					t.Error("compress = false, want true")
				}
			},
		},
		{
			name:  "WithCompression false",
			option: WithCompression(false),
			check: func(t *testing.T, c createConfig) {
				if c.compress {
					t.Error("compress = true, want false")
				}
			},
		},
		{
			name:  "WithVoiceMode Poly",
			option: WithVoiceMode(VoiceModePoly),
			check: func(t *testing.T, c createConfig) {
				if c.voiceMode != VoiceModePoly {
					t.Errorf("voiceMode = %v, want %v", c.voiceMode, VoiceModePoly)
				}
			},
		},
		{
			name:  "WithVoiceMode Drum",
			option: WithVoiceMode(VoiceModeDrum),
			check: func(t *testing.T, c createConfig) {
				if c.voiceMode != VoiceModeDrum {
					t.Errorf("voiceMode = %v, want %v", c.voiceMode, VoiceModeDrum)
				}
			},
		},
		{
			name:  "WithPriority",
			option: WithPriority(5),
			check: func(t *testing.T, c createConfig) {
				if c.priority != 5 {
					t.Errorf("priority = %d, want 5", c.priority)
				}
			},
		},
		{
			name:  "WithStereo true",
			option: WithStereo(true),
			check: func(t *testing.T, c createConfig) {
				if !c.stereo {
					t.Error("stereo = false, want true")
				}
			},
		},
		{
			name:  "WithEnvelope",
			option: WithEnvelope(Envelope{Attack: 99}),
			check: func(t *testing.T, c createConfig) {
				if c.envelope.Attack != 99 {
					t.Errorf("envelope.Attack = %d, want 99", c.envelope.Attack)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := defaultCreateConfig()
			tc.option(&cfg)
			tc.check(t, cfg)
		})
	}
}

// -----------------------------------------------------------------------
// Tests for defaultCreateConfig
// -----------------------------------------------------------------------

func TestDefaultCreateConfig(t *testing.T) {
	cfg := defaultCreateConfig()

	if cfg.fileName != "SliceProgram" {
		t.Errorf("fileName = %q, want %q", cfg.fileName, "SliceProgram")
	}
	if cfg.version != 0x0001 {
		t.Errorf("version = 0x%04x, want 0x0001", cfg.version)
	}
	if cfg.compress {
		t.Error("compress = true, want false")
	}
	if cfg.voiceMode != VoiceModeDrum {
		t.Errorf("voiceMode = %v, want %v", cfg.voiceMode, VoiceModeDrum)
	}
	if cfg.priority != 1 {
		t.Errorf("priority = %d, want 1", cfg.priority)
	}
	if cfg.stereo {
		t.Error("stereo = true, want false")
	}
	// Envelope should match DefaultDrumEnvelope
	expectedEnv := DefaultDrumEnvelope()
	if cfg.envelope != expectedEnv {
		t.Errorf("envelope = %+v, want %+v", cfg.envelope, expectedEnv)
	}
}

// -----------------------------------------------------------------------
// Tests for VoiceMode constants
// -----------------------------------------------------------------------

func TestVoiceModeConstants(t *testing.T) {
	if VoiceModeDrum != 0 {
		t.Errorf("VoiceModeDrum = %d, want 0", VoiceModeDrum)
	}
	if VoiceModePoly != 1 {
		t.Errorf("VoiceModePoly = %d, want 1", VoiceModePoly)
	}
}