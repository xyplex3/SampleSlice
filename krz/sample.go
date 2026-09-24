package krz

import (
	"encoding/binary"
	"math"
)

// Sample represents a Kurzweil Sample object (T_SAMPLE): a compact header
// describing one mono PCM sample that lives, once, in the file's shared PCM
// region. Every field below was confirmed by decoding real K2000 KRZ files
// (KRZDRMS.KRZ, atari.KRZ) byte-for-byte, including the maxPitch formula and
// the fixed one-shot envelope.
type Sample struct {
	// ID is the object's numeric ID (>= 200 for user objects).
	ID uint16
	// Name is the sample's display name, up to 16 bytes.
	Name string
	// RootNote is the MIDI note the sample plays back at its original pitch.
	RootNote uint8
	// SampleRate is the PCM sample rate in Hz.
	SampleRate uint32
	// NumSamples is the number of 16-bit PCM samples (words).
	NumSamples int
	// StartWord is this sample's starting offset, in words (samples), within
	// the file's shared PCM region. Consecutive samples are laid out back to
	// back, so StartWord is the running total of prior samples' NumSamples.
	StartWord uint32

	// pcmBytes is this sample's raw 16-bit big-endian PCM data, appended to
	// the file's shared PCM region in KRZFile.Serialize.
	pcmBytes []byte
}

// Hash returns this sample's object hash.
func (s *Sample) Hash() uint16 {
	return GenerateHash(s.ID, T_SAMPLE)
}

// maxPitch reproduces the K2000's stored maxPitch field: the sample's root
// note expressed in cents, adjusted for how its stored rate compares to a
// 48kHz reference. Confirmed exactly against two real samples at different
// root notes and rates: maxPitch = round(100*rootNote + 1200*log2(48000/sr)).
func (s *Sample) maxPitch() int16 {
	v := 100*float64(s.RootNote) + 1200*math.Log2(48000/float64(s.SampleRate))
	return int16(math.Round(v))
}

// Serialize encodes the sample into its binary KRZ object form, including
// the common object header (hash/size/name).
//
// Binary layout of the object-specific payload (70 bytes), all fields
// confirmed against real files except where noted:
//
//	0:2   name continuation  = 0x0053 ('\0' + 'S'): the K2000's name field
//	      is 18 bytes, not 16 — these are its last 2 bytes, and every
//	      object type stamps them with '\0' + its own initial (confirmed
//	      against the "kurzfile" Python library's header parsing, which
//	      independently corroborates this package's whole object model)
//	2:4   baseID            = 1
//	4:6   numHeaders        = 0 (mono; this implementation is mono-only)
//	6:8   headersOfs        = 8 (constant; not a live pointer in practice)
//	8     flags             = 0 (mono)
//	9     ks1               = 0
//	10:12 copyID            = 0
//	12:14 ks2               = 0
//	14    rootkey           = RootNote
//	15    flags             = 0xF0 (one-shot; no real loop support yet)
//	16    volumeAdjust      = 0
//	17    altVolumeAdjust   = 0
//	18:20 maxPitch          (see maxPitch())
//	20:22 reserved          = 0
//	22:26 sampleStart       = StartWord
//	26:30 altSampleStart    = StartWord
//	30:34 loopStart         = StartWord (matches real one-shot samples, which
//	      set this equal to sampleStart rather than to the sample's end)
//	34:38 sampleEnd         = StartWord + NumSamples - 1
//	38:40 envOffset         = 8 (constant)
//	40:42 altEnvOffset      = 6 (constant)
//	42:46 samplePeriod      = 1_000_000_000 / SampleRate (integer division)
//	46:70 envelope          = two copies of {-1, 1, 0, 0, -1600, 0} (int16),
//	      the exact bytes every real one-shot sample in both reference files
//	      used, regardless of sample content.
func (s *Sample) Serialize() []byte {
	payload := make([]byte, 70)
	binary.BigEndian.PutUint16(payload[0:2], 0x0053)
	binary.BigEndian.PutUint16(payload[2:4], 1)   // baseID
	binary.BigEndian.PutUint16(payload[4:6], 0)   // numHeaders (mono)
	binary.BigEndian.PutUint16(payload[6:8], 8)   // headersOfs
	payload[8] = 0                                // flags (mono)
	payload[9] = 0                                // ks1
	binary.BigEndian.PutUint16(payload[10:12], 0) // copyID
	binary.BigEndian.PutUint16(payload[12:14], 0) // ks2

	sampleEnd := s.StartWord
	if s.NumSamples > 0 {
		sampleEnd = s.StartWord + uint32(s.NumSamples) - 1
	}

	payload[14] = s.RootNote
	payload[15] = 0xF0 // one-shot
	payload[16] = 0    // volumeAdjust
	payload[17] = 0    // altVolumeAdjust
	binary.BigEndian.PutUint16(payload[18:20], uint16(s.maxPitch()))
	binary.BigEndian.PutUint16(payload[20:22], 0)
	binary.BigEndian.PutUint32(payload[22:26], s.StartWord)
	binary.BigEndian.PutUint32(payload[26:30], s.StartWord)
	binary.BigEndian.PutUint32(payload[30:34], s.StartWord) // loopStart == sampleStart for one-shot
	binary.BigEndian.PutUint32(payload[34:38], sampleEnd)
	binary.BigEndian.PutUint16(payload[38:40], 8) // envOffset
	binary.BigEndian.PutUint16(payload[40:42], 6) // altEnvOffset
	samplePeriod := uint32(0)
	if s.SampleRate > 0 {
		samplePeriod = 1_000_000_000 / s.SampleRate
	}
	binary.BigEndian.PutUint32(payload[42:46], samplePeriod)

	envelope := []int16{-1, 1, 0, 0, -1600, 0}
	for copyIdx := 0; copyIdx < 2; copyIdx++ {
		base := 46 + copyIdx*12
		for i, v := range envelope {
			binary.BigEndian.PutUint16(payload[base+i*2:base+i*2+2], uint16(v))
		}
	}

	return buildObject(s.Hash(), s.Name, payload, 4)
}
