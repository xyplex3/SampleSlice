package krz

import (
	"bytes"
	"fmt"
)

// Keymap represents a Kurzweil Keymap object containing sample data.
// A keymap groups one or more samples that share a common name and identifier,
// and can be referenced by program parts.
type Keymap struct {
	// Name is the display name of the keymap, up to 16 bytes.
	Name string
	// ID is the unique identifier for this keymap within a KRZ file.
	ID uint16
	// Hash is the 16-bit hash derived from ID and object type, used for binary serialization.
	Hash uint16
	// Samples is the list of sample blocks contained in this keymap.
	Samples []SampleBlock
}

// SampleBlock represents a single sample in a keymap.
// It holds the raw audio data, playback range (MIDI notes), format, and compression settings.
type SampleBlock struct {
	// SampleType indicates the channel layout: 0=mono, 1=stereo left, 2=stereo right.
	SampleType uint8
	// Flags contains bit flags for loop mode and other sample behavior.
	Flags uint8
	// StartNote is the lowest MIDI note (0-127) that triggers this sample.
	StartNote uint16
	// EndNote is the highest MIDI note (0-127) that triggers this sample.
	EndNote uint16
	// RootNote is the MIDI note at which the sample plays at its original pitch.
	RootNote uint16
	// Format is the sample encoding: 0=8-bit unsigned, 2=16-bit signed, 3=ADPCM.
	Format uint16
	// RawData is the uncompressed sample data in the format specified by Format.
	RawData []byte
	// Compressed indicates whether ADPCM compression should be applied during serialization.
	Compressed bool
}

// NewKeymap creates a new [Keymap] with the given ID and name.
// The name is truncated to 16 bytes if it exceeds that length.
func NewKeymap(id uint16, name string) *Keymap {
	if len(name) > 16 {
		name = name[:16]
	}
	return &Keymap{
		Name: name,
		ID:   id,
		Hash: GenerateHash(id, T_KEYMAP),
	}
}

// AddSample adds a sample block to the keymap.
// MIDI note values (rootNote, loKey, hiKey) are clamped to 0-127.
func (k *Keymap) AddSample(sampleData []int16, rootNote uint16, loKey uint16, hiKey uint16, sampleFormat int, compress bool) {
	// Validate MIDI note range
	rootNote = validateMidiNoteUint(rootNote)
	loKey = validateMidiNoteUint(loKey)
	hiKey = validateMidiNoteUint(hiKey)

	// Convert sample data to KRZ format
	var rawData []byte
	if compress {
		// For ADPCM, first convert to 8-bit unsigned
		rawData = ConvertWAVToKRZFormat(sampleData, 8)
	} else {
		rawData = ConvertWAVToKRZFormat(sampleData, sampleFormat)
	}

	block := SampleBlock{
		SampleType: 0, // mono
		Flags:      0,
		StartNote:  loKey,
		EndNote:    hiKey,
		RootNote:   rootNote,
		Format:     uint16(sampleFormat),
		RawData:    rawData,
		Compressed: compress,
	}

	if compress {
		block.Format = 3 // ADPCM format
	}

	k.Samples = append(k.Samples, block)
}

// Serialize encodes the keymap into the binary KRZ keymap format.
// Binary layout:
//   Offset 0-1: Hash (big-endian uint16)
//   Offset 2-3: Sample count (big-endian uint16)
//   Offset 4-9: Reserved (6 zero bytes)
//   Offset 10: Name length (uint8)
//   Offset 11-26: Name data (16 bytes, null-padded)
//   Then: sample blocks
// If a sample block is marked as Compressed, ADPCM compression is applied
// before writing. Returns the serialized bytes or an error if serialization fails.
func (k *Keymap) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	ew := &errWriter{w: &buf}

	ew.write(k.Hash)
	ew.write(uint16(len(k.Samples)))
	ew.writeBytes([]byte{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}) // reserved
	nameBytes := []byte(k.Name)
	nameLen := len(nameBytes)
	if nameLen > 16 {
		nameLen = 16
	}
	padded := make([]byte, 16)
	copy(padded, nameBytes)
	ew.write(byte(nameLen))
	ew.writeBytes(padded)

	for _, block := range k.Samples {
		sampleData := block.RawData
		compressedDataSize := uint32(len(sampleData))

		if block.Compressed {
			compressed := CompressADPCM(sampleData)
			sampleData = compressed
			compressedDataSize = uint32(len(sampleData))
		}

		rawSampleSize := uint32(len(block.RawData))

		ew.write(block.SampleType)
		ew.write(block.Flags)
		ew.write(block.StartNote)
		ew.write(block.EndNote)
		ew.write(block.RootNote)
		ew.write(block.Format)
		ew.write(compressedDataSize)
		ew.write(rawSampleSize)
		ew.writeBytes(sampleData)
	}

	if ew.err != nil {
		return nil, fmt.Errorf("serializing keymap %q: %w", k.Name, ew.err)
	}
	return buf.Bytes(), nil
}

// CalculateSize returns the serialized byte size of the keymap,
// rounded up to the nearest 2-byte boundary.
func (k *Keymap) CalculateSize() (int, error) {
	data, err := k.Serialize()
	if err != nil {
		return 0, fmt.Errorf("failed to serialize keymap: %w", err)
	}
	// Pad to 2-byte boundary
	paddedSize := (len(data) + 1) & ^1
	return paddedSize, nil
}