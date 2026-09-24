package krz

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
)

// Object is one object read back from a serialized KRZ file: its type,
// numeric ID, name, and payload (the object-specific data that follows the
// name field).
//
// Payload starts exactly at the real object data — unlike programTemplate's
// internal representation (which, for historical reasons tied to how this
// package always writes a fixed 16-byte name, begins 2 bytes earlier, at
// the tail of the name region). Callers parsing an Object's Payload with
// parseSegments must pass start=0, not start=2.
type Object struct {
	Type    uint16
	ID      uint16
	Name    string
	Payload []byte
}

// ParseObjects walks a serialized KRZ file's flat object section — the
// inverse of [KRZFile.Serialize] — and returns every object between the
// 32-byte header and the terminating int32(0) end marker.
//
// The object name field's length is read from each object's own "ofs"
// field rather than assumed to be a fixed size: real K2000-authored files
// use a minimal null-terminated name (padded to an even total length), not
// this package's own fixed-16-byte convention, so name length varies
// object to object in files this package didn't write itself.
func ParseObjects(data []byte) ([]Object, error) {
	if len(data) < 32 {
		return nil, errors.New("file too short for a KRZ header")
	}
	magic := string(data[0:4])
	if magic != "PRAM" && magic != "SROM" {
		return nil, fmt.Errorf("not a KRZ file: magic = %q, want \"PRAM\" or \"SROM\"", magic)
	}
	osize := binary.BigEndian.Uint32(data[4:8])
	if int(osize) > len(data) {
		return nil, fmt.Errorf("declared object-section size %d exceeds file size %d", osize, len(data))
	}

	var objects []Object
	pos := 32
	for pos < int(osize) {
		if pos+4 > len(data) {
			return nil, fmt.Errorf("truncated block length at offset %d", pos)
		}
		blockSize := int32(binary.BigEndian.Uint32(data[pos : pos+4]))
		if blockSize == 0 {
			break // end marker
		}
		if blockSize > 0 {
			return nil, fmt.Errorf("expected a negative block length at offset %d, got %d", pos, blockSize)
		}
		total := int(-blockSize)
		if total < 10 || pos+total > len(data) {
			return nil, fmt.Errorf("object block at offset %d (declared length %d) is invalid or runs past end of file", pos, total)
		}

		obj, err := parseObject(data[pos : pos+total])
		if err != nil {
			return nil, fmt.Errorf("object at offset %d: %w", pos, err)
		}
		objects = append(objects, obj)
		pos += total
	}

	return objects, nil
}

// parseObject decodes one object block's hash/size/name header, given the
// full block including its leading 4-byte length field.
func parseObject(block []byte) (Object, error) {
	if len(block) < 10 {
		return Object{}, errors.New("block too short for an object header")
	}
	hash := binary.BigEndian.Uint16(block[4:6])
	nameOfs := int(binary.BigEndian.Uint16(block[8:10]))

	// The name-offset field is measured from its own start (block offset 8),
	// so object data begins at block[8+nameOfs:], and the name region (name
	// bytes + null terminator + optional even-alignment pad byte) is
	// everything from block[10:8+nameOfs].
	dataStart := 8 + nameOfs
	if nameOfs < 2 || dataStart > len(block) {
		return Object{}, fmt.Errorf("invalid name offset %d for block of length %d", nameOfs, len(block))
	}
	nameRegion := block[10:dataStart]
	name := nameRegion
	if i := indexByte(nameRegion, 0); i >= 0 {
		name = nameRegion[:i]
	}

	return Object{
		Type:    TypeFromHash(hash),
		ID:      IDFromHash(hash),
		Name:    strings.TrimRight(string(name), " "),
		Payload: block[dataStart:],
	}, nil
}

// indexByte returns the index of the first occurrence of b in data, or -1.
func indexByte(data []byte, b byte) int {
	for i, v := range data {
		if v == b {
			return i
		}
	}
	return -1
}

// FindObject returns the first object of the given type whose name matches
// name case-insensitively, and whether one was found.
func FindObject(objects []Object, objType uint16, name string) (Object, bool) {
	for _, o := range objects {
		if o.Type == objType && strings.EqualFold(o.Name, name) {
			return o, true
		}
	}
	return Object{}, false
}
