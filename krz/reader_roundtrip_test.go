package krz

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// TestParseObjects_AllRealFiles walks every .KRZ file found under the local
// ~/samples tree (skipped when none are present or the directory doesn't
// exist) through ParseObjects and asserts every file parses with zero walk
// errors and yields at least one Program object. This encodes the
// Python-walker verification of the flat object framing as a permanent
// regression test; the Dropbox tree is deliberately excluded because
// cloud-placeholder files can block indefinitely on read.
func TestParseObjects_AllRealFiles(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home directory: %v", err)
	}
	root := filepath.Join(home, "samples")
	if _, err := os.Stat(root); err != nil {
		t.Skipf("no ~/samples directory on this machine: %v", err)
	}

	var files []string
	err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".KRZ" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Skipf("walking ~/samples failed: %v", err)
	}
	if len(files) == 0 {
		t.Skip("no .KRZ files found under ~/samples")
	}

	progsFound := 0
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: reading: %v", path, err)
			continue
		}
		objects, err := ParseObjects(data)
		if err != nil {
			t.Errorf("%s: ParseObjects: %v", path, err)
			continue
		}
		if len(objects) == 0 {
			t.Errorf("%s: parsed with zero objects", path)
			continue
		}
		counts := map[uint16]int{}
		for _, o := range objects {
			counts[o.Type]++
		}
		if counts[T_PROGRAM] == 0 {
			t.Errorf("%s: no Program objects found (types: %v)", path, counts)
			continue
		}
		progsFound += counts[T_PROGRAM]
	}
	if progsFound == 0 {
		t.Errorf("walked %d files, found no Program objects at all", len(files))
	}
}

// TestCreateFromSlices_EnvelopeRoundTrip generates a multi-slice file with a
// custom envelope, reads it back through ParseObjects + parseSegments, and
// asserts every per-hit Program's 0x21 AMPENV segment carries exactly the
// configured values — tying the envelope patch end-to-end through the reader.
func TestCreateFromSlices_EnvelopeRoundTrip(t *testing.T) {
	env := Envelope{
		Att1Level: 100, Att1Time: 10,
		Att2Level: 80, Att2Time: 20,
		Att3Level: 60, Att3Time: 30,
		Dec1Level: 50, Dec1Time: 40,
		Rel1Level: 40, Rel1Time: 50,
		Rel2Level: 20, Rel2Time: 60,
		Rel3Time: 70,
	}
	data, err := CreateFromSlices([]SliceData{
		{Samples: []int16{100, 200, 300, 400}, Note: 36},
		{Samples: []int16{10, 20, 30}, Note: 38},
	}, WithFileName("TestKit"), WithEnvelope(env))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	objects, err := ParseObjects(data)
	if err != nil {
		t.Fatalf("ParseObjects: %v", err)
	}

	perHit1, ok := FindObject(objects, T_PROGRAM, "slice001")
	if !ok {
		t.Fatal("per-hit program \"slice001\" not found")
	}
	perHit2, ok := FindObject(objects, T_PROGRAM, "slice002")
	if !ok {
		t.Fatal("per-hit program \"slice002\" not found")
	}

	want := map[int]byte{
		0:  0,   // loop flag
		1:  100, // Att1 level
		2:  10,  // Att1 time
		3:  80,  // Att2 level
		4:  20,  // Att2 time
		5:  60,  // Att3 level
		6:  30,  // Att3 time
		7:  50,  // Dec1 level
		8:  40,  // Dec1 time
		9:  40,  // Rel1 level
		10: 50,  // Rel1 time
		11: 20,  // Rel2 level
		12: 60,  // Rel2 time
		13: 0,   // reserved (AMPENV has no Rel3 level byte)
		14: 70,  // Rel3 time
	}
	for name, prog := range map[string]Object{"slice001": perHit1, "slice002": perHit2} {
		segs, err := parseSegments(prog.Payload, 0)
		if err != nil {
			t.Fatalf("%s: parseSegments: %v", name, err)
		}
		amp, ok := findSegment(segs, 0x21)
		if !ok {
			t.Fatalf("%s: no AMPENV (0x21) segment", name)
		}
		for i, w := range want {
			if amp.data[i] != w {
				t.Errorf("%s: AMPENV byte %d = %d, want %d", name, i, amp.data[i], w)
			}
		}
	}
}

// TestCreateFromSlices_EnvelopeClampRoundTrip verifies through the reader
// that config levels above 100 clamp to 100 in the written AMPENV bytes.
func TestCreateFromSlices_EnvelopeClampRoundTrip(t *testing.T) {
	env := Envelope{Att1Level: 200, Dec1Level: 255}
	data, err := CreateFromSlices([]SliceData{
		{Samples: []int16{1, 2, 3}, Note: 36},
	}, WithFileName("TestKit"), WithEnvelope(env))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	objects, err := ParseObjects(data)
	if err != nil {
		t.Fatalf("ParseObjects: %v", err)
	}
	prog, ok := FindObject(objects, T_PROGRAM, "slice001")
	if !ok {
		t.Fatal("per-hit program not found")
	}
	segs, err := parseSegments(prog.Payload, 0)
	if err != nil {
		t.Fatalf("parseSegments: %v", err)
	}
	amp, ok := findSegment(segs, 0x21)
	if !ok {
		t.Fatal("no AMPENV (0x21) segment")
	}
	if amp.data[1] != 100 {
		t.Errorf("Att1 level = %d, want 100 (clamped from 200)", amp.data[1])
	}
	if amp.data[7] != 100 {
		t.Errorf("Dec1 level = %d, want 100 (clamped from 255)", amp.data[7])
	}
}
