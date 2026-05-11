package report_test

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"sampleslice/detect"
	"sampleslice/report"
	"sampleslice/slice"
)

func makeSlices(n int) []slice.AudioSlice {
	s := make([]slice.AudioSlice, n)
	for i := range s {
		s[i] = slice.AudioSlice{
			StartSample: i * 1000,
			EndSample:   (i + 1) * 1000,
			Note:        "C3",
			Transient:   detect.Transient{Energy: 0.5 + float64(i)*0.1},
		}
	}
	return s
}

func TestBuild_PopulatesFields(t *testing.T) {
	slices := makeSlices(3)
	r := report.Build("input.wav", 44100, 0.5, slices)

	if r.InputFile != "input.wav" {
		t.Errorf("InputFile = %q, want %q", r.InputFile, "input.wav")
	}
	if r.SampleRate != 44100 {
		t.Errorf("SampleRate = %d, want 44100", r.SampleRate)
	}
	if r.Sensitivity != 0.5 {
		t.Errorf("Sensitivity = %v, want 0.5", r.Sensitivity)
	}
	if r.TotalSlices != 3 {
		t.Errorf("TotalSlices = %d, want 3", r.TotalSlices)
	}
	if len(r.Slices) != 3 {
		t.Fatalf("len(Slices) = %d, want 3", len(r.Slices))
	}
	if r.Slices[0].MIDINote != 48 {
		t.Errorf("Slices[0].MIDINote = %d, want 48 (C3)", r.Slices[0].MIDINote)
	}
	if r.Slices[1].StartSample != 1000 {
		t.Errorf("Slices[1].StartSample = %d, want 1000", r.Slices[1].StartSample)
	}
}

func TestWriteJSON_ValidRoundtrip(t *testing.T) {
	slices := makeSlices(2)
	r := report.Build("in.wav", 44100, 0.7, slices)
	path := filepath.Join(t.TempDir(), "report.json")

	if err := report.WriteJSON(path, r); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}

	var got report.Report
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.TotalSlices != 2 {
		t.Errorf("TotalSlices = %d, want 2", got.TotalSlices)
	}
}

func TestWriteCSV_HeaderAndRowCount(t *testing.T) {
	slices := makeSlices(4)
	r := report.Build("in.wav", 44100, 0.5, slices)
	path := filepath.Join(t.TempDir(), "report.csv")

	if err := report.WriteCSV(path, r); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer f.Close()

	rows, err := csv.NewReader(f).ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	// 1 header + 4 data rows
	if len(rows) != 5 {
		t.Errorf("rows = %d, want 5", len(rows))
	}
	if rows[0][0] != "index" {
		t.Errorf("header[0] = %q, want %q", rows[0][0], "index")
	}
}

func TestWriteCSV_RowValues(t *testing.T) {
	slices := makeSlices(1)
	r := report.Build("in.wav", 44100, 0.5, slices)
	path := filepath.Join(t.TempDir(), "report.csv")

	if err := report.WriteCSV(path, r); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}

	f, _ := os.Open(path)
	defer f.Close()
	rows, _ := csv.NewReader(f).ReadAll()

	// row[1] is the first data row
	if rows[1][0] != "0" {
		t.Errorf("index = %q, want %q", rows[1][0], "0")
	}
	if rows[1][1] != "C3" {
		t.Errorf("note = %q, want %q", rows[1][1], "C3")
	}
	if rows[1][2] != "48" {
		t.Errorf("midi_note = %q, want %q", rows[1][2], "48")
	}
}

func TestWriteJSON_EmptySlices(t *testing.T) {
	r := report.Build("in.wav", 44100, 0.5, nil)
	path := filepath.Join(t.TempDir(), "report.json")

	if err := report.WriteJSON(path, r); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	data, _ := os.ReadFile(path)
	var got report.Report
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if got.Slices == nil {
		t.Error("Slices should be empty array, not nil")
	}
	if len(got.Slices) != 0 {
		t.Errorf("len(Slices) = %d, want 0", len(got.Slices))
	}
}

func TestWriteCSV_EmptySlices(t *testing.T) {
	r := report.Build("in.wav", 44100, 0.5, nil)
	path := filepath.Join(t.TempDir(), "report.csv")

	if err := report.WriteCSV(path, r); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	f, _ := os.Open(path)
	defer f.Close()
	rows, _ := csv.NewReader(f).ReadAll()
	if len(rows) != 1 {
		t.Errorf("rows = %d, want 1 (header only)", len(rows))
	}
}
