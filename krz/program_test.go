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
// with different IDs/names/keymap refs (same envelope) — matching the
// real-file evidence that the rest of the object is a reusable constant template.
func TestProgram_Serialize_TemplateUnchangedElsewhere(t *testing.T) {
	p1 := NewProgram(201, "kick001", 201, VoiceModeDrum, 1, false, DefaultPolyEnvelope())
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

// TestProgram_Serialize_EnvelopePatch verifies that WithEnvelope's values
// land field-for-field in the 0x21 AMPENV segment, that the reserved byte
// (which has no config field) always stays 0, and that an all-zero envelope
// leaves the template's own shape untouched.
func TestProgram_Serialize_EnvelopePatch(t *testing.T) {
	e := Envelope{
		Att1Level: 100, Att1Time: 10,
		Att2Level: 20, Att2Time: 15,
		Att3Level: 40, Att3Time: 35,
		Dec1Level: 70, Dec1Time: 40,
		Rel1Level: 30, Rel1Time: 60,
		Rel2Level: 50, Rel2Time: 80,
		Rel3Time: 25,
	}
	p := NewProgram(201, "kick001", 201, VoiceModeDrum, 1, false, e)
	obj := p.Serialize()

	if got := obj[26+ampEnvLoopOffset]; got != 0 {
		t.Errorf("loop flag = %d, want 0", got)
	}

	type fieldOffset struct {
		name string
		off  int
		want byte
	}
	fields := []fieldOffset{
		{"Att1Level", ampEnvAtt1LevelOffset, 100},
		{"Att1Time", ampEnvAtt1TimeOffset, 10},
		{"Att2Level", ampEnvAtt2LevelOffset, 20},
		{"Att2Time", ampEnvAtt2TimeOffset, 15},
		{"Att3Level", ampEnvAtt3LevelOffset, 40},
		{"Att3Time", ampEnvAtt3TimeOffset, 35},
		{"Dec1Level", ampEnvDec1LevelOffset, 70},
		{"Dec1Time", ampEnvDec1TimeOffset, 40},
		{"Rel1Level", ampEnvRel1LevelOffset, 30},
		{"Rel1Time", ampEnvRel1TimeOffset, 60},
		{"Rel2Level", ampEnvRel2LevelOffset, 50},
		{"Rel2Time", ampEnvRel2TimeOffset, 80},
		{"Reserved", ampEnvReservedOffset, 0},
		{"Rel3Time", ampEnvRel3TimeOffset, 25},
	}
	for _, f := range fields {
		if got := obj[26+f.off]; got != f.want {
			t.Errorf("%s = %d, want %d", f.name, got, f.want)
		}
	}

	empty := Envelope{}
	p0 := NewProgram(201, "kick001", 201, VoiceModeDrum, 1, false, empty)
	obj0 := p0.Serialize()
	for _, f := range fields {
		if f.name == "Reserved" {
			continue // always 0 regardless of the template
		}
		if obj0[26+f.off] != programTemplate[f.off] {
			t.Errorf("%s: empty envelope should leave the template untouched", f.name)
		}
	}

	pClamp := NewProgram(201, "kick001", 201, VoiceModeDrum, 1, false, Envelope{Dec1Level: 200})
	objClamp := pClamp.Serialize()
	if lvl := objClamp[26+ampEnvDec1LevelOffset]; lvl != 100 {
		t.Errorf("clamped level = %d, want 100", lvl)
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
	if e.Att1Level != 100 || e.Att1Time != 0 || e.Dec1Time != 20 || e.Rel1Time != 5 {
		t.Errorf("unexpected drum envelope defaults: %+v", e)
	}
}

func TestDefaultPolyEnvelope(t *testing.T) {
	e := DefaultPolyEnvelope()
	if e.Att1Time != 5 || e.Rel1Time != 40 {
		t.Errorf("unexpected poly envelope defaults: %+v", e)
	}
}
