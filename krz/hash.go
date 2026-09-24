package krz

// Object type codes used in the on-disk KRZ object format. Verified by
// directly decoding real Kurzweil K2000 .KRZ files (not derived from any
// third-party spec): every object's hash packs its type code into the high
// bits and the object's numeric ID into the low 10 bits.
const (
	T_PROGRAM = 36 // Program object (instrument/voice definition)
	T_KEYMAP  = 37 // Keymap object (per-key sample/program mapping)
	T_SAMPLE  = 38 // Sample object (sample header + PCM reference)
)

// GenerateHash creates a 16-bit hash from an object ID and type code.
// Format: (type_code << 10) | (id & 0x3FF). Confirmed against real K2000
// KRZ files: a Program object with id 200 hashes to 0x90c8 == (36<<10)|200.
func GenerateHash(id uint16, objectType uint16) uint16 {
	return (objectType << 10) | (id & 0x3FF)
}

// TypeFromHash extracts the object type code from a hash produced by
// [GenerateHash].
func TypeFromHash(hash uint16) uint16 {
	return hash >> 10
}

// IDFromHash extracts the object ID (low 10 bits) from a hash produced by
// [GenerateHash].
func IDFromHash(hash uint16) uint16 {
	return hash & 0x3FF
}
