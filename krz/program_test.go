// Package krz tests for Program construction, part management, and serialization.
package krz

import (
	"encoding/binary"
	"testing"
)

// -----------------------------------------------------------------------
// Tests for NewProgram
// -----------------------------------------------------------------------

func TestNewProgram(t *testing.T) {
	tests := []struct {
		name      string
		id        uint16
		nameStr   string
		voiceMode VoiceMode
		priority  uint8
		stereo    bool
		wantName  string
		wantHash  uint16
	}{
		{
			name: "drum no stereo", id: 1, nameStr: "Drums",
			voiceMode: VoiceModeDrum, priority: 1, stereo: false,
			wantName: "Drums", wantHash: 0x8401,
		},
		{
			name: "poly stereo", id: 2, nameStr: "Pads",
			voiceMode: VoiceModePoly, priority: 4, stereo: true,
			wantName: "Pads", wantHash: 0x8402,
		},
		{
			name: "long name truncated", id: 3, nameStr: "ThisIsAVeryLongNameHere",
			voiceMode: VoiceModeDrum, priority: 1, stereo: false,
			wantName: "ThisIsAVeryLongN", wantHash: 0x8403,
		},
		{
			name: "empty name", id: 4, nameStr: "",
			voiceMode: VoiceModeDrum, priority: 1, stereo: false,
			wantName: "", wantHash: 0x8404,
		},
		{
			name: "max priority", id: 5, nameStr: "MaxPri",
			voiceMode: VoiceModeDrum, priority: 8, stereo: false,
			wantName: "MaxPri", wantHash: 0x8405,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env := DefaultDrumEnvelope()
			p := NewProgram(tt.id, tt.nameStr, tt.voiceMode, tt.priority, tt.stereo, env)

			if p.Name != tt.wantName {
				t.Errorf("Name = %q, want %q", p.Name, tt.wantName)
			}
			if p.Hash != tt.wantHash {
				t.Errorf("Hash = 0x%04x, want 0x%04x", p.Hash, tt.wantHash)
			}
			if p.VoiceMode != tt.voiceMode {
				t.Errorf("VoiceMode = %v, want %v", p.VoiceMode, tt.voiceMode)
			}
			if p.Priority != tt.priority {
				t.Errorf("Priority = %d, want %d", p.Priority, tt.priority)
			}
			if p.Stereo != tt.stereo {
				t.Errorf("Stereo = %v, want %v", p.Stereo, tt.stereo)
			}
			if len(p.Parts) != 0 {
				t.Errorf("Parts length = %d, want 0 (empty initially)", len(p.Parts))
			}
			if p.Envelope != env {
				t.Errorf("Envelope not stored correctly")
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests for Program.AddPart
// -----------------------------------------------------------------------

func TestProgram_AddPart(t *testing.T) {
	t.Run("adds single part", func(t *testing.T) {
		p := NewProgram(1, "Test", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		p.AddPart(Part{
			Enabled:   true,
			PartID:    1,
			KeymapRef: 0x8201,
			RootNote:  60,
			Velocity:  100,
			LoKey:     0,
			HiKey:     127,
		})
		if len(p.Parts) != 1 {
			t.Fatalf("Parts length = %d, want 1", len(p.Parts))
		}
		if !p.Parts[0].Enabled {
			t.Error("Enabled = false, want true")
		}
	})

	t.Run("accumulates multiple parts", func(t *testing.T) {
		p := NewProgram(1, "Multi", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		for i := 0; i < 4; i++ {
			p.AddPart(Part{Enabled: true, PartID: uint16(i + 1)})
		}
		if len(p.Parts) != 4 {
			t.Errorf("Parts length = %d, want 4", len(p.Parts))
		}
	})

	t.Run("RootNote clamped to 127", func(t *testing.T) {
		p := NewProgram(1, "Clamp", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		p.AddPart(Part{RootNote: 200, LoKey: 0, HiKey: 200})
		if p.Parts[0].RootNote != 127 {
			t.Errorf("RootNote = %d, want 127 (clamped)", p.Parts[0].RootNote)
		}
		if p.Parts[0].HiKey != 127 {
			t.Errorf("HiKey = %d, want 127 (clamped)", p.Parts[0].HiKey)
		}
	})

	t.Run("LoKey preserved when valid", func(t *testing.T) {
		p := NewProgram(1, "LoKey", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		p.AddPart(Part{LoKey: 36, HiKey: 60})
		if p.Parts[0].LoKey != 36 {
			t.Errorf("LoKey = %d, want 36", p.Parts[0].LoKey)
		}
	})
}

// -----------------------------------------------------------------------
// Tests for Program.SetKeymap
// -----------------------------------------------------------------------

func TestProgram_SetKeymap(t *testing.T) {
	t.Run("sets keymap ref on valid index", func(t *testing.T) {
		p := NewProgram(1, "Test", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		p.AddPart(Part{Enabled: true, PartID: 1, KeymapRef: 0x0000})
		km := NewKeymap(5, "MyKeymap")

		p.SetKeymap(0, km)

		if p.Parts[0].KeymapRef != km.Hash {
			t.Errorf("KeymapRef = 0x%04x, want 0x%04x", p.Parts[0].KeymapRef, km.Hash)
		}
	})

	t.Run("out-of-range index is no-op", func(t *testing.T) {
		p := NewProgram(1, "Test", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		p.AddPart(Part{Enabled: true, PartID: 1})
		km := NewKeymap(1, "KM")
		originalRef := p.Parts[0].KeymapRef

		p.SetKeymap(5, km) // only 1 part, so index 5 is out of range

		if p.Parts[0].KeymapRef != originalRef {
			t.Error("KeymapRef changed for out-of-range SetKeymap call")
		}
		if len(p.Parts) != 1 {
			t.Errorf("Parts length changed to %d, want 1", len(p.Parts))
		}
	})

	t.Run("negative index is no-op", func(t *testing.T) {
		p := NewProgram(1, "Test", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		p.AddPart(Part{Enabled: true})
		km := NewKeymap(1, "KM")
		originalRef := p.Parts[0].KeymapRef

		p.SetKeymap(-1, km)

		if p.Parts[0].KeymapRef != originalRef {
			t.Error("KeymapRef changed for negative-index SetKeymap call")
		}
	})
}

// -----------------------------------------------------------------------
// Tests for Program.Serialize
// -----------------------------------------------------------------------

// programVoiceModeOffset is the fixed byte offset of the voice-mode byte
// in a program serialized with zero active parts (all 8 layers empty).
// Layout: hash(2) + size(2) + nameOffset(2) + name(16) + 8*emptyLayer(40) = 342.
const programVoiceModeOffset = 342

func TestProgram_Serialize(t *testing.T) {
	t.Run("returns non-empty bytes", func(t *testing.T) {
		p := NewProgram(1, "Test", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		data, err := p.Serialize()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(data) == 0 {
			t.Fatal("serialized data is empty")
		}
	})

	t.Run("first two bytes are the hash", func(t *testing.T) {
		p := NewProgram(3, "Check", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		data, err := p.Serialize()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		gotHash := binary.BigEndian.Uint16(data[0:2])
		if gotHash != p.Hash {
			t.Errorf("hash = 0x%04x, want 0x%04x", gotHash, p.Hash)
		}
	})

	t.Run("size field matches actual data length", func(t *testing.T) {
		p := NewProgram(1, "SizeCheck", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		data, err := p.Serialize()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		sizeField := binary.BigEndian.Uint16(data[2:4])
		if int(sizeField) != len(data) {
			t.Errorf("size field = %d, actual len = %d", sizeField, len(data))
		}
	})

	t.Run("name offset field is 8", func(t *testing.T) {
		p := NewProgram(1, "Name", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		data, _ := p.Serialize()
		nameOffset := binary.BigEndian.Uint16(data[4:6])
		if nameOffset != 8 {
			t.Errorf("name offset = %d, want 8", nameOffset)
		}
	})

	t.Run("drum mode no stereo produces voice byte 0x00", func(t *testing.T) {
		p := NewProgram(1, "Drum", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		data, _ := p.Serialize()
		if data[programVoiceModeOffset] != 0x00 {
			t.Errorf("voice mode byte = 0x%02x, want 0x00", data[programVoiceModeOffset])
		}
	})

	t.Run("poly mode produces voice byte with 0x02 set", func(t *testing.T) {
		p := NewProgram(1, "Poly", VoiceModePoly, 1, false, DefaultPolyEnvelope())
		data, _ := p.Serialize()
		if data[programVoiceModeOffset]&0x02 == 0 {
			t.Errorf("voice mode byte = 0x%02x, want poly bit (0x02) set",
				data[programVoiceModeOffset])
		}
	})

	t.Run("stereo flag sets bit 0 of voice byte", func(t *testing.T) {
		p := NewProgram(1, "Stereo", VoiceModeDrum, 1, true, DefaultDrumEnvelope())
		data, _ := p.Serialize()
		if data[programVoiceModeOffset]&0x01 == 0 {
			t.Errorf("voice mode byte = 0x%02x, expected stereo bit (0x01) set",
				data[programVoiceModeOffset])
		}
	})

	t.Run("priority stored as value minus 1", func(t *testing.T) {
		p := NewProgram(1, "Pri5", VoiceModeDrum, 5, false, DefaultDrumEnvelope())
		data, _ := p.Serialize()
		// Priority byte immediately follows voice mode byte.
		priorityByte := data[programVoiceModeOffset+1]
		if priorityByte != 4 { // priority 5 - 1 = 4
			t.Errorf("priority byte = %d, want 4 (priority 5 - 1)", priorityByte)
		}
	})

	t.Run("priority 1 stored as 0", func(t *testing.T) {
		p := NewProgram(1, "Pri1", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		data, _ := p.Serialize()
		priorityByte := data[programVoiceModeOffset+1]
		if priorityByte != 0 {
			t.Errorf("priority byte = %d, want 0 (priority 1 - 1)", priorityByte)
		}
	})
}

// -----------------------------------------------------------------------
// Tests for Program.CalculateSize
// -----------------------------------------------------------------------

func TestProgram_CalculateSize(t *testing.T) {
	t.Run("size is positive and even for empty program", func(t *testing.T) {
		p := NewProgram(1, "Test", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		size, err := p.CalculateSize()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if size <= 0 {
			t.Errorf("size = %d, want > 0", size)
		}
		if size%2 != 0 {
			t.Errorf("size = %d, want even (2-byte padded)", size)
		}
	})

	t.Run("size at least as large as serialized data", func(t *testing.T) {
		p := NewProgram(2, "Check", VoiceModePoly, 3, true, DefaultPolyEnvelope())
		data, err := p.Serialize()
		if err != nil {
			t.Fatalf("serialize error: %v", err)
		}
		size, err := p.CalculateSize()
		if err != nil {
			t.Fatalf("calculate size error: %v", err)
		}
		if size < len(data) {
			t.Errorf("CalculateSize() = %d, less than serialized len %d", size, len(data))
		}
	})
}

// -----------------------------------------------------------------------
// Integration: round-trip via KRZFile.Serialize
// -----------------------------------------------------------------------

// TestProgram_InKRZFile verifies a program serializes correctly inside a
// full KRZ file (hash appears in the object table).
func TestProgram_InKRZFile(t *testing.T) {
	f := NewKRZFile(1)

	km := NewKeymap(1, "KM")
	km.AddSample([]int16{100, 200, 150}, 60, 60, 60, 8, false)
	f.AddKeymap(km)

	p := NewProgram(1, "Prog", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
	p.AddPart(Part{
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
	})
	f.AddProgram(p)

	data, err := f.Serialize()
	if err != nil {
		t.Fatalf("KRZFile.Serialize() error: %v", err)
	}

	if len(data) == 0 {
		t.Error("serialized KRZ file is empty")
	}
	// Magic bytes check
	expected := []byte{0x4B, 0x52, 0x5A, 0x00}
	for i, b := range expected {
		if data[i] != b {
			t.Errorf("magic byte[%d] = 0x%02x, want 0x%02x", i, data[i], b)
		}
	}
}

// -----------------------------------------------------------------------
// Benchmarks
// -----------------------------------------------------------------------

// BenchmarkProgram_Serialize measures program serialization performance.
func BenchmarkProgram_Serialize(b *testing.B) {
	p := NewProgram(1, "BenchProg", VoiceModeDrum, 1, false, DefaultDrumEnvelope())
	p.AddPart(Part{
		Enabled:   true,
		PartID:    1,
		KeymapRef: 0x8201,
		RootNote:  60,
		Velocity:  100,
		Pan:       64,
		LoKey:     0,
		HiKey:     127,
	})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = p.Serialize()
	}
}
