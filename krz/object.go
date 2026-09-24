package krz

import (
	"bytes"
	"encoding/binary"
)

// objectNameLen is the fixed name field length used by every object type,
// confirmed by direct inspection of real K2000 KRZ files (names are
// space-padded to 16 bytes, not null-padded).
const objectNameLen = 16

// padObjectName truncates or space-pads name to exactly objectNameLen bytes,
// matching the padding real K2000-authored files use.
func padObjectName(name string) []byte {
	b := []byte(name)
	if len(b) > objectNameLen {
		b = b[:objectNameLen]
	}
	out := bytes.Repeat([]byte{' '}, objectNameLen)
	copy(out, b)
	return out
}

// buildObject assembles one complete length-prefixed KRZ object block:
//
//	blocksize (int32 BE, negative, = -(total bytes including this field))
//	hash      (uint16 BE)
//	size      (uint16 BE, = total - sizeAdjust)
//	ofs       (uint16 BE, = objectNameLen + 4, since the name length is
//	           always even in this implementation)
//	name      (objectNameLen bytes, space-padded)
//	payload   (type-specific body)
//
// sizeAdjust is empirically 4 for Sample/Keymap objects and 6 for Program
// objects — confirmed by diffing real object headers of each type.
func buildObject(hash uint16, name string, payload []byte, sizeAdjust int) []byte {
	nameBytes := padObjectName(name)
	total := 4 + 2 + 2 + 2 + objectNameLen + len(payload)

	buf := make([]byte, 0, total)
	buf = binary.BigEndian.AppendUint32(buf, uint32(int32(-total)))
	buf = binary.BigEndian.AppendUint16(buf, hash)
	buf = binary.BigEndian.AppendUint16(buf, uint16(total-sizeAdjust))
	buf = binary.BigEndian.AppendUint16(buf, uint16(objectNameLen+4))
	buf = append(buf, nameBytes...)
	buf = append(buf, payload...)
	return buf
}
