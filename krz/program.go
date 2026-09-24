package krz

import (
	"encoding/binary"

	"sampleslice/config"
)

// VoiceMode determines how a KRZ voice behaves: either as a fixed-pitch drum (mono)
// or as a multi-note polyphonic instrument using sample rate modulation.
type VoiceMode int

const (
	VoiceModeDrum VoiceMode = iota // Fixed-pitch mono, one voice per drum note
	VoiceModePoly                  // Multi-note polyphonic via sample rate modulation
)

// Envelope is the shared ASDR envelope type for KRZ layers.
// It aliases config.Envelope so callers can pass config.Envelope values directly
// without a conversion step.
//
// NOTE: Envelope, VoiceMode, priority and stereo are accepted by CreateFromSlices
// for API compatibility, but none of them currently change the serialized
// Program bytes. Program objects are built from a byte-for-byte template
// captured from a real, working K2000 KRZ file (see programTemplate below);
// which of its 254 bytes control voice mode / priority / envelope / stereo
// has not been independently reverse-engineered yet. Until that mapping is
// confirmed, every generated Program uses the template's own (drum, one-shot)
// defaults regardless of these settings.
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

// programTemplate is the 254-byte object-specific payload of a real, working
// K2000 Program object, captured verbatim from a hardware-authored KRZ file
// (KRZDRMS.KRZ, a one-shot drum voice). Only one field within it is known to
// vary per instance — the referenced Keymap ID at payload offset
// programKeymapRefOffset — confirmed by diffing three real Program objects
// (a "master" drum program and two per-hit voices) that were otherwise
// byte-for-byte identical. Everything else (VAST algorithm routing, filter,
// pan, and amplitude parameters) is reused as-is: its semantics have not
// been independently decoded, but it is known to produce a working,
// audible, correctly one-shot voice on real hardware.
var programTemplate = []byte{
	0x00, 0x50, 0x08, 0x02, 0x01, 0x00, 0x37, 0x40, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x0f, 0x00, 0x01, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x09, 0x00, 0x00, 0x00, 0x0c, 0x6c, 0x00, 0x7f, 0x00, 0x04,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x10, 0x00, 0x7f, 0x01, 0x35, 0x35,
	0x00, 0x35, 0x11, 0x00, 0x7f, 0x02, 0x35, 0x35, 0x00, 0x35, 0x18, 0x01,
	0x00, 0x00, 0x19, 0x02, 0x00, 0x00, 0x14, 0x00, 0x00, 0x2e, 0x00, 0x00,
	0x01, 0x00, 0x15, 0x00, 0x00, 0x2e, 0x00, 0x00, 0x01, 0x00, 0x1a, 0x03,
	0x00, 0x00, 0x1b, 0x04, 0x00, 0x00, 0x20, 0x00, 0x01, 0x00, 0x00, 0x00,
	0x49, 0x00, 0x00, 0x00, 0x48, 0x00, 0x00, 0x00, 0x48, 0x00, 0x21, 0x00,
	0x64, 0x00, 0x00, 0x00, 0x00, 0x00, 0x64, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x22, 0x00, 0x64, 0x67, 0x9c, 0x67, 0x64, 0x67, 0x64, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x23, 0x00, 0x64, 0x00, 0x00, 0x67,
	0x9c, 0x67, 0x00, 0x67, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x40, 0x7f,
	0x00, 0x00, 0x2b, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xc9,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x50, 0x3e, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x51, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x52, 0x28, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x03, 0x00, 0x53, 0x01, 0x06, 0x00, 0x00, 0x14,
	0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x04, 0x04, 0x00, 0x00,
	0x00, 0x00,
}

// programKeymapRefOffset is the byte offset, within programTemplate, of the
// 2-byte big-endian Keymap ID this program references.
const programKeymapRefOffset = 166

// Program represents a Kurzweil Program object (T_PROGRAM): the object a
// user actually selects to play a voice or drum kit. It references a
// Keymap by ID; the Keymap in turn determines whether that Program plays a
// single sample (a per-hit voice) or dispatches to other Programs per key
// (a drum kit's "master" Program+Keymap pair).
type Program struct {
	// ID is the object's numeric ID (>= 200 for user objects).
	ID uint16
	// Name is the display name, up to 16 bytes.
	Name string
	// KeymapID is the ID of the Keymap object this program references.
	KeymapID uint16
}

// NewProgram creates a Program with the given ID, name, and Keymap
// reference. voiceMode, priority, stereo, and envelope are accepted for API
// compatibility but do not currently affect the serialized bytes — see the
// Envelope doc comment.
func NewProgram(id uint16, name string, keymapID uint16, _ VoiceMode, _ uint8, _ bool, _ Envelope) *Program {
	return &Program{ID: id, Name: name, KeymapID: keymapID}
}

// Hash returns this program's object hash.
func (p *Program) Hash() uint16 {
	return GenerateHash(p.ID, T_PROGRAM)
}

// Serialize encodes the program into its binary KRZ object form.
func (p *Program) Serialize() []byte {
	payload := make([]byte, len(programTemplate))
	copy(payload, programTemplate)
	binary.BigEndian.PutUint16(payload[programKeymapRefOffset:programKeymapRefOffset+2], p.KeymapID)
	return buildObject(p.Hash(), p.Name, payload, 6)
}
