// Package krz provides tools for creating and serializing Kurzweil KRZ
// sample library files. The on-disk layout implemented here — the "PRAM"
// file header, the flat length-prefixed object stream, the object hash
// formula, and the Program/Keymap/Sample object bodies — was reverse
// engineered by directly decoding real, hardware-authored K2000 KRZ files
// (KRZDRMS.KRZ and atari.KRZ), not derived from any third-party spec.
//
// Basic usage:
//
//	slices := []krz.SliceData{...}
//	data, err := krz.CreateFromSlices(slices, krz.WithFileName("MyDrums"))
package krz

import (
	"encoding/binary"
	"errors"
)

// fileHeaderTail is the 24 bytes that follow the magic and osize fields in
// every real KRZ file's 32-byte header. These bytes were identical across
// every real file inspected regardless of content, so they're treated as
// fixed format/version boilerplate rather than something callers configure.
var fileHeaderTail = []byte{
	0x00, 0x00, 0xcd, 0x08, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x2d, 0x03, 0x10, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
}

// KRZFile represents a complete Kurzweil KRZ file: a header, a flat object
// section (samples, keymaps, and programs, in that declaration order), and
// a shared PCM data region that Sample objects address by word offset.
type KRZFile struct {
	Samples  []*Sample
	Keymaps  []*Keymap
	Programs []*Program
}

// NewKRZFile creates an empty KRZ file.
func NewKRZFile() *KRZFile {
	return &KRZFile{}
}

// AddSample appends a sample to the file's sample pool.
func (f *KRZFile) AddSample(s *Sample) {
	f.Samples = append(f.Samples, s)
}

// AddKeymap appends a keymap to the file's keymap collection.
func (f *KRZFile) AddKeymap(km *Keymap) {
	f.Keymaps = append(f.Keymaps, km)
}

// AddProgram appends a program to the file's program collection.
func (f *KRZFile) AddProgram(prog *Program) {
	f.Programs = append(f.Programs, prog)
}

// Serialize converts the KRZ file to its binary representation: a 32-byte
// header (magic "PRAM" + osize + fixed tail), the object section (each
// Sample, then each Keymap, then each Program, as a length-prefixed block),
// an int32(0) end marker, and finally the raw 16-bit big-endian PCM data
// referenced by the Sample objects' word offsets.
func (f *KRZFile) Serialize() ([]byte, error) {
	if len(f.Keymaps) == 0 {
		return nil, errors.New("KRZ file must contain at least one keymap")
	}
	if len(f.Programs) == 0 {
		return nil, errors.New("KRZ file must contain at least one program")
	}

	var body []byte
	for _, s := range f.Samples {
		body = append(body, s.Serialize()...)
	}
	for _, km := range f.Keymaps {
		body = append(body, km.Serialize()...)
	}
	for _, prog := range f.Programs {
		body = append(body, prog.Serialize()...)
	}
	body = binary.BigEndian.AppendUint32(body, 0) // end marker

	osize := 32 + len(body)

	out := make([]byte, 0, osize+f.pcmLen())
	out = append(out, 0x50, 0x52, 0x41, 0x4D) // "PRAM"
	out = binary.BigEndian.AppendUint32(out, uint32(osize))
	out = append(out, fileHeaderTail...)
	out = append(out, body...)

	for _, s := range f.Samples {
		out = append(out, s.pcmBytes...)
	}

	return out, nil
}

// pcmLen returns the total size, in bytes, of all samples' PCM data.
func (f *KRZFile) pcmLen() int {
	n := 0
	for _, s := range f.Samples {
		n += len(s.pcmBytes)
	}
	return n
}

// validateMidiNote clamps a MIDI note value to the valid range 0-127.
func validateMidiNote(note int) uint8 {
	if note < 0 {
		return 0
	}
	if note > 127 {
		return 127
	}
	return uint8(note)
}

// truncateName truncates a name to maxLen bytes.
func truncateName(name string, maxLen int) string {
	if len(name) > maxLen {
		return name[:maxLen]
	}
	return name
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

// baseUserObjectID is the first object ID in the K2000's user/RAM object
// space; IDs 0-199 are reserved for ROM/factory banks (confirmed via the
// K2000/K2000RS service manual's memory-bank documentation, and via real
// KRZ files whose user-created objects all start numbering at 200).
const baseUserObjectID = 200

// createConfig holds the configuration for CreateFromSlices.
type createConfig struct {
	fileName   string
	compress   bool
	voiceMode  VoiceMode
	priority   uint8
	stereo     bool
	envelope   Envelope
	sampleRate uint32
}

// defaultCreateConfig returns a createConfig with sensible defaults.
func defaultCreateConfig() createConfig {
	return createConfig{
		fileName:   "SliceProgram",
		voiceMode:  VoiceModeDrum,
		priority:   1,
		envelope:   DefaultDrumEnvelope(),
		sampleRate: 44100,
	}
}

// WithFileName sets the program display name (max 16 bytes).
func WithFileName(name string) CreateOption {
	return func(c *createConfig) {
		c.fileName = name
	}
}

// WithVersion is accepted for API compatibility but has no effect: the real
// KRZ header has no per-file "version" field (see fileHeaderTail).
func WithVersion(uint16) CreateOption {
	return func(*createConfig) {}
}

// WithCompression is accepted for API compatibility but has no effect: the
// real KRZ sample format is always raw 16-bit signed PCM. No ADPCM or
// 8-bit sample format was found in any real file inspected.
func WithCompression(bool) CreateOption {
	return func(*createConfig) {}
}

// WithVoiceMode sets the voice mode (drum or poly). See the Envelope doc
// comment in program.go: this does not yet affect the serialized bytes.
func WithVoiceMode(mode VoiceMode) CreateOption {
	return func(c *createConfig) {
		c.voiceMode = mode
	}
}

// WithPriority sets the voice priority (1-8). Currently a no-op; see the
// Envelope doc comment in program.go.
func WithPriority(priority uint8) CreateOption {
	return func(c *createConfig) {
		c.priority = priority
	}
}

// WithStereo enables or disables stereo voice mode. Currently a no-op; see
// the Envelope doc comment in program.go.
func WithStereo(stereo bool) CreateOption {
	return func(c *createConfig) {
		c.stereo = stereo
	}
}

// WithEnvelope sets the ADSR envelope. Currently a no-op; see the Envelope
// doc comment in program.go.
func WithEnvelope(envelope Envelope) CreateOption {
	return func(c *createConfig) {
		c.envelope = envelope
	}
}

// WithSampleRate sets the sample rate (Hz) used to compute each sample's
// pitch/period fields. Defaults to 44100.
func WithSampleRate(rate uint32) CreateOption {
	return func(c *createConfig) {
		c.sampleRate = rate
	}
}

// CreateFromSlices creates a KRZ file from sliced audio data.
//
// It builds, per slice, a Sample object, a single-sample Keymap referencing
// it, and a Program referencing that Keymap — exactly the three-object
// chain a real K2000 uses for one drum hit. A "master" Keymap (mapping each
// slice's MIDI note to its Program's ID) and a "master" Program referencing
// it tie the kit together into the one Program a user actually selects,
// mirroring how real K2000 drum-kit banks are structured.
func CreateFromSlices(slices []SliceData, opts ...CreateOption) ([]byte, error) {
	if len(slices) == 0 {
		return nil, errors.New("no slices provided")
	}

	cfg := defaultCreateConfig()
	for _, opt := range opts {
		opt(&cfg)
	}

	file := NewKRZFile()

	masterID := uint16(baseUserObjectID)
	firstSliceID := uint16(baseUserObjectID + 1)
	masterKeymap := NewMasterKeymap(masterID, truncateName(cfg.fileName, 16), firstSliceID)

	var pcmWord uint32
	for i, s := range slices {
		id := uint16(baseUserObjectID + 1 + i)
		note := validateMidiNote(s.Note)
		name := truncateName(sliceObjectName(i), 16)

		sample := &Sample{
			ID:         id,
			Name:       name,
			RootNote:   note,
			SampleRate: cfg.sampleRate,
			NumSamples: len(s.Samples),
			StartWord:  pcmWord,
		}
		sample.pcmBytes = int16SamplesToBE(s.Samples)
		pcmWord += uint32(len(s.Samples))
		file.AddSample(sample)

		file.AddKeymap(NewSingleSampleKeymap(id, name, id))
		file.AddProgram(NewProgram(id, name, id, cfg.voiceMode, cfg.priority, cfg.stereo, cfg.envelope))

		if int(note) < len(masterKeymap.entries) {
			masterKeymap.SetEntry(int(note), id)
		}
	}

	file.AddKeymap(masterKeymap)
	file.AddProgram(NewProgram(masterID, truncateName(cfg.fileName, 16), masterID, cfg.voiceMode, cfg.priority, cfg.stereo, cfg.envelope))

	return file.Serialize()
}

// sliceObjectName returns a short, unique per-slice object name.
func sliceObjectName(i int) string {
	const letters = "0123456789"
	n := i + 1
	if n < 1000 {
		return "slice" + string(letters[n/100%10]) + string(letters[n/10%10]) + string(letters[n%10])
	}
	return "slice" + string(letters[n%10])
}

// int16SamplesToBE converts signed 16-bit PCM samples to raw big-endian bytes.
func int16SamplesToBE(samples []int16) []byte {
	out := make([]byte, len(samples)*2)
	for i, s := range samples {
		binary.BigEndian.PutUint16(out[i*2:i*2+2], uint16(s))
	}
	return out
}
