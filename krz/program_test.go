package krz

import (
	"encoding/binary"
	"testing"
)

func TestProgram_Serialize_CommonHeader(t *testing.T) {
	p := NewProgram(201, "kick001", 201, VoiceModeDrum, 1, false, DefaultDrumEnvelope())
	obj := p.Serialize()

	wantHash := GenerateHash(201, T_PROGRAM)
	if got := binary.BigEndian.Uint16(obj[4:6]); got != wantHash {
		t.Errorf("hash = 0x%04x, want 0x%04x", got, wantHash)
	}
	blocksize := int32(binary.BigEndian.Uint32(obj[0:4]))
	if int(-blocksize) != len(obj) {
		t.Errorf("blocksize = %d, want -%d", blocksize, len(obj))
	}
	name := obj[10:26]
	if string(name) != "kick001         " {
		t.Errorf("name = %q, want %q", name, "kick001         ")
	}
}

// TestProgram_Serialize_KeymapReference verifies the one field confirmed to
// vary between real Program objects: the referenced Keymap ID at payload
// offset programKeymapRefOffset.
func TestProgram_Serialize_KeymapReference(t *testing.T) {
	tests := []struct {
		progID, keymapID uint16
	}{
		{200, 200}, // master program -> master keymap
		{201, 201}, // per-hit program -> its own per-hit keymap
		{202, 500}, // arbitrary program/keymap ID pairing
	}
	for _, tt := range tests {
		p := NewProgram(tt.progID, "Test", tt.keymapID, VoiceModeDrum, 1, false, DefaultDrumEnvelope())
		obj := p.Serialize()
		payload := obj[26:]
		got := binary.BigEndian.Uint16(payload[programKeymapRefOffset : programKeymapRefOffset+2])
		if got != tt.keymapID {
			t.Errorf("progID %d: keymap ref = %d, want %d", tt.progID, got, tt.keymapID)
		}
	}
}

// TestProgram_Serialize_TemplateUnchangedElsewhere verifies that only the
// hash, name, and keymap-reference field differ between two Program objects
// with different IDs/names/keymap refs — matching the real-file evidence
// that the rest of the 280-byte object is a reusable constant template.
func TestProgram_Serialize_TemplateUnchangedElsewhere(t *testing.T) {
	p1 := NewProgram(201, "kick001", 201, VoiceModeDrum, 1, false, DefaultDrumEnvelope())
	p2 := NewProgram(202, "kick002", 202, VoiceModePoly, 5, true, DefaultPolyEnvelope())

	obj1 := p1.Serialize()
	obj2 := p2.Serialize()
	if len(obj1) != len(obj2) {
		t.Fatalf("lengths differ: %d vs %d", len(obj1), len(obj2))
	}

	allowedDiff := map[int]bool{
		5:  true, // hash low byte
		16: true, // name digit
	}
	for i := 26 + programKeymapRefOffset; i < 26+programKeymapRefOffset+2; i++ {
		allowedDiff[i] = true
	}

	for i := range obj1 {
		if obj1[i] != obj2[i] && !allowedDiff[i] {
			t.Errorf("unexpected diff at byte %d: %02x vs %02x (only hash/name/keymap-ref should vary)", i, obj1[i], obj2[i])
		}
	}
}

func TestProgram_Hash(t *testing.T) {
	p := NewProgram(200, "DRM02", 200, VoiceModeDrum, 1, false, DefaultDrumEnvelope())
	want := GenerateHash(200, T_PROGRAM)
	if p.Hash() != want {
		t.Errorf("Hash() = 0x%04x, want 0x%04x", p.Hash(), want)
	}
}

func TestDefaultDrumEnvelope(t *testing.T) {
	e := DefaultDrumEnvelope()
	if e.Attack != 0 || e.Decay1 != 20 || e.Release != 5 {
		t.Errorf("unexpected drum envelope defaults: %+v", e)
	}
}

func TestDefaultPolyEnvelope(t *testing.T) {
	e := DefaultPolyEnvelope()
	if e.Attack != 5 || e.Release != 40 {
		t.Errorf("unexpected poly envelope defaults: %+v", e)
	}
}
