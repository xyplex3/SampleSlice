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
				expectedMagic := []byte{0x50, 0x52, 0x41, 0x4D} // "PRAM"
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
			name: "note zero is a valid explicit note, not remapped",
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
				expectedMagic := []byte{0x50, 0x52, 0x41, 0x4D} // "PRAM"
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

// -----------------------------------------------------------------------
// Regression tests for the header/table/format/model/note-0 fixes
// -----------------------------------------------------------------------

// firstObjectTableEntry decodes the hash and recorded offset of the first
// object table entry in a serialized KRZ file, using only the header's own
// declared model count so the test tracks headerSize's real definition
// instead of hardcoding it.
func firstObjectTableEntry(t *testing.T, data []byte) (hash uint16, offset uint32) {
	t.Helper()
	modelCount := binary.BigEndian.Uint16(data[6:8])
	headerSize := 4 + 2 + 2 + 2*int(modelCount) + 2
	hash = binary.BigEndian.Uint16(data[headerSize : headerSize+2])
	offset = binary.BigEndian.Uint32(data[headerSize+2 : headerSize+6])
	return hash, offset
}

// TestSerialize_ObjectTableOffsetsMatchData verifies the object table's
// recorded offset for the first object (the keymap, written first) actually
// points at that object's data in the file — i.e. the hash at data[offset:]
// matches the hash recorded in the table entry. This guards against
// headerSize drifting out of sync with the bytes actually written for the
// header (it previously did: a hardcoded headerSize=16 didn't match the
// real header length whenever len(Models) != 1).
func TestSerialize_ObjectTableOffsetsMatchData(t *testing.T) {
	data, err := CreateFromSlices([]SliceData{{Samples: []int16{100, 200}, Note: 36}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantHash, offset := firstObjectTableEntry(t, data)
	if int(offset)+2 > len(data) {
		t.Fatalf("recorded offset %d is out of bounds (len %d)", offset, len(data))
	}
	gotHash := binary.BigEndian.Uint16(data[offset : offset+2])
	if gotHash != wantHash {
		t.Errorf("hash at recorded offset %d = 0x%04x, want 0x%04x (table offset doesn't point at the object's own data)",
			offset, gotHash, wantHash)
	}
}

// TestSerialize_ObjectTableOffsetsMatchData_MultiModel is the same check
// with a non-default Models list, since headerSize depends on len(Models).
func TestSerialize_ObjectTableOffsetsMatchData_MultiModel(t *testing.T) {
	data, err := CreateFromSlices(
		[]SliceData{{Samples: []int16{100, 200}, Note: 36}},
		WithModels([]uint16{0x0064, 0x0065, 0x0066}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	wantHash, offset := firstObjectTableEntry(t, data)
	if int(offset)+2 > len(data) {
		t.Fatalf("recorded offset %d is out of bounds (len %d)", offset, len(data))
	}
	gotHash := binary.BigEndian.Uint16(data[offset : offset+2])
	if gotHash != wantHash {
		t.Errorf("hash at recorded offset %d = 0x%04x, want 0x%04x (table offset doesn't point at the object's own data)",
			offset, gotHash, wantHash)
	}
}

// keymapRootNote decodes the RootNote of the first sample block in the first
// (keymap) object of a serialized KRZ file produced with default options.
func keymapRootNote(t *testing.T, data []byte) uint16 {
	t.Helper()
	_, offset := firstObjectTableEntry(t, data)
	// Keymap.Serialize layout: hash(2) count(2) reserved(6) nameLen(1) name(16)
	// = 27 bytes, then per sample: type(1) flags(1) startNote(2) endNote(2)
	// rootNote(2) ...
	rootNoteOffset := offset + 27 + 1 + 1 + 2 + 2
	return binary.BigEndian.Uint16(data[rootNoteOffset : rootNoteOffset+2])
}

// TestCreateFromSlices_NoteZeroNotRemapped verifies MIDI note 0 is written
// through as-is instead of being silently remapped to 60 (C4).
func TestCreateFromSlices_NoteZeroNotRemapped(t *testing.T) {
	data, err := CreateFromSlices([]SliceData{{Samples: []int16{100, 200}, Note: 0}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := keymapRootNote(t, data); got != 0 {
		t.Errorf("RootNote = %d, want 0 (note 0 should not be remapped)", got)
	}
}

// TestKeymap_AddSample_Format2Is16Bit verifies format 2 (16-bit signed, the
// format CreateFromSlices now uses by default) produces two raw bytes per
// input sample, not the crushed-to-8-bit output format 8 used to produce.
func TestKeymap_AddSample_Format2Is16Bit(t *testing.T) {
	km := NewKeymap(1, "Test")
	samples := []int16{100, 200, 300, 400}
	km.AddSample(samples, 60, 60, 60, 2, false)

	got := len(km.Samples[0].RawData)
	want := len(samples) * 2
	if got != want {
		t.Errorf("RawData length = %d, want %d (2 bytes/sample for 16-bit format)", got, want)
	}
}

// TestCreateFromSlices_DefaultFormatIs16Bit verifies the default sample
// format used by CreateFromSlices is 2 (16-bit signed), not the old
// undefined format code 8.
func TestCreateFromSlices_DefaultFormatIs16Bit(t *testing.T) {
	data, err := CreateFromSlices([]SliceData{{Samples: []int16{100, 200}, Note: 36}})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, offset := firstObjectTableEntry(t, data)
	// Format field follows rootNote: type(1) flags(1) startNote(2) endNote(2)
	// rootNote(2) format(2).
	formatOffset := offset + 27 + 1 + 1 + 2 + 2 + 2
	format := binary.BigEndian.Uint16(data[formatOffset : formatOffset+2])
	if format != 2 {
		t.Errorf("default sample Format = %d, want 2 (16-bit signed)", format)
	}
}

// TestCreateFromSlices_WithModels verifies WithModels overrides the default
// PC2/PC3 model list.
func TestCreateFromSlices_WithModels(t *testing.T) {
	data, err := CreateFromSlices(
		[]SliceData{{Samples: []int16{100}, Note: 36}},
		WithModels([]uint16{0x1234}),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	modelCount := binary.BigEndian.Uint16(data[6:8])
	if modelCount != 1 {
		t.Fatalf("model count = %d, want 1", modelCount)
	}
	model := binary.BigEndian.Uint16(data[8:10])
	if model != 0x1234 {
		t.Errorf("model = 0x%04x, want 0x1234", model)
	}
}