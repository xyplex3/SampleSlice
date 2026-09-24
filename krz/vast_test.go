package krz

import (
	"encoding/binary"
	"os"
	"testing"
)

const realReferenceFile = "/Users/xyplex2/samples/Percussion & FX/newKRZdrms/KRZDRMS.KRZ"

func skipIfMissing(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Stat(path); err != nil {
		t.Skipf("real reference file not present on this machine: %v", err)
	}
}

// TestLoadVAST_RealFile verifies LoadVAST finds a named program in a real
// K2000 file and extracts its tone-shaping segments.
func TestLoadVAST_RealFile(t *testing.T) {
	skipIfMissing(t, realReferenceFile)

	v, err := LoadVAST(realReferenceFile, "kick001")
	if err != nil {
		t.Fatalf("LoadVAST: %v", err)
	}
	if len(v.segments) == 0 {
		t.Fatal("expected at least one borrowed segment")
	}
	if _, ok := findSegment(v.segments, 0x40); !ok {
		t.Error("expected a Calvin (0x40) segment among the borrowed segments")
	}
	if _, ok := findSegment(v.segments, 0x53); !ok {
		t.Error("expected a Hobbes 4 (0x53) segment among the borrowed segments")
	}
}

// TestLoadVAST_ProgramNotFound verifies a clear error for an unknown name.
func TestLoadVAST_ProgramNotFound(t *testing.T) {
	skipIfMissing(t, realReferenceFile)

	if _, err := LoadVAST(realReferenceFile, "definitely-not-a-real-program-name"); err == nil {
		t.Fatal("expected an error for a nonexistent program name")
	}
}

// TestLoadVAST_MissingFile verifies a clear error when the file can't be read.
func TestLoadVAST_MissingFile(t *testing.T) {
	if _, err := LoadVAST("/nonexistent/path/to/nothing.krz", "anything"); err == nil {
		t.Fatal("expected an error for a missing file")
	}
}

// TestVASTApply_PreservesPayloadLength verifies splicing a donor's segments
// never changes the recipient payload's total length (segment length is
// determined entirely by tag, so substitution can't resize anything).
func TestVASTApply_PreservesPayloadLength(t *testing.T) {
	skipIfMissing(t, realReferenceFile)

	v, err := LoadVAST(realReferenceFile, "kick001")
	if err != nil {
		t.Fatalf("LoadVAST: %v", err)
	}

	payload := make([]byte, len(programTemplate))
	copy(payload, programTemplate)
	if err := v.apply(payload); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if len(payload) != len(programTemplate) {
		t.Fatalf("payload length changed: %d, want %d", len(payload), len(programTemplate))
	}
}

// TestVASTApply_KeymapRefGetsOverwritten verifies that after splicing a
// donor's Calvin segment (which carries the donor's own keymap reference),
// Program.Serialize still ends up with THIS program's KeymapID, not the
// donor's — the keymap-ref patch must run after the VAST splice.
func TestVASTApply_KeymapRefGetsOverwritten(t *testing.T) {
	skipIfMissing(t, realReferenceFile)

	v, err := LoadVAST(realReferenceFile, "kick001") // donor's own keymap ref is 201
	if err != nil {
		t.Fatalf("LoadVAST: %v", err)
	}

	p := NewProgram(999, "test", 555, VoiceModeDrum, 1, false, Envelope{})
	p.VAST = v
	obj := p.Serialize()

	got := binary.BigEndian.Uint16(obj[26+programKeymapRefOffset : 26+programKeymapRefOffset+2])
	if got != 555 {
		t.Errorf("keymap ref after VAST splice = %d, want 555 (this program's own KeymapID, not the donor's)", got)
	}
}

// TestCreateFromSlices_WithVAST verifies the public CreateFromSlices ->
// WithVAST path produces a file whose per-hit Program's Hobbes 4 segment
// matches the donor's, while everything else about the file still parses
// as a normal, valid KRZ file.
func TestCreateFromSlices_WithVAST(t *testing.T) {
	skipIfMissing(t, realReferenceFile)

	v, err := LoadVAST(realReferenceFile, "kick001")
	if err != nil {
		t.Fatalf("LoadVAST: %v", err)
	}
	donorHobbes4, _ := findSegment(v.segments, 0x53)

	data, err := CreateFromSlices(
		[]SliceData{{Samples: []int16{1, 2, 3, 4}, Note: 36}},
		WithVAST(v),
	)
	if err != nil {
		t.Fatalf("CreateFromSlices: %v", err)
	}

	objects, err := ParseObjects(data)
	if err != nil {
		t.Fatalf("ParseObjects: %v", err)
	}
	prog, ok := FindObject(objects, T_PROGRAM, "slice001")
	if !ok {
		t.Fatal("per-hit program \"slice001\" not found")
	}
	segs, err := parseSegments(prog.Payload, 0)
	if err != nil {
		t.Fatalf("parsing program segments: %v", err)
	}
	gotHobbes4, ok := findSegment(segs, 0x53)
	if !ok {
		t.Fatal("expected a Hobbes 4 (0x53) segment in the generated program")
	}
	if string(gotHobbes4.data) != string(donorHobbes4.data) {
		t.Errorf("Hobbes 4 data = %x, want donor's %x", gotHobbes4.data, donorHobbes4.data)
	}
}
