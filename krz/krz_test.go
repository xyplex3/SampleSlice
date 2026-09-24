// Package krz tests cover KRZ file creation, serialization, and the
// end-to-end object/PCM layout, verified against real K2000 KRZ files.
package krz

import (
	"encoding/binary"
	"testing"
)

func TestKRZFile_Serialize_RequiresKeymapAndProgram(t *testing.T) {
	f := NewKRZFile()
	if _, err := f.Serialize(); err == nil {
		t.Error("expected error with no keymaps/programs")
	}

	f.AddKeymap(NewSingleSampleKeymap(201, "km", 201))
	if _, err := f.Serialize(); err == nil {
		t.Error("expected error with no programs")
	}
}

func TestKRZFile_Serialize_Header(t *testing.T) {
	f := NewKRZFile()
	f.AddSample(&Sample{ID: 201, Name: "kick001", RootNote: 36, SampleRate: 44100, NumSamples: 4, pcmBytes: []byte{0, 0, 0, 0, 0, 0, 0, 0}})
	f.AddKeymap(NewSingleSampleKeymap(201, "kick001", 201))
	f.AddProgram(NewProgram(201, "kick001", 201, VoiceModeDrum, 1, false, DefaultDrumEnvelope()))

	data, err := f.Serialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data[0:4]) != "PRAM" {
		t.Errorf("magic = %q, want %q", data[0:4], "PRAM")
	}

	osize := binary.BigEndian.Uint32(data[4:8])
	wantTail := []byte{
		0x00, 0x00, 0xcd, 0x08, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x01, 0x2d, 0x03, 0x10, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	}
	if string(data[8:32]) != string(wantTail) {
		t.Errorf("header tail = %x, want %x", data[8:32], wantTail)
	}

	// The bytes at the declared osize offset must be the PCM data, not more
	// object bytes: the int32(0) end marker should sit exactly before it.
	endMarker := binary.BigEndian.Uint32(data[osize-4 : osize])
	if endMarker != 0 {
		t.Errorf("bytes just before osize = %d, want 0 (end marker)", endMarker)
	}
	if int(osize)+8 != len(data) {
		t.Errorf("file length = %d, want osize(%d)+8 (4 samples * 2 bytes)", len(data), osize)
	}
}

func TestKRZFile_Serialize_ObjectOrderAndParsing(t *testing.T) {
	// Verifies the file can be walked as a flat sequence of length-prefixed
	// blocks terminated by int32(0), exactly like real K2000 files.
	f := NewKRZFile()
	f.AddSample(&Sample{ID: 201, Name: "kick001", RootNote: 36, SampleRate: 44100, NumSamples: 2, pcmBytes: []byte{0, 0, 0, 0}})
	f.AddKeymap(NewSingleSampleKeymap(201, "kick001", 201))
	f.AddProgram(NewProgram(201, "kick001", 201, VoiceModeDrum, 1, false, DefaultDrumEnvelope()))

	data, err := f.Serialize()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pos := 32
	var seenTypes []uint16
	for {
		blocksize := int32(binary.BigEndian.Uint32(data[pos : pos+4]))
		if blocksize == 0 {
			break
		}
		total := int(-blocksize)
		hash := binary.BigEndian.Uint16(data[pos+4 : pos+6])
		seenTypes = append(seenTypes, TypeFromHash(hash))
		pos += total
		if pos > len(data) {
			t.Fatalf("object block ran past end of file")
		}
	}

	want := []uint16{T_SAMPLE, T_KEYMAP, T_PROGRAM}
	if len(seenTypes) != len(want) {
		t.Fatalf("object types = %v, want %v", seenTypes, want)
	}
	for i := range want {
		if seenTypes[i] != want[i] {
			t.Errorf("object %d type = %d, want %d", i, seenTypes[i], want[i])
		}
	}
}

func TestCreateFromSlices_EmptyReturnsError(t *testing.T) {
	if _, err := CreateFromSlices(nil); err == nil {
		t.Error("expected error for empty slices")
	}
}

func TestCreateFromSlices_ProducesParsableFile(t *testing.T) {
	slices := []SliceData{
		{Samples: []int16{100, 200, 300, 400}, Note: 36},
		{Samples: []int16{10, 20, 30}, Note: 38},
	}
	data, err := CreateFromSlices(slices, WithFileName("TestKit"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(data[0:4]) != "PRAM" {
		t.Fatalf("magic = %q, want PRAM", data[0:4])
	}
	osize := binary.BigEndian.Uint32(data[4:8])

	pos := 32
	counts := map[uint16]int{}
	for {
		blocksize := int32(binary.BigEndian.Uint32(data[pos : pos+4]))
		if blocksize == 0 {
			pos += 4
			break
		}
		total := int(-blocksize)
		hash := binary.BigEndian.Uint16(data[pos+4 : pos+6])
		counts[TypeFromHash(hash)]++
		pos += total
	}

	if uint32(pos) != osize {
		t.Errorf("end marker ended at %d, want osize %d", pos, osize)
	}
	// 2 slices -> 2 Sample, 2 per-hit Keymap + 1 master Keymap, 2 per-hit
	// Program + 1 master Program.
	if counts[T_SAMPLE] != 2 {
		t.Errorf("sample count = %d, want 2", counts[T_SAMPLE])
	}
	if counts[T_KEYMAP] != 3 {
		t.Errorf("keymap count = %d, want 3 (2 per-hit + 1 master)", counts[T_KEYMAP])
	}
	if counts[T_PROGRAM] != 3 {
		t.Errorf("program count = %d, want 3 (2 per-hit + 1 master)", counts[T_PROGRAM])
	}

	wantPCMLen := (4 + 3) * 2 // two slices, 16-bit samples
	if len(data)-int(osize) != wantPCMLen {
		t.Errorf("PCM region length = %d, want %d", len(data)-int(osize), wantPCMLen)
	}
}

func TestCreateFromSlices_MasterKeymapMapsNotesToPrograms(t *testing.T) {
	slices := []SliceData{
		{Samples: []int16{1, 2}, Note: 36},
		{Samples: []int16{3, 4}, Note: 38},
	}
	data, err := CreateFromSlices(slices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Walk objects to find the master keymap (id 200, multi-entry: its
	// payload is 30+128*3 bytes, unlike the 30+128-byte per-hit keymaps).
	pos := 32
	var masterPayload []byte
	for {
		blocksize := int32(binary.BigEndian.Uint32(data[pos : pos+4]))
		if blocksize == 0 {
			break
		}
		total := int(-blocksize)
		hash := binary.BigEndian.Uint16(data[pos+4 : pos+6])
		if TypeFromHash(hash) == T_KEYMAP {
			payload := data[pos+26 : pos+total]
			if len(payload) == 30+numKeys*3 {
				masterPayload = payload
			}
		}
		pos += total
	}
	if masterPayload == nil {
		t.Fatal("master (multi-entry) keymap not found")
	}

	entryID := func(key int) uint16 {
		off := 30 + key*3
		return binary.BigEndian.Uint16(masterPayload[off : off+2])
	}

	firstProgramID := uint16(baseUserObjectID + 1)  // 201
	secondProgramID := uint16(baseUserObjectID + 2) // 202

	if got := entryID(36); got != firstProgramID {
		t.Errorf("key 36 -> program %d, want %d", got, firstProgramID)
	}
	if got := entryID(38); got != secondProgramID {
		t.Errorf("key 38 -> program %d, want %d", got, secondProgramID)
	}
	if got := entryID(0); got != firstProgramID {
		t.Errorf("unmapped key 0 -> program %d, want default %d", got, firstProgramID)
	}
}

func TestCreateFromSlices_NoteZeroNotRemapped(t *testing.T) {
	slices := []SliceData{{Samples: []int16{1, 2}, Note: 0}}
	data, err := CreateFromSlices(slices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pos := 32
	for {
		blocksize := int32(binary.BigEndian.Uint32(data[pos : pos+4]))
		if blocksize == 0 {
			break
		}
		total := int(-blocksize)
		hash := binary.BigEndian.Uint16(data[pos+4 : pos+6])
		if TypeFromHash(hash) == T_SAMPLE {
			rootNote := data[pos+26+14]
			if rootNote != 0 {
				t.Errorf("sample rootNote = %d, want 0 (not remapped)", rootNote)
			}
			return
		}
		pos += total
	}
	t.Fatal("no sample object found")
}

func TestCreateFromSlices_ObjectIDsStartAt200(t *testing.T) {
	slices := []SliceData{{Samples: []int16{1, 2}, Note: 36}}
	data, err := CreateFromSlices(slices)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	pos := 32
	minID := uint16(0xFFFF)
	for {
		blocksize := int32(binary.BigEndian.Uint32(data[pos : pos+4]))
		if blocksize == 0 {
			break
		}
		total := int(-blocksize)
		hash := binary.BigEndian.Uint16(data[pos+4 : pos+6])
		if id := IDFromHash(hash); id < minID {
			minID = id
		}
		pos += total
	}
	if minID != baseUserObjectID {
		t.Errorf("smallest object ID = %d, want %d", minID, baseUserObjectID)
	}
}
