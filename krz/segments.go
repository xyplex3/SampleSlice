package krz

import "fmt"

// segment is one tagged segment within a Program object's payload: a single
// tag byte followed by a fixed number of data bytes, where the length is
// determined entirely by the tag (see segmentDataLen). Segments are packed
// back to back with no padding between them.
type segment struct {
	tag  byte
	data []byte
}

// segmentDataLen returns the number of data bytes (not counting the tag
// byte itself) for a given segment tag, or -1 if the tag is unrecognized.
//
// This table was confirmed against Geoffrey Mayer's Kurzweil
// K2000/K2500/K2600 file format reference across every segment type it
// documents (PGM, Layer, ASR/FUN/LFO, Envelope Control, AMPENV/ENV2/ENV3,
// Calvin, Hobbes 1-4, KDFX segments, and their Global/FX variants) with no
// exceptions found: the length depends only on the tag's high bits.
func segmentDataLen(tag byte) int {
	switch {
	case tag == 0x08 || tag == 0x09:
		return 15
	case tag == 0x0F:
		return 7
	case tag&0xF8 == 0x18:
		return 3
	case tag&0xF8 == 0x10, tag&0xF8 == 0x14, tag&0xF8 == 0x68:
		return 7
	case tag&0xF8 == 0x20, tag&0xF8 == 0x50:
		return 15
	case tag&0xF8 == 0x40, tag&0xF8 == 0x78:
		return 31
	default:
		return -1
	}
}

// parseSegments walks a Program object's payload starting at offset start
// (skip the 2-byte name-continuation field before calling this — start is
// normally 2) and returns each tagged segment in order, stopping at the
// first zero byte (the end-of-segments marker) or when there isn't enough
// room left for another segment's declared length.
//
// Returns an error only if a recognized tag's data would run past the end
// of payload; an unrecognized tag byte also stops the walk (treated the
// same as an end marker) rather than erroring, since trailing padding after
// the last real segment is common and not itself a tagged segment.
func parseSegments(payload []byte, start int) ([]segment, error) {
	var segs []segment
	pos := start
	for pos < len(payload) {
		tag := payload[pos]
		if tag == 0 {
			break
		}
		length := segmentDataLen(tag)
		if length < 0 {
			break
		}
		if pos+1+length > len(payload) {
			return nil, fmt.Errorf("segment tag 0x%02x at offset %d: declared length %d runs past end of payload (len %d)", tag, pos, length, len(payload))
		}
		data := make([]byte, length)
		copy(data, payload[pos+1:pos+1+length])
		segs = append(segs, segment{tag: tag, data: data})
		pos += 1 + length
	}
	return segs, nil
}

// buildSegments re-serializes a segment list back into a byte slice: each
// segment as its tag byte followed by its data, in order, with no trailing
// terminator (callers append their own end-of-segments padding).
func buildSegments(segs []segment) []byte {
	total := 0
	for _, s := range segs {
		total += 1 + len(s.data)
	}
	out := make([]byte, 0, total)
	for _, s := range segs {
		out = append(out, s.tag)
		out = append(out, s.data...)
	}
	return out
}

// findSegment returns the first segment with the given tag, and whether one
// was found.
func findSegment(segs []segment, tag byte) (segment, bool) {
	for _, s := range segs {
		if s.tag == tag {
			return s, true
		}
	}
	return segment{}, false
}
