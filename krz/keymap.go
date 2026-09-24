package krz

import "encoding/binary"

// numKeys is the K2000's keyboard range (MIDI notes 0-127).
const numKeys = 128

// keymapReserved is a fixed 2-byte field observed at the start of every
// real Keymap object's payload, regardless of content. Its meaning hasn't
// been independently decoded; it's reused verbatim.
const keymapReserved = 0x004b

// Keymap represents a Kurzweil Keymap object (T_KEYMAP). Two variants were
// confirmed by decoding real K2000 KRZ files:
//
//   - A single-sample keymap (method 1): every key plays the same
//     referenced Sample, used for one drum hit's per-hit voice.
//   - A "master" multi-entry keymap (method 3): each of the 128 keys holds
//     its own 2-byte Program-ID reference (plus a 1-byte flag), used to
//     dispatch a drum kit's top-level Program to the right per-hit voice.
type Keymap struct {
	ID   uint16
	Name string

	multi    bool
	sampleID uint16   // single-sample mode: the referenced Sample's ID
	entries  []uint16 // multi mode: per-key referenced Program ID, len 128
}

// NewSingleSampleKeymap creates a Keymap that plays sampleID at every key —
// the per-hit keymap shape a real K2000 drum voice uses.
func NewSingleSampleKeymap(id uint16, name string, sampleID uint16) *Keymap {
	return &Keymap{ID: id, Name: name, sampleID: sampleID}
}

// NewMasterKeymap creates a multi-entry Keymap with every key defaulted to
// defaultProgramID. Call SetEntry to assign specific keys to specific
// per-hit Programs.
func NewMasterKeymap(id uint16, name string, defaultProgramID uint16) *Keymap {
	entries := make([]uint16, numKeys)
	for i := range entries {
		entries[i] = defaultProgramID
	}
	return &Keymap{ID: id, Name: name, multi: true, entries: entries}
}

// SetEntry assigns MIDI key (0-127) to reference programID. No-op on a
// single-sample keymap or an out-of-range key.
func (k *Keymap) SetEntry(key int, programID uint16) {
	if !k.multi || key < 0 || key >= len(k.entries) {
		return
	}
	k.entries[key] = programID
}

// Hash returns this keymap's object hash.
func (k *Keymap) Hash() uint16 {
	return GenerateHash(k.ID, T_KEYMAP)
}

// Serialize encodes the keymap into its binary KRZ object form.
//
// Payload layout (all fields confirmed against real files):
//
//	0:2   reserved       = keymapReserved
//	2:4   sampleId       = referenced Sample ID (single-sample) or 0 (multi)
//	4:6   method         = 1 (single-sample) or 3 (multi)
//	6:8   basePitch      = 0
//	8:10  centsPerEntry  = 100
//	10:12 entriesPerVel  = 127
//	12:14 entrySize      = 1 (single-sample) or 3 (multi)
//	14:30 Level[8]       = (8-j)*2 for j in 0..7
//	30:   key entries    = numKeys * entrySize bytes:
//	      single-sample: numKeys bytes, each 0x01
//	      multi: numKeys * (uint16 BE programID + 1 byte flag=1)
func (k *Keymap) Serialize() []byte {
	entrySize := 1
	if k.multi {
		entrySize = 3
	}
	payload := make([]byte, 30+numKeys*entrySize)

	binary.BigEndian.PutUint16(payload[0:2], keymapReserved)
	if k.multi {
		binary.BigEndian.PutUint16(payload[2:4], 0)
		binary.BigEndian.PutUint16(payload[4:6], 3)
	} else {
		binary.BigEndian.PutUint16(payload[2:4], k.sampleID)
		binary.BigEndian.PutUint16(payload[4:6], 1)
	}
	binary.BigEndian.PutUint16(payload[6:8], 0)     // basePitch
	binary.BigEndian.PutUint16(payload[8:10], 100)  // centsPerEntry
	binary.BigEndian.PutUint16(payload[10:12], 127) // entriesPerVel
	binary.BigEndian.PutUint16(payload[12:14], uint16(entrySize))
	for j := 0; j < 8; j++ {
		binary.BigEndian.PutUint16(payload[14+j*2:16+j*2], uint16((8-j)*2))
	}

	if k.multi {
		for i, id := range k.entries {
			off := 30 + i*3
			binary.BigEndian.PutUint16(payload[off:off+2], id)
			payload[off+2] = 1
		}
	} else {
		for i := 30; i < len(payload); i++ {
			payload[i] = 1
		}
	}

	return buildObject(k.Hash(), k.Name, payload, 4)
}
