package krz

// Object type constants used to identify the type of each object
// in the KRZ file format. Each constant corresponds to a specific
// Kurzweil data structure (programs, keymaps, songs, etc.).
const (
	T_PROGRAM      = 0x84 // Program object (instrument definition)
	T_KEYMAP       = 0x82 // Keymap object (sample collection)
	T_EFFECT       = 0x71 // Effect object
	T_SONG         = 0x70 // Song/sequence object
	T_SETUP        = 0x87 // System setup object
	T_SOUNDBLOCK   = 0x86 // Soundblock object
	T_VELMAP       = 0x68 // Velocity map object
	T_PRESSURE     = 0x69 // Aftertouch pressure object
	T_QUICKACCESS = 0x6F // Quick access preset object
	T_INTONATION   = 0x67 // Intonation/pitch bend object
)

// GenerateHash creates a 16-bit hash from an object ID and type.
// The hash combines the object type as the high byte and the low byte of the ID as the low byte.
// Format: [objectType (8 bits)][id & 0xFF (8 bits)]
func GenerateHash(id uint16, objectType uint16) uint16 {
	return (objectType << 8) | (id & 0xFF)
}

// TypeFromHash extracts the object type byte from a hash produced by [GenerateHash].
// The object type is stored in the high byte of the hash; the low byte holds the
// low 8 bits of the original ID.
func TypeFromHash(hash uint16) uint16 {
	return (hash >> 8) & 0xFF
}

// IDFromHash extracts the low-byte ID from a hash produced by [GenerateHash].
func IDFromHash(hash uint16) uint16 {
	return hash & 0xFF
}