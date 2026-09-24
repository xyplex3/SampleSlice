package krz

import (
	"encoding/binary"
	"fmt"

	"sampleslice/config"
)

// VoiceMode determines how a KRZ voice behaves: either as a fixed-pitch drum (mono)
// or as a multi-note polyphonic instrument using sample rate modulation.
type VoiceMode int

const (
	VoiceModeDrum VoiceMode = iota // Fixed-pitch mono, one voice per drum note
	VoiceModePoly                  // Multi-note polyphonic via sample rate modulation
)

// Envelope is the shared amplitude-envelope type for KRZ layers. It aliases
// config.Envelope, whose fields mirror the real K2000 AMPENV segment
// field-for-field (see [config.Envelope]'s doc comment), so callers can pass
// config.Envelope values directly without a conversion step.
//
// The field layout was confirmed against Geoffrey Mayer's Kurzweil
// K2000/K2500/K2600 file format reference (the reverse-engineer's own
// technical writeup) and cross-checked by diffing real Program objects.
// WithEnvelope patches these bytes directly into the 0x21 AMPENV segment;
// VoiceMode, priority, and stereo remain accepted for API compatibility but
// do not yet change the serialized Program bytes.
type Envelope = config.Envelope

// DefaultDrumEnvelope returns a fast, tight envelope configuration optimized for percussive sounds
// with quick attack, moderate decay, and short release.
func DefaultDrumEnvelope() Envelope {
	return Envelope{
		Att1Level: 100, Att1Time: 0,
		Dec1Level: 0, Dec1Time: 20,
		Rel1Level: 0, Rel1Time: 5,
	}
}

// DefaultPolyEnvelope returns a smooth, expressive envelope configuration suitable for polyphonic
// hits and pad sounds with gradual attack, multi-stage decay, and longer release.
func DefaultPolyEnvelope() Envelope {
	return Envelope{
		Att1Level: 100, Att1Time: 5,
		Dec1Level: 60, Dec1Time: 40,
		Rel1Level: 0, Rel1Time: 40,
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

// The 0x21 AMPENV (amplitude envelope) segment inside programTemplate: tag
// at ampEnvSegOffset, then the loop flag, three attack (level, time) pairs,
// one decay (level, time) pair, two release (level, time) pairs, a reserved
// byte, and a final release time — matching config.Envelope field-for-field.
// Offsets confirmed against Geoffrey Mayer's Kurzweil file format reference
// and cross-checked against real Program objects.
const (
	ampEnvSegOffset  = 106
	ampEnvLoopOffset = 107

	ampEnvAtt1LevelOffset = 108
	ampEnvAtt1TimeOffset  = 109
	ampEnvAtt2LevelOffset = 110
	ampEnvAtt2TimeOffset  = 111
	ampEnvAtt3LevelOffset = 112
	ampEnvAtt3TimeOffset  = 113
	ampEnvDec1LevelOffset = 114
	ampEnvDec1TimeOffset  = 115
	ampEnvRel1LevelOffset = 116
	ampEnvRel1TimeOffset  = 117
	ampEnvRel2LevelOffset = 118
	ampEnvRel2TimeOffset  = 119
	ampEnvReservedOffset  = 120 // always 0: AMPENV has no Rel3 level byte
	ampEnvRel3TimeOffset  = 121
)

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
	// Envelope shapes the amplitude envelope (the 0x21 segment). An all-zero
	// Envelope leaves the template's own (drum, one-shot) shape untouched.
	Envelope Envelope
	// VAST, if set, splices a donor Program's tone-shaping segments
	// (envelope/Calvin/Hobbes) onto this one — see [LoadVAST]. Applied
	// before Envelope and before the KeymapID patch, so an explicit
	// Envelope still wins over a borrowed one, and the Calvin segment's
	// keymap reference always ends up pointing at this Program's own
	// KeymapID rather than the donor's.
	VAST *VAST
}

// NewProgram creates a Program with the given ID, name, and Keymap
// reference. voiceMode, priority, and stereo are accepted for API
// compatibility but do not currently affect the serialized bytes — see the
// Envelope doc comment.
func NewProgram(id uint16, name string, keymapID uint16, _ VoiceMode, _ uint8, _ bool, envelope Envelope) *Program {
	return &Program{ID: id, Name: name, KeymapID: keymapID, Envelope: envelope}
}

// Hash returns this program's object hash.
func (p *Program) Hash() uint16 {
	return GenerateHash(p.ID, T_PROGRAM)
}

// Serialize encodes the program into its binary KRZ object form.
func (p *Program) Serialize() []byte {
	payload := make([]byte, len(programTemplate))
	copy(payload, programTemplate)
	if p.VAST != nil {
		// programTemplate always has every tag a VAST borrows (confirmed by
		// TestParseSegments_RoundTripsProgramTemplate), so this can only
		// fail if the template itself is corrupt — a bug, not user input.
		if err := p.VAST.apply(payload); err != nil {
			panic(fmt.Sprintf("bug: VAST.apply failed on programTemplate: %v", err))
		}
	}
	binary.BigEndian.PutUint16(payload[programKeymapRefOffset:programKeymapRefOffset+2], p.KeymapID)
	if !p.Envelope.IsEmpty() {
		patchAmpEnv(payload, p.Envelope)
	}
	return buildObject(p.Hash(), p.Name, payload, 6)
}

// patchAmpEnv writes the configured envelope into the 0x21 segment's bytes.
// config.Envelope's fields map directly onto AMPENV's real layout, so this
// is a straight field-to-byte copy — no design-choice mapping needed. The
// loop flag and the reserved byte (which has no corresponding config field)
// are always written as 0.
func patchAmpEnv(payload []byte, e Envelope) {
	payload[ampEnvLoopOffset] = 0
	payload[ampEnvAtt1LevelOffset] = clampEnvLevel(e.Att1Level)
	payload[ampEnvAtt1TimeOffset] = e.Att1Time
	payload[ampEnvAtt2LevelOffset] = clampEnvLevel(e.Att2Level)
	payload[ampEnvAtt2TimeOffset] = e.Att2Time
	payload[ampEnvAtt3LevelOffset] = clampEnvLevel(e.Att3Level)
	payload[ampEnvAtt3TimeOffset] = e.Att3Time
	payload[ampEnvDec1LevelOffset] = clampEnvLevel(e.Dec1Level)
	payload[ampEnvDec1TimeOffset] = e.Dec1Time
	payload[ampEnvRel1LevelOffset] = clampEnvLevel(e.Rel1Level)
	payload[ampEnvRel1TimeOffset] = e.Rel1Time
	payload[ampEnvRel2LevelOffset] = clampEnvLevel(e.Rel2Level)
	payload[ampEnvRel2TimeOffset] = e.Rel2Time
	payload[ampEnvReservedOffset] = 0
	payload[ampEnvRel3TimeOffset] = e.Rel3Time
}

// clampEnvLevel clamps a raw 0-255 config level to the 0-100 range real
// AMPENV level bytes use (values above 100 are treated as full level; the
// byte is unsigned, so no negative side exists).
func clampEnvLevel(v uint8) byte {
	if v > 100 {
		return 100
	}
	return v
}
