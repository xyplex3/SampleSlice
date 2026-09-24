package krz

import (
	"encoding/binary"
	"testing"
)

func decodeKeymapPayload(t *testing.T, obj []byte) []byte {
	t.Helper()
	if len(obj) < 26 {
		t.Fatalf("object too short: %d bytes", len(obj))
	}
	return obj[26:]
}

func TestSingleSampleKeymap_Serialize(t *testing.T) {
	km := NewSingleSampleKeymap(201, "kick001", 201)
	payload := decodeKeymapPayload(t, km.Serialize())

	if len(payload) != 30+numKeys {
		t.Fatalf("payload length = %d, want %d", len(payload), 30+numKeys)
	}
	if got := binary.BigEndian.Uint16(payload[2:4]); got != 201 {
		t.Errorf("sampleId = %d, want 201", got)
	}
	if method := binary.BigEndian.Uint16(payload[4:6]); method != 1 {
		t.Errorf("method = %d, want 1", method)
	}
	if basePitch := binary.BigEndian.Uint16(payload[6:8]); basePitch != 0 {
		t.Errorf("basePitch = %d, want 0", basePitch)
	}
	if cents := binary.BigEndian.Uint16(payload[8:10]); cents != 100 {
		t.Errorf("centsPerEntry = %d, want 100", cents)
	}
	if epv := binary.BigEndian.Uint16(payload[10:12]); epv != 127 {
		t.Errorf("entriesPerVel = %d, want 127", epv)
	}
	if es := binary.BigEndian.Uint16(payload[12:14]); es != 1 {
		t.Errorf("entrySize = %d, want 1", es)
	}
	for j := 0; j < 8; j++ {
		want := uint16((8 - j) * 2)
		if got := binary.BigEndian.Uint16(payload[14+j*2 : 16+j*2]); got != want {
			t.Errorf("Level[%d] = %d, want %d", j, got, want)
		}
	}
	for i := 30; i < len(payload); i++ {
		if payload[i] != 1 {
			t.Fatalf("key entry byte at payload offset %d = %d, want 1", i, payload[i])
		}
	}
}

func TestMasterKeymap_DefaultAndSetEntry(t *testing.T) {
	km := NewMasterKeymap(200, "DRM02", 201)
	km.SetEntry(36, 202)
	km.SetEntry(37, 203)
	payload := decodeKeymapPayload(t, km.Serialize())

	if len(payload) != 30+numKeys*3 {
		t.Fatalf("payload length = %d, want %d", len(payload), 30+numKeys*3)
	}
	if got := binary.BigEndian.Uint16(payload[2:4]); got != 0 {
		t.Errorf("sampleId = %d, want 0 (multi-sample keymap)", got)
	}
	if method := binary.BigEndian.Uint16(payload[4:6]); method != 3 {
		t.Errorf("method = %d, want 3", method)
	}
	if es := binary.BigEndian.Uint16(payload[12:14]); es != 3 {
		t.Errorf("entrySize = %d, want 3", es)
	}

	entryAt := func(key int) (id uint16, flag byte) {
		off := 30 + key*3
		return binary.BigEndian.Uint16(payload[off : off+2]), payload[off+2]
	}

	if id, flag := entryAt(0); id != 201 || flag != 1 {
		t.Errorf("key 0 = (id=%d flag=%d), want (201, 1) [default]", id, flag)
	}
	if id, flag := entryAt(36); id != 202 || flag != 1 {
		t.Errorf("key 36 = (id=%d flag=%d), want (202, 1)", id, flag)
	}
	if id, flag := entryAt(37); id != 203 || flag != 1 {
		t.Errorf("key 37 = (id=%d flag=%d), want (203, 1)", id, flag)
	}
	if id, _ := entryAt(127); id != 201 {
		t.Errorf("key 127 = id %d, want 201 (untouched default)", id)
	}
}

func TestMasterKeymap_SetEntry_OutOfRangeIsNoOp(t *testing.T) {
	km := NewMasterKeymap(200, "DRM02", 201)
	km.SetEntry(-1, 999)
	km.SetEntry(128, 999)
	payload := decodeKeymapPayload(t, km.Serialize())
	for key := 0; key < numKeys; key++ {
		off := 30 + key*3
		if id := binary.BigEndian.Uint16(payload[off : off+2]); id != 201 {
			t.Fatalf("key %d = %d, want unchanged default 201", key, id)
		}
	}
}

func TestSingleSampleKeymap_SetEntryIsNoOp(t *testing.T) {
	km := NewSingleSampleKeymap(201, "kick001", 201)
	km.SetEntry(60, 999) // should have no effect: not a multi keymap
	payload := decodeKeymapPayload(t, km.Serialize())
	for i := 30; i < len(payload); i++ {
		if payload[i] != 1 {
			t.Fatalf("payload offset %d = %d, want 1 (unaffected by SetEntry)", i, payload[i])
		}
	}
}

func TestKeymap_Hash(t *testing.T) {
	km := NewSingleSampleKeymap(201, "kick001", 201)
	want := GenerateHash(201, T_KEYMAP)
	if km.Hash() != want {
		t.Errorf("Hash() = 0x%04x, want 0x%04x", km.Hash(), want)
	}
}
