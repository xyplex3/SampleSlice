package krz

import (
	"os"
	"testing"
)

// TestParseObjects_RoundTripsOwnOutput verifies that a file this package
// generates can be read back: every Sample/Keymap/Program object we wrote
// is found again with the right type, ID, and name.
func TestParseObjects_RoundTripsOwnOutput(t *testing.T) {
	data, err := CreateFromSlices([]SliceData{
		{Samples: []int16{1, 2, 3, 4}, Note: 36},
	}, WithFileName("TestKit"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	objects, err := ParseObjects(data)
	if err != nil {
		t.Fatalf("ParseObjects: %v", err)
	}

	counts := map[uint16]int{}
	for _, o := range objects {
		counts[o.Type]++
	}
	if counts[T_SAMPLE] != 1 {
		t.Errorf("sample count = %d, want 1", counts[T_SAMPLE])
	}
	if counts[T_KEYMAP] != 2 { // 1 per-hit + 1 master
		t.Errorf("keymap count = %d, want 2", counts[T_KEYMAP])
	}
	if counts[T_PROGRAM] != 2 { // 1 per-hit + 1 master
		t.Errorf("program count = %d, want 2", counts[T_PROGRAM])
	}

	perHitProgramID := uint16(baseUserObjectID + 1)
	prog, ok := FindObject(objects, T_PROGRAM, "slice001")
	if !ok {
		t.Fatal("per-hit program \"slice001\" not found")
	}
	if prog.ID != perHitProgramID {
		t.Errorf("per-hit program ID = %d, want %d", prog.ID, perHitProgramID)
	}

	// Its Payload should parse as a valid segment stream starting at 0 (not
	// 2 — that offset is specific to this package's internal programTemplate
	// representation, not to objects read back from a real file).
	segs, err := parseSegments(prog.Payload, 0)
	if err != nil {
		t.Fatalf("parsing program payload as segments: %v", err)
	}
	if _, ok := findSegment(segs, 0x08); !ok {
		t.Error("expected a PGM (0x08) segment in the per-hit program")
	}
	if _, ok := findSegment(segs, 0x40); !ok {
		t.Error("expected a Calvin (0x40) segment in the per-hit program")
	}
}

// TestParseObjects_RealFile parses an actual hardware-authored K2000 file,
// if present on this machine, and sanity-checks a known object. This test
// is skipped (not failed) when the file isn't present, since it documents
// real-world compatibility rather than exercising package-internal logic.
func TestParseObjects_RealFile(t *testing.T) {
	const path = "/Users/xyplex2/samples/Percussion & FX/newKRZdrms/KRZDRMS.KRZ"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("real reference file not present on this machine: %v", err)
	}

	objects, err := ParseObjects(data)
	if err != nil {
		t.Fatalf("ParseObjects: %v", err)
	}

	prog, ok := FindObject(objects, T_PROGRAM, "kick001")
	if !ok {
		t.Fatal("\"kick001\" program not found in real reference file")
	}
	if prog.ID != 201 {
		t.Errorf("kick001 program ID = %d, want 201", prog.ID)
	}

	segs, err := parseSegments(prog.Payload, 0)
	if err != nil {
		t.Fatalf("parsing real program payload as segments: %v", err)
	}
	pgm, ok := findSegment(segs, 0x08)
	if !ok {
		t.Fatal("expected a PGM (0x08) segment in kick001")
	}
	// PGM byte 0 = mode; doc-confirmed value 2 = "K2000, or K2500 program
	// without impact feature", matching this one-shot drum voice.
	if pgm.data[0] != 2 {
		t.Errorf("PGM mode = %d, want 2", pgm.data[0])
	}

	cal, ok := findSegment(segs, 0x40)
	if !ok {
		t.Fatal("expected a Calvin (0x40) segment in kick001")
	}
	// Calvin's keymap reference (doc field 0x0C-0D, i.e. data[11:13] since
	// data excludes the tag byte the doc counts as field 0x00) should be
	// kick001's own keymap, ID 201.
	keymapRef := uint16(cal.data[11])<<8 | uint16(cal.data[12])
	if keymapRef != 201 {
		t.Errorf("Calvin keymap ref = %d, want 201", keymapRef)
	}
}
