package krz

import "testing"

func TestSegmentDataLen(t *testing.T) {
	tests := []struct {
		name string
		tag  byte
		want int
	}{
		{"PGM", 0x08, 15},
		{"LYR", 0x09, 15},
		{"Effect Control", 0x0F, 7},
		{"FUN1 (masked 0x18)", 0x18, 3},
		{"GFUN2 (masked 0x18)", 0x19, 3},
		{"FXFUN1 (masked 0x18)", 0x1A, 3},
		{"GFUN4 (masked 0x18)", 0x1B, 3},
		{"ASR1 (masked 0x10)", 0x10, 7},
		{"ASR2 (masked 0x10)", 0x11, 7},
		{"LFO1 (masked 0x14)", 0x14, 7},
		{"GLFO2 (masked 0x14)", 0x15, 7},
		{"KDFX Studio (masked 0x68)", 0x68, 7},
		{"KDFX FXMod (masked 0x68)", 0x69, 7},
		{"Envelope Control", 0x20, 15},
		{"Impact Envelope Control (masked 0x20)", 0x27, 15},
		{"AMPENV", 0x21, 15},
		{"ENV2", 0x22, 15},
		{"ENV3", 0x23, 15},
		{"Hobbes 1 (masked 0x50)", 0x50, 15},
		{"Hobbes 2 (masked 0x50)", 0x51, 15},
		{"Hobbes 3 (masked 0x50)", 0x52, 15},
		{"Hobbes 4 (masked 0x50)", 0x53, 15},
		{"Calvin", 0x40, 31},
		{"unrecognized", 0x00, -1},
		{"unrecognized 2", 0xFF, -1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := segmentDataLen(tt.tag); got != tt.want {
				t.Errorf("segmentDataLen(0x%02x) = %d, want %d", tt.tag, got, tt.want)
			}
		})
	}
}

// TestParseSegments_RoundTripsProgramTemplate verifies the segment parser
// against known-good real bytes: programTemplate was captured verbatim from
// a real K2000 Program object, so parsing it into segments and rebuilding
// should reproduce every byte up to the trailing end-of-segments padding.
func TestParseSegments_RoundTripsProgramTemplate(t *testing.T) {
	segs, err := parseSegments(programTemplate, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segs) == 0 {
		t.Fatal("expected at least one segment")
	}

	// Every tag we know from the doc should be recognized (none skipped as
	// "unrecognized" partway through a real, valid program).
	wantTags := []byte{0x08, 0x0F, 0x09, 0x10, 0x11, 0x18, 0x19, 0x14, 0x15,
		0x1A, 0x1B, 0x20, 0x21, 0x22, 0x23, 0x40, 0x50, 0x51, 0x52, 0x53}
	gotTags := make(map[byte]bool, len(segs))
	for _, s := range segs {
		gotTags[s.tag] = true
	}
	for _, want := range wantTags {
		if !gotTags[want] {
			t.Errorf("expected tag 0x%02x in parsed segments, not found", want)
		}
	}

	rebuilt := buildSegments(segs)
	original := programTemplate[2 : 2+len(rebuilt)]
	if len(rebuilt) != len(original) {
		t.Fatalf("rebuilt length = %d, want %d", len(rebuilt), len(original))
	}
	for i := range rebuilt {
		if rebuilt[i] != original[i] {
			t.Fatalf("byte %d: rebuilt = 0x%02x, want 0x%02x", i, rebuilt[i], original[i])
		}
	}
}

// TestParseSegments_StopsAtZeroTag verifies the walk stops cleanly at the
// end-of-segments marker rather than erroring on the trailing padding.
func TestParseSegments_StopsAtZeroTag(t *testing.T) {
	payload := []byte{0x00, 0x50, 0x08, 0x02, 0x01, 0x00, 0x37, 0x40, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00} // PGM segment then zero padding
	segs, err := parseSegments(payload, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(segs) != 1 || segs[0].tag != 0x08 {
		t.Fatalf("segs = %+v, want one PGM (0x08) segment", segs)
	}
}

// TestParseSegments_TruncatedSegmentErrors verifies a segment whose declared
// length runs past the end of the payload is reported, not silently dropped.
func TestParseSegments_TruncatedSegmentErrors(t *testing.T) {
	payload := []byte{0x00, 0x50, 0x08, 0x02, 0x01} // PGM tag but only 3 data bytes, needs 15
	if _, err := parseSegments(payload, 2); err == nil {
		t.Fatal("expected an error for a truncated segment, got nil")
	}
}

func TestFindSegment(t *testing.T) {
	segs := []segment{{tag: 0x08, data: []byte{1}}, {tag: 0x40, data: []byte{2}}}

	if s, ok := findSegment(segs, 0x40); !ok || s.data[0] != 2 {
		t.Errorf("findSegment(0x40) = %+v, %v, want data[0]=2, true", s, ok)
	}
	if _, ok := findSegment(segs, 0x99); ok {
		t.Error("findSegment(0x99) = true, want false (not present)")
	}
}
