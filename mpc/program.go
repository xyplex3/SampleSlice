// Package mpc generates Akai MPC program files (.prg) from sliced audio.
// Each program maps slices across up to 8 pad banks of 16 pads each
// (128 pads total). When a slice set exceeds the per-program limit,
// additional program files are created automatically.
package mpc

import (
	"encoding/binary"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"sampleslice/midi"
	"sampleslice/slice"
)

const (
	// PadsPerBank is the number of pads in each MPC bank (fixed at 16).
	PadsPerBank = 16
	// BanksPerProg is the number of pad banks per MPC program (A through H).
	BanksPerProg = 8
	// MaxPadsPerProg is the total pad capacity of one program (8 banks × 16).
	MaxPadsPerProg = PadsPerBank * BanksPerProg // 128
)

// Program represents an MPC program with pads and metadata.
type Program struct {
	Name    string
	Pads    []Pad
	RootDir string // Output directory path
}

// Pad represents a single pad in the MPC program.
type Pad struct {
	Bank     int // 0=A, 1=B, 2=C, 3=D, 4=E, 5=F, 6=G, 7=H
	BankPad  int // 0-15 within bank
	Note     int // MIDI note number
	NoteName string
	FileName string // Relative path to the WAV file
	Color    PadColor
}

// PadColor represents the color of an MPC pad.
type PadColor int

const (
	PadColorWhite  PadColor = 0 // Bank A (and E) pads
	PadColorRed    PadColor = 1 // Bank B (and F) pads
	PadColorGreen  PadColor = 2 // Bank C (and G) pads
	PadColorOrange PadColor = 3 // Bank D (and H) pads
)

// GenerateProgram creates an MPC program from audio slices.
// It exports individual WAV files and generates a .prg program file.
func GenerateProgram(slices []slice.AudioSlice, programName string, outputDir string, startNote int) (*Program, error) {
	if len(slices) == 0 {
		return nil, fmt.Errorf("no slices to generate program")
	}

	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create output directory: %w", err)
	}

	program := &Program{
		Name:    programName,
		RootDir: outputDir,
	}

	for i, s := range slices {
		fileName := fmt.Sprintf("slice_%03d.wav", i+1)
		bank := i / PadsPerBank
		bankPad := i % PadsPerBank
		color := padColorForIndex(i)

		note, err := midi.NoteToMIDI(s.Note)
		if err != nil {
			return nil, fmt.Errorf("slice %d: invalid note name %q: %w", i, s.Note, err)
		}

		program.Pads = append(program.Pads, Pad{
			Bank:     bank,
			BankPad:  bankPad,
			Note:     note,
			NoteName: s.Note,
			FileName: fileName,
			Color:    color,
		})
	}

	prgPath := filepath.Join(outputDir, programName+".prg")
	if err := writePrgFile(prgPath, program); err != nil {
		return nil, fmt.Errorf("failed to write program file: %w", err)
	}

	return program, nil
}

// padColorForIndex assigns a color by bank: A=White, B=Red, C=Green, D=Orange.
func padColorForIndex(index int) PadColor {
	colors := []PadColor{PadColorWhite, PadColorRed, PadColorGreen, PadColorOrange}
	return colors[(index/PadsPerBank)%len(colors)]
}

// writePrgFile writes an MPC .prg program file.
//
// Binary format v2 (big-endian):
//
//	Offset  Size  Field
//	0       4     Magic: "MPC " (0x4D504320)
//	4       2     Version: 2
//	6       32    Program name, null-padded
//	38      2     Pad count (max 128, 8 banks × 16)
//	40+     N     Pad entries, each 43 bytes:
//	               0   1  MIDI note (0-127)
//	               1   1  Pad color (0=white,1=red,2=green,3=orange)
//	               2   1  Bank (0=A,1=B,...,7=H)
//	               3  32  Sample filename, null-padded
//	              35   2  Volume (0-2048, 1024=100%)
//	              37   2  Pan (0-128, 64=center)
//	              39   2  Tuning, signed cents (-1200 to +1200)
//	              41   2  Reserved
//
// Note: this is a SampleSlice-defined format; it is not the official Akai XPM/PRG XML format.
func writePrgFile(path string, program *Program) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	if _, err := file.Write([]byte("MPC ")); err != nil {
		return err
	}

	if err := binary.Write(file, binary.BigEndian, uint16(2)); err != nil {
		return err
	}

	nameBytes := make([]byte, 32)
	copy(nameBytes, []byte(program.Name))
	if _, err := file.Write(nameBytes); err != nil {
		return err
	}

	padCount := uint16(len(program.Pads))
	if padCount > MaxPadsPerProg {
		padCount = MaxPadsPerProg
	}
	if err := binary.Write(file, binary.BigEndian, padCount); err != nil {
		return err
	}

	written := 0
	for _, pad := range program.Pads {
		if written >= MaxPadsPerProg {
			break
		}
		if pad.Note > 127 {
			continue
		}

		if err := binary.Write(file, binary.BigEndian, uint8(pad.Note)); err != nil {
			return err
		}
		if err := binary.Write(file, binary.BigEndian, uint8(pad.Color)); err != nil {
			return err
		}
		if err := binary.Write(file, binary.BigEndian, uint8(pad.Bank)); err != nil {
			return err
		}

		sampleName := pad.FileName
		if len(sampleName) > 31 {
			sampleName = sampleName[:31]
		}
		sampleBytes := make([]byte, 32)
		copy(sampleBytes, []byte(sampleName))
		if _, err := file.Write(sampleBytes); err != nil {
			return err
		}

		if err := binary.Write(file, binary.BigEndian, uint16(1024)); err != nil {
			return err
		}
		if err := binary.Write(file, binary.BigEndian, uint16(64)); err != nil {
			return err
		}
		if err := binary.Write(file, binary.BigEndian, int16(0)); err != nil {
			return err
		}
		if err := binary.Write(file, binary.BigEndian, uint16(0)); err != nil {
			return err
		}
		written++
	}

	return nil
}

// PrintProgramInfo writes a human-readable summary of program to stdout,
// listing each pad's note name, MIDI number, and sample filename.
func PrintProgramInfo(program *Program) {
	fmt.Printf("MPC Program: %s\n", program.Name)
	fmt.Printf("Directory: %s\n", program.RootDir)
	fmt.Printf("Pads: %d\n\n", len(program.Pads))

	for i, pad := range program.Pads {
		colorName := colorToString(pad.Color)
		fmt.Printf("  Pad %d: Note %s (MIDI %d) - %s [%s]\n",
			i+1, pad.NoteName, pad.Note, pad.FileName, colorName)
	}

	xpmPath := filepath.Join(program.RootDir, program.Name+".xpm")
	fmt.Printf("\nProgram file: %s\n", xpmPath)
	fmt.Printf("Pads: %d\n", len(program.Pads))
}

// colorToString converts a PadColor to a human-readable string.
func colorToString(color PadColor) string {
	switch color {
	case PadColorWhite:
		return "White"
	case PadColorRed:
		return "Red"
	case PadColorGreen:
		return "Green"
	case PadColorOrange:
		return "Orange"
	default:
		return "Unknown"
	}
}

// ValidateProgram checks if a program is valid and returns any warnings.
func ValidateProgram(program *Program) []string {
	var warnings []string

	if len(program.Pads) == 0 {
		warnings = append(warnings, "Program has no pads")
	}

	if len(program.Pads) > MaxPadsPerProg {
		warnings = append(warnings, fmt.Sprintf("Program has more than %d pads (will be truncated)", MaxPadsPerProg))
	}

	seen := make(map[int]bool)
	for _, pad := range program.Pads {
		if seen[pad.Note] {
			warnings = append(warnings, fmt.Sprintf("Duplicate note: %s (MIDI %d)", pad.NoteName, pad.Note))
		}
		seen[pad.Note] = true
	}

	xpmPath := filepath.Join(program.RootDir, program.Name+".xpm")
	if _, err := os.Stat(xpmPath); os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf("Missing XPM file: %s.xpm", program.Name))
	}

	return warnings
}

// CountSlicesForProgram returns how many slices fit in a single program (up to
// [MaxPadsPerProg]) and how many additional programs are needed for the remainder.
func CountSlicesForProgram(count int) (firstProgramCount int, additionalPrograms int) {
	if count <= MaxPadsPerProg {
		return count, 0
	}
	return MaxPadsPerProg, (count - MaxPadsPerProg + MaxPadsPerProg - 1) / MaxPadsPerProg
}

// GenerateMultiProgram generates multiple .prg programs when the slice count
// exceeds [MaxPadsPerProg] (8 banks × 16 pads = 128). Notes reset to startNote
// at the start of each program so that MIDI note numbers never exceed 127.
func GenerateMultiProgram(slices []slice.AudioSlice, baseName string, outputDir string, startNote int) ([]*Program, error) {
	if len(slices) == 0 {
		return nil, fmt.Errorf("no slices to generate programs")
	}

	var programs []*Program
	programIndex := 0

	for i := 0; i < len(slices); i += MaxPadsPerProg {
		end := i + MaxPadsPerProg
		if end > len(slices) {
			end = len(slices)
		}

		chunk := slices[i:end]

		// Remap notes locally within each program so they start at startNote.
		// This prevents MIDI note numbers from exceeding 127 across large slice sets.
		remapped := make([]slice.AudioSlice, len(chunk))
		for j, s := range chunk {
			remapped[j] = s
			localNote := startNote + j
			if localNote > 127 {
				localNote = 127
			}
			remapped[j].Note = midi.MIDIToNote(localNote)
		}

		var progDir string
		if len(slices) > MaxPadsPerProg {
			progName := fmt.Sprintf("%s_%d", baseName, programIndex+1)
			progDir = filepath.Join(outputDir, progName)
		} else {
			progDir = outputDir
		}

		program, err := GenerateProgram(remapped, filepath.Base(progDir), progDir, startNote)
		if err != nil {
			return nil, fmt.Errorf("failed to generate program %d: %w", programIndex+1, err)
		}

		programs = append(programs, program)
		programIndex++
	}

	return programs, nil
}

// SanitizeProgramName removes or replaces characters that are not valid in file names.
func SanitizeProgramName(name string) string {
	var result strings.Builder
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-' {
			result.WriteRune(r)
		} else {
			result.WriteRune('_')
		}
	}
	sanitized := result.String()

	parts := strings.Split(sanitized, "_")
	var cleaned []string
	for _, p := range parts {
		if p != "" {
			cleaned = append(cleaned, p)
		}
	}
	return strings.Join(cleaned, "_")
}
