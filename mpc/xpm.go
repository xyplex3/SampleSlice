package mpc

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"

	"sampleslice/midi"
	"sampleslice/output"
	"sampleslice/slice"
)

// XPMDocument is the root element of an Akai MPC .xpm XML program file.
// It contains version metadata and the instrument definition.
type XPMDocument struct {
	XMLName    xml.Name      `xml:"MPCVObject"`
	Version    XPMVersion    `xml:"Version"`
	Instrument XPMInstrument `xml:"Instrument"`
}

// XPMVersion carries the file version and application metadata written into
// the .xpm header. MPC software uses these fields for compatibility checks.
type XPMVersion struct {
	FileVersion string `xml:"File_Version"`
	Application string `xml:"Application"`
	Platform    string `xml:"Platform"`
}

// XPMInstrument is the drum-sampler instrument block inside an [XPMDocument].
// It holds the program name and the complete pad assignment list.
type XPMInstrument struct {
	Type        string  `xml:"type,attr"`
	ProgramName string  `xml:"ProgramName"`
	Pads        XPMPads `xml:"Pads"`
}

// XPMPads is the container element for the pad list inside an [XPMInstrument].
type XPMPads struct {
	Pad []XPMPad `xml:"Pad"`
}

// XPMPad maps a single MPC pad number to its sample layer assignment.
// Number is zero-based and corresponds to the pad's position in the program.
type XPMPad struct {
	Number int      `xml:"number,attr"`
	Layer  XPMLayer `xml:"Layer"`
}

// XPMLayer describes the sample assignment and playback parameters for one
// pad layer. Volume is on a 0–100 scale; pan is –100 (left) to +100 (right);
// tuning is in cents. SampleStart and SampleEnd are frame offsets within the
// referenced WAV file; –1 disables loop points.
type XPMLayer struct {
	Number        int     `xml:"number,attr"`
	SampleName    string  `xml:"SampleName"`
	SampleFile    string  `xml:"SampleFile"`
	Active        int     `xml:"Active"`
	Volume        int     `xml:"Volume"`
	Pan           int     `xml:"Pan"`
	Tuning        int     `xml:"Tuning"`
	VolumeAttack  float64 `xml:"VolumeAttack"`
	VolumeHold    float64 `xml:"VolumeHold"`
	VolumeDecay   float64 `xml:"VolumeDecay"`
	VolumeSustain int     `xml:"VolumeSustain"`
	VolumeRelease float64 `xml:"VolumeRelease"`
	PlaybackMode  string  `xml:"PlaybackMode"`
	SampleStart   int     `xml:"SampleStart"`
	SampleEnd     int     `xml:"SampleEnd"`
	LoopStart     int     `xml:"LoopStart"`
	LoopEnd       int     `xml:"LoopEnd"`
}

// GenerateXPMProgram writes individual WAV files to outputDir/Samples/ and
// produces an Akai MPC-compatible <programName>.xpm XML program file.
// It returns a *Program populated with pad metadata.
func GenerateXPMProgram(
	slices []slice.AudioSlice,
	programName string,
	outputDir string,
	sliceSamples [][]int16,
	sampleRate uint32,
) (*Program, error) {
	if len(slices) == 0 {
		return nil, fmt.Errorf("no slices to generate program")
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("creating output directory: %w", err)
	}

	samplesDir := filepath.Join(outputDir, "Samples")
	if err := os.MkdirAll(samplesDir, 0755); err != nil {
		return nil, fmt.Errorf("creating Samples directory: %w", err)
	}

	program := &Program{
		Name:    programName,
		RootDir: outputDir,
	}

	doc := XPMDocument{
		Version: XPMVersion{
			FileVersion: "2.1",
			Application: "SampleSlice",
			Platform:    "MPC",
		},
		Instrument: XPMInstrument{
			Type:        "MPC_DRUM_SAMPLER",
			ProgramName: programName,
		},
	}

	for i, s := range slices {
		fileName := fmt.Sprintf("slice_%03d.wav", i+1)
		wavPath := filepath.Join(samplesDir, fileName)

		var floatSamples []float64
		if i < len(sliceSamples) && len(sliceSamples[i]) > 0 {
			floatSamples = make([]float64, len(sliceSamples[i]))
			for j, v := range sliceSamples[i] {
				floatSamples[j] = float64(v) / 32768.0
			}
		}
		if err := output.WriteWAV(wavPath, floatSamples, sampleRate); err != nil {
			return nil, fmt.Errorf("writing WAV for slice %d: %w", i, err)
		}

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
			FileName: filepath.Join("Samples", fileName),
			Color:    color,
		})

		sampleCount := 0
		if i < len(sliceSamples) {
			sampleCount = len(sliceSamples[i])
		}

		doc.Instrument.Pads.Pad = append(doc.Instrument.Pads.Pad, XPMPad{
			Number: i,
			Layer: XPMLayer{
				Number:        0,
				SampleName:    fileName,
				SampleFile:    "Samples/" + fileName,
				Active:        1,
				Volume:        100,
				Pan:           0,
				Tuning:        0,
				VolumeAttack:  0,
				VolumeHold:    0,
				VolumeDecay:   0,
				VolumeSustain: 100,
				VolumeRelease: 0,
				PlaybackMode:  "ONE_SHOT",
				SampleStart:   0,
				SampleEnd:     sampleCount,
				LoopStart:     -1,
				LoopEnd:       -1,
			},
		})
	}

	xmlData, err := xml.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshalling XPM XML: %w", err)
	}

	xpmPath := filepath.Join(outputDir, programName+".xpm")
	content := append([]byte(xml.Header), xmlData...)
	if err := os.WriteFile(xpmPath, content, 0644); err != nil {
		return nil, fmt.Errorf("writing XPM file: %w", err)
	}

	return program, nil
}
