package krz

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// Program represents a Kurzweil Program object.
// A program is a multi-part instrument definition containing up to 8 layers,
// each referencing a keymap, along with voice mode, priority, and envelope settings.
type Program struct {
	// Name is the display name of the program, up to 16 bytes.
	Name string
	// ID is the unique identifier for this program within a KRZ file.
	ID uint16
	// Hash is the 16-bit hash derived from ID and object type, used for binary serialization.
	Hash uint16
	// Parts is the list of program layers (up to 8), each defining sample playback parameters.
	Parts []Part
	// Keymap is the primary keymap associated with this program (used for convenience).
	Keymap *Keymap
	// VoiceMode determines whether the program plays as drum (mono/fixed) or poly (multi-note).
	VoiceMode VoiceMode
	// Priority is the voice-stealing priority (1-8, where 8 is highest).
	Priority uint8
	// Stereo indicates whether polyphonic stereo mode is enabled.
	Stereo bool
	// Envelope defines the 10-parameter ADSR amplitude envelope applied to all parts.
	Envelope Envelope
}

// Part represents a single layer in a Kurzweil program (up to 8 layers per program).
// Each part references a keymap and defines playback parameters such as key range,
// velocity range, panning, transposition, looping, and effects.
type Part struct {
	// Enabled indicates whether this part is active in the program.
	Enabled bool
	// PartID is the internal part identifier (1-8).
	PartID uint16
	// KeymapRef is the hash of the keymap this part uses for sample playback.
	KeymapRef uint16
	// RootNote is the base MIDI note (0-127) at which the sample plays at original pitch.
	RootNote uint16
	// Transpose shifts the playback pitch in semitones (-64 to +63).
	Transpose int16
	// Velocity is the base playback velocity (0-127).
	Velocity uint16
	// Pan is the stereo position (0=full left, 64=center, 127=full right).
	Pan uint16
	// Output is the audio output bus number (1-16).
	Output uint16
	// LoKey is the lowest MIDI note (0-127) that triggers this part.
	LoKey uint16
	// HiKey is the highest MIDI note (0-127) that triggers this part.
	HiKey uint16
	// LoVel is the lowest velocity (0-127) that triggers this part.
	LoVel uint16
	// HiVel is the highest velocity (0-127) that triggers this part.
	HiVel uint16
	// Reverse enables backward sample playback.
	Reverse bool
	// Loop enables looping between LoopStart and LoopEnd.
	Loop bool
	// LoopStart is the sample position where looping begins, in samples.
	LoopStart uint32
	// LoopEnd is the sample position where looping ends, in samples.
	LoopEnd uint32
	// Offset is the sample position where playback begins, in samples.
	Offset uint32
	// End is the sample position where playback stops, in samples.
	End uint32
	// FineTune adjusts pitch in cents (-100 to +100).
	FineTune int16
	// Portamento enables glide between note pitch changes.
	Portamento bool
	// PortamentoTime is the glide duration in milliseconds.
	PortamentoTime uint16
}

// NewProgram creates a new Program with the given ID, name, voice mode, priority, stereo flag, and envelope.
// The name is truncated to 16 bytes if it exceeds that length.
func NewProgram(id uint16, name string, voiceMode VoiceMode, priority uint8, stereo bool, envelope Envelope) *Program {
	if len(name) > 16 {
		name = name[:16]
	}
	return &Program{
		Name:      name,
		ID:        id,
		Hash:      GenerateHash(id, T_PROGRAM),
		VoiceMode: voiceMode,
		Priority:  priority,
		Stereo:    stereo,
		Envelope:  envelope,
	}
}

// AddPart appends a part to the program's layer list.
// MIDI note values (RootNote, LoKey, HiKey) are clamped to the valid range 0-127.
func (p *Program) AddPart(part Part) {
	// Validate MIDI note range
	part.RootNote = validateMidiNoteUint(part.RootNote)
	part.LoKey = validateMidiNoteUint(part.LoKey)
	part.HiKey = validateMidiNoteUint(part.HiKey)
	p.Parts = append(p.Parts, part)
}

// SetKeymap sets the keymap hash reference for the part at the given index.
// If partIndex is out of range, the call has no effect.
func (p *Program) SetKeymap(partIndex int, keymap *Keymap) {
	if partIndex >= 0 && partIndex < len(p.Parts) {
		p.Parts[partIndex].KeymapRef = keymap.Hash
	}
}

// Serialize converts the program to its binary KRZ representation.
// The output includes a 6-byte header, 8 layer blocks (40 bytes each),
// voice mode, priority, and the 10-parameter envelope.
func (p *Program) Serialize() ([]byte, error) {
	var buf bytes.Buffer
	ew := &errWriter{w: &buf}

	ew.write(p.Hash)
	sizeOffset := buf.Len()
	ew.writeBytes([]byte{0x00, 0x00}) // total size placeholder
	ew.write(uint16(8))               // name offset

	nameBytes := []byte(p.Name)
	if len(nameBytes) < 16 {
		nameBytes = append(nameBytes, make([]byte, 16-len(nameBytes))...)
	}
	ew.writeBytes(nameBytes[:16])

	// LYR segment: 8 layers, each 40 bytes
	for i := 0; i < 8; i++ {
		if i < len(p.Parts) {
			part := p.Parts[i]

			enabledByte := byte(0)
			if part.Enabled {
				enabledByte = 1
			}
			ew.write(enabledByte)
			ew.write(part.PartID)
			ew.write(part.KeymapRef)
			ew.write(part.RootNote)
			ew.write(part.Transpose)
			ew.write(part.Velocity)
			ew.write(byte(part.Pan))
			ew.write(byte(part.Output))
			ew.write(part.LoKey)
			ew.write(part.HiKey)
			ew.write(part.LoVel)
			ew.write(part.HiVel)

			flags := byte(0)
			if part.Reverse {
				flags |= 0x01
			}
			if part.Loop {
				flags |= 0x02
			}
			ew.write(flags)
			ew.write(part.Offset)
			ew.write(part.End)
			ew.write(part.LoopStart)
			ew.write(part.LoopEnd)
			ew.write(part.FineTune)

			portamentoFlags := byte(0)
			if part.Portamento {
				portamentoFlags = 1
			}
			ew.write(portamentoFlags)
			ew.write(part.PortamentoTime)
		} else {
			ew.writeBytes(make([]byte, 40))
		}
	}

	// Voice mode byte: bit 0 = stereo, bit 1 = poly
	voiceModeByte := byte(0)
	if p.VoiceMode == VoiceModePoly {
		voiceModeByte |= 0x02
	}
	if p.Stereo {
		voiceModeByte |= 0x01
	}
	ew.write(voiceModeByte)

	// Priority 1-8 stored as 0-7
	priorityByte := byte(0)
	if p.Priority >= 1 && p.Priority <= 8 {
		priorityByte = p.Priority - 1
	}
	ew.write(priorityByte)

	// Envelope: 9 bytes + 1 reserved
	e := p.Envelope
	ew.writeBytes([]byte{e.Attack, e.Decay1, e.Level1, e.Decay2, e.Level2, e.Decay3, e.Level3, e.Sustain, e.Release, 0})

	if ew.err != nil {
		return nil, fmt.Errorf("serializing program %q: %w", p.Name, ew.err)
	}

	// Fill in the total-size field now that we know the final length.
	data := buf.Bytes()
	binary.BigEndian.PutUint16(data[sizeOffset:sizeOffset+2], uint16(len(data)))

	return data, nil
}

// CalculateSize returns the serialized byte length of the program,
// rounded up to the nearest 2-byte boundary.
func (p *Program) CalculateSize() (int, error) {
	data, err := p.Serialize()
	if err != nil {
		return 0, fmt.Errorf("failed to serialize program: %w", err)
	}
	// Pad to 2-byte boundary
	paddedSize := (len(data) + 1) & ^1
	return paddedSize, nil
}