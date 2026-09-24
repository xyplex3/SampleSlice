package krz

import (
	"fmt"
	"os"
)

// vastSegmentTags lists the tone-shaping segment tags a VAST borrows: the
// envelope-control and amplitude/filter/pitch envelopes, the Calvin segment
// (VAST algorithm number + pitch), and the four Hobbes filter/pan/amp
// blocks. Everything else in a Program (PGM globals, the Layer key/velocity
// range, ASR/FUN/LFO) is deliberately left as the recipient's own, since a
// donor's key range or triggers don't make sense to inherit blindly.
var vastSegmentTags = []byte{0x20, 0x21, 0x22, 0x23, 0x40, 0x50, 0x51, 0x52, 0x53}

// VAST holds a donor Program's tone-shaping segments, extracted from a real
// KRZ file with [LoadVAST], for splicing onto a newly generated Program via
// [WithVAST].
type VAST struct {
	segments []segment
}

// LoadVAST reads path (an existing KRZ file) and extracts the named
// Program's tone-shaping segments (envelope, Calvin/algorithm, and the four
// Hobbes filter/pan/amp blocks) for reuse. programName is matched
// case-insensitively. Returns an error if the file can't be read or parsed,
// the program isn't found, or it has none of the tone-shaping segments.
func LoadVAST(path string, programName string) (*VAST, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	objects, err := ParseObjects(data)
	if err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	prog, ok := FindObject(objects, T_PROGRAM, programName)
	if !ok {
		return nil, fmt.Errorf("program %q not found in %s", programName, path)
	}
	segs, err := parseSegments(prog.Payload, 0)
	if err != nil {
		return nil, fmt.Errorf("parsing program %q's segments: %w", programName, err)
	}

	var vastSegs []segment
	for _, tag := range vastSegmentTags {
		if s, ok := findSegment(segs, tag); ok {
			vastSegs = append(vastSegs, s)
		}
	}
	if len(vastSegs) == 0 {
		return nil, fmt.Errorf("program %q in %s has no VAST tone segments to borrow", programName, path)
	}
	return &VAST{segments: vastSegs}, nil
}

// apply substitutes v's segments into payload's own segment stream (which
// uses this package's internal convention: segments start at offset 2, see
// programTemplate), replacing whichever of the recipient's own segments
// share the same tag, or appending the donor's segment if the recipient
// didn't already have one with that tag. Because a segment's length is
// determined entirely by its tag, substitution never changes payload's
// overall length.
func (v *VAST) apply(payload []byte) error {
	ownSegs, err := parseSegments(payload, 2)
	if err != nil {
		return fmt.Errorf("parsing recipient program's segments: %w", err)
	}

	for _, donor := range v.segments {
		replaced := false
		for i := range ownSegs {
			if ownSegs[i].tag == donor.tag {
				ownSegs[i] = donor
				replaced = true
				break
			}
		}
		if !replaced {
			ownSegs = append(ownSegs, donor)
		}
	}

	rebuilt := buildSegments(ownSegs)
	if 2+len(rebuilt) > len(payload) {
		return fmt.Errorf("spliced segments (%d bytes) don't fit in the program payload (%d bytes available)", len(rebuilt), len(payload)-2)
	}
	copy(payload[2:], rebuilt)
	for i := 2 + len(rebuilt); i < len(payload); i++ {
		payload[i] = 0
	}
	return nil
}
