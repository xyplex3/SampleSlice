// Package krz provides tools for creating and serializing Kurzweil KRZ
// sample library files. It supports sample slicing, ADPCM compression,
// multi-part programs, and configurable voice modes (drum and polyphonic).
//
// Basic usage:
//
//	slices := []krz.SliceData{...}
//	data, err := krz.CreateFromSlices(slices,
//	    krz.WithFileName("MyLibrary"),
//	    krz.WithVoiceMode(krz.VoiceModePoly),
//	)
//
// The package handles binary serialization, keymap management, program
// construction, and WAV-to-KRZ format conversion.
package krz

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io"

	"sampleslice/config"
)

// errWriter wraps an io.Writer and records the first error encountered.
// Subsequent writes are no-ops once an error is set, so callers can chain
// writes and check ew.err once at the end.
type errWriter struct {
	w   io.Writer
	err error
}

func (ew *errWriter) write(data any) {
	if ew.err == nil {
		ew.err = binary.Write(ew.w, binary.BigEndian, data)
	}
}

func (ew *errWriter) writeBytes(p []byte) {
	if ew.err == nil {
		_, ew.err = ew.w.Write(p)
	}
}

// VoiceMode determines how a KRZ voice behaves: either as a fixed-pitch drum (mono)
// or as a multi-note polyphonic instrument using sample rate modulation.
type VoiceMode int

const (
	VoiceModeDrum VoiceMode = iota // Fixed-pitch mono, one voice per drum note
	VoiceModePoly                   // Multi-note polyphonic via sample rate modulation
)

// Envelope is the shared ASDR envelope type for KRZ layers.
// It aliases config.Envelope so callers can pass config.Envelope values directly
// without a conversion step.
type Envelope = config.Envelope

// DefaultDrumEnvelope returns a fast, tight envelope configuration optimized for percussive sounds
// with quick attack, moderate decay, and short release.
func DefaultDrumEnvelope() Envelope {
	return Envelope{
		Attack:  0,
		Decay1:  20,
		Level1:  70,
		Decay2:  30,
		Level2:  0,
		Decay3:  0,
		Level3:  0,
		Sustain: 0,
		Release: 5,
	}
}

// DefaultPolyEnvelope returns a smooth, expressive envelope configuration suitable for polyphonic
// hits and pad sounds with gradual attack, multi-stage decay, and longer release.
func DefaultPolyEnvelope() Envelope {
	return Envelope{
		Attack:  5,
		Decay1:  40,
		Level1:  70,
		Decay2:  60,
		Level2:  30,
		Decay3:  80,
		Level3:  0,
		Sustain: 0,
		Release: 40,
	}
}

// KRZFile represents a complete Kurzweil KRZ file.
// It contains keymaps (sample collections) and programs (multi-part instrument definitions)
// that can be serialized into the binary KRZ format for use on Kurzweil synthesizers.
type KRZFile struct {
	// Version is the KRZ file format version number.
	Version uint16
	// Models is the list of compatible Kurzweil model identifiers.
	Models []uint16
	// Keymaps is the collection of sample keymaps in the file.
	Keymaps []*Keymap
	// Programs is the collection of instrument programs in the file.
	Programs []*Program
}

// NewKRZFile creates a new KRZ file with the specified format version and a default model (PC2/PC3).
func NewKRZFile(version uint16) *KRZFile {
	return &KRZFile{
		Version: version,
		Models:  []uint16{0x0064}, // Default model (PC2/PC3)
	}
}

// AddKeymap appends a keymap to the file's keymap collection.
func (f *KRZFile) AddKeymap(km *Keymap) {
	f.Keymaps = append(f.Keymaps, km)
}

// AddProgram appends a program to the file's program collection.
func (f *KRZFile) AddProgram(prog *Program) {
	f.Programs = append(f.Programs, prog)
}

// Serialize converts the KRZ file to its binary representation.
// It writes the file header, object table entries, and serialized object data
// for all keymaps and programs. Returns an error if the file contains no keymaps or programs.
func (f *KRZFile) Serialize() ([]byte, error) {
	if len(f.Keymaps) == 0 {
		return nil, errors.New("KRZ file must contain at least one keymap")
	}
	if len(f.Programs) == 0 {
		return nil, errors.New("KRZ file must contain at least one program")
	}

	var buf bytes.Buffer
	ew := &errWriter{w: &buf}

	// ---- File Header (16 bytes) ----
	// Magic bytes "KRZ" + null
	ew.writeBytes([]byte{0x4B, 0x52, 0x5A, 0x00}) // "KRZ\0"

	// Version
	ew.write(f.Version)

	// Model count
	ew.write(uint16(len(f.Models)))

	// Model list
	for _, model := range f.Models {
		ew.write(model)
	}

	// Object count (keymaps + programs)
	objCount := uint16(len(f.Keymaps) + len(f.Programs))
	ew.write(objCount)

	if ew.err != nil {
		return nil, fmt.Errorf("writing KRZ header: %w", ew.err)
	}

	// ---- Object Table ----
	// Calculate offset to first object data (header + table)
	headerSize := 16
	tableEntrySize := 6 // 2 bytes hash + 4 bytes offset
	tableSize := int(objCount) * tableEntrySize
	firstObjectOffset := headerSize + tableSize
	// Pad to 2-byte boundary
	if firstObjectOffset%2 != 0 {
		firstObjectOffset++
	}

	// Calculate all object sizes and offsets first
	type objectInfo struct {
		data   []byte
		size   int
		offset int
	}

	objects := make([]objectInfo, 0, int(objCount))
	currentOffset := firstObjectOffset

	// Serialize keymaps first
	for _, km := range f.Keymaps {
		data, err := km.Serialize()
		if err != nil {
			return nil, fmt.Errorf("failed to serialize keymap: %w", err)
		}
		paddedSize := (len(data) + 1) & ^1 // Pad to 2-byte boundary
		objects = append(objects, objectInfo{
			data:   data,
			size:   paddedSize,
			offset: currentOffset,
		})
		currentOffset += paddedSize
	}

	// Serialize programs
	for _, prog := range f.Programs {
		data, err := prog.Serialize()
		if err != nil {
			return nil, fmt.Errorf("failed to serialize program: %w", err)
		}
		paddedSize := (len(data) + 1) & ^1
		objects = append(objects, objectInfo{
			data:   data,
			size:   paddedSize,
			offset: currentOffset,
		})
		currentOffset += paddedSize
	}

	// Write object table entries: 2-byte hash + 4-byte file offset per entry.
	for _, obj := range objects {
		// Hash (first 2 bytes of object data)
		ew.write(binary.BigEndian.Uint16(obj.data[0:2]))
		// Offset from start of file
		ew.write(uint32(obj.offset))
	}

	// Pad table to 2-byte boundary if needed
	if buf.Len()%2 != 0 {
		ew.writeBytes([]byte{0x00})
	}

	// Write object data
	for _, obj := range objects {
		ew.writeBytes(obj.data)
		// Pad to 2-byte boundary
		if buf.Len()%2 != 0 {
			ew.writeBytes([]byte{0x00})
		}
	}

	if ew.err != nil {
		return nil, fmt.Errorf("writing KRZ body: %w", ew.err)
	}

	return buf.Bytes(), nil
}

// truncateName truncates a name to maxLen bytes.
func truncateName(name string, maxLen int) string {
	if len(name) > maxLen {
		return name[:maxLen]
	}
	return name
}

// validateMidiNote clamps a MIDI note value to the valid range 0-127.
func validateMidiNote(note int) uint16 {
	if note < 0 {
		return 0
	}
	if note > 127 {
		return 127
	}
	return uint16(note)
}

// validateMidiNoteUint clamps a uint16 MIDI note value to the valid range 0-127.
func validateMidiNoteUint(note uint16) uint16 {
	if note > 127 {
		return 127
	}
	return note
}

// SliceData represents a single sliced audio region.
// It holds the raw sample data and metadata for one slice, including
// the target MIDI note and time boundaries.
type SliceData struct {
	// Samples is the raw 16-bit signed PCM sample data for this slice.
	Samples []int16
	// Note is the target MIDI note number (0-127) for this slice.
	Note int
	// Start is the start time of the slice in seconds, relative to the source audio.
	Start float64
	// End is the end time of the slice in seconds, relative to the source audio.
	End float64
}

// CreateOption is a functional option for configuring CreateFromSlices.
// Each option modifies a createConfig field before file generation.
type CreateOption func(*createConfig)

// createConfig holds the configuration for CreateFromSlices.
type createConfig struct {
	fileName  string
	version   uint16
	compress  bool
	voiceMode VoiceMode
	priority  uint8
	stereo    bool
	envelope  Envelope
}

// defaultCreateConfig returns a createConfig with sensible defaults.
func defaultCreateConfig() createConfig {
	return createConfig{
		fileName:  "SliceProgram",
		version:   0x0001,
		compress:  false,
		voiceMode: VoiceModeDrum,
		priority:  1,
		stereo:    false,
		envelope:  DefaultDrumEnvelope(),
	}
}

// WithFileName sets the program display name (max 16 bytes).
func WithFileName(name string) CreateOption {
	return func(c *createConfig) {
		c.fileName = name
	}
}

// WithVersion sets the KRZ file format version.
func WithVersion(version uint16) CreateOption {
	return func(c *createConfig) {
		c.version = version
	}
}

// WithCompression enables or disables ADPCM compression.
func WithCompression(compress bool) CreateOption {
	return func(c *createConfig) {
		c.compress = compress
	}
}

// WithVoiceMode sets the voice mode (drum or poly).
func WithVoiceMode(mode VoiceMode) CreateOption {
	return func(c *createConfig) {
		c.voiceMode = mode
	}
}

// WithPriority sets the voice priority (1-8).
func WithPriority(priority uint8) CreateOption {
	return func(c *createConfig) {
		c.priority = priority
	}
}

// WithStereo enables or disables stereo voice mode.
func WithStereo(stereo bool) CreateOption {
	return func(c *createConfig) {
		c.stereo = stereo
	}
}

// WithEnvelope sets the ADSR envelope.
func WithEnvelope(envelope Envelope) CreateOption {
	return func(c *createConfig) {
		c.envelope = envelope
	}
}

// CreateFromSlices creates a KRZ file from sliced audio data.
//
// Parameters:
//
//	slices: audio regions with sample data and target MIDI notes
//	opts: functional options for configuring the output (fileName, version, compress, etc.)
//
// Example:
//
//	data, err := CreateFromSlices(slices,
//	    WithFileName("MyDrums"),
//	    WithVoiceMode(VoiceModePoly),
//	    WithCompression(true),
//	)
func CreateFromSlices(slices []SliceData, opts ...CreateOption) ([]byte, error) {
	if len(slices) == 0 {
		return nil, errors.New("no slices provided")
	}

	cfg := defaultCreateConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	krz := NewKRZFile(cfg.version)

	// Create a single keymap with all slices as samples
	keymap := NewKeymap(1, truncateName("SliceKeymap", 16))

	// Determine sample format
	sampleFormat := 8 // default 8-bit unsigned
	if cfg.compress {
		sampleFormat = 3 // ADPCM
	}

	// Add each slice as a sample in the keymap
	for _, s := range slices {
		rootNote := validateMidiNote(s.Note)
		if s.Note == 0 {
			rootNote = 60 // default to C4
		}

		keymap.AddSample(
			s.Samples,
			rootNote,
			rootNote,
			rootNote,
			sampleFormat,
			cfg.compress,
		)
	}

	krz.AddKeymap(keymap)

	// Create a program that references the keymap
	programName := truncateName(cfg.fileName, 16)
	program := NewProgram(1, programName, cfg.voiceMode, cfg.priority, cfg.stereo, cfg.envelope)

	// Create a single part that uses the keymap
	part := Part{
		Enabled:   true,
		PartID:    1,
		KeymapRef: keymap.Hash,
		RootNote:  60,
		Velocity:  100,
		Pan:       64, // Center
		Output:    1,
		LoKey:     0,
		HiKey:     127,
		LoVel:     0,
		HiVel:     127,
		Loop:      false,
	}

	program.AddPart(part)
	krz.AddProgram(program)

	// Serialize the KRZ file
	return krz.Serialize()
}
